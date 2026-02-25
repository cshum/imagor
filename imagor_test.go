package imagor

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/cshum/imagor/imagorpath"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestWithUnsafe(t *testing.T) {
	logger := zap.NewExample()
	app := New(WithOptions(
		WithUnsafe(true),
		WithLoaders(loaderFunc(func(r *http.Request, image string) (*Blob, error) {
			return NewBlobFromBytes([]byte("foo")), nil
		})),
		WithLogger(logger),
	))
	assert.Equal(t, false, app.Debug)
	assert.Equal(t, logger, app.Logger)

	w := httptest.NewRecorder()
	app.ServeHTTP(w, httptest.NewRequest(
		http.MethodGet, "https://example.com/unsafe/foo.jpg", nil))
	assert.Equal(t, 200, w.Code)

	w = httptest.NewRecorder()
	app.ServeHTTP(w, httptest.NewRequest(
		http.MethodPost, "https://example.com/unsafe/foo.jpg", nil))
	assert.Equal(t, 405, w.Code)
	assert.Equal(t, "", w.Body.String())

	w = httptest.NewRecorder()
	app.ServeHTTP(w, httptest.NewRequest(
		http.MethodGet, "https://example.com/foo.jpg", nil))
	assert.Equal(t, 403, w.Code)
	assert.Equal(t, w.Body.String(), jsonStr(ErrSignatureMismatch))
}

func TestWithEnablePostRequests(t *testing.T) {
	logger := zap.NewExample()

	t.Run("POST requests disabled by default", func(t *testing.T) {
		app := New(WithOptions(
			WithUnsafe(true),
			WithLoaders(loaderFunc(func(r *http.Request, image string) (*Blob, error) {
				return NewBlobFromBytes([]byte("foo")), nil
			})),
			WithLogger(logger),
		))
		assert.False(t, app.EnablePostRequests)

		// GET should work
		w := httptest.NewRecorder()
		app.ServeHTTP(w, httptest.NewRequest(
			http.MethodGet, "https://example.com/unsafe/foo.jpg", nil))
		assert.Equal(t, 200, w.Code)

		// POST should be rejected
		w = httptest.NewRecorder()
		app.ServeHTTP(w, httptest.NewRequest(
			http.MethodPost, "https://example.com/unsafe/foo.jpg", nil))
		assert.Equal(t, 405, w.Code)
	})

	t.Run("POST requests enabled", func(t *testing.T) {
		app := New(WithOptions(
			WithUnsafe(true),
			WithEnablePostRequests(true),
			WithLoaders(loaderFunc(func(r *http.Request, image string) (*Blob, error) {
				return NewBlobFromBytes([]byte("uploaded")), nil
			})),
			WithLogger(logger),
		))
		assert.True(t, app.EnablePostRequests)

		// GET should still work
		w := httptest.NewRecorder()
		app.ServeHTTP(w, httptest.NewRequest(
			http.MethodGet, "https://example.com/unsafe/foo.jpg", nil))
		assert.Equal(t, 200, w.Code)

		// POST should now be accepted
		w = httptest.NewRecorder()
		app.ServeHTTP(w, httptest.NewRequest(
			http.MethodPost, "https://example.com/unsafe/foo.jpg", nil))
		assert.Equal(t, 200, w.Code)
		assert.Equal(t, "uploaded", w.Body.String())
	})

	t.Run("POST requests require unsafe mode", func(t *testing.T) {
		app := New(WithOptions(
			WithEnablePostRequests(true), // POST enabled but unsafe disabled
			WithLoaders(loaderFunc(func(r *http.Request, image string) (*Blob, error) {
				return NewBlobFromBytes([]byte("foo")), nil
			})),
			WithLogger(logger),
		))
		assert.True(t, app.EnablePostRequests)
		assert.False(t, app.Unsafe)

		// POST should be rejected due to unsafe mode being disabled
		w := httptest.NewRecorder()
		app.ServeHTTP(w, httptest.NewRequest(
			http.MethodPost, "https://example.com/unsafe/foo.jpg", nil))
		assert.Equal(t, 405, w.Code)
	})

	t.Run("POST requests with both unsafe and POST enabled", func(t *testing.T) {
		app := New(WithOptions(
			WithUnsafe(true),
			WithEnablePostRequests(true),
			WithLoaders(loaderFunc(func(r *http.Request, image string) (*Blob, error) {
				return NewBlobFromBytes([]byte("both-enabled")), nil
			})),
			WithLogger(logger),
		))
		assert.True(t, app.EnablePostRequests)
		assert.True(t, app.Unsafe)

		// POST should work with both flags enabled
		w := httptest.NewRecorder()
		app.ServeHTTP(w, httptest.NewRequest(
			http.MethodPost, "https://example.com/unsafe/foo.jpg", nil))
		assert.Equal(t, 200, w.Code)
		assert.Equal(t, "both-enabled", w.Body.String())
	})
}

func TestWithInternal(t *testing.T) {
	logger := zap.NewExample()
	ctx := context.Background()
	app := New(WithOptions(
		WithLoaders(loaderFunc(func(r *http.Request, image string) (*Blob, error) {
			return NewBlobFromBytes([]byte("foo")), nil
		})),
		WithProcessors(processorFunc(func(ctx context.Context, blob *Blob, p imagorpath.Params, load LoadFunc) (*Blob, error) {
			assert.Contains(t, p.Path, p.Image)
			assert.Positive(t, p.Width)
			assert.Positive(t, p.Height)
			return blob, nil
		})),
		WithLogger(logger),
	))
	assert.Equal(t, false, app.Debug)
	assert.Equal(t, logger, app.Logger)

	blob, err := app.Serve(ctx, imagorpath.Params{
		Image: "foo.jpg",
		Width: 167, Height: 199,
		Path: "ghjk",
	})
	require.NoError(t, err)
	buf, err := blob.ReadAll()
	assert.Equal(t, "foo", string(buf))
	require.NoError(t, err)

	blob, err = app.Serve(ctx, imagorpath.Params{
		Image: "foo.jpg",
		Width: -167, Height: -199,
		Path: "ghjk",
	})
	require.NoError(t, err)
	buf, err = blob.ReadAll()
	assert.Equal(t, "foo", string(buf))
	require.NoError(t, err)

	blob, err = app.Serve(nil, imagorpath.Params{
		Image: "foo.jpg",
		Width: 167, Height: 199,
		Path: "ghjk",
	})
	assert.Empty(t, blob)
	assert.Error(t, err)

	blob, err = app.ServeBlob(nil, nil, imagorpath.Params{
		Image: "foo.jpg",
		Width: 167, Height: 199,
		Path: "ghjk",
	})
	assert.Empty(t, blob)
	assert.Error(t, err)

	blob, err = app.ServeBlob(ctx, NewBlobFromBytes([]byte("asdf")), imagorpath.Params{
		Image: "foo.jpg",
		Width: 167, Height: 199,
		Path: "ghjk",
	})
	require.NoError(t, err)
	buf, err = blob.ReadAll()
	assert.Equal(t, "asdf", string(buf))
	require.NoError(t, err)
}

func TestWithContentDisposition(t *testing.T) {
	logger := zap.NewExample()
	app := New(
		WithUnsafe(true),
		WithLoaders(loaderFunc(func(r *http.Request, image string) (*Blob, error) {
			return NewBlobFromFile(image), nil
		})),
		WithResultStorages(saverFunc(func(ctx context.Context, image string, blob *Blob) error {
			assert.NotContains(t, image, "attachment")
			return nil
		})),
		WithLogger(logger))
	assert.Equal(t, false, app.Debug)
	assert.Equal(t, logger, app.Logger)

	w := httptest.NewRecorder()
	app.ServeHTTP(w, httptest.NewRequest(
		http.MethodGet, "https://example.com/unsafe/testdata/gopher.png", nil))
	assert.Equal(t, 200, w.Code)
	assert.Equal(t, "inline", w.Header().Get("Content-Disposition"))

	w = httptest.NewRecorder()
	app.ServeHTTP(w, httptest.NewRequest(
		http.MethodGet, "https://example.com/unsafe/filters:attachment()/testdata/gopher.png", nil))
	assert.Equal(t, 200, w.Code)
	assert.Equal(t, `attachment; filename="gopher.png"`, w.Header().Get("Content-Disposition"))

	w = httptest.NewRecorder()
	app.ServeHTTP(w, httptest.NewRequest(
		http.MethodGet, "https://example.com/unsafe/filters:attachment(foo.png)/testdata/demo1.jpg", nil))
	assert.Equal(t, 200, w.Code)
	assert.Equal(t, `attachment; filename="foo.png.jpg"`, w.Header().Get("Content-Disposition"))

	w = httptest.NewRecorder()
	app.ServeHTTP(w, httptest.NewRequest(
		http.MethodGet, "https://example.com/unsafe/filters:attachment(foo.jpeg)/testdata/demo1.jpg", nil))
	assert.Equal(t, 200, w.Code)
	assert.Equal(t, `attachment; filename="foo.jpeg"`, w.Header().Get("Content-Disposition"))

}

func TestSuppressDeadlockResolve(t *testing.T) {
	ctx := context.Background()
	app := New()
	f, err := app.suppress(ctx, "a", func(ctx context.Context, _ func(*Blob, error)) (*Blob, error) {
		return app.suppress(ctx, "b", func(ctx context.Context, _ func(*Blob, error)) (*Blob, error) {
			return app.suppress(ctx, "a", func(ctx context.Context, _ func(*Blob, error)) (*Blob, error) {
				return NewEmptyBlob(), nil
			})
		})
	})
	assert.Equal(t, NewEmptyBlob(), f)
	require.NoError(t, err)
}

func TestSuppressTimeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*10)
	defer cancel()
	app := New()
	f, err := app.suppress(ctx, "a", func(ctx context.Context, _ func(*Blob, error)) (*Blob, error) {
		time.Sleep(time.Second)
		return &Blob{}, nil
	})
	assert.Nil(t, f)
	assert.Equal(t, context.DeadlineExceeded, err)
}

func TestSuppressForgetCanceled(t *testing.T) {
	n := 10
	app := New()
	var wg sync.WaitGroup
	wg.Add(n * 2)
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			_, err := app.suppress(context.Background(), "a", func(ctx context.Context, _ func(*Blob, error)) (*Blob, error) {
				time.Sleep(time.Millisecond)
				return NewEmptyBlob(), nil
			})
			assert.Nil(t, err)
		}()
		go func() {
			defer wg.Done()
			_, _ = app.suppress(context.Background(), "a", func(ctx context.Context, _ func(*Blob, error)) (*Blob, error) {
				time.Sleep(time.Millisecond)
				return nil, context.Canceled
			})
		}()
	}
	wg.Wait()
}

func TestWithSigner(t *testing.T) {
	app := New(
		WithDebug(true),
		WithLogger(zap.NewExample()),
		WithLoaders(loaderFunc(func(r *http.Request, image string) (*Blob, error) {
			return NewBlobFromBytes([]byte("foo")), nil
		})),
		WithSigner(imagorpath.NewDefaultSigner("1234")))
	assert.Equal(t, true, app.Debug)

	w := httptest.NewRecorder()
	app.ServeHTTP(w, httptest.NewRequest(
		http.MethodGet, "https://example.com/_-19cQt1szHeUV0WyWFntvTImDI=/foo.jpg", nil))
	assert.Equal(t, 200, w.Code)

	w = httptest.NewRecorder()
	app.ServeHTTP(w, httptest.NewRequest(
		http.MethodGet, "https://example.com/_-19cQt1szHeUV0WyWFntvTIm/foo.jpg", nil))
	assert.Equal(t, 403, w.Code)
	assert.Equal(t, w.Body.String(), jsonStr(ErrSignatureMismatch))

	w = httptest.NewRecorder()
	app.ServeHTTP(w, httptest.NewRequest(
		http.MethodGet, "https://example.com/foo.jpg", nil))
	assert.Equal(t, 403, w.Code)
	assert.Equal(t, w.Body.String(), jsonStr(ErrSignatureMismatch))
}

