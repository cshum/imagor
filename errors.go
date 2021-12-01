package imagor

import (
	"context"
	"errors"
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
)

type Error struct {
	Message string `json:"message,omitempty"`
	Code    int    `json:"status,omitempty"`
}

func (e Error) Error() string {
	return "imagor: " + e.Message
}

func NewError(msg string, code int) Error {
	return Error{Message: msg, Code: code}
}

func wrapError(err error) Error {
	if err == ErrPass {
		// passed till the end means no handler
		return ErrMethodNotAllowed
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return ErrTimeout
	}
	if e, ok := err.(Error); ok {
		return e
	}
	msg := strings.Replace(err.Error(), "\n", "", -1)
	return NewError(msg, http.StatusInternalServerError)
}
