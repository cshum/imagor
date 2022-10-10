package imagor

import (
	"errors"
	"io"
	"os"
)

type SeekStream struct {
	source    io.ReadCloser
	file      *os.File
	reader    io.Reader
	size      int64
	seekReady chan struct{}
}

func NewSeekStream(source io.ReadCloser) (*SeekStream, error) {
	file, err := os.CreateTemp("", "imagor-")
	if err != nil {
		return nil, err
	}
	reader := io.TeeReader(source, file)
	return &SeekStream{
		file:   file,
		reader: reader,
		source: source,
	}, nil
}

func (s *SeekStream) Read(p []byte) (n int, err error) {
	if s.reader != nil {
		return s.reader.Read(p)
	}
	<-s.seekReady
	if s.file != nil {
		return s.file.Read(p)
	}
	return 0, errors.New("todo")
}

func (s *SeekStream) Seek(offset int64, whence int) (int64, error) {
	s.reader = nil
	return 0, errors.New("todo")
}

func (s *SeekStream) Close() error {
	if s.file != nil {
		_ = s.file.Close()
	}
	return s.source.Close()
}

func (s *SeekStream) Size() int64 {
	<-s.seekReady
	return s.size
}
