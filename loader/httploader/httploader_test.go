package httploader

import (
	"bytes"
	"compress/gzip"
	"io"
	"math/rand"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/cshum/imagor"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testTransport map[string]string

func (t testTransport) RoundTrip(r *http.Request) (w *http.Response, err error) {
	if res, ok := t[r.URL.String()]; ok {
		w = &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(res)),
			Header:     make(http.Header),
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
	header map[string]string
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
			if tt.header != nil {
				for key, val := range tt.header {
					assert.Equal(t, val, b.Header.Get(key))
				}
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
			err:    "imagor: 403 http source not allowed",
		},
		{
			name:   "not allowed source",
			target: "https://foo.barr/baz",
			err:    "imagor: 403 http source not allowed",
		},
		{
			name:   "not allowed source",
			target: "https://boo.bar/baz",
			err:    "imagor: 403 http source not allowed",
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

func TestWithAllowedSourceRegexp(t *testing.T) {
	doTests(t, New(
		WithTransport(testTransport{
			"https://goo.org/image1.png":   "goo_image1",
			"https://foo.com/dogs/dog.jpg": "dog",
		}),
		WithAllowedSourceRegexps(
			`^https://(foo|bar)\.com/dogs/.*\.jpg$`,
			`^https://goo\.org/.*`,
		),
	), []test{
		{
			name:   "allowed source",
			target: "https://goo.org/image1.png",
			result: "goo_image1",
		},
		{
			name:   "allowed source",
			target: "https://foo.com/dogs/dog.jpg",
			result: "dog",
		},
		{
			name:   "allowed not found",
			target: "https://goo.org/image2.png",
			result: "not found",
			err:    "imagor: 404 Not Found",
		},
		{
			name:   "not allowed source",
			target: "https://goo2.org/https://goo.org/image.png",
			err:    "imagor: 403 http source not allowed",
		},
		{
			name:   "not allowed source",
			target: "https://foo.com/dogs/../cats/cat.jpg",
			err:    "imagor: 403 http source not allowed",
		},
		{
			name:   "not allowed source",
			target: "https://foo.com/dogs/dog.jpg?size=small",
			err:    "imagor: 403 http source not allowed",
		},
	})
}

func TestWithAllowedSourcesRedirect(t *testing.T) {

	t.Run("Forbidden redirect", func(t *testing.T) {
		ts2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Error("should not redirect to here")
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("error"))
		}))
		defer ts2.Close()

		ts1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Log("redirecting")
			http.Redirect(w, r, ts2.URL, http.StatusTemporaryRedirect)
		}))
		defer ts1.Close()

		loader := New(
			WithAllowedSources(strings.TrimPrefix(ts1.URL, "http://")),
		)
		req, err := http.NewRequest(http.MethodGet, ts1.URL, nil)
		assert.NoError(t, err)
		blob, err := loader.Get(req, ts1.URL)

		b, err := blob.ReadAll()
		assert.Empty(t, b)
		assert.ErrorIs(t, err, imagor.ErrSourceNotAllowed)
	})

	t.Run("Allowed redirect", func(t *testing.T) {
		ts2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "image/jpeg")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("ok"))

		}))
		defer ts2.Close()

		ts1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Log("redirecting")
			http.Redirect(w, r, ts2.URL, http.StatusTemporaryRedirect)
		}))
		defer ts1.Close()

		loader := New()
		req, err := http.NewRequest(http.MethodGet, ts1.URL, nil)
		assert.NoError(t, err)
		blob, err := loader.Get(req, ts1.URL)
		assert.NoError(t, err)
		if !assert.NotNil(t, blob) {
			return
		}
		b, err := blob.ReadAll()
		assert.Equal(t, []byte("ok"), b)
	})

}

