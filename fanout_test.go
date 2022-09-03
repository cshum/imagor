package imagor

import (
	"bytes"
	"context"
	"errors"
	"github.com/stretchr/testify/assert"
	"golang.org/x/sync/errgroup"
	"io"
	"testing"
	"time"
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
	newReader := fanoutReader(source, 5)
	doFanoutTest(t, func() {
		reader, _, closer := newReader()
		res1, err := io.ReadAll(reader)
		assert.NoError(t, err)
		assert.NoError(t, closer.Close())
		assert.Equal(t, buf[:5], res1)
	}, 100, 1)
}

func TestFanoutSizeBelow(t *testing.T) {
	time.Sleep(time.Millisecond)
	buf := []byte("abcd")
	source := io.NopCloser(bytes.NewReader(buf))
	newReader := fanoutReader(source, 5)
	doFanoutTest(t, func() {
		reader, _, closer := newReader()
		res1, err := io.ReadAll(reader)
		assert.NoError(t, err)
		assert.NoError(t, closer.Close())
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
	newReader := fanoutReader(source, 10000)
	doFanoutTest(t, func() {
		reader, _, _ := newReader()
		res, err := io.ReadAll(reader)
		assert.ErrorIs(t, err, e)
		assert.Equal(t, []byte("abcdefghi"), res)
	}, 100, 1)
}

func TestFanoutErrClosedPipe(t *testing.T) {
	buf := []byte("abcdefghi")
	source := io.NopCloser(bytes.NewReader(buf))
	newReader := fanoutReader(source, len(buf))
	reader, _, closer := newReader()
	b := make([]byte, 5)
	n, err := reader.Read(b)
	assert.NoError(t, err)
	assert.Equal(t, n, 5)
	assert.Equal(t, buf[:5], b)
	assert.NoError(t, closer.Close())
	b = make([]byte, 5)
	n, err = reader.Read(b)
	assert.ErrorIs(t, err, io.ErrClosedPipe)
	assert.Empty(t, n)
}

func TestFanoutSeek(t *testing.T) {
	buf := []byte("0123456789")
	source := io.NopCloser(bytes.NewReader(buf))
	newReader := fanoutReader(source, len(buf))
	reader, seeker, _ := newReader()
	tests := []struct {
		off     int64
		seek    int
		n       int
		want    string
		wantpos int64
		readerr error
		seekerr string
	}{
		{seek: io.SeekStart, off: 0, n: 20, want: "0123456789"},
		{seek: io.SeekStart, off: 1, n: 1, want: "1"},
		{seek: io.SeekCurrent, off: 1, wantpos: 3, n: 2, want: "34"},
		{seek: io.SeekStart, off: -1, seekerr: "bytes.Reader.Seek: negative position"},
		{seek: io.SeekStart, off: 1 << 33, wantpos: 1 << 33, readerr: io.EOF},
		{seek: io.SeekCurrent, off: 1, wantpos: 1<<33 + 1, readerr: io.EOF},
		{seek: io.SeekStart, n: 5, want: "01234"},
		{seek: io.SeekCurrent, n: 5, want: "56789"},
		{seek: io.SeekEnd, off: -1, n: 1, wantpos: 9, want: "9"},
	}

	for i, tt := range tests {
		pos, err := seeker.Seek(tt.off, tt.seek)
		if err == nil && tt.seekerr != "" {
			t.Errorf("%d. want seek error %q", i, tt.seekerr)
			continue
		}
		if err != nil && err.Error() != tt.seekerr {
			t.Errorf("%d. seek error = %q; want %q", i, err.Error(), tt.seekerr)
			continue
		}
		if tt.wantpos != 0 && tt.wantpos != pos {
			t.Errorf("%d. pos = %d, want %d", i, pos, tt.wantpos)
		}
		buf := make([]byte, tt.n)
		n, err := reader.Read(buf)
		if err != tt.readerr {
			t.Errorf("%d. read = %v; want %v", i, err, tt.readerr)
			continue
		}
		got := string(buf[:n])
		if got != tt.want {
			t.Errorf("%d. got %q; want %q", i, got, tt.want)
		}
	}
}
