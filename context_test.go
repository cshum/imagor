package imagor

import (
	"context"
	"testing"
	"time"
)

func TestDetachContext(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Nanosecond)
	defer cancel()
	if IsDetached(ctx) {
		t.Error("not detached ctx")
	}
	time.Sleep(time.Millisecond)
	ctx, cancel2 := context.WithTimeout(DetachContext(ctx), time.Millisecond*5)
	defer cancel2()
	if err := ctx.Err(); err != nil {
		t.Error(err, "should not inherit timeout")
	}
	if !IsDetached(ctx) {
		t.Error("detached ctx")
	}
	time.Sleep(time.Millisecond * 10)
	if err := ctx.Err(); err != context.DeadlineExceeded {
		t.Error("should new timeout")
	}
}
