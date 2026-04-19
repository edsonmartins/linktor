package flows

import (
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// rewriteTransport redirects outbound requests at the Graph API to the
// in-process test server while keeping the original path + query intact.
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

func newTestFlowClient(handler http.HandlerFunc) (*FlowClient, *httptest.Server) {
	server := httptest.NewServer(handler)
	c := NewFlowClient(&FlowClientConfig{
		AccessToken: "test-token",
		WABAID:      "waba-42",
		APIVersion:  "v23.0",
	})
	c.httpClient = &http.Client{Transport: &rewriteTransport{baseURL: server.URL, rt: http.DefaultTransport}}
	return c, server
}

func newTestFlowSender(handler http.HandlerFunc) (*FlowSender, *httptest.Server) {
	server := httptest.NewServer(handler)
	s := NewFlowSender("phone-99", "test-token", "v23.0")
	s.httpClient = &http.Client{Transport: &rewriteTransport{baseURL: server.URL, rt: http.DefaultTransport}}
	return s, server
}

// -----------------------------------------------------------------------------
// CreateFlow
// -----------------------------------------------------------------------------

func TestFlowClient_CreateFlow_Success(t *testing.T) {
	var capturedPath, capturedMethod, capturedAuth string
	var capturedBody CreateFlowRequest

	client, server := newTestFlowClient(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		capturedMethod = r.Method
		capturedAuth = r.Header.Get("Authorization")
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &capturedBody)

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(FlowInfo{ID: "flow-123", Name: capturedBody.Name, Status: "DRAFT"})
	})
	defer server.Close()

	info, err := client.CreateFlow(context.Background(), &CreateFlowRequest{
		Name:       "Lead Form",
		Categories: []string{"SIGN_UP"},
	})
	require.NoError(t, err)
	require.NotNil(t, info)

	assert.Equal(t, http.MethodPost, capturedMethod)
	assert.Equal(t, "/v23.0/waba-42/flows", capturedPath)
	assert.Equal(t, "Bearer test-token", capturedAuth)
	assert.Equal(t, "Lead Form", capturedBody.Name)
	assert.Equal(t, []string{"SIGN_UP"}, capturedBody.Categories)
	assert.Equal(t, "flow-123", info.ID)
	assert.Equal(t, FlowStatusDraft, info.Status)
}

