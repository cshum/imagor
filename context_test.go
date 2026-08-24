package imagor

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDefer(t *testing.T) {
	var called atomic.Int32
	ctx, cancel := context.WithCancel(context.Background())
	assert.Panics(t, func() {
		contextDefer(ctx, func() {
			t.Fatal("should not call")
		})
	})
	ctx = withContext(ctx)
	contextDefer(ctx, func() {
		called.Add(1)
	})
	contextDefer(ctx, func() {
		called.Add(1)
	})
	cancel()
	assert.EqualValues(t, 0, called.Load(), "should call after signal")
	time.Sleep(time.Millisecond * 10)
	contextDefer(ctx, func() {
		called.Add(1)
	})
	assert.EqualValues(t, 2, called.Load(), "should count all defers before cancel")
}

func TestDetachContext(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Nanosecond)
	defer cancel()
	ctx = context.WithValue(ctx, "foo", "bar")
	assert.False(t, isDetached(ctx))
	time.Sleep(time.Millisecond)
	assert.Equal(t, ctx.Err(), context.DeadlineExceeded)
	ctx = detachContext(ctx)
	assert.True(t, isDetached(ctx))
	assert.Equal(t, "bar", ctx.Value("foo"))
	assert.NoError(t, ctx.Err())
	ctx, cancel2 := context.WithTimeout(ctx, time.Millisecond*5)
	defer cancel2()
	assert.NoError(t, ctx.Err())
	assert.True(t, isDetached(ctx))
	time.Sleep(time.Millisecond * 10)
	assert.Equal(t, ctx.Err(), context.DeadlineExceeded)
}