func TestWithCustomSigner(t *testing.T) {
	app := New(
		WithDebug(true),
		WithLogger(zap.NewExample()),
		WithLoaders(loaderFunc(func(r *http.Request, image string) (*Blob, error) {
			return NewBlobFromBytes([]byte("foo")), nil
		})),
		WithSigner(imagorpath.NewHMACSigner(sha256.New, 40, "1234")))
	assert.Equal(t, true, app.Debug)

	w := httptest.NewRecorder()
	app.ServeHTTP(w, httptest.NewRequest(
		http.MethodGet, "https://example.com/91DBDJtTFePFnbaj5Qq8JLvq5sM5VTipE685f4Gp/foo.jpg", nil))
	assert.Equal(t, 200, w.Code)

	w = httptest.NewRecorder()
	app.ServeHTTP(w, httptest.NewRequest(
		http.MethodGet, "https://example.com/_-19cQt1szHeUV0WyWFntvTImDI=/foo.jpg", nil))
	assert.Equal(t, 403, w.Code)
	assert.Equal(t, w.Body.String(), jsonStr(ErrSignatureMismatch))

	w = httptest.NewRecorder()
	app.ServeHTTP(w, httptest.NewRequest(
		http.MethodGet, "https://example.com/foo.jpg", nil))
	assert.Equal(t, 403, w.Code)
	assert.Equal(t, w.Body.String(), jsonStr(ErrSignatureMismatch))
}

func TestWithRetryQueryUnescape(t *testing.T) {
	opts := WithOptions(
		WithDebug(true),
		WithLogger(zap.NewExample()),
		WithLoaders(loaderFunc(func(r *http.Request, image string) (*Blob, error) {
			if image != "gopher.png" {
				return nil, ErrInvalid
			}
			return NewBlobFromBytes([]byte("foo")), nil
		})),
		WithProcessors(processorFunc(func(ctx context.Context, blob *Blob, p imagorpath.Params, load LoadFunc) (*Blob, error) {
			return NewBlobFromBytes([]byte(p.Path)), nil
		})),
	)

	app := New(opts, WithUnsafe(true))
	w := httptest.NewRecorder()
	app.ServeHTTP(w, httptest.NewRequest(
		http.MethodGet, "https://example.com/unsafe/filters%3Afill%28red%29/gopher.png", nil))
	assert.Equal(t, 200, w.Code)
	assert.Equal(t, "filters:fill(red)/gopher.png", w.Body.String())

	app = New(opts, WithSigner(imagorpath.NewDefaultSigner("1234")))
	w = httptest.NewRecorder()
	app.ServeHTTP(w, httptest.NewRequest(
		http.MethodGet, "https://example.com/GSqGN9NrCj-DlGT38BaycScxM9g=/filters%3Afill%28red%29/gopher.png", nil))
	assert.Equal(t, 200, w.Code)
	assert.Equal(t, "filters:fill(red)/gopher.png", w.Body.String())
}

func TestWithRaw(t *testing.T) {
	app := New(
		WithDebug(true),
		WithUnsafe(true),
		WithLogger(zap.NewExample()),
		WithLoaders(loaderFunc(func(r *http.Request, image string) (*Blob, error) {
			if image != "gopher.png" {
				return nil, ErrInvalid
			}
			blob := NewBlobFromBytes([]byte("foo"))
			blob.SetContentType("bar")
			return blob, nil
		})),
		WithProcessors(processorFunc(func(ctx context.Context, blob *Blob, p imagorpath.Params, load LoadFunc) (*Blob, error) {
			out := NewBlobFromBytes([]byte(p.Path))
			out.SetContentType("boom")
			return out, nil
		})),
	)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, httptest.NewRequest(
		http.MethodGet, "https://example.com/unsafe/filters:fill(red)/gopher.png", nil))
	assert.Equal(t, 200, w.Code)
	assert.Equal(t, "filters:fill(red)/gopher.png", w.Body.String())
	assert.Equal(t, "boom", w.Header().Get("Content-Type"))
	assert.Empty(t, w.Header().Get("Content-Security-Policy"))

	w = httptest.NewRecorder()
	app.ServeHTTP(w, httptest.NewRequest(
		http.MethodGet, "https://example.com/unsafe/filters:fill(red):raw()/gopher.png", nil))
	assert.Equal(t, 200, w.Code)
	assert.Equal(t, "foo", w.Body.String())
	assert.Equal(t, "script-src 'none'", w.Header().Get("Content-Security-Policy"))
	assert.Equal(t, "bar", w.Header().Get("Content-Type"))
}

func TestWithOverrideHeader(t *testing.T) {
	app := New(
		WithDebug(true),
		WithUnsafe(true),
		WithLogger(zap.NewExample()),
		WithLoaders(loaderFunc(func(r *http.Request, image string) (*Blob, error) {
			blob := NewBlobFromBytes([]byte("foo"))
			blob.SetContentType("bar")
			blob.Header = make(http.Header)
			blob.Header.Set("Content-Type", "tada")
			blob.Header.Set("Foo", "bar")
			blob.Header.Set("asdf", "fghj")
			return blob, nil
		})),
		WithProcessors(processorFunc(func(ctx context.Context, blob *Blob, p imagorpath.Params, load LoadFunc) (*Blob, error) {
			out := NewBlobFromBytes([]byte("processed"))
			out.SetContentType("boom")
			return out, nil
		})),
	)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, httptest.NewRequest(
		http.MethodGet, "https://example.com/unsafe/filters:fill(red)/gopher.png", nil))
	assert.Equal(t, 200, w.Code)
	assert.Equal(t, "processed", w.Body.String())
	assert.Equal(t, "tada", w.Header().Get("Content-Type"))
	assert.Equal(t, "fghj", w.Header().Get("ASDF"))
}

func TestNewBlobFromPathNotFound(t *testing.T) {
	loader := loaderFunc(func(r *http.Request, image string) (*Blob, error) {
		return NewBlobFromFile("./non-exists-path"), nil
	})
	app := New(
		WithDebug(true),
		WithLogger(zap.NewExample()),
		WithUnsafe(true),
		WithLoaders(loader))

	r := httptest.NewRequest(
		http.MethodGet, "https://example.com/unsafe/foobar", nil)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, r)
	assert.Equal(t, 404, w.Code)
	assert.Equal(t, jsonStr(ErrNotFound), w.Body.String())

	app = New(
		WithDebug(true),
		WithLogger(zap.NewExample()),
		WithUnsafe(true),
		WithDisableErrorBody(true),
		WithLoaders(loader))

	r = httptest.NewRequest(
		http.MethodGet, "https://example.com/unsafe/foobar", nil)
	w = httptest.NewRecorder()
	app.ServeHTTP(w, r)
	assert.Equal(t, 404, w.Code)
	assert.Empty(t, w.Body.String())
}

func TestWithDisableErrorBody(t *testing.T) {
	app := New(
		WithDebug(true),
		WithLogger(zap.NewExample()),
		WithDisableErrorBody(true),
		WithSigner(imagorpath.NewDefaultSigner("1234")))
	assert.True(t, app.DisableErrorBody)

	w := httptest.NewRecorder()
	app.ServeHTTP(w, httptest.NewRequest(
		http.MethodGet, "https://example.com/foo.jpg", nil))
	assert.Equal(t, 403, w.Code)
	assert.Empty(t, w.Body.String())
}

func TestWithCacheHeaderTTL(t *testing.T) {
	loader := loaderFunc(func(r *http.Request, image string) (blob *Blob, err error) {
		return NewBlobFromBytes([]byte("ok")), nil
	})
	t.Run("default", func(t *testing.T) {
		app := New(
			WithLogger(zap.NewExample()),
			WithLoaders(loader),
			WithUnsafe(true))
		w := httptest.NewRecorder()
		app.ServeHTTP(w, httptest.NewRequest(
			http.MethodGet, "https://example.com/unsafe/foo.jpg", nil))
		assert.Equal(t, 200, w.Code)
		assert.Equal(t, "public, s-maxage=604800, max-age=604800, no-transform, stale-while-revalidate=86400", w.Header().Get("Cache-Control"))
	})
	t.Run("custom ttl swr", func(t *testing.T) {
		app := New(
			WithLogger(zap.NewExample()),
			WithCacheHeaderSWR(time.Second*167),
			WithCacheHeaderTTL(time.Second*169),
			WithLoaders(loader),
			WithUnsafe(true))
		w := httptest.NewRecorder()
		app.ServeHTTP(w, httptest.NewRequest(
			http.MethodGet, "https://example.com/unsafe/foo.jpg", nil))
		assert.Equal(t, 200, w.Code)
		assert.Equal(t, "public, s-maxage=169, max-age=169, no-transform, stale-while-revalidate=167", w.Header().Get("Cache-Control"))
	})
	t.Run("custom ttl swr private", func(t *testing.T) {
		app := New(
			WithLogger(zap.NewExample()),
			WithCacheHeaderSWR(time.Second*167),
			WithCacheHeaderTTL(time.Second*169),
			WithLoaders(loader),
			WithUnsafe(true))
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "https://example.com/unsafe/foo.jpg", nil)
		r.Header.Set("Cache-Control", "private")
		app.ServeHTTP(w, r)
		assert.Equal(t, 200, w.Code)
		assert.Equal(t, "private, max-age=169, no-transform, stale-while-revalidate=167", w.Header().Get("Cache-Control"))
	})
	t.Run("custom ttl no swr", func(t *testing.T) {
		app := New(
			WithLogger(zap.NewExample()),
			WithCacheHeaderSWR(time.Second*169),
			WithCacheHeaderTTL(time.Second*169),
			WithLoaders(loader),
			WithUnsafe(true))
		w := httptest.NewRecorder()
		app.ServeHTTP(w, httptest.NewRequest(
			http.MethodGet, "https://example.com/unsafe/foo.jpg", nil))
		assert.Equal(t, 200, w.Code)
		assert.Equal(t, "public, s-maxage=169, max-age=169, no-transform", w.Header().Get("Cache-Control"))
	})
	t.Run("no cache", func(t *testing.T) {
		app := New(
			WithLoaders(loader),
			WithCacheHeaderNoCache(true),
			WithUnsafe(true))
		w := httptest.NewRecorder()
		app.ServeHTTP(w, httptest.NewRequest(
			http.MethodGet, "https://example.com/unsafe/foo.jpg", nil))
		assert.Equal(t, 200, w.Code)
		assert.NotEmpty(t, w.Header().Get("Expires"))
		assert.Equal(t, "private, no-cache, no-store, must-revalidate", w.Header().Get("Cache-Control"))
	})
}

