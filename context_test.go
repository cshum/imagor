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
	ctx = WithContext(ctx)
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

func TestContextCache(t *testing.T) {
	ctx := context.Background()
	assert.NotPanics(t, func() {
		ContextCachePut(ctx, "foo", "bar")
	})
	ctx = WithContext(ctx)
	s, ok := ContextCacheGet(ctx, "foo")
	assert.False(t, ok)
	assert.Nil(t, s)
	ContextCachePut(ctx, "foo", "bar")
	s, ok = ContextCacheGet(ctx, "foo")
	assert.True(t, ok)
	assert.Equal(t, "bar", s)
}
