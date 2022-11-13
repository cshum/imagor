package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCIDRSliceFlag(t *testing.T) {
	t.Run("set and get", func(t *testing.T) {
		var f CIDRSliceFlag
		input := "127.0.0.0/12,200.100.0.0/28"
		assert.NoError(t, f.Set(input))
		assert.Equal(t, input, f.String())
		assert.Equal(t, &f, f.Get())

	})
	t.Run("parse error", func(t *testing.T) {
		var f CIDRSliceFlag
		input := "127.0.0.0/12,200.100.0.0/28."
		assert.Error(t, f.Set(input))
	})
}
