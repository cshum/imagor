package httpsource

import (
	"crypto/tls"
	"fmt"
	"github.com/cshum/imagor"
	"io"
	"net/http"
	"net/url"
	"strings"
)

var (
	// InsecureTransport DefaultTransport with InsecureSkipVerify
	InsecureTransport = newInsecureSkipVerifyTransport()
)

type HTTPSource struct {
	HTTPTransport   *http.Transport
	ForwardHeaders  []string
	OverrideHeaders map[string]string
	AllowedOrigins  []*url.URL
	MaxAllowedSize  int
}

func (h HTTPSource) Match(r *http.Request, key string) bool {
	if r.Method == http.MethodGet && key != "" {
		if u, err := url.Parse(key); err == nil && u.Host != "" && u.Scheme != "" {
			return true
		}
	}
	return false
}

func (h HTTPSource) Do(r *http.Request, key string) ([]byte, error) {
	u, err := url.Parse(key)
	if err != nil {
		return nil, err
	}
	if shouldRestrictOrigin(u, h.AllowedOrigins) {
		return nil, fmt.Errorf("not allowed remote URL origin: %s%s", u.Host, u.Path)
	}
	client := &http.Client{Transport: h.HTTPTransport}
	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, key, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "imagor/"+imagor.Version)

	for _, header := range h.ForwardHeaders {
		if header == "*" {
			req.Header = r.Header
			break
		}
		if _, ok := r.Header[header]; ok {
			req.Header.Set(header, r.Header.Get(header))
		}
	}
	for key, value := range h.OverrideHeaders {
		req.Header.Set(key, value)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	buf, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return buf, nil
}

func shouldRestrictOrigin(url *url.URL, origins []*url.URL) bool {
	if len(origins) == 0 {
		return false
	}
	for _, origin := range origins {
		if origin.Host == url.Host {
			if strings.HasPrefix(url.Path, origin.Path) {
				return false
			}
		}
		if origin.Host[0:2] == "*." {
			// Testing if "*.example.org" matches "example.org"
			if url.Host == origin.Host[2:] {
				if strings.HasPrefix(url.Path, origin.Path) {
					return false
				}
			}
			// Testing if "*.example.org" matches "foo.example.org"
			if strings.HasSuffix(url.Host, origin.Host[1:]) {
				if strings.HasPrefix(url.Path, origin.Path) {
					return false
				}
			}
		}
	}
	return true
}

func newInsecureSkipVerifyTransport() *http.Transport {
	tr := http.DefaultTransport.(*http.Transport).Clone()
	tr.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	return tr
}