func TestBlockNetworks(t *testing.T) {

	t.Run("block loopback", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Error("should have been blocked")
			w.Header().Set("Content-Type", "image/jpeg")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("ok"))

		}))
		defer ts.Close()
		loader := New(
			WithBlockLoopbackNetworks(true),
		)
		req, err := http.NewRequest(http.MethodGet, ts.URL, nil)
		assert.NoError(t, err)
		blob, err := loader.Get(req, ts.URL)

		b, err := blob.ReadAll()
		assert.Empty(t, b)
		assert.ErrorContains(t, err, "unauthorized request")

	})

	t.Run("block network", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Error("should have been blocked")
			w.Header().Set("Content-Type", "image/jpeg")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("ok"))

		}))
		defer ts.Close()
		var networks []*net.IPNet

		for _, v := range []string{"::1/128", "127.0.0.0/8"} {
			_, network, err := net.ParseCIDR(v)
			assert.NoError(t, err)
			networks = append(networks, network)
		}
		loader := New(
			WithBlockNetworks(networks...),
		)
		req, err := http.NewRequest(http.MethodGet, ts.URL, nil)
		assert.NoError(t, err)
		blob, err := loader.Get(req, ts.URL)

		b, err := blob.ReadAll()
		assert.Empty(t, b)
		assert.ErrorContains(t, err, "unauthorized request")
	})

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("should have been blocked")
		w.Header().Set("Content-Type", "image/jpeg")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))

	}))
	defer ts.Close()

	for _, v := range []struct {
		name string
		addr string
		opt  Option
	}{
		{
			name: "link local",
			addr: "169.254.5.8:2000",
			opt:  WithBlockLinkLocalNetworks(true),
		},
		{
			name: "private",
			addr: "10.0.4.3:1000",
			opt:  WithBlockPrivateNetworks(true),
		},
	} {
		t.Run(v.name, func(t *testing.T) {
			loader := New(
				v.opt,
			)
			err := loader.DialControl("ipv4", v.addr, nil)
			assert.ErrorContains(t, err, "unauthorized request")
		})
	}
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

