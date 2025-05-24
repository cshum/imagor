package vips

import "C"
import (
	"github.com/cshum/vipsgen/pointer"
	"io"
	"reflect"
	"unsafe"
)

//export goLoggingHandler
func goLoggingHandler(domain *C.char, level C.int, message *C.char) {
	log(C.GoString(domain), LogLevel(level), C.GoString(message))
}

//export goSourceRead
func goSourceRead(
	ptr unsafe.Pointer, buffer unsafe.Pointer, size C.longlong,
) C.longlong {
	src, ok := pointer.Restore(ptr).(*Source)
	if !ok {
		return -1
	}
	sh := &reflect.SliceHeader{
		Data: uintptr(buffer),
		Len:  int(size),
		Cap:  int(size),
	}
	buf := *(*[]byte)(unsafe.Pointer(sh))
	n, err := src.reader.Read(buf)
	if err == io.EOF {
		return C.longlong(n)
	} else if err != nil {
		return -1
	}
	return C.longlong(n)
}

//export goSourceSeek
func goSourceSeek(
	ptr unsafe.Pointer, offset C.longlong, whence int,
) C.longlong {
	src, ok := pointer.Restore(ptr).(*Source)
	if ok && src.seeker != nil {
		switch whence {
		case io.SeekStart, io.SeekCurrent, io.SeekEnd:
			if n, err := src.seeker.Seek(int64(offset), whence); err == nil {
				return C.longlong(n)
			}
		}
	}
	return -1
}
