package email

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

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
