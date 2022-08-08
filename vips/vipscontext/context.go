package vipscontext

import (
	"context"
	"sync"
)

type contextRefKey struct{}

type contextRef struct {
	l        sync.Mutex
	cnt      int
	cbs      []func()
	Rotate90 bool
	PageN    int
}

func (r *contextRef) Add(cnt int) {
	r.l.Lock()
	r.cnt += cnt
	r.l.Unlock()
}

func (r *contextRef) Defer(cb func()) {
	r.l.Lock()
	r.cbs = append(r.cbs, cb)
	r.l.Unlock()
}

func (r *contextRef) Done() {
	r.l.Lock()
	r.cnt--
	if r.cnt <= 0 {
		for _, cb := range r.cbs {
			cb()
		}
		r.cbs = nil
	}
	r.l.Unlock()
}

// WithContext with callback tracking
func WithContext(ctx context.Context, cnt int) context.Context {
	return context.WithValue(ctx, contextRefKey{}, &contextRef{cnt: cnt})
}

// Defer context add func for callback tracking for callback gc
func Defer(ctx context.Context, cb func()) {
	if r, ok := ctx.Value(contextRefKey{}).(*contextRef); ok {
		r.Defer(cb)
	}
}

// Done closes all image refs that are being tracked through the context
func Done(ctx context.Context) {
	if r, ok := ctx.Value(contextRefKey{}).(*contextRef); ok {
		r.Done()
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
