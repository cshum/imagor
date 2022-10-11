package imagor

import (
	"bytes"
	"io"
	"sync"
)

func fanoutReader(source io.ReadCloser, size int) func() io.ReadCloser {
	var lock sync.RWMutex
	var once sync.Once
	var consumers []chan []byte
	var fullBufReady = make(chan struct{})
	var closed []bool
	var err error
	var buf = make([]byte, size)
	var currentSize int

	var init = func() {
		defer func() {
			_ = source.Close()
		}()
		for {
			b := buf[currentSize:]
			n, e := source.Read(b)
			if currentSize+n > size {
				n = size - currentSize
			}
			var bn []byte
			if n > 0 {
				bn = b[:n]
			}
			lock.Lock()
			currentSize += n
			if e != nil {
				if e == io.EOF {
					e = nil
					if n == 0 {
						if currentSize < size {
							buf = buf[:currentSize]
						}
						size = currentSize
					}
				} else {
					err = e
				}
			}
			consumersCopy := consumers
			lock.Unlock()
			lock.RLock()
			for i, ch := range consumersCopy {
				if !closed[i] {
					ch <- bn
				}
			}
			lock.RUnlock()
			if currentSize >= size {
				close(fullBufReady)
			}
			if e != nil || currentSize >= size {
				return
			}
		}
	}

	return func() io.ReadCloser {
		ch := make(chan []byte, size/4096+1)

		lock.Lock()
		i := len(consumers)
		consumers = append(consumers, ch)
		closed = append(closed, false)
		cnt := currentSize
		bufReader := bytes.NewReader(buf[:currentSize])
		lock.Unlock()

		var readerClosed bool
		var b []byte
		var closeCh = func(closeReader bool) (e error) {
			lock.Lock()
			e = err
			readerClosed = closeReader
			if closed[i] {
				lock.Unlock()
			} else {
				closed[i] = true
				lock.Unlock()
				close(ch)
			}
			return
		}
		return &readCloser{
			Reader: readerFunc(func(p []byte) (n int, e error) {
				once.Do(func() {
					go init()
				})
				if readerClosed {
					return 0, io.ErrClosedPipe
				}
				if bufReader != nil {
					n, e = bufReader.Read(p)
					if e == io.EOF {
						bufReader = nil
						e = nil
						// Don't return EOF, pass to next reader instead
					} else {
						return
					}
				}

				lock.RLock()
				e = err
				sizeCopy := size
				closedCopy := closed[i]
				lock.RUnlock()

				for {
					if cnt >= sizeCopy {
						return 0, io.EOF
					}
					if closedCopy {
						return 0, io.ErrClosedPipe
					}
					if e != nil {
						_ = closeCh(true)
						return
					}
					if len(b) == 0 {
						b = <-ch
					}
					nn := copy(p[n:], b)
					if nn == 0 {
						return
					}
					b = b[nn:]
					cnt += nn
					n += nn
					if cnt >= sizeCopy {
						_ = closeCh(false)
						return
					}
				}
			}),
			Closer: closerFunc(func() error {
				return closeCh(true)
			}),
		}
	}
}

type readerFunc func(p []byte) (n int, err error)

func (rf readerFunc) Read(p []byte) (n int, err error) { return rf(p) }

type closerFunc func() error

func (cf closerFunc) Close() error { return cf() }
