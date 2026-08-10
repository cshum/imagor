package httploader

import (
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"path"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/cshum/imagor"
)

// AllowedSource represents a source the HTTPLoader is allowed to load from.
// It supports host glob patterns such as *.google.com and a full URL regex.
type AllowedSource struct {
	HostPattern string
	URLRegex    *regexp.Regexp
}

// NewRegexpAllowedSource creates a new AllowedSource from the regex pattern
func NewRegexpAllowedSource(pattern string) (AllowedSource, error) {
	regex, err := regexp.Compile(pattern)
	if err != nil {
		return AllowedSource{}, err
	}
	return AllowedSource{
		URLRegex: regex,
	}, nil
}

// NewHostPatternAllowedSource creates a new AllowedSource from the host glob pattern
func NewHostPatternAllowedSource(pattern string) AllowedSource {
	return AllowedSource{
		HostPattern: pattern,
	}
}

// Match checks if the url matches the AllowedSource
func (s AllowedSource) Match(u *url.URL) bool {
	if s.URLRegex != nil {
		return s.URLRegex.MatchString(u.String())
	}
	matched, e := path.Match(s.HostPattern, u.Host)
	return matched && e == nil
}

// HTTPLoader HTTP Loader implements imagor.Loader interface
type HTTPLoader struct {
	// The Transport used to request images, default http.DefaultTransport.
	Transport http.RoundTripper

	// ForwardHeaders copy request headers to image request headers
	ForwardHeaders []string

	// OverrideHeaders override image request headers
	OverrideHeaders map[string]string

	// OverrideResponseHeaders override image response header from HTTP Loader response
	OverrideResponseHeaders []string

	// AllowedSources list of sources allowed to load from
	AllowedSources []AllowedSource

	// Accept set request Accept and validate response Content-Type header
	Accept string

	// MaxAllowedSize maximum bytes allowed for image
	MaxAllowedSize int

	// DefaultScheme default image URL scheme
	DefaultScheme string

	// UserAgent default user agent for image request.
	// Can be overridden by ForwardHeaders and OverrideHeaders
	UserAgent string

	// BlockLoopbackNetworks rejects HTTP connections to loopback network IP addresses.
	BlockLoopbackNetworks bool

	// BlockPrivateNetworks rejects HTTP connections to private network IP addresses.
	BlockPrivateNetworks bool

	// BlockLinkLocalNetworks rejects HTTP connections to link local IP addresses.
	BlockLinkLocalNetworks bool

	// BlockNetworks rejects HTTP connections to a configurable list of networks.
	BlockNetworks []*net.IPNet

	// BaseURL base URL for HTTP loader
	BaseURL *url.URL

	accepts []string
}

