package imagor

import (
	"bytes"
	"io"
	"sync"
)

// FanoutReader fan-out io.ReadCloser - known size, unknown number of consumers
func FanoutReader(reader io.ReadCloser, size int) func() io.ReadCloser {
	var lock sync.RWMutex
	var consumers []chan []byte
	var closed []bool
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
			if cnt+n > size {
				n = size - cnt
			}
			bn := b[:n]

			lock.Lock()
			buf = append(buf, bn...)
			cnt += n
			if e != nil {
				size = cnt
				if e != io.EOF {
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
			if e != nil || cnt >= size {
				return
			}
		}
	}()
	return func() io.ReadCloser {
		ch := make(chan []byte, size/512+1)

		lock.Lock()
		i := len(consumers)
		consumers = append(consumers, ch)
		closed = append(closed, false)
		cnt := len(buf)
		bufReader := bytes.NewReader(buf)
		lock.Unlock()

		closeCh := func() {
			lock.Lock()
			if closed[i] {
				lock.Unlock()
				return
			}
			closed[i] = true
			lock.Unlock()
			close(ch)
		}
		var b []byte
		return io.NopCloser(io.MultiReader(
			bufReader,
			readerFunc(func(p []byte) (n int, e error) {
				lock.RLock()
				e = err
				s := size
				lock.RUnlock()

				if cnt >= s {
					return 0, io.EOF
				}
				if e != nil {
					closeCh()
					return
				}
				if len(b) == 0 {
					b = <-ch
				}
				n = copy(p, b)
				b = b[n:]
				cnt += n
				if cnt >= s {
					closeCh()
					e = io.EOF
				}
				return
			}),
		))
	}
}

type readerFunc func(p []byte) (n int, err error)

func (rf readerFunc) Read(p []byte) (n int, err error) { return rf(p) }
