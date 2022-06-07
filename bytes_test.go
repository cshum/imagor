package imagor

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"testing"
)

func TestBytesTypes(t *testing.T) {
	tests := []struct {
		name              string
		path              string
		contentType       string
		bytesType         BytesType
		supportsAnimation bool
	}{
		{
			name:        "jpeg",
			path:        "demo1.jpg",
			contentType: "image/jpeg",
			bytesType:   BytesTypeJPEG,
		},
		{
			name:        "png",
			path:        "gopher.png",
			contentType: "image/png",
			bytesType:   BytesTypePNG,
		},
		{
			name:              "gif",
			path:              "dancing-banana.gif",
			contentType:       "image/gif",
			bytesType:         BytesTypeGIF,
			supportsAnimation: true,
		},
		{
			name:              "webp",
			path:              "demo3.webp",
			contentType:       "image/webp",
			bytesType:         BytesTypeWEBP,
			supportsAnimation: true,
		},
		{
			name:        "avif",
			path:        "gopher-front.avif",
			contentType: "image/avif",
			bytesType:   BytesTypeAVIF,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := NewBytesFilePath("testdata/" + tt.path)
			assert.Equal(t, tt.supportsAnimation, b.SupportsAnimation())
			assert.Equal(t, tt.contentType, b.ContentType())
			assert.Equal(t, tt.bytesType, b.BytesType())
			assert.False(t, b.IsEmpty())

			buf, err := b.ReadAll()
			require.NoError(t, err)
			b = NewBytes(buf)
			assert.Equal(t, tt.supportsAnimation, b.SupportsAnimation())
			assert.Equal(t, tt.contentType, b.ContentType())
			assert.Equal(t, tt.bytesType, b.BytesType())
			assert.False(t, b.IsEmpty())
		})
	}
}

func TestNewBytesEmpty(t *testing.T) {
	b := NewBytes([]byte{})
	buf, err := b.ReadAll()
	assert.NoError(t, err)
	assert.Empty(t, buf)
	assert.Equal(t, BytesTypeEmpty, b.BytesType())
	assert.True(t, b.IsEmpty())

	b = NewEmptyBytes()
	buf, err = b.ReadAll()
	assert.NoError(t, err)
	assert.Empty(t, buf)
	assert.Equal(t, BytesTypeEmpty, b.BytesType())
	assert.True(t, b.IsEmpty())

	f, err := os.CreateTemp("", "tmpfile-")
	require.NoError(t, err)
	defer f.Close()
	defer os.Remove(f.Name())
	fmt.Println(f.Name())
	b = NewBytesFilePath(f.Name())
	buf, err = b.ReadAll()
	assert.NoError(t, err)
	assert.Empty(t, buf)
	assert.Equal(t, BytesTypeEmpty, b.BytesType())
	assert.True(t, b.IsEmpty())
}
