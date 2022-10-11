package seekstream

import (
	"errors"
	"io"
	"os"
)

type Buffer interface {
	io.ReadWriteSeeker
	Cleanup()
}

type TempFileBuffer struct {
	*os.File
}

func (b *TempFileBuffer) Cleanup() {
	filename := b.File.Name()
	_ = b.File.Close()
	_ = os.Remove(filename)
}

func NewTempFileBuffer(dir, pattern string) (*TempFileBuffer, error) {
	file, err := os.CreateTemp(dir, pattern)
	if err != nil {
		return nil, err
	}
	return &TempFileBuffer{file}, nil
}

type MemoryBuffer struct {
	buf []byte
	i   int64 // current reading index
	s   int64 // size
}

func NewMemoryBuffer(size int64) *MemoryBuffer {
	return &MemoryBuffer{buf: make([]byte, size)}
}

func (r *MemoryBuffer) Read(b []byte) (n int, err error) {
	if r.i >= r.s {
		return 0, io.EOF
	}
	rs := r.buf[:r.s]
	n = copy(b, rs[r.i:])
	r.i += int64(n)
	return
}

func (r *MemoryBuffer) Write(p []byte) (n int, err error) {
	n = copy(r.buf[r.i:], p)
	r.s += int64(n)
	r.i += int64(n)
	return n, nil
}

func (r *MemoryBuffer) Seek(offset int64, whence int) (int64, error) {
	var abs int64
	switch whence {
	case io.SeekStart:
		abs = offset
	case io.SeekCurrent:
		abs = r.i + offset
	case io.SeekEnd:
		abs = r.s + offset
	}
	if abs < 0 {
		return 0, errors.New("invalid argument")
	}
	r.i = abs
	return abs, nil
}

func (r *MemoryBuffer) Cleanup() {
	r.buf = nil
}
