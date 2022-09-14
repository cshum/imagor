package vipscontext

import (
	"context"
	"sync"
)

type contextRefKey struct{}

type contextRef struct {
	l        sync.Mutex
	cbs      []func()
	Rotate90 bool
}

func (r *contextRef) Defer(cb func()) {
	r.l.Lock()
	r.cbs = append(r.cbs, cb)
	r.l.Unlock()
}

func (r *contextRef) Done() {
	r.l.Lock()
	for _, cb := range r.cbs {
		cb()
	}
	r.cbs = nil
	r.l.Unlock()
}

// WithContext with callback tracking
func WithContext(ctx context.Context) context.Context {
	return context.WithValue(ctx, contextRefKey{}, &contextRef{})
}

// Defer context add func for callback tracking for callback gc
func Defer(ctx context.Context, cb func()) {
	ctx.Value(contextRefKey{}).(*contextRef).Defer(cb)
}

// Done closes all image refs that are being tracked through the context
func Done(ctx context.Context) {
	ctx.Value(contextRefKey{}).(*contextRef).Done()
}

func SetRotate90(ctx context.Context) {
	if r, ok := ctx.Value(contextRefKey{}).(*contextRef); ok {
		r.Rotate90 = !r.Rotate90
	}
}

func IsRotate90(ctx context.Context) bool {
	if r, ok := ctx.Value(contextRefKey{}).(*contextRef); ok {
		return r.Rotate90
	}
	return false
}
