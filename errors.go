package imagor

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

var (
	ErrNotFound          = NewError("not found", http.StatusNotFound)
	ErrPass              = NewError("pass", http.StatusBadRequest)
	ErrMethodNotAllowed  = NewError("method not allowed", http.StatusMethodNotAllowed)
	ErrHashMismatch      = NewError("hash mismatch", http.StatusForbidden)
	ErrTimeout           = NewError("timeout", http.StatusRequestTimeout)
	ErrUnsupportedFormat = NewError("unsupported format", http.StatusNotAcceptable)
	ErrUnknown           = NewError("unknown", http.StatusInternalServerError)
)

var errorMap = (func() map[string]Error {
	m := make(map[string]Error, 0)
	for _, err := range []Error{
		ErrNotFound,
		ErrPass,
		ErrMethodNotAllowed,
		ErrHashMismatch,
		ErrTimeout,
		ErrUnsupportedFormat,
		ErrUnknown,
	} {
		m[err.Error()] = err
	}
	return m
})()

type Error struct {
	Message string `json:"message,omitempty"`
	Code    int    `json:"status,omitempty"`
}

func (e Error) Error() string {
	return fmt.Sprintf("imagor: %d %s", e.Code, e.Message)
}

func NewError(msg string, code int) Error {
	return Error{Message: msg, Code: code}
}

func wrapError(err error) Error {
	if err == nil {
		return ErrUnknown
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return ErrTimeout
	}
	if e, ok := err.(Error); ok {
		return e
	}
	if e, ok := errorMap[err.Error()]; ok {
		return e
	}
	msg := strings.Replace(err.Error(), "\n", "", -1)
	return NewError(msg, http.StatusInternalServerError)
}
