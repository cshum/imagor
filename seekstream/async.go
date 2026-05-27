package seekstream

import (
	"errors"
	"io"
	"sync"
)

const asyncChunkSize = 32 << 10

// AsyncReadSeeker provides a seekable view over a non-seekable reader while
// filling chunk buffers in the background.
type AsyncReadSeeker struct {
	source   io.ReadCloser
	expected int64
	pos      int64

	mu         sync.Mutex
	cond       *sync.Cond
	chunks     [][]byte
	bytesRead  int64
	finished   bool
	closed     bool
	sourceDone bool
	readErr    error
}

// NewAsync creates an async chunk-buffered read seeker over source.
func NewAsync(source io.ReadCloser, expected int64) *AsyncReadSeeker {
	s := &AsyncReadSeeker{source: source, expected: expected}
	s.cond = sync.NewCond(&s.mu)
	go s.fill()
	return s
}

func (s *AsyncReadSeeker) fill() {
	source := s.source
	if source == nil {
		s.mu.Lock()
		s.finished = true
		s.readErr = io.ErrClosedPipe
		s.cond.Broadcast()
		s.mu.Unlock()
		return
	}
	for {
		buf := make([]byte, asyncChunkSize)
		n, err := source.Read(buf)

		s.mu.Lock()
		if s.closed {
			s.mu.Unlock()
			return
		}
		if n > 0 {
			s.chunks = append(s.chunks, buf[:n])
			s.bytesRead += int64(n)
		}
		if err != nil {
			s.finished = true
			if err != io.EOF {
				s.readErr = err
			}
		}
		s.cond.Broadcast()
		done := s.finished
		var src io.ReadCloser
		if done {
			src = s.detachSourceLocked()
		}
		s.mu.Unlock()

		if done {
			if src != nil {
				_ = src.Close()
			}
			return
		}
	}
}

func (s *AsyncReadSeeker) detachSourceLocked() io.ReadCloser {
	if s.sourceDone {
		return nil
	}
	s.sourceDone = true
	src := s.source
	s.source = nil
	return src
}

func (s *AsyncReadSeeker) copyLocked(dst []byte, off int64) int {
	if off >= s.bytesRead || len(dst) == 0 {
		return 0
	}
	total := 0
	chunkIndex := int(off / asyncChunkSize)
	chunkOffset := int(off % asyncChunkSize)
	for total < len(dst) && chunkIndex < len(s.chunks) {
		chunk := s.chunks[chunkIndex]
		if chunkOffset >= len(chunk) {
			chunkIndex++
			chunkOffset = 0
			continue
		}
		n := copy(dst[total:], chunk[chunkOffset:])
		total += n
		chunkIndex++
		chunkOffset = 0
	}
	return total
}

// Read implements io.Reader.
func (s *AsyncReadSeeker) Read(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return 0, io.ErrClosedPipe
	}
	if len(p) == 0 {
		return 0, nil
	}

	total := 0
	for total < len(p) {
		for !s.closed && s.pos >= s.bytesRead && !s.finished {
			s.cond.Wait()
		}
		if s.closed {
			if total > 0 {
				return total, nil
			}
			return 0, io.ErrClosedPipe
		}
		if s.pos >= s.bytesRead {
			if s.readErr != nil {
				if total > 0 {
					return total, nil
				}
				return 0, s.readErr
			}
			if s.finished {
				if total > 0 {
					return total, nil
				}
				return 0, io.EOF
			}
		}
		n := s.copyLocked(p[total:], s.pos)
		if n == 0 {
			continue
		}
		total += n
		s.pos += int64(n)
	}
	return total, nil
}

// Seek implements io.Seeker.
func (s *AsyncReadSeeker) Seek(offset int64, whence int) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return 0, io.ErrClosedPipe
	}

	var dest int64
	switch whence {
	case io.SeekStart:
		dest = offset
	case io.SeekCurrent:
		dest = s.pos + offset
	case io.SeekEnd:
		for !s.closed && !s.finished {
			s.cond.Wait()
		}
		if s.closed {
			return 0, io.ErrClosedPipe
		}
		dest = s.bytesRead + offset
	default:
		return 0, errors.New("invalid argument")
	}
	if dest < 0 {
		return 0, errors.New("invalid argument")
	}
	s.pos = dest
	return dest, nil
}

// Close implements io.Closer.
func (s *AsyncReadSeeker) Close() error {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return nil
	}
	s.closed = true
	s.chunks = nil
	src := s.detachSourceLocked()
	s.cond.Broadcast()
	s.mu.Unlock()
	if src != nil {
		return src.Close()
	}
	return nil
}

// Len returns the unread buffered bytes when the logical position is within the loaded range.
func (s *AsyncReadSeeker) Len() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	limit := s.bytesRead
	if !s.finished && s.expected > limit {
		limit = s.expected
	}
	if s.pos >= limit {
		return 0
	}
	return int(limit - s.pos)
}

// Size returns the loaded size, or the expected size when known and larger.
func (s *AsyncReadSeeker) Size() int64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.finished && s.expected > 0 {
		return s.expected
	}
	return s.bytesRead
}
