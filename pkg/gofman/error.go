package gofman

import (
	"errors"
	"fmt"
)

// Application error codes.
const (
	ECONFLICT       = "conflict"
	EINTERNAL       = "internal"
	EINVALID        = "invalid"
	ENOTFOUND       = "not_found"
	ENOTIMPLEMENTED = "not_implemented"
	EUNAUTHORIZED   = "unauthorized"
)

// Error represents an application-specific error.
// Any non-application error (disk error, ram error, etc.) will be reported as
// internal error, only logged and not exposed to the end-user.
type Error struct {
	Code    string
	Message string
}

// Error implements the error interface.
func (e *Error) Error() string {
	return fmt.Sprintf("gofman error: code=%s message=%s", e.Code, e.Message)
}

// ErrorCode returns the application error code.
func ErrorCode(err error) string {
	var e *Error
	if err == nil {
		return ""
	} else if errors.As(err, &e) {
		return e.Code
	} else {
		return EINTERNAL
	}
}

// ErrorMessage returns the application error message.
func ErrorMessage(err error) string {
	var e *Error
	if err == nil {
		return ""
	} else if errors.As(err, &e) {
		return e.Message
	} else {
		return "Internal error."
	}
}

// NewError is a helper function to return an Error with a given code and formatted message.
func NewError(code string, format string, args ...interface{}) *Error {
	return &Error{
		Code:    code,
		Message: fmt.Sprintf(format, args...),
	}
}
