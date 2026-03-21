// Package validate provides generic, domain-agnostic input validation helpers.
package validate

import (
	"errors"
	"strings"
)

// ErrBlank is returned when a required string value is empty or whitespace-only.
var ErrBlank = errors.New("value must not be blank")

// NotBlank returns ErrBlank if s is empty or consists entirely of whitespace.
func NotBlank(s string) error {
	if strings.TrimSpace(s) == "" {
		return ErrBlank
	}
	return nil
}
