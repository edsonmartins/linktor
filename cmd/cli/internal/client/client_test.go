package client

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// newTestClient creates a Client pointing at the given test server URL with an API key.
func newTestClient(serverURL, apiKey string) *Client {
	viper.Set("base_url", serverURL)
	c, _ := NewWithAPIKey(apiKey)
	return c
}

// newTestClientAnonymous creates a Client with no auth pointing at the test server.
func newTestClientAnonymous(serverURL string) *Client {
	viper.Set("base_url", serverURL)
	c, _ := NewAnonymous()
	return c
}

// readBody is a small helper that reads and returns the request body as string.
func readBody(r *http.Request) string {
	b, _ := io.ReadAll(r.Body)
	return string(b)
}

// writeJSON writes a JSON response to the http.ResponseWriter.
func writeJSON(w http.ResponseWriter, code int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}

// wrapped returns the standard API wrapper {"success":true,"data":...}.
func wrapped(data interface{}) map[string]interface{} {
	raw, _ := json.Marshal(data)
	return map[string]interface{}{
		"success": true,
		"data":    json.RawMessage(raw),
	}
}

// ---------------------------------------------------------------------------
// 1. Data types & JSON marshaling
// ---------------------------------------------------------------------------

func TestTypesJSONRoundTrip(t *testing.T) {
	t.Run("User", func(t *testing.T) {
		u := User{ID: "u1", Email: "a@b.com", Name: "Alice", Role: "admin", TenantID: "t1", TenantName: "Acme"}
		data, err := json.Marshal(u)
		require.NoError(t, err)
		var decoded User
		require.NoError(t, json.Unmarshal(data, &decoded))
		assert.Equal(t, u, decoded)
	})

	t.Run("APIError", func(t *testing.T) {
		e := APIError{Message: "not found", Code: "NOT_FOUND"}
		data, err := json.Marshal(e)
		require.NoError(t, err)
		var decoded APIError
		require.NoError(t, json.Unmarshal(data, &decoded))
		assert.Equal(t, e, decoded)
	})

	t.Run("Channel", func(t *testing.T) {
		ch := Channel{ID: "ch1", Name: "WA", Type: "whatsapp", Status: "connected", Enabled: true}
		data, err := json.Marshal(ch)
		require.NoError(t, err)
		var decoded Channel
		require.NoError(t, json.Unmarshal(data, &decoded))
		assert.Equal(t, ch.ID, decoded.ID)
		assert.Equal(t, ch.Enabled, decoded.Enabled)
	})

	t.Run("FlowValidationResult", func(t *testing.T) {
		fv := FlowValidationResult{Valid: false, Errors: []string{"missing node"}, Warnings: []string{"unused var"}}
		data, err := json.Marshal(fv)
		require.NoError(t, err)
		var decoded FlowValidationResult
		require.NoError(t, json.Unmarshal(data, &decoded))
		assert.Equal(t, fv, decoded)
	})
}

// ---------------------------------------------------------------------------
// 2. NewWithAPIKey / NewAnonymous
// ---------------------------------------------------------------------------

func TestNewWithAPIKey(t *testing.T) {
	viper.Set("base_url", "http://example.com")
	c, err := NewWithAPIKey("test-key-123")
	require.NoError(t, err)
	assert.Equal(t, "http://example.com", c.baseURL)
	assert.Equal(t, "test-key-123", c.apiKey)
	assert.Empty(t, c.accessToken)
	assert.NotNil(t, c.httpClient)
}

func TestNewAnonymous(t *testing.T) {
	viper.Set("base_url", "http://example.com")
	c, err := NewAnonymous()
	require.NoError(t, err)
	assert.Equal(t, "http://example.com", c.baseURL)
	assert.Empty(t, c.apiKey)
	assert.Empty(t, c.accessToken)
	assert.NotNil(t, c.httpClient)
}

// ---------------------------------------------------------------------------
// 3. buildQuery
// ---------------------------------------------------------------------------

func TestBuildQuery(t *testing.T) {
	t.Run("empty map", func(t *testing.T) {
		assert.Equal(t, "", buildQuery(map[string]string{}))
	})

	t.Run("single param", func(t *testing.T) {
		assert.Equal(t, "status=open", buildQuery(map[string]string{"status": "open"}))
	})

	t.Run("skips empty values", func(t *testing.T) {
		q := buildQuery(map[string]string{"a": "1", "b": ""})
		assert.Equal(t, "a=1", q)
	})

	t.Run("multiple params", func(t *testing.T) {
		q := buildQuery(map[string]string{"a": "1", "b": "2"})
		// Map iteration is non-deterministic, so sort and check
		parts := strings.Split(q, "&")
		sort.Strings(parts)
		assert.Equal(t, []string{"a=1", "b=2"}, parts)
	})
}

