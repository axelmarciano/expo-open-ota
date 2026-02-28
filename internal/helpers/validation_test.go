package helpers

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateResourceName_ValidNames(t *testing.T) {
	validNames := []string{
		"main",
		"production",
		"staging",
		"v2.1.2",
		"staging+2",
		"release-1.0",
		"my_branch",
		"user@feature",
		"1",
		"123456789",
		"a",
		"A.B.C+d-e_f@g",
	}
	for _, name := range validNames {
		assert.NoError(t, ValidateResourceName(name, "test"), "expected %q to be valid", name)
	}
}

func TestValidateResourceName_InvalidNames(t *testing.T) {
	tests := []struct {
		name   string
		reason string
	}{
		{"", "empty string"},
		{"../etc/passwd", "path traversal with ../"},
		{"branch/name", "contains slash"},
		{"branch\\name", "contains backslash"},
		{".hidden", "starts with dot"},
		{"-flag", "starts with hyphen"},
		{"+plus", "starts with plus"},
		{"branch name", "contains space"},
		{"branch\x00name", "contains null byte"},
		{"branch\nname", "contains newline"},
		{"branch{name}", "contains braces"},
		{`branch"name`, "contains double quote"},
	}
	for _, tt := range tests {
		assert.Error(t, ValidateResourceName(tt.name, "test"), "expected %q to be invalid (%s)", tt.name, tt.reason)
	}
}

func TestValidateResourceName_TooLong(t *testing.T) {
	longName := strings.Repeat("a", 130)
	assert.Error(t, ValidateResourceName(longName, "test"), "expected name longer than 128 chars to be invalid")
}

func TestValidateResourceName_MaxLength(t *testing.T) {
	name := strings.Repeat("a", 128)
	assert.NoError(t, ValidateResourceName(name, "test"), "expected 128 char name to be valid")
}
