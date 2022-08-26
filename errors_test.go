package imagor

import (
	"context"
	"errors"
	"github.com/cshum/imagor/imagorpath"
	"github.com/stretchr/testify/assert"
	"net"
	"net/http"
	"net/url"
	"testing"
)

func TestWrapError(t *testing.T) {
	var err error
	var e Error

	assert.Equal(t, WrapError(nil), ErrInternal)

	assert.Equal(t, ErrMethodNotAllowed, WrapError(ErrMethodNotAllowed))

	err = NewError("errorrrr", 167)
	assert.Equal(t, WrapError(errors.New(err.Error())), err)

	assert.Equal(t, ErrTimeout, WrapError(context.DeadlineExceeded))

	assert.Equal(t, true, ErrTimeout.Timeout())

	assert.Equal(t, ErrTimeout, WrapError(&url.Error{Err: context.DeadlineExceeded}))

	err = errors.New("asdfsdfsaf")
	e = WrapError(err)
	assert.Equal(t, 500, e.Code)
	assert.Contains(t, e.Error(), err.Error())

	e = NewErrorFromStatusCode(403)
	assert.Equal(t, 403, e.Code)
	assert.Contains(t, e.Error(), http.StatusText(403))

	err = &net.DNSError{IsTimeout: true}
	assert.Equal(t, ErrTimeout, WrapError(err))

	err = ErrForward{imagorpath.Params{Width: 167, Height: 169, Image: "foo"}}
	assert.Equal(t, "167x169/foo", err.Error())

}
