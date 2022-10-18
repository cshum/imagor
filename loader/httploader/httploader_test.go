package httploader

import (
	"bytes"
	"compress/gzip"
	"github.com/cshum/imagor"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"math/rand"
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
			Body:       io.NopCloser(strings.NewReader(res)),
			Header:     map[string][]string{},
		}
		w.Header.Set("Content-Type", "image/jpeg")
		return
	}
	if !strings.Contains(r.URL.Host, ".") {
		return
	}
	w = &http.Response{
		StatusCode: http.StatusNotFound,
		Body:       io.NopCloser(strings.NewReader("not found")),
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
			var err, err2 error
			var buf []byte
			b, err := loader.Get(r, tt.target)
			if tt.err == "" {
				require.NoError(t, err)
			}
			if tt.result != "" {
				buf, err2 = b.ReadAll()
				if tt.err == "" {
					require.NoError(t, err2)
				}
				assert.Equal(t, tt.result, string(buf))
			}
			if tt.err != "" {
				var msg string
				if err != nil {
					msg = err.Error()
				} else if err2 != nil {
					msg = err2.Error()
				}
				assert.Equal(t, tt.err, msg)
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
			err:    "imagor: 400 invalid",
		},
		{
			name:   "not allowed source",
			target: "https://foo.barr/baz",
			err:    "imagor: 400 invalid",
		},
		{
			name:   "not allowed source",
			target: "https://boo.bar/baz",
			err:    "imagor: 400 invalid",
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
			name:   "empty",
			target: "",
			err:    "imagor: 400 invalid",
		},
		{
			name:   "invalid url",
			target: "abc*abc",
			err:    "imagor: 400 invalid",
		},
		{
			name:   "default scheme set nil not found",
			target: "foo.bar/baz",
			err:    "imagor: 400 invalid",
		},
		{
			name:   "default scheme set nil not found",
			target: "foo.boo/boo",
			err:    "imagor: 400 invalid",
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
			res := &http.Response{
				StatusCode: http.StatusOK,
				Header:     map[string][]string{},
				Body:       io.NopCloser(strings.NewReader("ok")),
			}
			res.Header.Set("Content-Type", "image/jpeg")
			return res, nil
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
			res := &http.Response{
				StatusCode: http.StatusOK,
				Header:     map[string][]string{},
				Body:       io.NopCloser(strings.NewReader("ok")),
			}
			res.Header.Set("Content-Type", "image/jpeg")
			return res, nil
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
			res := &http.Response{
				StatusCode: http.StatusOK,
				Header:     map[string][]string{},
				Body:       io.NopCloser(strings.NewReader("ok")),
			}
			res.Header.Set("Content-Type", "image/jpeg")
			return res, nil
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

func TestWithForwardClientHeaders(t *testing.T) {
	doTests(t, New(
		WithTransport(roundTripFunc(func(r *http.Request) (w *http.Response, err error) {
			assert.Equal(t, r.Header.Get("User-Agent"), "Test")
			assert.Equal(t, r.Header.Get("X-Imagor-Foo"), "Bar")
			assert.Equal(t, r.Header.Get("X-Imagor-Ping"), "Pong")
			res := &http.Response{
				StatusCode: http.StatusOK,
				Header:     map[string][]string{},
				Body:       io.NopCloser(strings.NewReader("ok")),
			}
			res.Header.Set("Content-Type", "image/jpeg")
			return res, nil
		})),
		WithUserAgent("foobar"),
		WithForwardClientHeaders(true),
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
			res := &http.Response{
				StatusCode: http.StatusOK,
				Header:     map[string][]string{},
				Body:       io.NopCloser(strings.NewReader("ok")),
			}
			res.Header.Set("Content-Type", "image/jpeg")
			return res, nil
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
			res := &http.Response{
				StatusCode: http.StatusOK,
				Header:     map[string][]string{},
				Body:       io.NopCloser(strings.NewReader("ok")),
			}
			res.Header.Set("Content-Type", "image/jpeg")
			return res, nil
		})),
		WithUserAgent("foobar"),
		WithForwardClientHeaders(true),
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

func TestWithMaxAllowedSize(t *testing.T) {
	test1024Bytes := make([]byte, 1024)
	rand.Read(test1024Bytes)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "1024")
		w.Header().Set("Content-Type", "image/jpeg")
		_, _ = w.Write(test1024Bytes)
	}))
	defer ts.Close()

	doTests(t, New(
		WithMaxAllowedSize(1025),
		WithInsecureSkipVerifyTransport(true),
	), []test{
		{
			name:   "max allowed size ok",
			target: ts.URL,
			result: string(test1024Bytes),
		},
	})

	doTests(t, New(
		WithMaxAllowedSize(1023),
		WithInsecureSkipVerifyTransport(true),
	), []test{
		{
			name:   "max allowed size exceeded",
			target: ts.URL,
			err:    "imagor: 400 maximum size exceeded",
		},
	})
}