// ---------------------------------------------------------------------------
// 4. joinStrings
// ---------------------------------------------------------------------------

func TestJoinStrings(t *testing.T) {
	assert.Equal(t, "", joinStrings(nil, "&"))
	assert.Equal(t, "a", joinStrings([]string{"a"}, "&"))
	assert.Equal(t, "a&b&c", joinStrings([]string{"a", "b", "c"}, "&"))
	assert.Equal(t, "x,y", joinStrings([]string{"x", "y"}, ","))
}

// ---------------------------------------------------------------------------
// 5. handleResponse
// ---------------------------------------------------------------------------

func TestHandleResponse(t *testing.T) {
	c := &Client{}

	t.Run("success with wrapped response", func(t *testing.T) {
		user := User{ID: "u1", Name: "Alice"}
		body := wrapped(user)
		data, _ := json.Marshal(body)

		resp := &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader(string(data))),
		}

		var result User
		err := c.handleResponse(resp, &result)
		require.NoError(t, err)
		assert.Equal(t, "u1", result.ID)
		assert.Equal(t, "Alice", result.Name)
	})

	t.Run("success with direct response", func(t *testing.T) {
		user := User{ID: "u2", Name: "Bob"}
		data, _ := json.Marshal(user)

		resp := &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader(string(data))),
		}

		var result User
		err := c.handleResponse(resp, &result)
		require.NoError(t, err)
		assert.Equal(t, "u2", result.ID)
	})

	t.Run("nil result pointer", func(t *testing.T) {
		resp := &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader(`{}`)),
		}
		err := c.handleResponse(resp, nil)
		assert.NoError(t, err)
	})

	t.Run("error response", func(t *testing.T) {
		apiErr := APIError{Message: "forbidden", Code: "FORBIDDEN"}
		data, _ := json.Marshal(apiErr)

		resp := &http.Response{
			StatusCode: 403,
			Body:       io.NopCloser(strings.NewReader(string(data))),
		}

		var result User
		err := c.handleResponse(resp, &result)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "forbidden")
	})
}

// ---------------------------------------------------------------------------
// 6. parseError
// ---------------------------------------------------------------------------

func TestParseError(t *testing.T) {
	c := &Client{}

	t.Run("with API error JSON", func(t *testing.T) {
		data, _ := json.Marshal(APIError{Message: "resource not found", Code: "NOT_FOUND"})
		resp := &http.Response{
			StatusCode: 404,
			Body:       io.NopCloser(strings.NewReader(string(data))),
		}
		err := c.parseError(resp)
		assert.EqualError(t, err, "resource not found")
	})

	t.Run("with non-JSON body", func(t *testing.T) {
		resp := &http.Response{
			StatusCode: 502,
			Body:       io.NopCloser(strings.NewReader("Bad Gateway")),
		}
		err := c.parseError(resp)
		assert.EqualError(t, err, "API error: 502")
	})

	t.Run("with empty message in JSON", func(t *testing.T) {
		data, _ := json.Marshal(APIError{Message: "", Code: "UNKNOWN"})
		resp := &http.Response{
			StatusCode: 500,
			Body:       io.NopCloser(strings.NewReader(string(data))),
		}
		err := c.parseError(resp)
		assert.EqualError(t, err, "API error: 500")
	})
}

// ---------------------------------------------------------------------------
// 7. doRequest - auth headers
// ---------------------------------------------------------------------------

func TestDoRequestHeaders(t *testing.T) {
	t.Run("API key header", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "my-api-key", r.Header.Get("X-API-Key"))
			assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
			assert.Empty(t, r.Header.Get("Authorization"))
			w.WriteHeader(200)
		}))
		defer srv.Close()

		c := newTestClient(srv.URL, "my-api-key")
		resp, err := c.doRequest("GET", "/test", nil)
		require.NoError(t, err)
		resp.Body.Close()
		assert.Equal(t, 200, resp.StatusCode)
	})

	t.Run("access token header", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "Bearer tok-123", r.Header.Get("Authorization"))
			assert.Empty(t, r.Header.Get("X-API-Key"))
			w.WriteHeader(200)
		}))
		defer srv.Close()

		viper.Set("base_url", srv.URL)
		c := &Client{
			baseURL:     srv.URL,
			accessToken: "tok-123",
			httpClient:  http.DefaultClient,
		}
		resp, err := c.doRequest("GET", "/test", nil)
		require.NoError(t, err)
		resp.Body.Close()
	})

	t.Run("no auth for anonymous", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Empty(t, r.Header.Get("X-API-Key"))
			assert.Empty(t, r.Header.Get("Authorization"))
			w.WriteHeader(200)
		}))
		defer srv.Close()

		c := newTestClientAnonymous(srv.URL)
		resp, err := c.doRequest("GET", "/test", nil)
		require.NoError(t, err)
		resp.Body.Close()
	})

	t.Run("sends JSON body", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body := readBody(r)
			assert.Contains(t, body, `"key":"value"`)
			w.WriteHeader(200)
		}))
		defer srv.Close()

		c := newTestClient(srv.URL, "k")
		resp, err := c.doRequest("POST", "/test", map[string]string{"key": "value"})
		require.NoError(t, err)
		resp.Body.Close()
	})
}