func TestExpire(t *testing.T) {
	loader := loaderFunc(func(r *http.Request, image string) (blob *Blob, err error) {
		return NewBlobFromBytes([]byte("ok")), nil
	})
	app := New(
		WithLogger(zap.NewExample()),
		WithCacheHeaderSWR(time.Second*169),
		WithCacheHeaderTTL(time.Second*169),
		WithLoaders(loader),
		WithResultStorages(saverFunc(func(ctx context.Context, image string, blob *Blob) error {
			assert.NotContains(t, image, "expire")
			return nil
		})),
		WithUnsafe(true))
	w := httptest.NewRecorder()
	app.ServeHTTP(w, httptest.NewRequest(
		http.MethodGet, "https://example.com/unsafe/filters:foo(bar)/foo.jpg", nil))
	assert.Equal(t, 200, w.Code)
	assert.Equal(t, "public, s-maxage=169, max-age=169, no-transform", w.Header().Get("Cache-Control"))

	now := time.Now()
	time.Sleep(time.Millisecond)

	w = httptest.NewRecorder()
	app.ServeHTTP(w, httptest.NewRequest(
		http.MethodGet,
		fmt.Sprintf("https://example.com/unsafe/filters:expire(%d):foo(bar)/foo.jpg",
			now.Add(time.Second).UnixMilli(),
		), nil))
	assert.Equal(t, 200, w.Code)
	assert.Equal(t, "private, max-age=1, no-transform", w.Header().Get("Cache-Control"))

	w = httptest.NewRecorder()
	app.ServeHTTP(w, httptest.NewRequest(
		http.MethodGet,
		fmt.Sprintf("https://example.com/unsafe/filters:expire(%d):foo(bar)/foo.jpg",
			now.Add(time.Second*170).UnixMilli(),
		), nil))
	assert.Equal(t, 200, w.Code)
	assert.Equal(t, "private, max-age=169, no-transform", w.Header().Get("Cache-Control"))

	w = httptest.NewRecorder()
	app.ServeHTTP(w, httptest.NewRequest(
		http.MethodGet,
		fmt.Sprintf("https://example.com/unsafe/filters:expire(%d):foo(bar)/foo.jpg",
			now.UnixMilli(),
		), nil))
	assert.Equal(t, 410, w.Code)
}

func TestVersion(t *testing.T) {
	app := New(
		WithDebug(true),
		WithLogger(zap.NewExample()),
	)

	r := httptest.NewRequest(
		http.MethodGet, "https://example.com/", nil)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, r)
	assert.Equal(t, 200, w.Code)
	assert.Contains(t, w.Body.String(), Version)
}

func TestWithBasePathRedirect(t *testing.T) {
	app := New(
		WithDebug(true),
		WithBasePathRedirect("https://www.bar.com"),
		WithLogger(zap.NewExample()),
	)
	r := httptest.NewRequest(
		http.MethodGet, "https://foo.com/", nil)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, r)
	assert.Equal(t, http.StatusTemporaryRedirect, w.Code)
	assert.Equal(t, "https://www.bar.com", w.Header().Get("Location"))
}

func TestParams(t *testing.T) {
	app := New(
		WithDebug(true),
		WithLogger(zap.NewExample()),
		WithSigner(imagorpath.NewDefaultSigner("1234")))

	r := httptest.NewRequest(
		http.MethodGet, "https://example.com/params/_-19cQt1szHeUV0WyWFntvTImDI=/foo.jpg", nil)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, r)
	assert.Equal(t, 200, w.Code)
	buf, _ := json.MarshalIndent(imagorpath.Parse(r.URL.EscapedPath()), "", "  ")
	assert.Equal(t, string(buf), w.Body.String())

	r = httptest.NewRequest(
		http.MethodGet, "https://example.com/params/foo.jpg", nil)
	w = httptest.NewRecorder()
	app.ServeHTTP(w, r)
	assert.Equal(t, 200, w.Code)
	buf, _ = json.MarshalIndent(imagorpath.Parse(r.URL.EscapedPath()), "", "  ")
	assert.Equal(t, string(buf), w.Body.String())

	app = New(
		WithDebug(true),
		WithLogger(zap.NewExample()),
		WithDisableParamsEndpoint(true),
		WithSigner(imagorpath.NewDefaultSigner("1234")))
	r = httptest.NewRequest(
		http.MethodGet, "https://example.com/params/_-19cQt1szHeUV0WyWFntvTImDI=/foo.jpg", nil)
	w = httptest.NewRecorder()
	app.ServeHTTP(w, r)
	assert.Equal(t, 200, w.Code)
	assert.Empty(t, w.Body.String())
}

var clock time.Time

type mapStore struct {
	l       sync.RWMutex
	Map     map[string]*Blob
	ModTime map[string]time.Time
	LoadCnt map[string]int
	SaveCnt map[string]int
	DelCnt  map[string]int
}

func newMapStore() *mapStore {
	return &mapStore{
		Map: map[string]*Blob{}, LoadCnt: map[string]int{}, SaveCnt: map[string]int{},
		DelCnt: map[string]int{}, ModTime: map[string]time.Time{},
	}
}

func (s *mapStore) Get(r *http.Request, image string) (*Blob, error) {
	s.l.RLock()
	defer s.l.RUnlock()
	buf, ok := s.Map[image]
	if !ok {
		return nil, ErrNotFound
	}
	buf.Stat, _ = s.Stat(r.Context(), image)
	s.LoadCnt[image] = s.LoadCnt[image] + 1
	return buf, nil
}

func (s *mapStore) Put(ctx context.Context, image string, blob *Blob) error {
	s.l.Lock()
	defer s.l.Unlock()
	clock = clock.Add(1)
	s.Map[image] = blob
	s.SaveCnt[image] = s.SaveCnt[image] + 1
	s.ModTime[image] = clock
	return nil
}

func (s *mapStore) Delete(ctx context.Context, image string) error {
	s.l.Lock()
	defer s.l.Unlock()
	delete(s.Map, image)
	s.DelCnt[image] = s.DelCnt[image] + 1
	return nil
}

func (s *mapStore) Stat(ctx context.Context, image string) (*Stat, error) {
	s.l.RLock()
	defer s.l.RUnlock()
	t, ok := s.ModTime[image]
	if !ok {
		return nil, ErrNotFound
	}
	b, ok := s.Map[image]
	if !ok {
		return nil, ErrNotFound
	}
	return &Stat{
		ModifiedTime: t,
		Size:         b.Size(),
	}, nil
}

func TestWithLoadersStoragesProcessors(t *testing.T) {
	store := newMapStore()
	resultStore := newMapStore()
	newFakeBlob := func(str string) *Blob {
		return NewBlob(func() (io.ReadCloser, int64, error) {
			return io.NopCloser(bytes.NewReader([]byte(str))), 0, nil
		})
	}
	app := New(
		WithDebug(true), WithLogger(zap.NewExample()),
		WithLoaders(
			loaderFunc(func(r *http.Request, image string) (*Blob, error) {
				if image == "foo" {
					return newFakeBlob("bar"), nil
				}
				if image == "bar" {
					return newFakeBlob("foo"), nil
				}
				if image == "ping" {
					return newFakeBlob("pong"), nil
				}
				if image == "empty" {
					return nil, nil
				}
				return nil, ErrNotFound
			}),
			loaderFunc(func(r *http.Request, image string) (*Blob, error) {
				if image == "beep" {
					return newFakeBlob("boop"), nil
				}
				if image == "boom" {
					return nil, errors.New("unexpected error")
				}
				if image == "poop" {
					return newFakeBlob("poop"), nil
				}
				if image == "bond" {
					return newFakeBlob("bond"), nil
				}
				if image == "timeout" {
					return newFakeBlob("timeout"), nil
				}
				if image == "dood" {
					return newFakeBlob("dood"), errors.New("error with value")
				}
				return nil, ErrNotFound
			}),
		),
		WithStorages(
			store,
			saverFunc(func(ctx context.Context, image string, buf *Blob) error {
				time.Sleep(time.Millisecond * 2)
				require.NotEqual(t, "dood", image, "should not save error")
				assert.Error(t, context.DeadlineExceeded, ctx.Err())
				return ctx.Err()
			}),
		),
		WithResultStorages(resultStore),
		WithProcessors(
			processorFunc(func(ctx context.Context, blob *Blob, p imagorpath.Params, load LoadFunc) (*Blob, error) {
				buf, _ := blob.ReadAll()
				if string(buf) == "bar" {
					p.Width = 167
					return newFakeBlob("tar"), ErrForward{p}
				}
				if string(buf) == "poop" {
					p.Height = 169
					return nil, ErrForward{p}
				}
				if string(buf) == "bond" {
					return nil, ErrInternal
				}
				if string(buf) == "foo" {
					file, err := load("foo")
					if err != nil {
						return nil, err
					}
					return file, err
				}
				return blob, nil
			}),
			processorFunc(func(ctx context.Context, blob *Blob, p imagorpath.Params, load LoadFunc) (*Blob, error) {
				buf, _ := blob.ReadAll()
				if string(buf) == "tar" {
					b := newFakeBlob(imagorpath.GeneratePath(p))
					return b, nil
				}
				if string(buf) == "poop" {
					assert.Equal(t, 169, p.Height)
					return nil, ErrForward{}
				}
				if string(buf) == "bond" {
					return nil, ErrInvalid
				}
				return blob, nil
			}),
		),
		WithSaveTimeout(time.Millisecond),
		WithProcessTimeout(time.Second),
		WithUnsafe(true),
	)
	require.NoError(t, app.Startup(context.Background()))
	assert.Equal(t, time.Second, app.ProcessTimeout)
	assert.Equal(t, time.Millisecond, app.SaveTimeout)
	defer require.NoError(t, app.Shutdown(context.Background()))
	t.Parallel()
	for i := 0; i < 2; i++ {
		t.Run(fmt.Sprintf("ok %d", i), func(t *testing.T) {
			w := httptest.NewRecorder()
			app.ServeHTTP(w, httptest.NewRequest(
				http.MethodGet, "https://example.com/unsafe/foo", nil))
			time.Sleep(time.Millisecond * 10)
			assert.Equal(t, 200, w.Code)
			assert.Equal(t, "167x0/foo", w.Body.String())

			w = httptest.NewRecorder()
			app.ServeHTTP(w, httptest.NewRequest(
				http.MethodGet, "https://example.com/unsafe/bar", nil))
			time.Sleep(time.Millisecond * 10)
			assert.Equal(t, 200, w.Code)
			assert.Equal(t, "bar", w.Body.String())

			w = httptest.NewRecorder()
			app.ServeHTTP(w, httptest.NewRequest(
				http.MethodGet, "https://example.com/unsafe/ping", nil))
			time.Sleep(time.Millisecond * 10)
			assert.Equal(t, 200, w.Code)
			assert.Equal(t, "pong", w.Body.String())
			time.Sleep(time.Millisecond * 10) // make sure storage reached
			require.NotNil(t, store.Map["ping"])
			buf, err := store.Map["ping"].ReadAll()
			require.NoError(t, err)
			assert.Equal(t, "pong", string(buf))
		})
		t.Run(fmt.Sprintf("not found on empty path %d", i), func(t *testing.T) {
			w := httptest.NewRecorder()
			app.ServeHTTP(w, httptest.NewRequest(
				http.MethodGet, "https://example.com/unsafe/", nil))
			time.Sleep(time.Millisecond * 10)
			assert.Equal(t, 404, w.Code)
			assert.Equal(t, jsonStr(ErrNotFound), w.Body.String())
		})
		t.Run(fmt.Sprintf("empty %d", i), func(t *testing.T) {
			w := httptest.NewRecorder()
			app.ServeHTTP(w, httptest.NewRequest(
				http.MethodGet, "https://example.com/unsafe/empty", nil))
			time.Sleep(time.Millisecond * 10)
			assert.Equal(t, 404, w.Code)
			assert.Equal(t, jsonStr(ErrNotFound), w.Body.String())
			assert.Nil(t, store.Map["empty"])
		})
		t.Run(fmt.Sprintf("not found on pass %d", i), func(t *testing.T) {
			w := httptest.NewRecorder()
			app.ServeHTTP(w, httptest.NewRequest(
				http.MethodGet, "https://example.com/unsafe/boooo", nil))
			time.Sleep(time.Millisecond * 10)
			assert.Equal(t, 404, w.Code)
			assert.Equal(t, jsonStr(ErrNotFound), w.Body.String())
		})
		t.Run(fmt.Sprintf("unexpected error %d", i), func(t *testing.T) {
			w := httptest.NewRecorder()
			app.ServeHTTP(w, httptest.NewRequest(
				http.MethodGet, "https://example.com/unsafe/boom", nil))
			time.Sleep(time.Millisecond * 10)
			assert.Equal(t, 500, w.Code)
			assert.Equal(t, jsonStr(NewError("unexpected error", 500)), w.Body.String())
			assert.Nil(t, store.Map["boom"])
		})
		t.Run(fmt.Sprintf("error with value %d", i), func(t *testing.T) {
			w := httptest.NewRecorder()
			app.ServeHTTP(w, httptest.NewRequest(
				http.MethodGet, "https://example.com/unsafe/dood", nil))
			time.Sleep(time.Millisecond * 10)
			assert.Equal(t, 500, w.Code)
			assert.Equal(t, jsonStr(NewError("error with value", 500)), w.Body.String())
			assert.Nil(t, store.Map["dood"])
		})
		t.Run(fmt.Sprintf("processor error return original %d", i), func(t *testing.T) {
			w := httptest.NewRecorder()
			app.ServeHTTP(w, httptest.NewRequest(
				http.MethodGet, "https://example.com/unsafe/poop", nil))
			time.Sleep(time.Millisecond * 10)
			assert.Equal(t, ErrUnsupportedFormat.Code, w.Code)
			assert.Equal(t, jsonStr(ErrUnsupportedFormat), w.Body.String())
			assert.Nil(t, store.Map["poop"])
		})
		t.Run(fmt.Sprintf("processor error return last error %d", i), func(t *testing.T) {
			w := httptest.NewRecorder()
			app.ServeHTTP(w, httptest.NewRequest(
				http.MethodGet, "https://example.com/unsafe/bond", nil))
			time.Sleep(time.Millisecond * 10)
			assert.Equal(t, ErrInternal.Code, w.Code)
			assert.Equal(t, jsonStr(ErrInternal), w.Body.String())
			assert.Nil(t, store.Map["bond"])
		})
	}
}

