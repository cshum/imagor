package imagor

import (
	"bytes"
	"io"
	"sync"
)

func FanoutReader(reader io.ReadCloser, size int) func() (io.ReadCloser, error) {
	var lock sync.RWMutex
	var consumers []chan []byte
	var err error
	var buf []byte
	var cnt int
	go func() {
		defer func() {
			_ = reader.Close()
		}()
		for {
			b := make([]byte, 512)
			n, e := reader.Read(b)
			lock.Lock()
			buf = append(buf, b[:n]...)
			cnt += n
			if cnt <= size {
				for _, ch := range consumers {
					ch <- b[:n]
				}
				if e != nil && e != io.EOF {
					err = e
				}
			}
			lock.Unlock()
			if e != nil {
				return
			}
		}
	}()
	return func() (io.ReadCloser, error) {
		lock.Lock()
		ch := make(chan []byte, size)
		consumers = append(consumers, ch)
		var cnt = len(buf)
		var b []byte
		r := io.NopCloser(io.MultiReader(bytes.NewReader(buf), readFunc(func(p []byte) (n int, e error) {
			if cnt >= size {
				return 0, io.EOF
			}
			lock.RLock()
			e = err
			lock.RUnlock()
			if e != nil {
				return
			}
			if len(b) == 0 {
				b = <-ch
			}
			n = copy(p, b)
			b = b[n:]
			cnt += n
			return
		})))
		e := err
		lock.Unlock()
		return r, e
	}
}

type readFunc func(p []byte) (n int, err error)

func (rf readFunc) Read(p []byte) (n int, err error) { return rf(p) }