func TestWithNoProxy(t *testing.T) {
	h := New()
	r := httptest.NewRequest(http.MethodGet, "https://example.com/imagor", nil)
	pu, err := h.Transport.(*http.Transport).Proxy(r)
	assert.Nil(t, pu)
	assert.NoError(t, err)
}

func TestWithProxy(t *testing.T) {
	h := New(WithProxyTransport("https://user:pass@proxy.com:1667", ""))

	r := httptest.NewRequest(http.MethodGet, "https://example.com/imagor", nil)
	pu, err := h.Transport.(*http.Transport).Proxy(r)
	require.NotNil(t, pu)
	assert.Equal(t, "https://user:pass@proxy.com:1667", pu.String())
	assert.NoError(t, err)
}

func TestWithProxyAllowedSources(t *testing.T) {
	proxyURL := "https://user:pass@proxy.com:1667"
	h := New(WithProxyTransport(proxyURL, "*.foo.com,example.com"))
	tests := []struct {
		target  string
		isProxy bool
	}{
		{
			target:  "https://example.com/imagor",
			isProxy: true,
		},
		{
			target:  "https://fff.example.com/imagor",
			isProxy: false,
		},
		{
			target:  "https://abc.foo.com/imagor",
			isProxy: true,
		},
		{
			target:  "https://foo.com/imagor",
			isProxy: false,
		},
		{
			target:  "https://example2.com/imagor",
			isProxy: false,
		},
	}
	for _, tt := range tests {
		r := httptest.NewRequest(http.MethodGet, tt.target, nil)
		pu, err := h.Transport.(*http.Transport).Proxy(r)
		if tt.isProxy {
			require.NotNil(t, pu)
			assert.Equal(t, proxyURL, pu.String())
			assert.NoError(t, err)
		} else {
			assert.Nil(t, pu)
			assert.NoError(t, err)
		}
	}
}

func TestWithAccept(t *testing.T) {
	doTests(t, New(
		WithTransport(roundTripFunc(func(r *http.Request) (w *http.Response, err error) {
			assert.Equal(t, r.Header.Get("Accept"), "image/*")
			resp := &http.Response{
				StatusCode: http.StatusOK,
				Header:     map[string][]string{},
				Body:       io.NopCloser(strings.NewReader("ok")),
			}
			resp.Header.Set("Content-Type", strings.TrimPrefix(r.URL.Path, "/"))
			return resp, nil
		})),
		WithAccept("image/*"),
	), []test{
		{
			name:   "content type ok",
			target: "https://foo.bar/image/jpeg",
			result: "ok",
		},
		{
			name:   "content type ok",
			target: "https://foo.bar/image/png",
			result: "ok",
		},
		{
			name:   "content type not ok",
			target: "https://foo.bar/text/html",
			err:    imagor.ErrUnsupportedFormat.Error(),
			result: "ok",
		},
	})
}

func gzipBytes(a []byte) []byte {
	var b bytes.Buffer
	gz := gzip.NewWriter(&b)
	if _, err := gz.Write(a); err != nil {
		gz.Close()
		panic(err)
	}
	gz.Close()
	return b.Bytes()
}

func TestWithGzip(t *testing.T) {
	doTests(t, New(
		WithTransport(roundTripFunc(func(r *http.Request) (w *http.Response, err error) {
			resp := &http.Response{
				StatusCode: http.StatusOK,
				Header:     map[string][]string{},
				Body:       io.NopCloser(bytes.NewReader(gzipBytes([]byte("ok")))),
			}
			resp.Header.Set("Content-Encoding", "gzip")
			resp.Header.Set("Content-Type", "image/jpeg")
			return resp, nil
		})),
	), []test{
		{
			name:   "content type ok",
			target: "https://foo.bar/image/jpeg",
			result: "ok",
		},
	})
}

func TestWithInvalidHost(t *testing.T) {
	r, err := http.NewRequest(http.MethodGet, "http://example.com/unsafe/foo/bar", nil)
	assert.NoError(t, err)
	loader := New()
	blob, err := loader.Get(r, "foo/bar")
	assert.NoError(t, err)
	b, err := blob.ReadAll()
	assert.Empty(t, b)
	assert.Equal(t, "imagor: 404 no such host: https://foo/bar", err.Error())
}
