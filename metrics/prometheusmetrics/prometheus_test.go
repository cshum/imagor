package prometheusmetrics

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestWithOption(t *testing.T) {
	t.Run("default options", func(t *testing.T) {
		v := New()
		assert.Empty(t, v.Addr)
		assert.Equal(t, v.Path, "/")
		assert.NotNil(t, v.Logger)
	})

	t.Run("options", func(t *testing.T) {
		l := &zap.Logger{}
		v := New(
			WithAddr("domain.example.com:1111"),
			WithPath("/myprom"),
			WithLogger(l),
		)
		assert.Equal(t, "/myprom", v.Path)
		assert.Equal(t, "domain.example.com:1111", v.Addr)
		assert.Equal(t, &l, &v.Logger)
		w := httptest.NewRecorder()
		v.Handler.ServeHTTP(w, httptest.NewRequest(
			http.MethodGet, "https://example.com/", nil))
		assert.Equal(t, http.StatusPermanentRedirect, w.Code)
	})
}
