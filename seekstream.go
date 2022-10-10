package imagor

import (
	"io"
	"os"
	"sync"
)

type seekStream struct {
	source  io.ReadCloser
	total   int64
	current int64
	loaded  bool
	file    *os.File
	l       sync.RWMutex
}

func newSeekStream(source io.ReadCloser) (*seekStream, error) {
	file, err := os.CreateTemp("", "imagor-")
	if err != nil {
		return nil, err
	}
	return &seekStream{
		source: source,
		file:   file,
	}, nil
}

func (s *seekStream) Read(p []byte) (n int, err error) {
	s.l.RLock()
	defer s.l.RUnlock()
	if s.file == nil || s.source == nil {
		return 0, io.ErrClosedPipe
	}
	if s.current < s.total {
		n, err = s.file.Read(p)
		s.current += int64(n)
		if err == io.EOF {
			err = nil
		}
		if err != nil {
			return
		}
	}
	if len(p) == n {
		return
	}
	pn := p[n:]
	nn, err := s.source.Read(pn)
	n += nn
	if nn > 0 {
		if n, err := s.file.Write(pn[:nn]); err != nil {
			return n, err
		}
	}
	if err == io.EOF {
		s.loaded = true
	}
	s.total += int64(nn)
	s.current += int64(nn)
	return
}

func (s *seekStream) Seek(offset int64, whence int) (int64, error) {
	s.l.Lock()
	defer s.l.Unlock()
	if s.file == nil || s.source == nil {
		return 0, io.ErrClosedPipe
	}
	var dest int64
	switch whence {
	case io.SeekStart:
		dest = offset
	case io.SeekCurrent:
		dest = s.current + offset
	case io.SeekEnd:
		if !s.loaded {
			if s.current != s.total {
				n, err := s.file.Seek(s.total, io.SeekStart)
				if err != nil {
					return 0, err
				}
				s.current = n
			}
			n, err := io.Copy(s.file, s.source)
			if err != nil {
				return 0, err
			}
			s.current += n
			s.total += n
			s.loaded = true
		}
		dest = s.total + offset
	}
	n, err := s.file.Seek(dest, io.SeekStart)
	s.current = n
	return n, err
}

func (s *seekStream) Close() (err error) {
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
