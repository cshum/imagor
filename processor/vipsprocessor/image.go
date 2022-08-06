package vipsprocessor

import (
	"github.com/cshum/imagor"
	"math"
)

func (v *VipsProcessor) newThumbnail(
	blob *imagor.Blob, width, height int, crop Interesting, size Size, n int,
) (*ImageRef, error) {
	if blob == nil || blob.IsEmpty() {
		return nil, imagor.ErrNotFound
	}
	buf, err := blob.ReadAll()
	if err != nil {
		return nil, err
	}
	var params *ImportParams
	var img *ImageRef
	if isBlobAnimated(blob, n) {
		params = NewImportParams()
		if n < -1 {
			params.NumPages.Set(-n)
		} else {
			params.NumPages.Set(-1)
		}
		if crop == InterestingNone || size == SizeForce {
			if img, err = v.checkResolution(
				LoadThumbnailFromBuffer(buf, width, height, crop, size, params),
			); err != nil {
				return nil, wrapErr(err)
			}
			if n > 1 && img.Pages() > n {
				// reload image to restrict frames loaded
				img.Close()
				return v.newThumbnail(blob, width, height, crop, size, -n)
			}
		} else {
			if img, err = v.checkResolution(LoadImageFromBuffer(buf, params)); err != nil {
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
	} else if blob.BlobType() == imagor.BlobTypePNG {
		return v.newThumbnailPNG(buf, width, height, crop, size)
	} else {
		img, err = LoadThumbnailFromBuffer(buf, width, height, crop, size, nil)
	}
	return v.checkResolution(img, wrapErr(err))
}

func (v *VipsProcessor) newThumbnailPNG(
	buf []byte, width, height int, crop Interesting, size Size,
) (img *ImageRef, err error) {
	if img, err = v.checkResolution(NewImageFromBuffer(buf)); err != nil {
		return
	}
	if err = img.ThumbnailWithSize(width, height, crop, size); err != nil {
		img.Close()
		return
	}
	return v.checkResolution(img, wrapErr(err))
}

func (v *VipsProcessor) newImage(blob *imagor.Blob, n int) (*ImageRef, error) {
	if blob == nil || blob.IsEmpty() {
		return nil, imagor.ErrNotFound
	}
	buf, err := blob.ReadAll()
	if err != nil {
		return nil, err
	}
	var params *ImportParams
	if isBlobAnimated(blob, n) {
		params = NewImportParams()
		if n < -1 {
			params.NumPages.Set(-n)
		} else {
			params.NumPages.Set(-1)
		}
		img, err := v.checkResolution(LoadImageFromBuffer(buf, params))
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
		img, err := v.checkResolution(LoadImageFromBuffer(buf, params))
		if err != nil {
			return nil, wrapErr(err)
		}
		return img, nil
	}
}

func (v *VipsProcessor) thumbnail(
	img *ImageRef, width, height int, crop Interesting, size Size,
) error {
	if crop == InterestingNone || size == SizeForce || img.Height() == img.PageHeight() {
		return img.ThumbnailWithSize(width, height, crop, size)
	}
	return v.animatedThumbnailWithCrop(img, width, height, crop, size)
}

func (v *VipsProcessor) focalThumbnail(img *ImageRef, w, h int, fx, fy float64) (err error) {
	if float64(w)/float64(h) > float64(img.Width())/float64(img.PageHeight()) {
		if err = img.Thumbnail(w, v.MaxHeight, InterestingNone); err != nil {
			return
		}
	} else {
		if err = img.Thumbnail(v.MaxWidth, h, InterestingNone); err != nil {
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
	img *ImageRef, w, h int, crop Interesting, size Size,
) (err error) {
	if size == SizeDown && img.Width() < w && img.PageHeight() < h {
		return
	}
	var top, left int
	if float64(w)/float64(h) > float64(img.Width())/float64(img.PageHeight()) {
		if err = img.ThumbnailWithSize(w, v.MaxHeight, InterestingNone, size); err != nil {
			return
		}
	} else {
		if err = img.ThumbnailWithSize(v.MaxWidth, h, InterestingNone, size); err != nil {
			return
		}
	}
	if crop == InterestingHigh {
		left = img.Width() - w
		top = img.PageHeight() - h
	} else if crop == InterestingCentre || crop == InterestingAttention {
		left = (img.Width() - w) / 2
		top = (img.PageHeight() - h) / 2
	}
	return img.ExtractArea(left, top, w, h)
}

func (v *VipsProcessor) checkResolution(img *ImageRef, err error) (*ImageRef, error) {
	if err != nil || img == nil {
		return img, err
	}
	if img.Width() > v.MaxWidth || img.PageHeight() > v.MaxHeight ||
		(img.Width()*img.PageHeight()) > v.MaxResolution {
		img.Close()
		return nil, imagor.ErrMaxResolutionExceeded
	}
	return img, nil
}

func isBlobAnimated(blob *imagor.Blob, n int) bool {
	return blob != nil && blob.SupportsAnimation() && n != 1 && n != 0
}
