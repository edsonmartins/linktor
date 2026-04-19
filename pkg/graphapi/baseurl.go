// Package graphapi centralises the Meta Graph API base URL, with an
// environment-variable override used by tests and local development
// to redirect traffic to a Prism mock server.
package graphapi

import "os"

// Default is the production Meta Graph API endpoint.
const Default = "https://graph.facebook.com"

// EnvVar is the name of the environment variable that, when set to a
// non-empty string, overrides BaseURL for every caller in the codebase.
// This is the hook used by the deploy/mocks/ Prism setup (see
// deploy/mocks/README.md) and by integration tests that want to avoid
// hitting graph.facebook.com.
const EnvVar = "LINKTOR_GRAPH_API_URL"

// BaseURL returns the Graph API base URL, honoring LINKTOR_GRAPH_API_URL
// when set. It is safe to call on every request — the env lookup is
// cheap and dynamic lookup lets t.Setenv in tests take effect.
func BaseURL() string {
	if v := os.Getenv(EnvVar); v != "" {
		return v
	}
	return Default
}
