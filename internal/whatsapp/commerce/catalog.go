package commerce

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"
)

// CatalogClient manages interactions with Facebook Commerce Manager catalogs
type CatalogClient struct {
	httpClient    *http.Client
	accessToken   string
	businessID    string
	phoneNumberID string
	apiVersion    string
	baseURL       string
}

// CatalogClientConfig represents configuration for the catalog client
type CatalogClientConfig struct {
	AccessToken   string
	BusinessID    string
	PhoneNumberID string
	APIVersion    string
}

// NewCatalogClient creates a new catalog client
func NewCatalogClient(config *CatalogClientConfig) *CatalogClient {
	apiVersion := config.APIVersion
	if apiVersion == "" {
		apiVersion = "v21.0"
	}

	return &CatalogClient{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		accessToken:   config.AccessToken,
		businessID:    config.BusinessID,
		phoneNumberID: config.PhoneNumberID,
		apiVersion:    apiVersion,
		baseURL:       "https://graph.facebook.com",
	}
}

// buildURL builds the API URL
func (c *CatalogClient) buildURL(path string) string {
	return fmt.Sprintf("%s/%s%s", c.baseURL, c.apiVersion, path)
}

// doRequest executes an HTTP request
func (c *CatalogClient) doRequest(ctx context.Context, method, url string, body interface{}) ([]byte, error) {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.accessToken)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode >= 400 {
		var errResp struct {
			Error struct {
				Message string `json:"message"`
				Code    int    `json:"code"`
			} `json:"error"`
		}
		json.Unmarshal(respBody, &errResp)
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, errResp.Error.Message)
	}

	return respBody, nil
}

// =============================================================================
// Catalog Management
// =============================================================================

