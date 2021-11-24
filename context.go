package imagor

import (
	"context"
	"time"
)

type contextKey struct {
	name string
}

var detachedCtxKey = &contextKey{"Detached"}

type detached struct {
	ctx context.Context
}

func (detached) Deadline() (time.Time, bool) {
	return time.Time{}, false
}

func (detached) Done() <-chan struct{} {
	return nil
}

func (detached) Err() error {
	return nil
}

func (d detached) Value(key interface{}) interface{} {
	return d.ctx.Value(key)
}

// DetachContext returns a context that keeps all the values of its parent context
// but detaches from cancellation and timeout
func DetachContext(ctx context.Context) context.Context {
	return context.WithValue(detached{ctx: ctx}, detachedCtxKey, true)
}

// IsDetached returns if context is detached
func IsDetached(ctx context.Context) bool {
	_, ok := ctx.Value(detachedCtxKey).(bool)
	return ok
}
