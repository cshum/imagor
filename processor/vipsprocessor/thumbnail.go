package vipsprocessor

import (
	"github.com/cshum/imagor"
	"github.com/davidbyttow/govips/v2/vips"
)

func (v *VipsProcessor) newThumbnail(
	blob *imagor.Blob, width, height int, crop vips.Interesting, size vips.Size, n int,
) (*vips.ImageRef, error) {
	if imagor.IsBlobEmpty(blob) {
		return nil, imagor.ErrNotFound
	}
	buf, err := blob.ReadAll()
	if err != nil {
		return nil, err
	}
	var params *vips.ImportParams
	var img *vips.ImageRef
	if isAnimated(blob, n) {
		params = vips.NewImportParams()
		params.NumPages.Set(n)
		if crop == vips.InterestingNone || size == vips.SizeForce {
			img, err = vips.LoadThumbnailFromBuffer(buf, width, height, crop, size, params)
		} else {
			if img, err = vips.LoadImageFromBuffer(buf, params); err != nil {
				return nil, wrapErr(err)
			}
			if err = v.animatedThumbnailWithCrop(img, width, height, crop, size); err != nil {
				img.Close()
				return img, wrapErr(err)
			}
		}
	} else if blob.IsPNG() {
		// avoid vips pngload error
		return newThumbnailFix(buf, width, height, crop, size)
	} else {
		img, err = vips.LoadThumbnailFromBuffer(buf, width, height, crop, size, nil)
	}
	return img, wrapErr(err)
}

func newThumbnailFix(
	buf []byte, width, height int, crop vips.Interesting, size vips.Size,
) (img *vips.ImageRef, err error) {
	if img, err = vips.NewImageFromBuffer(buf); err != nil {
		return
	}
	if err = img.ThumbnailWithSize(width, height, crop, size); err != nil {
		img.Close()
		return
	}
	err = wrapErr(err)
	return
}

func (v *VipsProcessor) newImage(blob *imagor.Blob, n int) (*vips.ImageRef, error) {
	if imagor.IsBlobEmpty(blob) {
		return nil, imagor.ErrNotFound
	}
	buf, err := blob.ReadAll()
	if err != nil {
		return nil, err
	}
	var params *vips.ImportParams
	if isAnimated(blob, n) {
		params = vips.NewImportParams()
		params.NumPages.Set(n)
	}
	img, err := vips.LoadImageFromBuffer(buf, params)
	return img, wrapErr(err)
}

func (v *VipsProcessor) thumbnail(
	img *vips.ImageRef, width, height int, crop vips.Interesting, size vips.Size,
) error {
	if crop == vips.InterestingNone || size == vips.SizeForce || img.Height() == img.PageHeight() {
		return img.ThumbnailWithSize(width, height, crop, size)
	}
	return v.animatedThumbnailWithCrop(img, width, height, crop, size)
}

func (v *VipsProcessor) animatedThumbnailWithCrop(
	img *vips.ImageRef, w, h int, crop vips.Interesting, size vips.Size,
) (err error) {
	if size == vips.SizeDown && img.Width() < w && img.PageHeight() < h {
		return
	}
	// use ExtractArea for animated cropping
	var top, left int
	if float64(w)/float64(h) > float64(img.Width())/float64(img.PageHeight()) {
		if err = img.ThumbnailWithSize(w, v.MaxHeight, vips.InterestingNone, size); err != nil {
			return
		}
	} else {
		if err = img.ThumbnailWithSize(v.MaxWidth, h, vips.InterestingNone, size); err != nil {
			return
		}
	}
	if crop == vips.InterestingHigh {
		left = img.Width() - w
		top = img.PageHeight() - h
	} else if crop == vips.InterestingCentre || crop == vips.InterestingAttention {
		left = (img.Width() - w) / 2
		top = (img.PageHeight() - h) / 2
	}
	return img.ExtractArea(left, top, w, h)
}

func isAnimated(blob *imagor.Blob, n int) bool {
	return blob != nil && blob.SupportsAnimation() && n != 1 && n != 0
}
