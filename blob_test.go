package imagor

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func doTestBlobReaders(t *testing.T, b *Blob, buf []byte) {
	r, size, err := b.NewReader()
	assert.NotNil(t, r)
	assert.NotEmpty(t, size)
	assert.NoError(t, err)

	buf2, err := io.ReadAll(r)
	require.NoError(t, err)
	assert.NotEmpty(t, buf2)
	assert.Equal(t, buf, buf2, "bytes not equal")

	rs, size, err := b.NewReadSeeker()
	assert.NotNil(t, rs)
	assert.NotEmpty(t, size)
	assert.NoError(t, err)
	defer rs.Close()

	buf3, err := io.ReadAll(rs)
	require.NoError(t, err)
	assert.NotEmpty(t, buf3)
	assert.Equal(t, buf, buf3, "bytes not equal")

	for i := 0; i < 3; i++ {
		_, err = rs.Seek(0, io.SeekStart)
		require.NoError(t, err)
		buf4, err := io.ReadAll(rs)
		require.NoError(t, err)
		assert.NotEmpty(t, buf4)
		assert.Equal(t, buf, buf4, "bytes not equal")
	}
}

func TestBlobTypes(t *testing.T) {
	tests := []struct {
		name              string
		path              string
		contentType       string
		extension         string
		bytesType         BlobType
		supportsAnimation bool
	}{
		{
			name:        "jpeg",
			path:        "demo1.jpg",
			contentType: "image/jpeg",
			extension:   ".jpg",
			bytesType:   BlobTypeJPEG,
		},
		{
			name:        "png",
			path:        "gopher.png",
			contentType: "image/png",
			extension:   ".png",
			bytesType:   BlobTypePNG,
		},
		{
			name:        "tiff",
			path:        "gopher.tiff",
			contentType: "image/tiff",
			extension:   ".tiff",
			bytesType:   BlobTypeTIFF,
		},
		{
			name:              "gif",
			path:              "dancing-banana.gif",
			contentType:       "image/gif",
			extension:         ".gif",
			bytesType:         BlobTypeGIF,
			supportsAnimation: true,
		},
		{
			name:              "webp",
			path:              "demo3.webp",
			contentType:       "image/webp",
			extension:         ".webp",
			bytesType:         BlobTypeWEBP,
			supportsAnimation: true,
		},
		{
			name:        "jxl",
			path:        "jxl-isobmff.jxl",
			contentType: "image/jxl",
			extension:   ".jxl",
			bytesType:   BlobTypeJXL,
		},
		{
			name:        "avif",
			path:        "gopher-front.avif",
			contentType: "image/avif",
			extension:   ".avif",
			bytesType:   BlobTypeAVIF,
		},
		{
			name:        "heif",
			path:        "gopher-front.heif",
			contentType: "image/heif",
			extension:   ".heif",
			bytesType:   BlobTypeHEIF,
		},
		{
			name:        "jp2",
			path:        "gopher.jp2",
			contentType: "image/jp2",
			extension:   ".jp2",
			bytesType:   BlobTypeJP2,
		},
		{
			name:        "pdf",
			path:        "sample.pdf",
			contentType: "application/pdf",
			extension:   ".pdf",
			bytesType:   BlobTypePDF,
		},
		{
			name:        "bmp",
			path:        "bmp_24.bmp",
			contentType: "image/bmp",
			extension:   ".bmp",
			bytesType:   BlobTypeBMP,
		},
		{
			name:        "svg",
			path:        "test.svg",
			contentType: "image/svg+xml",
			extension:   ".svg",
			bytesType:   BlobTypeSVG,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filepath := "testdata/" + tt.path
			b := NewBlobFromFile(filepath, func(info os.FileInfo) error {
				// noop
				return nil
			})
			assert.Equal(t, tt.supportsAnimation, b.SupportsAnimation())
			assert.Equal(t, tt.contentType, b.ContentType())
			assert.Equal(t, filepath, b.FilePath())
			assert.Equal(t, tt.bytesType, b.BlobType())
			assert.Equal(t, tt.extension, getExtension(b.BlobType()))
			assert.False(t, b.IsEmpty())
			assert.NotEmpty(t, b.Sniff())
			assert.NotEmpty(t, b.Size())
			require.NoError(t, b.Err())

			buf, err := b.ReadAll()
			require.NoError(t, err)

			doTestBlobReaders(t, b, buf)

			b = NewBlobFromBytes(buf)
			assert.Equal(t, tt.supportsAnimation, b.SupportsAnimation())
			assert.Equal(t, tt.contentType, b.ContentType())
			assert.Equal(t, tt.bytesType, b.BlobType())
			assert.False(t, b.IsEmpty())
			assert.NotEmpty(t, b.Sniff())
			assert.NotEmpty(t, b.Size())
			require.NoError(t, b.Err())

			doTestBlobReaders(t, b, buf)

			b = NewBlob(func() (reader io.ReadCloser, size int64, err error) {
				return io.NopCloser(bytes.NewReader(buf)), int64(len(buf)), nil
			})
			assert.Equal(t, tt.supportsAnimation, b.SupportsAnimation())
			assert.Equal(t, tt.contentType, b.ContentType())
			assert.Equal(t, tt.bytesType, b.BlobType())
			assert.False(t, b.IsEmpty())
			assert.NotEmpty(t, b.Sniff())
			assert.NotEmpty(t, b.Size())
			require.NoError(t, b.Err())

			doTestBlobReaders(t, b, buf)

			b = NewBlob(func() (reader io.ReadCloser, size int64, err error) {
				// unknown size to force discard fanout
				return io.NopCloser(bytes.NewReader(buf)), 0, nil
			})
			assert.Equal(t, tt.supportsAnimation, b.SupportsAnimation())
			assert.Equal(t, tt.contentType, b.ContentType())
			assert.Equal(t, tt.bytesType, b.BlobType())
			assert.False(t, b.IsEmpty())
			assert.NotEmpty(t, b.Sniff())
			assert.Empty(t, b.Size())
			require.NoError(t, b.Err())

			r, size, err := b.NewReader()
			assert.NotNil(t, r)
			assert.Empty(t, size)
			assert.NoError(t, err)

			buf2, err := io.ReadAll(r)
			require.NoError(t, err)
			assert.NotEmpty(t, buf2)
			assert.Equal(t, buf, buf2, "bytes not equal")

			rs, size, err := b.NewReadSeeker()
			require.NoError(t, err)
			defer rs.Close()
			buf3, err := io.ReadAll(rs)
			require.NoError(t, err)
			assert.NotEmpty(t, buf3)
			assert.Equal(t, buf, buf3, "bytes not equal")

			for i := 0; i < 3; i++ {
				_, err = rs.Seek(0, io.SeekStart)
				require.NoError(t, err)
				buf4, err := io.ReadAll(rs)
				require.NoError(t, err)
				assert.NotEmpty(t, buf4)
				assert.Equal(t, buf, buf4, "bytes not equal")
			}
		})
	}
}

