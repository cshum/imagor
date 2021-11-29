package httploader

import (
	"net/http"
	"net/url"
)

type Option func(h *HTTPLoader)

func New(options ...Option) *HTTPLoader {
	h := &HTTPLoader{
		OverrideHeaders: map[string]string{},
	}
	for _, option := range options {
		option(h)
	}
	return h
}

func WithTransport(transport http.RoundTripper) Option {
	return func(h *HTTPLoader) {
		h.Transport = transport
	}
}

func WithForwardHeaders(headers ...string) Option {
	return func(h *HTTPLoader) {
		h.ForwardHeaders = append(h.ForwardHeaders, headers...)
	}
}

func WithOverrideHeader(name, value string) Option {
	return func(h *HTTPLoader) {
		h.OverrideHeaders[name] = value
	}
}

func WithAllowedOrigins(urls ...string) Option {
	return func(h *HTTPLoader) {
		for _, rawUrl := range urls {
			if u, err := url.Parse(rawUrl); err == nil {
				h.AllowedOrigins = append(h.AllowedOrigins, u)
			}
		}
	}
}

func WithMaxAllowedSize(maxAllowedSize int) Option {
	return func(h *HTTPLoader) {
		h.MaxAllowedSize = maxAllowedSize
	}
}
