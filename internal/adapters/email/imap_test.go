package email

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewIMAPClient(t *testing.T) {
	t.Run("valid config", func(t *testing.T) {
		cfg := &Config{
			IMAPHost:         "imap.example.com",
			IMAPPort:         993,
			IMAPUsername:     "user",
			IMAPPassword:     "pass",
			IMAPFolder:       "INBOX",
			IMAPPollInterval: 30,
		}
		client, err := NewIMAPClient(cfg)
		require.NoError(t, err)
		require.NotNil(t, client)
		assert.Equal(t, 993, client.config.IMAPPort)
		assert.Equal(t, "INBOX", client.config.IMAPFolder)
		assert.Equal(t, 30, client.config.IMAPPollInterval)
	})

	t.Run("missing host", func(t *testing.T) {
		cfg := &Config{
			IMAPPort: 993,
		}
		client, err := NewIMAPClient(cfg)
		assert.Error(t, err)
		assert.Nil(t, client)
		assert.Contains(t, err.Error(), "imap_host is required")
	})

	t.Run("default values", func(t *testing.T) {
		cfg := &Config{
			IMAPHost: "imap.example.com",
			// Port, folder, and interval are all zero/empty
		}
		client, err := NewIMAPClient(cfg)
		require.NoError(t, err)
		require.NotNil(t, client)
		assert.Equal(t, 993, client.config.IMAPPort)
		assert.Equal(t, "INBOX", client.config.IMAPFolder)
		assert.Equal(t, 30, client.config.IMAPPollInterval)
	})
}

func TestParseFromName(t *testing.T) {
	t.Run("with name", func(t *testing.T) {
		assert.Equal(t, "Alice", parseFromName("Alice <alice@ex.com>"))
	})

	t.Run("email only", func(t *testing.T) {
		assert.Equal(t, "", parseFromName("alice@ex.com"))
	})

	t.Run("name with spaces", func(t *testing.T) {
		assert.Equal(t, "Alice Bob", parseFromName("Alice Bob <alice@ex.com>"))
	})

	t.Run("empty string", func(t *testing.T) {
		assert.Equal(t, "", parseFromName(""))
	})
}

func TestParseAddressList(t *testing.T) {
	t.Run("single address", func(t *testing.T) {
		result := parseAddressList("user@example.com")
		assert.Equal(t, []string{"user@example.com"}, result)
	})

	t.Run("multiple addresses", func(t *testing.T) {
		result := parseAddressList("user1@example.com, user2@example.com")
		assert.Equal(t, []string{"user1@example.com", "user2@example.com"}, result)
	})

	t.Run("with names", func(t *testing.T) {
		result := parseAddressList("Alice <alice@example.com>, Bob <bob@example.com>")
		assert.Equal(t, []string{"alice@example.com", "bob@example.com"}, result)
	})

	t.Run("mixed format", func(t *testing.T) {
		result := parseAddressList("alice@example.com, Bob <bob@example.com>")
		assert.Equal(t, []string{"alice@example.com", "bob@example.com"}, result)
	})

	t.Run("empty string produces empty result", func(t *testing.T) {
		result := parseAddressList("")
		assert.Empty(t, result)
	})
}
