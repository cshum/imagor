package imagor

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/cshum/imagor/imagorpath"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"io"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func jsonStr(v interface{}) string {
	buf, _ := json.Marshal(v)
	return string(buf)
}

type loaderFunc func(r *http.Request, image string) (file *File, err error)

func (f loaderFunc) Load(r *http.Request, image string) (*File, error) {
	return f(r, image)
}

type storageFunc func(ctx context.Context, image string, file *File) error

func (f storageFunc) Save(ctx context.Context, image string, file *File) error {
	return f(ctx, image, file)
}

type processorFunc func(ctx context.Context, file *File, p imagorpath.Params, load LoadFunc) (*File, error)

func (f processorFunc) Process(ctx context.Context, file *File, p imagorpath.Params, load LoadFunc) (*File, error) {
	return f(ctx, file, p, load)
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
		http.MethodGet, "https://example.com/foo.jpg", nil))
	assert.Equal(t, 403, w.Code)
	assert.Equal(t, w.Body.String(), jsonStr(ErrSignatureMismatch))
}

func TestAcquireDeadlock(t *testing.T) {
	ctx := context.Background()
	app := New()
	f, err := app.Acquire(ctx, "a", func(ctx context.Context) (*File, error) {
		return app.Acquire(ctx, "b", func(ctx context.Context) (*File, error) {
			return app.Acquire(ctx, "a", func(ctx context.Context) (*File, error) {
				return &File{}, nil
			})
		})
	})
	assert.Nil(t, f)
	assert.Equal(t, ErrDeadlock, err)
}

func TestAcquireTimeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*10)
	defer cancel()
	app := New()
	f, err := app.Acquire(ctx, "a", func(ctx context.Context) (*File, error) {
		time.Sleep(time.Second)
		return &File{}, nil
	})
	assert.Nil(t, f)
	assert.Equal(t, context.DeadlineExceeded, err)
}

func TestWithSecret(t *testing.T) {
	app := New(
		WithDebug(true),
		WithLogger(zap.NewExample()),
		WithSecret("1234"))
	assert.Equal(t, true, app.Debug)

	w := httptest.NewRecorder()
	app.ServeHTTP(w, httptest.NewRequest(
		http.MethodGet, "https://example.com/_-19cQt1szHeUV0WyWFntvTImDI=/foo.jpg", nil))
	assert.Equal(t, 200, w.Code)

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
		assert.NotEmpty(t, w.Header().Get("Expires"))
		assert.Contains(t, w.Header().Get("Cache-Control"), "public, s-maxage=")
	})
	t.Run("no cache", func(t *testing.T) {
		app := New(WithDebug(true), WithCacheHeaderTTL(-1), WithUnsafe(true))
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
		WithVersion("test"))

	r := httptest.NewRequest(
		http.MethodGet, "https://example.com/", nil)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, r)
	assert.Equal(t, 200, w.Code)
	assert.Equal(t, `{"imagor":{"version":"test"}}`, w.Body.String())
}

