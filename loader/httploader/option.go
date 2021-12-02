package httploader

import (
	"crypto/tls"
	"net/http"
	"net/url"
)

type Option func(h *HTTPLoader)

func WithTransport(transport http.RoundTripper) Option {
	return func(h *HTTPLoader) {
		if transport != nil {
			h.Transport = transport
		}
	}
}

func WithInsecureSkipVerifyTransport(enable bool) Option {
	return func(h *HTTPLoader) {
		if enable {
			transport := http.DefaultTransport.(*http.Transport).Clone()
			transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
			h.Transport = transport
		}
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
		if maxAllowedSize > 0 {
			h.MaxAllowedSize = maxAllowedSize
		}
	}
}
