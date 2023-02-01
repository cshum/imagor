package prometheusmetrics

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestWithOption(t *testing.T) {
	t.Run("default options", func(t *testing.T) {
		v := New()
		assert.Empty(t, v.Addr)
		assert.Empty(t, v.Namespace)
		assert.NotNil(t, v.Logger)
	})

	t.Run("options", func(t *testing.T) {
		l := &zap.Logger{}
		v := New(
			WithAddr("domain.example.com:1111"),
			WithNamespace("myprom"),
			WithLogger(l),
		)
		assert.Equal(t, "myprom", v.Namespace)
		assert.Equal(t, "domain.example.com:1111", v.Addr)
		assert.Equal(t, &l, &v.Logger)
	})
}
