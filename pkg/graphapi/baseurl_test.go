package graphapi

import "testing"

func TestBaseURL_DefaultWhenUnset(t *testing.T) {
	t.Setenv(EnvVar, "")
	if got := BaseURL(); got != Default {
		t.Errorf("BaseURL() = %q, want %q", got, Default)
	}
}

func TestBaseURL_OverrideFromEnv(t *testing.T) {
	t.Setenv(EnvVar, "http://localhost:4010")
	if got := BaseURL(); got != "http://localhost:4010" {
		t.Errorf("BaseURL() = %q, want override", got)
	}
}