// ---------------------------------------------------------------------------
// 8. API method tests — shared router
// ---------------------------------------------------------------------------

// apiTestRouter creates a test HTTP server that routes by method+path and
// records each request for inspection.
type requestRecord struct {
	Method string
	Path   string
	Body   string
}

func newAPITestServer(t *testing.T) (*httptest.Server, *[]requestRecord) {
	t.Helper()
	records := &[]requestRecord{}

	mux := http.NewServeMux()

	// Helper to register a route that records the request and responds with wrapped JSON.
	reg := func(pattern string, respData interface{}) {
		mux.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
			body := readBody(r)
			*records = append(*records, requestRecord{
				Method: r.Method,
				Path:   r.URL.Path,
				Body:   body,
			})
			if respData != nil {
				writeJSON(w, 200, wrapped(respData))
			} else {
				w.WriteHeader(204)
			}
		})
	}

	// Auth
	reg("/auth/login", LoginResponse{AccessToken: "tok", ExpiresIn: 3600, User: &User{ID: "u1"}})
	reg("/auth/me", User{ID: "u1", Email: "a@b.com", Name: "Alice"})
	reg("/auth/api-keys", []APIKey{{ID: "k1", Name: "default"}})

	// Channels
	reg("/channels", PaginatedResponse[Channel]{Data: []Channel{{ID: "ch1"}}, Total: 1})
	reg("/channels/ch1", Channel{ID: "ch1", Name: "WhatsApp"})
	reg("/channels/ch1/connect", Channel{ID: "ch1", Status: "connected"})
	reg("/channels/ch1/disconnect", Channel{ID: "ch1", Status: "disconnected"})

	// Conversations
	reg("/conversations", PaginatedResponse[Conversation]{Data: []Conversation{{ID: "cv1"}}, Total: 1})
	reg("/conversations/cv1", Conversation{ID: "cv1", Status: "open"})
	reg("/conversations/cv1/messages", PaginatedResponse[Message]{Data: []Message{{ID: "m1"}}, Total: 1})
	reg("/conversations/cv1/resolve", Conversation{ID: "cv1", Status: "resolved"})
	reg("/conversations/cv1/reopen", Conversation{ID: "cv1", Status: "open"})

	// Contacts
	reg("/contacts", PaginatedResponse[Contact]{Data: []Contact{{ID: "ct1"}}, Total: 1})
	reg("/contacts/ct1", Contact{ID: "ct1", Name: "Alice"})

	// Bots
	reg("/bots", PaginatedResponse[Bot]{Data: []Bot{{ID: "b1"}}, Total: 1})
	reg("/bots/b1", Bot{ID: "b1", Name: "MyBot"})
	reg("/bots/b1/start", Bot{ID: "b1", Status: "running"})
	reg("/bots/b1/stop", Bot{ID: "b1", Status: "stopped"})
	reg("/bots/b1/logs", []BotLog{{Level: "info", Message: "started"}})

	// Flows
	reg("/flows", PaginatedResponse[Flow]{Data: []Flow{{ID: "f1"}}, Total: 1})
	reg("/flows/f1", Flow{ID: "f1", Name: "Welcome"})
	reg("/flows/f1/publish", Flow{ID: "f1", Status: "published"})
	reg("/flows/f1/unpublish", Flow{ID: "f1", Status: "draft"})
	reg("/flows/f1/execute", FlowExecutionResult{ExecutionID: "exec1", Status: "completed"})
	reg("/flows/f1/validate", FlowValidationResult{Valid: true})

	// Knowledge Bases
	reg("/knowledge-bases", PaginatedResponse[KnowledgeBase]{Data: []KnowledgeBase{{ID: "kb1"}}, Total: 1})
	reg("/knowledge-bases/kb1", KnowledgeBase{ID: "kb1", Name: "FAQ"})
	reg("/knowledge-bases/kb1/query", struct {
		Results []QueryResult `json:"results"`
	}{Results: []QueryResult{{Score: 0.95, Title: "FAQ", Content: "answer"}}})
	reg("/knowledge-bases/kb1/documents", PaginatedResponse[Document]{Data: []Document{{ID: "d1"}}, Total: 1})
	reg("/knowledge-bases/kb1/documents/d1", Document{ID: "d1", Title: "Doc1"})
	reg("/knowledge-bases/kb1/documents/d1/reprocess", Document{ID: "d1", Status: "processing"})

	// Webhooks
	reg("/webhooks", PaginatedResponse[Webhook]{Data: []Webhook{{ID: "wh1"}}, Total: 1})
	reg("/webhooks/events", []WebhookEvent{{ID: "we1", EventType: "message.created"}})

	// Direct message
	reg("/api/v1/messages/send", Message{ID: "dm1", Text: "hello"})

	srv := httptest.NewServer(mux)
	return srv, records
}

