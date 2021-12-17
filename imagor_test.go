package imagor

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

func jsonStr(v interface{}) string {
	buf, _ := json.Marshal(v)
	return string(buf)
}

func TestWithUnsafe(t *testing.T) {
	app := New(WithUnsafe(true))

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
	app := New(WithSecret("1234"))

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

type mapStore struct {
	Map map[string][]byte
}

func (s *mapStore) Load(r *http.Request, image string) ([]byte, error) {
	buf, ok := s.Map[image]
	if !ok {
		return nil, ErrNotFound
	}
	return buf, nil
}
func (s *mapStore) Save(ctx context.Context, image string, buf []byte) error {
	if _, ok := s.Map[image]; ok {
		panic(errors.New("booommm"))
	}
	s.Map[image] = buf
	return nil
}

func TestWithLoaders(t *testing.T) {
	store := &mapStore{Map: map[string][]byte{}}
	app := New(
		WithLoaders(
			store,
			LoaderFunc(func(r *http.Request, image string) ([]byte, error) {
				if image == "foo" {
					return []byte("bar"), nil
				}
				return nil, ErrPass
			}),
			LoaderFunc(func(r *http.Request, image string) ([]byte, error) {
				if image == "ping" {
					return []byte("pong"), nil
				}
				return nil, ErrPass
			}),
		),
		WithStorages(store),
		WithUnsafe(true),
	)
	t.Run("ok", func(t *testing.T) {
		w := httptest.NewRecorder()
		app.ServeHTTP(w, httptest.NewRequest(
			http.MethodGet, "https://example.com/unsafe/foo", nil))
		assert.Equal(t, 200, w.Code)
		assert.Equal(t, "bar", w.Body.String())

		w = httptest.NewRecorder()
		app.ServeHTTP(w, httptest.NewRequest(
			http.MethodGet, "https://example.com/unsafe/ping", nil))
		assert.Equal(t, 200, w.Code)
		assert.Equal(t, "pong", w.Body.String())
	})
	t.Run("should not save from same store", func(t *testing.T) {
		store.Map["beep"] = []byte("boop")

		for i := 0; i < 5; i++ {
			w := httptest.NewRecorder()
			app.ServeHTTP(w, httptest.NewRequest(
				http.MethodGet, "https://example.com/unsafe/beep", nil))
			assert.Equal(t, 200, w.Code)
			assert.Equal(t, "boop", w.Body.String())
		}
	})
	t.Run("not found on pass", func(t *testing.T) {
		w := httptest.NewRecorder()
		app.ServeHTTP(w, httptest.NewRequest(
			http.MethodGet, "https://example.com/unsafe/boooo", nil))
		assert.Equal(t, 404, w.Code)
		assert.Equal(t, jsonStr(ErrNotFound), w.Body.String())
	})
}
