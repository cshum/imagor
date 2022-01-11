package httploader

import (
	"crypto/tls"
	"math/rand"
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

func randomProxyFunc(proxyURLs, hosts string) func(*http.Request) (*url.URL, error) {
	var urls []*url.URL
	var allowedSources []string
	for _, split := range strings.Split(proxyURLs, ",") {
		if u, err := url.Parse(strings.TrimSpace(split)); err == nil {
			urls = append(urls, u)
		}
	}
	ln := len(urls)
	for _, host := range strings.Split(hosts, ",") {
		host = strings.TrimSpace(host)
		if len(host) > 0 {
			allowedSources = append(allowedSources, host)
		}
	}
	return func(r *http.Request) (u *url.URL, err error) {
		if len(urls) == 0 {
			return
		}
		if !isURLAllowed(r.URL, allowedSources) {
			return
		}
		u = urls[rand.Intn(ln)]
		return
	}
}

func WithProxyTransport(proxyURLs, hosts string) Option {
	return func(h *HTTPLoader) {
		if proxyURLs != "" {
			t, ok := h.Transport.(*http.Transport)
			if !ok {
				t = http.DefaultTransport.(*http.Transport).Clone()
			}
			t.Proxy = randomProxyFunc(proxyURLs, hosts)
			h.Transport = t
		}
	}
}

func WithInsecureSkipVerifyTransport(enabled bool) Option {
	return func(h *HTTPLoader) {
		if enabled {
			t, ok := h.Transport.(*http.Transport)
			if !ok {
				t = http.DefaultTransport.(*http.Transport).Clone()
			}
			t.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
			h.Transport = t
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

func WithDefaultScheme(scheme string) Option {
	return func(h *HTTPLoader) {
		if scheme != "" {
			h.DefaultScheme = scheme
		}
	}
}
