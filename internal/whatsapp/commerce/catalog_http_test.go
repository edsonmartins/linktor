package commerce

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// rewriteTransport redirects outbound Graph API calls to the test server.
type rewriteTransport struct {
	baseURL string
	rt      http.RoundTripper
}

func (t *rewriteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	host := strings.TrimPrefix(t.baseURL, "http://")
	req.URL.Scheme = "http"
	req.URL.Host = host
	return t.rt.RoundTrip(req)
}

func newTestCatalogClient(handler http.HandlerFunc) (*CatalogClient, *httptest.Server) {
	server := httptest.NewServer(handler)
	c := NewCatalogClient(&CatalogClientConfig{
		AccessToken:   "test-token",
		BusinessID:    "biz-77",
		PhoneNumberID: "phone-55",
		APIVersion:    "v23.0",
	})
	c.httpClient = &http.Client{Transport: &rewriteTransport{baseURL: server.URL, rt: http.DefaultTransport}}
	return c, server
}

func newTestOrderManager(handler http.HandlerFunc) (*OrderManager, *httptest.Server) {
	server := httptest.NewServer(handler)
	om := NewOrderManager(&OrderManagerConfig{
		AccessToken:   "test-token",
		PhoneNumberID: "phone-55",
		APIVersion:    "v23.0",
	})
	om.httpClient = &http.Client{Transport: &rewriteTransport{baseURL: server.URL, rt: http.DefaultTransport}}
	return om, server
}

func newTestCartManager(handler http.HandlerFunc) (*CartManager, *httptest.Server) {
	server := httptest.NewServer(handler)
	cm := NewCartManager(&CartManagerConfig{
		AccessToken:   "test-token",
		PhoneNumberID: "phone-55",
		APIVersion:    "v23.0",
	})
	cm.httpClient = &http.Client{Transport: &rewriteTransport{baseURL: server.URL, rt: http.DefaultTransport}}
	return cm, server
}

// -----------------------------------------------------------------------------
// ListCatalogs / GetCatalog
// -----------------------------------------------------------------------------

func TestCatalogClient_ListCatalogs_Success(t *testing.T) {
	var capturedPath, capturedQuery string
	client, server := newTestCatalogClient(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		capturedQuery = r.URL.RawQuery
		_, _ = w.Write([]byte(`{"data":[{"id":"cat-1","name":"Main","product_count":10,"vertical":"ecommerce"}]}`))
	})
	defer server.Close()

	catalogs, err := client.ListCatalogs(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "/v23.0/biz-77/owned_product_catalogs", capturedPath)
	assert.Contains(t, capturedQuery, "fields=")
	require.Len(t, catalogs, 1)
	assert.Equal(t, "cat-1", catalogs[0].ID)
	assert.Equal(t, 10, catalogs[0].ProductCount)
}

