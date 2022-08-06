package vipsprocessor

// #include "conversion.h"
import "C"

// https://libvips.github.io/libvips/API/current/libvips-conversion.html#vips-copy
func vipsCopyImage(in *C.VipsImage) (*C.VipsImage, error) {
	var out *C.VipsImage

	if err := C.copy_image(in, &out); int(err) != 0 {
		return nil, handleImageError(out)
	}

	return out, nil
}

// https://libvips.github.io/libvips/API/current/libvips-conversion.html#vips-embed
func vipsEmbed(in *C.VipsImage, left, top, width, height int, extend ExtendStrategy) (*C.VipsImage, error) {
	var out *C.VipsImage

	if err := C.embed_image(in, &out, C.int(left), C.int(top), C.int(width), C.int(height), C.int(extend)); err != 0 {
		return nil, handleImageError(out)
	}

	return out, nil
}

// https://libvips.github.io/libvips/API/current/libvips-conversion.html#vips-embed
func vipsEmbedBackground(in *C.VipsImage, left, top, width, height int, backgroundColor *ColorRGBA) (*C.VipsImage, error) {
	var out *C.VipsImage

	if err := C.embed_image_background(in, &out, C.int(left), C.int(top), C.int(width),
		C.int(height), C.double(backgroundColor.R),
		C.double(backgroundColor.G), C.double(backgroundColor.B), C.double(backgroundColor.A)); err != 0 {
		return nil, handleImageError(out)
	}

	return out, nil
}

func vipsEmbedMultiPage(in *C.VipsImage, left, top, width, height int, extend ExtendStrategy) (*C.VipsImage, error) {
	var out *C.VipsImage

	if err := C.embed_multi_page_image(in, &out, C.int(left), C.int(top), C.int(width), C.int(height), C.int(extend)); err != 0 {
		return nil, handleImageError(out)
	}

	return out, nil
}

func vipsEmbedMultiPageBackground(in *C.VipsImage, left, top, width, height int, backgroundColor *ColorRGBA) (*C.VipsImage, error) {
	var out *C.VipsImage

	if err := C.embed_multi_page_image_background(in, &out, C.int(left), C.int(top), C.int(width),
		C.int(height), C.double(backgroundColor.R),
		C.double(backgroundColor.G), C.double(backgroundColor.B), C.double(backgroundColor.A)); err != 0 {
		return nil, handleImageError(out)
	}

	return out, nil
}

// https://libvips.github.io/libvips/API/current/libvips-conversion.html#vips-flip
func vipsFlip(in *C.VipsImage, direction Direction) (*C.VipsImage, error) {
	var out *C.VipsImage

	if err := C.flip_image(in, &out, C.int(direction)); err != 0 {
		return nil, handleImageError(out)
	}

	return out, nil
}

// https://libvips.github.io/libvips/API/current/libvips-conversion.html#vips-extract-area
func vipsExtractArea(in *C.VipsImage, left, top, width, height int) (*C.VipsImage, error) {
	var out *C.VipsImage

	if err := C.extract_image_area(in, &out, C.int(left), C.int(top), C.int(width), C.int(height)); err != 0 {
		return nil, handleImageError(out)
	}

	return out, nil
}

func vipsExtractAreaMultiPage(in *C.VipsImage, left, top, width, height int) (*C.VipsImage, error) {
	var out *C.VipsImage

	if err := C.extract_area_multi_page(in, &out, C.int(left), C.int(top), C.int(width), C.int(height)); err != 0 {
		return nil, handleImageError(out)
	}

	return out, nil
}

// https://libvips.github.io/libvips/API/current/libvips-conversion.html#vips-rot
func vipsRotate(in *C.VipsImage, angle Angle) (*C.VipsImage, error) {
	var out *C.VipsImage

	if err := C.rotate_image(in, &out, C.VipsAngle(angle)); err != 0 {
		return nil, handleImageError(out)
	}

	return out, nil
}

// https://libvips.github.io/libvips/API/current/libvips-conversion.html#vips-rot
func vipsRotateMultiPage(in *C.VipsImage, angle Angle) (*C.VipsImage, error) {
	var out *C.VipsImage

	if err := C.rotate_image_multi_page(in, &out, C.VipsAngle(angle)); err != 0 {
		return nil, handleImageError(out)
	}

	return out, nil
}

// https://libvips.github.io/libvips/API/current/libvips-conversion.html#vips-flatten
func vipsFlatten(in *C.VipsImage, color *Color) (*C.VipsImage, error) {
	var out *C.VipsImage

	err := C.flatten_image(in, &out, C.double(color.R), C.double(color.G), C.double(color.B))
	if int(err) != 0 {
		return nil, handleImageError(out)
	}

	return out, nil
}

func vipsAddAlpha(in *C.VipsImage) (*C.VipsImage, error) {
	var out *C.VipsImage

	if err := C.add_alpha(in, &out); err != 0 {
		return nil, handleImageError(out)
	}

	return out, nil
}

func vipsCast(in *C.VipsImage, bandFormat BandFormat) (*C.VipsImage, error) {
	var out *C.VipsImage

	if err := C.cast(in, &out, C.int(bandFormat)); err != 0 {
		return nil, handleImageError(out)
	}

	return out, nil
}

// https://libvips.github.io/libvips/API/current/libvips-conversion.html#vips-composite2
func vipsComposite2(base *C.VipsImage, overlay *C.VipsImage, mode BlendMode, x, y int) (*C.VipsImage, error) {
	var out *C.VipsImage

	if err := C.composite2_image(base, overlay, &out, C.int(mode), C.gint(x), C.gint(y)); err != 0 {
		return nil, handleImageError(out)
	}

	return out, nil
}

// https://www.libvips.org/API/current/libvips-conversion.html#vips-replicate
func vipsReplicate(in *C.VipsImage, across int, down int) (*C.VipsImage, error) {
	var out *C.VipsImage

	if err := C.replicate(in, &out, C.int(across), C.int(down)); err != 0 {
		return nil, handleImageError(out)
	}
	return out, nil
}
