package imagor

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
)

var (
	ErrNotFound          = NewError("not found", http.StatusNotFound)
	ErrPass              = NewError("pass", http.StatusBadRequest)
	ErrHashMismatch      = NewError("hash mismatch", http.StatusForbidden)
	ErrTimeout           = NewError("timeout", http.StatusRequestTimeout)
	ErrMethodNotAllowed  = NewError("method not allowed", http.StatusMethodNotAllowed)
	ErrUnsupportedFormat = NewError("unsupported format", http.StatusNotAcceptable)
)

type Error struct {
	Message string `json:"message,omitempty"`
	Code    int    `json:"status,omitempty"`
}

func (e Error) JSON() []byte {
	buf, _ := json.Marshal(e)
	return buf
}

func (e Error) Error() string {
	return "imagor: " + e.Message
}

func NewError(msg string, code int) Error {
	return Error{Message: msg, Code: code}
}

func wrapError(err error) Error {
	if errors.Is(err, context.DeadlineExceeded) {
		return ErrTimeout
	}
	if e, ok := err.(Error); ok {
		return e
	}
	return NewError(err.Error(), http.StatusInternalServerError)
}
