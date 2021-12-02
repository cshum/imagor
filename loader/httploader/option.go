package httploader

import (
	"crypto/tls"
	"net/http"
	"net/url"
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

func WithAllowedSources(urls ...string) Option {
	return func(h *HTTPLoader) {
		for _, raw := range urls {
			rawUrls := strings.Split(raw, ",")
			for _, rawUrl := range rawUrls {
				rawUrl = strings.TrimSpace(rawUrl)
				if !strings.Contains(rawUrl, "://") {
					rawUrl = "https://" + rawUrl
				}
				if u, err := url.Parse(rawUrl); err == nil && len(u.Host) > 0 {
					h.AllowedSources = append(h.AllowedSources, u)
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
