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
	ErrNotFound              = NewError("not found", http.StatusNotFound)
	ErrInvalid               = NewError("invalid", http.StatusBadRequest)
	ErrMethodNotAllowed      = NewError("method not allowed", http.StatusMethodNotAllowed)
	ErrSignatureMismatch     = NewError("url signature mismatch", http.StatusForbidden)
	ErrTimeout               = NewError("timeout", http.StatusRequestTimeout)
	ErrExpired               = NewError("expired", http.StatusGone)
	ErrUnsupportedFormat     = NewError("unsupported format", http.StatusNotAcceptable)
	ErrMaxSizeExceeded       = NewError("maximum size exceeded", http.StatusBadRequest)
	ErrMaxResolutionExceeded = NewError("maximum resolution exceeded", http.StatusUnprocessableEntity)
	ErrTooManyRequests       = NewError("too many requests", http.StatusTooManyRequests)
	ErrInternal              = NewError("internal error", http.StatusInternalServerError)
)

const errPrefix = "imagor:"

var errMsgRegexp = regexp.MustCompile(fmt.Sprintf("^%s ([0-9]+) (.*)$", errPrefix))

type ErrPass struct {
	imagorpath.Params
}

func (p ErrPass) Error() string {
	return fmt.Sprintf("%s pass %s", errPrefix, imagorpath.GeneratePath(p.Params))
}

// Error imagor error convention
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
	if _, ok := err.(ErrPass); ok {
		return ErrUnsupportedFormat // ErrPass till the end means no supported processor
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
