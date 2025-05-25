package vipsprocessor

import (
	"context"
)

type contextRefKey struct{}

type contextRef struct {
	cbs      []func()
	Rotate90 bool
}

func (r *contextRef) Defer(cb func()) {
	r.cbs = append(r.cbs, cb)
}

func (r *contextRef) Done() {
	for _, cb := range r.cbs {
		cb()
	}
	r.cbs = nil
}

// withContext with callback tracking
func withContext(ctx context.Context) context.Context {
	return context.WithValue(ctx, contextRefKey{}, &contextRef{})
}

// contextDefer context add func for callback tracking for callback gc
func contextDefer(ctx context.Context, cb func()) {
	ctx.Value(contextRefKey{}).(*contextRef).Defer(cb)
}

// contextDone closes all image refs that are being tracked through the context
func contextDone(ctx context.Context) {
	ctx.Value(contextRefKey{}).(*contextRef).Done()
}

func setRotate90(ctx context.Context) {
	if r, ok := ctx.Value(contextRefKey{}).(*contextRef); ok {
		r.Rotate90 = !r.Rotate90
	}
}

func isRotate90(ctx context.Context) bool {
	if r, ok := ctx.Value(contextRefKey{}).(*contextRef); ok {
		return r.Rotate90
	}
	return false
}
