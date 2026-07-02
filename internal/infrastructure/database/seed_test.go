package database

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSeedResetDestructive_DefaultsOff(t *testing.T) {
	t.Setenv("SEED_RESET_DESTRUCTIVE", "")
	assert.False(t, seedResetDestructiveEnabled(), "destructive reset must default to OFF so an empty users table never wipes data")

	t.Setenv("SEED_RESET_DESTRUCTIVE", "false")
	assert.False(t, seedResetDestructiveEnabled())

	t.Setenv("SEED_RESET_DESTRUCTIVE", "1")
	assert.False(t, seedResetDestructiveEnabled(), "only the exact string \"true\" enables destructive reset")

	t.Setenv("SEED_RESET_DESTRUCTIVE", "true")
	assert.True(t, seedResetDestructiveEnabled())
}

func TestGenerateSeedPassword_NotHardcoded(t *testing.T) {
	t.Setenv("SEED_ADMIN_PASSWORD", "")

	p1, err := generateSeedPassword()
	require.NoError(t, err)
	assert.NotEmpty(t, p1)
	assert.NotEqual(t, "admin123", p1, "seed password must not be the old hardcoded constant")

	p2, err := generateSeedPassword()
	require.NoError(t, err)
	assert.NotEqual(t, p1, p2, "generated seed passwords must be random per invocation")
}

func TestGenerateSeedPassword_EnvOverride(t *testing.T) {
	t.Setenv("SEED_ADMIN_PASSWORD", "my-explicit-pass")
	p, err := generateSeedPassword()
	require.NoError(t, err)
	assert.Equal(t, "my-explicit-pass", p)
}
