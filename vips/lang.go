package vips

// #include <vips/vips.h>
import "C"
import (
	"fmt"
	"sync"
	"unsafe"
)

func handleImageError(out *C.VipsImage) error {
	if out != nil {
		clearImage(out)
	}
	return handleVipsError()
}

func handleVipsError() error {
	s := C.GoString(C.vips_error_buffer())
	C.vips_error_clear()

	return fmt.Errorf("%v", s)
}

func freeCString(s *C.char) {
	C.free(unsafe.Pointer(s))
}

func gFreePointer(ref unsafe.Pointer) {
	C.g_free(C.gpointer(ref))
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func boolToStr(v bool) string {
	if v {
		return "TRUE"
	}
	return "FALSE"
}

func toGboolean(b bool) C.gboolean {
	if b {
		return C.gboolean(1)
	}
	return C.gboolean(0)
}

func fromGboolean(b C.gboolean) bool {
	return b != 0
}

var cStringsCache sync.Map

func cachedCString(str string) *C.char {
	if cstr, ok := cStringsCache.Load(str); ok {
		return cstr.(*C.char)
	}
	cstr := C.CString(str)
	cStringsCache.Store(str, cstr)
	return cstr
}
