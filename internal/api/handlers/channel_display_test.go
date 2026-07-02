package handlers

import (
	"testing"

	"github.com/msgfy/linktor/internal/adapters/teams"
	"github.com/msgfy/linktor/internal/domain/entity"
)

func TestWithDisplayConfigExposesOnlyNonSecrets(t *testing.T) {
	ch := &entity.Channel{
		Type: entity.ChannelType(teams.ChannelType),
		Config: map[string]string{
			"existing": "keep",
		},
		Credentials: map[string]string{
			teams.CredAppID:       "app-123",
			teams.CredTenantID:    "org-1",
			teams.CredServiceURL:  "https://smba.example",
			teams.CredAppPassword: "TOP-SECRET",
		},
	}

	out := withDisplayConfig(ch)

	// Non-secret fields surfaced for prefill.
	if out.Config[teams.CredAppID] != "app-123" || out.Config[teams.CredTenantID] != "org-1" {
		t.Errorf("expected non-secret creds in display config, got %+v", out.Config)
	}
	if out.Config["existing"] != "keep" {
		t.Error("existing config keys must be preserved")
	}
	// Secret MUST NOT be exposed.
	if _, leaked := out.Config[teams.CredAppPassword]; leaked {
		t.Error("app_password (secret) leaked into display config")
	}

	// Original entity must not be mutated.
	if _, mutated := ch.Config[teams.CredAppID]; mutated {
		t.Error("withDisplayConfig must not mutate the original channel's Config")
	}
}

func TestWithDisplayConfigIgnoresUnmappedTypes(t *testing.T) {
	ch := &entity.Channel{
		Type:        entity.ChannelType("telegram"),
		Credentials: map[string]string{"bot_token": "secret"},
	}
	out := withDisplayConfig(ch)
	if _, leaked := out.Config["bot_token"]; leaked {
		t.Error("unmapped type must not surface any credentials")
	}
}
