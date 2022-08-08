package imagor

import (
	"context"
	"errors"
	"sync"
)

type deferKey struct{}

type deferRef struct {
	funcs []func()
	l     sync.Mutex
}

func (r *deferRef) Add(fn func()) {
	r.l.Lock()
	r.funcs = append(r.funcs, fn)
	r.l.Unlock()
}

func (r *deferRef) Call() {
	r.l.Lock()
	for _, fn := range r.funcs {
		fn()
	}
	r.funcs = nil
	r.l.Unlock()
}

// DeferContext context with func defer calls
func DeferContext(ctx context.Context) context.Context {
	r := &deferRef{}
	ctx = context.WithValue(ctx, deferKey{}, r)
	go func() {
		<-ctx.Done()
		r.Call()
	}()
	return ctx
}

// Defer add func to context, defer called at the end of request
func Defer(ctx context.Context, fn func()) {
	if r, ok := ctx.Value(deferKey{}).(*deferRef); ok {
		r.Add(fn)
	} else {
		panic(errors.New("not a defer context"))
	}
}
