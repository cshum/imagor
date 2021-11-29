package httploader

import (
	"net/http"
	"net/url"
)

type Option func(h *httpLoader)

func WithTransport(transport http.RoundTripper) Option {
	return func(h *httpLoader) {
		h.Transport = transport
	}
}

func WithForwardHeaders(headers ...string) Option {
	return func(h *httpLoader) {
		h.ForwardHeaders = append(h.ForwardHeaders, headers...)
	}
}

func WithOverrideHeader(name, value string) Option {
	return func(h *httpLoader) {
		h.OverrideHeaders[name] = value
	}
}

func WithAllowedOrigins(urls ...string) Option {
	return func(h *httpLoader) {
		for _, rawUrl := range urls {
			if u, err := url.Parse(rawUrl); err == nil {
				h.AllowedOrigins = append(h.AllowedOrigins, u)
			}
		}
	}
}

func WithMaxAllowedSize(maxAllowedSize int) Option {
	return func(h *httpLoader) {
		h.MaxAllowedSize = maxAllowedSize
	}
}

func WithAutoScheme(autoScheme bool) Option {
	return func(h *httpLoader) {
		h.AutoScheme = autoScheme
	}
}
