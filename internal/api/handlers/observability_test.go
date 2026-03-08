package handlers

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewObservabilityHandler(t *testing.T) {
	// Test that constructor works with nil service without panicking
	h := NewObservabilityHandler(nil)
	require.NotNil(t, h)
	assert.Nil(t, h.observabilityService)
}
