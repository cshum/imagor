package vips

// #include "target.h"
import "C"
import (
	"fmt"
	"github.com/cshum/imagor"
	"github.com/cshum/imagor/vips/pointer"
	"io"
	"runtime"
	"sync"
	"unsafe"
)

type Target struct {
	writer io.WriteCloser
	target *C.VipsTargetCustom
	ptr    unsafe.Pointer
	once   sync.Once
}

func NewTarget(writer io.WriteCloser) *Target {
	startupIfNeeded()

	t := &Target{writer: writer}
	t.ptr = pointer.Save(t)
	t.target = C.create_go_custom_target(t.ptr)

	runtime.SetFinalizer(t, finalizeTarget)
	return t
}

func finalizeTarget(target *Target) {
	target.Close()
}

func (s *Target) Close() {
	s.once.Do(func() {
		if s.ptr != nil {
			C.clear_target(&s.target)
			pointer.Unref(s.ptr)
			s.ptr = nil
			_ = s.writer.Close()
			log("vips", LogLevelDebug, fmt.Sprintf("closing target %p", s))
		}
	})
}

func NewBlobFromTarget(handler func(*Target) error) *imagor.Blob {
	pr, pw := io.Pipe()
	target := NewTarget(pw)
	go func() {
		defer target.Close()
		if err := handler(target); err != nil {
			_ = pr.CloseWithError(err)
		}
	}()
	return imagor.NewBlobFromReader(pr)
}