func TestNewEmptyBlob(t *testing.T) {
	b := NewBlobFromBytes([]byte{})
	assert.Empty(t, b.Sniff())
	assert.True(t, b.IsEmpty())
	assert.Equal(t, BlobTypeEmpty, b.BlobType())

	buf, err := b.ReadAll()
	assert.NoError(t, err)
	assert.Empty(t, buf)

	b = NewEmptyBlob()
	assert.Equal(t, BlobTypeEmpty, b.BlobType())
	assert.True(t, b.IsEmpty())
	assert.Empty(t, b.Sniff())
	assert.Empty(t, b.Size())

	buf, err = b.ReadAll()
	assert.NoError(t, err)
	assert.Empty(t, buf)

	r, size, err := b.NewReader()
	assert.NoError(t, err)
	assert.Empty(t, size)
	assert.Empty(t, b.Size())

	buf, err = io.ReadAll(r)
	assert.NoError(t, err)
	assert.Empty(t, buf)

	f, err := os.CreateTemp("", "tmpfile-")
	require.NoError(t, err)
	defer f.Close()
	defer os.Remove(f.Name())
	fmt.Println(f.Name())
	b = NewBlobFromFile(f.Name())
	assert.Equal(t, BlobTypeEmpty, b.BlobType())
	assert.True(t, b.IsEmpty())
	assert.Empty(t, b.Sniff())
	assert.Empty(t, b.Size())

	buf, err = b.ReadAll()
	assert.NoError(t, err)
	assert.Empty(t, buf)
}