// ListCatalogs lists all catalogs for the business
func (c *CatalogClient) ListCatalogs(ctx context.Context) ([]Catalog, error) {
	url := c.buildURL(fmt.Sprintf("/%s/owned_product_catalogs", c.businessID))
	url += "?fields=id,name,product_count,is_catalog_segment,vertical"

	respBody, err := c.doRequest(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	var result struct {
		Data []struct {
			ID           string `json:"id"`
			Name         string `json:"name"`
			ProductCount int    `json:"product_count"`
			Vertical     string `json:"vertical"`
		} `json:"data"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	catalogs := make([]Catalog, len(result.Data))
	for i, cat := range result.Data {
		catalogs[i] = Catalog{
			ID:           cat.ID,
			Name:         cat.Name,
			BusinessID:   c.businessID,
			ProductCount: cat.ProductCount,
			VerticalType: cat.Vertical,
		}
	}

	return catalogs, nil
}

// GetCatalog retrieves a specific catalog by ID
func (c *CatalogClient) GetCatalog(ctx context.Context, catalogID string) (*Catalog, error) {
	url := c.buildURL(fmt.Sprintf("/%s", catalogID))
	url += "?fields=id,name,product_count,vertical"

	respBody, err := c.doRequest(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	var result struct {
		ID           string `json:"id"`
		Name         string `json:"name"`
		ProductCount int    `json:"product_count"`
		Vertical     string `json:"vertical"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &Catalog{
		ID:           result.ID,
		Name:         result.Name,
		BusinessID:   c.businessID,
		ProductCount: result.ProductCount,
		VerticalType: result.Vertical,
	}, nil
}

// =============================================================================
// Product Management
// =============================================================================

// ListProducts lists products from a catalog
func (c *CatalogClient) ListProducts(ctx context.Context, catalogID string, limit int, after string) (*ProductListResponse, error) {
	apiURL := c.buildURL(fmt.Sprintf("/%s/products", catalogID))

	params := url.Values{}
	params.Set("fields", "id,retailer_id,name,description,price,currency,image_url,url,availability,condition,brand,category")
	if limit > 0 {
		params.Set("limit", fmt.Sprintf("%d", limit))
	}
	if after != "" {
		params.Set("after", after)
	}
	apiURL += "?" + params.Encode()

	respBody, err := c.doRequest(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, err
	}

	var result struct {
		Data   []Product `json:"data"`
		Paging *Paging   `json:"paging"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &ProductListResponse{
		Products: result.Data,
		Paging:   result.Paging,
	}, nil
}

// GetProduct retrieves a specific product by ID
func (c *CatalogClient) GetProduct(ctx context.Context, productID string) (*Product, error) {
	apiURL := c.buildURL(fmt.Sprintf("/%s", productID))
	apiURL += "?fields=id,retailer_id,name,description,price,currency,image_url,url,availability,condition,brand,category"

	respBody, err := c.doRequest(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, err
	}

	var product Product
	if err := json.Unmarshal(respBody, &product); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &product, nil
}

// SearchProducts searches for products in a catalog
func (c *CatalogClient) SearchProducts(ctx context.Context, catalogID, query string, limit int) ([]Product, error) {
	apiURL := c.buildURL(fmt.Sprintf("/%s/products", catalogID))

	params := url.Values{}
	params.Set("fields", "id,retailer_id,name,description,price,currency,image_url,availability")

	// Build filter JSON safely using json.Marshal to escape special characters
	filterObj := map[string]map[string]string{
		"name": {"i_contains": query},
	}
	filterJSON, err := json.Marshal(filterObj)
	if err != nil {
		return nil, fmt.Errorf("failed to build filter: %w", err)
	}
	params.Set("filter", string(filterJSON))

	if limit > 0 {
		params.Set("limit", fmt.Sprintf("%d", limit))
	}
	apiURL += "?" + params.Encode()

	respBody, err := c.doRequest(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, err
	}

	var result struct {
		Data []Product `json:"data"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return result.Data, nil
}

// =============================================================================
// Product Messages
// =============================================================================

// SendSingleProductMessage sends a single product message
func (c *CatalogClient) SendSingleProductMessage(ctx context.Context, msg *SingleProductMessage) (string, error) {
	apiURL := c.buildURL(fmt.Sprintf("/%s/messages", c.phoneNumberID))

	body := map[string]interface{}{
		"messaging_product": "whatsapp",
		"recipient_type":    "individual",
		"to":                msg.To,
		"type":              "interactive",
		"interactive": map[string]interface{}{
			"type": "product",
			"body": map[string]string{
				"text": msg.BodyText,
			},
			"action": map[string]interface{}{
				"catalog_id":          msg.CatalogID,
				"product_retailer_id": msg.ProductID,
			},
		},
	}

	if msg.FooterText != "" {
		body["interactive"].(map[string]interface{})["footer"] = map[string]string{
			"text": msg.FooterText,
		}
	}

	respBody, err := c.doRequest(ctx, http.MethodPost, apiURL, body)
	if err != nil {
		return "", err
	}

	var result struct {
		Messages []struct {
			ID string `json:"id"`
		} `json:"messages"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if len(result.Messages) == 0 {
		return "", fmt.Errorf("no message ID in response")
	}

	return result.Messages[0].ID, nil
}

// SendMultiProductMessage sends a multi-product message (up to 30 products)
func (c *CatalogClient) SendMultiProductMessage(ctx context.Context, msg *MultiProductMessage) (string, error) {
	// Validate product count
	totalProducts := 0
	for _, section := range msg.Sections {
		totalProducts += len(section.ProductIDs)
	}
	if totalProducts > 30 {
		return "", fmt.Errorf("multi-product message can contain at most 30 products, got %d", totalProducts)
	}
	if totalProducts == 0 {
		return "", fmt.Errorf("multi-product message must contain at least 1 product")
	}

	apiURL := c.buildURL(fmt.Sprintf("/%s/messages", c.phoneNumberID))

	// Build sections
	sections := make([]map[string]interface{}, len(msg.Sections))
	for i, section := range msg.Sections {
		productItems := make([]map[string]string, len(section.ProductIDs))
		for j, pid := range section.ProductIDs {
			productItems[j] = map[string]string{
				"product_retailer_id": pid,
			}
		}
		sections[i] = map[string]interface{}{
			"title":         section.Title,
			"product_items": productItems,
		}
	}

	body := map[string]interface{}{
		"messaging_product": "whatsapp",
		"recipient_type":    "individual",
		"to":                msg.To,
		"type":              "interactive",
		"interactive": map[string]interface{}{
			"type": "product_list",
			"header": map[string]string{
				"type": "text",
				"text": msg.HeaderText,
			},
			"body": map[string]string{
				"text": msg.BodyText,
			},
			"action": map[string]interface{}{
				"catalog_id": msg.CatalogID,
				"sections":   sections,
			},
		},
	}

	if msg.FooterText != "" {
		body["interactive"].(map[string]interface{})["footer"] = map[string]string{
			"text": msg.FooterText,
		}
	}

	respBody, err := c.doRequest(ctx, http.MethodPost, apiURL, body)
	if err != nil {
		return "", err
	}

	var result struct {
		Messages []struct {
			ID string `json:"id"`
		} `json:"messages"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if len(result.Messages) == 0 {
		return "", fmt.Errorf("no message ID in response")
	}

	return result.Messages[0].ID, nil
}

// SendCatalogMessage sends a catalog message (shows entire catalog)
func (c *CatalogClient) SendCatalogMessage(ctx context.Context, msg *CatalogMessage) (string, error) {
	apiURL := c.buildURL(fmt.Sprintf("/%s/messages", c.phoneNumberID))

	action := map[string]interface{}{
		"name": "catalog_message",
	}
	if msg.ThumbnailProductID != "" {
		action["parameters"] = map[string]string{
			"thumbnail_product_retailer_id": msg.ThumbnailProductID,
		}
	}

	body := map[string]interface{}{
		"messaging_product": "whatsapp",
		"recipient_type":    "individual",
		"to":                msg.To,
		"type":              "interactive",
		"interactive": map[string]interface{}{
			"type": "catalog_message",
			"body": map[string]string{
				"text": msg.BodyText,
			},
			"action": action,
		},
	}

	if msg.FooterText != "" {
		body["interactive"].(map[string]interface{})["footer"] = map[string]string{
			"text": msg.FooterText,
		}
	}

	respBody, err := c.doRequest(ctx, http.MethodPost, apiURL, body)
	if err != nil {
		return "", err
	}

	var result struct {
		Messages []struct {
			ID string `json:"id"`
		} `json:"messages"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if len(result.Messages) == 0 {
		return "", fmt.Errorf("no message ID in response")
	}

	return result.Messages[0].ID, nil
}

// =============================================================================
// Catalog Cache
// =============================================================================

// CatalogCache provides an in-memory cache for catalog data
type CatalogCache struct {
	mu       sync.RWMutex
	products map[string]*Product         // key: product_id
	catalogs map[string]*Catalog         // key: catalog_id
	ttl      time.Duration
	expiry   map[string]time.Time        // key: id, value: expiry time
}

// NewCatalogCache creates a new catalog cache
func NewCatalogCache(ttl time.Duration) *CatalogCache {
	return &CatalogCache{
		products: make(map[string]*Product),
		catalogs: make(map[string]*Catalog),
		ttl:      ttl,
		expiry:   make(map[string]time.Time),
	}
}

// GetProduct retrieves a product from cache
func (cc *CatalogCache) GetProduct(productID string) (*Product, bool) {
	cc.mu.RLock()
	defer cc.mu.RUnlock()

	if expiry, ok := cc.expiry["product:"+productID]; ok {
		if time.Now().After(expiry) {
			return nil, false
		}
	}

	product, ok := cc.products[productID]
	return product, ok
}

// SetProduct stores a product in cache
func (cc *CatalogCache) SetProduct(product *Product) {
	cc.mu.Lock()
	defer cc.mu.Unlock()

	cc.products[product.ID] = product
	cc.expiry["product:"+product.ID] = time.Now().Add(cc.ttl)
}

// GetCatalog retrieves a catalog from cache
func (cc *CatalogCache) GetCatalog(catalogID string) (*Catalog, bool) {
	cc.mu.RLock()
	defer cc.mu.RUnlock()

	if expiry, ok := cc.expiry["catalog:"+catalogID]; ok {
		if time.Now().After(expiry) {
			return nil, false
		}
	}

	catalog, ok := cc.catalogs[catalogID]
	return catalog, ok
}

// SetCatalog stores a catalog in cache
func (cc *CatalogCache) SetCatalog(catalog *Catalog) {
	cc.mu.Lock()
	defer cc.mu.Unlock()

	cc.catalogs[catalog.ID] = catalog
	cc.expiry["catalog:"+catalog.ID] = time.Now().Add(cc.ttl)
}

// Invalidate removes an item from cache
func (cc *CatalogCache) Invalidate(key string) {
	cc.mu.Lock()
	defer cc.mu.Unlock()

	delete(cc.products, key)
	delete(cc.catalogs, key)
	delete(cc.expiry, "product:"+key)
	delete(cc.expiry, "catalog:"+key)
}

// Clear clears the entire cache
func (cc *CatalogCache) Clear() {
	cc.mu.Lock()
	defer cc.mu.Unlock()

	cc.products = make(map[string]*Product)
	cc.catalogs = make(map[string]*Catalog)
	cc.expiry = make(map[string]time.Time)
}

// Cleanup removes expired entries
func (cc *CatalogCache) Cleanup() {
	cc.mu.Lock()
	defer cc.mu.Unlock()

	now := time.Now()
	for key, expiry := range cc.expiry {
		if now.After(expiry) {
			// Extract the actual ID from "product:id" or "catalog:id"
			if len(key) > 8 && key[:8] == "product:" {
				delete(cc.products, key[8:])
			} else if len(key) > 8 && key[:8] == "catalog:" {
				delete(cc.catalogs, key[8:])
			}
			delete(cc.expiry, key)
		}
	}
}

// =============================================================================
// Catalog Sync Service
// =============================================================================

// CatalogSyncer handles catalog synchronization
type CatalogSyncer struct {
	client    *CatalogClient
	cache     *CatalogCache
	mu        sync.RWMutex
	status    map[string]*CatalogSyncStatus
}

// NewCatalogSyncer creates a new catalog syncer
func NewCatalogSyncer(client *CatalogClient, cache *CatalogCache) *CatalogSyncer {
	return &CatalogSyncer{
		client: client,
		cache:  cache,
		status: make(map[string]*CatalogSyncStatus),
	}
}

// SyncCatalog syncs a catalog from Facebook Commerce Manager
func (cs *CatalogSyncer) SyncCatalog(ctx context.Context, catalogID string) error {
	cs.mu.Lock()
	cs.status[catalogID] = &CatalogSyncStatus{
		CatalogID:  catalogID,
		Status:     "syncing",
		LastSyncAt: time.Now(),
	}
	cs.mu.Unlock()

	// Get catalog info
	catalog, err := cs.client.GetCatalog(ctx, catalogID)
	if err != nil {
		cs.updateStatus(catalogID, "error", 0, err.Error())
		return err
	}
	cs.cache.SetCatalog(catalog)

	// Sync products
	productsSynced := 0
	after := ""
	for {
		resp, err := cs.client.ListProducts(ctx, catalogID, 100, after)
		if err != nil {
			cs.updateStatus(catalogID, "error", productsSynced, err.Error())
			return err
		}

		for _, product := range resp.Products {
			p := product // Create copy for pointer
			cs.cache.SetProduct(&p)
			productsSynced++
		}

		if resp.Paging == nil || resp.Paging.Next == "" {
			break
		}
		after = resp.Paging.Cursors.After
	}

	cs.updateStatus(catalogID, "synced", productsSynced, "")
	return nil
}

// updateStatus updates the sync status
func (cs *CatalogSyncer) updateStatus(catalogID, status string, productsSynced int, errorMsg string) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	if s, ok := cs.status[catalogID]; ok {
		s.Status = status
		s.ProductsSynced = productsSynced
		s.ErrorMessage = errorMsg
		s.LastSyncAt = time.Now()
	}
}

// GetSyncStatus returns the sync status for a catalog
func (cs *CatalogSyncer) GetSyncStatus(catalogID string) (*CatalogSyncStatus, bool) {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	status, ok := cs.status[catalogID]
	return status, ok
}
