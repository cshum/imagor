package imagor

import (
	"bytes"
	"io"
	"sync"
)

func fanoutReader(source io.ReadCloser, size int) func() (io.Reader, io.Seeker, io.Closer) {
	var lock sync.RWMutex
	var once sync.Once
	var consumers []chan []byte
	var done = make(chan struct{})
	var closed []bool
	var err error
	var buf []byte
	var curr int

	var init = func() {
		defer func() {
			_ = source.Close()
		}()
		for {
			b := make([]byte, 512)
			n, e := source.Read(b)
			if curr+n > size {
				n = size - curr
			}
			bn := b[:n]

			lock.Lock()
			buf = append(buf, bn...)
			curr += n
			if e != nil {
				if e == io.EOF {
					size = curr
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
			if curr >= size {
				close(done)
			}
			if e != nil || curr >= size {
				return
			}
		}
	}

	return func() (reader io.Reader, seeker io.Seeker, closer io.Closer) {
		ch := make(chan []byte, size/512+1)

		lock.Lock()
		i := len(consumers)
		consumers = append(consumers, ch)
		closed = append(closed, false)
		cnt := len(buf)
		bufReader := bytes.NewReader(buf)
		lock.Unlock()

		var readerClosed bool
		var fullBufReader *bytes.Reader

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
		closer = closerFunc(func() error {
			return closeCh(true)
		})
		reader = readerFunc(func(p []byte) (n int, e error) {
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
				}
				if n > 0 || e != nil {
					return
				}
			}

			if fullBufReader != nil && !readerClosed {
				// proxy to full buf if ready
				return fullBufReader.Read(p)
			}
			for {
				lock.RLock()
				e = err
				sizeCopy := size
				closedCopy := closed[i]
				lock.RUnlock()
				if cnt >= sizeCopy {
					_ = closeCh(false)
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
		})
		seeker = seekerFunc(func(offset int64, whence int) (int64, error) {
			once.Do(func() {
				go init()
			})

			if fullBufReader != nil && !readerClosed {
				return fullBufReader.Seek(offset, whence)
			} else if fullBufReader == nil && !readerClosed {
				<-done
				fullBufReader = bytes.NewReader(buf)
				_ = closeCh(false)
				return fullBufReader.Seek(offset, whence)
			} else {
				return 0, io.ErrClosedPipe
			}
		})
		return
	}
}

type readerFunc func(p []byte) (n int, err error)

func (rf readerFunc) Read(p []byte) (n int, err error) { return rf(p) }

type closerFunc func() error

func (cf closerFunc) Close() error { return cf() }

type seekerFunc func(offset int64, whence int) (int64, error)

func (sf seekerFunc) Seek(offset int64, whence int) (int64, error) { return sf(offset, whence) }
