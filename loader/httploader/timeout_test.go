package httploader

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/cshum/imagor"
	"github.com/stretchr/testify/assert"
)

func jsonStr(v interface{}) string {
	buf, _ := json.Marshal(v)
	return string(buf)
}

func TestWithLoadTimeout(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.String(), "sleep") {
			time.Sleep(time.Millisecond * 1500)
		}
		w.Header().Set("Content-Type", "image/jpeg")
		_, _ = w.Write([]byte("ok"))
	}))
	defer ts.Close()

	// Same margins as the timeout test in imagor itself: wide enough that the ok
	// path cannot miss its deadline on a loaded runner, and the sleep wider still
	// so the timeout path always exceeds it.
	tests := []struct {
		name string
		app  *imagor.Imagor
	}{
		{
			name: "load timeout",
			app: imagor.New(
				imagor.WithUnsafe(true),
				imagor.WithLoadTimeout(time.Millisecond*300),
				imagor.WithLoaders(New()),
			),
		},
		{
			name: "request timeout",
			app: imagor.New(
				imagor.WithUnsafe(true),
				imagor.WithRequestTimeout(time.Millisecond*300),
				imagor.WithLoaders(New()),
			),
		},
		{
			name: "load timeout > request timeout",
			app: imagor.New(
				imagor.WithUnsafe(true),
				imagor.WithLoadTimeout(time.Millisecond*300),
				imagor.WithRequestTimeout(time.Second*3),
				imagor.WithLoaders(New()),
			),
		},
		{
			name: "load timeout < request timeout",
			app: imagor.New(
				imagor.WithUnsafe(true),
				imagor.WithLoadTimeout(time.Second*3),
				imagor.WithRequestTimeout(time.Millisecond*300),
				imagor.WithLoaders(New()),
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
			assert.Equal(t, jsonStr(imagor.ErrTimeout), w.Body.String())
		})
	}
}
