package imagor

import (
	"io"
	"os"
	"sync"
)

type SeekStream struct {
	source io.ReadCloser
	file   *os.File
	seeked bool
	l      sync.RWMutex
}

func NewSeekStream(source io.ReadCloser) (*SeekStream, error) {
	file, err := os.CreateTemp("", "imagor-")
	if err != nil {
		return nil, err
	}
	return &SeekStream{
		source: source,
		file:   file,
	}, nil
}

func (s *SeekStream) Read(p []byte) (n int, err error) {
	s.l.RLock()
	defer s.l.RUnlock()
	if s.file == nil || s.source == nil {
		return 0, io.ErrClosedPipe
	}
	if !s.seeked {
		n, err = s.source.Read(p)
		if n > 0 {
			if n, err := s.file.Write(p[:n]); err != nil {
				return n, err
			}
		}
		return
	}
	return s.file.Read(p)
}

func (s *SeekStream) Seek(offset int64, whence int) (int64, error) {
	s.l.Lock()
	defer s.l.Unlock()
	if s.file == nil || s.source == nil {
		return 0, io.ErrClosedPipe
	}
	if !s.seeked {
		_, err := io.Copy(s.file, s.source)
		if err != nil {
			return 0, err
		}
		filename := s.file.Name()
		_ = s.file.Close()
		if s.file, err = os.Open(filename); err != nil {
			_ = s.Close()
			return 0, err
		}
		s.seeked = true
	}
	return s.file.Seek(offset, whence)
}

func (s *SeekStream) Close() (err error) {
	s.l.Lock()
	defer s.l.Unlock()
	if s.file != nil {
		filename := s.file.Name()
		_ = s.file.Close()
		_ = os.Remove(filename)
		s.file = nil
	}
	if s.source != nil {
		err = s.source.Close()
		s.source = nil
	}
	return
}
