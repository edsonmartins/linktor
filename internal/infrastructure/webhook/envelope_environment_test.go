package webhook

import (
	"encoding/json"
	"testing"
	"time"
)

// legacyEnvelope mirrors the linktor-channel-v1 envelope as consumers compiled
// it BEFORE the environment field existed. Round-tripping through it proves the
// new field is additive and backward-compatible.
type legacyEnvelope struct {
	ID        string          `json:"id"`
	Type      string          `json:"type"`
	Timestamp time.Time       `json:"timestamp"`
	TenantID  string          `json:"tenantId"`
	Data      json.RawMessage `json:"data"`
}

func TestEnvelope_EnvironmentFieldIsBackwardCompatible(t *testing.T) {
	env := &Envelope{
		ID: "evt_1", Type: TypeMessageReceived, Timestamp: time.Now().UTC(),
		TenantID: "t1", Environment: "sandbox",
		Data: map[string]string{"k": "v"},
	}
	body, err := json.Marshal(env)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	// A consumer that predates the field keeps parsing everything it knows.
	var legacy legacyEnvelope
	if err := json.Unmarshal(body, &legacy); err != nil {
		t.Fatalf("legacy consumer failed to parse envelope with new field: %v", err)
	}
	if legacy.ID != "evt_1" || legacy.TenantID != "t1" || legacy.Type != TypeMessageReceived {
		t.Fatalf("legacy fields corrupted: %+v", legacy)
	}

	// An OLD envelope (no environment) parses into the new struct with the
	// field empty, which readers treat as production.
	oldBody := []byte(`{"id":"evt_2","type":"message.received","timestamp":"2026-01-01T00:00:00Z","tenantId":"t1","data":{}}`)
	var current Envelope
	if err := json.Unmarshal(oldBody, &current); err != nil {
		t.Fatalf("new struct failed to parse old envelope: %v", err)
	}
	if current.Environment != "" {
		t.Fatalf("environment on legacy envelope = %q, want empty (production)", current.Environment)
	}

	// A production envelope omits the field entirely (omitempty), so the wire
	// format for existing channels is byte-for-byte unchanged.
	prod := &Envelope{ID: "evt_3", Type: TypeMessageReceived, Timestamp: time.Now().UTC(), TenantID: "t1", Data: map[string]string{}}
	prodBody, _ := json.Marshal(prod)
	var asMap map[string]json.RawMessage
	_ = json.Unmarshal(prodBody, &asMap)
	if _, present := asMap["environment"]; present {
		t.Fatal("production envelope must not carry an environment key (omitempty)")
	}
}