func TestWithResultStorageNotModified(t *testing.T) {
	resultStore := newMapStore()
	app := New(
		WithDebug(true),
		WithLogger(zap.NewExample()),
		WithResultStorages(resultStore),
		WithLoaders(loaderFunc(func(r *http.Request, image string) (*Blob, error) {
			return NewBlobFromBytes([]byte(image)), nil
		})),
		WithUnsafe(true),
	)
	w := httptest.NewRecorder()
	r := httptest.NewRequest(
		http.MethodGet, "https://example.com/unsafe/foo", nil)
	app.ServeHTTP(w, r)
	time.Sleep(time.Millisecond * 10) // make sure storage reached
	assert.Equal(t, 200, w.Code)
	assert.Equal(t, "foo", w.Body.String())
	assert.Empty(t, w.Header().Get("ETag"))

	w = httptest.NewRecorder()
	r = httptest.NewRequest(
		http.MethodGet, "https://example.com/unsafe/foo", nil)
	app.ServeHTTP(w, r)
	assert.Equal(t, 200, w.Code)
	assert.Equal(t, "foo", w.Body.String())
	etag := w.Header().Get("ETag")
	lastModified := w.Header().Get("Last-Modified")
	assert.NotEmpty(t, etag)
	assert.NotEmpty(t, lastModified)

	w = httptest.NewRecorder()
	r = httptest.NewRequest(
		http.MethodGet, "https://example.com/unsafe/foo", nil)
	r.Header.Set("If-None-Match", etag)
	app.ServeHTTP(w, r)
	assert.Equal(t, 304, w.Code)
	assert.Empty(t, w.Body.String())

	w = httptest.NewRecorder()
	r = httptest.NewRequest(
		http.MethodGet, "https://example.com/unsafe/foo", nil)
	r.Header.Set("If-None-Match", etag)
	r.Header.Set("Cache-Control", "no-cache")
	app.ServeHTTP(w, r)
	assert.Equal(t, 200, w.Code)
	assert.Equal(t, "foo", w.Body.String())

	w = httptest.NewRecorder()
	r = httptest.NewRequest(
		http.MethodGet, "https://example.com/unsafe/foo", nil)
	r.Header.Set("If-None-Match", "abcd")
	app.ServeHTTP(w, r)
	assert.Equal(t, 200, w.Code)
	assert.Equal(t, "foo", w.Body.String())

	w = httptest.NewRecorder()
	r = httptest.NewRequest(
		http.MethodGet, "https://example.com/unsafe/foo", nil)
	r.Header.Set("If-Modified-Since", clock.Add(time.Hour).Format(http.TimeFormat))
	app.ServeHTTP(w, r)
	assert.Equal(t, 304, w.Code)
	assert.Empty(t, w.Body.String())

	w = httptest.NewRecorder()
	r = httptest.NewRequest(
		http.MethodGet, "https://example.com/unsafe/foo", nil)
	r.Header.Set("If-Modified-Since", time.Time{}.Format(http.TimeFormat))
	app.ServeHTTP(w, r)
	assert.Equal(t, 200, w.Code)
	assert.Equal(t, "foo", w.Body.String())

	w = httptest.NewRecorder()
	r = httptest.NewRequest(
		http.MethodGet, "https://example.com/unsafe/foo", nil)
	r.Header.Set("If-Unmodified-Since", time.Time{}.Format(http.TimeFormat))
	app.ServeHTTP(w, r)
	assert.Equal(t, 304, w.Code)
	assert.Empty(t, w.Body.String())
}

type storageKeyFunc func(img string) string

func (fn storageKeyFunc) Hash(img string) string {
	return fn(img)
}

type resultKeyFunc func(p imagorpath.Params) string

func (fn resultKeyFunc) HashResult(p imagorpath.Params) string {
	return fn(p)
}

func TestWithResultStorageHasher(t *testing.T) {
	store := newMapStore()
	resultStore := newMapStore()
	app := New(
		WithDebug(true), WithLogger(zap.NewExample()),
		WithStorages(store),
		WithResultStorages(resultStore),
		WithLoaders(loaderFunc(func(r *http.Request, image string) (*Blob, error) {
			return NewBlobFromBytes([]byte(image)), nil
		})),
		WithResultStoragePathStyle(resultKeyFunc(func(p imagorpath.Params) string {
			if strings.Contains(p.Path, "bar") {
				return ""
			}
			return "prefix:" + strings.TrimPrefix(p.Path, "meta/")
		})),
		WithUnsafe(true),
		WithModifiedTimeCheck(true),
	)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, httptest.NewRequest(
		http.MethodGet, "https://example.com/unsafe/foo", nil))
	time.Sleep(time.Millisecond * 10) // make sure storage reached
	assert.Equal(t, 200, w.Code)
	assert.Equal(t, "foo", w.Body.String())

	w = httptest.NewRecorder()
	app.ServeHTTP(w, httptest.NewRequest(
		http.MethodGet, "https://example.com/unsafe/foo", nil))
	time.Sleep(time.Millisecond * 10) // make sure storage reached
	assert.Equal(t, 200, w.Code)
	assert.Equal(t, "foo", w.Body.String())

	assert.Equal(t, 0, store.LoadCnt["foo"])
	assert.Equal(t, 1, store.SaveCnt["foo"])
	assert.Equal(t, 1, resultStore.LoadCnt["prefix:foo"])
	assert.Equal(t, 1, resultStore.SaveCnt["prefix:foo"])
	assert.Equal(t, 1, len(resultStore.LoadCnt))
	assert.Equal(t, 1, len(resultStore.SaveCnt))

	w = httptest.NewRecorder()
	app.ServeHTTP(w, httptest.NewRequest(
		http.MethodGet, "https://example.com/unsafe/bar", nil))
	time.Sleep(time.Millisecond * 10) // make sure storage reached
	assert.Equal(t, 200, w.Code)
	assert.Equal(t, "bar", w.Body.String())

	w = httptest.NewRecorder()
	app.ServeHTTP(w, httptest.NewRequest(
		http.MethodGet, "https://example.com/unsafe/bar", nil))
	time.Sleep(time.Millisecond * 10) // make sure storage reached
	assert.Equal(t, 200, w.Code)
	assert.Equal(t, "bar", w.Body.String())

	assert.Equal(t, 1, store.LoadCnt["bar"])
	assert.Equal(t, 1, store.SaveCnt["bar"])
	assert.Equal(t, 1, len(resultStore.LoadCnt))
	assert.Equal(t, 1, len(resultStore.SaveCnt))

	w = httptest.NewRecorder()
	app.ServeHTTP(w, httptest.NewRequest(
		http.MethodGet, "https://example.com/unsafe/filters:preview()/hi", nil))
	time.Sleep(time.Millisecond * 10) // make sure storage reached
	assert.Equal(t, 200, w.Code)
	assert.Equal(t, "hi", w.Body.String())

	w = httptest.NewRecorder()
	app.ServeHTTP(w, httptest.NewRequest(
		http.MethodGet, "https://example.com/unsafe/filters:preview()/hi", nil))
	time.Sleep(time.Millisecond * 10) // make sure storage reached
	assert.Equal(t, 200, w.Code)
	assert.Equal(t, "hi", w.Body.String())

	assert.Equal(t, 1, store.LoadCnt["hi"])
	assert.Equal(t, 1, store.SaveCnt["hi"])
	assert.Equal(t, 1, len(resultStore.LoadCnt))
	assert.Equal(t, 1, len(resultStore.SaveCnt))
}

func TestWithStorageHasher(t *testing.T) {
	var loadCnt = map[string]int{}
	store := newMapStore()
	app := New(
		WithDebug(true), WithLogger(zap.NewExample()),
		WithStorages(store),
		WithLoaders(loaderFunc(func(r *http.Request, image string) (*Blob, error) {
			loadCnt[image]++
			return NewBlobFromBytes([]byte(image)), nil
		})),
		WithStoragePathStyle(storageKeyFunc(func(img string) string {
			if img == "bar" {
				return ""
			}
			return "storage:" + img
		})),
		WithProcessors(processorFunc(func(ctx context.Context, blob *Blob, p imagorpath.Params, load LoadFunc) (*Blob, error) {
			if p.Image == "err" {
				return nil, ErrInternal
			}
			return blob, nil
		})),
		WithUnsafe(true),
		WithModifiedTimeCheck(true),
	)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, httptest.NewRequest(
		http.MethodGet, "https://example.com/unsafe/foo", nil))
	time.Sleep(time.Millisecond * 10) // make sure storage reached
	assert.Equal(t, 200, w.Code)
	assert.Equal(t, "foo", w.Body.String())

	w = httptest.NewRecorder()
	app.ServeHTTP(w, httptest.NewRequest(
		http.MethodGet, "https://example.com/unsafe/foo", nil))
	time.Sleep(time.Millisecond * 10) // make sure storage reached
	assert.Equal(t, 200, w.Code)
	assert.Equal(t, "foo", w.Body.String())

	assert.Equal(t, 1, store.LoadCnt["storage:foo"])
	assert.Equal(t, 1, store.SaveCnt["storage:foo"])

	w = httptest.NewRecorder()
	app.ServeHTTP(w, httptest.NewRequest(
		http.MethodGet, "https://example.com/unsafe/bar", nil))
	time.Sleep(time.Millisecond * 10) // make sure storage reached
	assert.Equal(t, 200, w.Code)
	assert.Equal(t, "bar", w.Body.String())

	w = httptest.NewRecorder()
	app.ServeHTTP(w, httptest.NewRequest(
		http.MethodGet, "https://example.com/unsafe/bar", nil))
	time.Sleep(time.Millisecond * 10) // make sure storage reached
	assert.Equal(t, 200, w.Code)
	assert.Equal(t, "bar", w.Body.String())

	w = httptest.NewRecorder()
	app.ServeHTTP(w, httptest.NewRequest(
		http.MethodGet, "https://example.com/unsafe/err", nil))
	time.Sleep(time.Millisecond * 10) // make sure storage reached
	assert.Equal(t, 500, w.Code)
	assert.Equal(t, jsonStr(ErrInternal), w.Body.String())

	assert.Equal(t, 0, store.LoadCnt["storage:bar"])
	assert.Equal(t, 0, store.SaveCnt["storage:bar"])
	assert.Equal(t, 1, len(store.LoadCnt))
	assert.Equal(t, 2, len(store.SaveCnt))
	assert.Equal(t, 1, len(store.DelCnt))
	assert.Equal(t, 1, loadCnt["foo"], 1)
	assert.Equal(t, 2, loadCnt["bar"], 2)
	assert.Equal(t, 1, store.DelCnt["storage:err"])
}

