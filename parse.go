package imagor

import (
	"net/http"
	"net/url"
	"strings"
)

func ParseRequest(r *http.Request) (params []string, key string, ok bool) {
	var from int
	params = strings.Split(r.URL.Path, "/")
	for to, seg := range params {
		if seg == "" {
			from++
		}
		if strings.HasPrefix(seg, "http") {
			key, _ = url.QueryUnescape(strings.Join(params[to:], "/"))
			params = params[from:to]
			ok = true
			return
		}
	}
	return
}
