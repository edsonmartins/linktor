package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/internal/infrastructure/vre"
)

// VREService provides visual response engine functionality
type VREService struct {
	renderer   *vre.ChromeRenderer
	registry   *vre.TemplateRegistry
	captionGen *vre.CaptionGenerator
	cache      *redis.Client
	cacheTTL   time.Duration
	cachePrefix string
}

// VREServiceConfig holds configuration for VREService
type VREServiceConfig struct {
	TemplatesPath   string
	CacheTTL        time.Duration
	ChromePoolSize  int
	DefaultWidth    int
	DefaultQuality  int
}

// DefaultVREServiceConfig returns sensible defaults
func DefaultVREServiceConfig() *VREServiceConfig {
	return &VREServiceConfig{
		TemplatesPath:   "./templates",
		CacheTTL:        5 * time.Minute,
		ChromePoolSize:  3,
		DefaultWidth:    800,
		DefaultQuality:  85,
	}
}

// NewVREService creates a new VRE service
func NewVREService(cfg *VREServiceConfig, redisClient *redis.Client) (*VREService, error) {
	if cfg == nil {
		cfg = DefaultVREServiceConfig()
	}

	// Create renderer
	rendererCfg := vre.DefaultRendererConfig()
	rendererCfg.ChromePoolSize = cfg.ChromePoolSize
	rendererCfg.DefaultWidth = cfg.DefaultWidth
	rendererCfg.DefaultQuality = cfg.DefaultQuality

	renderer, err := vre.NewChromeRenderer(rendererCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create renderer: %w", err)
	}

	// Create registry
	registry, err := vre.NewTemplateRegistry(cfg.TemplatesPath)
	if err != nil {
		renderer.Close()
		return nil, fmt.Errorf("failed to create registry: %w", err)
	}

	// Create caption generator
	captionGen := vre.NewCaptionGenerator()

	return &VREService{
		renderer:    renderer,
		registry:    registry,
		captionGen:  captionGen,
		cache:       redisClient,
		cacheTTL:    cfg.CacheTTL,
		cachePrefix: "vre:",
	}, nil
}

