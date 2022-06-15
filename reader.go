package imagor

import (
	"bytes"
	"io"
)

func FanoutReader(r io.ReadCloser) func() (io.ReadCloser, error) {
	buf, err := io.ReadAll(r)
	_ = r.Close()
	return func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(buf)), err
	}
}
