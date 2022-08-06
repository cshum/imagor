package vipsprocessor

// #include "color.h"
import "C"

// https://libvips.github.io/libvips/API/current/libvips-colour.html#vips-colourspace
func vipsToColorSpace(in *C.VipsImage, interpretation Interpretation) (*C.VipsImage, error) {
	var out *C.VipsImage

	inter := C.VipsInterpretation(interpretation)

	if err := C.to_colorspace(in, &out, inter); err != 0 {
		return nil, handleImageError(out)
	}

	return out, nil
}
