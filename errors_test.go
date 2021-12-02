package imagor

import (
	"errors"
	"net/http"
	"testing"
)

func TestWrapError(t *testing.T) {
	if err := WrapError(nil); err != nil {
		t.Error(err, "should error nil")
	}
	if err := WrapError(ErrMethodNotAllowed); err != ErrMethodNotAllowed {
		t.Errorf("= %v, should wrap default errors", err)
	}
	if err := NewError("errorrrr", 167); WrapError(errors.New(err.Error())) != err {
		t.Errorf("= %v, should wrap error equals", err)
	}
	if err := WrapError(errors.New("asdfsdfsaf")).(Error); err.Code != http.StatusInternalServerError {
		t.Errorf("= %v, should wrap error fallback", err)
	}
}
