package httploader

import (
	"github.com/cshum/imagor"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type testTransport map[string]string

func (t testTransport) RoundTrip(r *http.Request) (w *http.Response, err error) {
	if res, ok := t[r.URL.String()]; ok {
		w = &http.Response{
			StatusCode: http.StatusOK,
			Body:       ioutil.NopCloser(strings.NewReader(res)),
		}
		return
	}
	w = &http.Response{
		StatusCode: http.StatusNotFound,
		Body:       ioutil.NopCloser(strings.NewReader("not found")),
	}
	return
}

type roundTripFunc func(r *http.Request) (w *http.Response, err error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

type test struct {
	name   string
	target string
	result string
	err    string
}

func doTests(t *testing.T, loader imagor.Loader, tests []test) {
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "https://example.com/imagor", nil)
			r.Header.Set("User-Agent", "Test")
			r.Header.Set("X-Imagor-Foo", "Bar")
			r.Header.Set("X-Imagor-Ping", "Pong")
			b, err := loader.Load(r, tt.target)
			assert.Equal(t, string(b), tt.result)
			if tt.err == "" {
				assert.NoError(t, err)
			} else {
				assert.EqualError(t, err, tt.err)
			}
		})
	}
}

func TestWithAllowedSources(t *testing.T) {
	doTests(t, New(
		WithTransport(testTransport{
			"https://foo.bar/baz":   "baz",
			"https://foo.boo/boooo": "boom",
			"https://def.def/boo":   "boo",
			"https://foo.abc/bar":   "foobar",
		}),
		WithAllowedSources("foo.bar", "*.abc", "def.def,ghi.ghi"),
	), []test{
		{
			name:   "allowed source",
			target: "https://foo.bar/baz",
			result: "baz",
		},
		{
			name:   "allowed not found",
			target: "https://foo.bar/boooo",
			result: "not found",
			err:    "imagor: 404 Not Found",
		},
		{
			name:   "not allowed source",
			target: "https://foo.boo/boooo",
			err:    "imagor: 400 pass",
		},
		{
			name:   "not allowed source",
			target: "https://foo.barr/baz",
			err:    "imagor: 400 pass",
		},
		{
			name:   "not allowed source",
			target: "https://boo.bar/baz",
			err:    "imagor: 400 pass",
		},
		{
			name:   "csv allowed source",
			target: "https://def.def/boo",
			result: "boo",
		},
		{
			name:   "glob allowed source",
			target: "https://foo.abc/bar",
			result: "foobar",
		},
	})
}

func TestWithDefaultScheme(t *testing.T) {
	trans := testTransport{
		"https://foo.bar/baz": "baz",
		"http://foo.boo/boo":  "boom",
	}
	doTests(t, New(
		WithTransport(trans),
	), []test{
		{
			name:   "default scheme found",
			target: "foo.bar/baz",
			result: "baz",
		},
		{
			name:   "default scheme not found http",
			target: "foo.boo/boo",
			result: "not found",
			err:    "imagor: 404 Not Found",
		},
	})
	doTests(t, New(
		WithTransport(trans),
		WithDefaultScheme("http"),
	), []test{
		{
			name:   "default scheme set http not found",
			target: "foo.bar/baz",
			result: "not found",
			err:    "imagor: 404 Not Found",
		},
		{
			name:   "default scheme set http found",
			target: "foo.boo/boo",
			result: "boom",
		},
	})
	doTests(t, New(
		WithTransport(trans),
		WithDefaultScheme("nil"),
	), []test{
		{
			name:   "default scheme set nil not found",
			target: "foo.bar/baz",
			err:    "imagor: 400 pass",
		},
		{
			name:   "default scheme set nil not found",
			target: "foo.boo/boo",
			err:    "imagor: 400 pass",
		},
		{
			name:   "default scheme set nil found",
			target: "https://foo.bar/baz",
			result: "baz",
		},
	})
}

