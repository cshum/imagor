package vipsprocessor

// #include "source.h"
import "C"
import (
	"fmt"
	"github.com/cshum/imagor/processor/vipsprocessor/pointer"
	"io"
	"runtime"
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
	}
	s.ptr = pointer.Save(s)
	s.src = C.create_go_custom_source(s.ptr)

	runtime.SetFinalizer(s, finalizeSource)
	return s
}

func (s *Source) LoadImage(params *ImportParams) (*ImageRef, error) {
	if params == nil {
		params = NewImportParams()
	}

	vipsImage, format, err := vipsImageFromSource(s.src, params)
	if err != nil {
		return nil, err
	}

	ref := newImageRef(vipsImage, format, nil)
	log("govips", LogLevelDebug, fmt.Sprintf("created imageRef %p", ref))
	return ref, nil
}

func (s *Source) LoadThumbnail(width, height int, crop Interesting, size Size, params *ImportParams) (*ImageRef, error) {
	if params == nil {
		params = NewImportParams()
	}

	vipsImage, format, err := vipsThumbnailFromSource(
		s.src, width, height, crop, size, params)
	if err != nil {
		return nil, err
	}

	ref := newImageRef(vipsImage, format, nil)
	log("govips", LogLevelDebug, fmt.Sprintf("created imageRef %p", ref))
	return ref, nil
}

func finalizeSource(src *Source) {
	log("govips", LogLevelDebug, fmt.Sprintf("closing source %p", src))
	src.Close()
}

func (s *Source) Close() {
	s.lock.Lock()
	if s.ptr != nil {
		C.clear_source(&s.src)
		pointer.Unref(s.ptr)
		s.ptr = nil
		_ = s.reader.Close()
	}
	s.lock.Unlock()
}

// https://www.libvips.org/API/current/VipsImage.html#vips-image-new-from-source
func vipsImageFromSource(
	src *C.VipsSourceCustom, params *ImportParams,
) (*C.VipsImage, ImageType, error) {
	var out *C.VipsImage
	var code C.int
	var optionString string

	if params != nil {
		optionString = params.OptionString()
	}
	if optionString == "" {
		code = C.image_new_from_source(src, &out)
	} else {
		cOptionString := C.CString(optionString)
		defer freeCString(cOptionString)

		code = C.image_new_from_source_with_option(src, &out, cOptionString)
	}
	if code != 0 {
		return nil, ImageTypeUnknown, handleImageError(out)
	}

	imageType := vipsDetermineImageTypeFromMetaLoader(out)
	return out, imageType, nil
}

// https://www.libvips.org/API/current/VipsImage.html#vips-image-new-from-source
func vipsThumbnailFromSource(
	src *C.VipsSourceCustom, width, height int, crop Interesting, size Size, params *ImportParams) (*C.VipsImage, ImageType, error) {
	var out *C.VipsImage
	var code C.int
	var optionString string

	if params != nil {
		optionString = params.OptionString()
	}
	if optionString == "" {
		code = C.thumbnail_source(src, &out, C.int(width), C.int(height), C.int(crop), C.int(size))
	} else {
		cOptionString := C.CString(optionString)
		defer freeCString(cOptionString)

		code = C.thumbnail_source_with_option(src, &out, C.int(width), C.int(height), C.int(crop), C.int(size), cOptionString)
	}
	if code != 0 {
		return nil, ImageTypeUnknown, handleImageError(out)
	}

	imageType := vipsDetermineImageTypeFromMetaLoader(out)
	return out, imageType, nil
}
