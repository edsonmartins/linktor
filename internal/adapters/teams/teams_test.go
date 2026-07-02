package teams

import (
	"testing"

	"github.com/msgfy/linktor/internal/outbound"
)

func TestBuildOutboundActivityText(t *testing.T) {
	a, err := buildOutboundActivity(outbound.Text{Body: "hello"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if a.Type != "message" || a.Text != "hello" {
		t.Errorf("unexpected activity: %+v", a)
	}
}

func TestBuildOutboundActivityMedia(t *testing.T) {
	a, err := buildOutboundActivity(outbound.Media{
		Type:     outbound.MediaImage,
		URL:      "https://cdn/x.jpg",
		Caption:  "cap",
		Filename: "x.jpg",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(a.Attachments) != 1 || a.Attachments[0].ContentURL != "https://cdn/x.jpg" {
		t.Errorf("unexpected attachments: %+v", a.Attachments)
	}
	if a.Text != "cap" {
		t.Errorf("expected caption as text, got %q", a.Text)
	}
}

func TestBuildOutboundActivityEmptyTextPermanent(t *testing.T) {
	_, err := buildOutboundActivity(outbound.Text{Body: ""})
	if !outbound.IsPermanent(err) {
		t.Errorf("expected permanent error, got %v", err)
	}
}

func TestClassifyError(t *testing.T) {
	cases := []struct {
		code      int
		permanent bool
	}{
		{400, true},
		{401, true},
		{404, true},
		{429, false},
		{500, false},
		{503, false},
	}
	for _, tc := range cases {
		err := classifyError(&httpError{StatusCode: tc.code})
		if got := outbound.IsPermanent(err); got != tc.permanent {
			t.Errorf("status %d: permanent=%v, want %v", tc.code, got, tc.permanent)
		}
	}
}

func TestReferenceFromActivityPrefersChannelDataTenant(t *testing.T) {
	a := &Activity{
		ServiceURL:   "https://smba.example/teams",
		Conversation: ConversationAccount{ID: "conv1", TenantID: "conv-tenant"},
		Recipient:    ChannelAccount{ID: "bot1"},
	}
	a.ChannelData.Tenant.ID = "cd-tenant"

	ref := ReferenceFromActivity(a)
	if ref.ConversationID != "conv1" || ref.ServiceURL != "https://smba.example/teams" {
		t.Errorf("unexpected reference: %+v", ref)
	}
	if ref.TenantID != "cd-tenant" {
		t.Errorf("expected channelData tenant to win, got %q", ref.TenantID)
	}
}

func TestConfigFromCreds(t *testing.T) {
	cfg := configFromCreds(map[string]string{
		CredAppID:       "app",
		CredAppPassword: "pw",
		CredTenantID:    "tenant",
		CredServiceURL:  "https://svc",
	})
	if cfg.AppID != "app" || cfg.AppPassword != "pw" || cfg.TenantID != "tenant" || cfg.ServiceURL != "https://svc" {
		t.Errorf("unexpected config: %+v", cfg)
	}
}
