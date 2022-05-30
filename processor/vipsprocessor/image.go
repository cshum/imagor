package vipsprocessor

import (
	"github.com/cshum/imagor"
	"github.com/davidbyttow/govips/v2/vips"
	"math"
)

func (v *VipsProcessor) newThumbnail(
	blob *imagor.Bytes, width, height int, crop vips.Interesting, size vips.Size, n int,
) (*vips.ImageRef, error) {
	if blob == nil || blob.IsEmpty() {
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
		if n < -1 {
			params.NumPages.Set(-n)
		} else {
			params.NumPages.Set(-1)
		}
		if crop == vips.InterestingNone || size == vips.SizeForce {
			if img, err = vips.LoadThumbnailFromBuffer(buf, width, height, crop, size, params); err != nil {
				return nil, wrapErr(err)
			}
			if n > 1 && img.Pages() > n {
				// reload image to restrict frames loaded
				img.Close()
				return v.newThumbnail(blob, width, height, crop, size, -n)
			}
		} else {
			if img, err = vips.LoadImageFromBuffer(buf, params); err != nil {
				return nil, wrapErr(err)
			}
			if n > 1 && img.Pages() > n {
				// reload image to restrict frames loaded
				img.Close()
				return v.newThumbnail(blob, width, height, crop, size, -n)
			}
			if err = v.animatedThumbnailWithCrop(img, width, height, crop, size); err != nil {
				img.Close()
				return nil, wrapErr(err)
			}
		}
	} else if blob.BytesType() == imagor.BytesTypePNG {
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

func (v *VipsProcessor) newImage(blob *imagor.Bytes, n int) (*vips.ImageRef, error) {
	if blob == nil || blob.IsEmpty() {
		return nil, imagor.ErrNotFound
	}
	buf, err := blob.ReadAll()
	if err != nil {
		return nil, err
	}
	var params *vips.ImportParams
	if isAnimated(blob, n) {
		params = vips.NewImportParams()
		if n < -1 {
			params.NumPages.Set(-n)
		} else {
			params.NumPages.Set(-1)
		}
		img, err := vips.LoadImageFromBuffer(buf, params)
		if err != nil {
			return nil, wrapErr(err)
		}
		// reload image to restrict frames loaded
		if n > 1 && img.Pages() > n {
			img.Close()
			return v.newImage(blob, -n)
		} else {
			return img, nil
		}
	} else {
		img, err := vips.LoadImageFromBuffer(buf, params)
		if err != nil {
			return nil, wrapErr(err)
		}
		return img, nil
	}
}

func (v *VipsProcessor) thumbnail(
	img *vips.ImageRef, width, height int, crop vips.Interesting, size vips.Size,
) error {
	if crop == vips.InterestingNone || size == vips.SizeForce || img.Height() == img.PageHeight() {
		return img.ThumbnailWithSize(width, height, crop, size)
	}
	return v.animatedThumbnailWithCrop(img, width, height, crop, size)
}

func (v *VipsProcessor) focalThumbnail(img *vips.ImageRef, w, h int, fx, fy float64) (err error) {
	if float64(w)/float64(h) > float64(img.Width())/float64(img.PageHeight()) {
		if err = img.Thumbnail(w, v.MaxHeight, vips.InterestingNone); err != nil {
			return
		}
	} else {
		if err = img.Thumbnail(v.MaxWidth, h, vips.InterestingNone); err != nil {
			return
		}
	}
	var top, left float64
	left = float64(img.Width())*fx - float64(w)/2
	top = float64(img.PageHeight())*fy - float64(h)/2
	left = math.Max(0, math.Min(left, float64(img.Width()-w)))
	top = math.Max(0, math.Min(top, float64(img.PageHeight()-h)))
	return img.ExtractArea(int(left), int(top), w, h)
}

func (v *VipsProcessor) animatedThumbnailWithCrop(
	img *vips.ImageRef, w, h int, crop vips.Interesting, size vips.Size,
) (err error) {
	if size == vips.SizeDown && img.Width() < w && img.PageHeight() < h {
		return
	}
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

func isAnimated(blob *imagor.Bytes, n int) bool {
	return blob != nil && blob.SupportsAnimation() && n != 1 && n != 0
}
