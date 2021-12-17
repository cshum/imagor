package server

import (
	"testing"
)

func TestIsPrivateIP(t *testing.T) {
	if isPrivate, err := IsPrivateIP("1.1.1.1"); isPrivate || err != nil {
		t.Error("should not private ip")
	}
	if isPrivate, err := IsPrivateIP("10.8.0.1"); !isPrivate || err != nil {
		t.Error("should private ip")
	}
	if isPrivate, err := IsPrivateIP("100.112.193.54"); !isPrivate || err != nil {
		t.Error("should private ip")
	}
	if _, err := IsPrivateIP("asdf"); err == nil {
		t.Error("should error for invalid address")
	}
}
