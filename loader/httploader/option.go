package httploader

import (
	"crypto/tls"
	"net/http"
	"strings"
)

type Option func(h *HTTPLoader)

func WithTransport(transport http.RoundTripper) Option {
	return func(h *HTTPLoader) {
		if transport != nil {
			h.Transport = transport
		}
	}
}

func WithInsecureSkipVerifyTransport(enabled bool) Option {
	return func(h *HTTPLoader) {
		if enabled {
			transport := http.DefaultTransport.(*http.Transport).Clone()
			transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
			h.Transport = transport
		}
	}
}

func WithForwardHeaders(headers ...string) Option {
	return func(h *HTTPLoader) {
		for _, raw := range headers {
			splits := strings.Split(raw, ",")
			for _, header := range splits {
				header = strings.TrimSpace(header)
				if len(header) > 0 {
					h.ForwardHeaders = append(h.ForwardHeaders, header)
				}
			}
		}
	}
}

func WithForwardAllHeaders(enabled bool) Option {
	return func(h *HTTPLoader) {
		if enabled {
			h.ForwardHeaders = []string{"*"}
		}
	}
}

func WithForwardUserAgent(enabled bool) Option {
	return func(h *HTTPLoader) {
		if enabled {
			h.ForwardHeaders = append(h.ForwardHeaders, "User-Agent")
		}
	}
}

func WithOverrideHeader(name, value string) Option {
	return func(h *HTTPLoader) {
		h.OverrideHeaders[name] = value
	}
}

func WithAllowedSources(hosts ...string) Option {
	return func(h *HTTPLoader) {
		for _, raw := range hosts {
			splits := strings.Split(raw, ",")
			for _, host := range splits {
				host = strings.TrimSpace(host)
				if len(host) > 0 {
					h.AllowedSources = append(h.AllowedSources, host)
				}
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
