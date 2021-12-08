package params

import "strings"

type URL struct {
	BaseURL string
	Secret  string
	Unsafe  bool
}

func NewURL(baseURL, secret string) URL {
	return URL{
		BaseURL: strings.TrimSuffix(baseURL, "/") + "/",
		Secret:  secret,
	}
}

func NewURLUnsafe(baseURL string) URL {
	return URL{
		BaseURL: strings.TrimSuffix(baseURL, "/") + "/",
		Unsafe:  true,
	}
}

func (r URL) Generate(p Params) string {
	if r.Unsafe {
		return r.BaseURL + GenerateUnsafe(p)
	} else {
		return r.BaseURL + Generate(p, r.Secret)
	}
}

func (r URL) Parse(url string) (p Params, ok bool) {
	if uri := strings.TrimPrefix(url, r.BaseURL); uri != url {
		return Parse(strings.Split(uri, "?")[0]), true
	}
	return
}
