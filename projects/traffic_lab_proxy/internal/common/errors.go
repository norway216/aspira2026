package common

import (
	"fmt"
	"net/http"
)

// AppError represents an application-level error with an HTTP status code.
type AppError struct {
	Code    int    `json:"-"`
	Message string `json:"error"`
	Err     error  `json:"-"`
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

func (e *AppError) Unwrap() error {
	return e.Err
}

// Convenience constructors.

func NewBadRequest(msg string) *AppError {
	return &AppError{Code: http.StatusBadRequest, Message: msg}
}

func NewUnauthorized(msg string) *AppError {
	return &AppError{Code: http.StatusUnauthorized, Message: msg}
}

func NewForbidden(msg string) *AppError {
	return &AppError{Code: http.StatusForbidden, Message: msg}
}

func NewNotFound(msg string) *AppError {
	return &AppError{Code: http.StatusNotFound, Message: msg}
}

func NewConflict(msg string) *AppError {
	return &AppError{Code: http.StatusConflict, Message: msg}
}

func NewInternal(msg string, err error) *AppError {
	return &AppError{Code: http.StatusInternalServerError, Message: msg, Err: err}
}

func NewTooManyRequests(msg string) *AppError {
	return &AppError{Code: http.StatusTooManyRequests, Message: msg}
}

// WrapError wraps a generic error into an AppError.
func WrapError(err error) *AppError {
	if ae, ok := err.(*AppError); ok {
		return ae
	}
	return NewInternal("internal server error", err)
}