// New creates HTTPLoader
func New(options ...Option) *HTTPLoader {
	h := &HTTPLoader{
		OverrideHeaders: map[string]string{},
		DefaultScheme:   "https",
		Accept:          "*/*",
		UserAgent:       fmt.Sprintf("imagor/%s", imagor.Version),
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	dialer := &net.Dialer{Control: h.DialControl}
	transport.DialContext = dialer.DialContext
	h.Transport = transport

	for _, option := range options {
		option(h)
	}
	if s := strings.ToLower(h.DefaultScheme); s == "nil" {
		h.DefaultScheme = ""
	}
	if h.Accept != "" {
		for _, seg := range strings.Split(h.Accept, ",") {
			if typ := parseContentType(seg); typ != "" {
				h.accepts = append(h.accepts, typ)
			}
		}
	}
	return h
}

// parseAndValidateURL validates and normalizes the image URL
// Returns the final URL string or an error
func (h *HTTPLoader) parseAndValidateURL(image string) (string, error) {
	if image == "" {
		return "", imagor.ErrInvalid
	}
	u, err := url.Parse(image)
	if err != nil {
		return "", imagor.ErrInvalid
	}
	if h.BaseURL != nil {
		newU := h.BaseURL.JoinPath(u.Path)
		newU.RawQuery = u.RawQuery
		image = newU.String()
		u = newU
	}
	if u.Host == "" || u.Scheme == "" {
		// If the image string contains percent-encoded characters, treat it as a
		// literal filename rather than a URL missing its scheme. This prevents
		// filenames like "https%3A%2F%2Fexample.com.avif" (often from b64-decoded
		// content) from being incorrectly interpreted as URLs by prepending the
		// default scheme.
		if containsPercentEncoding(image) {
			return "", imagor.ErrInvalid
		}
		if h.DefaultScheme != "" {
			image = h.DefaultScheme + "://" + image
			if u, err = url.Parse(image); err != nil {
				return "", imagor.ErrInvalid
			}
		} else {
			return "", imagor.ErrInvalid
		}
	}

	// Basic cleanup of the URL by dropping the fragment and cleaning up the
	// path which is important for matching against allowed sources.
	u = u.JoinPath()
	u.Fragment = ""

	if !isURLAllowed(u, h.AllowedSources) {
		return "", imagor.ErrSourceNotAllowed
	}

	return image, nil
}

// Get implements imagor.Loader interface
func (h *HTTPLoader) Get(r *http.Request, image string) (*imagor.Blob, error) {
	image, err := h.parseAndValidateURL(image)
	if err != nil {
		return nil, err
	}

	client := &http.Client{
		Transport:     h.Transport,
		CheckRedirect: h.checkRedirect,
	}
	if h.MaxAllowedSize > 0 {
		req, err := h.newRequest(r, http.MethodHead, image)
		if err != nil {
			return nil, err
		}
		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}
		_ = resp.Body.Close()
		if resp.StatusCode < 200 && resp.StatusCode > 206 {
			return nil, imagor.NewErrorFromStatusCode(resp.StatusCode)
		}
		contentLength, _ := strconv.Atoi(resp.Header.Get("Content-Length"))
		if contentLength > h.MaxAllowedSize {
			return nil, imagor.ErrMaxSizeExceeded
		}
	}
	req, err := h.newRequest(r, http.MethodGet, image)
	if err != nil {
		return nil, err
	}
	var blob *imagor.Blob
	var once sync.Once
	blob = imagor.NewBlob(func() (io.ReadCloser, int64, error) {
		resp, err := client.Do(req)
		if err != nil {
			if errors.Is(err, ErrUnauthorizedRequest) {
				err = imagor.NewError(
					fmt.Sprintf("%s: %s", err.Error(), image),
					http.StatusForbidden)
			} else if idx := strings.Index(err.Error(), "dial tcp: "); idx > -1 {
				err = imagor.NewError(
					fmt.Sprintf("%s: %s", err.Error()[idx:], image),
					http.StatusNotFound)
			}
			return nil, 0, err
		}
		once.Do(func() {
			blob.SetContentType(resp.Header.Get("Content-Type"))
			if len(h.OverrideResponseHeaders) > 0 {
				blob.Header = make(http.Header)
				for _, key := range h.OverrideResponseHeaders {
					if val := resp.Header.Get(key); val != "" {
						blob.Header.Set(key, val)
					}
				}
			}
		})
		body := resp.Body
		size, _ := strconv.ParseInt(resp.Header.Get("Content-Length"), 10, 64)
		if resp.Header.Get("Content-Encoding") == "gzip" {
			gzipBody, err := gzip.NewReader(resp.Body)
			if err != nil {
				return nil, 0, err
			}
			body = gzipBody
			size = 0 // size unknown after decompress
		}
		if resp.StatusCode >= 400 {
			return body, size, imagor.NewErrorFromStatusCode(resp.StatusCode)
		}
		if !validateContentType(resp.Header.Get("Content-Type"), h.accepts) {
			return body, size, imagor.ErrUnsupportedFormat
		}
		return body, size, nil
	})
	return blob, nil
}