func lastRecord(records *[]requestRecord) requestRecord {
	recs := *records
	return recs[len(recs)-1]
}

// ---- Auth methods ----

func TestLogin(t *testing.T) {
	srv, records := newAPITestServer(t)
	defer srv.Close()
	c := newTestClientAnonymous(srv.URL)

	result, err := c.Login("a@b.com", "pass")
	require.NoError(t, err)
	assert.Equal(t, "tok", result.AccessToken)
	assert.Equal(t, "u1", result.User.ID)

	rec := lastRecord(records)
	assert.Equal(t, "POST", rec.Method)
	assert.Equal(t, "/auth/login", rec.Path)
	assert.Contains(t, rec.Body, `"email":"a@b.com"`)
	assert.Contains(t, rec.Body, `"password":"pass"`)
}

func TestGetCurrentUser(t *testing.T) {
	srv, records := newAPITestServer(t)
	defer srv.Close()
	c := newTestClient(srv.URL, "key")

	result, err := c.GetCurrentUser()
	require.NoError(t, err)
	assert.Equal(t, "u1", result.ID)
	assert.Equal(t, "Alice", result.Name)
	assert.Equal(t, "GET", lastRecord(records).Method)
}

func TestListAPIKeys(t *testing.T) {
	srv, records := newAPITestServer(t)
	defer srv.Close()
	c := newTestClient(srv.URL, "key")

	result, err := c.ListAPIKeys()
	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, "k1", result[0].ID)
	assert.Equal(t, "GET", lastRecord(records).Method)
}

// ---- Channel methods ----

func TestListChannels(t *testing.T) {
	srv, records := newAPITestServer(t)
	defer srv.Close()
	c := newTestClient(srv.URL, "key")

	result, err := c.ListChannels(nil)
	require.NoError(t, err)
	assert.Equal(t, 1, result.Total)
	assert.Equal(t, "ch1", result.Data[0].ID)
	assert.Equal(t, "GET", lastRecord(records).Method)
	assert.Equal(t, "/channels", lastRecord(records).Path)
}

func TestGetChannel(t *testing.T) {
	srv, records := newAPITestServer(t)
	defer srv.Close()
	c := newTestClient(srv.URL, "key")

	result, err := c.GetChannel("ch1")
	require.NoError(t, err)
	assert.Equal(t, "WhatsApp", result.Name)
	assert.Equal(t, "/channels/ch1", lastRecord(records).Path)
}

func TestCreateChannel(t *testing.T) {
	srv, records := newAPITestServer(t)
	defer srv.Close()
	c := newTestClient(srv.URL, "key")

	input := map[string]interface{}{"name": "NewChannel", "type": "telegram"}
	_, err := c.CreateChannel(input)
	require.NoError(t, err)

	rec := lastRecord(records)
	assert.Equal(t, "POST", rec.Method)
	assert.Equal(t, "/channels", rec.Path)
	assert.Contains(t, rec.Body, `"name":"NewChannel"`)
}

func TestUpdateChannel(t *testing.T) {
	srv, records := newAPITestServer(t)
	defer srv.Close()
	c := newTestClient(srv.URL, "key")

	input := map[string]interface{}{"name": "Updated"}
	result, err := c.UpdateChannel("ch1", input)
	require.NoError(t, err)
	assert.Equal(t, "ch1", result.ID)

	rec := lastRecord(records)
	assert.Equal(t, "PATCH", rec.Method)
	assert.Equal(t, "/channels/ch1", rec.Path)
	assert.Contains(t, rec.Body, `"name":"Updated"`)
}

func TestDeleteChannel(t *testing.T) {
	srv, records := newAPITestServer(t)
	defer srv.Close()
	c := newTestClient(srv.URL, "key")

	err := c.DeleteChannel("ch1")
	require.NoError(t, err)
	rec := lastRecord(records)
	assert.Equal(t, "DELETE", rec.Method)
	assert.Equal(t, "/channels/ch1", rec.Path)
}

func TestConnectChannel(t *testing.T) {
	srv, records := newAPITestServer(t)
	defer srv.Close()
	c := newTestClient(srv.URL, "key")

	result, err := c.ConnectChannel("ch1")
	require.NoError(t, err)
	assert.Equal(t, "connected", result.Status)
	assert.Equal(t, "POST", lastRecord(records).Method)
	assert.Equal(t, "/channels/ch1/connect", lastRecord(records).Path)
}

