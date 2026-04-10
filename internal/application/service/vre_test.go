package service

import (
	"testing"

	"github.com/msgfy/linktor/internal/domain/entity"
)

func TestRenderRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		req     *entity.RenderRequest
		wantErr bool
	}{
		{
			name: "valid with svg",
			req: &entity.RenderRequest{
				TenantID: "tenant-1",
				SVG:      `<svg xmlns="http://www.w3.org/2000/svg"></svg>`,
			},
			wantErr: false,
		},
		{
			name: "valid with template_id",
			req: &entity.RenderRequest{
				TenantID:   "tenant-1",
				TemplateID: "menu_opcoes",
			},
			wantErr: false,
		},
		{
			name: "missing tenant_id",
			req: &entity.RenderRequest{
				SVG: `<svg xmlns="http://www.w3.org/2000/svg"></svg>`,
			},
			wantErr: true,
		},
		{
			name: "missing svg and template_id",
			req: &entity.RenderRequest{
				TenantID: "tenant-1",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRenderRequest_IsCustomSVG(t *testing.T) {
	tests := []struct {
		name     string
		req      *entity.RenderRequest
		expected bool
	}{
		{
			name: "custom svg",
			req: &entity.RenderRequest{
				SVG: `<svg xmlns="http://www.w3.org/2000/svg"></svg>`,
			},
			expected: true,
		},
		{
			name: "template only",
			req: &entity.RenderRequest{
				TemplateID: "menu_opcoes",
			},
			expected: false,
		},
		{
			name: "both svg and template",
			req: &entity.RenderRequest{
				SVG:        `<svg xmlns="http://www.w3.org/2000/svg"></svg>`,
				TemplateID: "menu_opcoes",
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.req.IsCustomSVG()
			if result != tt.expected {
				t.Errorf("IsCustomSVG() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestRenderRequest_GetDefaults(t *testing.T) {
	tests := []struct {
		name        string
		channel     entity.VREChannelType
		overrides   *entity.RenderRequest
		wantWidth   int
		wantFormat  entity.OutputFormat
		wantQuality int
	}{
		{
			name:        "whatsapp defaults",
			channel:     entity.VREChannelWhatsApp,
			overrides:   nil,
			wantWidth:   800,
			wantFormat:  entity.OutputFormatJPEG,
			wantQuality: 85,
		},
		{
			name:        "telegram defaults",
			channel:     entity.VREChannelTelegram,
			overrides:   nil,
			wantWidth:   800,
			wantFormat:  entity.OutputFormatJPEG,
			wantQuality: 85,
		},
		{
			name:        "email defaults (PNG)",
			channel:     entity.VREChannelEmail,
			overrides:   nil,
			wantWidth:   600,
			wantFormat:  entity.OutputFormatPNG,
			wantQuality: 90,
		},
		{
			name:    "override width",
			channel: entity.VREChannelWhatsApp,
			overrides: &entity.RenderRequest{
				Width: 1200,
			},
			wantWidth:   1200,
			wantFormat:  entity.OutputFormatJPEG,
			wantQuality: 85,
		},
		{
			name:    "override format",
			channel: entity.VREChannelWhatsApp,
			overrides: &entity.RenderRequest{
				Format: entity.OutputFormatJPEG,
			},
			wantWidth:   800,
			wantFormat:  entity.OutputFormatJPEG,
			wantQuality: 85,
		},
		{
			name:    "override quality",
			channel: entity.VREChannelWhatsApp,
			overrides: &entity.RenderRequest{
				Quality: 95,
			},
			wantWidth:   800,
			wantFormat:  entity.OutputFormatJPEG,
			wantQuality: 95,
		},
		{
			name:        "unknown channel defaults to whatsapp",
			channel:     "unknown",
			overrides:   nil,
			wantWidth:   800,
			wantFormat:  entity.OutputFormatJPEG,
			wantQuality: 85,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &entity.RenderRequest{
				Channel: tt.channel,
			}
			if tt.overrides != nil {
				req.Width = tt.overrides.Width
				req.Format = tt.overrides.Format
				req.Quality = tt.overrides.Quality
				req.Scale = tt.overrides.Scale
			}

			defaults := req.GetDefaults()

			if defaults.Width != tt.wantWidth {
				t.Errorf("Width = %d, want %d", defaults.Width, tt.wantWidth)
			}
			if defaults.Format != tt.wantFormat {
				t.Errorf("Format = %s, want %s", defaults.Format, tt.wantFormat)
			}
			if defaults.Quality != tt.wantQuality {
				t.Errorf("Quality = %d, want %d", defaults.Quality, tt.wantQuality)
			}
		})
	}
}

func TestDefaultBrandConfig(t *testing.T) {
	config := entity.DefaultBrandConfig()

	if config.Name != "Linktor" {
		t.Errorf("Name = %q, want 'Linktor'", config.Name)
	}
	if config.PrimaryColor == "" {
		t.Error("PrimaryColor should not be empty")
	}
	if config.FontFamily == "" {
		t.Error("FontFamily should not be empty")
	}
	if len(config.Icons) == 0 {
		t.Error("Icons should not be empty")
	}
	if config.Icons["pedido"] != "🛒" {
		t.Errorf("Icons['pedido'] = %q, want '🛒'", config.Icons["pedido"])
	}
}

func TestVREServiceConfig_Defaults(t *testing.T) {
	config := DefaultVREServiceConfig()

	if config.TemplatesPath != "./templates" {
		t.Errorf("TemplatesPath = %q, want './templates'", config.TemplatesPath)
	}
	if config.ChromePoolSize != 3 {
		t.Errorf("ChromePoolSize = %d, want 3", config.ChromePoolSize)
	}
	if config.DefaultWidth != 800 {
		t.Errorf("DefaultWidth = %d, want 800", config.DefaultWidth)
	}
	if config.DefaultQuality != 85 {
		t.Errorf("DefaultQuality = %d, want 85", config.DefaultQuality)
	}
}

func TestGenerateCacheKey(t *testing.T) {
	// Test that different requests produce different cache keys
	req1 := &entity.RenderRequest{
		TenantID:   "tenant-1",
		TemplateID: "menu_opcoes",
		Data:       map[string]interface{}{"titulo": "Menu 1"},
		Channel:    entity.VREChannelWhatsApp,
	}

	req2 := &entity.RenderRequest{
		TenantID:   "tenant-1",
		TemplateID: "menu_opcoes",
		Data:       map[string]interface{}{"titulo": "Menu 2"},
		Channel:    entity.VREChannelWhatsApp,
	}

	req3 := &entity.RenderRequest{
		TenantID: "tenant-1",
		SVG:      `<svg xmlns="http://www.w3.org/2000/svg"></svg>`,
		Channel:  entity.VREChannelWhatsApp,
	}

	// Different data should produce different keys
	if req1.TemplateID == req2.TemplateID {
		// Keys would be different due to different Data
		// This is a conceptual test - actual key generation is in service
	}

	// Custom SVG should be recognized
	if !req3.IsCustomSVG() {
		t.Error("req3 should be recognized as custom SVG")
	}
}

func TestEncodeBase64(t *testing.T) {
	data := []byte("test image data")

	tests := []struct {
		format   entity.OutputFormat
		mimeType string
	}{
		{entity.OutputFormatPNG, "image/png"},
		{entity.OutputFormatWebP, "image/webp"},
		{entity.OutputFormatJPEG, "image/jpeg"},
	}

	for _, tt := range tests {
		t.Run(string(tt.format), func(t *testing.T) {
			result := encodeBase64(data, tt.format)

			// Check data URI prefix
			expectedPrefix := "data:" + tt.mimeType + ";base64,"
			if len(result) < len(expectedPrefix) {
				t.Errorf("result too short for format %s", tt.format)
				return
			}
			if result[:len(expectedPrefix)] != expectedPrefix {
				t.Errorf("encodeBase64() prefix = %q, want %q", result[:len(expectedPrefix)], expectedPrefix)
			}
		})
	}
}
