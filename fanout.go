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

	return func() (reader io.Reader, seeker io.Seeker, closer io.Closer) {
		ch := make(chan []byte, size/4096+1)

		lock.Lock()
		i := len(consumers)
		consumers = append(consumers, ch)
		closed = append(closed, false)
		cnt := currentSize
		bufReader := bytes.NewReader(buf[:currentSize])
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
			if fullBufReader != nil {
				// proxy to full buf if ready
				return fullBufReader.Read(p)
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
		})
		seeker = seekerFunc(func(offset int64, whence int) (int64, error) {
			once.Do(func() {
				go init()
			})
			if readerClosed {
				return 0, io.ErrClosedPipe
			}
			if bufReader != nil &&
				((whence == io.SeekStart && offset < bufReader.Size()) ||
					(whence == io.SeekCurrent && offset < int64(bufReader.Len()))) {
				return bufReader.Seek(offset, whence)
			}
			if fullBufReader != nil {
				return fullBufReader.Seek(offset, whence)
			}
			<-fullBufReady
			fullBufReader = bytes.NewReader(buf)
			bufReader = nil
			_ = closeCh(false)
			return fullBufReader.Seek(offset, whence)
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
