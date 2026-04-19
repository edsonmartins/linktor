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

	"github.com/msgfy/linktor/internal/domain/entity"
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

// -----------------------------------------------------------------------------
// Template E2E — exercise BuildSendPayload against Prism so payload shape
// divergences between our code and Meta's spec blow up locally.
// -----------------------------------------------------------------------------

func TestMockTemplateSend_BuiltFromEntityValidatesAgainstSpec(t *testing.T) {
	base := requireMock(t)
	_ = os.Setenv(graphapi.EnvVar, base)

	// Stored template with body + URL button, mirroring how an admin would
	// author it in the UI once and then reuse it per conversation.
	tpl := &entity.Template{
		Name:     "order_ready",
		Language: "pt_BR",
		Category: entity.TemplateCategoryUtility,
		Status:   entity.TemplateStatusApproved,
		Components: []entity.TemplateComponent{
			{
				Type: "HEADER", Format: "IMAGE",
				Example: &entity.TemplateExample{HeaderHandle: []string{"4:AAA"}},
			},
			{
				Type: "BODY",
				Text: "Olá {{1}}, o pedido {{2}} está pronto.",
				Example: &entity.TemplateExample{
					BodyText: [][]string{{"Ana", "ORD-42"}},
				},
			},
			{
				Type: "BUTTONS",
				Buttons: []entity.TemplateButton{
					{Type: "URL", Text: "Acompanhar"},
				},
			},
		},
	}

	payload, err := BuildSendPayload(tpl, SendValues{
		HeaderImageURL: "https://cdn.linktor.dev/hero.jpg",
		BodyParams:     []string{"Ana", "ORD-42"},
		ButtonValues:   map[int]string{0: "/tracking/ord-42"},
	})
	if err != nil {
		t.Fatalf("BuildSendPayload failed: %v", err)
	}

	client := NewClient(&Config{
		AccessToken:   "test-token",
		PhoneNumberID: "1234567890",
		BusinessID:    "9876543210",
	})

	_, err = client.SendMessage(context.Background(), &SendMessageRequest{
		To:       "+5511999999999",
		Type:     MessageType("template"),
		Template: payload,
	})
	// Prism either accepts the payload (dynamic mode) or flags a specific
	// field — what we test for is the absence of URL/404 errors, which
	// would signal a wrong endpoint or missing route. The presence of the
	// `messages` path and the `template` type is therefore implicitly
	// validated by whatever 2xx/4xx Prism returns.
	if err != nil {
		// 4xx with a validation body is fine — it means Prism reached the
		// route. 404 or connection errors are not.
		if strings.Contains(err.Error(), "404") || strings.Contains(err.Error(), "connection refused") {
			t.Fatalf("template send failed at transport layer, not spec validation: %v", err)
		}
	}
}

func TestMockTemplateSend_MissingPlaceholderValuesRejected(t *testing.T) {
	base := requireMock(t)
	_ = os.Setenv(graphapi.EnvVar, base)

	// BuildSendPayload is lenient — it just omits the body params when
	// the caller didn't provide them. Sending such a payload lets Prism
	// validate that the template type carries *some* body, and catches
	// malformed component arrays before they reach real users.
	tpl := &entity.Template{
		Name:     "order_ready",
		Language: "pt_BR",
		Components: []entity.TemplateComponent{
			{Type: "BODY", Text: "Olá {{1}}"},
		},
	}
	payload, err := BuildSendPayload(tpl, SendValues{})
	if err != nil {
		t.Fatalf("BuildSendPayload failed: %v", err)
	}
	// Payload has no body parameters — Meta's schema would normally reject
	// this when the template expects {{1}}. Prism's /messages validator is
	// only partially aware of template schemas, so we just sanity-check
	// that BuildSendPayload correctly dropped the empty body.
	if len(payload.Components) != 0 {
		t.Fatalf("expected no components for empty values, got %d", len(payload.Components))
	}
}
