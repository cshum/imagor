package imagor

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/cshum/imagor/imagorpath"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"io"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

func jsonStr(v interface{}) string {
	buf, _ := json.Marshal(v)
	return string(buf)
}

type loaderFunc func(r *http.Request, image string) (blob *Bytes, err error)

func (f loaderFunc) Get(r *http.Request, image string) (*Bytes, error) {
	return f(r, image)
}

type saverFunc func(ctx context.Context, image string, blob *Bytes) error

func (f saverFunc) Get(r *http.Request, image string) (*Bytes, error) {
	// dummy
	return nil, ErrNotFound
}

func (f saverFunc) Stat(ctx context.Context, image string) (*Stat, error) {
	// dummy
	return nil, ErrNotFound
}

func (f saverFunc) Put(ctx context.Context, image string, blob *Bytes) error {
	return f(ctx, image, blob)
}

type processorFunc func(ctx context.Context, blob *Bytes, p imagorpath.Params, load LoadFunc) (*Bytes, error)

func (f processorFunc) Process(ctx context.Context, blob *Bytes, p imagorpath.Params, load LoadFunc) (*Bytes, error) {
	return f(ctx, blob, p, load)
}
func (f processorFunc) Startup(_ context.Context) error {
	return nil
}
func (f processorFunc) Shutdown(_ context.Context) error {
	return nil
}

