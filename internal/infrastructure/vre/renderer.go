package vre

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"net/url"
	"os/exec"
	"time"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
	"github.com/msgfy/linktor/internal/domain/entity"
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

// ChromeRenderer implements Renderer using chromedp
type ChromeRenderer struct {
	allocCtx    context.Context
	allocCancel context.CancelFunc
	pool        *BrowserPool
	config      *RendererConfig
}

// RendererConfig holds configuration for the renderer
type RendererConfig struct {
	ChromePoolSize int
	DefaultWidth   int
	DefaultFormat  entity.OutputFormat
	DefaultQuality int
	DefaultScale   float64
	RenderTimeout  time.Duration
	Headless       bool
	DisableGPU     bool
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
		Headless:       true,
		DisableGPU:     true,
	}
}

// NewChromeRenderer creates a new Chrome-based renderer
func NewChromeRenderer(cfg *RendererConfig) (*ChromeRenderer, error) {
	if cfg == nil {
		cfg = DefaultRendererConfig()
	}

	// Create Chrome options
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", cfg.Headless),
		chromedp.Flag("disable-gpu", cfg.DisableGPU),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.Flag("disable-extensions", true),
		chromedp.Flag("disable-background-networking", true),
		chromedp.Flag("disable-sync", true),
		chromedp.Flag("disable-translate", true),
		chromedp.Flag("mute-audio", true),
		chromedp.Flag("hide-scrollbars", true),
		chromedp.WindowSize(cfg.DefaultWidth, 1080),
	)

	allocCtx, allocCancel := chromedp.NewExecAllocator(context.Background(), opts...)

	// Create browser pool
	pool, err := NewBrowserPool(allocCtx, cfg.ChromePoolSize)
	if err != nil {
		allocCancel()
		return nil, fmt.Errorf("failed to create browser pool: %w", err)
	}

	return &ChromeRenderer{
		allocCtx:    allocCtx,
		allocCancel: allocCancel,
		pool:        pool,
		config:      cfg,
	}, nil
}

// RenderSVG renders SVG content to an image. The SVG is used directly as the
// browser document, avoiding an intermediate markup layout.
func (r *ChromeRenderer) RenderSVG(ctx context.Context, svg string, opts RenderOpts) ([]byte, error) {
	return r.renderDataURL(ctx, "data:image/svg+xml;charset=utf-8,"+url.QueryEscape(svg), "svg", opts)
}

func (r *ChromeRenderer) renderDataURL(ctx context.Context, dataURL, readySelector string, opts RenderOpts) ([]byte, error) {
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

	// Get a browser context from the pool
	browserCtx, release, err := r.pool.Acquire(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to acquire browser: %w", err)
	}
	defer release()

	// Create a new tab context with timeout
	tabCtx, tabCancel := context.WithTimeout(browserCtx, r.config.RenderTimeout)
	defer tabCancel()

	taskCtx, taskCancel := chromedp.NewContext(tabCtx)
	defer taskCancel()

	var buf []byte

	err = chromedp.Run(taskCtx,
		// Set viewport with scaling for better quality
		chromedp.EmulateViewport(int64(float64(opts.Width)*opts.Scale), 1, chromedp.EmulateScale(opts.Scale)),

		chromedp.Navigate(dataURL),

		// Wait for content to be ready
		chromedp.WaitReady(readySelector),

		// Small delay for fonts/images to load
		chromedp.Sleep(100*time.Millisecond),

		// Capture full page screenshot
		chromedp.ActionFunc(func(ctx context.Context) error {
			// Get the page dimensions
			var contentHeight int64
			if err := chromedp.Evaluate(`Math.ceil(Math.max(
				document.body ? document.body.scrollHeight : 0,
				document.documentElement ? document.documentElement.scrollHeight : 0,
				document.documentElement && document.documentElement.getBoundingClientRect ? document.documentElement.getBoundingClientRect().height : 0
			))`, &contentHeight).Do(ctx); err != nil {
				return err
			}
			if contentHeight <= 0 {
				contentHeight = 1
			}

			// Capture screenshot with exact dimensions
			var screenshotBuf []byte
			screenshotBuf, err = page.CaptureScreenshot().
				WithFormat(page.CaptureScreenshotFormatPng). // Always capture as PNG first
				WithClip(&page.Viewport{
					X:      0,
					Y:      0,
					Width:  float64(opts.Width) * opts.Scale,
					Height: float64(contentHeight),
					Scale:  1,
				}).
				WithCaptureBeyondViewport(true).
				Do(ctx)
			if err != nil {
				return err
			}

			buf = screenshotBuf
			return nil
		}),
	)

	if err != nil {
		return nil, fmt.Errorf("failed to render SVG: %w", err)
	}

	// Convert to desired format with optimization
	return r.convertFormat(buf, opts)
}

// convertFormat converts PNG to the desired format with optimization
func (r *ChromeRenderer) convertFormat(pngData []byte, opts RenderOpts) ([]byte, error) {
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
		if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: opts.Quality}); err != nil {
			return nil, fmt.Errorf("failed to encode JPEG: %w", err)
		}

	default:
		return pngData, nil
	}

	return buf.Bytes(), nil
}

// Close releases all resources
func (r *ChromeRenderer) Close() error {
	if r.pool != nil {
		r.pool.Close()
	}
	if r.allocCancel != nil {
		r.allocCancel()
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

	// Run pngquant
	// pngquant reads from stdin and outputs to stdout with --output -
	cmd := exec.Command(pngquantPath, "--quality", qualityArg, "--speed", "3", "-")
	cmd.Stdin = bytes.NewReader(pngData)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		// If pngquant fails (e.g., quality too high), return original
		// Exit code 99 means quality can't be achieved, which is fine
		if exitErr, ok := err.(*exec.ExitError); ok {
			if exitErr.ExitCode() == 99 {
				return pngData, nil
			}
		}
		// For other errors, just return original
		return pngData, nil
	}

	// If compressed is smaller, use it
	if stdout.Len() > 0 && stdout.Len() < len(pngData) {
		return stdout.Bytes(), nil
	}

	return pngData, nil
}
