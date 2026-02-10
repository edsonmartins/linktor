package linktor

import (
	"testing"
	"time"
)

// TestNewClient tests creating a new client
func TestNewClient(t *testing.T) {
	client := NewClient()

	if client == nil {
		t.Fatal("Expected client to be created")
	}

	// Check resources are initialized
	if client.Auth == nil {
		t.Error("Expected Auth resource to be initialized")
	}
	if client.Conversations == nil {
		t.Error("Expected Conversations resource to be initialized")
	}
	if client.Contacts == nil {
		t.Error("Expected Contacts resource to be initialized")
	}
	if client.Channels == nil {
		t.Error("Expected Channels resource to be initialized")
	}
	if client.Bots == nil {
		t.Error("Expected Bots resource to be initialized")
	}
	if client.AI == nil {
		t.Error("Expected AI resource to be initialized")
	}
	if client.KnowledgeBases == nil {
		t.Error("Expected KnowledgeBases resource to be initialized")
	}
	if client.Flows == nil {
		t.Error("Expected Flows resource to be initialized")
	}
	if client.Analytics == nil {
		t.Error("Expected Analytics resource to be initialized")
	}
}

// TestNewClientWithOptions tests creating client with options
func TestNewClientWithOptions(t *testing.T) {
	client := NewClient(
		WithBaseURL("https://custom.api.com"),
		WithAPIKey("test-api-key"),
		WithTimeout(60*time.Second),
		WithMaxRetries(5),
	)

	if client == nil {
		t.Fatal("Expected client to be created")
	}

	if client.config.BaseURL != "https://custom.api.com" {
		t.Errorf("Expected BaseURL to be https://custom.api.com, got %s", client.config.BaseURL)
	}

	if client.config.APIKey != "test-api-key" {
		t.Errorf("Expected APIKey to be test-api-key, got %s", client.config.APIKey)
	}

	if client.config.Timeout != 60*time.Second {
		t.Errorf("Expected Timeout to be 60s, got %v", client.config.Timeout)
	}

	if client.config.MaxRetries != 5 {
		t.Errorf("Expected MaxRetries to be 5, got %d", client.config.MaxRetries)
	}
}

// TestNewClientWithAccessToken tests creating client with access token
func TestNewClientWithAccessToken(t *testing.T) {
	client := NewClient(
		WithAccessToken("test-access-token"),
	)

	if client == nil {
		t.Fatal("Expected client to be created")
	}

	if client.config.AccessToken != "test-access-token" {
		t.Errorf("Expected AccessToken to be test-access-token, got %s", client.config.AccessToken)
	}
}

// TestSetAPIKey tests updating API key
func TestSetAPIKey(t *testing.T) {
	client := NewClient()

	client.SetAPIKey("new-api-key")

	if client.config.APIKey != "new-api-key" {
		t.Errorf("Expected APIKey to be new-api-key, got %s", client.config.APIKey)
	}
}

// TestSetAccessToken tests updating access token
func TestSetAccessToken(t *testing.T) {
	client := NewClient()

	client.SetAccessToken("new-access-token")

	if client.config.AccessToken != "new-access-token" {
		t.Errorf("Expected AccessToken to be new-access-token, got %s", client.config.AccessToken)
	}
}

// TestDefaultConfig tests default configuration
func TestDefaultConfig(t *testing.T) {
	client := NewClient()

	if client.config.BaseURL != "https://api.linktor.io" {
		t.Errorf("Expected default BaseURL to be https://api.linktor.io, got %s", client.config.BaseURL)
	}

	if client.config.Timeout != 30*time.Second {
		t.Errorf("Expected default Timeout to be 30s, got %v", client.config.Timeout)
	}

	if client.config.MaxRetries != 3 {
		t.Errorf("Expected default MaxRetries to be 3, got %d", client.config.MaxRetries)
	}

	if client.config.RetryDelay != time.Second {
		t.Errorf("Expected default RetryDelay to be 1s, got %v", client.config.RetryDelay)
	}
}

// TestErrorString tests Error.Error() method
func TestErrorString(t *testing.T) {
	err := &Error{
		Code:    "VALIDATION_ERROR",
		Message: "Invalid input",
		Status:  400,
	}

	expected := "[VALIDATION_ERROR] Invalid input"
	if err.Error() != expected {
		t.Errorf("Expected error string to be %s, got %s", expected, err.Error())
	}
}

// TestWithBaseURL tests WithBaseURL option
func TestWithBaseURL(t *testing.T) {
	config := &ClientConfig{}
	opt := WithBaseURL("https://test.com")
	opt(config)

	if config.BaseURL != "https://test.com" {
		t.Errorf("Expected BaseURL to be https://test.com, got %s", config.BaseURL)
	}
}

// TestWithAPIKey tests WithAPIKey option
func TestWithAPIKey(t *testing.T) {
	config := &ClientConfig{}
	opt := WithAPIKey("test-key")
	opt(config)

	if config.APIKey != "test-key" {
		t.Errorf("Expected APIKey to be test-key, got %s", config.APIKey)
	}
}

// TestWithAccessToken tests WithAccessToken option
func TestWithAccessToken(t *testing.T) {
	config := &ClientConfig{}
	opt := WithAccessToken("test-token")
	opt(config)

	if config.AccessToken != "test-token" {
		t.Errorf("Expected AccessToken to be test-token, got %s", config.AccessToken)
	}
}

// TestWithTimeout tests WithTimeout option
func TestWithTimeout(t *testing.T) {
	config := &ClientConfig{}
	opt := WithTimeout(120 * time.Second)
	opt(config)

	if config.Timeout != 120*time.Second {
		t.Errorf("Expected Timeout to be 120s, got %v", config.Timeout)
	}
}

// TestWithMaxRetries tests WithMaxRetries option
func TestWithMaxRetries(t *testing.T) {
	config := &ClientConfig{}
	opt := WithMaxRetries(10)
	opt(config)

	if config.MaxRetries != 10 {
		t.Errorf("Expected MaxRetries to be 10, got %d", config.MaxRetries)
	}
}
