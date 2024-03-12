package vips

import (
	"github.com/stretchr/testify/require"
	"os"
	"testing"
)

func TestDetermineImageType(t *testing.T) {
	tests := []struct {
		name      string
		path      string
		imageType ImageType
	}{
		{
			name:      "jpeg",
			path:      "demo1.jpg",
			imageType: ImageTypeJPEG,
		},
		{
			name:      "png",
			path:      "gopher.png",
			imageType: ImageTypePNG,
		},
		{
			name:      "tiff",
			path:      "gopher.tiff",
			imageType: ImageTypeTIFF,
		},
		{
			name:      "heif",
			path:      "gopher-front.heif",
			imageType: ImageTypeHEIF,
		},
		{
			name:      "gif",
			path:      "dancing-banana.gif",
			imageType: ImageTypeGIF,
		},
		{
			name:      "webp",
			path:      "demo3.webp",
			imageType: ImageTypeWEBP,
		},
		{
			name:      "jp2",
			path:      "gopher.jp2",
			imageType: ImageTypeJP2K,
		},
		{
			name:      "pdf",
			path:      "sample.pdf",
			imageType: ImageTypePDF,
		},
		{
			name:      "svg",
			path:      "test.svg",
			imageType: ImageTypeSVG,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filepath := "../testdata/" + tt.path
			buf, err := os.ReadFile(filepath)
			require.NoError(t, err)
			img, err := LoadImageFromBuffer(buf, nil)
			require.NoError(t, err)
			require.Equal(t, tt.imageType, img.Format())
		})
	}
}
