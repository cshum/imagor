package prometheusmetrics

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestWithOption(t *testing.T) {
	t.Run("default options", func(t *testing.T) {
		v := New()
		assert.Equal(t, "", v.Host)
		assert.Equal(t, 9000, v.Port)
		assert.Equal(t, "/metrics", v.Path)
		assert.Equal(t, ":9000", v.Addr)
		assert.NotNil(t, v.Logger)
	})

	t.Run("options", func(t *testing.T) {
		l := &zap.Logger{}
		v := New(
			WithHost("domain.example.com"),
			WithPort(1111),
			WithPath("/path"),
			WithLogger(l),
		)
		assert.Equal(t, "domain.example.com", v.Host)
		assert.Equal(t, 1111, v.Port)
		assert.Equal(t, "/path", v.Path)
		assert.Equal(t, "domain.example.com:1111", v.Addr)
		assert.Equal(t, &l, &v.Logger)
	})
}
