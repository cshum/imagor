package vipsconfig

import (
	"flag"

	"github.com/cshum/imagor"
	"github.com/cshum/imagor/processor/vipsprocessor"
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
		vipsAvifSpeed = fs.Int("vips-avif-speed", 5,
			"VIPS avif speed, the lowest is at 0 and the fastest is at 9 (Default 5).")
		vipsStripMetadata = fs.Bool("vips-strip-metadata", false,
			"VIPS strips all metadata from the resulting image")
		vipsUnlimited = fs.Bool("vips-unlimited", false,
			"VIPS bypass image max resolution check and remove all denial of service limits")
		imagorWatermarkCacheSize = fs.Int64("imagor-watermark-cache-size", 0,
			"In-memory watermark cache size in bytes (0 to disable). e.g. 104857600 for 100MB")

		logger, isDebug = cb()
	)
	return imagor.WithProcessors(
		vipsprocessor.NewProcessor(
			vipsprocessor.WithMaxAnimationFrames(*vipsMaxAnimationFrames),
			vipsprocessor.WithDisableBlur(*vipsDisableBlur),
			vipsprocessor.WithDisableFilters(*vipsDisableFilters),
			vipsprocessor.WithConcurrency(*vipsConcurrency),
			vipsprocessor.WithMaxCacheFiles(*vipsMaxCacheFiles),
			vipsprocessor.WithMaxCacheMem(*vipsMaxCacheMem),
			vipsprocessor.WithMaxCacheSize(*vipsMaxCacheSize),
			vipsprocessor.WithMaxFilterOps(*vipsMaxFilterOps),
			vipsprocessor.WithMaxWidth(*vipsMaxWidth),
			vipsprocessor.WithMaxHeight(*vipsMaxHeight),
			vipsprocessor.WithMaxResolution(*vipsMaxResolution),
			vipsprocessor.WithMozJPEG(*vipsMozJPEG),
			vipsprocessor.WithAvifSpeed(*vipsAvifSpeed),
			vipsprocessor.WithStripMetadata(*vipsStripMetadata),
			vipsprocessor.WithUnlimited(*vipsUnlimited),
			vipsprocessor.WithWatermarkCacheSize(*imagorWatermarkCacheSize),
			vipsprocessor.WithLogger(logger),
			vipsprocessor.WithDebug(isDebug),
		),
	)
}
