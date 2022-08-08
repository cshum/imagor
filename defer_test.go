package imagor

import (
	"context"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestDefer(t *testing.T) {
	var called int
	ctx, cancel := context.WithCancel(context.Background())
	assert.Panics(t, func() {
		Defer(ctx, func() {
			t.Fatal("should not call")
		})
	})
	ctx = DeferContext(ctx)
	Defer(ctx, func() {
		called++
	})
	Defer(ctx, func() {
		called++
	})
	cancel()
	assert.Equal(t, 0, called, "should call after signal")
	time.Sleep(time.Millisecond * 10)
	Defer(ctx, func() {
		called++
	})
	assert.Equal(t, 2, called, "should count all defers before cancel")
}
