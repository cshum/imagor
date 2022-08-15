package imagor

import (
	"context"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestDetachContext(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Nanosecond)
	defer cancel()
	assert.False(t, IsDetached(ctx))
	time.Sleep(time.Millisecond)
	assert.Equal(t, ctx.Err(), context.DeadlineExceeded)
	ctx = DetachContext(ctx)
	assert.True(t, IsDetached(ctx))
	assert.NoError(t, ctx.Err())
	ctx, cancel2 := context.WithTimeout(ctx, time.Millisecond*5)
	defer cancel2()
	assert.NoError(t, ctx.Err())
	assert.True(t, IsDetached(ctx))
	time.Sleep(time.Millisecond * 10)
	assert.Equal(t, ctx.Err(), context.DeadlineExceeded)
}