func TestFlowClient_CreateFlow_APIError(t *testing.T) {
	client, server := newTestFlowClient(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"message":"invalid category","code":100}}`))
	})
	defer server.Close()

	_, err := client.CreateFlow(context.Background(), &CreateFlowRequest{Name: "bad"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid category")
	assert.Contains(t, err.Error(), "400")
}

// -----------------------------------------------------------------------------
// GetFlow + GetFlowPreviewURL
// -----------------------------------------------------------------------------

func TestFlowClient_GetFlow_Success(t *testing.T) {
	var capturedPath, capturedQuery string

	client, server := newTestFlowClient(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		capturedQuery = r.URL.RawQuery
		_ = json.NewEncoder(w).Encode(FlowInfo{
			ID: "flow-123", Name: "Lead Form", Status: "PUBLISHED", PreviewURL: "https://preview/abc",
		})
	})
	defer server.Close()

	info, err := client.GetFlow(context.Background(), "flow-123")
	require.NoError(t, err)
	assert.Equal(t, "/v23.0/flow-123", capturedPath)
	assert.Contains(t, capturedQuery, "fields=")
	assert.Equal(t, FlowStatusPublished, info.Status)
	assert.Equal(t, "https://preview/abc", info.PreviewURL)
}

func TestFlowClient_GetFlowPreviewURL(t *testing.T) {
	client, server := newTestFlowClient(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(FlowInfo{ID: "f", PreviewURL: "https://preview/xyz"})
	})
	defer server.Close()

	url, err := client.GetFlowPreviewURL(context.Background(), "f")
	require.NoError(t, err)
	assert.Equal(t, "https://preview/xyz", url)
}

// -----------------------------------------------------------------------------
// ListFlows
// -----------------------------------------------------------------------------

func TestFlowClient_ListFlows_Success(t *testing.T) {
	var capturedPath string
	client, server := newTestFlowClient(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		_, _ = w.Write([]byte(`{"data":[{"id":"f1","name":"A"},{"id":"f2","name":"B"}]}`))
	})
	defer server.Close()

	flows, err := client.ListFlows(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "/v23.0/waba-42/flows", capturedPath)
	assert.Len(t, flows, 2)
	assert.Equal(t, "f1", flows[0].ID)
}

// -----------------------------------------------------------------------------
// UpdateFlow
// -----------------------------------------------------------------------------

func TestFlowClient_UpdateFlow_Success(t *testing.T) {
	var capturedPath, capturedMethod string
	var captured UpdateFlowRequest

	client, server := newTestFlowClient(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		capturedMethod = r.Method
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &captured)
		_ = json.NewEncoder(w).Encode(FlowInfo{ID: "f1", Name: captured.Name})
	})
	defer server.Close()

	info, err := client.UpdateFlow(context.Background(), "f1", &UpdateFlowRequest{Name: "Renamed"})
	require.NoError(t, err)
	assert.Equal(t, http.MethodPost, capturedMethod)
	assert.Equal(t, "/v23.0/f1", capturedPath)
	assert.Equal(t, "Renamed", info.Name)
}

// -----------------------------------------------------------------------------
// UpdateFlowJSON (multipart upload)
// -----------------------------------------------------------------------------

func TestFlowClient_UpdateFlowJSON_Success(t *testing.T) {
	var capturedPath, capturedContentType, capturedName, capturedAssetType, capturedFile string

	client, server := newTestFlowClient(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		capturedContentType = r.Header.Get("Content-Type")

		_, params, err := mimeParse(capturedContentType)
		require.NoError(t, err)
		reader := multipart.NewReader(r.Body, params["boundary"])
		for {
			part, err := reader.NextPart()
			if err != nil {
				break
			}
			b, _ := io.ReadAll(part)
			switch part.FormName() {
			case "name":
				capturedName = string(b)
			case "asset_type":
				capturedAssetType = string(b)
			case "file":
				capturedFile = string(b)
			}
			_ = part.Close()
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"success":true}`))
	})
	defer server.Close()

	err := client.UpdateFlowJSON(context.Background(), "flow-1", `{"version":"3.0","screens":[]}`)
	require.NoError(t, err)
	assert.Equal(t, "/v23.0/flow-1/assets", capturedPath)
	assert.True(t, strings.HasPrefix(capturedContentType, "multipart/form-data"))
	assert.Equal(t, "flow.json", capturedName)
	assert.Equal(t, "FLOW_JSON", capturedAssetType)
	assert.Contains(t, capturedFile, `"version":"3.0"`)
}

func TestFlowClient_UpdateFlowJSON_APIError(t *testing.T) {
	client, server := newTestFlowClient(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = w.Write([]byte(`{"error":{"message":"invalid flow json"}}`))
	})
	defer server.Close()

	err := client.UpdateFlowJSON(context.Background(), "flow-1", `{}`)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "422")
}

// -----------------------------------------------------------------------------
// PublishFlow / DeprecateFlow / DeleteFlow
// -----------------------------------------------------------------------------

func TestFlowClient_PublishFlow(t *testing.T) {
	var capturedPath, capturedMethod string
	client, server := newTestFlowClient(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		capturedMethod = r.Method
		_, _ = w.Write([]byte(`{"success":true}`))
	})
	defer server.Close()

	require.NoError(t, client.PublishFlow(context.Background(), "flow-1"))
	assert.Equal(t, http.MethodPost, capturedMethod)
	assert.Equal(t, "/v23.0/flow-1/publish", capturedPath)
}

func TestFlowClient_DeprecateFlow(t *testing.T) {
	var capturedPath string
	client, server := newTestFlowClient(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		_, _ = w.Write([]byte(`{"success":true}`))
	})
	defer server.Close()

	require.NoError(t, client.DeprecateFlow(context.Background(), "flow-1"))
	assert.Equal(t, "/v23.0/flow-1/deprecate", capturedPath)
}

func TestFlowClient_DeleteFlow(t *testing.T) {
	var capturedPath, capturedMethod string
	client, server := newTestFlowClient(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		capturedMethod = r.Method
		_, _ = w.Write([]byte(`{"success":true}`))
	})
	defer server.Close()

	require.NoError(t, client.DeleteFlow(context.Background(), "flow-1"))
	assert.Equal(t, http.MethodDelete, capturedMethod)
	assert.Equal(t, "/v23.0/flow-1", capturedPath)
}

// -----------------------------------------------------------------------------
// GetFlowAssets (two-step: assets list + download URL)
// -----------------------------------------------------------------------------