func TestWithUnsafe(t *testing.T) {
	logger := zap.NewExample()
	app := New(WithUnsafe(true), WithLogger(logger))
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

func TestSuppressDeadlockResolve(t *testing.T) {
	ctx := context.Background()
	app := New()
	f, err := app.suppress(ctx, "a", func(ctx context.Context) (*Bytes, error) {
		return app.suppress(ctx, "b", func(ctx context.Context) (*Bytes, error) {
			return app.suppress(ctx, "a", func(ctx context.Context) (*Bytes, error) {
				return &Bytes{path: "abc"}, nil
			})
		})
	})
	assert.Equal(t, &Bytes{path: "abc"}, f)
	require.NoError(t, err)
}

func TestSuppressTimeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*10)
	defer cancel()
	app := New()
	f, err := app.suppress(ctx, "a", func(ctx context.Context) (*Bytes, error) {
		time.Sleep(time.Second)
		return &Bytes{}, nil
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
			_, err := app.suppress(context.Background(), "a", func(ctx context.Context) (*Bytes, error) {
				time.Sleep(time.Millisecond)
				return NewBytesFilePath("a"), nil
			})
			assert.Nil(t, err)
		}()
		go func() {
			defer wg.Done()
			_, _ = app.suppress(context.Background(), "a", func(ctx context.Context) (*Bytes, error) {
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

func TestWithCacheHeaderTTL(t *testing.T) {
	t.Run("default", func(t *testing.T) {
		app := New(
			WithDebug(true),
			WithLogger(zap.NewExample()),
			WithUnsafe(true))
		w := httptest.NewRecorder()
		app.ServeHTTP(w, httptest.NewRequest(
			http.MethodGet, "https://example.com/unsafe/foo.jpg", nil))
		assert.Equal(t, 200, w.Code)
		assert.Equal(t, "public, s-maxage=604800, max-age=604800, no-transform, stale-while-revalidate=86400", w.Header().Get("Cache-Control"))
	})
	t.Run("custom ttl swr", func(t *testing.T) {
		app := New(
			WithDebug(true),
			WithLogger(zap.NewExample()),
			WithCacheHeaderSWR(time.Second*167),
			WithCacheHeaderTTL(time.Second*169),
			WithUnsafe(true))
		w := httptest.NewRecorder()
		app.ServeHTTP(w, httptest.NewRequest(
			http.MethodGet, "https://example.com/unsafe/foo.jpg", nil))
		assert.Equal(t, 200, w.Code)
		assert.Equal(t, "public, s-maxage=169, max-age=169, no-transform, stale-while-revalidate=167", w.Header().Get("Cache-Control"))
	})
	t.Run("custom ttl no swr", func(t *testing.T) {
		app := New(
			WithDebug(true),
			WithLogger(zap.NewExample()),
			WithCacheHeaderSWR(time.Second*169),
			WithCacheHeaderTTL(time.Second*169),
			WithUnsafe(true))
		w := httptest.NewRecorder()
		app.ServeHTTP(w, httptest.NewRequest(
			http.MethodGet, "https://example.com/unsafe/foo.jpg", nil))
		assert.Equal(t, 200, w.Code)
		assert.Equal(t, "public, s-maxage=169, max-age=169, no-transform", w.Header().Get("Cache-Control"))
	})
	t.Run("no cache", func(t *testing.T) {
		app := New(WithDebug(true), WithCacheHeaderNoCache(true), WithUnsafe(true))
		w := httptest.NewRecorder()
		app.ServeHTTP(w, httptest.NewRequest(
			http.MethodGet, "https://example.com/unsafe/foo.jpg", nil))
		assert.Equal(t, 200, w.Code)
		assert.NotEmpty(t, w.Header().Get("Expires"))
		assert.Equal(t, "private, no-cache, no-store, must-revalidate", w.Header().Get("Cache-Control"))
	})
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
	assert.Equal(t, fmt.Sprintf(`{"imagor":{"version":"%s"}}`, Version), w.Body.String())
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
}

var clock time.Time

type mapStore struct {
	Map     map[string]*Bytes
	ModTime map[string]time.Time
	LoadCnt map[string]int
	SaveCnt map[string]int
}

func newMapStore() *mapStore {
	return &mapStore{
		Map: map[string]*Bytes{}, LoadCnt: map[string]int{}, SaveCnt: map[string]int{},
		ModTime: map[string]time.Time{},
	}
}

func (s *mapStore) Get(r *http.Request, image string) (*Bytes, error) {
	buf, ok := s.Map[image]
	if !ok {
		return nil, ErrNotFound
	}
	s.LoadCnt[image] = s.LoadCnt[image] + 1
	return buf, nil
}
func (s *mapStore) Put(ctx context.Context, image string, blob *Bytes) error {
	clock = clock.Add(1)
	s.Map[image] = blob
	s.SaveCnt[image] = s.SaveCnt[image] + 1
	s.ModTime[image] = clock
	return nil
}

func (s *mapStore) Stat(ctx context.Context, image string) (*Stat, error) {
	t, ok := s.ModTime[image]
	if !ok {
		return nil, ErrNotFound
	}
	return &Stat{
		ModifiedTime: t,
	}, nil
}

func TestWithLoadersStoragesProcessors(t *testing.T) {
	store := newMapStore()
	resultStore := newMapStore()
	fakeMeta := &Meta{Format: "a", ContentType: "b", Width: 167, Height: 167}
	fakeMetaBuf, _ := json.Marshal(fakeMeta)
	fakeMetaStr := string(fakeMetaBuf)
	app := New(
		WithDebug(true), WithLogger(zap.NewExample()),
		WithLoaders(
			loaderFunc(func(r *http.Request, image string) (*Bytes, error) {
				if image == "foo" {
					return NewBytes([]byte("bar")), nil
				}
				if image == "bar" {
					return NewBytes([]byte("foo")), nil
				}
				if image == "ping" {
					return NewBytes([]byte("pong")), nil
				}
				if image == "empty" {
					return nil, nil
				}
				return nil, ErrPass
			}),
			loaderFunc(func(r *http.Request, image string) (*Bytes, error) {
				if image == "beep" {
					return NewBytes([]byte("boop")), nil
				}
				if image == "boom" {
					return nil, errors.New("unexpected error")
				}
				if image == "poop" {
					return NewBytes([]byte("poop")), nil
				}
				if image == "timeout" {
					return NewBytes([]byte("timeout")), nil
				}
				if image == "dood" {
					return NewBytes([]byte("dood")), errors.New("error with value")
				}
				return nil, ErrPass
			}),
		),
		WithStorages(
			store,
			saverFunc(func(ctx context.Context, image string, buf *Bytes) error {
				time.Sleep(time.Millisecond * 2)
				require.NotEqual(t, "dood", image, "should not save error")
				assert.Error(t, context.DeadlineExceeded, ctx.Err())
				return ctx.Err()
			}),
		),
		WithProcessConcurrency(1000),
		WithResultStorages(resultStore),
		WithProcessors(
			processorFunc(func(ctx context.Context, blob *Bytes, p imagorpath.Params, load LoadFunc) (*Bytes, error) {
				buf, _ := blob.ReadAll()
				if string(buf) == "bar" {
					return NewBytes([]byte("tar")), ErrPass
				}
				if string(buf) == "poop" {
					return nil, ErrPass
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
			processorFunc(func(ctx context.Context, blob *Bytes, p imagorpath.Params, load LoadFunc) (*Bytes, error) {
				buf, _ := blob.ReadAll()
				if string(buf) == "tar" {
					return NewBytesWithMeta([]byte("bark"), fakeMeta), nil
				}
				if string(buf) == "poop" {
					return nil, ErrUnsupportedFormat
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
			assert.Equal(t, 200, w.Code)
			assert.Equal(t, "bark", w.Body.String())

			w = httptest.NewRecorder()
			app.ServeHTTP(w, httptest.NewRequest(
				http.MethodGet, "https://example.com/unsafe/bar", nil))
			assert.Equal(t, 200, w.Code)
			assert.Equal(t, "bar", w.Body.String())

			w = httptest.NewRecorder()
			app.ServeHTTP(w, httptest.NewRequest(
				http.MethodGet, "https://example.com/unsafe/meta/foo", nil))
			assert.Equal(t, 200, w.Code)
			assert.Equal(t, fakeMetaStr, w.Body.String())

			w = httptest.NewRecorder()
			app.ServeHTTP(w, httptest.NewRequest(
				http.MethodGet, "https://example.com/unsafe/ping", nil))
			assert.Equal(t, 200, w.Code)
			assert.Equal(t, "pong", w.Body.String())
		})
		t.Run(fmt.Sprintf("empty %d", i), func(t *testing.T) {
			w := httptest.NewRecorder()
			app.ServeHTTP(w, httptest.NewRequest(
				http.MethodGet, "https://example.com/unsafe/empty", nil))
			assert.Equal(t, 404, w.Code)
			assert.Equal(t, jsonStr(ErrNotFound), w.Body.String())
		})
		t.Run(fmt.Sprintf("not found on pass %d", i), func(t *testing.T) {
			w := httptest.NewRecorder()
			app.ServeHTTP(w, httptest.NewRequest(
				http.MethodGet, "https://example.com/unsafe/boooo", nil))
			assert.Equal(t, 404, w.Code)
			assert.Equal(t, jsonStr(ErrNotFound), w.Body.String())
		})
		t.Run(fmt.Sprintf("unexpected error %d", i), func(t *testing.T) {
			w := httptest.NewRecorder()
			app.ServeHTTP(w, httptest.NewRequest(
				http.MethodGet, "https://example.com/unsafe/boom", nil))
			assert.Equal(t, 500, w.Code)
			assert.Equal(t, jsonStr(NewError("unexpected error", 500)), w.Body.String())
		})
		t.Run(fmt.Sprintf("error with value %d", i), func(t *testing.T) {
			w := httptest.NewRecorder()
			app.ServeHTTP(w, httptest.NewRequest(
				http.MethodGet, "https://example.com/unsafe/dood", nil))
			assert.Equal(t, 500, w.Code)
			assert.Equal(t, "dood", w.Body.String())
		})
		t.Run(fmt.Sprintf("processor error return original %d", i), func(t *testing.T) {
			w := httptest.NewRecorder()
			app.ServeHTTP(w, httptest.NewRequest(
				http.MethodGet, "https://example.com/unsafe/poop", nil))
			assert.Equal(t, ErrUnsupportedFormat.Code, w.Code)
			assert.Equal(t, "poop", w.Body.String())
		})
	}
}

type resultKeyFunc func(p imagorpath.Params) string

func (fn resultKeyFunc) Generate(p imagorpath.Params) string {
	return fn(p)
}

func TestWithResultKey(t *testing.T) {
	store := newMapStore()
	resultStore := newMapStore()
	app := New(
		WithDebug(true), WithLogger(zap.NewExample()),
		WithStorages(store),
		WithResultStorages(resultStore),
		WithLoaders(loaderFunc(func(r *http.Request, image string) (*Bytes, error) {
			return NewBytes([]byte(image)), nil
		})),
		WithResultKey(resultKeyFunc(func(p imagorpath.Params) string {
			return "prefix:" + strings.TrimPrefix(p.Path, "meta/")
		})),
		WithUnsafe(true),
		WithModifiedTimeCheck(true),
	)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, httptest.NewRequest(
		http.MethodGet, "https://example.com/unsafe/foo", nil))
	assert.Equal(t, 200, w.Code)
	assert.Equal(t, "foo", w.Body.String())

	w = httptest.NewRecorder()
	app.ServeHTTP(w, httptest.NewRequest(
		http.MethodGet, "https://example.com/unsafe/foo", nil))
	assert.Equal(t, 200, w.Code)
	assert.Equal(t, "foo", w.Body.String())

	assert.Equal(t, 0, store.LoadCnt["foo"])
	assert.Equal(t, 1, store.SaveCnt["foo"])
	assert.Equal(t, 1, resultStore.LoadCnt["prefix:foo"])
	assert.Equal(t, 1, resultStore.SaveCnt["prefix:foo"])
}

func TestWithModifiedTimeCheck(t *testing.T) {
	store := newMapStore()
	resultStore := newMapStore()
	app := New(
		WithDebug(true), WithLogger(zap.NewExample()),
		WithStorages(store),
		WithResultStorages(resultStore),
		WithLoaders(loaderFunc(func(r *http.Request, image string) (*Bytes, error) {
			return NewBytes([]byte(image)), nil
		})),
		WithUnsafe(true),
		WithModifiedTimeCheck(true),
	)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, httptest.NewRequest(
		http.MethodGet, "https://example.com/unsafe/foo", nil))
	assert.Equal(t, 200, w.Code)
	assert.Equal(t, "foo", w.Body.String())
	assert.Equal(t, 0, store.LoadCnt["foo"])
	assert.Equal(t, 1, store.SaveCnt["foo"])
	assert.Equal(t, 0, resultStore.LoadCnt["foo"])
	assert.Equal(t, 1, resultStore.SaveCnt["foo"])

	w = httptest.NewRecorder()
	app.ServeHTTP(w, httptest.NewRequest(
		http.MethodGet, "https://example.com/unsafe/foo", nil))
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
			loaderFunc(func(r *http.Request, image string) (*Bytes, error) {
				if image == "beep" {
					return NewBytes([]byte("boop")), nil
				}
				return nil, ErrPass
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
		}
		assert.Equal(t, n-1, store.LoadCnt["beep"])
		assert.Equal(t, 1, store.SaveCnt["beep"])
	})
}

func TestAutoWebP(t *testing.T) {
	factory := func(isAuto bool) *Imagor {
		return New(
			WithDebug(true),
			WithUnsafe(true),
			WithAutoWebP(isAuto),
			WithLoaders(loaderFunc(func(r *http.Request, image string) (*Bytes, error) {
				return NewBytes([]byte("foo")), nil
			})),
			WithProcessors(processorFunc(func(ctx context.Context, blob *Bytes, p imagorpath.Params, load LoadFunc) (*Bytes, error) {
				return NewBytes([]byte(p.Path)), nil
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
		assert.Equal(t, w.Body.String(), "filters:format(webp)/abc.png")
	})
	t.Run("supported not image tag auto", func(t *testing.T) {
		app := factory(true)
		w := httptest.NewRecorder()
		r := httptest.NewRequest(
			http.MethodGet, "https://example.com/unsafe/abc.png", nil)
		r.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*")
		app.ServeHTTP(w, r)
		assert.Equal(t, 200, w.Code)
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
			WithLoaders(loaderFunc(func(r *http.Request, image string) (*Bytes, error) {
				return NewBytes([]byte("foo")), nil
			})),
			WithProcessors(processorFunc(func(ctx context.Context, blob *Bytes, p imagorpath.Params, load LoadFunc) (*Bytes, error) {
				return NewBytes([]byte(p.Path)), nil
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

func TestWithLoadTimeout(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.String(), "sleep") {
			time.Sleep(time.Millisecond * 50)
		}
		_, _ = w.Write([]byte("ok"))
	}))
	defer ts.Close()

	loader := loaderFunc(func(r *http.Request, image string) (blob *Bytes, err error) {
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
		return NewBytes(buf), err
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
			loaderFunc(func(r *http.Request, image string) (*Bytes, error) {
				randBytes := make([]byte, 100)
				rand.Read(randBytes)
				time.Sleep(time.Millisecond * 100)
				return NewBytes(randBytes), nil
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