func TestCatalogClient_ListCatalogs_APIError(t *testing.T) {
	client, server := newTestCatalogClient(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"error":{"message":"permission denied","code":10}}`))
	})
	defer server.Close()

	_, err := client.ListCatalogs(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "403")
}

func TestCatalogClient_GetCatalog_Success(t *testing.T) {
	var capturedPath string
	client, server := newTestCatalogClient(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		_, _ = w.Write([]byte(`{"id":"cat-1","name":"Main","product_count":5,"vertical":"ecommerce"}`))
	})
	defer server.Close()

	cat, err := client.GetCatalog(context.Background(), "cat-1")
	require.NoError(t, err)
	assert.Equal(t, "/v23.0/cat-1", capturedPath)
	assert.Equal(t, "Main", cat.Name)
}

// -----------------------------------------------------------------------------
// ListProducts / GetProduct / SearchProducts
// -----------------------------------------------------------------------------

func TestCatalogClient_ListProducts_Success(t *testing.T) {
	var capturedPath, capturedQuery string
	client, server := newTestCatalogClient(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		capturedQuery = r.URL.RawQuery
		_, _ = w.Write([]byte(`{"data":[{"id":"p1","name":"Shirt"}],"paging":{"cursors":{"after":"cur"}}}`))
	})
	defer server.Close()

	resp, err := client.ListProducts(context.Background(), "cat-1", 25, "")
	require.NoError(t, err)
	assert.Equal(t, "/v23.0/cat-1/products", capturedPath)
	assert.Contains(t, capturedQuery, "limit=25")
	assert.Contains(t, capturedQuery, "fields=")
	require.Len(t, resp.Products, 1)
	assert.Equal(t, "p1", resp.Products[0].ID)
}

func TestCatalogClient_GetProduct_Success(t *testing.T) {
	var capturedPath string
	client, server := newTestCatalogClient(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		_, _ = w.Write([]byte(`{"id":"p1","name":"Shirt","price":2999,"currency":"USD"}`))
	})
	defer server.Close()

	p, err := client.GetProduct(context.Background(), "p1")
	require.NoError(t, err)
	assert.Equal(t, "/v23.0/p1", capturedPath)
	assert.Equal(t, "Shirt", p.Name)
}

func TestCatalogClient_SearchProducts_Success(t *testing.T) {
	var capturedPath, capturedQuery string
	client, server := newTestCatalogClient(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		capturedQuery = r.URL.RawQuery
		_, _ = w.Write([]byte(`{"data":[{"id":"p1","name":"Red shirt"}]}`))
	})
	defer server.Close()

	products, err := client.SearchProducts(context.Background(), "cat-1", "shirt", 10)
	require.NoError(t, err)
	assert.Equal(t, "/v23.0/cat-1/products", capturedPath)
	assert.Contains(t, capturedQuery, "filter=")
	assert.Contains(t, capturedQuery, "i_contains")
	assert.Contains(t, capturedQuery, "limit=10")
	require.Len(t, products, 1)
}

// -----------------------------------------------------------------------------
// Product Messages
// -----------------------------------------------------------------------------

func TestCatalogClient_SendSingleProductMessage_Success(t *testing.T) {
	var capturedPath, capturedMethod string
	var captured map[string]interface{}

	client, server := newTestCatalogClient(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		capturedMethod = r.Method
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &captured)
		_, _ = w.Write([]byte(`{"messages":[{"id":"wamid.single-1"}]}`))
	})
	defer server.Close()

	id, err := client.SendSingleProductMessage(context.Background(), &SingleProductMessage{
		To:         "+15551234567",
		CatalogID:  "cat-1",
		ProductID:  "sku-7",
		BodyText:   "Check this out",
		FooterText: "Free shipping",
	})
	require.NoError(t, err)
	assert.Equal(t, "wamid.single-1", id)
	assert.Equal(t, http.MethodPost, capturedMethod)
	assert.Equal(t, "/v23.0/phone-55/messages", capturedPath)

	assert.Equal(t, "whatsapp", captured["messaging_product"])
	assert.Equal(t, "interactive", captured["type"])
	inter := captured["interactive"].(map[string]interface{})
	assert.Equal(t, "product", inter["type"])
	action := inter["action"].(map[string]interface{})
	assert.Equal(t, "cat-1", action["catalog_id"])
	assert.Equal(t, "sku-7", action["product_retailer_id"])
}

func TestCatalogClient_SendSingleProductMessage_NoMessageID(t *testing.T) {
	client, server := newTestCatalogClient(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"messages":[]}`))
	})
	defer server.Close()

	_, err := client.SendSingleProductMessage(context.Background(), &SingleProductMessage{
		To: "+15551234567", CatalogID: "c", ProductID: "p", BodyText: "x",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no message ID")
}

func TestCatalogClient_SendMultiProductMessage_Success(t *testing.T) {
	var capturedPath string
	var captured map[string]interface{}

	client, server := newTestCatalogClient(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &captured)
		_, _ = w.Write([]byte(`{"messages":[{"id":"wamid.multi-1"}]}`))
	})
	defer server.Close()

	id, err := client.SendMultiProductMessage(context.Background(), &MultiProductMessage{
		To:         "+15551234567",
		CatalogID:  "cat-1",
		HeaderText: "Top picks",
		BodyText:   "Our best selling items",
		Sections: []ProductSection{
			{Title: "Shirts", ProductIDs: []string{"p1", "p2"}},
			{Title: "Pants", ProductIDs: []string{"p3"}},
		},
	})
	require.NoError(t, err)
	assert.Equal(t, "wamid.multi-1", id)
	assert.Equal(t, "/v23.0/phone-55/messages", capturedPath)

	inter := captured["interactive"].(map[string]interface{})
	assert.Equal(t, "product_list", inter["type"])
	action := inter["action"].(map[string]interface{})
	sections := action["sections"].([]interface{})
	require.Len(t, sections, 2)
}

