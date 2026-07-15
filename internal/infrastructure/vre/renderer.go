package vre

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"image/png"
	"math"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	resvg "github.com/kanrichan/resvg-go"
	"github.com/msgfy/linktor/internal/domain/entity"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/gomedium"
	"golang.org/x/image/font/gofont/goregular"
)

// Renderer defines the interface for SVG to image rendering
type Renderer interface {
	RenderSVG(ctx context.Context, svg string, opts RenderOpts) ([]byte, error)
	Close() error
}

// RenderOpts contains options for rendering
type RenderOpts struct {
	Width   int
	Format  entity.OutputFormat
	Quality int     // 0-100, for webp/jpeg
	Scale   float64 // 1.0 = normal, 2.0 = retina
}

// defaultFontFamily is the family name embedded fonts are registered under and
// the fallback used to resolve generic families (system-ui, sans-serif, ...)
// referenced by the templates. The Go font (Bigelow & Holmes) is a clean,
// permissively-licensed sans-serif shipped with golang.org/x/image.
const defaultFontFamily = "Go"

// emojiFontFamily is the family name of the embedded Noto Emoji face.
const emojiFontFamily = "Noto Emoji"

// notoEmoji provides monochrome glyphs for the emoji the templates use
// (🛒 🚚 📦 📍 keycaps, ✓, ...). Loaded alongside the text font so resvg can do
// per-glyph fallback for codepoints the Go font lacks.
//
//go:embed fonts/NotoEmoji-Regular.ttf
var notoEmoji []byte

// SVGRenderer implements Renderer using resvg (via resvg-go / WASM), rendering
// SVG natively without a headless browser. Each pooled worker owns an isolated
// WASM instance (single linear memory, not concurrency-safe), so a fixed pool
// bounds concurrency while allowing parallel renders across workers.
type SVGRenderer struct {
	config  *RendererConfig
	workers chan *resvgWorker
	all     []*resvgWorker
	mu      sync.Mutex
	closed  bool
}

type resvgWorker struct {
	ctx *resvg.Context
	r   *resvg.Renderer
}

// RendererConfig holds configuration for the renderer
type RendererConfig struct {
	// PoolSize bounds how many SVGs can be rasterized concurrently. Kept named
	// ChromePoolSize for backward-compatibility with existing callers/config.
	ChromePoolSize int
	DefaultWidth   int
	DefaultFormat  entity.OutputFormat
	DefaultQuality int
	DefaultScale   float64
	RenderTimeout  time.Duration
}

// DefaultRendererConfig returns sensible defaults
func DefaultRendererConfig() *RendererConfig {
	return &RendererConfig{
		ChromePoolSize: 3,
		DefaultWidth:   800,
		DefaultFormat:  entity.OutputFormatJPEG,
		DefaultQuality: 85,
		DefaultScale:   1.5,
		RenderTimeout:  10 * time.Second,
	}
}

// NewSVGRenderer creates a new resvg-based renderer with a warm worker pool.
func NewSVGRenderer(cfg *RendererConfig) (*SVGRenderer, error) {
	if cfg == nil {
		cfg = DefaultRendererConfig()
	}
	size := cfg.ChromePoolSize
	if size <= 0 {
		size = 3
	}

	r := &SVGRenderer{
		config:  cfg,
		workers: make(chan *resvgWorker, size),
	}

	for i := 0; i < size; i++ {
		w, err := newResvgWorker()
		if err != nil {
			r.Close()
			return nil, fmt.Errorf("failed to warm up renderer pool: %w", err)
		}
		r.all = append(r.all, w)
		r.workers <- w
	}

	return r, nil
}

// NewChromeRenderer is a backward-compatible alias. The renderer no longer uses
// Chrome; it rasterizes SVG natively via resvg.
//
// Deprecated: use NewSVGRenderer.
func NewChromeRenderer(cfg *RendererConfig) (*SVGRenderer, error) {
	return NewSVGRenderer(cfg)
}

func newResvgWorker() (*resvgWorker, error) {
	rc, err := resvg.NewContext(context.Background())
	if err != nil {
		return nil, err
	}
	rr, err := rc.NewRenderer()
	if err != nil {
		rc.Close()
		return nil, err
	}
	// Register embedded fonts (regular / medium / bold) so text — including the
	// 600-weight labels the templates use — resolves without any system fonts.
	for _, data := range [][]byte{goregular.TTF, gomedium.TTF, gobold.TTF, notoEmoji} {
		if err := rr.LoadFontData(data); err != nil {
			rr.Close()
			rc.Close()
			return nil, fmt.Errorf("failed to load font: %w", err)
		}
	}
	// Default family used to resolve generic/unknown families in the SVG.
	if err := rr.SetFontFamily(defaultFontFamily); err != nil {
		rr.Close()
		rc.Close()
		return nil, err
	}
	return &resvgWorker{ctx: rc, r: rr}, nil
}

