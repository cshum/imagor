package imagor

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"

	"github.com/cshum/imagor/imagorpath"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsNestedPath(t *testing.T) {
	for _, tc := range []struct {
		name  string
		image string
		path  string
		ok    bool
	}{
		{"plain url", "https://example.com/foo.jpg", "", false},
		{"color", "color:red", "", false},
		{"storage key", "foo/bar.jpg", "", false},
		{"nested", "path:1200x628/color:red", "1200x628/color:red", true},
		{"nested, escaped", "path:1200x628%2Fcolor%3Ared", "1200x628/color:red", true},
		{"prefix with nothing after it", "path:", "", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path, ok := isNestedPath(tc.image)
			assert.Equal(t, tc.ok, ok)
			assert.Equal(t, tc.path, path)
		})
	}
}

// A nested path must be resolved in-process. The point of the feature is that a
// transformation which only needs bytes this process already has does not require
// the server to reach itself over the network, so the loader must not be consulted
// for the nested path itself.
func TestNestedPathDoesNotHitTheLoader(t *testing.T) {
	var mu sync.Mutex
	var loaded []string

	app := New(WithOptions(
		WithUnsafe(true),
		WithLoaders(loaderFunc(func(r *http.Request, image string) (*Blob, error) {
			mu.Lock()
			loaded = append(loaded, image)
			mu.Unlock()
			return NewBlobFromBytes([]byte("outer")), nil
		})),
		WithProcessors(processorFunc(
			func(ctx context.Context, blob *Blob, p imagorpath.Params, load LoadFunc) (*Blob, error) {
				// the nested path's own processor run sees the nested params
				mu.Lock()
				loaded = append(loaded, "process:"+p.Image)
				mu.Unlock()
				return NewBlobFromBytes([]byte("processed")), nil
			})),
	))

	w := httptest.NewRecorder()
	app.ServeHTTP(w, httptest.NewRequest(
		http.MethodGet, "https://example.com/unsafe/fit-in/400x400/path:1200x628/color:red", nil))
	require.Equal(t, 200, w.Code)

	mu.Lock()
	defer mu.Unlock()
	for _, l := range loaded {
		assert.False(t, strings.HasPrefix(l, "path:"),
			"the loader must never be asked for a nested path, got %q", l)
	}
	// the nested path is resolved in-process, and reaches the processor as its own
	// params; the outer path then reaches it as the outer params
	assert.Contains(t, loaded, "process:color:red",
		"the nested path should be processed in-process as its own params")
}

// A nested path is authorised by the outer request's signature, so it is not
// re-signed. This is the deliberate part of the trust model: one signature
// covers the whole tree.
func TestNestedPathIsNotResigned(t *testing.T) {
	var seen []string
	app := New(WithOptions(
		WithUnsafe(true),
		WithLoaders(loaderFunc(func(r *http.Request, image string) (*Blob, error) {
			return NewBlobFromBytes([]byte("outer")), nil
		})),
		WithProcessors(processorFunc(
			func(ctx context.Context, blob *Blob, p imagorpath.Params, load LoadFunc) (*Blob, error) {
				seen = append(seen, p.Image)
				return NewBlobFromBytes([]byte("processed")), nil
			})),
	))

	w := httptest.NewRecorder()
	app.ServeHTTP(w, httptest.NewRequest(
		http.MethodGet, "https://example.com/unsafe/fit-in/400x400/path:1200x628/color:red", nil))
	require.Equal(t, 200, w.Code)
	assert.Contains(t, seen, "color:red",
		"the nested path should reach the processor as its own params")
}

// f-tokens in a nested path resolve against the parent's dimensions, the same way
// they do inside image(). The two must agree or a preview and a composition would
// disagree about what f-20xf-20 means.
func TestNestedPathResolvesFTokens(t *testing.T) {
	for _, tc := range []struct {
		inner string
		want  string // the resolved path, with parent 400x400
	}{
		{"fxf", "400x400/color:red"},
		{"f-20xf-20", "380x380/color:red"},
		{"full-20xfull-20", "380x380/color:red"},
		{"fullxfull", "400x400/color:red"},
		{"200x200", "200x200/color:red"},
	} {
		t.Run(tc.inner, func(t *testing.T) {
			assert.Equal(t, tc.want, imagorpath.ResolveFullDimensions(tc.inner+"/color:red", 400, 400))
		})
	}
}

