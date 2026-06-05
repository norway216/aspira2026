package util

import (
	"fmt"
	"net/mail"
	"strings"
)

// ValidationError represents a single validation error.
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// ValidationErrors is a collection of validation errors.
type ValidationErrors []ValidationError

func (ve ValidationErrors) Error() string {
	var msgs []string
	for _, e := range ve {
		msgs = append(msgs, fmt.Sprintf("%s: %s", e.Field, e.Message))
	}
	return strings.Join(msgs, "; ")
}

// HasErrors returns true if there are any validation errors.
func (ve ValidationErrors) HasErrors() bool {
	return len(ve) > 0
}

// ValidateRequired validates a required field.
func ValidateRequired(field, value string, errs *ValidationErrors) {
	if strings.TrimSpace(value) == "" {
		*errs = append(*errs, ValidationError{Field: field, Message: "is required"})
	}
}

// ValidateMaxLength validates maximum string length.
func ValidateMaxLength(field, value string, max int, errs *ValidationErrors) {
	if len([]rune(value)) > max {
		*errs = append(*errs, ValidationError{
			Field:   field,
			Message: fmt.Sprintf("must be at most %d characters", max),
		})
	}
}

// ValidateMinLength validates minimum string length.
func ValidateMinLength(field, value string, min int, errs *ValidationErrors) {
	if len([]rune(strings.TrimSpace(value))) < min {
		*errs = append(*errs, ValidationError{
			Field:   field,
			Message: fmt.Sprintf("must be at least %d characters", min),
		})
	}
}

// ValidateEmail validates an email address.
func ValidateEmail(field, value string, errs *ValidationErrors) {
	if strings.TrimSpace(value) == "" {
		return // optional; use ValidateRequired for mandatory
	}
	if _, err := mail.ParseAddress(value); err != nil {
		*errs = append(*errs, ValidationError{Field: field, Message: "must be a valid email address"})
	}
}

// ValidateSlug validates a slug format.
func ValidateSlug(field, value string, errs *ValidationErrors) {
	if strings.TrimSpace(value) == "" {
		return
	}
	for _, r := range value {
		if !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-') {
			*errs = append(*errs, ValidationError{
				Field:   field,
				Message: "must contain only lowercase letters, numbers, and hyphens",
			})
			return
		}
	}
}

// ValidateURL validates a URL field (basic check).
func ValidateURL(field, value string, errs *ValidationErrors) {
	if strings.TrimSpace(value) == "" {
		return
	}
	if !strings.HasPrefix(value, "http://") && !strings.HasPrefix(value, "https://") {
		*errs = append(*errs, ValidationError{
			Field:   field,
			Message: "must start with http:// or https://",
		})
	}
}
