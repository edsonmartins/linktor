package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVersionCmd_Use(t *testing.T) {
	assert.Equal(t, "version", versionCmd.Use)
}

func TestVersionCmd_ShortNotEmpty(t *testing.T) {
	assert.NotEmpty(t, versionCmd.Short)
}

func TestVersionCmd_LongNotEmpty(t *testing.T) {
	assert.NotEmpty(t, versionCmd.Long)
}

func TestVersionCmd_HasRunFunction(t *testing.T) {
	assert.NotNil(t, versionCmd.Run)
}

func TestVersionVars_Exist(t *testing.T) {
	// Package-level variables should have default values set in root.go
	assert.NotEmpty(t, Version)
	assert.NotEmpty(t, Commit)
	assert.NotEmpty(t, BuildDate)
}

func TestVersionVars_DefaultValues(t *testing.T) {
	// Default values set in root.go (unless overridden at build time)
	assert.Equal(t, "dev", Version)
	assert.Equal(t, "none", Commit)
	assert.Equal(t, "unknown", BuildDate)
}
