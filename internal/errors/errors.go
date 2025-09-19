package errors

import "fmt"

// Error is a custom error type that contains a code and a message.
type Error struct {
	Code    int
	Message string
}

// Error returns the error message.
func (e *Error) Error() string {
	return e.Message
}

// New creates a new error.
func New(code int, message string) *Error {
	return &Error{
		Code:    code,
		Message: message,
	}
}

// Newf creates a new error with a formatted message.
func Newf(code int, format string, a ...interface{}) *Error {
	return &Error{
		Code:    code,
		Message: fmt.Sprintf(format, a...),
	}
}

// Error codes
const (
	// ErrUnknown is an unknown error.
	ErrUnknown = iota
	// ErrNotFound is a not found error.
	ErrNotFound
	// ErrForbidden is a forbidden error.
	ErrForbidden
	// ErrBadRequest is a bad request error.
	ErrBadRequest
	// ErrUnauthorized is an unauthorized error.
	ErrUnauthorized
	// ErrInternal is an internal error.
	ErrInternal
)
