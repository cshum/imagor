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

	b := make([]byte, 5)
	n, err := rs.Read(b)
	assert.NoError(t, err)
	assert.Equal(t, 5, n)
	assert.Equal(t, []byte("01234"), b)

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
		//{seek: io.SeekStart, off: 1 << 33, wantpos: 1 << 33, readerr: io.EOF},
		//{seek: io.SeekCurrent, off: 1, wantpos: 1<<33 + 1, readerr: io.EOF},
		{seek: io.SeekStart, off: 1 << 33, wantpos: 1 << 33},
		{seek: io.SeekCurrent, off: 1, wantpos: 1<<33 + 1},
		{seek: io.SeekStart, n: 5, want: "01234"},
		{seek: io.SeekCurrent, n: 5, want: "56789"},
		{seek: io.SeekEnd, off: -1, n: 1, wantpos: 9, want: "9"},
	}

	for i, tt := range tests {
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
	}
	assert.Equal(t, int64(10), rs.Size())
	assert.NoError(t, rs.Close())
	b = make([]byte, 1)
	n64, err := rs.Seek(0, 0)
	assert.Equal(t, io.ErrClosedPipe, err)
	assert.Empty(t, n64)
	n, err = rs.Read(b)
	assert.Equal(t, io.ErrClosedPipe, err)
	assert.Empty(t, n)
	assert.Empty(t, b[0])
}