func TestDisconnectChannel(t *testing.T) {
	srv, records := newAPITestServer(t)
	defer srv.Close()
	c := newTestClient(srv.URL, "key")

	result, err := c.DisconnectChannel("ch1")
	require.NoError(t, err)
	assert.Equal(t, "disconnected", result.Status)
	assert.Equal(t, "POST", lastRecord(records).Method)
}

// ---- Conversation methods ----

func TestListConversations(t *testing.T) {
	srv, records := newAPITestServer(t)
	defer srv.Close()
	c := newTestClient(srv.URL, "key")

	result, err := c.ListConversations(map[string]string{"status": "open"})
	require.NoError(t, err)
	assert.Equal(t, 1, result.Total)
	assert.Equal(t, "GET", lastRecord(records).Method)
}

func TestGetConversation(t *testing.T) {
	srv, records := newAPITestServer(t)
	defer srv.Close()
	c := newTestClient(srv.URL, "key")

	result, err := c.GetConversation("cv1")
	require.NoError(t, err)
	assert.Equal(t, "open", result.Status)
	assert.Equal(t, "/conversations/cv1", lastRecord(records).Path)
}

func TestGetConversationMessages(t *testing.T) {
	srv, records := newAPITestServer(t)
	defer srv.Close()
	c := newTestClient(srv.URL, "key")

	result, err := c.GetConversationMessages("cv1", nil)
	require.NoError(t, err)
	assert.Len(t, result.Data, 1)
	assert.Equal(t, "/conversations/cv1/messages", lastRecord(records).Path)
}

func TestSendMessage(t *testing.T) {
	srv, records := newAPITestServer(t)
	defer srv.Close()
	c := newTestClient(srv.URL, "key")

	input := map[string]interface{}{"text": "hello"}
	_, err := c.SendMessage("cv1", input)
	require.NoError(t, err)

	rec := lastRecord(records)
	assert.Equal(t, "POST", rec.Method)
	assert.Equal(t, "/conversations/cv1/messages", rec.Path)
	assert.Contains(t, rec.Body, `"text":"hello"`)
}

func TestCloseConversation(t *testing.T) {
	srv, records := newAPITestServer(t)
	defer srv.Close()
	c := newTestClient(srv.URL, "key")

	result, err := c.CloseConversation("cv1")
	require.NoError(t, err)
	assert.Equal(t, "resolved", result.Status)
	assert.Equal(t, "POST", lastRecord(records).Method)
	assert.Equal(t, "/conversations/cv1/resolve", lastRecord(records).Path)
}

func TestReopenConversation(t *testing.T) {
	srv, records := newAPITestServer(t)
	defer srv.Close()
	c := newTestClient(srv.URL, "key")

	result, err := c.ReopenConversation("cv1")
	require.NoError(t, err)
	assert.Equal(t, "open", result.Status)
	assert.Equal(t, "/conversations/cv1/reopen", lastRecord(records).Path)
}

// ---- Contact methods ----

func TestListContacts(t *testing.T) {
	srv, records := newAPITestServer(t)
	defer srv.Close()
	c := newTestClient(srv.URL, "key")

	result, err := c.ListContacts(nil)
	require.NoError(t, err)
	assert.Equal(t, 1, result.Total)
	assert.Equal(t, "GET", lastRecord(records).Method)
	assert.Equal(t, "/contacts", lastRecord(records).Path)
}

func TestGetContact(t *testing.T) {
	srv, records := newAPITestServer(t)
	defer srv.Close()
	c := newTestClient(srv.URL, "key")

	result, err := c.GetContact("ct1")
	require.NoError(t, err)
	assert.Equal(t, "Alice", result.Name)
	assert.Equal(t, "/contacts/ct1", lastRecord(records).Path)
}

func TestCreateContact(t *testing.T) {
	srv, records := newAPITestServer(t)
	defer srv.Close()
	c := newTestClient(srv.URL, "key")

	input := map[string]interface{}{"name": "Bob", "email": "bob@test.com"}
	_, err := c.CreateContact(input)
	require.NoError(t, err)

	rec := lastRecord(records)
	assert.Equal(t, "POST", rec.Method)
	assert.Equal(t, "/contacts", rec.Path)
	assert.Contains(t, rec.Body, `"name":"Bob"`)
}

func TestUpdateContact(t *testing.T) {
	srv, records := newAPITestServer(t)
	defer srv.Close()
	c := newTestClient(srv.URL, "key")

	input := map[string]interface{}{"name": "Updated"}
	_, err := c.UpdateContact("ct1", input)
	require.NoError(t, err)

	rec := lastRecord(records)
	assert.Equal(t, "PATCH", rec.Method)
	assert.Equal(t, "/contacts/ct1", rec.Path)
}

