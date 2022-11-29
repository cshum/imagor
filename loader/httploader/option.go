package httploader

import (
	"crypto/tls"
	"net"
	"net/http"
	"strings"
)

// Option HTTPLoader option
type Option func(h *HTTPLoader)

// WithTransport with custom http.RoundTripper transport option
func WithTransport(transport http.RoundTripper) Option {
	return func(h *HTTPLoader) {
		if transport != nil {
			h.Transport = transport
		}
	}
}

// WithProxyTransport with random proxy rotation option for selected proxy URLs
func WithProxyTransport(proxyURLs, hosts string) Option {
	return func(h *HTTPLoader) {
		if proxyURLs != "" {
			if t, ok := h.Transport.(*http.Transport); ok {
				t.Proxy = randomProxyFunc(proxyURLs, hosts)
				h.Transport = t
			}
		}
	}
}

// WithInsecureSkipVerifyTransport with insecure HTTPs option
func WithInsecureSkipVerifyTransport(enabled bool) Option {
	return func(h *HTTPLoader) {
		if enabled {
			if t, ok := h.Transport.(*http.Transport); ok {
				t.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
				h.Transport = t
			}
		}
	}
}

// WithForwardHeaders with forward selected request headers option
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

// WithForwardClientHeaders with forward browser request headers option
func WithForwardClientHeaders(enabled bool) Option {
	return func(h *HTTPLoader) {
		if enabled {
			h.ForwardHeaders = []string{"*"}
		}
	}
}

// WithOverrideHeader with override request header with name value pair option
func WithOverrideHeader(name, value string) Option {
	return func(h *HTTPLoader) {
		h.OverrideHeaders[name] = value
	}
}

// WithAllowedSources with allowed source hosts option.
// Accept csv wth glob pattern e.g. *.google.com,*.github.com
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

// WithMaxAllowedSize with maximum allowed size option
func WithMaxAllowedSize(maxAllowedSize int) Option {
	return func(h *HTTPLoader) {
		if maxAllowedSize > 0 {
			h.MaxAllowedSize = maxAllowedSize
		}
	}
}

// WithUserAgent with custom user agent option
func WithUserAgent(userAgent string) Option {
	return func(h *HTTPLoader) {
		if userAgent != "" {
			h.UserAgent = userAgent
		}
	}
}

// WithAccept with accepted content type option
func WithAccept(contentType string) Option {
	return func(h *HTTPLoader) {
		if contentType != "" {
			h.Accept = contentType
		}
	}
}

// WithDefaultScheme with default URL scheme option https or http, if not specified
func WithDefaultScheme(scheme string) Option {
	return func(h *HTTPLoader) {
		if scheme != "" {
			h.DefaultScheme = scheme
		}
	}
}

// WithBlockLoopbackNetworks with option to reject HTTP connections
// to loopback network IP addresses
func WithBlockLoopbackNetworks(enabled bool) Option {
	return func(h *HTTPLoader) {
		if enabled {
			h.BlockLoopbackNetworks = true
		}
	}
}

// WithBlockLinkLocalNetworks with option to reject HTTP connections
// to link local IP addresses
func WithBlockLinkLocalNetworks(enabled bool) Option {
	return func(h *HTTPLoader) {
		if enabled {
			h.BlockLinkLocalNetworks = true
		}
	}
}

// WithBlockPrivateNetworks with option to reject HTTP connections
// to private network IP addresses
func WithBlockPrivateNetworks(enabled bool) Option {
	return func(h *HTTPLoader) {
		if enabled {
			h.BlockPrivateNetworks = true
		}
	}
}

// WithBlockNetworks with option to reject
// HTTP connections to a configurable list of networks
func WithBlockNetworks(networks ...*net.IPNet) Option {
	return func(h *HTTPLoader) {
		h.BlockNetworks = networks
	}
}
