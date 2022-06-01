package imagor

import (
	"github.com/stretchr/testify/assert"
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
		})
	}
}
