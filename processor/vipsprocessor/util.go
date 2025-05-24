package vipsprocessor

import "github.com/cshum/vipsgen/vips"

// Modulate the colors
func Modulate(r *vips.Image, brightness, saturation, hue float64) error {
	var err error
	var multiplications []float64
	var additions []float64

	colorspace := r.Interpretation()
	if colorspace == vips.InterpretationRgb {
		colorspace = vips.InterpretationSrgb
	}

	multiplications = []float64{brightness, saturation, 1}
	additions = []float64{0, 0, hue}

	if r.HasAlpha() {
		multiplications = append(multiplications, 1)
		additions = append(additions, 0)
	}

	err = r.Colourspace(vips.InterpretationLch, nil)
	if err != nil {
		return err
	}

	err = r.Linear(multiplications, additions, nil)
	if err != nil {
		return err
	}

	err = r.Colourspace(colorspace, nil)
	if err != nil {
		return err
	}

	return nil
}
