package vipsconfig

import (
	"flag"
	"github.com/cshum/imagor"
	"github.com/cshum/imagor/vips"
	"go.uber.org/zap"
)

// WithVips with libvips processor config option
func WithVips(fs *flag.FlagSet, cb func() (*zap.Logger, bool)) imagor.Option {
	var (
		vipsDisableBlur = fs.Bool("vips-disable-blur", false,
			"VIPS disable blur operations for vips processor")
		vipsMaxAnimationFrames = fs.Int("vips-max-animation-frames", -1,
			"VIPS maximum number of animation frames to be loaded. Set 1 to disable animation, -1 for unlimited")
		vipsDisableFilters = fs.String("vips-disable-filters", "",
			"VIPS disable filters by csv e.g. blur,watermark,rgb")
		vipsMaxFilterOps = fs.Int("vips-max-filter-ops", -1,
			"VIPS maximum number of filter operations allowed. Set -1 for unlimited")
		vipsConcurrency = fs.Int("vips-concurrency", 1,
			"VIPS concurrency. Set -1 to be the number of CPU cores")
		vipsMaxCacheFiles = fs.Int("vips-max-cache-files", 0,
			"VIPS max cache files")
		vipsMaxCacheSize = fs.Int("vips-max-cache-size", 0,
			"VIPS max cache size")
		vipsMaxCacheMem = fs.Int("vips-max-cache-mem", 0,
			"VIPS max cache mem")
		vipsMaxWidth = fs.Int("vips-max-width", 0,
			"VIPS max image width")
		vipsMaxHeight = fs.Int("vips-max-height", 0,
			"VIPS max image height")
		vipsMaxResolution = fs.Int("vips-max-resolution", 0,
			"VIPS max image resolution")
		vipsMozJPEG = fs.Bool("vips-mozjpeg", false,
			"VIPS enable maximum compression with MozJPEG. Requires mozjpeg to be installed")

		logger, isDebug = cb()
	)
	return imagor.WithProcessors(
		vips.NewProcessor(
			vips.WithMaxAnimationFrames(*vipsMaxAnimationFrames),
			vips.WithDisableBlur(*vipsDisableBlur),
			vips.WithDisableFilters(*vipsDisableFilters),
			vips.WithConcurrency(*vipsConcurrency),
			vips.WithMaxCacheFiles(*vipsMaxCacheFiles),
			vips.WithMaxCacheMem(*vipsMaxCacheMem),
			vips.WithMaxCacheSize(*vipsMaxCacheSize),
			vips.WithMaxFilterOps(*vipsMaxFilterOps),
			vips.WithMaxWidth(*vipsMaxWidth),
			vips.WithMaxHeight(*vipsMaxHeight),
			vips.WithMaxResolution(*vipsMaxResolution),
			vips.WithMozJPEG(*vipsMozJPEG),
			vips.WithLogger(logger),
			vips.WithDebug(isDebug),
		),
	)
}
