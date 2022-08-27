package imagor

import (
	"bytes"
	"io"
	"sync"
)

func fanoutReader(reader io.ReadCloser, size int) func(seekable bool) io.ReadCloser {
	var lock sync.RWMutex
	var once sync.Once
	var consumers []chan []byte
	var closed []bool
	var err error
	var buf []byte
	var curr int

	var init = func() {
		defer func() {
			_ = reader.Close()
		}()
		for {
			b := make([]byte, 512)
			n, e := reader.Read(b)
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
			lock.Unlock()
			lock.RLock()
			for i, ch := range consumers {
				if !closed[i] {
					ch <- bn
				}
			}
			lock.RUnlock()
			if e != nil || curr >= size {
				return
			}
		}
	}

	return func(seekable bool) io.ReadCloser {
		ch := make(chan []byte, size/512+1)

		lock.Lock()
		i := len(consumers)
		consumers = append(consumers, ch)
		closed = append(closed, false)
		cnt := len(buf)
		bufReader := bytes.NewReader(buf)
		lock.Unlock()

		var b []byte
		var closer = closerFunc(func() (e error) {
			lock.Lock()
			e = err
			if closed[i] {
				lock.Unlock()
				return
			}
			closed[i] = true
			lock.Unlock()
			close(ch)
			return
		})
		var reader = io.MultiReader(
			bufReader,
			readerFunc(func(p []byte) (n int, e error) {
				once.Do(func() {
					go init()
				})

				lock.RLock()
				e = err
				s := size
				c := closed[i]
				lock.RUnlock()

				if cnt >= s {
					return 0, io.EOF
				}
				if c {
					return 0, io.ErrClosedPipe
				}
				if e != nil {
					_ = closer()
					return
				}
				if len(b) == 0 {
					b = <-ch
				}
				n = copy(p, b)
				b = b[n:]
				cnt += n
				if cnt >= s {
					_ = closer()
					e = io.EOF
				}
				return
			}),
		)
		if seekable {
			return &readSeekCloser{
				Reader: reader,
				Closer: closer,
			}
		} else {
			return &readCloser{
				Reader: reader,
				Closer: closer,
			}
		}
	}
}

type readSeekCloser struct {
	io.Reader
	io.Seeker
	io.Closer
}

type readCloser struct {
	io.Reader
	io.Closer
}

type readerFunc func(p []byte) (n int, err error)

func (rf readerFunc) Read(p []byte) (n int, err error) { return rf(p) }

type closerFunc func() error

func (cf closerFunc) Close() error { return cf() }
