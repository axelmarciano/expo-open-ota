// Package validation holds the input-validation rules for user-supplied
// dashboard values (resource names, display labels). Services call these before
// persisting so bad input fails fast with a caller-facing 400 instead of a deep
// store/bucket error — or, worse, a malformed row/key that only breaks later.
package validation

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

// maxNameLen caps resource names used as storage-path segments. Kept equal to
// the bucket layer's maxSegmentLen so a name accepted here can never be
// rejected later when an update is written to the bucket.
const maxNameLen = 128

// maxDisplayNameLen caps human-facing labels (app name, API key name). These
// are never path segments, so the limit is only about sane storage/UI bounds.
const maxDisplayNameLen = 255

// Error is a validation failure on user-supplied input. Handlers detect it with
// errors.As and map it to HTTP 400, surfacing Message to the caller, while
// unrecognized errors stay a 500.
type Error struct {
	Field   string
	Message string
}

func (e *Error) Error() string {
	if e.Field == "" {
		return e.Message
	}
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// IsValidationError reports whether err is (or wraps) a validation Error.
func IsValidationError(err error) bool {
	var ve *Error
	return errors.As(err, &ve)
}

func fail(field, format string, args ...any) *Error {
	return &Error{Field: field, Message: fmt.Sprintf(format, args...)}
}

// Errorf builds a validation Error, e.g. to wrap a lower-level validator's
// message (config.ValidateKeys) so it still maps to a 400.
func Errorf(field, format string, args ...any) *Error {
	return fail(field, format, args...)
}

// Name validates a resource name used as a single storage-path segment and DB
// value (branch, channel, release channel).
//
// The rules are a mirror of internal/bucket.validateSegment — keep the two in
// sync. Rejects empties, path separators, "." / "..", null bytes, control
// characters, and anything over maxNameLen.
func Name(field, value string) error {
	if value == "" {
		return fail(field, "must not be empty")
	}
	if len(value) > maxNameLen {
		return fail(field, "exceeds max length %d", maxNameLen)
	}
	if strings.ContainsAny(value, "/\\") {
		return fail(field, "must not contain path separators")
	}
	if value == "." || value == ".." {
		return fail(field, "%q is reserved", value)
	}
	for _, r := range value {
		if r == 0x00 {
			return fail(field, "must not contain null bytes")
		}
		if unicode.IsControl(r) {
			return fail(field, "must not contain control characters")
		}
	}
	return nil
}

// DisplayName validates a human-facing label (app name, API key name). Looser
// than Name — it is never a path segment, so spaces and unicode are allowed —
// but still non-empty (ignoring surrounding whitespace), bounded, and free of
// control characters (log / UI injection).
func DisplayName(field, value string) error {
	if strings.TrimSpace(value) == "" {
		return fail(field, "must not be empty")
	}
	if len(value) > maxDisplayNameLen {
		return fail(field, "exceeds max length %d", maxDisplayNameLen)
	}
	for _, r := range value {
		if r == 0x00 {
			return fail(field, "must not contain null bytes")
		}
		if unicode.IsControl(r) && r != '\t' {
			return fail(field, "must not contain control characters")
		}
	}
	return nil
}

// NumericID validates a positive integer id passed as a string (branch id, API
// key id). Guards the string→int64 conversion the stores do so a malformed id
// fails with a 400 instead of a 500.
func NumericID(field, value string) error {
	if value == "" {
		return fail(field, "must not be empty")
	}
	n, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return fail(field, "must be a numeric id")
	}
	if n <= 0 {
		return fail(field, "must be a positive id")
	}
	return nil
}
