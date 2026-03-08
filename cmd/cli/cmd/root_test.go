package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRootCmd_HasExpectedSubcommands(t *testing.T) {
	expectedNames := []string{
		"version", "auth", "channel", "send", "conv",
		"contact", "bot", "flow", "kb", "webhook",
		"config", "server",
	}

	subcommandNames := make(map[string]bool)
	for _, cmd := range rootCmd.Commands() {
		subcommandNames[cmd.Name()] = true
	}

	for _, name := range expectedNames {
		assert.True(t, subcommandNames[name], "rootCmd should have subcommand %q", name)
	}
}

func TestRootCmd_Use(t *testing.T) {
	assert.Equal(t, "msgfy", rootCmd.Use)
}

func TestRootCmd_SilenceUsage(t *testing.T) {
	assert.True(t, rootCmd.SilenceUsage)
}

func TestRootCmd_SilenceErrors(t *testing.T) {
	assert.True(t, rootCmd.SilenceErrors)
}

func TestRootCmd_HasPersistentFlags(t *testing.T) {
	assert.NotNil(t, rootCmd.PersistentFlags().Lookup("config"))
	assert.NotNil(t, rootCmd.PersistentFlags().Lookup("profile"))
	assert.NotNil(t, rootCmd.PersistentFlags().Lookup("output"))
	assert.NotNil(t, rootCmd.PersistentFlags().Lookup("no-color"))
}

func TestLoadProfile_WithData(t *testing.T) {
	// loadProfile should not panic even when no profiles are configured.
	// Since viper may not have "profiles" key, it just returns silently.
	assert.NotPanics(t, func() {
		loadProfile("nonexistent")
	})
}

func TestSuccess_DoesNotPanic(t *testing.T) {
	assert.NotPanics(t, func() {
		success("test %s", "message")
	})
}

func TestErrorMsg_DoesNotPanic(t *testing.T) {
	assert.NotPanics(t, func() {
		errorMsg("test %s", "message")
	})
}

func TestInfo_DoesNotPanic(t *testing.T) {
	assert.NotPanics(t, func() {
		info("test %s", "message")
	})
}

func TestWarn_DoesNotPanic(t *testing.T) {
	assert.NotPanics(t, func() {
		warn("test %s", "message")
	})
}
