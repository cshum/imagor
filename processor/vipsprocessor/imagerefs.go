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

func withInitImageRefs(ctx context.Context) context.Context {
	return context.WithValue(ctx, imageRefKey{}, &imageRefs{})
}

func addImageRef(ctx context.Context, img *vips.ImageRef) {
	if r, ok := ctx.Value(imageRefKey{}).(*imageRefs); ok {
		r.Add(img)
	}
}

func closeImageRefs(ctx context.Context) {
	if r, ok := ctx.Value(imageRefKey{}).(*imageRefs); ok {
		r.Close()
	}
}
