package vipsprocessor

import "github.com/cshum/vipsgen/v2/vips"

// rotateGainmap rotates the gain map of an UltraHDR image along with its base
// image. libvips only does this for thumbnail, so geometry applied outside it
// leaves the original gain map in place. Rotation keeps the scale factor, so
// the gain map only has to be rotated by the same angle.
func rotateGainmap(img *vips.Image, angle float64) error {
	if !img.HasField("gainmap-data") {
		return nil
	}
	data, err := img.GetBlob("gainmap-data")
	if err != nil || len(data) == 0 {
		return err
	}
	gainmap, err := vips.NewImageFromBuffer(data, nil)
	if err != nil {
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
	buf, err := gainmap.JpegsaveBuffer(&vips.JpegsaveBufferOptions{Q: 95})
	if err != nil {
		return err
	}
	// uhdrsave prefers a processed "gainmap" image over the blob, so drop it.
	img.RemoveField("gainmap")
	img.SetBlob("gainmap-data", buf)
	return nil
}