// RenderSVG rasterizes SVG content to an image in the requested format.
func (r *SVGRenderer) RenderSVG(ctx context.Context, svg string, opts RenderOpts) ([]byte, error) {
	// Apply defaults
	if opts.Width == 0 {
		opts.Width = r.config.DefaultWidth
	}
	if opts.Format == "" {
		opts.Format = r.config.DefaultFormat
	}
	if opts.Quality == 0 {
		opts.Quality = r.config.DefaultQuality
	}
	if opts.Scale == 0 {
		opts.Scale = r.config.DefaultScale
	}

	// Acquire a worker (bounded concurrency).
	acqCtx := ctx
	if r.config.RenderTimeout > 0 {
		var cancel context.CancelFunc
		acqCtx, cancel = context.WithTimeout(ctx, r.config.RenderTimeout)
		defer cancel()
	}

	var w *resvgWorker
	select {
	case w = <-r.workers:
	case <-acqCtx.Done():
		return nil, fmt.Errorf("failed to acquire renderer: %w", acqCtx.Err())
	}
	defer func() { r.workers <- w }()

	// Compute output dimensions: normalize the SVG's intrinsic width to the
	// requested Width, then apply Scale for high-DPI output while preserving
	// aspect ratio.
	// resvg-go (v0.0.1) does not fall back for generic/unknown families
	// (system-ui, sans-serif, ...) nor across fonts per-glyph, and it mishandles
	// per-tspan font-family. So: point every family at the embedded text font,
	// then give pure-emoji text elements the emoji font (their icon renders) while
	// stripping stray pictographs from mixed text (avoids tofu boxes). Keycaps
	// (1️⃣) degrade to the plain digit.
	svg = normalizeFontFamily(svg)
	svg = handleEmoji(svg)

	iw, ih := parseSVGSize(svg)
	if iw <= 0 {
		iw = float64(opts.Width)
	}
	factor := (float64(opts.Width) / iw) * opts.Scale
	outW := uint32(math.Round(iw * factor))
	var outH uint32
	if ih > 0 {
		outH = uint32(math.Round(ih * factor))
	}
	if outW == 0 {
		outW = uint32(math.Round(float64(opts.Width) * opts.Scale))
	}

	pngData, err := w.r.RenderWithSize([]byte(svg), outW, outH)
	if err != nil {
		return nil, fmt.Errorf("failed to render SVG: %w", err)
	}

	return r.convertFormat(pngData, opts)
}

// svgWidthRe / svgHeightRe / svgViewBoxRe extract intrinsic dimensions from the
// root <svg> element. width/height take precedence; viewBox is the fallback.
var (
	svgWidthRe   = regexp.MustCompile(`(?is)<svg\b[^>]*?\bwidth\s*=\s*"([0-9.]+)`)
	svgHeightRe  = regexp.MustCompile(`(?is)<svg\b[^>]*?\bheight\s*=\s*"([0-9.]+)`)
	svgViewBoxRe = regexp.MustCompile(`(?is)<svg\b[^>]*?\bviewBox\s*=\s*"\s*[0-9.+-]+\s+[0-9.+-]+\s+([0-9.]+)\s+([0-9.]+)`)
)

// font-family normalization: attribute form (font-family="...") and CSS form
// (font-family: ...) both get pointed at the single embedded family.
var (
	fontFamilyAttrRe = regexp.MustCompile(`(?i)font-family\s*=\s*("[^"]*"|'[^']*')`)
	fontFamilyCSSRe  = regexp.MustCompile(`(?i)font-family\s*:\s*[^;}"']+`)
)

func normalizeFontFamily(svg string) string {
	svg = fontFamilyAttrRe.ReplaceAllString(svg, `font-family="`+defaultFontFamily+`"`)
	svg = fontFamilyCSSRe.ReplaceAllString(svg, `font-family:`+defaultFontFamily)
	return svg
}

// Emoji handling. VS16/ZWJ are excluded from the pictographic class so keycap
// handling (which owns the VS16) is independent.
var (
	keycapRe    = regexp.MustCompile(`([0-9#*])\x{FE0F}?\x{20E3}`)
	vs16Re      = regexp.MustCompile(`\x{FE0F}`)
	emojiRunRe  = regexp.MustCompile(`[\x{2600}-\x{27BF}\x{2B00}-\x{2BFF}\x{1F000}-\x{1FAFF}]+`)
	textElemRe  = regexp.MustCompile(`(?s)<text\b[^>]*>[^<]*</text>`)
	familyAttrR = regexp.MustCompile(`(?i)font-family\s*=\s*("[^"]*"|'[^']*')`)
)

