package vipsprocessor

// #include "arithmetic.h"
import "C"
import "unsafe"

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
