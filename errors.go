package imagor

import (
	"context"
	"errors"
	"fmt"
	"github.com/cshum/imagor/imagorpath"
	"net/http"
	"regexp"
	"strconv"
	"strings"
)

var (
	// ErrNotFound not found error
	ErrNotFound = NewError("not found", http.StatusNotFound)
	// ErrInvalid syntactic invalid path error
	ErrInvalid = NewError("invalid", http.StatusBadRequest)
	// ErrMethodNotAllowed method not allowed error
	ErrMethodNotAllowed = NewError("method not allowed", http.StatusMethodNotAllowed)
	// ErrSignatureMismatch URL signature mismatch error
	ErrSignatureMismatch = NewError("url signature mismatch", http.StatusForbidden)
	// ErrTimeout timeout error
	ErrTimeout = NewError("timeout", http.StatusRequestTimeout)
	// ErrExpired expire error
	ErrExpired = NewError("expired", http.StatusGone)
	// ErrUnsupportedFormat unsupported format error
	ErrUnsupportedFormat = NewError("unsupported format", http.StatusNotAcceptable)
	// ErrMaxSizeExceeded maximum size exceeded error
	ErrMaxSizeExceeded = NewError("maximum size exceeded", http.StatusBadRequest)
	// ErrMaxResolutionExceeded maximum resolution exceeded error
	ErrMaxResolutionExceeded = NewError("maximum resolution exceeded", http.StatusUnprocessableEntity)
	// ErrTooManyRequests too many requests error
	ErrTooManyRequests = NewError("too many requests", http.StatusTooManyRequests)
	// ErrInternal internal error
	ErrInternal = NewError("internal error", http.StatusInternalServerError)
)

const errPrefix = "imagor:"

var errMsgRegexp = regexp.MustCompile(fmt.Sprintf("^%s ([0-9]+) (.*)$", errPrefix))

// ErrForward indicator passing imagorpath.Params to next processor
type ErrForward struct {
	imagorpath.Params
}

// Error implements error
func (p ErrForward) Error() string {
	return fmt.Sprintf("%s forward %s", errPrefix, imagorpath.GeneratePath(p.Params))
}

// Error imagor error convention
type Error struct {
	Message string `json:"message,omitempty"`
	Code    int    `json:"status,omitempty"`
}

type timeoutErr interface {
	Timeout() bool
}

// Error implements error
func (e Error) Error() string {
	return fmt.Sprintf("%s %d %s", errPrefix, e.Code, e.Message)
}

// Timeout indicates if error is timeout
func (e Error) Timeout() bool {
	return e.Code == http.StatusRequestTimeout || e.Code == http.StatusGatewayTimeout
}

// NewError creates imagor Error from message and status code
func NewError(msg string, code int) Error {
	return Error{Message: msg, Code: code}
}

// NewErrorFromStatusCode creates imagor Error solely from status code
func NewErrorFromStatusCode(code int) Error {
	return NewError(http.StatusText(code), code)
}

// WrapError wraps Go error into imagor Error
func WrapError(err error) Error {
	if err == nil {
		return ErrInternal
	}
	if e, ok := err.(Error); ok {
		return e
	}
	if _, ok := err.(ErrForward); ok {
		// ErrForward till the end means no supported processor
		return ErrUnsupportedFormat
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
