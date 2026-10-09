package utils

import "fmt"

// RuntimeError is a structured error used across the container runtime.
type RuntimeError struct {
	Code    string
	Message string
}

func (e *RuntimeError) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// NewError creates a new structured runtime error.
func NewError(code, msg string) error {
	return &RuntimeError{
		Code:    code,
		Message: msg,
	}
}

// WrapError adds context to an existing error.
func WrapError(code string, err error) error {
	return &RuntimeError{
		Code:    code,
		Message: err.Error(),
	}
}
