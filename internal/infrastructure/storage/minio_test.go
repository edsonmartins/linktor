package storage

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMinIOClient_ImplementsInterface(t *testing.T) {
	var _ Client = (*MinIOClient)(nil)
}

func TestNewMinIOClient_InvalidEndpoint(t *testing.T) {
	// Using an invalid endpoint should fail during bucket check
	_, err := NewMinIOClient(
		"invalid-host-that-does-not-exist:99999",
		"minioadmin",
		"minioadmin",
		"test-bucket",
		"us-east-1",
		false,
	)
	assert.Error(t, err)
}