func TestDeleteContact(t *testing.T) {
	srv, records := newAPITestServer(t)
	defer srv.Close()
	c := newTestClient(srv.URL, "key")

	err := c.DeleteContact("ct1")
	require.NoError(t, err)
	assert.Equal(t, "DELETE", lastRecord(records).Method)
	assert.Equal(t, "/contacts/ct1", lastRecord(records).Path)
}

// ---- Bot methods ----

func TestListBots(t *testing.T) {
	srv, records := newAPITestServer(t)
	defer srv.Close()
	c := newTestClient(srv.URL, "key")

	result, err := c.ListBots(nil)
	require.NoError(t, err)
	assert.Equal(t, 1, result.Total)
	assert.Equal(t, "GET", lastRecord(records).Method)
}

func TestGetBot(t *testing.T) {
	srv, records := newAPITestServer(t)
	defer srv.Close()
	c := newTestClient(srv.URL, "key")

	result, err := c.GetBot("b1")
	require.NoError(t, err)
	assert.Equal(t, "MyBot", result.Name)
	assert.Equal(t, "/bots/b1", lastRecord(records).Path)
}

func TestCreateBot(t *testing.T) {
	srv, records := newAPITestServer(t)
	defer srv.Close()
	c := newTestClient(srv.URL, "key")

	input := map[string]interface{}{"name": "NewBot"}
	_, err := c.CreateBot(input)
	require.NoError(t, err)

	rec := lastRecord(records)
	assert.Equal(t, "POST", rec.Method)
	assert.Equal(t, "/bots", rec.Path)
	assert.Contains(t, rec.Body, `"name":"NewBot"`)
}

func TestUpdateBot(t *testing.T) {
	srv, records := newAPITestServer(t)
	defer srv.Close()
	c := newTestClient(srv.URL, "key")

	input := map[string]interface{}{"name": "UpdatedBot"}
	_, err := c.UpdateBot("b1", input)
	require.NoError(t, err)
	assert.Equal(t, "PATCH", lastRecord(records).Method)
	assert.Equal(t, "/bots/b1", lastRecord(records).Path)
}

func TestDeleteBot(t *testing.T) {
	srv, records := newAPITestServer(t)
	defer srv.Close()
	c := newTestClient(srv.URL, "key")

	err := c.DeleteBot("b1")
	require.NoError(t, err)
	assert.Equal(t, "DELETE", lastRecord(records).Method)
	assert.Equal(t, "/bots/b1", lastRecord(records).Path)
}

func TestStartBot(t *testing.T) {
	srv, records := newAPITestServer(t)
	defer srv.Close()
	c := newTestClient(srv.URL, "key")

	result, err := c.StartBot("b1")
	require.NoError(t, err)
	assert.Equal(t, "running", result.Status)
	assert.Equal(t, "POST", lastRecord(records).Method)
	assert.Equal(t, "/bots/b1/start", lastRecord(records).Path)
}

func TestStopBot(t *testing.T) {
	srv, records := newAPITestServer(t)
	defer srv.Close()
	c := newTestClient(srv.URL, "key")

	result, err := c.StopBot("b1")
	require.NoError(t, err)
	assert.Equal(t, "stopped", result.Status)
	assert.Equal(t, "POST", lastRecord(records).Method)
	assert.Equal(t, "/bots/b1/stop", lastRecord(records).Path)
}

func TestGetBotLogs(t *testing.T) {
	srv, records := newAPITestServer(t)
	defer srv.Close()
	c := newTestClient(srv.URL, "key")

	result, err := c.GetBotLogs("b1", nil)
	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, "started", result[0].Message)
	assert.Equal(t, "/bots/b1/logs", lastRecord(records).Path)
}

// ---- Flow methods ----

func TestListFlows(t *testing.T) {
	srv, records := newAPITestServer(t)
	defer srv.Close()
	c := newTestClient(srv.URL, "key")

	result, err := c.ListFlows(nil)
	require.NoError(t, err)
	assert.Equal(t, 1, result.Total)
	assert.Equal(t, "GET", lastRecord(records).Method)
}

func TestGetFlow(t *testing.T) {
	srv, records := newAPITestServer(t)
	defer srv.Close()
	c := newTestClient(srv.URL, "key")

	result, err := c.GetFlow("f1")
	require.NoError(t, err)
	assert.Equal(t, "Welcome", result.Name)
	assert.Equal(t, "/flows/f1", lastRecord(records).Path)
}

func TestCreateFlow(t *testing.T) {
	srv, records := newAPITestServer(t)
	defer srv.Close()
	c := newTestClient(srv.URL, "key")

	input := map[string]interface{}{"name": "NewFlow"}
	_, err := c.CreateFlow(input)
	require.NoError(t, err)
	assert.Equal(t, "POST", lastRecord(records).Method)
	assert.Equal(t, "/flows", lastRecord(records).Path)
}

