package server

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
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

func TestRealIP(t *testing.T) {
	tests := []struct {
		name       string
		remoteAddr string
		xRealIP    string
		xForwarded string
		expected   string
	}{
		{
			name:       "remote addr with port",
			remoteAddr: "203.0.113.1:8080",
			expected:   "203.0.113.1",
		},
		{
			name:       "remote addr without port",
			remoteAddr: "203.0.113.2",
			expected:   "203.0.113.2",
		},
		{
			name:       "x-forwarded-for picks first public ip",
			remoteAddr: "127.0.0.1:1234",
			xForwarded: "10.0.0.1, 198.51.100.7, 192.168.0.9",
			expected:   "198.51.100.7",
		},
		{
			name:       "x-forwarded-for with invalid and private falls back to x-real-ip",
			remoteAddr: "127.0.0.1:1234",
			xRealIP:    "198.51.100.8",
			xForwarded: "not-an-ip, 10.0.0.2",
			expected:   "198.51.100.8",
		},
		{
			name:       "x-real-ip used when x-forwarded-for empty",
			remoteAddr: "127.0.0.1:1234",
			xRealIP:    "198.51.100.9",
			expected:   "198.51.100.9",
		},
		{
			name:       "invalid x-forwarded-for and empty x-real-ip returns empty",
			remoteAddr: "127.0.0.1:1234",
			xForwarded: "not-an-ip",
			expected:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptestRequest(tt.remoteAddr, tt.xRealIP, tt.xForwarded)
			assert.Equal(t, tt.expected, RealIP(r))
		})
	}
}

func httptestRequest(remoteAddr, xRealIP, xForwardedFor string) *http.Request {
	r := &http.Request{Header: make(http.Header), RemoteAddr: remoteAddr}
	if xRealIP != "" {
		r.Header.Set("X-Real-Ip", xRealIP)
	}
	if xForwardedFor != "" {
		r.Header.Set("X-Forwarded-For", xForwardedFor)
	}
	return r
}
