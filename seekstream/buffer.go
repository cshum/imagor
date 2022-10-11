package seekstream

import (
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
