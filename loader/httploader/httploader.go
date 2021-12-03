package httploader

import (
	"github.com/cshum/imagor"
	"io"
	"net/http"
	"net/url"
	"path"
)

type HTTPLoader struct {
	// The Transport used to request images.
	// If nil, http.DefaultTransport is used.
	Transport http.RoundTripper

	ForwardHeaders []string

	OverrideHeaders map[string]string

	// AllowedSources list of host names allowed to load from,
	// supports glob patterns such as *.google.com
	AllowedSources []string

	MaxAllowedSize int
}

func New(options ...Option) *HTTPLoader {
	h := &HTTPLoader{
		OverrideHeaders: map[string]string{},
	}
	for _, option := range options {
		option(h)
	}
	return h
}

func (h *HTTPLoader) Load(r *http.Request, image string) ([]byte, error) {
	if r.Method != http.MethodGet || image == "" {
		return nil, imagor.ErrPass
	}
	u, err := url.Parse(image)
	if err != nil {
		return nil, imagor.ErrPass
	}
	if u.Host == "" || u.Scheme == "" {
		return nil, imagor.ErrPass
	}
	if !isSourceAllowed(u, h.AllowedSources) {
		return nil, imagor.ErrPass
	}
	client := &http.Client{Transport: h.Transport}
	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, image, nil)
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
	if resp.StatusCode >= 400 {
		return buf, imagor.NewError(http.StatusText(resp.StatusCode), resp.StatusCode)
	}
	return buf, nil
}

func isSourceAllowed(u *url.URL, sources []string) bool {
	if len(sources) == 0 {
		return true
	}
	for _, source := range sources {
		if matched, e := path.Match(source, u.Host); matched && e == nil {
			return true
		}
	}
	return false
}