// Render renders a template and returns the image
func (s *VREService) Render(ctx context.Context, req *entity.RenderRequest) (*entity.RenderResponse, error) {
	startTime := time.Now()

	// Validate request
	if err := req.Validate(); err != nil {
		return nil, err
	}

	// Generate cache key
	cacheKey := s.generateCacheKey(req)

	// Check cache
	if s.cache != nil {
		if cached, err := s.getFromCache(ctx, cacheKey); err == nil && cached != nil {
			cached.CacheHit = true
			cached.RenderTime = time.Since(startTime)
			return cached, nil
		}
	}

	// Get rendering defaults based on channel
	defaults := req.GetDefaults()

	// Determine the HTML to render
	var html string
	var err error

	if req.IsCustomHTML() {
		// Use custom HTML directly
		html = req.HTML
	} else {
		// Render from predefined template
		html, err = s.registry.RenderTemplate(req.TenantID, req.TemplateID, req.Data)
		if err != nil {
			return nil, fmt.Errorf("failed to render template: %w", err)
		}
	}

	// Render HTML to image
	imageData, err := s.renderer.RenderHTML(ctx, html, vre.RenderOpts{
		Width:   defaults.Width,
		Format:  defaults.Format,
		Quality: defaults.Quality,
		Scale:   defaults.Scale,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to render image: %w", err)
	}

	// Get image dimensions
	width, height, _ := vre.GetImageDimensions(imageData)

	// Generate caption (only for predefined templates)
	caption := req.Caption
	if caption == "" && !req.IsCustomHTML() {
		caption = s.captionGen.Generate(req.TemplateID, req.Data, "")
	}

	// Build response
	response := &entity.RenderResponse{
		ImageBase64:  encodeBase64(imageData, defaults.Format),
		Caption:      caption,
		FollowUpText: req.FollowUpText,
		Width:        width,
		Height:       height,
		Format:       defaults.Format,
		SizeBytes:    int64(len(imageData)),
		RenderTime:   time.Since(startTime),
		CacheHit:     false,
	}

	// Save to cache
	if s.cache != nil {
		s.saveToCache(ctx, cacheKey, response)
	}

	return response, nil
}

// RenderHTML renders custom HTML directly (shorthand method)
func (s *VREService) RenderHTML(ctx context.Context, tenantID, html string, opts *entity.RenderRequest) (*entity.RenderResponse, error) {
	req := &entity.RenderRequest{
		TenantID: tenantID,
		HTML:     html,
		Channel:  entity.VREChannelWhatsApp,
	}
	if opts != nil {
		req.Caption = opts.Caption
		req.Channel = opts.Channel
		req.Width = opts.Width
		req.Format = opts.Format
		req.Quality = opts.Quality
		req.Scale = opts.Scale
	}
	return s.Render(ctx, req)
}

// RenderToURL renders and returns a URL (for CDN upload)
func (s *VREService) RenderToURL(ctx context.Context, req *entity.RenderRequest) (*entity.RenderResponse, error) {
	// For now, just render to base64
	// TODO: Implement CDN upload (S3, MinIO, etc.)
	return s.Render(ctx, req)
}

// ListTemplates returns available templates for a tenant
func (s *VREService) ListTemplates(ctx context.Context, tenantID string) ([]string, error) {
	return s.registry.ListTemplates(tenantID)
}

// GetBrandConfig returns the brand configuration for a tenant
func (s *VREService) GetBrandConfig(ctx context.Context, tenantID string) (*entity.TenantBrandConfig, error) {
	return s.registry.GetBrandConfig(tenantID)
}

// SaveBrandConfig saves the brand configuration for a tenant
func (s *VREService) SaveBrandConfig(ctx context.Context, config *entity.TenantBrandConfig) error {
	return s.registry.SaveBrandConfig(config)
}

// SaveTemplate saves a custom template for a tenant
func (s *VREService) SaveTemplate(ctx context.Context, tenantID, templateID, content string) error {
	return s.registry.SaveTemplate(tenantID, templateID, content)
}

// PreviewTemplate renders a preview with sample data
func (s *VREService) PreviewTemplate(ctx context.Context, tenantID, templateID string) (*entity.RenderResponse, error) {
	sampleData := s.getSampleData(templateID)
	return s.Render(ctx, &entity.RenderRequest{
		TenantID:   tenantID,
		TemplateID: templateID,
		Data:       sampleData,
		Channel:    entity.VREChannelWhatsApp,
	})
}

// InvalidateCache invalidates cached renders for a tenant
func (s *VREService) InvalidateCache(ctx context.Context, tenantID string) error {
	if s.cache == nil {
		return nil
	}

	pattern := fmt.Sprintf("%s%s:*", s.cachePrefix, tenantID)
	iter := s.cache.Scan(ctx, 0, pattern, 0).Iterator()
	for iter.Next(ctx) {
		s.cache.Del(ctx, iter.Val())
	}
	return iter.Err()
}

// Close releases resources
func (s *VREService) Close() error {
	return s.renderer.Close()
}

// generateCacheKey generates a unique cache key for a render request
func (s *VREService) generateCacheKey(req *entity.RenderRequest) string {
	var contentToHash []byte

	if req.IsCustomHTML() {
		// Hash the custom HTML content
		contentToHash = []byte(req.HTML)
	} else {
		// Hash the template ID + data
		dataBytes, _ := json.Marshal(req.Data)
		contentToHash = append([]byte(req.TemplateID), dataBytes...)
	}

	hash := sha256.Sum256(contentToHash)
	contentHash := hex.EncodeToString(hash[:8]) // Use first 8 bytes

	defaults := req.GetDefaults()
	return fmt.Sprintf("%s%s:%s:%d:%s",
		s.cachePrefix,
		req.TenantID,
		contentHash,
		defaults.Width,
		defaults.Format,
	)
}

// getFromCache retrieves a cached response
func (s *VREService) getFromCache(ctx context.Context, key string) (*entity.RenderResponse, error) {
	data, err := s.cache.Get(ctx, key).Bytes()
	if err != nil {
		return nil, err
	}

	var response entity.RenderResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, err
	}

	return &response, nil
}