func TestWithBaseURL(t *testing.T) {
	trans := testTransport{
		"https://foo.com/bar.org/some/path/ping.jpg":         "pong",
		"https://foo.com/bar.org/some/path/ping.jpg?abc=123": "boom",
	}
	doTests(t, New(
		WithBaseURL("https://foo.com/bar.org"),
		WithTransport(trans),
	), []test{
		{
			name:   "base URL matched",
			target: "some/path/ping.jpg",
			result: "pong",
		},
		{
			name:   "base URL with query matched",
			target: "some/path/ping.jpg?abc=123",
			result: "boom",
		},
		{
			name:   "not found",
			target: "https://foo.com/bar.org/some/path/ping.jpg",
			result: "not found",
			err:    "imagor: 404 Not Found",
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

func TestWithOverrideResponseHeader(t *testing.T) {
	doTests(t, New(
		WithTransport(roundTripFunc(func(r *http.Request) (w *http.Response, err error) {
			res := &http.Response{
				StatusCode: http.StatusOK,
				Header:     map[string][]string{},
				Body:       io.NopCloser(strings.NewReader("ok")),
			}
			res.Header.Set("Content-Type", "image/jpeg")
			res.Header.Set("Foo", "Bar")
			return res, nil
		})),
		WithOverrideResponseHeaders("foo"),
	), []test{
		{
			name:   "user agent",
			target: "https://foo.bar/baz",
			result: "ok",
			header: map[string]string{
				"Foo": "Bar",
			},
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
	assert.Equal(t, 404, err.(imagor.Error).Code)
}

func TestPercentEncodedFilename(t *testing.T) {
	// Test that percent-encoded filenames are rejected by HTTPLoader
	// so that other loaders (like FileStorage) can handle them.
	// This addresses issue #627 where filenames like "https%3A%2F%2Fexample.com.avif"
	// were incorrectly interpreted as URLs.
	loader := New()
	r := httptest.NewRequest(http.MethodGet, "https://example.com/imagor", nil)

	tests := []struct {
		name   string
		image  string
		errMsg string
	}{
		{
			name:   "percent-encoded filename with https scheme",
			image:  "https%3A%2F%2Fexample.com.avif",
			errMsg: "imagor: 400 invalid",
		},
		{
			name:   "percent-encoded filename with http scheme",
			image:  "http%3A%2F%2Fexample.com%2Fimage.jpg",
			errMsg: "imagor: 400 invalid",
		},
		{
			name:   "percent-encoded slash in filename",
			image:  "some%2Fpath%2Ffile.jpg",
			errMsg: "imagor: 400 invalid",
		},
		{
			name:   "double percent-encoded filename",
			image:  "https%253A%252F%252Fexample.com.avif",
			errMsg: "imagor: 400 invalid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := loader.Get(r, tt.image)
			require.Error(t, err)
			assert.Equal(t, tt.errMsg, err.Error())
		})
	}
}

func TestContainsPercentEncoding(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"https%3A%2F%2Fexample.com", true},
		{"file%20name.jpg", true},
		{"%20", true},
		{"%2F", true},
		{"normal-filename.jpg", false},
		{"https://example.com", false},
		{"", false},
		{"%", false},    // incomplete encoding
		{"%2", false},   // incomplete encoding
		{"%GG", false},  // invalid hex
		{"%2g", false},  // invalid hex (lowercase g)
		{"%G2", false},  // invalid hex (uppercase G)
		{"a%2fb", true}, // lowercase hex is valid
		{"a%2Fb", true}, // mixed case hex is valid
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := containsPercentEncoding(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestHTTPLoader_Stat(t *testing.T) {
	// Test server that returns proper headers
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodHead, r.Method)

		switch r.URL.Path {
		case "/with-last-modified":
			w.Header().Set("Last-Modified", "Mon, 02 Jan 2006 15:04:05 GMT")
			w.Header().Set("Content-Length", "12345")
			w.Header().Set("ETag", `"abc123"`)
			w.WriteHeader(http.StatusOK)
		case "/with-etag-only":
			w.Header().Set("ETag", `"xyz789"`)
			w.Header().Set("Content-Length", "54321")
			w.WriteHeader(http.StatusOK)
		case "/no-metadata":
			w.Header().Set("Content-Length", "99999")
			w.WriteHeader(http.StatusOK)
		case "/not-found":
			w.WriteHeader(http.StatusNotFound)
		default:
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer ts.Close()

	loader := New(WithDefaultScheme("http"))
	r := httptest.NewRequest(http.MethodGet, "https://example.com/imagor", nil)
	ctx := r.Context()

	t.Run("with last modified and etag", func(t *testing.T) {
		stat, err := loader.Stat(ctx, ts.URL+"/with-last-modified")
		require.NoError(t, err)
		require.NotNil(t, stat)
		assert.False(t, stat.ModifiedTime.IsZero())
		assert.Equal(t, "Mon, 02 Jan 2006 15:04:05 GMT", stat.ModifiedTime.Format(http.TimeFormat))
		assert.Equal(t, `"abc123"`, stat.ETag)
		assert.Equal(t, int64(12345), stat.Size)
	})

	t.Run("with etag only returns not found", func(t *testing.T) {
		// ETag without ModifiedTime is not sufficient for modified-time-check
		stat, err := loader.Stat(ctx, ts.URL+"/with-etag-only")
		assert.Equal(t, imagor.ErrNotFound, err)
		assert.Nil(t, stat)
	})

	t.Run("no metadata returns not found", func(t *testing.T) {
		stat, err := loader.Stat(ctx, ts.URL+"/no-metadata")
		assert.Equal(t, imagor.ErrNotFound, err)
		assert.Nil(t, stat)
	})

	t.Run("not found error", func(t *testing.T) {
		stat, err := loader.Stat(ctx, ts.URL+"/not-found")
		assert.Error(t, err)
		assert.Equal(t, 404, err.(imagor.Error).Code)
		assert.Nil(t, stat)
	})

	t.Run("invalid image path", func(t *testing.T) {
		stat, err := loader.Stat(ctx, "")
		assert.Equal(t, imagor.ErrInvalid, err)
		assert.Nil(t, stat)
	})
}

func TestHTTPLoader_StatWithBaseURL(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodHead, r.Method)
		assert.Equal(t, "/images/test.jpg", r.URL.Path)
		w.Header().Set("Last-Modified", "Tue, 15 Nov 1994 12:45:26 GMT")
		w.Header().Set("ETag", `"base-url-test"`)
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	baseURLReq, _ := http.NewRequest(http.MethodGet, ts.URL+"/images/", nil)
	loader := New(WithBaseURL(baseURLReq.URL.String()))
	r := httptest.NewRequest(http.MethodGet, "https://example.com/imagor", nil)
	ctx := r.Context()

	stat, err := loader.Stat(ctx, "test.jpg")
	require.NoError(t, err)
	require.NotNil(t, stat)
	assert.Equal(t, `"base-url-test"`, stat.ETag)
}

func TestHTTPLoader_StatWithAllowedSources(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Last-Modified", "Wed, 21 Oct 2015 07:28:00 GMT")
		w.Header().Set("ETag", `"allowed"`)
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	loader := New(
		WithAllowedSources("allowed.com"),
		WithDefaultScheme("http"),
	)
	r := httptest.NewRequest(http.MethodGet, "https://example.com/imagor", nil)
	ctx := r.Context()

	t.Run("disallowed source", func(t *testing.T) {
		stat, err := loader.Stat(ctx, "notallowed.com/image.jpg")
		assert.Equal(t, imagor.ErrSourceNotAllowed, err)
		assert.Nil(t, stat)
	})
}
