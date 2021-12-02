package imagor

import (
	"errors"
	"net/http"
	"testing"
)

func TestWrapError(t *testing.T) {
	if err := WrapError(nil); err != ErrUnknown {
		t.Error(err)
	}
	if err := NewError("errorrrr", 167); WrapError(err) != err {
		t.Errorf("= %v, should wrap error equals", err)
	}
	if err := WrapError(
		errors.New(ErrMethodNotAllowed.Error()),
	); err != ErrMethodNotAllowed {
		t.Errorf("= %v, should wrap error", err)
	}
	if err := WrapError(errors.New("asdfsdfsaf")); err.Code != http.StatusInternalServerError {
		t.Errorf("= %v, should wrap error fallback", err)
	}
}
