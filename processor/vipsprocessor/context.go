package vipsprocessor

import (
	"context"
)

type imageRefKey struct{}

type imageRefs struct {
	PageN int
}

// withInitImageRefs context with image ref tracking
func withInitImageRefs(ctx context.Context) context.Context {
	return context.WithValue(ctx, imageRefKey{}, &imageRefs{})
}

func SetPageN(ctx context.Context, n int) {
	if r, ok := ctx.Value(imageRefKey{}).(*imageRefs); ok {
		r.PageN = n
	}
}

func GetPageN(ctx context.Context) int {
	if r, ok := ctx.Value(imageRefKey{}).(*imageRefs); ok {
		return r.PageN
	}
	return 1
}

func IsAnimated(ctx context.Context) bool {
	return GetPageN(ctx) > 1
}