func TestParams(t *testing.T) {
	app := New(
		WithDebug(true),
		WithLogger(zap.NewExample()),
		WithSecret("1234"))

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

type mapStore struct {
	Map     map[string]*File
	LoadCnt map[string]int
	SaveCnt map[string]int
}

func (s *mapStore) Load(r *http.Request, image string) (*File, error) {
	buf, ok := s.Map[image]
	if !ok {
		return nil, ErrNotFound
	}
	s.LoadCnt[image] = s.LoadCnt[image] + 1
	return buf, nil
}
func (s *mapStore) Save(ctx context.Context, image string, file *File) error {
	s.Map[image] = file
	s.SaveCnt[image] = s.SaveCnt[image] + 1
	return nil
}

func TestWithLoadersStoragesProcessors(t *testing.T) {
	store := &mapStore{
		Map: map[string]*File{}, LoadCnt: map[string]int{}, SaveCnt: map[string]int{},
	}
	resultStore := &mapStore{
		Map: map[string]*File{}, LoadCnt: map[string]int{}, SaveCnt: map[string]int{},
	}
	fakeMeta := &Meta{Format: "a", ContentType: "b", Width: 167, Height: 167}
	fakeMetaBuf, _ := json.Marshal(fakeMeta)
	fakeMetaStr := string(fakeMetaBuf)
	app := New(
		WithDebug(true), WithLogger(zap.NewExample()),
		WithLoaders(
			store,
			loaderFunc(func(r *http.Request, image string) (*File, error) {
				if image == "foo" {
					return NewFileBytes([]byte("bar")), nil
				}
				if image == "bar" {
					return NewFileBytes([]byte("foo")), nil
				}
				if image == "ping" {
					return NewFileBytes([]byte("pong")), nil
				}
				if image == "empty" {
					return nil, nil
				}
				return nil, ErrPass
			}),
			loaderFunc(func(r *http.Request, image string) (*File, error) {
				if image == "beep" {
					return NewFileBytes([]byte("boop")), nil
				}
				if image == "boom" {
					return nil, errors.New("unexpected error")
				}
				if image == "poop" {
					return NewFileBytes([]byte("poop")), nil
				}
				if image == "timeout" {
					return NewFileBytes([]byte("timeout")), nil
				}
				return nil, ErrPass
			}),
		),
		WithStorages(
			store,
			storageFunc(func(ctx context.Context, image string, buf *File) error {
				time.Sleep(time.Millisecond * 2)
				assert.Error(t, context.DeadlineExceeded, ctx.Err())
				return ctx.Err()
			}),
		),
		WithResultLoaders(resultStore),
		WithResultStorages(resultStore),
		WithProcessors(
			processorFunc(func(ctx context.Context, file *File, p imagorpath.Params, load LoadFunc) (*File, error) {
				buf, _ := file.Bytes()
				if string(buf) == "bar" {
					return NewFileBytes([]byte("tar")), ErrPass
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
				if string(buf) == "timeout" {
					time.Sleep(time.Millisecond * 10)
					return file, ctx.Err()
				}
				return file, nil
			}),
			processorFunc(func(ctx context.Context, file *File, p imagorpath.Params, load LoadFunc) (*File, error) {
				buf, _ := file.Bytes()
				if string(buf) == "tar" {
					return NewFileBytesWithMeta([]byte("bark"), fakeMeta), nil
				}
				if string(buf) == "poop" {
					return nil, ErrUnsupportedFormat
				}
				return file, nil
			}),
		),
		WithSaveTimeout(time.Millisecond),
		WithProcessTimeout(time.Millisecond*2),
		WithUnsafe(true),
	)
	assert.NoError(t, app.Startup(context.Background()))
	assert.Equal(t, time.Millisecond*2, app.ProcessTimeout)
	assert.Equal(t, time.Millisecond, app.SaveTimeout)
	defer assert.NoError(t, app.Shutdown(context.Background()))
	t.Run("consistent", func(t *testing.T) {
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
				assert.Equal(t, 200, w.Code)
				assert.Equal(t, "", w.Body.String())
			})
			t.Run(fmt.Sprintf("process timeout %d", i), func(t *testing.T) {
				w := httptest.NewRecorder()
				app.ServeHTTP(w, httptest.NewRequest(
					http.MethodGet, "https://example.com/unsafe/timeout", nil))
				assert.Equal(t, http.StatusRequestTimeout, w.Code)
				assert.Equal(t, "timeout", w.Body.String())
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
			t.Run(fmt.Sprintf("processor error return original %d", i), func(t *testing.T) {
				w := httptest.NewRecorder()
				app.ServeHTTP(w, httptest.NewRequest(
					http.MethodGet, "https://example.com/unsafe/poop", nil))
				assert.Equal(t, ErrUnsupportedFormat.Code, w.Code)
				assert.Equal(t, "poop", w.Body.String())
			})
		}
	})
}
func TestWithSameStore(t *testing.T) {
	store := &mapStore{
		Map: map[string]*File{}, LoadCnt: map[string]int{}, SaveCnt: map[string]int{},
	}
	app := New(
		WithDebug(true), WithLogger(zap.NewExample()),
		WithLoaders(
			store,
			loaderFunc(func(r *http.Request, image string) (*File, error) {
				if image == "beep" {
					return NewFileBytes([]byte("boop")), nil
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

func TestWithLoadTimeout(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.String(), "sleep") {
			time.Sleep(time.Millisecond * 50)
		}
		_, _ = w.Write([]byte("ok"))
	}))
	defer ts.Close()

	loader := loaderFunc(func(r *http.Request, image string) (file *File, err error) {
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
		return NewFileBytes(buf), err
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
			loaderFunc(func(r *http.Request, image string) (*File, error) {
				randBytes := make([]byte, 100)
				rand.Read(randBytes)
				time.Sleep(time.Millisecond * 100)
				return NewFileBytes(randBytes), nil
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
		// should Acquire calls so every call of same image must be same value
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
