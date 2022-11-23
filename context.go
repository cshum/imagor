package imagor

import (
	"context"
	"errors"
	"sync"
	"time"
)

type contextKey struct {
	Type int8
}

var imagorContextKey = contextKey{1}
var detachContextKey = contextKey{2}

type imagorContextRef struct {
	funcs []func()
	l     sync.Mutex
}

func (r *imagorContextRef) Defer(fn func()) {
	r.l.Lock()
	r.funcs = append(r.funcs, fn)
	r.l.Unlock()
}

func (r *imagorContextRef) Done() {
	r.l.Lock()
	for _, fn := range r.funcs {
		fn()
	}
	r.funcs = nil
	r.l.Unlock()
}

// withContext context with imagor defer handling and cache
func withContext(ctx context.Context) context.Context {
	r := &imagorContextRef{}
	ctx = context.WithValue(ctx, imagorContextKey, r)
	go func() {
		<-ctx.Done()
		r.Done()
	}()
	return ctx
}

func mustContextValue(ctx context.Context) *imagorContextRef {
	if r, ok := ctx.Value(imagorContextKey).(*imagorContextRef); ok && r != nil {
		return r
	}
	panic(errors.New("not imagor context"))
}

// contextDefer add func to context, defer called at the end of request
func contextDefer(ctx context.Context, fn func()) {
	mustContextValue(ctx).Defer(fn)
}

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
	if key == detachContextKey {
		return true
	}
	return d.ctx.Value(key)
}

// detachContext returns a context that keeps all the values of its parent context
// but detaches from cancellation and timeout
func detachContext(ctx context.Context) context.Context {
	return detachedContext{ctx: ctx}
}

// isDetached returns if context is detached
func isDetached(ctx context.Context) bool {
	_, ok := ctx.Value(detachContextKey).(bool)
	return ok
}
