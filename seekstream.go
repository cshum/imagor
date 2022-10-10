package imagor

import (
	"io"
	"os"
	"sync"
)

type SeekStream struct {
	source io.ReadCloser
	size   int64
	curr   int64
	loaded bool
	file   *os.File
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
	if s.curr < s.size {
		n, err = s.file.Read(p)
		s.curr += int64(n)
		if err != nil && err != io.EOF {
			return
		}
	}
	if s.loaded || len(p) == n {
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
	s.size += int64(nn)
	s.curr += int64(nn)
	return
}

func (s *SeekStream) Seek(offset int64, whence int) (int64, error) {
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
		dest = s.curr + offset
	case io.SeekEnd:
		if !s.loaded {
			if s.curr != s.size {
				n, err := s.file.Seek(s.size, io.SeekStart)
				if err != nil {
					return n, err
				}
				s.curr = n
			}
			n, err := io.Copy(s.file, s.source)
			if err != nil {
				return 0, err
			}
			s.curr += n
			s.size += n
			s.loaded = true
		}
		dest = s.size + offset
	}
	if !s.loaded && dest > s.size {
		nn, err := io.CopyN(s.file, s.source, dest-s.size)
		s.size += nn
		if err == io.EOF {
			s.loaded = true
		} else if err != nil {
			return 0, err
		}
	}
	n, err := s.file.Seek(dest, io.SeekStart)
	s.curr = n
	return n, err
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

func (s *SeekStream) Len() int {
	if s.curr >= s.size {
		return 0
	}
	return int(s.size - s.curr)
}

func (s *SeekStream) Size() int64 {
	return s.size
}
