package vipsprocessor

import (
	"context"
	"github.com/davidbyttow/govips/v2/vips"
)

type imageRefKey struct{}

type imageRefs struct {
	imageRefs []*vips.ImageRef
}

func (r *imageRefs) Add(imageRef *vips.ImageRef) {
	r.imageRefs = append(r.imageRefs, imageRef)
}

func (r *imageRefs) Close() {
	for _, img := range r.imageRefs {
		img.Close()
	}
	r.imageRefs = nil
}

// WithInitImageRefs context with image ref tracking
func WithInitImageRefs(ctx context.Context) context.Context {
	return context.WithValue(ctx, imageRefKey{}, &imageRefs{})
}

// AddImageRef context add vips image ref for keeping track of gc
func AddImageRef(ctx context.Context, img *vips.ImageRef) {
	if r, ok := ctx.Value(imageRefKey{}).(*imageRefs); ok {
		r.Add(img)
	}
}

// CloseImageRefs closes all image refs that are being tracked through the context
func CloseImageRefs(ctx context.Context) {
	if r, ok := ctx.Value(imageRefKey{}).(*imageRefs); ok {
		r.Close()
	}
}