func TestCatalogClient_SendMultiProductMessage_TooManyProducts(t *testing.T) {
	client, _ := newTestCatalogClient(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called when validation fails")
	})

	sections := make([]ProductSection, 0)
	// 31 products across sections
	section := ProductSection{Title: "S", ProductIDs: make([]string, 31)}
	for i := range section.ProductIDs {
		section.ProductIDs[i] = "p"
	}
	sections = append(sections, section)

	_, err := client.SendMultiProductMessage(context.Background(), &MultiProductMessage{
		To: "+15551234567", CatalogID: "c", BodyText: "x", Sections: sections,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "at most 30 products")
}

func TestCatalogClient_SendMultiProductMessage_Empty(t *testing.T) {
	client, _ := newTestCatalogClient(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called when validation fails")
	})

	_, err := client.SendMultiProductMessage(context.Background(), &MultiProductMessage{
		To: "+15551234567", CatalogID: "c", BodyText: "x",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "at least 1 product")
}

func TestCatalogClient_SendCatalogMessage_Success(t *testing.T) {
	var capturedPath string
	var captured map[string]interface{}

	client, server := newTestCatalogClient(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &captured)
		_, _ = w.Write([]byte(`{"messages":[{"id":"wamid.cat-1"}]}`))
	})
	defer server.Close()

	id, err := client.SendCatalogMessage(context.Background(), &CatalogMessage{
		To:                 "+15551234567",
		BodyText:           "See our catalog",
		FooterText:         "Limited time",
		ThumbnailProductID: "sku-9",
	})
	require.NoError(t, err)
	assert.Equal(t, "wamid.cat-1", id)
	assert.Equal(t, "/v23.0/phone-55/messages", capturedPath)

	inter := captured["interactive"].(map[string]interface{})
	assert.Equal(t, "catalog_message", inter["type"])
	action := inter["action"].(map[string]interface{})
	assert.Equal(t, "catalog_message", action["name"])
	params := action["parameters"].(map[string]interface{})
	assert.Equal(t, "sku-9", params["thumbnail_product_retailer_id"])
}

// -----------------------------------------------------------------------------
// Order HTTP
// -----------------------------------------------------------------------------

func TestOrderManager_SendOrderConfirmation_Success(t *testing.T) {
	var capturedPath string
	var captured map[string]interface{}

	om, server := newTestOrderManager(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &captured)
		_, _ = w.Write([]byte(`{"messages":[{"id":"wamid.order-1"}]}`))
	})
	defer server.Close()

	id, err := om.SendOrderConfirmation(context.Background(), "+15551234567", &OrderConfirmation{
		OrderID:           "ord-1",
		Status:            "confirmed",
		Description:       "Thanks!",
		EstimatedDelivery: "2026-04-25",
		TrackingNumber:    "TRK-99",
	})
	require.NoError(t, err)
	assert.Equal(t, "wamid.order-1", id)
	assert.Equal(t, "/v23.0/phone-55/messages", capturedPath)

	text := captured["text"].(map[string]interface{})["body"].(string)
	assert.Contains(t, text, "ord-1")
	assert.Contains(t, text, "confirmed")
	assert.Contains(t, text, "TRK-99")
}

