package httploader

import (
	"math/rand"
	"net/http"
	"net/url"
	"path"
	"strings"
)

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

func isURLAllowed(u *url.URL, allowedSources []string) bool {
	if len(allowedSources) == 0 {
		return true
	}
	for _, source := range allowedSources {
		if matched, e := path.Match(source, u.Host); matched && e == nil {
			return true
		}
	}
	return false
}

func parseContentType(contentType string) string {
	idx := strings.Index(contentType, ";")
	if idx == -1 {
		idx = len(contentType)
	}
	return strings.TrimSpace(strings.ToLower(contentType[0:idx]))
}

func validateContentType(contentType string, accepts []string) bool {
	if len(accepts) == 0 {
		return true
	}
	contentType = parseContentType(contentType)
	for _, accept := range accepts {
		if ok, err := path.Match(accept, contentType); ok && err == nil {
			return true
		}
	}
	return false
}
