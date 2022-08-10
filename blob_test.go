package imagor

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"os"
	"testing"
)

func TestBlobTypes(t *testing.T) {
	tests := []struct {
		name              string
		path              string
		contentType       string
		bytesType         BlobType
		supportsAnimation bool
	}{
		{
			name:        "jpeg",
			path:        "demo1.jpg",
			contentType: "image/jpeg",
			bytesType:   BlobTypeJPEG,
		},
		{
			name:        "png",
			path:        "gopher.png",
			contentType: "image/png",
			bytesType:   BlobTypePNG,
		},
		{
			name:        "tiff",
			path:        "gopher.tiff",
			contentType: "image/tiff",
			bytesType:   BlobTypeTIFF,
		},
		{
			name:              "gif",
			path:              "dancing-banana.gif",
			contentType:       "image/gif",
			bytesType:         BlobTypeGIF,
			supportsAnimation: true,
		},
		{
			name:              "webp",
			path:              "demo3.webp",
			contentType:       "image/webp",
			bytesType:         BlobTypeWEBP,
			supportsAnimation: true,
		},
		{
			name:        "avif",
			path:        "gopher-front.avif",
			contentType: "image/avif",
			bytesType:   BlobTypeAVIF,
		},
		{
			name:        "heif",
			path:        "gopher-front.heif",
			contentType: "image/heif",
			bytesType:   BlobTypeHEIF,
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
			assert.False(t, b.IsEmpty())
			assert.NotEmpty(t, b.Sniff())
			assert.NotEmpty(t, b.Size())
			require.NoError(t, b.Err())

			buf, err := b.ReadAll()
			require.NoError(t, err)
			require.NoError(t, b.Err())
			b = NewBlobFromBytes(buf)
			assert.Equal(t, tt.supportsAnimation, b.SupportsAnimation())
			assert.Equal(t, tt.contentType, b.ContentType())
			assert.Equal(t, tt.bytesType, b.BlobType())
			assert.False(t, b.IsEmpty())
			assert.NotEmpty(t, b.Sniff())
			assert.NotEmpty(t, b.Size())
			require.NoError(t, b.Err())
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

func TestNewJsonMarshalBlob(t *testing.T) {
	b := NewBlobFromJsonMarshal(map[string]string{
		"foo": "bar",
	})
	assert.Equal(t, b.BlobType(), BlobTypeJSON)
	assert.Equal(t, b.ContentType(), "application/json")
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
