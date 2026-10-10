package vipsprocessor

import (
	"context"
)

// Global resource tracking context (persists across parent/child)
type contextResourceKey struct{}

type contextResource struct {
	cbs []func()
}

func (r *contextResource) Defer(cb func()) {
	r.cbs = append(r.cbs, cb)
}

func (r *contextResource) Done() {
	for _, cb := range r.cbs {
		cb()
	}
	r.cbs = nil
}

// contextDefer adds callback for resource cleanup (global across parent/child)
func contextDefer(ctx context.Context, cb func()) {
	if r, ok := ctx.Value(contextResourceKey{}).(*contextResource); ok {
		r.Defer(cb)
	}
}

// contextDone closes all tracked resources (global)
func contextDone(ctx context.Context) {
	if r, ok := ctx.Value(contextResourceKey{}).(*contextResource); ok {
		r.Done()
	}
}

// Local rotation context (resets for each processing level)
type contextRotateKey struct{}

type contextRotate struct {
	Rotate90        bool
	RotateArbitrary bool
}

// setRotate90 toggles rotation flag in current context (local to processing level)
func setRotate90(ctx context.Context) {
	if r, ok := ctx.Value(contextRotateKey{}).(*contextRotate); ok {
		r.Rotate90 = !r.Rotate90
	}
}

// isRotate90 checks rotation flag in current context (local to processing level)
func isRotate90(ctx context.Context) bool {
	if r, ok := ctx.Value(contextRotateKey{}).(*contextRotate); ok {
		return r.Rotate90
	}
	return false
}

// setRotateArbitrary sets flag indicating a non-orthogonal rotation was applied.
// This signals to downstream filters (e.g. fill) that the image dimensions have
// changed unpredictably and they should use the current image bounds rather than
// the original target dimensions.
func setRotateArbitrary(ctx context.Context) {
	if r, ok := ctx.Value(contextRotateKey{}).(*contextRotate); ok {
		r.RotateArbitrary = true
	}
}

// isRotateArbitrary checks if an arbitrary-angle rotation was applied.
func isRotateArbitrary(ctx context.Context) bool {
	if r, ok := ctx.Value(contextRotateKey{}).(*contextRotate); ok {
		return r.RotateArbitrary
	}
	return false
}

// withContext creates processing context with both global resource tracking and local rotation state
// - Preserves parent's resource context (if exists) for global cleanup tracking
// - Always creates fresh rotation context (local to this processing level)
func withContext(ctx context.Context) context.Context {
	// Check if resource context already exists (from parent)
	if _, ok := ctx.Value(contextResourceKey{}).(*contextResource); !ok {
		// No parent resource context, create new one
		ctx = context.WithValue(ctx, contextResourceKey{}, &contextResource{})
	}
	// Always create fresh rotation context (local to this level)
	// This prevents parent's rotation from affecting nested image() processing
	ctx = context.WithValue(ctx, contextRotateKey{}, &contextRotate{})
	return ctx
}

// Report context for a path a filter loads (fresh for each filter).
type contextReportsKey struct{}

type contextReports struct {
	child *filterReports
}

// withReportsHolder gives the next handler somewhere to leave the report for a
// path it loaded, so the dispatch loop can nest it under that filter's entry.
// The holder is shared by pointer: the handler writes into it and the loop reads
// it once the handler returns.
func withReportsHolder(ctx context.Context) (context.Context, *contextReports) {
	h := &contextReports{}
	return context.WithValue(ctx, contextReportsKey{}, h), h
}

// setChildReports records what the filters in a loaded path did.
func setChildReports(ctx context.Context, r *filterReports) {
	if h, ok := ctx.Value(contextReportsKey{}).(*contextReports); ok {
		h.child = r
	}
}
