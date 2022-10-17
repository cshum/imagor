package imagor

import (
	"context"
	"errors"
	"sync"
)

type imagorContextKey struct{}

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

// WithContext context with imagor defer handling and cache
func WithContext(ctx context.Context) context.Context {
	r := &imagorContextRef{}
	ctx = context.WithValue(ctx, imagorContextKey{}, r)
	go func() {
		<-ctx.Done()
		r.Done()
	}()
	return ctx
}

func mustContextValue(ctx context.Context) *imagorContextRef {
	if r, ok := ctx.Value(imagorContextKey{}).(*imagorContextRef); ok && r != nil {
		return r
	}
	panic(errors.New("not imagor context"))
}

// Defer add func to context, defer called at the end of request
func Defer(ctx context.Context, fn func()) {
	mustContextValue(ctx).Defer(fn)
}
