package vipsprocessor

import (
	"math"

	"github.com/cshum/vipsgen/v2/vips"
)

// libvips updates a gain map only in thumbnail; these mirror the rest.

func gainmapOf(img *vips.Image) (*vips.Image, error) {
	// Only UltraHDR images carry a gain map.
	if img.Format() != vips.ImageTypeUhdr {
		return nil, nil
	}
	gainmap, ok := img.Gainmap()
	if !ok {
		return nil, nil
	}
	return gainmap, nil
}

// rotateGainmap rotates the gain map with the image.
func rotateGainmap(img *vips.Image, angle float64) error {
	gainmap, err := gainmapOf(img)
	if err != nil || gainmap == nil {
		return err
	}
	defer gainmap.Close()
	if angle == float64(int(angle)) && int(angle)%90 == 0 {
		if err = gainmap.RotMultiPage(getAngle(int(angle))); err != nil {
			return err
		}
	} else if err = gainmap.Rotate(angle, &vips.RotateOptions{}); err != nil {
		return err
	}
	img.SetGainmap(gainmap)
	return nil
}

// flipGainmap flips the gain map with the image.
func flipGainmap(img *vips.Image, direction vips.Direction) error {
	gainmap, err := gainmapOf(img)
	if err != nil || gainmap == nil {
		return err
	}
	defer gainmap.Close()
	if err = gainmap.Flip(direction); err != nil {
		return err
	}
	img.SetGainmap(gainmap)
	return nil
}

// cropGainmap crops the gain map to the region the image keeps. Call it
// before the crop.
func cropGainmap(img *vips.Image, left, top, width, height int) error {
	gainmap, err := gainmapOf(img)
	if err != nil || gainmap == nil {
		return err
	}
	defer gainmap.Close()
	if img.Width() == 0 || img.PageHeight() == 0 {
		return nil
	}
	scaleX := float64(gainmap.Width()) / float64(img.Width())
	scaleY := float64(gainmap.PageHeight()) / float64(img.PageHeight())
	x := int(math.Round(float64(left) * scaleX))
	y := int(math.Round(float64(top) * scaleY))
	w := int(math.Round(float64(width) * scaleX))
	h := int(math.Round(float64(height) * scaleY))
	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}
	if w > gainmap.Width()-x {
		w = gainmap.Width() - x
	}
	if h > gainmap.PageHeight()-y {
		h = gainmap.PageHeight() - y
	}
	if w <= 0 || h <= 0 {
		return nil
	}
	if err = gainmap.ExtractArea(x, y, w, h); err != nil {
		return err
	}
	img.SetGainmap(gainmap)
	return nil
}

// attachGainmap attaches the gain map so thumbnail updates it on resize.
// Call it before the resize.
func attachGainmap(img *vips.Image) error {
	gainmap, err := gainmapOf(img)
	if err != nil || gainmap == nil {
		return err
	}
	defer gainmap.Close()
	img.SetGainmap(gainmap)
	return nil
}
