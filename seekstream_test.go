package imagor

import (
	"bytes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"strings"
	"testing"
)

func TestSeekStream(t *testing.T) {
	buf := []byte("0123456789")
	source := io.NopCloser(bytes.NewReader(buf))
	rs, err := NewSeekStream(source)
	require.NoError(t, err)

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
		{seek: io.SeekCurrent, off: 2, n: 2, size: 9, want: "78"},
		{seek: io.SeekStart, off: 2, n: 2, len: 5, size: 9, want: "23"},
		{seek: io.SeekEnd, off: -2, n: 20, size: 10, want: "89"},
		{seek: io.SeekStart, off: 20, n: 2, size: 10},
		{seek: io.SeekStart, off: 0, n: 20, size: 10, want: "0123456789"},
		{seek: io.SeekStart, off: 1, n: 1, len: 8, size: 10, want: "1"},
		{seek: io.SeekCurrent, off: 1, wantpos: 3, n: 2, len: 5, size: 10, want: "34"},
		{seek: io.SeekStart, off: -1, len: 10, size: 10, seekerr: "invalid argument"},
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
