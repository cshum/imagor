package imagor

import (
	"bytes"
	"context"
	"errors"
	"github.com/stretchr/testify/assert"
	"golang.org/x/sync/errgroup"
	"io"
	"testing"
)

func doFanoutTest(t *testing.T, do func(), n, m int) {
	g, _ := errgroup.WithContext(context.Background())
	for i := 0; i < n; i++ {
		g.Go(func() error {
			do()
			return nil
		})
	}
	assert.NoError(t, g.Wait())
	for i := 0; i < m; i++ {
		do()
	}
}

func TestFanoutSizeOver(t *testing.T) {
	buf := []byte("abcdefghi")
	source := io.NopCloser(bytes.NewReader(buf))
	newReader := FanoutReader(source, 5)
	doFanoutTest(t, func() {
		reader := newReader()
		res1, err := io.ReadAll(reader)
		assert.NoError(t, err)
		assert.NoError(t, reader.Close())
		assert.Equal(t, buf[:5], res1)
	}, 100, 1)
}

func TestFanoutSizeBelow(t *testing.T) {
	buf := []byte("abcd")
	source := io.NopCloser(bytes.NewReader(buf))
	newReader := FanoutReader(source, 5)
	doFanoutTest(t, func() {
		reader := newReader()
		res1, err := io.ReadAll(reader)
		assert.NoError(t, err)
		assert.NoError(t, reader.Close())
		assert.Equal(t, buf, res1)
	}, 100, 1)
}

func TestFanoutUpstreamError(t *testing.T) {
	e := errors.New("upstream error")
	buf := []byte("abcdefghi")
	called := false
	source := io.NopCloser(readerFunc(func(p []byte) (n int, err error) {
		if called {
			return 0, e
		}
		called = true
		n = copy(p, buf)
		return
	}))
	newReader := FanoutReader(source, 10000)
	doFanoutTest(t, func() {
		reader := newReader()
		res, err := io.ReadAll(reader)
		assert.ErrorIs(t, err, e)
		assert.Equal(t, []byte("abcdefghi"), res)
	}, 100, 1)
}

func TestFanoutErrClosedPipe(t *testing.T) {
	buf := []byte("abcdefghi")
	source := io.NopCloser(bytes.NewReader(buf))
	newReader := FanoutReader(source, len(buf))
	reader := newReader()
	b := make([]byte, 5)
	n, err := reader.Read(b)
	assert.NoError(t, err)
	assert.Equal(t, n, 5)
	assert.Equal(t, buf[:5], b)
	assert.NoError(t, reader.Close())
	b = make([]byte, 5)
	n, err = reader.Read(b)
	assert.ErrorIs(t, err, io.ErrClosedPipe)
	assert.Empty(t, n)
}
