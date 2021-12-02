package imagor

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
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

var errMsgRegexp = regexp.MustCompile("^imagor: ([0-9]+) (.*)$")

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

func WrapError(err error) error {
	if err == nil {
		return nil
	}
	if e, ok := err.(Error); ok {
		return e
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return ErrTimeout
	}
	if msg := err.Error(); errMsgRegexp.MatchString(msg) {
		if match := errMsgRegexp.FindStringSubmatch(msg); len(match) == 3 {
			code, _ := strconv.Atoi(match[1])
			return NewError(match[2], code)
		}
	}
	msg := strings.Replace(err.Error(), "\n", "", -1)
	return NewError(msg, http.StatusInternalServerError)
}