func TestClientCancel(t *testing.T) {
	app := New(
		WithDebug(true),
		WithUnsafe(true),
		WithLogger(zap.NewExample()),
		WithLoaders(loaderFunc(func(r *http.Request, image string) (*Blob, error) {
			time.Sleep(time.Second)
			return NewBlobFromBytes([]byte(image)), nil
		})),
	)
	for i := 0; i < 5; i++ {
		ctx, cancel := context.WithCancel(context.Background())
		go func() {
			time.Sleep(time.Millisecond)
			cancel()
		}()
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "https://example.com/unsafe/foo", nil).WithContext(ctx)
		app.ServeHTTP(w, r)
		assert.Equal(t, 499, w.Code)
		assert.Empty(t, w.Body.String())
	}
}

func TestWithProcessQueueSize(t *testing.T) {
	n := 20
	conn := 3
	size := 6
	app := New(
		WithDebug(true),
		WithUnsafe(true),
		WithLogger(zap.NewExample()),
		WithProcessQueueSize(int64(size)),
		WithProcessConcurrency(int64(conn)),
		WithLoaders(loaderFunc(func(r *http.Request, image string) (*Blob, error) {
			time.Sleep(time.Millisecond * 10) // make sure storage reached
			return NewBlobFromBytes([]byte(image)), nil
		})),
	)
	cnt := make(chan int, n)
	for i := 0; i < n; i++ {
		go func(i int) {
			w := httptest.NewRecorder()
			app.ServeHTTP(w, httptest.NewRequest(
				http.MethodGet, fmt.Sprintf("https://example.com/unsafe/%d", i), nil))
			//fmt.Println(w.Body.String())
			cnt <- w.Code
		}(i)
	}
	result := map[int]int{}
	for i := 0; i < n; i++ {
		code := <-cnt
		result[code]++
	}
	assert.Equal(t, size+conn, result[200])
	assert.Equal(t, n-size-conn, result[429])
}

func TestWithProcessConcurrency(t *testing.T) {
	n := 5
	app := New(
		WithDebug(true),
		WithUnsafe(true),
		WithLogger(zap.NewExample()),
		WithProcessConcurrency(1),
		WithRequestTimeout(time.Millisecond*13),
		WithLoaders(loaderFunc(func(r *http.Request, image string) (*Blob, error) {
			time.Sleep(time.Millisecond * 10) // make sure storage reached
			return NewBlobFromBytes([]byte(image)), nil
		})),
	)
	cnt := make(chan int, n)
	for i := 0; i < n; i++ {
		go func(i int) {
			w := httptest.NewRecorder()
			app.ServeHTTP(w, httptest.NewRequest(
				http.MethodGet, fmt.Sprintf("https://example.com/unsafe/%d", i), nil))
			cnt <- w.Code
		}(i)
	}
	result := map[int]int{}
	for i := 0; i < n; i++ {
		code := <-cnt
		result[code]++
	}
	assert.Equal(t, 1, result[200])
	assert.Equal(t, 4, result[429])
}

func TestWithModifiedTimeCheck(t *testing.T) {
	store := newMapStore()
	resultStore := newMapStore()
	app := New(
		WithDebug(true), WithLogger(zap.NewExample()),
		WithStorages(store),
		WithResultStorages(resultStore),
		WithLoaders(loaderFunc(func(r *http.Request, image string) (*Blob, error) {
			return NewBlobFromBytes([]byte(image)), nil
		})),
		WithUnsafe(true),
		WithModifiedTimeCheck(true),
	)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, httptest.NewRequest(
		http.MethodGet, "https://example.com/unsafe/foo", nil))
	time.Sleep(time.Millisecond * 10) // make sure storage reached
	assert.Equal(t, 200, w.Code)
	assert.Equal(t, "foo", w.Body.String())
	assert.Equal(t, 0, store.LoadCnt["foo"])
	assert.Equal(t, 1, store.SaveCnt["foo"])
	assert.Equal(t, 0, resultStore.LoadCnt["foo"])
	assert.Equal(t, 1, resultStore.SaveCnt["foo"])

	w = httptest.NewRecorder()
	app.ServeHTTP(w, httptest.NewRequest(
		http.MethodGet, "https://example.com/unsafe/foo", nil))
	time.Sleep(time.Millisecond * 10) // make sure storage reached
	assert.Equal(t, 200, w.Code)
	assert.Equal(t, "foo", w.Body.String())
	assert.Equal(t, 0, store.LoadCnt["foo"])
	assert.Equal(t, 1, store.SaveCnt["foo"])
	assert.Equal(t, 1, resultStore.LoadCnt["foo"])
	assert.Equal(t, 1, resultStore.SaveCnt["foo"])

	clock = clock.Add(1)
	store.ModTime["foo"] = clock

	w = httptest.NewRecorder()
	app.ServeHTTP(w, httptest.NewRequest(
		http.MethodGet, "https://example.com/unsafe/foo", nil))
	time.Sleep(time.Millisecond * 10) // make sure storage reached
	assert.Equal(t, 200, w.Code)
	assert.Equal(t, 1, store.LoadCnt["foo"])
	assert.Equal(t, 1, store.SaveCnt["foo"])
	assert.Equal(t, 2, resultStore.LoadCnt["foo"])
	assert.Equal(t, 2, resultStore.SaveCnt["foo"])
}

func TestWithSameStore(t *testing.T) {
	store := newMapStore()
	app := New(
		WithDebug(true), WithLogger(zap.NewExample()),
		WithLoaders(
			store,
			loaderFunc(func(r *http.Request, image string) (*Blob, error) {
				if image == "beep" {
					return NewBlobFromBytes([]byte("boop")), nil
				}
				return nil, ErrNotFound
			}),
		),
		WithStorages(store),
		WithSaveTimeout(time.Millisecond),
		WithProcessTimeout(time.Millisecond*2),
		WithUnsafe(true),
	)
	t.Run("should not save from same store", func(t *testing.T) {
		n := 5
		for i := 0; i < n; i++ {
			w := httptest.NewRecorder()
			app.ServeHTTP(w, httptest.NewRequest(
				http.MethodGet, "https://example.com/unsafe/beep", nil))
			assert.Equal(t, 200, w.Code)
			assert.Equal(t, "boop", w.Body.String())
			time.Sleep(time.Millisecond * 10) // make sure storage reached
		}
		assert.Equal(t, n-1, store.LoadCnt["beep"])
		assert.Equal(t, 1, store.SaveCnt["beep"])
	})
}

func TestBaseParams(t *testing.T) {
	app := New(
		WithDebug(true),
		WithUnsafe(true),
		WithBaseParams("filters:watermark(example.jpg)"),
		WithLoaders(loaderFunc(func(r *http.Request, image string) (*Blob, error) {
			return NewBlobFromBytes([]byte("foo")), nil
		})),
		WithProcessors(processorFunc(func(ctx context.Context, blob *Blob, p imagorpath.Params, load LoadFunc) (*Blob, error) {
			return NewBlobFromBytes([]byte(p.Path)), nil
		})),
	)
	w := httptest.NewRecorder()
	r := httptest.NewRequest(
		http.MethodGet, "https://example.com/unsafe/fit-in/200x0/filters:format(jpg)/abc.png", nil)
	app.ServeHTTP(w, r)
	assert.Equal(t, 200, w.Code)
	assert.Equal(t, "fit-in/200x0/filters:format(jpg):watermark(example.jpg)/abc.png", w.Body.String())
}

func TestAutoWebP(t *testing.T) {
	factory := func(isAuto bool) *Imagor {
		return New(
			WithDebug(true),
			WithUnsafe(true),
			WithAutoWebP(isAuto),
			WithLoaders(loaderFunc(func(r *http.Request, image string) (*Blob, error) {
				return NewBlobFromBytes([]byte("foo")), nil
			})),
			WithProcessors(processorFunc(func(ctx context.Context, blob *Blob, p imagorpath.Params, load LoadFunc) (*Blob, error) {
				return NewBlobFromBytes([]byte(p.Path)), nil
			})),
			WithDebug(true))
	}

	t.Run("supported auto img tag not enabled", func(t *testing.T) {
		app := factory(false)
		w := httptest.NewRecorder()
		r := httptest.NewRequest(
			http.MethodGet, "https://example.com/unsafe/abc.png", nil)
		r.Header.Set("Accept", "image/avif,image/webp,image/apng,image/svg+xml,image/*,*/*;q=0.8")
		app.ServeHTTP(w, r)
		assert.Equal(t, 200, w.Code)
		assert.Equal(t, w.Body.String(), "abc.png")
	})
	t.Run("supported auto img tag", func(t *testing.T) {
		app := factory(true)
		w := httptest.NewRecorder()
		r := httptest.NewRequest(
			http.MethodGet, "https://example.com/unsafe/abc.png", nil)
		r.Header.Set("Accept", "image/avif,image/webp,image/apng,image/svg+xml,image/*,*/*;q=0.8")
		app.ServeHTTP(w, r)
		assert.Equal(t, 200, w.Code)
		assert.Equal(t, "Accept", w.Header().Get("Vary"))
		assert.Equal(t, "filters:format(webp)/abc.png", w.Body.String())
	})
	t.Run("supported not image tag auto", func(t *testing.T) {
		app := factory(true)
		w := httptest.NewRecorder()
		r := httptest.NewRequest(
			http.MethodGet, "https://example.com/unsafe/abc.png", nil)
		r.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*")
		app.ServeHTTP(w, r)
		assert.Equal(t, 200, w.Code)
		assert.Equal(t, "Accept", w.Header().Get("Vary"))
		assert.Equal(t, w.Body.String(), "filters:format(webp)/abc.png")
	})
	t.Run("no supported no auto", func(t *testing.T) {
		app := factory(true)
		w := httptest.NewRecorder()
		r := httptest.NewRequest(
			http.MethodGet, "https://example.com/unsafe/abc.png", nil)
		r.Header.Set("Accept", "image/apng,image/svg+xml,image/*,*/*;q=0.8")
		app.ServeHTTP(w, r)
		assert.Equal(t, 200, w.Code)
		assert.Equal(t, w.Body.String(), "abc.png")
	})
	t.Run("explicit format no auto", func(t *testing.T) {
		app := factory(true)
		w := httptest.NewRecorder()
		r := httptest.NewRequest(
			http.MethodGet, "https://example.com/unsafe/filters:format(jpg)/abc.png", nil)
		r.Header.Set("Accept", "image/avif,image/webp,image/apng,image/svg+xml,image/*,*/*;q=0.8")
		app.ServeHTTP(w, r)
		assert.Equal(t, 200, w.Code)
		assert.Equal(t, w.Body.String(), "filters:format(jpg)/abc.png")
	})
}

