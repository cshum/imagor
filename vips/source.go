package vips

// #include "source.h"
import "C"
import (
	"fmt"
	"github.com/cshum/imagor/vips/pointer"
	"io"
	"sync"
	"unsafe"
)

type Source struct {
	reader io.ReadCloser
	seeker io.Seeker
	src    *C.VipsSourceCustom
	ptr    unsafe.Pointer
	lock   sync.Mutex
}

func NewSource(reader io.ReadCloser) *Source {
	startupIfNeeded()

	s := &Source{reader: reader}
	seeker, ok := reader.(io.ReadSeeker)
	if ok {
		s.seeker = seeker
		s.ptr = pointer.Save(s)
		s.src = C.create_go_custom_source_with_seek(s.ptr)
	} else {
		s.ptr = pointer.Save(s)
		s.src = C.create_go_custom_source(s.ptr)
	}
	return s
}

func (s *Source) Close() {
	s.lock.Lock()
	if s.ptr != nil {
		C.clear_source(&s.src)
		pointer.Unref(s.ptr)
		s.ptr = nil
		_ = s.reader.Close()
		log("vips", LogLevelDebug, fmt.Sprintf("closing source %p", s))
	}
	s.lock.Unlock()
}

func (s *Source) LoadImage(params *ImportParams) (*Image, error) {
	return LoadImageFromSource(s, params)
}

func (s *Source) LoadThumbnail(width, height int, crop Interesting, size Size, params *ImportParams) (*Image, error) {
	return LoadThumbnailFromSource(s, width, height, crop, size, params)
}
