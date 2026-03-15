package vipsconfig

import (
	"testing"

	"github.com/cshum/imagor"
	"github.com/cshum/imagor/config"
	"github.com/cshum/imagor/processor/vipsprocessor"
	"github.com/stretchr/testify/assert"
)

func TestWithVips(t *testing.T) {
	srv := config.CreateServer([]string{
		"-vips-max-animation-frames", "167",
		"-vips-disable-filters", "blur,watermark,rgb",
	}, WithVips)
	app := srv.App.(*imagor.Imagor)
	processor := app.Processors[0].(*vipsprocessor.Processor)
	assert.Equal(t, 167, processor.MaxAnimationFrames)
	assert.Equal(t, []string{"blur", "watermark", "rgb"}, processor.DisableFilters)
}

func TestWithVipsCacheFormat(t *testing.T) {
	tests := []struct {
		flag string
		want imagor.BlobType
		name string
	}{
		{"pixel", imagor.BlobTypeMemory, "pixel → BlobTypeMemory (raw pixels, default)"},
		{"", imagor.BlobTypeMemory, "empty → BlobTypeMemory (raw pixels, default)"},
		{"png", imagor.BlobTypePNG, "png → BlobTypePNG (lossless)"},
		{"webp", imagor.BlobTypeWEBP, "webp → BlobTypeWEBP (lossy)"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := []string{}
			if tt.flag != "" {
				args = append(args, "-vips-cache-format", tt.flag)
			}
			srv := config.CreateServer(args, WithVips)
			app := srv.App.(*imagor.Imagor)
			processor := app.Processors[0].(*vipsprocessor.Processor)
			assert.Equal(t, tt.want, processor.CacheFormat, tt.name)
		})
	}
}
