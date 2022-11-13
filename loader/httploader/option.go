package httploader

import (
	"crypto/tls"
	"net"
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

func WithForwardClientHeaders(enabled bool) Option {
	return func(h *HTTPLoader) {
		if enabled {
			h.ForwardHeaders = []string{"*"}
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

func WithUserAgent(userAgent string) Option {
	return func(h *HTTPLoader) {
		if userAgent != "" {
			h.UserAgent = userAgent
		}
	}
}

func WithAccept(contentType string) Option {
	return func(h *HTTPLoader) {
		if contentType != "" {
			h.Accept = contentType
		}
	}
}

func WithDefaultScheme(scheme string) Option {
	return func(h *HTTPLoader) {
		if scheme != "" {
			h.DefaultScheme = scheme
		}
	}
}

func WithBlockLoopbackNetworks(enabled bool) Option {
	return func(h *HTTPLoader) {
		if enabled {
			h.BlockLoopbackNetworks = true
		}
	}
}

func WithBlockLinkLocalNetworks(enabled bool) Option {
	return func(h *HTTPLoader) {
		if enabled {
			h.BlockLinkLocalNetworks = true
		}
	}
}

func WithBlockPrivateNetworks(enabled bool) Option {
	return func(h *HTTPLoader) {
		if enabled {
			h.BlockPrivateNetworks = true
		}
	}
}

func WithBlockNetworks(networks ...*net.IPNet) Option {
	return func(h *HTTPLoader) {
		h.BlockNetworks = networks
	}
}
