package imagor

import (
	"context"
	"time"
)

type detachContextKey struct {
	name string
}

var detachedCtxKey = &detachContextKey{"Detached"}

type detachedContext struct {
	ctx context.Context
}

func (detachedContext) Deadline() (time.Time, bool) {
	return time.Time{}, false
}

func (detachedContext) Done() <-chan struct{} {
	return nil
}

func (detachedContext) Err() error {
	return nil
}

func (d detachedContext) Value(key any) any {
	if key == detachedCtxKey {
		return true
	}
	return d.ctx.Value(key)
}

// DetachContext returns a context that keeps all the values of its parent context
// but detaches from cancellation and timeout
func DetachContext(ctx context.Context) context.Context {
	return detachedContext{ctx: ctx}
}

// IsDetached returns if context is detached
func IsDetached(ctx context.Context) bool {
	_, ok := ctx.Value(detachedCtxKey).(bool)
	return ok
}
