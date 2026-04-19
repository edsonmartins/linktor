//go:build mock

// Integration tests that target the Prism mock server at
// LINKTOR_GRAPH_API_URL (default http://localhost:4010). Start the mock
// first: `make mock-whatsapp-up`. Run with `go test -tags=mock ./...`.
package whatsapp_official

import (
	"context"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/msgfy/linktor/pkg/graphapi"
)

func requireMock(t *testing.T) string {
	t.Helper()
	base := graphapi.BaseURL()
	if strings.HasPrefix(base, "https://graph.facebook.com") {
		t.Skip("LINKTOR_GRAPH_API_URL not pointed at mock; skipping. Run `make mock-whatsapp-up` and re-run with the env set.")
	}
	req, _ := http.NewRequest(http.MethodGet, base+"/v23.0/me", nil)
	req.Header.Set("Authorization", "Bearer test")
	resp, err := (&http.Client{Timeout: 3 * time.Second}).Do(req)
	if err != nil {
		t.Skipf("mock unreachable at %s: %v", base, err)
	}
	_ = resp.Body.Close()
	return base
}

func TestMockSendMessage_RejectsInvalidPayload(t *testing.T) {
	base := requireMock(t)
	_ = os.Setenv(graphapi.EnvVar, base) // make sure the client resolves to the mock

	client := NewClient(&Config{
		AccessToken:   "test-token",
		PhoneNumberID: "1234567890",
		BusinessID:    "9876543210",
	})

	// Missing required fields on every branch of the polymorphic schema —
	// Prism should reject (or return a validation note). Either way the
	// contract here is: the client surfaces the error rather than silently
	// succeeding.
	_, err := client.SendMessage(context.Background(), &SendMessageRequest{
		Type: MessageTypeText,
	})
	if err == nil {
		t.Fatal("expected error from mock for missing required fields, got nil")
	}
}

// Note: a happy-path test isn't included because Prism's response body for
// POST /messages with --errors=false and --dynamic is lorem-ipsum strings
// that sometimes trip our response parser. If you need round-tripping,
// pair this with Prism's proxy mode against a real staging WABA.
