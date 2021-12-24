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
	ErrSignatureMismatch = NewError("url signature mismatch", http.StatusForbidden)
	ErrTimeout           = NewError("timeout", http.StatusRequestTimeout)
	ErrUnsupportedFormat = NewError("unsupported format", http.StatusNotAcceptable)
	ErrMaxSizeExceeded   = NewError("maximum size exceeded", http.StatusBadRequest)
	ErrDeadlock          = NewError("deadlock", http.StatusInternalServerError)
	ErrInternal          = NewError("internal error", http.StatusInternalServerError)
)

const errPrefix = "imagor:"

var errMsgRegexp = regexp.MustCompile(fmt.Sprintf("^%s ([0-9]+) (.*)$", errPrefix))

type Error struct {
	Message string `json:"message,omitempty"`
	Code    int    `json:"status,omitempty"`
}

type timeoutErr interface {
	Timeout() bool
}

func (e Error) Error() string {
	return fmt.Sprintf("%s %d %s", errPrefix, e.Code, e.Message)
}

func (e Error) Timeout() bool {
	return e.Code == http.StatusRequestTimeout || e.Code == http.StatusGatewayTimeout
}

func NewError(msg string, code int) Error {
	return Error{Message: msg, Code: code}
}

func NewErrorFromStatusCode(code int) Error {
	return NewError(http.StatusText(code), code)
}

func WrapError(err error) error {
	if err == nil {
		return nil
	}
	if e, ok := err.(Error); ok {
		return e
	}
	if e, ok := err.(timeoutErr); ok {
		if e.Timeout() {
			return ErrTimeout
		}
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