func TestNewBlobFromMemory(t *testing.T) {
	b := NewEmptyBlob()
	data, width, height, bands, ok := b.Memory()
	assert.Empty(t, data)
	assert.Empty(t, width)
	assert.Empty(t, height)
	assert.Empty(t, bands)
	assert.False(t, ok)
	assert.True(t, b.IsEmpty())
	b = NewBlobFromMemory([]byte{167, 169}, 2, 1, 1)
	assert.Equal(t, BlobTypeMemory, b.BlobType())
	assert.False(t, b.IsEmpty())
	data, width, height, bands, ok = b.Memory()
	assert.Equal(t, []byte{167, 169}, data)
	assert.Equal(t, 2, width)
	assert.Equal(t, 1, height)
	assert.Equal(t, 1, bands)
	assert.True(t, ok)
}

func TestNewJsonMarshalBlob(t *testing.T) {
	b := NewBlobFromJsonMarshal(map[string]string{
		"foo": "bar",
	})
	assert.Equal(t, BlobTypeJSON, b.BlobType())
	assert.Equal(t, "application/json", b.ContentType())
	assert.Equal(t, `{"foo":"bar"}`, string(b.Sniff()))
	buf, _ := b.ReadAll()
	assert.Equal(t, `{"foo":"bar"}`, string(buf))
}

func TestBlobOverrideContentType(t *testing.T) {
	b := NewBlobFromFile("testdata/demo1.jpg")
	b.SetContentType("foo/bar")
	assert.Equal(t, BlobTypeJPEG, b.BlobType())
	assert.Equal(t, "foo/bar", b.ContentType())
}

func TestBlobJsonBytes(t *testing.T) {
	b := NewBlobFromBytes([]byte(`{"foo": "bar"}`))
	assert.Equal(t, BlobTypeJSON, b.BlobType())
	assert.Equal(t, "application/json", b.ContentType())
	assert.Equal(t, ".json", getExtension(b.BlobType()))
}

type readerFunc func(p []byte) (n int, err error)

func (rf readerFunc) Read(p []byte) (n int, err error) { return rf(p) }

func TestBlobCreateError(t *testing.T) {
	e := errors.New("some error")
	b := NewBlob(func() (reader io.ReadCloser, size int64, err error) {
		return nil, 0, e
	})
	assert.Equal(t, e, b.Err())
	buf, err := b.ReadAll()
	assert.Empty(t, buf)
	assert.Equal(t, e, err)
}

func TestBlobReaderError(t *testing.T) {
	e := errors.New("some error")
	buf, err := os.ReadFile("testdata/demo1.jpg")
	require.NoError(t, err)
	var called int
	b := NewBlob(func() (reader io.ReadCloser, size int64, err error) {
		return io.NopCloser(readerFunc(func(p []byte) (n int, err error) {
			if called > 4 {
				return 0, e
			}
			called++
			if len(p) > 100 {
				p = p[:100]
			}
			n = copy(p, buf)
			buf = buf[n:]
			return
		})), int64(len(buf)), nil
	})
	assert.Equal(t, e, b.Err())
	assert.Equal(t, 500, len(b.Sniff()))
	buf, err = b.ReadAll()
	assert.Equal(t, 500, len(buf))
	assert.Equal(t, e, err)
}