func TestDeleteFlow(t *testing.T) {
	srv, records := newAPITestServer(t)
	defer srv.Close()
	c := newTestClient(srv.URL, "key")

	err := c.DeleteFlow("f1")
	require.NoError(t, err)
	assert.Equal(t, "DELETE", lastRecord(records).Method)
	assert.Equal(t, "/flows/f1", lastRecord(records).Path)
}

func TestPublishFlow(t *testing.T) {
	srv, records := newAPITestServer(t)
	defer srv.Close()
	c := newTestClient(srv.URL, "key")

	result, err := c.PublishFlow("f1")
	require.NoError(t, err)
	assert.Equal(t, "published", result.Status)
	assert.Equal(t, "POST", lastRecord(records).Method)
	assert.Equal(t, "/flows/f1/publish", lastRecord(records).Path)
}

func TestUnpublishFlow(t *testing.T) {
	srv, records := newAPITestServer(t)
	defer srv.Close()
	c := newTestClient(srv.URL, "key")

	result, err := c.UnpublishFlow("f1")
	require.NoError(t, err)
	assert.Equal(t, "draft", result.Status)
	assert.Equal(t, "/flows/f1/unpublish", lastRecord(records).Path)
}

func TestExecuteFlow(t *testing.T) {
	srv, records := newAPITestServer(t)
	defer srv.Close()
	c := newTestClient(srv.URL, "key")

	input := map[string]interface{}{"input": "test"}
	result, err := c.ExecuteFlow("f1", input)
	require.NoError(t, err)
	assert.Equal(t, "exec1", result.ExecutionID)
	assert.Equal(t, "completed", result.Status)
	assert.Equal(t, "POST", lastRecord(records).Method)
	assert.Equal(t, "/flows/f1/execute", lastRecord(records).Path)
}

func TestValidateFlow(t *testing.T) {
	srv, records := newAPITestServer(t)
	defer srv.Close()
	c := newTestClient(srv.URL, "key")

	result, err := c.ValidateFlow("f1")
	require.NoError(t, err)
	assert.True(t, result.Valid)
	assert.Equal(t, "POST", lastRecord(records).Method)
	assert.Equal(t, "/flows/f1/validate", lastRecord(records).Path)
}

// ---- Knowledge Base methods ----

func TestListKnowledgeBases(t *testing.T) {
	srv, records := newAPITestServer(t)
	defer srv.Close()
	c := newTestClient(srv.URL, "key")

	result, err := c.ListKnowledgeBases(nil)
	require.NoError(t, err)
	assert.Equal(t, 1, result.Total)
	assert.Equal(t, "GET", lastRecord(records).Method)
}

func TestGetKnowledgeBase(t *testing.T) {
	srv, records := newAPITestServer(t)
	defer srv.Close()
	c := newTestClient(srv.URL, "key")

	result, err := c.GetKnowledgeBase("kb1")
	require.NoError(t, err)
	assert.Equal(t, "FAQ", result.Name)
	assert.Equal(t, "/knowledge-bases/kb1", lastRecord(records).Path)
}

func TestCreateKnowledgeBase(t *testing.T) {
	srv, records := newAPITestServer(t)
	defer srv.Close()
	c := newTestClient(srv.URL, "key")

	input := map[string]interface{}{"name": "NewKB"}
	_, err := c.CreateKnowledgeBase(input)
	require.NoError(t, err)
	assert.Equal(t, "POST", lastRecord(records).Method)
	assert.Equal(t, "/knowledge-bases", lastRecord(records).Path)
}

func TestDeleteKnowledgeBase(t *testing.T) {
	srv, records := newAPITestServer(t)
	defer srv.Close()
	c := newTestClient(srv.URL, "key")

	err := c.DeleteKnowledgeBase("kb1")
	require.NoError(t, err)
	assert.Equal(t, "DELETE", lastRecord(records).Method)
	assert.Equal(t, "/knowledge-bases/kb1", lastRecord(records).Path)
}

func TestQueryKnowledgeBase(t *testing.T) {
	srv, records := newAPITestServer(t)
	defer srv.Close()
	c := newTestClient(srv.URL, "key")

	input := map[string]interface{}{"query": "how to reset password"}
	result, err := c.QueryKnowledgeBase("kb1", input)
	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, 0.95, result[0].Score)
	assert.Equal(t, "POST", lastRecord(records).Method)
	assert.Equal(t, "/knowledge-bases/kb1/query", lastRecord(records).Path)
}

// ---- Document methods ----

func TestListDocuments(t *testing.T) {
	srv, records := newAPITestServer(t)
	defer srv.Close()
	c := newTestClient(srv.URL, "key")

	result, err := c.ListDocuments("kb1", nil)
	require.NoError(t, err)
	assert.Equal(t, 1, result.Total)
	assert.Equal(t, "/knowledge-bases/kb1/documents", lastRecord(records).Path)
}

