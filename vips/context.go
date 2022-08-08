package vips

import (
	"context"
)

type contextRefKey struct{}

type contextRef struct {
	cbs      []func()
	Rotate90 bool
	PageN    int
}

func (r *contextRef) AddCallback(cb func()) {
	r.cbs = append(r.cbs, cb)
}

func (r *contextRef) Callback() {
	for _, cb := range r.cbs {
		cb()
	}
	r.cbs = nil
}

// WithContextRef context with callback tracking
func WithContextRef(ctx context.Context) context.Context {
	return context.WithValue(ctx, contextRefKey{}, &contextRef{})
}

// AddCallback context add func for callback tracking for callback gc
func AddCallback(ctx context.Context, cb func()) {
	if r, ok := ctx.Value(contextRefKey{}).(*contextRef); ok {
		r.AddCallback(cb)
	}
}

// Callback closes all image refs that are being tracked through the context
func Callback(ctx context.Context) {
	if r, ok := ctx.Value(contextRefKey{}).(*contextRef); ok {
		r.Callback()
	}
}

func SetPageN(ctx context.Context, n int) {
	if r, ok := ctx.Value(contextRefKey{}).(*contextRef); ok {
		r.PageN = n
	}
}

func GetPageN(ctx context.Context) int {
	if r, ok := ctx.Value(contextRefKey{}).(*contextRef); ok {
		return r.PageN
	}
	return 1
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

func IsAnimated(ctx context.Context) bool {
	return GetPageN(ctx) > 1
}