func TestWithUserAgent(t *testing.T) {
	doTests(t, New(
		WithTransport(roundTripFunc(func(r *http.Request) (w *http.Response, err error) {
			assert.Equal(t, r.Header.Get("User-Agent"), "foobar")
			assert.Equal(t, r.Header.Get("X-Imagor-Foo"), "")
			assert.Equal(t, r.Header.Get("X-Imagor-Ping"), "")
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       ioutil.NopCloser(strings.NewReader("ok")),
			}, nil
		})),
		WithUserAgent("foobar"),
	), []test{
		{
			name:   "user agent",
			target: "https://foo.bar/baz",
			result: "ok",
		},
	})
}

func TestWithForwardHeaders(t *testing.T) {
	doTests(t, New(
		WithTransport(roundTripFunc(func(r *http.Request) (w *http.Response, err error) {
			assert.Equal(t, r.Header.Get("User-Agent"), "foobar")
			assert.Equal(t, r.Header.Get("X-Imagor-Foo"), "Bar")
			assert.Equal(t, r.Header.Get("X-Imagor-Ping"), "")
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       ioutil.NopCloser(strings.NewReader("ok")),
			}, nil
		})),
		WithUserAgent("foobar"),
		WithForwardHeaders("X-Imagor-Foo"),
	), []test{
		{
			name:   "user agent",
			target: "https://foo.bar/baz",
			result: "ok",
		},
	})
}

func TestWithForwardHeadersOverrideUserAgent(t *testing.T) {
	doTests(t, New(
		WithTransport(roundTripFunc(func(r *http.Request) (w *http.Response, err error) {
			assert.Equal(t, r.Header.Get("User-Agent"), "Test")
			assert.Equal(t, r.Header.Get("X-Imagor-Foo"), "")
			assert.Equal(t, r.Header.Get("X-Imagor-Ping"), "Pong")
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       ioutil.NopCloser(strings.NewReader("ok")),
			}, nil
		})),
		WithUserAgent("foobar"),
		WithForwardHeaders("User-Agent, X-Imagor-Ping"),
	), []test{
		{
			name:   "user agent",
			target: "https://foo.bar/baz",
			result: "ok",
		},
	})
}

func TestWithForwardAllHeaders(t *testing.T) {
	doTests(t, New(
		WithTransport(roundTripFunc(func(r *http.Request) (w *http.Response, err error) {
			assert.Equal(t, r.Header.Get("User-Agent"), "Test")
			assert.Equal(t, r.Header.Get("X-Imagor-Foo"), "Bar")
			assert.Equal(t, r.Header.Get("X-Imagor-Ping"), "Pong")
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       ioutil.NopCloser(strings.NewReader("ok")),
			}, nil
		})),
		WithUserAgent("foobar"),
		WithForwardAllHeaders(true),
	), []test{
		{
			name:   "user agent",
			target: "https://foo.bar/baz",
			result: "ok",
		},
	})
}

func TestWithOverrideHeaders(t *testing.T) {
	doTests(t, New(
		WithTransport(roundTripFunc(func(r *http.Request) (w *http.Response, err error) {
			assert.Equal(t, r.Header.Get("User-Agent"), "foobar")
			assert.Equal(t, r.Header.Get("X-Imagor-Foo"), "Boom")
			assert.Equal(t, r.Header.Get("X-Imagor-Ping"), "")
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       ioutil.NopCloser(strings.NewReader("ok")),
			}, nil
		})),
		WithUserAgent("foobar"),
		WithOverrideHeader("x-Imagor-Foo", "Boom"),
	), []test{
		{
			name:   "user agent",
			target: "https://foo.bar/baz",
			result: "ok",
		},
	})
}

func TestWithOverrideForwardHeaders(t *testing.T) {
	doTests(t, New(
		WithTransport(roundTripFunc(func(r *http.Request) (w *http.Response, err error) {
			assert.Equal(t, r.Header.Get("User-Agent"), "Ha")
			assert.Equal(t, r.Header.Get("X-Imagor-Foo"), "Boom")
			assert.Equal(t, r.Header.Get("X-Imagor-Ping"), "Pong")
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       ioutil.NopCloser(strings.NewReader("ok")),
			}, nil
		})),
		WithUserAgent("foobar"),
		WithForwardAllHeaders(true),
		WithOverrideHeader("x-Imagor-Foo", "Boom"),
		WithOverrideHeader("User-Agent", "Ha"),
	), []test{
		{
			name:   "user agent",
			target: "https://foo.bar/baz",
			result: "ok",
		},
	})
}
