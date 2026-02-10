package vre

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"sync"

	"github.com/msgfy/linktor/internal/domain/entity"
)

// TemplateRegistry manages VRE templates with tenant-specific overrides
type TemplateRegistry struct {
	basePath   string
	templates  map[string]*template.Template // cache: tenant_id:template_id -> template
	configs    map[string]*entity.TenantBrandConfig
	mu         sync.RWMutex
}

// NewTemplateRegistry creates a new template registry
func NewTemplateRegistry(basePath string) (*TemplateRegistry, error) {
	// Ensure base path exists
	if _, err := os.Stat(basePath); os.IsNotExist(err) {
		if err := os.MkdirAll(basePath, 0755); err != nil {
			return nil, fmt.Errorf("failed to create templates directory: %w", err)
		}
	}

	return &TemplateRegistry{
		basePath:  basePath,
		templates: make(map[string]*template.Template),
		configs:   make(map[string]*entity.TenantBrandConfig),
	}, nil
}

// Resolve returns the template for a tenant and template ID
// Resolution order:
//   1. tenants/{tenant_id}/{template_id}.html (custom tenant template)
//   2. default/{template_id}.html (default template)
func (r *TemplateRegistry) Resolve(tenantID, templateID string) (*template.Template, error) {
	cacheKey := fmt.Sprintf("%s:%s", tenantID, templateID)

	// Check cache first
	r.mu.RLock()
	if tmpl, ok := r.templates[cacheKey]; ok {
		r.mu.RUnlock()
		return tmpl, nil
	}
	r.mu.RUnlock()

	// Try to load template
	tmpl, err := r.loadTemplate(tenantID, templateID)
	if err != nil {
		return nil, err
	}

	// Cache it
	r.mu.Lock()
	r.templates[cacheKey] = tmpl
	r.mu.Unlock()

	return tmpl, nil
}

// loadTemplate loads a template from disk
func (r *TemplateRegistry) loadTemplate(tenantID, templateID string) (*template.Template, error) {
	var templatePath string

	// 1. Try tenant-specific template
	tenantPath := filepath.Join(r.basePath, "tenants", tenantID, templateID+".html")
	if _, err := os.Stat(tenantPath); err == nil {
		templatePath = tenantPath
	} else {
		// 2. Fall back to default template
		defaultPath := filepath.Join(r.basePath, "default", templateID+".html")
		if _, err := os.Stat(defaultPath); err == nil {
			templatePath = defaultPath
		} else {
			return nil, fmt.Errorf("template not found: %s (checked %s and %s)", templateID, tenantPath, defaultPath)
		}
	}

	// Read template content
	content, err := os.ReadFile(templatePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read template: %w", err)
	}

	// Parse with custom functions
	tmpl, err := template.New(templateID).
		Funcs(TemplateFuncs).
		Parse(string(content))
	if err != nil {
		return nil, fmt.Errorf("failed to parse template: %w", err)
	}

	return tmpl, nil
}

// GetBrandConfig returns the brand configuration for a tenant
func (r *TemplateRegistry) GetBrandConfig(tenantID string) (*entity.TenantBrandConfig, error) {
	// Check cache
	r.mu.RLock()
	if config, ok := r.configs[tenantID]; ok {
		r.mu.RUnlock()
		return config, nil
	}
	r.mu.RUnlock()

	// Try to load from disk
	config, err := r.loadBrandConfig(tenantID)
	if err != nil {
		// Return default config if not found
		config = entity.DefaultBrandConfig()
		config.TenantID = tenantID
	}

	// Cache it
	r.mu.Lock()
	r.configs[tenantID] = config
	r.mu.Unlock()

	return config, nil
}