func TestAutoAVIF(t *testing.T) {
	factory := func(isAuto bool) *Imagor {
		return New(
			WithDebug(true),
			WithUnsafe(true),
			WithAutoAVIF(isAuto),
			WithLoaders(loaderFunc(func(r *http.Request, image string) (*Blob, error) {
				return NewBlobFromBytes([]byte("foo")), nil
			})),
			WithProcessors(processorFunc(func(ctx context.Context, blob *Blob, p imagorpath.Params, load LoadFunc) (*Blob, error) {
				return NewBlobFromBytes([]byte(p.Path)), nil
			})),
			WithDebug(true))
	}

	t.Run("supported auto img tag not enabled", func(t *testing.T) {
		app := factory(false)
		w := httptest.NewRecorder()
		r := httptest.NewRequest(
			http.MethodGet, "https://example.com/unsafe/abc.png", nil)
		r.Header.Set("Accept", "image/avif,image/webp,image/apng,image/svg+xml,image/*,*/*;q=0.8")
		app.ServeHTTP(w, r)
		assert.Equal(t, 200, w.Code)
		assert.Equal(t, w.Body.String(), "abc.png")
	})
	t.Run("supported auto img tag", func(t *testing.T) {
		app := factory(true)
		w := httptest.NewRecorder()
		r := httptest.NewRequest(
			http.MethodGet, "https://example.com/unsafe/abc.png", nil)
		r.Header.Set("Accept", "image/avif,image/webp,image/apng,image/svg+xml,image/*,*/*;q=0.8")
		app.ServeHTTP(w, r)
		assert.Equal(t, 200, w.Code)
		assert.Equal(t, "Accept", w.Header().Get("Vary"))
		assert.Equal(t, w.Body.String(), "filters:format(avif)/abc.png")
	})
	t.Run("supported not image tag auto", func(t *testing.T) {
		app := factory(true)
		w := httptest.NewRecorder()
		r := httptest.NewRequest(
			http.MethodGet, "https://example.com/unsafe/abc.png", nil)
		r.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*")
		app.ServeHTTP(w, r)
		assert.Equal(t, 200, w.Code)
		assert.Equal(t, "Accept", w.Header().Get("Vary"))
		assert.Equal(t, w.Body.String(), "filters:format(avif)/abc.png")
	})
	t.Run("no supported no auto", func(t *testing.T) {
		app := factory(true)
		w := httptest.NewRecorder()
		r := httptest.NewRequest(
			http.MethodGet, "https://example.com/unsafe/abc.png", nil)
		r.Header.Set("Accept", "image/apng,image/svg+xml,image/*,*/*;q=0.8")
		app.ServeHTTP(w, r)
		assert.Equal(t, 200, w.Code)
		assert.Equal(t, w.Body.String(), "abc.png")
	})
	t.Run("explicit format no auto", func(t *testing.T) {
		app := factory(true)
		w := httptest.NewRecorder()
		r := httptest.NewRequest(
			http.MethodGet, "https://example.com/unsafe/filters:format(jpg)/abc.png", nil)
		r.Header.Set("Accept", "image/avif,image/webp,image/apng,image/svg+xml,image/*,*/*;q=0.8")
		app.ServeHTTP(w, r)
		assert.Equal(t, 200, w.Code)
		assert.Equal(t, w.Body.String(), "filters:format(jpg)/abc.png")
	})
}

func TestAutoJPEG(t *testing.T) {
	factory := func(isAuto bool) *Imagor {
		return New(
			WithDebug(true),
			WithUnsafe(true),
			WithAutoJPEG(isAuto),
			WithLoaders(loaderFunc(func(r *http.Request, image string) (*Blob, error) {
				return NewBlobFromBytes([]byte("foo")), nil
			})),
			WithProcessors(processorFunc(func(ctx context.Context, blob *Blob, p imagorpath.Params, load LoadFunc) (*Blob, error) {
				return NewBlobFromBytes([]byte(p.Path)), nil
			})),
			WithDebug(true))
	}

	t.Run("supported auto img tag not enabled", func(t *testing.T) {
		app := factory(false)
		w := httptest.NewRecorder()
		r := httptest.NewRequest(
			http.MethodGet, "https://example.com/unsafe/abc.png", nil)
		r.Header.Set("Accept", "image/jpeg,image/png,image/*,*/*;q=0.8")
		app.ServeHTTP(w, r)
		assert.Equal(t, 200, w.Code)
		assert.Equal(t, w.Body.String(), "abc.png")
	})
	t.Run("supported auto img tag", func(t *testing.T) {
		app := factory(true)
		w := httptest.NewRecorder()
		r := httptest.NewRequest(
			http.MethodGet, "https://example.com/unsafe/abc.png", nil)
		r.Header.Set("Accept", "image/jpeg,image/png,image/*,*/*;q=0.8")
		app.ServeHTTP(w, r)
		assert.Equal(t, 200, w.Code)
		assert.Equal(t, "Accept", w.Header().Get("Vary"))
		assert.Equal(t, "filters:format(jpeg)/abc.png", w.Body.String())
	})
	t.Run("supported auto fallback image/*", func(t *testing.T) {
		app := factory(true)
		w := httptest.NewRecorder()
		r := httptest.NewRequest(
			http.MethodGet, "https://example.com/unsafe/abc.png", nil)
		r.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/*,*/*")
		app.ServeHTTP(w, r)
		assert.Equal(t, 200, w.Code)
		assert.Equal(t, "Accept", w.Header().Get("Vary"))
		assert.Equal(t, w.Body.String(), "filters:format(jpeg)/abc.png")
	})
	t.Run("supported auto fallback */*", func(t *testing.T) {
		app := factory(true)
		w := httptest.NewRecorder()
		r := httptest.NewRequest(
			http.MethodGet, "https://example.com/unsafe/abc.png", nil)
		r.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*")
		app.ServeHTTP(w, r)
		assert.Equal(t, 200, w.Code)
		assert.Equal(t, "Accept", w.Header().Get("Vary"))
		assert.Equal(t, w.Body.String(), "filters:format(jpeg)/abc.png")
	})
	t.Run("supported auto no accept header", func(t *testing.T) {
		app := factory(true)
		w := httptest.NewRecorder()
		r := httptest.NewRequest(
			http.MethodGet, "https://example.com/unsafe/abc.png", nil)
		// No Accept header set
		app.ServeHTTP(w, r)
		assert.Equal(t, 200, w.Code)
		assert.Equal(t, "Accept", w.Header().Get("Vary"))
		assert.Equal(t, w.Body.String(), "filters:format(jpeg)/abc.png")
	})
	t.Run("no supported no auto", func(t *testing.T) {
		app := factory(true)
		w := httptest.NewRecorder()
		r := httptest.NewRequest(
			http.MethodGet, "https://example.com/unsafe/abc.png", nil)
		r.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9")
		app.ServeHTTP(w, r)
		assert.Equal(t, 200, w.Code)
		assert.Equal(t, w.Body.String(), "abc.png")
	})
	t.Run("explicit format no auto", func(t *testing.T) {
		app := factory(true)
		w := httptest.NewRecorder()
		r := httptest.NewRequest(
			http.MethodGet, "https://example.com/unsafe/filters:format(png)/abc.png", nil)
		r.Header.Set("Accept", "image/jpeg,image/png,image/*,*/*;q=0.8")
		app.ServeHTTP(w, r)
		assert.Equal(t, 200, w.Code)
		assert.Equal(t, w.Body.String(), "filters:format(png)/abc.png")
	})
}

func TestAutoFormatPrecedence(t *testing.T) {
	factory := func(autoAVIF, autoWebP, autoJPEG bool) *Imagor {
		return New(
			WithDebug(true),
			WithUnsafe(true),
			WithAutoAVIF(autoAVIF),
			WithAutoWebP(autoWebP),
			WithAutoJPEG(autoJPEG),
			WithLoaders(loaderFunc(func(r *http.Request, image string) (*Blob, error) {
				return NewBlobFromBytes([]byte("foo")), nil
			})),
			WithProcessors(processorFunc(func(ctx context.Context, blob *Blob, p imagorpath.Params, load LoadFunc) (*Blob, error) {
				return NewBlobFromBytes([]byte(p.Path)), nil
			})),
			WithDebug(true))
	}

	t.Run("AVIF takes precedence over WebP and JPEG", func(t *testing.T) {
		app := factory(true, true, true)
		w := httptest.NewRecorder()
		r := httptest.NewRequest(
			http.MethodGet, "https://example.com/unsafe/abc.png", nil)
		r.Header.Set("Accept", "image/avif,image/webp,image/jpeg,image/*,*/*;q=0.8")
		app.ServeHTTP(w, r)
		assert.Equal(t, 200, w.Code)
		assert.Equal(t, "Accept", w.Header().Get("Vary"))
		assert.Equal(t, "filters:format(avif)/abc.png", w.Body.String())
	})

	t.Run("WebP takes precedence over JPEG when AVIF not supported", func(t *testing.T) {
		app := factory(true, true, true)
		w := httptest.NewRecorder()
		r := httptest.NewRequest(
			http.MethodGet, "https://example.com/unsafe/abc.png", nil)
		r.Header.Set("Accept", "image/webp,image/jpeg,image/*,*/*;q=0.8")
		app.ServeHTTP(w, r)
		assert.Equal(t, 200, w.Code)
		assert.Equal(t, "Accept", w.Header().Get("Vary"))
		assert.Equal(t, "filters:format(webp)/abc.png", w.Body.String())
	})

	t.Run("JPEG fallback when AVIF and WebP not supported", func(t *testing.T) {
		app := factory(true, true, true)
		w := httptest.NewRecorder()
		r := httptest.NewRequest(
			http.MethodGet, "https://example.com/unsafe/abc.png", nil)
		r.Header.Set("Accept", "image/jpeg,image/png,image/*,*/*;q=0.8")
		app.ServeHTTP(w, r)
		assert.Equal(t, 200, w.Code)
		assert.Equal(t, "Accept", w.Header().Get("Vary"))
		assert.Equal(t, "filters:format(jpeg)/abc.png", w.Body.String())
	})

	t.Run("JPEG only when AVIF and WebP disabled", func(t *testing.T) {
		app := factory(false, false, true)
		w := httptest.NewRecorder()
		r := httptest.NewRequest(
			http.MethodGet, "https://example.com/unsafe/abc.png", nil)
		r.Header.Set("Accept", "image/avif,image/webp,image/jpeg,image/*,*/*;q=0.8")
		app.ServeHTTP(w, r)
		assert.Equal(t, 200, w.Code)
		assert.Equal(t, "Accept", w.Header().Get("Vary"))
		assert.Equal(t, "filters:format(jpeg)/abc.png", w.Body.String())
	})

	t.Run("no auto format when none supported and JPEG fallback disabled", func(t *testing.T) {
		app := factory(true, true, false)
		w := httptest.NewRecorder()
		r := httptest.NewRequest(
			http.MethodGet, "https://example.com/unsafe/abc.png", nil)
		r.Header.Set("Accept", "image/jpeg,image/png,*/*;q=0.8")
		app.ServeHTTP(w, r)
		assert.Equal(t, 200, w.Code)
		assert.Equal(t, w.Body.String(), "abc.png")
	})
}

