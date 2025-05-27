package fanoutreader

import (
	"io"
	"sync"
)

// Fanout allows fanout arbitrary number of reader streams concurrently
// from one data source with known total size,
// using channel and memory buffer.
type Fanout struct {
	source   io.ReadCloser
	size     int
	current  int
	buf      []byte
	err      error
	lock     sync.RWMutex
	once     sync.Once
	readers  []*reader
	released bool
}

// reader io.ReadCloser spawned via Fanout
type reader struct {
	fanout       *Fanout
	channel      chan []byte
	closeChannel chan struct{}
	buf          []byte
	current      int
	readerClosed bool
}

// eofReader always returns EOF - used for new readers after Release()
type eofReader struct{}

func (eofReader) Read([]byte) (int, error) { return 0, io.EOF }
func (eofReader) Close() error             { return nil }

// New Fanout factory via single io.ReadCloser source with known size
func New(source io.ReadCloser, size int) *Fanout {
	return &Fanout{
		source: source,
		size:   size,
		buf:    make([]byte, size),
	}
}

// Release marks the fanout as released. Buffer is freed when all readers are also closed.
func (f *Fanout) Release() {
	f.lock.Lock()
	f.released = true
	f.tryCleanupBuffer()
	f.lock.Unlock()
}

// tryCleanupBuffer frees the buffer when safe to do so
func (f *Fanout) tryCleanupBuffer() {
	if f.released && len(f.readers) == 0 && f.buf != nil {
		f.buf = nil // Free the buffer when released and no active readers
	}
}

// do triggers reading data from source
func (f *Fanout) do() {
	f.once.Do(func() {
		go f.readAll()
	})
}

func (f *Fanout) readAll() {
	defer func() {
		_ = f.source.Close()

		f.lock.Lock()
		for _, r := range f.readers {
			if !r.readerClosed {
				select {
				case <-r.closeChannel:
				default:
					close(r.channel)
				}
			}
		}
		f.lock.Unlock()
	}()
	for f.current < f.size {
		b := f.buf[f.current:]
		n, e := f.source.Read(b)
		if f.current+n > f.size {
			n = f.size - f.current
		}
		var bn []byte
		if n > 0 {
			bn = b[:n]
		}
		f.lock.Lock()
		f.current += n
		if e != nil {
			if e == io.EOF {
				e = nil
			} else {
				f.err = e
			}
			if n == 0 {
				if f.current < f.size {
					f.buf = f.buf[:f.current]
				}
				f.size = f.current
			}
		}
		readersCopy := make([]*reader, len(f.readers))
		copy(readersCopy, f.readers)
		f.lock.Unlock()

		var closedReaders []*reader
		for _, r := range readersCopy {
			select {
			case <-r.closeChannel:
				close(r.channel)
				closedReaders = append(closedReaders, r)
			case r.channel <- bn:
			}
		}

		// Drop all the closed readers from readers list
		if len(closedReaders) > 0 {
			f.lock.Lock()
			crIdx := 0
			newPos := 0
			for i, r := range f.readers {
				if crIdx < len(closedReaders) && r == closedReaders[crIdx] {
					crIdx++
				} else {
					f.readers[newPos] = f.readers[i]
					newPos++
				}
			}
			f.readers = f.readers[:newPos]
			// Try to cleanup buffer after removing closed readers
			f.tryCleanupBuffer()
			f.lock.Unlock()
		}
	}
}

// NewReader spawns new io.ReadCloser
// After Release() is called, returns a reader that immediately gives EOF
func (f *Fanout) NewReader() io.ReadCloser {
	r := &reader{fanout: f}

	f.lock.Lock()
	defer f.lock.Unlock()

	if f.released {
		return eofReader{}
	}

	bufferSize := f.size/4096 + 1
	if bufferSize > 32 {
		bufferSize = 32
	}
	r.channel = make(chan []byte, bufferSize)
	r.closeChannel = make(chan struct{})

	// Set initial buffer if data already read
	if f.current > 0 && f.buf != nil {
		r.buf = f.buf[:f.current]
	}

	// Add to readers list
	f.readers = append(f.readers, r)
	return r
}

// Read implements the io.Reader interface.
func (r *reader) Read(p []byte) (n int, err error) {
	r.fanout.do()
	if r.readerClosed {
		return 0, io.ErrClosedPipe
	}
	r.fanout.lock.RLock()
	e := r.fanout.err
	size := r.fanout.size
	r.fanout.lock.RUnlock()
	for {
		if r.current >= size {
			if e != nil {
				return 0, e
			}
			return 0, io.EOF
		}
		if len(r.buf) == 0 {
			var ok bool
			r.buf, ok = <-r.channel
			if !ok {
				return 0, io.ErrClosedPipe
			}
		}
		nn := copy(p[n:], r.buf)
		if nn == 0 {
			return
		}
		r.buf = r.buf[nn:]
		r.current += nn
		n += nn
		if r.current >= size {
			_ = r.close(false)
			return
		}
	}
}

// close reader or just closing the underlying channel
func (r *reader) close(closeReader bool) (e error) {
	r.fanout.lock.Lock()
	defer r.fanout.lock.Unlock()

	e = r.fanout.err
	r.readerClosed = closeReader

	// Clear reader buffer to free memory immediately
	if closeReader {
		r.buf = nil
	}

	// Close channel if it's not closed yet
	select {
	case <-r.closeChannel:
	default:
		close(r.closeChannel)
	}

	// If this reader is being closed, remove it from the readers slice
	if closeReader {
		for i, reader := range r.fanout.readers {
			if reader == r {
				// Remove this reader from the slice
				copy(r.fanout.readers[i:], r.fanout.readers[i+1:])
				r.fanout.readers = r.fanout.readers[:len(r.fanout.readers)-1]
				break
			}
		}
		// Try to cleanup buffer after removing this reader
		r.fanout.tryCleanupBuffer()
	}

	return
}

// Close implements the io.Closer interface.
func (r *reader) Close() error {
	return r.close(true)
}