func TestGetDocument(t *testing.T) {
	srv, records := newAPITestServer(t)
	defer srv.Close()
	c := newTestClient(srv.URL, "key")

	result, err := c.GetDocument("kb1", "d1")
	require.NoError(t, err)
	assert.Equal(t, "Doc1", result.Title)
	assert.Equal(t, "/knowledge-bases/kb1/documents/d1", lastRecord(records).Path)
}

func TestAddDocument(t *testing.T) {
	srv, records := newAPITestServer(t)
	defer srv.Close()
	c := newTestClient(srv.URL, "key")

	input := map[string]interface{}{"title": "New Doc", "content": "Some content"}
	_, err := c.AddDocument("kb1", input)
	require.NoError(t, err)
	assert.Equal(t, "POST", lastRecord(records).Method)
	assert.Equal(t, "/knowledge-bases/kb1/documents", lastRecord(records).Path)
}

func TestDeleteDocument(t *testing.T) {
	srv, records := newAPITestServer(t)
	defer srv.Close()
	c := newTestClient(srv.URL, "key")

	err := c.DeleteDocument("kb1", "d1")
	require.NoError(t, err)
	assert.Equal(t, "DELETE", lastRecord(records).Method)
	assert.Equal(t, "/knowledge-bases/kb1/documents/d1", lastRecord(records).Path)
}

func TestReprocessDocument(t *testing.T) {
	srv, records := newAPITestServer(t)
	defer srv.Close()
	c := newTestClient(srv.URL, "key")

	result, err := c.ReprocessDocument("kb1", "d1")
	require.NoError(t, err)
	assert.Equal(t, "processing", result.Status)
	assert.Equal(t, "POST", lastRecord(records).Method)
	assert.Equal(t, "/knowledge-bases/kb1/documents/d1/reprocess", lastRecord(records).Path)
}

// ---- Webhook methods ----

func TestListWebhooks(t *testing.T) {
	srv, records := newAPITestServer(t)
	defer srv.Close()
	c := newTestClient(srv.URL, "key")

	result, err := c.ListWebhooks(nil)
	require.NoError(t, err)
	assert.Equal(t, 1, result.Total)
	assert.Equal(t, "GET", lastRecord(records).Method)
	assert.Equal(t, "/webhooks", lastRecord(records).Path)
}

func TestListWebhookEvents(t *testing.T) {
	srv, records := newAPITestServer(t)
	defer srv.Close()
	c := newTestClient(srv.URL, "key")

	result, err := c.ListWebhookEvents(nil)
	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, "message.created", result[0].EventType)
	assert.Equal(t, "/webhooks/events", lastRecord(records).Path)
}

// ---- SendDirectMessage ----

func TestSendDirectMessage(t *testing.T) {
	srv, records := newAPITestServer(t)
	defer srv.Close()
	c := newTestClient(srv.URL, "key")

	input := map[string]interface{}{"text": "hi there"}
	result, err := c.SendDirectMessage("ch1", "+5511999", input)
	require.NoError(t, err)
	assert.Equal(t, "dm1", result.ID)

	rec := lastRecord(records)
	assert.Equal(t, "POST", rec.Method)
	assert.Equal(t, "/api/v1/messages/send", rec.Path)
	assert.Contains(t, rec.Body, `"channel_id":"ch1"`)
	assert.Contains(t, rec.Body, `"to":"+5511999"`)
	assert.Contains(t, rec.Body, `"text":"hi there"`)
	assert.Contains(t, rec.Body, `"content_type":"text"`)
}

// ---------------------------------------------------------------------------
// 9. Error handling in API methods
// ---------------------------------------------------------------------------

func TestAPIMethodErrorResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 404, APIError{Message: "channel not found", Code: "NOT_FOUND"})
	}))
	defer srv.Close()
	c := newTestClient(srv.URL, "key")

	t.Run("get returns error", func(t *testing.T) {
		_, err := c.GetChannel("nonexistent")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "channel not found")
	})

	t.Run("delete returns error", func(t *testing.T) {
		err := c.DeleteChannel("nonexistent")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "channel not found")
	})

	t.Run("post returns error", func(t *testing.T) {
		_, err := c.CreateChannel(map[string]interface{}{"name": "bad"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "channel not found")
	})
}

// ---------------------------------------------------------------------------
// 10. Query parameters are appended correctly
// ---------------------------------------------------------------------------

func TestQueryParamsAppended(t *testing.T) {
	var capturedPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.RequestURI()
		writeJSON(w, 200, wrapped(PaginatedResponse[Channel]{Data: []Channel{}, Total: 0}))
	}))
	defer srv.Close()
	c := newTestClient(srv.URL, "key")

	_, err := c.ListChannels(map[string]string{"status": "connected"})
	require.NoError(t, err)
	assert.Contains(t, capturedPath, "status=connected")
}
