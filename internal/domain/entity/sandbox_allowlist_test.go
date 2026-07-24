package entity

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNormalizeE164(t *testing.T) {
	tests := []struct {
		input string
		want  string
		ok    bool
	}{
		// The same Brazilian number in every common shape must canonicalize
		// identically: a mismatch here either blocks a legitimate test
		// recipient or lets an unlisted one through.
		{"+55 44 99999-9999", "+5544999999999", true},
		{"5544999999999", "+5544999999999", true},
		{"+5544999999999", "+5544999999999", true},
		{"(55) 44 9.9999-9999", "+5544999999999", true},
		{" +1 (555) 123-4567 ", "+15551234567", true},

		{"", "", false},
		{"abc", "", false},
		{"+55 44 abc", "", false},
		{"0123456789", "", false},        // leading zero is not E.164
		{"1234567", "", false},           // too short
		{"+1234567890123456", "", false}, // 16 digits, over E.164 max
	}
	for _, tt := range tests {
		got, ok := NormalizeE164(tt.input)
		assert.Equal(t, tt.ok, ok, "input %q", tt.input)
		assert.Equal(t, tt.want, got, "input %q", tt.input)
	}
}
