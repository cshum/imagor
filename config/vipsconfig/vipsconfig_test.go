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
