package vipsprocessor

// #include <vips/vips.h>
import "C"

import (
	"fmt"
	"unsafe"
)

func handleImageError(out *C.VipsImage) error {
	if out != nil {
		clearImage(out)
	}

	return handleVipsError()
}

func handleSaveBufferError(out unsafe.Pointer) error {
	if out != nil {
		gFreePointer(out)
	}

	return handleVipsError()
}

func handleVipsError() error {
	s := C.GoString(C.vips_error_buffer())
	C.vips_error_clear()

	return fmt.Errorf("%v", s)
}
