package vips

// #include "vips.h"
import "C"
import (
	"runtime"
	"unsafe"
)

// https://www.libvips.org/API/current/VipsImage.html#vips-image-new-from-file
func vipsImageFromFile(filename string, params *ImportParams) (*C.VipsImage, ImageType, error) {
	var out *C.VipsImage
	filenameOption := filename
	if params != nil {
		filenameOption += "[" + params.OptionString() + "]"
	}
	cFileName := C.CString(filenameOption)
	defer freeCString(cFileName)

	if code := C.image_new_from_file(cFileName, &out); code != 0 {
		return nil, ImageTypeUnknown, handleImageError(out)
	}

	imageType := vipsDetermineImageTypeFromMetaLoader(out)
	return out, imageType, nil
}

func vipsImageFromBuffer(buf []byte, params *ImportParams) (*C.VipsImage, ImageType, error) {
	src := buf
	// Reference src here so it's not garbage collected during image initialization.
	defer runtime.KeepAlive(src)

	var out *C.VipsImage
	var code C.int
	var optionString string
	if params != nil {
		optionString = params.OptionString()
	}
	if optionString == "" {
		code = C.image_new_from_buffer(unsafe.Pointer(&src[0]), C.size_t(len(src)), &out)
	} else {
		cOptionString := C.CString(optionString)
		defer freeCString(cOptionString)

		code = C.image_new_from_buffer_with_option(unsafe.Pointer(&src[0]), C.size_t(len(src)), &out, cOptionString)
	}
	if code != 0 {
		return nil, ImageTypeUnknown, handleImageError(out)
	}

	imageType := vipsDetermineImageTypeFromMetaLoader(out)
	return out, imageType, nil
}

// https://www.libvips.org/API/current/libvips-resample.html#vips-thumbnail
func vipsThumbnailFromFile(filename string, width, height int, crop Interesting, size Size, params *ImportParams) (*C.VipsImage, ImageType, error) {
	var out *C.VipsImage

	filenameOption := filename
	if params != nil {
		filenameOption += "[" + params.OptionString() + "]"
	}

	cFileName := C.CString(filenameOption)
	defer freeCString(cFileName)

	if code := C.thumbnail(cFileName, &out, C.int(width), C.int(height), C.int(crop), C.int(size)); code != 0 {
		return nil, ImageTypeUnknown, handleImageError(out)
	}

	imageType := vipsDetermineImageTypeFromMetaLoader(out)
	return out, imageType, nil
}

func clearImage(ref *C.VipsImage) {
	C.clear_image(&ref)
}

func vipsHasAlpha(in *C.VipsImage) bool {
	return int(C.has_alpha_channel(in)) > 0
}

// https://libvips.github.io/libvips/API/current/libvips-conversion.html#vips-copy
func vipsCopyImage(in *C.VipsImage) (*C.VipsImage, error) {
	var out *C.VipsImage

	if err := C.copy_image(in, &out); int(err) != 0 {
		return nil, handleImageError(out)
	}

	return out, nil
}

func vipsThumbnail(in *C.VipsImage, width, height int, crop Interesting, size Size) (*C.VipsImage, error) {
	var out *C.VipsImage

	if err := C.thumbnail_image(in, &out, C.int(width), C.int(height), C.int(crop), C.int(size)); err != 0 {
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

//  https://libvips.github.io/libvips/API/current/libvips-arithmetic.html#vips-linear
func vipsLinear(in *C.VipsImage, a, b []float64, n int) (*C.VipsImage, error) {
	var out *C.VipsImage

	if err := C.linear(in, &out, (*C.double)(&a[0]), (*C.double)(&b[0]), C.int(n)); err != 0 {
		return nil, handleImageError(out)
	}

	return out, nil
}

// https://libvips.github.io/libvips/API/current/libvips-arithmetic.html#vips-find-trim
func vipsFindTrim(in *C.VipsImage, threshold float64, backgroundColor *Color) (int, int, int, int, error) {
	var left, top, width, height C.int

	if err := C.find_trim(in, &left, &top, &width, &height, C.double(threshold), C.double(backgroundColor.R),
		C.double(backgroundColor.G), C.double(backgroundColor.B)); err != 0 {
		return -1, -1, -1, -1, handleVipsError()
	}

	return int(left), int(top), int(width), int(height), nil
}

// https://libvips.github.io/libvips/API/current/libvips-arithmetic.html#vips-getpoint
func vipsGetPoint(in *C.VipsImage, n int, x int, y int) ([]float64, error) {
	var out *C.double
	defer gFreePointer(unsafe.Pointer(out))

	if err := C.getpoint(in, &out, C.int(n), C.int(x), C.int(y)); err != 0 {
		return nil, handleVipsError()
	}

	// maximum n is 4
	return (*[4]float64)(unsafe.Pointer(out))[:n:n], nil
}

// https://libvips.github.io/libvips/API/current/libvips-colour.html#vips-colourspace
func vipsToColorSpace(in *C.VipsImage, interpretation Interpretation) (*C.VipsImage, error) {
	var out *C.VipsImage

	inter := C.VipsInterpretation(interpretation)

	if err := C.to_colorspace(in, &out, inter); err != 0 {
		return nil, handleImageError(out)
	}

	return out, nil
}

// https://libvips.github.io/libvips/API/current/libvips-convolution.html#vips-gaussblur
func vipsGaussianBlur(in *C.VipsImage, sigma float64) (*C.VipsImage, error) {
	var out *C.VipsImage

	if err := C.gaussian_blur_image(in, &out, C.double(sigma)); err != 0 {
		return nil, handleImageError(out)
	}

	return out, nil
}

// https://libvips.github.io/libvips/API/current/libvips-convolution.html#vips-sharpen
func vipsSharpen(in *C.VipsImage, sigma float64, x1 float64, m2 float64) (*C.VipsImage, error) {
	var out *C.VipsImage

	if err := C.sharpen_image(in, &out, C.double(sigma), C.double(x1), C.double(m2)); err != 0 {
		return nil, handleImageError(out)
	}

	return out, nil
}

func vipsRemoveICCProfile(in *C.VipsImage) bool {
	return fromGboolean(C.remove_icc_profile(in))
}

func vipsGetMetaOrientation(in *C.VipsImage) int {
	return int(C.get_meta_orientation(in))
}

func vipsGetImageNPages(in *C.VipsImage) int {
	return int(C.get_image_n_pages(in))
}

func vipsGetPageHeight(in *C.VipsImage) int {
	return int(C.get_page_height(in))
}

func vipsSetPageHeight(in *C.VipsImage, height int) {
	C.set_page_height(in, C.int(height))
}

func vipsImageGetMetaLoader(in *C.VipsImage) (string, bool) {
	var out *C.char
	defer gFreePointer(unsafe.Pointer(out))
	code := int(C.get_meta_loader(in, &out))
	return C.GoString(out), code == 0
}

func vipsImageSetDelay(in *C.VipsImage, data []C.int) error {
	if n := len(data); n > 0 {
		C.set_image_delay(in, &data[0], C.int(n))
	}
	return nil
}

func vipsGetString(image *C.VipsImage, name string) string {
	return C.GoString(C.get_meta_string(image, cachedCString(name)))
}
