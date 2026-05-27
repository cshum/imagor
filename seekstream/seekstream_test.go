package seekstream

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type countingReadCloser struct {
	reader     io.Reader
	closeCount atomic.Int32
	closed     chan struct{}
	closeOnce  sync.Once
}

func newCountingReadCloser(reader io.Reader) *countingReadCloser {
	return &countingReadCloser{
		reader: reader,
		closed: make(chan struct{}),
	}
}

func (r *countingReadCloser) Read(p []byte) (int, error) {
	return r.reader.Read(p)
}

func (r *countingReadCloser) Close() error {
	r.closeCount.Add(1)
	r.closeOnce.Do(func() { close(r.closed) })
	return nil
}

type scriptedReadCloser struct {
	steps []readStep
	index int
}

type readStep struct {
	data []byte
	err  error
}

func (r *scriptedReadCloser) Read(p []byte) (int, error) {
	if r.index >= len(r.steps) {
		return 0, io.EOF
	}
	step := r.steps[r.index]
	r.index++
	n := copy(p, step.data)
	return n, step.err
}

func (r *scriptedReadCloser) Close() error { return nil }

func doSeekStreamTests(t *testing.T, buffer Buffer) {
	buf := []byte("0123456789")
	source := io.NopCloser(bytes.NewReader(buf))
	rs := New(source, buffer)

	tests := []struct {
		off     int64
		seek    int
		n       int
		len     int
		size    int
		want    string
		wantpos int64
		readerr error
		seekerr string
	}{
		{seek: -1, n: 3, size: 3, want: "012"},
		{seek: -1, n: 2, size: 5, want: "34"},
		{seek: io.SeekCurrent, off: 1, n: 1, size: 7, want: "6"},
		{seek: io.SeekCurrent, off: -1, n: 2, size: 8, want: "67"},
		{seek: io.SeekStart, off: 2, n: 2, len: 4, size: 8, want: "23"},
		{seek: io.SeekEnd, off: -2, n: 20, size: 10, want: "89"},
		{seek: io.SeekStart, off: 20, n: 2, size: 10, readerr: io.EOF},
		{seek: io.SeekStart, off: 0, n: 20, size: 10, want: "0123456789"},
		{seek: io.SeekStart, off: 1, n: 1, len: 8, size: 10, want: "1"},
		{seek: io.SeekCurrent, off: 1, wantpos: 3, n: 2, len: 5, size: 10, want: "34"},
		{seek: io.SeekStart, off: -1, len: 10, size: 10, seekerr: "invalid argument"},
		{seek: io.SeekStart, off: 1 << 33, wantpos: 1 << 33, size: 10, readerr: io.EOF},
		{seek: io.SeekCurrent, off: 1, wantpos: 1<<33 + 1, size: 10, readerr: io.EOF},
		{seek: io.SeekStart, n: 5, len: 5, size: 10, want: "01234"},
		{seek: io.SeekCurrent, n: 5, len: 0, size: 10, want: "56789"},
		{seek: io.SeekEnd, off: -1, n: 1, wantpos: 9, len: 0, size: 10, want: "9"},
	}

	for i, tt := range tests {
		if tt.seek >= 0 {
			pos, err := rs.Seek(tt.off, tt.seek)
			if err == nil && tt.seekerr != "" {
				t.Errorf("%d. want seek error %q", i, tt.seekerr)
				continue
			}
			if err != nil && !strings.Contains(err.Error(), tt.seekerr) {
				t.Errorf("%d. seek error = %q; want contains %q", i, err.Error(), tt.seekerr)
				continue
			}
			if tt.wantpos != 0 && tt.wantpos != pos {
				t.Errorf("%d. pos = %d, want %d", i, pos, tt.wantpos)
			}
		}
		buf := make([]byte, tt.n)
		n, err := rs.Read(buf)
		if err != tt.readerr {
			t.Errorf("%d. read = %v; want %v", i, err, tt.readerr)
			continue
		}
		got := string(buf[:n])
		if got != tt.want {
			t.Errorf("%d. got %q; want %q", i, got, tt.want)
		}
		assert.Equal(t, tt.len, rs.Len())
		assert.Equal(t, tt.size, int(rs.Size()))
	}
	n64, err := rs.Seek(0, io.SeekEnd)
	assert.NoError(t, err)
	assert.Equal(t, int64(10), n64)

	assert.NoError(t, rs.Close())

	n64, err = rs.Seek(0, 0)
	assert.Equal(t, io.ErrClosedPipe, err)
	assert.Empty(t, n64)

	b := make([]byte, 1)
	n, err := rs.Read(b)
	assert.Equal(t, io.ErrClosedPipe, err)
	assert.Empty(t, n)
	assert.Empty(t, b[0])
}

func TestSeekStream_TempFileBuffer(t *testing.T) {
	buffer, err := NewTempFileBuffer("", "imagor-")
	require.NoError(t, err)
	doSeekStreamTests(t, buffer)
}

func TestSeekStream_MemoryBuffer(t *testing.T) {
	doSeekStreamTests(t, NewMemoryBuffer(10))
}