func TestWithTimeout(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.String(), "sleep") {
			time.Sleep(time.Millisecond * 50)
		}
		_, _ = w.Write([]byte("ok"))
	}))
	defer ts.Close()

	loader := loaderFunc(func(r *http.Request, image string) (blob *Blob, err error) {
		req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, image, nil)
		if err != nil {
			return nil, err
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, err
		}
		buf, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		return NewBlobFromBytes(buf), err
	})

	tests := []struct {
		name string
		app  *Imagor
	}{
		{
			name: "load timeout",
			app: New(
				WithUnsafe(true),
				WithLoadTimeout(time.Millisecond*10),
				WithLoaders(loader),
			),
		},
		{
			name: "request timeout",
			app: New(
				WithUnsafe(true),
				WithRequestTimeout(time.Millisecond*10),
				WithLoaders(loader),
			),
		},
		{
			name: "load timeout > request timeout",
			app: New(
				WithUnsafe(true),
				WithLoadTimeout(time.Millisecond*10),
				WithRequestTimeout(time.Millisecond*100),
				WithLoaders(loader),
			),
		},
		{
			name: "load timeout < request timeout",
			app: New(
				WithUnsafe(true),
				WithLoadTimeout(time.Millisecond*100),
				WithRequestTimeout(time.Millisecond*10),
				WithLoaders(loader),
			),
		},
		{
			name: "process timeout",
			app: New(
				WithUnsafe(true),
				WithRequestTimeout(time.Millisecond*10),
				WithLoaders(loaderFunc(func(r *http.Request, image string) (blob *Blob, err error) {
					return NewBlobFromBytes([]byte("ok")), nil
				})),
				WithProcessors(processorFunc(func(ctx context.Context, blob *Blob, p imagorpath.Params, load LoadFunc) (*Blob, error) {
					if strings.Contains(p.Path, "sleep") {
						time.Sleep(time.Millisecond * 50)
					}
					return blob, nil
				})),
			),
		},
	}
	for _, tt := range tests {
		t.Run("ok", func(t *testing.T) {
			w := httptest.NewRecorder()
			tt.app.ServeHTTP(w, httptest.NewRequest(
				http.MethodGet, fmt.Sprintf("https://example.com/unsafe/%s", ts.URL), nil))
			assert.Equal(t, 200, w.Code)
			assert.Equal(t, w.Body.String(), "ok")
		})
		t.Run("timeout", func(t *testing.T) {
			w := httptest.NewRecorder()
			tt.app.ServeHTTP(w, httptest.NewRequest(
				http.MethodGet, fmt.Sprintf("https://example.com/unsafe/%s/sleep", ts.URL), nil))
			assert.Equal(t, http.StatusRequestTimeout, w.Code)
			assert.Equal(t, w.Body.String(), jsonStr(ErrTimeout))
		})
	}
}

func TestSuppression(t *testing.T) {
	app := New(
		WithDebug(true), WithLogger(zap.NewExample()),
		WithLoaders(
			loaderFunc(func(r *http.Request, image string) (*Blob, error) {
				randBytes := make([]byte, 100)
				rand.Read(randBytes)
				time.Sleep(time.Millisecond * 100)
				return NewBlobFromBytes(randBytes), nil
			}),
		),
		WithUnsafe(true),
	)
	n := 10
	type res struct {
		Name string
		Val  string
	}
	resChan := make(chan res)
	defer close(resChan)
	do := func(image string) {
		w := httptest.NewRecorder()
		app.ServeHTTP(w, httptest.NewRequest(
			http.MethodGet, "https://example.com/unsafe/"+image, nil))
		assert.Equal(t, 200, w.Code)
		resChan <- res{image, w.Body.String()}
	}
	for i := 0; i < n; i++ {
		// should suppress calls so every call of same image must be same value
		// though a and b must be different value
		go do("a")
		go do("b")
	}
	resMap := map[string]string{}
	for i := 0; i < n*2; i++ {
		res := <-resChan
		if val, ok := resMap[res.Name]; ok {
			assert.Equal(t, val, res.Val)
		} else {
			resMap[res.Name] = res.Val
		}
	}
	assert.NotEqual(t, resMap["a"], resMap["b"])
}

func jsonStr(v interface{}) string {
	buf, _ := json.Marshal(v)
	return string(buf)
}

type loaderFunc func(r *http.Request, image string) (blob *Blob, err error)

func (f loaderFunc) Get(r *http.Request, image string) (*Blob, error) {
	return f(r, image)
}

type saverFunc func(ctx context.Context, image string, blob *Blob) error

func (f saverFunc) Get(r *http.Request, image string) (*Blob, error) {
	// dummy
	return nil, ErrNotFound
}

func (f saverFunc) Stat(ctx context.Context, image string) (*Stat, error) {
	// dummy
	return nil, ErrNotFound
}

func (f saverFunc) Delete(ctx context.Context, image string) error {
	// dummy
	return nil
}

func (f saverFunc) Put(ctx context.Context, image string, blob *Blob) error {
	return f(ctx, image, blob)
}

// loaderWithStat implements both Loader and Stater interfaces for testing
type loaderWithStat struct {
	data      map[string]*Blob
	modTime   map[string]time.Time
	statErr   error
	statCalls int
	mu        sync.Mutex
}

func newLoaderWithStat() *loaderWithStat {
	return &loaderWithStat{
		data:    make(map[string]*Blob),
		modTime: make(map[string]time.Time),
	}
}

func (l *loaderWithStat) Get(r *http.Request, key string) (*Blob, error) {
	if blob, ok := l.data[key]; ok {
		return blob, nil
	}
	return nil, ErrNotFound
}

func (l *loaderWithStat) Stat(ctx context.Context, key string) (*Stat, error) {
	l.mu.Lock()
	l.statCalls++
	l.mu.Unlock()

	if l.statErr != nil {
		return nil, l.statErr
	}
	if modTime, ok := l.modTime[key]; ok {
		if blob, ok := l.data[key]; ok {
			return &Stat{
				ModifiedTime: modTime,
				Size:         blob.Size(),
			}, nil
		}
	}
	return nil, ErrNotFound
}

func (l *loaderWithStat) GetStatCalls() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.statCalls
}

func TestWithModifiedTimeCheckLoader(t *testing.T) {
	resultStore := newMapStore()
	loaderStat := newLoaderWithStat()

	// Set up loader with data
	loaderStat.data["test-image"] = NewBlobFromBytes([]byte("test content"))
	loaderStat.modTime["test-image"] = time.Now()

	app := New(
		WithDebug(true), WithLogger(zap.NewExample()),
		WithLoaders(loaderStat), // No storages configured - should use loader stat
		WithResultStorages(resultStore),
		WithUnsafe(true),
		WithModifiedTimeCheck(true),
	)

	// First request
	w := httptest.NewRecorder()
	app.ServeHTTP(w, httptest.NewRequest(
		http.MethodGet, "https://example.com/unsafe/test-image", nil))
	time.Sleep(time.Millisecond * 10)
	assert.Equal(t, 200, w.Code)

	// Second request - should call loader stat for comparison
	w = httptest.NewRecorder()
	app.ServeHTTP(w, httptest.NewRequest(
		http.MethodGet, "https://example.com/unsafe/test-image", nil))
	time.Sleep(time.Millisecond * 10)
	assert.Equal(t, 200, w.Code)

	// Verify that loader stat was called (key test for the new functionality)
	assert.Greater(t, loaderStat.GetStatCalls(), 0, "Loader Stat method should have been called")
}

func TestWithModifiedTimeCheckLoaderStatFallback(t *testing.T) {
	resultStore := newMapStore()
	store := newMapStore()
	loaderStat := newLoaderWithStat()

	// Set up loader to fail stat operation
	loaderStat.data["test-image"] = NewBlobFromBytes([]byte("test content"))
	loaderStat.statErr = errors.New("stat failed")

	// Set up storage with stat capability
	store.Map["test-image"] = NewBlobFromBytes([]byte("test content"))
	store.ModTime["test-image"] = time.Now()

	app := New(
		WithDebug(true), WithLogger(zap.NewExample()),
		WithLoaders(loaderStat),
		WithStorages(store), // Storage available for fallback
		WithResultStorages(resultStore),
		WithUnsafe(true),
		WithModifiedTimeCheck(true),
	)

	// Request should succeed and fall back to storage stat
	w := httptest.NewRecorder()
	app.ServeHTTP(w, httptest.NewRequest(
		http.MethodGet, "https://example.com/unsafe/test-image", nil))
	time.Sleep(time.Millisecond * 10)
	assert.Equal(t, 200, w.Code)
	assert.Equal(t, "test content", w.Body.String())
	assert.Equal(t, 1, resultStore.SaveCnt["test-image"])
}

func TestWithModifiedTimeCheckLoaderNoStater(t *testing.T) {
	resultStore := newMapStore()
	store := newMapStore()

	// Regular loader without Stater interface
	regularLoader := loaderFunc(func(r *http.Request, image string) (*Blob, error) {
		if image == "test-image" {
			return NewBlobFromBytes([]byte("test content")), nil
		}
		return nil, ErrNotFound
	})

	// Set up storage with stat capability
	store.Map["test-image"] = NewBlobFromBytes([]byte("test content"))
	store.ModTime["test-image"] = time.Now()

	app := New(
		WithDebug(true), WithLogger(zap.NewExample()),
		WithLoaders(regularLoader), // Loader without Stater
		WithStorages(store),
		WithResultStorages(resultStore),
		WithUnsafe(true),
		WithModifiedTimeCheck(true),
	)

	// Should fall back to storage stat since loader doesn't implement Stater
	w := httptest.NewRecorder()
	app.ServeHTTP(w, httptest.NewRequest(
		http.MethodGet, "https://example.com/unsafe/test-image", nil))
	time.Sleep(time.Millisecond * 10)
	assert.Equal(t, 200, w.Code)
	assert.Equal(t, "test content", w.Body.String())
	assert.Equal(t, 1, resultStore.SaveCnt["test-image"])
}

type processorFunc func(ctx context.Context, blob *Blob, p imagorpath.Params, load LoadFunc) (*Blob, error)

func (f processorFunc) Process(ctx context.Context, blob *Blob, p imagorpath.Params, load LoadFunc) (*Blob, error) {
	return f(ctx, blob, p, load)
}
func (f processorFunc) Startup(_ context.Context) error {
	return nil
}
func (f processorFunc) Shutdown(_ context.Context) error {
	return nil
}

// failingStorage is a mock storage that can be configured to fail on Put operations
type failingStorage struct {
	*mapStore
	failOnPut bool
	putCalls  int
	mu        sync.Mutex
}

func newFailingStorage() *failingStorage {
	return &failingStorage{
		mapStore: newMapStore(),
	}
}

func (s *failingStorage) Put(ctx context.Context, image string, blob *Blob) error {
	s.mu.Lock()
	s.putCalls++
	shouldFail := s.failOnPut
	s.mu.Unlock()

	if shouldFail {
		return errors.New("simulated storage failure")
	}
	return s.mapStore.Put(ctx, image, blob)
}

func (s *failingStorage) GetPutCalls() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.putCalls
}

func (s *failingStorage) SetFailOnPut(fail bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.failOnPut = fail
}

// slowPutStorage simulates slow storage operations for timeout testing
type slowPutStorage struct {
	*mapStore
	delay time.Duration
}

func (s *slowPutStorage) Put(ctx context.Context, image string, blob *Blob) error {
	time.Sleep(s.delay)
	// Check if context was cancelled during sleep
	if ctx.Err() != nil {
		return ctx.Err()
	}
	return s.mapStore.Put(ctx, image, blob)
}

