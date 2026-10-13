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
var sourceImageKeyContextKey = contextKey{3}
var nestedDepthContextKey = contextKey{4}

// MaxNestedDepth bounds how deep a path: source may nest inside another path:
// source. A nested path is resolved in-process, so without a bound a
// self-referencing path recurses until the stack runs out instead of failing.
// Bounds path: sources only, not the image() filter.
const MaxNestedDepth = 4

// withNestedDepth annotates a context with the current path: nesting depth.
func withNestedDepth(ctx context.Context, depth int) context.Context {
	return context.WithValue(ctx, nestedDepthContextKey, depth)
}

// nestedDepthFromContext returns the current path: nesting depth, 0 at the top.
func nestedDepthFromContext(ctx context.Context) int {
	if depth, ok := ctx.Value(nestedDepthContextKey).(int); ok {
		return depth
	}
	return 0
}

// isNestedRequest reports whether this request is a nested path: source being
// resolved inside another request. Such a request already occupies the outer
// request's concurrency slot, so it must not acquire one of its own — with
// ProcessConcurrency=1 it would take the only slot and then fail to get a
// second, turning every nested path into 429.
func isNestedRequest(ctx context.Context) bool {
	return nestedDepthFromContext(ctx) > 0
}

type imagorContextRef struct {
	funcs []func()
	l     sync.Mutex

	Blob *Blob
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
	if r, ok := ctx.Value(imagorContextKey).(*imagorContextRef); ok && r != nil {
		return ctx
	}
	r := &imagorContextRef{}
	ctx = context.WithValue(ctx, imagorContextKey, r)
	go func() {
		<-ctx.Done()
		r.Done()
	}()
	return ctx
}

func mustContextRef(ctx context.Context) *imagorContextRef {
	if r, ok := ctx.Value(imagorContextKey).(*imagorContextRef); ok && r != nil {
		return r
	}
	panic(errors.New("not imagor context"))
}

// contextDefer add func to context, defer called at the end of request
func contextDefer(ctx context.Context, fn func()) {
	mustContextRef(ctx).Defer(fn)
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

// ContextWithSourceImageKey annotates a context with the original image key used
// for wildcard bucket resolution.
func ContextWithSourceImageKey(ctx context.Context, key string) context.Context {
	if key == "" {
		return ctx
	}
	return context.WithValue(ctx, sourceImageKeyContextKey, key)
}

// SourceImageKeyFromContext returns the original image key used for wildcard
// bucket resolution.
func SourceImageKeyFromContext(ctx context.Context) string {
	if key, ok := ctx.Value(sourceImageKeyContextKey).(string); ok {
		return key
	}
	return ""
}
