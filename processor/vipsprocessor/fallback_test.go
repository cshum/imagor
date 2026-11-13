package vipsprocessor

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFallbackBufferLimit(t *testing.T) {
	// The buffer fallback may hold what a decode at the resolution limit would.
	p := NewProcessor(WithMaxResolution(1000))
	assert.Equal(t, int64(4000), p.fallbackBufferLimit())

	// Every denial of service limit is off under vips-unlimited, including this one.
	p = NewProcessor(WithUnlimited(true))
	assert.Negative(t, p.fallbackBufferLimit())
}
