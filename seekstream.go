package imagor

import (
	"io"
	"os"
	"sync"
)

type SeekStream struct {
	source io.ReadCloser
	file   *os.File
	size   int64
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
	if !s.seeked {
		n, err = s.source.Read(p)
		if n > 0 {
			s.size += int64(n)
			if n, err := s.file.Write(p[:n]); err != nil {
				return n, err
			}
		}
		return
	}
	if s.file != nil {
		return s.file.Read(p)
	}
	return 0, io.ErrClosedPipe
}

func (s *SeekStream) Seek(offset int64, whence int) (int64, error) {
	s.l.Lock()
	defer s.l.Unlock()
	if !s.seeked {
		s.seeked = true
		if s.file != nil {
			filename := s.file.Name()
			_ = s.file.Close()
			var err error
			if s.file, err = os.Open(filename); err != nil {
				_ = s.Close()
				return 0, err
			}
		} else {
			return 0, io.ErrClosedPipe
		}
		n, err := io.Copy(s.file, s.source)
		s.size += n
		if err != nil {
			return 0, err
		}
	}
	if s.file != nil {
		return s.file.Seek(offset, whence)
	}
	return 0, io.ErrClosedPipe
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

func (s *SeekStream) Size() int64 {
	s.l.RLock()
	defer s.l.RUnlock()
	return s.size
}