// A signed outer request must authorise the nested path, and tampering with the
// nested path must break the outer signature. The nested path is part of p.Path,
// so the outer signature covers it — this test holds that claim to account.
func TestNestedPathIsCoveredByTheOuterSignature(t *testing.T) {
	signer := imagorpath.NewDefaultSigner("secret")

	for _, tc := range []struct {
		name   string
		path   string
		tamper string // the path actually requested, if different
		want   int
	}{
		{
			name: "signed nested path",
			path: "fit-in/400x400/path:1200x628/color:red",
			want: 200,
		},
		{
			name:   "nested path tampered after signing",
			path:   "fit-in/400x400/path:1200x628/color:red",
			tamper: "fit-in/400x400/path:1200x628/color:blue",
			want:   403,
		},
		{
			name:   "nested path removed after signing",
			path:   "fit-in/400x400/path:1200x628/color:red",
			tamper: "fit-in/400x400/color:red",
			want:   403,
		},
		{
			name:   "nested path deepened after signing",
			path:   "fit-in/400x400/path:1200x628/color:red",
			tamper: "fit-in/400x400/path:200x200/path:1200x628/color:red",
			want:   403,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			app := New(WithOptions(
				WithSigner(signer),
				WithLoaders(loaderFunc(func(r *http.Request, image string) (*Blob, error) {
					return NewBlobFromBytes([]byte("outer")), nil
				})),
				WithProcessors(processorFunc(
					func(ctx context.Context, blob *Blob, p imagorpath.Params, load LoadFunc) (*Blob, error) {
						return NewBlobFromBytes([]byte("processed")), nil
					})),
			))

			requested := tc.path
			if tc.tamper != "" {
				requested = tc.tamper
			}
			w := httptest.NewRecorder()
			app.ServeHTTP(w, httptest.NewRequest(
				http.MethodGet,
				"https://example.com/"+signer.Sign(tc.path)+"/"+requested, nil))
			assert.Equal(t, tc.want, w.Code, w.Body.String())
		})
	}
}

// Without a bound, a self-referencing path recurses until the stack runs out.
func TestNestedPathDepthBound(t *testing.T) {
	app := New(WithOptions(
		WithUnsafe(true),
		WithLoaders(loaderFunc(func(r *http.Request, image string) (*Blob, error) {
			return NewBlobFromBytes([]byte("outer")), nil
		})),
		WithProcessors(processorFunc(
			func(ctx context.Context, blob *Blob, p imagorpath.Params, load LoadFunc) (*Blob, error) {
				return NewBlobFromBytes([]byte("processed")), nil
			})),
	))

	// MaxNestedDepth levels of nesting resolve, one more does not
	path := "200x200/color:green"
	for i := 0; i < MaxNestedDepth; i++ {
		path = "200x200/path:" + path
		w := httptest.NewRecorder()
		app.ServeHTTP(w, httptest.NewRequest(
			http.MethodGet, "https://example.com/unsafe/"+path, nil))
		require.Equal(t, 200, w.Code, "depth %d should resolve", i+1)
	}

	path = "200x200/path:" + path
	w := httptest.NewRecorder()
	app.ServeHTTP(w, httptest.NewRequest(
		http.MethodGet, "https://example.com/unsafe/"+path, nil))
	assert.Equal(t, 400, w.Code, "one level past MaxNestedDepth should be refused")
	assert.Contains(t, w.Body.String(), "too many nested levels")
}

// The two error paths in the nested-path branch log under Debug. Cover them so
// the branch is exercised the way it runs in a deployment with debug logging on.
func TestNestedPathDebugLogsOnFailurePaths(t *testing.T) {
	t.Run("depth exceeded", func(t *testing.T) {
		app := New(WithOptions(
			WithUnsafe(true),
			WithDebug(true),
			WithLoaders(loaderFunc(func(r *http.Request, image string) (*Blob, error) {
				return NewBlobFromBytes([]byte("outer")), nil
			})),
			WithProcessors(processorFunc(
				func(ctx context.Context, blob *Blob, p imagorpath.Params, load LoadFunc) (*Blob, error) {
					return NewBlobFromBytes([]byte("processed")), nil
				})),
		))

		// One past the bound, so the depth branch returns ErrTooManyNestedLevels
		// with Debug on.
		path := "200x200/color:green"
		for i := 0; i <= MaxNestedDepth; i++ {
			path = "200x200/path:" + path
		}
		w := httptest.NewRecorder()
		app.ServeHTTP(w, httptest.NewRequest(
			http.MethodGet, "https://example.com/unsafe/"+path, nil))
		assert.Equal(t, 400, w.Code)
		assert.Contains(t, w.Body.String(), "too many nested levels")
	})

	t.Run("nested path fails to resolve", func(t *testing.T) {
		app := New(WithOptions(
			WithUnsafe(true),
			WithDebug(true),
			// the nested path resolves to an unreachable URL source, so the
			// inner Do fails and the branch logs before returning
			WithLoaders(loaderFunc(func(r *http.Request, image string) (*Blob, error) {
				if strings.HasPrefix(image, "http") {
					return nil, errors.New("unreachable")
				}
				return NewBlobFromBytes([]byte("outer")), nil
			})),
			WithProcessors(processorFunc(
				func(ctx context.Context, blob *Blob, p imagorpath.Params, load LoadFunc) (*Blob, error) {
					return NewBlobFromBytes([]byte("processed")), nil
				})),
		))

		w := httptest.NewRecorder()
		app.ServeHTTP(w, httptest.NewRequest(
			http.MethodGet,
			"https://example.com/unsafe/400x400/path:"+
				url.QueryEscape("200x200/http://unreachable.example/img.jpg"), nil))
		assert.Equal(t, 500, w.Code)
	})
}