func (h *HTTPLoader) newRequest(r *http.Request, method, url string) (*http.Request, error) {
	req, err := http.NewRequestWithContext(r.Context(), method, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", h.UserAgent)
	if h.Accept != "" {
		req.Header.Set("Accept", h.Accept)
	}
	for _, header := range h.ForwardHeaders {
		if header == "*" {
			req.Header = r.Header.Clone()
			req.Header.Del("Accept-Encoding") // fix compressions
			break
		}
		if _, ok := r.Header[header]; ok {
			req.Header.Set(header, r.Header.Get(header))
		}
	}
	for key, value := range h.OverrideHeaders {
		req.Header.Set(key, value)
	}
	return req, nil
}

func (h *HTTPLoader) checkRedirect(r *http.Request, via []*http.Request) error {
	if len(via) >= 10 {
		return errors.New("stopped after 10 redirects")
	}
	if !isURLAllowed(r.URL, h.AllowedSources) {
		return imagor.ErrSourceNotAllowed
	}
	return nil
}

// ErrUnauthorizedRequest unauthorized request error
var ErrUnauthorizedRequest = errors.New("unauthorized request")

func dialTargetIPs(addr net.IP) []net.IP {
	if addr == nil {
		return nil
	}

	targets := []net.IP{addr}
	v4InV6 := extractEmbeddedIPv4(addr)
	if v4InV6 != nil && !v4InV6.Equal(addr) {
		targets = append(targets, v4InV6)
	}

	return targets
}

func extractEmbeddedIPv4(addr net.IP) net.IP {
	v6 := addr.To16()
	if v6 == nil || addr.To4() != nil {
		return nil
	}

	if v6[0] == 0x00 && v6[1] == 0x64 && v6[2] == 0xff && v6[3] == 0x9b &&
		v6[4] == 0x00 && v6[5] == 0x00 && v6[6] == 0x00 && v6[7] == 0x00 &&
		v6[8] == 0x00 && v6[9] == 0x00 && v6[10] == 0x00 && v6[11] == 0x00 {
		return net.IPv4(v6[12], v6[13], v6[14], v6[15])
	}

	if v6[0] == 0x20 && v6[1] == 0x02 {
		return net.IPv4(v6[2], v6[3], v6[4], v6[5])
	}

	if v6[0] == 0x20 && v6[1] == 0x01 && v6[2] == 0x00 && v6[3] == 0x00 {
		return net.IPv4(v6[12]^0xff, v6[13]^0xff, v6[14]^0xff, v6[15]^0xff)
	}

	return nil
}

// DialControl implements a net.Dialer.Control function which is automatically used with the default http.Transport.
// If the transport is replaced using the WithTransport option it is up to that
// transport if the control function is used or not.
func (h *HTTPLoader) DialControl(network string, address string, conn syscall.RawConn) error {
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return err
	}
	addr := net.ParseIP(host)
	for _, target := range dialTargetIPs(addr) {
		if h.BlockLoopbackNetworks && target.IsLoopback() {
			return ErrUnauthorizedRequest
		}
		if h.BlockLinkLocalNetworks && (target.IsLinkLocalUnicast() || target.IsLinkLocalMulticast()) {
			return ErrUnauthorizedRequest
		}
		if h.BlockPrivateNetworks && target.IsPrivate() {
			return ErrUnauthorizedRequest
		}
		for _, network := range h.BlockNetworks {
			if network.Contains(target) {
				return ErrUnauthorizedRequest
			}
		}
	}
	return nil
}

// Stat implements imagor.Stater interface for HTTP Loader
// Makes a HEAD request to retrieve Last-Modified, ETag, and Content-Length metadata
func (h *HTTPLoader) Stat(ctx context.Context, image string) (*imagor.Stat, error) {
	image, err := h.parseAndValidateURL(image)
	if err != nil {
		return nil, err
	}

	// Create HEAD request to get metadata
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, image, nil)
	if err != nil {
		return nil, err
	}

	// Apply headers
	req.Header.Set("User-Agent", h.UserAgent)
	if h.Accept != "" {
		req.Header.Set("Accept", h.Accept)
	}
	for key, value := range h.OverrideHeaders {
		req.Header.Set(key, value)
	}

	client := &http.Client{
		Transport:     h.Transport,
		CheckRedirect: h.checkRedirect,
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, imagor.NewErrorFromStatusCode(resp.StatusCode)
	}

	stat := &imagor.Stat{}

	// Parse Last-Modified header
	if lastModified := resp.Header.Get("Last-Modified"); lastModified != "" {
		if t, err := time.Parse(http.TimeFormat, lastModified); err == nil {
			stat.ModifiedTime = t
		}
	}

	// Parse Content-Length for size
	if contentLength := resp.Header.Get("Content-Length"); contentLength != "" {
		if size, err := strconv.ParseInt(contentLength, 10, 64); err == nil {
			stat.Size = size
		}
	}

	// Use ETag if available
	if etag := resp.Header.Get("ETag"); etag != "" {
		stat.ETag = etag
	}

	// Return stat only if we have ModifiedTime
	// ModifiedTime is required for the modified-time-check comparison
	// ETag and Size are captured for potential future use but not sufficient on their own
	if !stat.ModifiedTime.IsZero() {
		return stat, nil
	}

	// If no ModifiedTime available, return not found
	return nil, imagor.ErrNotFound
}

// containsPercentEncoding checks if a string contains valid percent-encoded
// characters (e.g., %3A, %2F). This is used to detect strings that are likely
// literal filenames rather than URLs missing a scheme.
func containsPercentEncoding(s string) bool {
	for i := 0; i < len(s)-2; i++ {
		if s[i] == '%' && isHexDigit(s[i+1]) && isHexDigit(s[i+2]) {
			return true
		}
	}
	return false
}

func isHexDigit(c byte) bool {
	return (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')
}