// saveToCache saves a response to cache
func (s *VREService) saveToCache(ctx context.Context, key string, response *entity.RenderResponse) error {
	data, err := json.Marshal(response)
	if err != nil {
		return err
	}

	return s.cache.Set(ctx, key, data, s.cacheTTL).Err()
}

// getSampleData returns sample data for a template
func (s *VREService) getSampleData(templateID string) map[string]interface{} {
	switch entity.TemplateType(templateID) {
	case entity.TemplateTypeMenuOpcoes:
		return map[string]interface{}{
			"titulo": "Como posso ajudar?",
			"opcoes": []map[string]interface{}{
				{"label": "Fazer pedido", "descricao": "Monte seu pedido", "icone": "pedido"},
				{"label": "Status do pedido", "descricao": "Rastreie sua entrega", "icone": "entrega"},
				{"label": "Ver catálogo", "descricao": "Conheça nossos produtos", "icone": "catalogo"},
			},
		}
	case entity.TemplateTypeCardProduto:
		return map[string]interface{}{
			"nome":     "Produto Exemplo",
			"sku":      "SKU-12345",
			"preco":    99.90,
			"unidade":  "un",
			"estoque":  100,
			"destaque": "novo",
		}
	case entity.TemplateTypeStatusPedido:
		return map[string]interface{}{
			"numero_pedido":    "1234",
			"status_atual":     "transporte",
			"itens_resumo":     "5 produtos",
			"valor_total":      250.00,
			"previsao_entrega": "Hoje, 18h",
			"motorista":        "João S.",
		}
	case entity.TemplateTypeListaProdutos:
		return map[string]interface{}{
			"titulo": "Produtos Disponíveis",
			"produtos": []map[string]interface{}{
				{"nome": "Produto A", "preco": 29.90, "unidade": "un", "estoque_status": "disponivel"},
				{"nome": "Produto B", "preco": 49.90, "unidade": "kg", "estoque_status": "baixo"},
				{"nome": "Produto C", "preco": 19.90, "unidade": "un", "estoque_status": "disponivel"},
			},
		}
	case entity.TemplateTypeConfirmacao:
		return map[string]interface{}{
			"titulo":           "Confirmar Pedido?",
			"previsao_entrega": "Amanhã",
			"itens": []map[string]interface{}{
				{"nome": "Item 1", "quantidade": "2 un", "preco": 59.80},
				{"nome": "Item 2", "quantidade": "1 kg", "preco": 35.00},
			},
			"valor_total": 94.80,
		}
	case entity.TemplateTypeCobrancaPix:
		return map[string]interface{}{
			"valor":         150.00,
			"numero_pedido": "5678",
			"pix_payload":   "00020126580014br.gov.bcb.pix0136example1234567890520400005303986",
			"expiracao":     "30 minutos",
		}
	default:
		return map[string]interface{}{}
	}
}

// encodeBase64 encodes image data to base64 with data URI prefix
func encodeBase64(data []byte, format entity.OutputFormat) string {
	mimeType := "image/png"
	switch format {
	case entity.OutputFormatWebP:
		mimeType = "image/webp"
	case entity.OutputFormatJPEG:
		mimeType = "image/jpeg"
	}

	encoded := make([]byte, (len(data)*4)/3+4)
	n := encodeBase64Raw(encoded, data)
	return fmt.Sprintf("data:%s;base64,%s", mimeType, string(encoded[:n]))
}

// encodeBase64Raw is a simple base64 encoder (avoiding import cycle)
func encodeBase64Raw(dst, src []byte) int {
	const encodeStd = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"

	n := 0
	for i := 0; i < len(src); i += 3 {
		var b0, b1, b2 byte
		b0 = src[i]
		if i+1 < len(src) {
			b1 = src[i+1]
		}
		if i+2 < len(src) {
			b2 = src[i+2]
		}

		dst[n] = encodeStd[b0>>2]
		dst[n+1] = encodeStd[((b0&0x03)<<4)|(b1>>4)]
		if i+1 < len(src) {
			dst[n+2] = encodeStd[((b1&0x0f)<<2)|(b2>>6)]
		} else {
			dst[n+2] = '='
		}
		if i+2 < len(src) {
			dst[n+3] = encodeStd[b2&0x3f]
		} else {
			dst[n+3] = '='
		}
		n += 4
	}
	return n
}