func TestFlowClient_GetFlowAssets_Success(t *testing.T) {
	var step int
	var assetsPath, downloadPath string

	client, server := newTestFlowClient(func(w http.ResponseWriter, r *http.Request) {
		step++
		switch step {
		case 1:
			assetsPath = r.URL.Path
			// the asset index — download_url points back at us
			dlURL := "http://example.invalid/download/flow-1.json"
			_, _ = w.Write([]byte(`{"data":[{"name":"flow.json","asset_type":"FLOW_JSON","download_url":"` + dlURL + `"}]}`))
		default:
			downloadPath = r.URL.Path
			_, _ = w.Write([]byte(`{"version":"3.0"}`))
		}
	})
	defer server.Close()

	assets, err := client.GetFlowAssets(context.Background(), "flow-1")
	require.NoError(t, err)
	assert.Equal(t, "/v23.0/flow-1/assets", assetsPath)
	assert.Equal(t, "/download/flow-1.json", downloadPath)
	assert.Contains(t, assets.FlowJSON, `"version":"3.0"`)
}

func TestFlowClient_GetFlowAssets_MissingFlowJSON(t *testing.T) {
	client, server := newTestFlowClient(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"data":[{"name":"other","asset_type":"OTHER"}]}`))
	})
	defer server.Close()

	_, err := client.GetFlowAssets(context.Background(), "flow-1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "flow.json not found")
}

// -----------------------------------------------------------------------------
// FlowSender.SendFlow
// -----------------------------------------------------------------------------

func TestFlowSender_SendFlow_Success(t *testing.T) {
	var capturedPath, capturedMethod, capturedAuth string
	var capturedMsg map[string]interface{}

	sender, server := newTestFlowSender(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		capturedMethod = r.Method
		capturedAuth = r.Header.Get("Authorization")
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &capturedMsg)
		_, _ = w.Write([]byte(`{"messages":[{"id":"wamid.flow-999"}]}`))
	})
	defer server.Close()

	id, err := sender.SendFlow(context.Background(), &SendFlowInput{
		To:            "+15551234567",
		FlowID:        "flow-1",
		FlowCTA:       "Abrir",
		BodyText:      "Preencha o formulário",
		HeaderText:    "Cadastro",
		FooterText:    "Obrigado",
		FlowToken:     "tok-abc",
		InitialScreen: "WELCOME",
		InitialData:   map[string]interface{}{"customer_id": "c1"},
	})
	require.NoError(t, err)
	assert.Equal(t, "wamid.flow-999", id)
	assert.Equal(t, http.MethodPost, capturedMethod)
	assert.Equal(t, "/v23.0/phone-99/messages", capturedPath)
	assert.Equal(t, "Bearer test-token", capturedAuth)

	// payload shape — key fields that Meta's spec requires
	assert.Equal(t, "whatsapp", capturedMsg["messaging_product"])
	assert.Equal(t, "individual", capturedMsg["recipient_type"])
	assert.Equal(t, "interactive", capturedMsg["type"])
	interactive := capturedMsg["interactive"].(map[string]interface{})
	assert.Equal(t, "flow", interactive["type"])
	action := interactive["action"].(map[string]interface{})
	params := action["parameters"].(map[string]interface{})
	assert.Equal(t, "flow-1", params["flow_id"])
	assert.Equal(t, "tok-abc", params["flow_token"])
	assert.Equal(t, "Abrir", params["flow_cta"])
	payload := params["flow_action_payload"].(map[string]interface{})
	assert.Equal(t, "WELCOME", payload["screen"])
}

func TestFlowSender_SendFlow_APIError(t *testing.T) {
	sender, server := newTestFlowSender(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"message":"invalid flow_token","code":131051}}`))
	})
	defer server.Close()

	_, err := sender.SendFlow(context.Background(), &SendFlowInput{
		To:       "+15551234567",
		FlowID:   "f",
		BodyText: "x",
	})
	require.Error(t, err)
}

// helper: small wrapper to keep the import list local to the Content-Type parse
func mimeParse(v string) (string, map[string]string, error) {
	parts := strings.SplitN(v, ";", 2)
	mediaType := strings.TrimSpace(parts[0])
	params := map[string]string{}
	if len(parts) == 2 {
		for _, p := range strings.Split(parts[1], ";") {
			kv := strings.SplitN(strings.TrimSpace(p), "=", 2)
			if len(kv) == 2 {
				params[kv[0]] = strings.Trim(kv[1], `"`)
			}
		}
	}
	return mediaType, params, nil
}
