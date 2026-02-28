package helpers

import (
	"fmt"
	"regexp"
)

// validResourceName allows alphanumeric, dots, underscores, plus, at, hyphens.
// Must start with an alphanumeric character. Max 128 chars.
var validResourceName = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._+@-]*$`)

func ValidateResourceName(name string, label string) error {
	if name == "" {
		return fmt.Errorf("%s cannot be empty", label)
	}
	if len(name) > 128 {
		return fmt.Errorf("%s is too long (max 128 characters)", label)
	}
	if !validResourceName.MatchString(name) {
		return fmt.Errorf("%s contains invalid characters (allowed: alphanumeric, dots, underscores, plus, at, hyphens; must start with alphanumeric)", label)
	}
	return nil
}
