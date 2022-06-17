package imagor

import (
	"context"
	"sync"
)

type cbKey struct{}

type cbRef struct {
	funcs []func()
	l     sync.Mutex
}

func (r *cbRef) Add(fn func()) {
	r.l.Lock()
	r.funcs = append(r.funcs, fn)
	r.l.Unlock()
}

func (r *cbRef) Close() {
	r.l.Lock()
	defer r.l.Unlock()
	for _, fn := range r.funcs {
		fn()
	}
	r.funcs = nil
}

// withInitDefer context with callDefer tracking
func withInitDefer(ctx context.Context) context.Context {
	return context.WithValue(ctx, cbKey{}, &cbRef{})
}

// Defer add func to context, defer called at the end of request
func Defer(ctx context.Context, fn func()) {
	if r, ok := ctx.Value(cbKey{}).(*cbRef); ok {
		r.Add(fn)
	}
}

// callDefer calls all funcs that are being tracked through the context
func callDefer(ctx context.Context) {
	if r, ok := ctx.Value(cbKey{}).(*cbRef); ok {
		r.Close()
	}
}