func handleEmoji(svg string) string {
	// Keycap sequences → their plain digit; drop stray variation selectors.
	svg = keycapRe.ReplaceAllString(svg, `$1`)
	svg = vs16Re.ReplaceAllString(svg, "")
	// Noto Emoji lacks the light check mark (U+2713) but has the heavy one
	// (U+2714); map so checkmarks render instead of tofu.
	svg = strings.ReplaceAll(svg, "✓", "✔")

	// Direct-text <text> elements: pure-emoji ones get the emoji font so the icon
	// renders; mixed ones keep the text font and have stray pictographs removed.
	return textElemRe.ReplaceAllStringFunc(svg, func(el string) string {
		open := el[:strings.Index(el, ">")+1]
		content := el[len(open) : len(el)-len("</text>")]
		if strings.TrimSpace(content) == "" {
			return el
		}
		if isPureEmoji(content) {
			return setFontFamily(open, emojiFontFamily) + content + "</text>"
		}
		if emojiRunRe.MatchString(content) {
			return open + emojiRunRe.ReplaceAllString(content, "") + "</text>"
		}
		return el
	})
}

// isPureEmoji reports whether content is only pictographic emoji and whitespace.
func isPureEmoji(content string) bool {
	hasEmoji := false
	for _, r := range content {
		switch {
		case r == ' ' || r == '\t' || r == '\n' || r == '\r' || r == 0x200D:
			continue
		case (r >= 0x2600 && r <= 0x27BF) || (r >= 0x2B00 && r <= 0x2BFF) || (r >= 0x1F000 && r <= 0x1FAFF):
			hasEmoji = true
		default:
			return false
		}
	}
	return hasEmoji
}

// setFontFamily rewrites (or inserts) the font-family attribute in an opening tag.
func setFontFamily(openTag, family string) string {
	repl := `font-family="` + family + `"`
	if familyAttrR.MatchString(openTag) {
		return familyAttrR.ReplaceAllString(openTag, repl)
	}
	return openTag[:len(openTag)-1] + ` ` + repl + `>`
}

func parseSVGSize(svg string) (w, h float64) {
	if m := svgWidthRe.FindStringSubmatch(svg); m != nil {
		w, _ = strconv.ParseFloat(m[1], 64)
	}
	if m := svgHeightRe.FindStringSubmatch(svg); m != nil {
		h, _ = strconv.ParseFloat(m[1], 64)
	}
	if w <= 0 || h <= 0 {
		if m := svgViewBoxRe.FindStringSubmatch(svg); m != nil {
			if w <= 0 {
				w, _ = strconv.ParseFloat(m[1], 64)
			}
			if h <= 0 {
				h, _ = strconv.ParseFloat(m[2], 64)
			}
		}
	}
	return w, h
}

// convertFormat converts PNG to the desired format with optimization
func (r *SVGRenderer) convertFormat(pngData []byte, opts RenderOpts) ([]byte, error) {
	// If PNG is desired, optimize with pngquant
	if opts.Format == entity.OutputFormatPNG {
		return compressPNG(pngData, opts.Quality)
	}

	// Decode the PNG
	img, err := png.Decode(bytes.NewReader(pngData))
	if err != nil {
		return nil, fmt.Errorf("failed to decode PNG: %w", err)
	}

	var buf bytes.Buffer

	switch opts.Format {
	case entity.OutputFormatWebP:
		return nil, fmt.Errorf("webp output is not supported by the current renderer")

	case entity.OutputFormatJPEG:
		// JPEG has no alpha; composite over white so transparent SVG areas don't
		// turn black.
		bounds := img.Bounds()
		rgba := image.NewRGBA(bounds)
		draw.Draw(rgba, bounds, image.NewUniform(color.White), image.Point{}, draw.Src)
		draw.Draw(rgba, bounds, img, bounds.Min, draw.Over)
		if err := jpeg.Encode(&buf, rgba, &jpeg.Options{Quality: opts.Quality}); err != nil {
			return nil, fmt.Errorf("failed to encode JPEG: %w", err)
		}

	default:
		return pngData, nil
	}

	return buf.Bytes(), nil
}

// Close releases all pooled WASM instances.
func (r *SVGRenderer) Close() error {
	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		return nil
	}
	r.closed = true
	all := r.all
	r.mu.Unlock()

	for _, w := range all {
		if w.r != nil {
			w.r.Close()
		}
		if w.ctx != nil {
			w.ctx.Close()
		}
	}
	return nil
}

// GetImageDimensions returns the dimensions of an image
func GetImageDimensions(data []byte) (width, height int, err error) {
	img, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return 0, 0, err
	}
	return img.Width, img.Height, nil
}

// compressPNG compresses PNG data using pngquant if available
// Falls back to original if pngquant is not installed
func compressPNG(pngData []byte, quality int) ([]byte, error) {
	// Check if pngquant is available
	pngquantPath, err := exec.LookPath("pngquant")
	if err != nil {
		// pngquant not installed, return original
		return pngData, nil
	}

	// Calculate quality range (pngquant uses min-max format)
	minQuality := quality - 10
	if minQuality < 0 {
		minQuality = 0
	}
	qualityArg := fmt.Sprintf("%d-%d", minQuality, quality)

	cmd := exec.Command(pngquantPath, "--quality", qualityArg, "--force", "--output", "-", "-")
	cmd.Stdin = bytes.NewReader(pngData)
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		// Compression failed, return original
		return pngData, nil
	}
	return out.Bytes(), nil
}