func TestAsyncReadSeeker(t *testing.T) {
	buf := []byte("0123456789")
	rs := NewAsync(io.NopCloser(bytes.NewReader(buf)), int64(len(buf)))

	tests := []struct {
		off     int64
		seek    int
		n       int
		len     int
		size    int
		want    string
		wantpos int64
		readerr error
		seekerr string
	}{
		{seek: -1, n: 3, len: 7, size: 10, want: "012"},
		{seek: -1, n: 2, len: 5, size: 10, want: "34"},
		{seek: io.SeekCurrent, off: 1, n: 1, len: 3, size: 10, want: "6"},
		{seek: io.SeekCurrent, off: -1, n: 2, len: 2, size: 10, want: "67"},
		{seek: io.SeekStart, off: 2, n: 2, len: 6, size: 10, want: "23"},
		{seek: io.SeekEnd, off: -2, n: 20, size: 10, want: "89"},
		{seek: io.SeekStart, off: 20, n: 2, size: 10, readerr: io.EOF},
		{seek: io.SeekStart, off: -1, size: 10, seekerr: "invalid argument"},
	}

	for i, tt := range tests {
		if tt.seek >= 0 {
			pos, err := rs.Seek(tt.off, tt.seek)
			if tt.seekerr != "" {
				require.Error(t, err, i)
				assert.Contains(t, err.Error(), tt.seekerr, i)
			} else {
				require.NoError(t, err, i)
			}
			if tt.wantpos != 0 {
				assert.Equal(t, tt.wantpos, pos, i)
			}
		}
		readBuf := make([]byte, tt.n)
		n, err := rs.Read(readBuf)
		assert.ErrorIs(t, err, tt.readerr, i)
		assert.Equal(t, tt.want, string(readBuf[:n]), i)
		assert.Equal(t, tt.len, rs.Len(), i)
		assert.Equal(t, tt.size, int(rs.Size()), i)
	}

	assert.NoError(t, rs.Close())
	_, err := rs.Seek(0, io.SeekStart)
	assert.ErrorIs(t, err, io.ErrClosedPipe)
}

func TestAsyncReadSeeker_CloseAfterEOFClosesSourceOnce(t *testing.T) {
	source := newCountingReadCloser(bytes.NewReader([]byte("0123456789")))
	rs := NewAsync(source, 10)

	buf, err := io.ReadAll(rs)
	require.NoError(t, err)
	assert.Equal(t, "0123456789", string(buf))

	select {
	case <-source.closed:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for async source close")
	}

	assert.NoError(t, rs.Close())
	assert.Equal(t, int32(1), source.closeCount.Load())
	assert.NoError(t, rs.Close())
	assert.Equal(t, int32(1), source.closeCount.Load())
}

func TestAsyncReadSeeker_PropagatesReadErrorAfterBufferedBytes(t *testing.T) {
	boom := errors.New("boom")
	source := &scriptedReadCloser{steps: []readStep{
		{data: []byte("0123")},
		{data: []byte("45"), err: boom},
	}}
	rs := NewAsync(source, 6)
	t.Cleanup(func() {
		assert.NoError(t, rs.Close())
	})

	buf := make([]byte, 8)
	n, err := rs.Read(buf)
	assert.NoError(t, err)
	assert.Equal(t, "012345", string(buf[:n]))

	n, err = rs.Read(buf)
	assert.Zero(t, n)
	assert.ErrorIs(t, err, boom)
	assert.Equal(t, 0, rs.Len())
	assert.Equal(t, int64(6), rs.Size())
}

func TestAsyncReadSeeker_ShortSourceClampsLenAndSizeAfterEOF(t *testing.T) {
	rs := NewAsync(io.NopCloser(bytes.NewReader([]byte("012345"))), 10)
	t.Cleanup(func() {
		assert.NoError(t, rs.Close())
	})

	buf, err := io.ReadAll(rs)
	require.NoError(t, err)
	assert.Equal(t, "012345", string(buf))
	assert.Equal(t, 0, rs.Len())
	assert.Equal(t, int64(6), rs.Size())

	pos, err := rs.Seek(0, io.SeekEnd)
	require.NoError(t, err)
	assert.Equal(t, int64(6), pos)

	_, err = rs.Seek(0, io.SeekStart)
	require.NoError(t, err)
	buf, err = io.ReadAll(rs)
	require.NoError(t, err)
	assert.Equal(t, "012345", string(buf))
}

func TestAsyncReadSeeker_SeekAfterShortIntermediateReads(t *testing.T) {
	source := &scriptedReadCloser{steps: []readStep{
		{data: []byte("01")},
		{data: []byte("2")},
		{data: []byte("345")},
		{data: []byte("67")},
		{data: []byte("89")},
	}}
	rs := NewAsync(source, 10)
	t.Cleanup(func() {
		assert.NoError(t, rs.Close())
	})

	buf := make([]byte, 6)
	n, err := rs.Read(buf)
	require.NoError(t, err)
	assert.Equal(t, "012345", string(buf[:n]))

	_, err = rs.Seek(2, io.SeekStart)
	require.NoError(t, err)

	buf, err = io.ReadAll(rs)
	require.NoError(t, err)
	assert.Equal(t, "23456789", string(buf))

	_, err = rs.Seek(0, io.SeekStart)
	require.NoError(t, err)
	buf, err = io.ReadAll(rs)
	require.NoError(t, err)
	assert.Equal(t, "0123456789", string(buf))
}

func TestMemoryBuffer_Seek(t *testing.T) {
	r := NewMemoryBuffer(10)
	n, err := r.Write([]byte("0123456789"))
	assert.Equal(t, 10, n)
	assert.NoError(t, err)
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
		{seek: io.SeekStart, off: -1, seekerr: "invalid argument"},
		{seek: io.SeekStart, off: 1 << 33, wantpos: 1 << 33, readerr: io.EOF},
		{seek: io.SeekCurrent, off: 1, wantpos: 1<<33 + 1, readerr: io.EOF},
		{seek: io.SeekStart, n: 5, want: "01234"},
		{seek: io.SeekCurrent, n: 5, want: "56789"},
		{seek: io.SeekEnd, off: -1, n: 1, wantpos: 9, want: "9"},
	}

	for i, tt := range tests {
		pos, err := r.Seek(tt.off, tt.seek)
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
		n, err := r.Read(buf)
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
