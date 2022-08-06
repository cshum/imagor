package vipsprocessor

// #include "header.h"
import "C"
import (
	"strings"
	"unsafe"
)

func vipsRemoveICCProfile(in *C.VipsImage) bool {
	return fromGboolean(C.remove_icc_profile(in))
}

func vipsGetMetaOrientation(in *C.VipsImage) int {
	return int(C.get_meta_orientation(in))
}

func vipsGetImageNPages(in *C.VipsImage) int {
	return int(C.get_image_n_pages(in))
}

func vipsSetImageNPages(in *C.VipsImage, pages int) {
	C.set_image_n_pages(in, C.int(pages))
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

// vipsDetermineImageTypeFromMetaLoader determine the image type from vips-loader metadata
func vipsDetermineImageTypeFromMetaLoader(in *C.VipsImage) ImageType {
	if in == nil {
		return ImageTypeUnknown
	}
	vipsLoader, ok := vipsImageGetMetaLoader(in)
	if vipsLoader == "" || !ok {
		return ImageTypeUnknown
	}
	if strings.HasPrefix(vipsLoader, "jpeg") {
		return ImageTypeJPEG
	}
	if strings.HasPrefix(vipsLoader, "png") {
		return ImageTypePNG
	}
	if strings.HasPrefix(vipsLoader, "gif") {
		return ImageTypeGIF
	}
	if strings.HasPrefix(vipsLoader, "svg") {
		return ImageTypeSVG
	}
	if strings.HasPrefix(vipsLoader, "webp") {
		return ImageTypeWEBP
	}
	if strings.HasPrefix(vipsLoader, "jp2k") {
		return ImageTypeJP2K
	}
	if strings.HasPrefix(vipsLoader, "magick") {
		return ImageTypeMagick
	}
	if strings.HasPrefix(vipsLoader, "tiff") {
		return ImageTypeTIFF
	}
	if strings.HasPrefix(vipsLoader, "heif") {
		return ImageTypeHEIF
	}
	if strings.HasPrefix(vipsLoader, "pdf") {
		return ImageTypePDF
	}
	return ImageTypeUnknown
}
