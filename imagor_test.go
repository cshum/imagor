package imagor

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/cshum/imagor/imagorpath"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func jsonStr(v interface{}) string {
	buf, _ := json.Marshal(v)
	return string(buf)
}

type loaderFunc func(r *http.Request, image string) (buf []byte, err error)

func (f loaderFunc) Load(r *http.Request, image string) ([]byte, error) {
	return f(r, image)
}

type storageFunc func(ctx context.Context, image string, buf []byte) error

func (f storageFunc) Save(ctx context.Context, image string, buf []byte) error {
	return f(ctx, image, buf)
}

type processorFunc func(ctx context.Context, buf []byte, p imagorpath.Params, load LoadFunc) ([]byte, *Meta, error)

func (f processorFunc) Process(ctx context.Context, buf []byte, p imagorpath.Params, load LoadFunc) ([]byte, *Meta, error) {
	return f(ctx, buf, p, load)
}
func (f processorFunc) Startup(_ context.Context) error {
	return nil
}
func (f processorFunc) Shutdown(_ context.Context) error {
	return nil
}

func TestWithUnsafe(t *testing.T) {
	logger := zap.NewNop()
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

func TestWithSecret(t *testing.T) {
	app := New(WithDebug(true), WithSecret("1234"))
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
		app := New(WithDebug(true), WithUnsafe(true))
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
	app := New(WithDebug(true), WithVersion("test"))

	r := httptest.NewRequest(
		http.MethodGet, "https://example.com/", nil)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, r)
	assert.Equal(t, 200, w.Code)
	assert.Equal(t, `{"imagor":{"version":"test"}}`, w.Body.String())
}

func TestParams(t *testing.T) {
	app := New(WithDebug(true), WithSecret("1234"))

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
	Map     map[string][]byte
	LoadCnt map[string]int
	SaveCnt map[string]int
}

func (s *mapStore) Load(r *http.Request, image string) ([]byte, error) {
	buf, ok := s.Map[image]
	if !ok {
		return nil, ErrNotFound
	}
	s.LoadCnt[image] = s.LoadCnt[image] + 1
	return buf, nil
}
func (s *mapStore) Save(ctx context.Context, image string, buf []byte) error {
	s.Map[image] = buf
	s.SaveCnt[image] = s.SaveCnt[image] + 1
	return nil
}

func TestWithLoadersStoragesProcessors(t *testing.T) {
	store := &mapStore{
		Map: map[string][]byte{}, LoadCnt: map[string]int{}, SaveCnt: map[string]int{},
	}
	fakeMeta := &Meta{Format: "a", ContentType: "b", Width: 167, Height: 167}
	fakeMetaBuf, _ := json.Marshal(fakeMeta)
	fakeMetaStr := string(fakeMetaBuf)
	app := New(
		WithDebug(true),
		WithLoaders(
			store,
			loaderFunc(func(r *http.Request, image string) ([]byte, error) {
				if image == "foo" {
					return []byte("bar"), nil
				}
				if image == "ping" {
					return []byte("pong"), nil
				}
				return nil, ErrPass
			}),
			loaderFunc(func(r *http.Request, image string) ([]byte, error) {
				if image == "beep" {
					return []byte("boop"), nil
				}
				if image == "boom" {
					return nil, errors.New("unexpected error")
				}
				return nil, ErrPass
			}),
		),
		WithStorages(
			store,
			storageFunc(func(ctx context.Context, image string, buf []byte) error {
				time.Sleep(time.Millisecond * 2)
				assert.Error(t, context.DeadlineExceeded, ctx.Err())
				return ctx.Err()
			}),
		),
		WithProcessors(
			processorFunc(func(ctx context.Context, buf []byte, p imagorpath.Params, load LoadFunc) ([]byte, *Meta, error) {
				if string(buf) == "bar" {
					return []byte("bark"), fakeMeta, nil
				}
				return buf, nil, nil
			}),
		),
		WithSaveTimeout(time.Millisecond),
		WithUnsafe(true),
	)
	assert.NoError(t, app.Startup(context.Background()))
	defer assert.NoError(t, app.Shutdown(context.Background()))
	t.Run("ok", func(t *testing.T) {
		w := httptest.NewRecorder()
		app.ServeHTTP(w, httptest.NewRequest(
			http.MethodGet, "https://example.com/unsafe/foo", nil))
		assert.Equal(t, 200, w.Code)
		assert.Equal(t, "bark", w.Body.String())

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
	t.Run("not found on pass", func(t *testing.T) {
		w := httptest.NewRecorder()
		app.ServeHTTP(w, httptest.NewRequest(
			http.MethodGet, "https://example.com/unsafe/boooo", nil))
		assert.Equal(t, 404, w.Code)
		assert.Equal(t, jsonStr(ErrNotFound), w.Body.String())
	})
	t.Run("unexpected error", func(t *testing.T) {
		w := httptest.NewRecorder()
		app.ServeHTTP(w, httptest.NewRequest(
			http.MethodGet, "https://example.com/unsafe/boom", nil))
		assert.Equal(t, 500, w.Code)
		assert.Equal(t, jsonStr(NewError("unexpected error", 500)), w.Body.String())
	})
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

func TestSuppression(t *testing.T) {
	app := New(
		WithDebug(true),
		WithLoaders(
			loaderFunc(func(r *http.Request, image string) (buf []byte, err error) {
				randBytes := make([]byte, 100)
				rand.Read(randBytes)
				time.Sleep(time.Millisecond * 100)
				return randBytes, nil
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