func TestResultStorageCleanupOnFailure(t *testing.T) {
	t.Run("cleanup on result storage save failure", func(t *testing.T) {
		resultStore := newFailingStorage()
		resultStore.SetFailOnPut(true) // Make it fail

		app := New(
			WithDebug(true),
			WithLogger(zap.NewExample()),
			WithUnsafe(true),
			WithLoaders(loaderFunc(func(r *http.Request, image string) (*Blob, error) {
				return NewBlobFromBytes([]byte("test content")), nil
			})),
			WithResultStorages(resultStore),
		)

		w := httptest.NewRecorder()
		app.ServeHTTP(w, httptest.NewRequest(
			http.MethodGet, "https://example.com/unsafe/test-image", nil))
		time.Sleep(time.Millisecond * 20) // Wait for async save to complete

		assert.Equal(t, 200, w.Code) // Request should still succeed
		assert.Equal(t, "test content", w.Body.String())

		// Verify Put was called
		assert.Equal(t, 1, resultStore.GetPutCalls())

		// Verify Delete was called for cleanup
		assert.Equal(t, 1, resultStore.DelCnt["test-image"])

		// Verify the file was NOT saved (due to cleanup)
		assert.Nil(t, resultStore.Map["test-image"])
	})

	t.Run("no cleanup on successful save", func(t *testing.T) {
		resultStore := newFailingStorage()
		resultStore.SetFailOnPut(false) // Make it succeed

		app := New(
			WithDebug(true),
			WithLogger(zap.NewExample()),
			WithUnsafe(true),
			WithLoaders(loaderFunc(func(r *http.Request, image string) (*Blob, error) {
				return NewBlobFromBytes([]byte("test content")), nil
			})),
			WithResultStorages(resultStore),
		)

		w := httptest.NewRecorder()
		app.ServeHTTP(w, httptest.NewRequest(
			http.MethodGet, "https://example.com/unsafe/test-image", nil))
		time.Sleep(time.Millisecond * 20) // Wait for async save to complete

		assert.Equal(t, 200, w.Code)
		assert.Equal(t, "test content", w.Body.String())

		// Verify Put was called
		assert.Equal(t, 1, resultStore.GetPutCalls())

		// Verify Delete was NOT called (no cleanup needed)
		assert.Equal(t, 0, resultStore.DelCnt["test-image"])

		// Verify the file WAS saved
		assert.NotNil(t, resultStore.Map["test-image"])
	})

	t.Run("cleanup on partial upload with multiple result storages", func(t *testing.T) {
		resultStore1 := newFailingStorage()
		resultStore2 := newFailingStorage()

		resultStore1.SetFailOnPut(true)  // First fails
		resultStore2.SetFailOnPut(false) // Second succeeds

		app := New(
			WithDebug(true),
			WithLogger(zap.NewExample()),
			WithUnsafe(true),
			WithLoaders(loaderFunc(func(r *http.Request, image string) (*Blob, error) {
				return NewBlobFromBytes([]byte("test content")), nil
			})),
			WithResultStorages(resultStore1, resultStore2),
		)

		w := httptest.NewRecorder()
		app.ServeHTTP(w, httptest.NewRequest(
			http.MethodGet, "https://example.com/unsafe/test-image", nil))
		time.Sleep(time.Millisecond * 20) // Wait for async save to complete

		assert.Equal(t, 200, w.Code)

		// Both storages should have been attempted
		assert.Equal(t, 1, resultStore1.GetPutCalls())
		assert.Equal(t, 1, resultStore2.GetPutCalls())

		// Independent cleanup: only failed storage should have cleanup called
		assert.Equal(t, 1, resultStore1.DelCnt["test-image"]) // Failed, so cleanup
		assert.Equal(t, 0, resultStore2.DelCnt["test-image"]) // Succeeded, no cleanup

		// First should not have the file (cleanup removed it), second should have it
		assert.Nil(t, resultStore1.Map["test-image"])
		assert.NotNil(t, resultStore2.Map["test-image"]) // Successful save preserved
	})

	t.Run("concurrent requests with save failures", func(t *testing.T) {
		resultStore := newFailingStorage()
		resultStore.SetFailOnPut(true)

		app := New(
			WithDebug(true),
			WithLogger(zap.NewExample()),
			WithUnsafe(true),
			WithLoaders(loaderFunc(func(r *http.Request, image string) (*Blob, error) {
				return NewBlobFromBytes([]byte(image)), nil
			})),
			WithResultStorages(resultStore),
		)

		n := 10
		var wg sync.WaitGroup
		wg.Add(n)

		for i := 0; i < n; i++ {
			go func(i int) {
				defer wg.Done()
				w := httptest.NewRecorder()
				app.ServeHTTP(w, httptest.NewRequest(
					http.MethodGet, fmt.Sprintf("https://example.com/unsafe/image-%d", i), nil))
				assert.Equal(t, 200, w.Code)
			}(i)
		}

		wg.Wait()
		time.Sleep(time.Millisecond * 50) // Wait for all async saves to complete

		// All should have attempted save
		assert.Equal(t, n, resultStore.GetPutCalls())

		// All should have cleanup
		totalDeletes := 0
		for i := 0; i < n; i++ {
			totalDeletes += resultStore.DelCnt[fmt.Sprintf("image-%d", i)]
		}
		assert.Equal(t, n, totalDeletes)
	})

	t.Run("no cleanup when result storage disabled", func(t *testing.T) {
		store := newMapStore()

		app := New(
			WithDebug(true),
			WithLogger(zap.NewExample()),
			WithUnsafe(true),
			WithLoaders(loaderFunc(func(r *http.Request, image string) (*Blob, error) {
				return NewBlobFromBytes([]byte("test content")), nil
			})),
			WithStorages(store),
			// No result storage configured
		)

		w := httptest.NewRecorder()
		app.ServeHTTP(w, httptest.NewRequest(
			http.MethodGet, "https://example.com/unsafe/test-image", nil))
		time.Sleep(time.Millisecond * 20)

		assert.Equal(t, 200, w.Code)

		// Source storage should have the file
		assert.NotNil(t, store.Map["test-image"])
		assert.Equal(t, 1, store.SaveCnt["test-image"])
		assert.Equal(t, 0, store.DelCnt["test-image"])
	})

	t.Run("cleanup on timeout during save", func(t *testing.T) {
		// Use a custom storage that sleeps during Put to simulate timeout
		slowStorage := &slowPutStorage{
			mapStore: newMapStore(),
			delay:    time.Millisecond * 100,
		}

		app := New(
			WithDebug(true),
			WithLogger(zap.NewExample()),
			WithUnsafe(true),
			WithSaveTimeout(time.Millisecond*10), // Short timeout
			WithLoaders(loaderFunc(func(r *http.Request, image string) (*Blob, error) {
				return NewBlobFromBytes([]byte("test content")), nil
			})),
			WithResultStorages(slowStorage),
		)

		w := httptest.NewRecorder()
		app.ServeHTTP(w, httptest.NewRequest(
			http.MethodGet, "https://example.com/unsafe/test-image", nil))
		time.Sleep(time.Millisecond * 150) // Wait for timeout and cleanup

		assert.Equal(t, 200, w.Code) // Request should still succeed

		// Cleanup should have been called due to timeout
		assert.Equal(t, 1, slowStorage.DelCnt["test-image"])
	})
}

func TestResultStorageCleanupWithProcessingError(t *testing.T) {
	t.Run("no result storage save on processing error", func(t *testing.T) {
		resultStore := newMapStore()

		app := New(
			WithDebug(true),
			WithLogger(zap.NewExample()),
			WithUnsafe(true),
			WithLoaders(loaderFunc(func(r *http.Request, image string) (*Blob, error) {
				return NewBlobFromBytes([]byte("test content")), nil
			})),
			WithProcessors(processorFunc(func(ctx context.Context, blob *Blob, p imagorpath.Params, load LoadFunc) (*Blob, error) {
				return nil, errors.New("processing failed")
			})),
			WithResultStorages(resultStore),
		)

		w := httptest.NewRecorder()
		app.ServeHTTP(w, httptest.NewRequest(
			http.MethodGet, "https://example.com/unsafe/test-image", nil))
		time.Sleep(time.Millisecond * 20)

		assert.Equal(t, 500, w.Code) // Processing error

		// No save should have been attempted
		assert.Equal(t, 0, resultStore.SaveCnt["test-image"])
		assert.Equal(t, 0, resultStore.DelCnt["test-image"])
		assert.Nil(t, resultStore.Map["test-image"])
	})
}

func TestResponseRawOnError(t *testing.T) {
	t.Run("disabled returns error JSON", func(t *testing.T) {
		testImageData := []byte("test image data")
		app := New(
			WithUnsafe(true),
			WithLoaders(loaderFunc(func(r *http.Request, key string) (*Blob, error) {
				return NewBlobFromBytes(testImageData), nil
			})),
			WithProcessors(processorFunc(func(ctx context.Context, blob *Blob, p imagorpath.Params, load LoadFunc) (*Blob, error) {
				return nil, ErrMaxResolutionExceeded
			})),
			WithResponseRawOnError(false),
		)

		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/unsafe/test.jpg", nil)
		app.ServeHTTP(w, req)

		assert.Equal(t, 422, w.Code)
		assert.Contains(t, w.Header().Get("Content-Type"), "application/json")
		assert.Contains(t, w.Body.String(), "maximum resolution exceeded")
	})

	t.Run("enabled returns raw image with error status", func(t *testing.T) {
		testImageData := []byte("test image data")
		app := New(
			WithUnsafe(true),
			WithLoaders(loaderFunc(func(r *http.Request, key string) (*Blob, error) {
				return NewBlobFromBytes(testImageData), nil
			})),
			WithProcessors(processorFunc(func(ctx context.Context, blob *Blob, p imagorpath.Params, load LoadFunc) (*Blob, error) {
				return nil, ErrMaxResolutionExceeded
			})),
			WithResponseRawOnError(true),
		)

		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/unsafe/test.jpg", nil)
		app.ServeHTTP(w, req)

		assert.Equal(t, 422, w.Code)
		assert.Equal(t, testImageData, w.Body.Bytes())
		assert.NotContains(t, w.Header().Get("Content-Type"), "application/json")
	})

	t.Run("no blob returns normal error", func(t *testing.T) {
		app := New(
			WithUnsafe(true),
			WithLoaders(loaderFunc(func(r *http.Request, key string) (*Blob, error) {
				return nil, ErrNotFound
			})),
			WithResponseRawOnError(true),
		)

		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/unsafe/test.jpg", nil)
		app.ServeHTTP(w, req)

		assert.Equal(t, 404, w.Code)
		assert.Contains(t, w.Header().Get("Content-Type"), "application/json")
	})

	t.Run("processing succeeds returns normal response", func(t *testing.T) {
		processedData := []byte("processed image")
		app := New(
			WithUnsafe(true),
			WithLoaders(loaderFunc(func(r *http.Request, key string) (*Blob, error) {
				return NewBlobFromBytes([]byte("original")), nil
			})),
			WithProcessors(processorFunc(func(ctx context.Context, blob *Blob, p imagorpath.Params, load LoadFunc) (*Blob, error) {
				return NewBlobFromBytes(processedData), nil
			})),
			WithResponseRawOnError(true),
		)

		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/unsafe/test.jpg", nil)
		app.ServeHTTP(w, req)

		assert.Equal(t, 200, w.Code)
		assert.Equal(t, processedData, w.Body.Bytes())
	})

	t.Run("different errors return correct status codes", func(t *testing.T) {
		testCases := []struct {
			name       string
			err        error
			expectCode int
		}{
			{"max resolution", ErrMaxResolutionExceeded, 422},
			{"unsupported format", ErrUnsupportedFormat, 406},
			{"timeout", ErrTimeout, 408},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				testImageData := []byte("test image")
				app := New(
					WithUnsafe(true),
					WithLoaders(loaderFunc(func(r *http.Request, key string) (*Blob, error) {
						return NewBlobFromBytes(testImageData), nil
					})),
					WithProcessors(processorFunc(func(ctx context.Context, blob *Blob, p imagorpath.Params, load LoadFunc) (*Blob, error) {
						return nil, tc.err
					})),
					WithResponseRawOnError(true),
				)

				w := httptest.NewRecorder()
				req := httptest.NewRequest(http.MethodGet, "/unsafe/test.jpg", nil)
				app.ServeHTTP(w, req)

				assert.Equal(t, tc.expectCode, w.Code)
				assert.Equal(t, testImageData, w.Body.Bytes())
			})
		}
	})

	t.Run("loader timeout with nil blob returns error", func(t *testing.T) {
		app := New(
			WithUnsafe(true),
			WithLoaders(loaderFunc(func(r *http.Request, key string) (*Blob, error) {
				// Simulate loader timeout - returns nil blob with timeout error
				return nil, ErrTimeout
			})),
			WithResponseRawOnError(true),
		)

		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/unsafe/test.jpg", nil)
		app.ServeHTTP(w, req)

		// Should return error JSON, not crash
		assert.Equal(t, 408, w.Code)
		assert.Contains(t, w.Header().Get("Content-Type"), "application/json")
		assert.Contains(t, w.Body.String(), "timeout")
	})
}