func TestOrderManager_SendOrderConfirmation_APIError(t *testing.T) {
	om, server := newTestOrderManager(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"error":{"message":"rate limited","code":131056}}`))
	})
	defer server.Close()

	_, err := om.SendOrderConfirmation(context.Background(), "+15551234567", &OrderConfirmation{OrderID: "o", Status: "s"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "429")
}

func TestOrderManager_SendOrderStatusUpdate_Success(t *testing.T) {
	var captured map[string]interface{}
	om, server := newTestOrderManager(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &captured)
		_, _ = w.Write([]byte(`{"messages":[{"id":"wamid.status-1"}]}`))
	})
	defer server.Close()

	id, err := om.SendOrderStatusUpdate(context.Background(), &OrderDetailsMessage{
		To:            "+15551234567",
		Description:   "Status updated",
		ReferenceID:   "ref-1",
		Status:        "shipped",
		PaymentStatus: "paid",
	})
	require.NoError(t, err)
	assert.Equal(t, "wamid.status-1", id)

	inter := captured["interactive"].(map[string]interface{})
	assert.Equal(t, "order_status", inter["type"])
	action := inter["action"].(map[string]interface{})
	params := action["parameters"].(map[string]interface{})
	assert.Equal(t, "ref-1", params["reference_id"])
	assert.Equal(t, "shipped", params["order_status"])
	assert.Equal(t, "paid", params["payment_status"])
}

// -----------------------------------------------------------------------------
// Cart HTTP (only path is sendTextMessage via SendCartSummary / abandoned reminder)
// -----------------------------------------------------------------------------

func TestCartManager_SendCartSummary_Success(t *testing.T) {
	var capturedPath string
	var captured map[string]interface{}

	cm, server := newTestCartManager(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &captured)
		_, _ = w.Write([]byte(`{"messages":[{"id":"wamid.cart-1"}]}`))
	})
	defer server.Close()

	cart := cm.GetOrCreateCart("+15551234567", "cat-1")
	_ = cm.AddItem("+15551234567", CartItem{ProductID: "p1", Quantity: 2, UnitPrice: 1000, Currency: "USD"})
	_ = cart // ensure created

	id, err := cm.SendCartSummary(context.Background(), "+15551234567")
	require.NoError(t, err)
	assert.Equal(t, "wamid.cart-1", id)
	assert.Equal(t, "/v23.0/phone-55/messages", capturedPath)
	assert.Equal(t, "whatsapp", captured["messaging_product"])
	assert.Equal(t, "text", captured["type"])
}

func TestCartManager_SendCartSummary_CartNotFound(t *testing.T) {
	cm, _ := newTestCartManager(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called — cart doesn't exist")
	})

	_, err := cm.SendCartSummary(context.Background(), "+15550000000")
	require.Error(t, err)
}

func TestCartManager_SendAbandonedCartReminder_Success(t *testing.T) {
	var captured map[string]interface{}
	cm, server := newTestCartManager(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &captured)
		_, _ = w.Write([]byte(`{"messages":[{"id":"wamid.abandon-1"}]}`))
	})
	defer server.Close()

	cart := cm.GetOrCreateCart("+15551234567", "cat-1")
	_ = cm.AddItem("+15551234567", CartItem{ProductID: "p1", Quantity: 1, UnitPrice: 500, Currency: "USD"})

	id, err := cm.SendAbandonedCartReminder(context.Background(), cart)
	require.NoError(t, err)
	assert.Equal(t, "wamid.abandon-1", id)
	text := captured["text"].(map[string]interface{})["body"].(string)
	assert.Contains(t, strings.ToLower(text), "cart")
}
