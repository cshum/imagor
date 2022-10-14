package seekstream

import (
	"io"
)

// SeekStream allows seeking on non-seekable io.ReadCloser source
// by buffering read data using memory or temp file.
type SeekStream struct {
	source io.ReadCloser
	buffer Buffer
	size   int64
	curr   int64
	loaded bool
}

// New SeekStream proving io.ReadCloser source and buffer interface
func New(source io.ReadCloser, buffer Buffer) *SeekStream {
	return &SeekStream{
		source: source,
		buffer: buffer,
	}
}

// Read implements the io.Reader interface.
func (s *SeekStream) Read(p []byte) (n int, err error) {
	if s.source == nil || s.buffer == nil {
		return 0, io.ErrClosedPipe
	}
	if s.loaded && s.curr >= s.size {
		err = io.EOF
		return
	} else if s.curr < s.size {
		n, err = s.buffer.Read(p)
		s.curr += int64(n)
		if err != nil && err != io.EOF {
			return
		}
	}
	if s.loaded || len(p) == n {
		return
	}
	pn := p[n:]
	var nn int
	nn, err = s.source.Read(pn)
	n += nn
	if nn > 0 {
		if n, err := s.buffer.Write(pn[:nn]); err != nil {
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

// Seek implements the io.Seeker interface.
func (s *SeekStream) Seek(offset int64, whence int) (int64, error) {
	if s.source == nil || s.buffer == nil {
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
				n, err := s.buffer.Seek(s.size, io.SeekStart)
				s.curr = n
				if err != nil {
					return n, err
				}
			}
			n, err := io.Copy(s.buffer, s.source)
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
		nn, err := io.CopyN(s.buffer, s.source, dest-s.size)
		s.size += nn
		if err == io.EOF {
			s.loaded = true
		} else if err != nil {
			return 0, err
		}
	}
	n, err := s.buffer.Seek(dest, io.SeekStart)
	s.curr = n
	return n, err
}

// Close implements the io.Closer interface.
func (s *SeekStream) Close() (err error) {
	if s.buffer != nil {
		s.buffer.Clear()
		s.buffer = nil
	}
	if s.source != nil {
		err = s.source.Close()
		s.source = nil
	}
	return
}

// Len returns the number of bytes of the unread portion of buffer
func (s *SeekStream) Len() int {
	if s.curr >= s.size {
		return 0
	}
	return int(s.size - s.curr)
}

// Size returns the length of the underlying buffer
func (s *SeekStream) Size() int64 {
	return s.size
}
