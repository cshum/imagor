package vipsprocessor

import (
	"context"
)

type imageRefKey struct{}

type imageRefs struct {
	PageN int
}

func withInitImageRefs(ctx context.Context) context.Context {
	return context.WithValue(ctx, imageRefKey{}, &imageRefs{})
}

func setPageN(ctx context.Context, n int) {
	if r, ok := ctx.Value(imageRefKey{}).(*imageRefs); ok {
		r.PageN = n
	}
}

func getPageN(ctx context.Context) int {
	if r, ok := ctx.Value(imageRefKey{}).(*imageRefs); ok {
		return r.PageN
	}
	return 1
}

func isAnimated(ctx context.Context) bool {
	return getPageN(ctx) > 1
}