// loadBrandConfig loads brand configuration from disk
func (r *TemplateRegistry) loadBrandConfig(tenantID string) (*entity.TenantBrandConfig, error) {
	configPath := filepath.Join(r.basePath, "tenants", tenantID, "config.json")

	content, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var config entity.TenantBrandConfig
	if err := json.Unmarshal(content, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	config.TenantID = tenantID
	return &config, nil
}

// RenderTemplate renders a template with data and brand config
func (r *TemplateRegistry) RenderTemplate(tenantID, templateID string, data interface{}) (string, error) {
	// Get template
	tmpl, err := r.Resolve(tenantID, templateID)
	if err != nil {
		return "", err
	}

	// Get brand config
	brand, err := r.GetBrandConfig(tenantID)
	if err != nil {
		return "", err
	}

	// Prepare template data
	templateData := entity.TemplateData{
		Brand: brand,
		Data:  make(map[string]interface{}),
	}

	// Convert data to map if needed
	switch v := data.(type) {
	case map[string]interface{}:
		templateData.Data = v
	default:
		// Try to marshal/unmarshal to convert to map
		jsonBytes, err := json.Marshal(data)
		if err != nil {
			return "", fmt.Errorf("failed to convert data: %w", err)
		}
		if err := json.Unmarshal(jsonBytes, &templateData.Data); err != nil {
			return "", fmt.Errorf("failed to convert data: %w", err)
		}
	}

	// Execute template
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, templateData); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.String(), nil
}

// InvalidateCache clears the cache for a tenant
func (r *TemplateRegistry) InvalidateCache(tenantID string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Clear all templates for this tenant
	for key := range r.templates {
		if len(key) > len(tenantID)+1 && key[:len(tenantID)+1] == tenantID+":" {
			delete(r.templates, key)
		}
	}

	// Clear config
	delete(r.configs, tenantID)
}

// InvalidateAll clears the entire cache
func (r *TemplateRegistry) InvalidateAll() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.templates = make(map[string]*template.Template)
	r.configs = make(map[string]*entity.TenantBrandConfig)
}

// ListTemplates returns available templates for a tenant
func (r *TemplateRegistry) ListTemplates(tenantID string) ([]string, error) {
	templates := make(map[string]bool)

	// List default templates
	defaultPath := filepath.Join(r.basePath, "default")
	if entries, err := os.ReadDir(defaultPath); err == nil {
		for _, entry := range entries {
			if !entry.IsDir() && filepath.Ext(entry.Name()) == ".html" {
				name := entry.Name()[:len(entry.Name())-5] // Remove .html
				templates[name] = true
			}
		}
	}

	// List tenant-specific templates (override defaults)
	tenantPath := filepath.Join(r.basePath, "tenants", tenantID)
	if entries, err := os.ReadDir(tenantPath); err == nil {
		for _, entry := range entries {
			if !entry.IsDir() && filepath.Ext(entry.Name()) == ".html" {
				name := entry.Name()[:len(entry.Name())-5]
				templates[name] = true
			}
		}
	}

	// Convert to slice
	result := make([]string, 0, len(templates))
	for name := range templates {
		result = append(result, name)
	}

	return result, nil
}

// SaveBrandConfig saves the brand configuration for a tenant
func (r *TemplateRegistry) SaveBrandConfig(config *entity.TenantBrandConfig) error {
	tenantPath := filepath.Join(r.basePath, "tenants", config.TenantID)

	// Create tenant directory if not exists
	if err := os.MkdirAll(tenantPath, 0755); err != nil {
		return fmt.Errorf("failed to create tenant directory: %w", err)
	}

	// Marshal config
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write file
	configPath := filepath.Join(tenantPath, "config.json")
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	// Invalidate cache
	r.InvalidateCache(config.TenantID)

	return nil
}

// SaveTemplate saves a custom template for a tenant
func (r *TemplateRegistry) SaveTemplate(tenantID, templateID, content string) error {
	tenantPath := filepath.Join(r.basePath, "tenants", tenantID)

	// Create tenant directory if not exists
	if err := os.MkdirAll(tenantPath, 0755); err != nil {
		return fmt.Errorf("failed to create tenant directory: %w", err)
	}

	// Validate template
	if _, err := template.New(templateID).Funcs(TemplateFuncs).Parse(content); err != nil {
		return fmt.Errorf("invalid template: %w", err)
	}

	// Write file
	templatePath := filepath.Join(tenantPath, templateID+".html")
	if err := os.WriteFile(templatePath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write template: %w", err)
	}

	// Invalidate cache
	r.InvalidateCache(tenantID)

	return nil
}
