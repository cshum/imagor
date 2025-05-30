package vipsprocessor

import (
	"go.uber.org/zap"
	"strings"
)

// Option Processor option
type Option func(v *Processor)

// WithFilter with filer option of name and FilterFunc pair
func WithFilter(name string, filter FilterFunc) Option {
	return func(v *Processor) {
		v.Filters[name] = filter
	}
}

// WithDisableBlur with disable blur option
func WithDisableBlur(disabled bool) Option {
	return func(v *Processor) {
		v.DisableBlur = disabled
	}
}

// WithDisableFilters with disable filters option
func WithDisableFilters(filters ...string) Option {
	return func(v *Processor) {
		for _, raw := range filters {
			splits := strings.Split(raw, ",")
			for _, name := range splits {
				name = strings.TrimSpace(name)
				if len(name) > 0 {
					v.DisableFilters = append(v.DisableFilters, name)
				}
			}
		}
	}
}

// WithMozJPEG with MozJPEG option. Require MozJPEG to be installed
func WithMozJPEG(enabled bool) Option {
	return func(v *Processor) {
		v.MozJPEG = enabled
	}
}

// WithStripMetadata with strip all metadata from image option
func WithStripMetadata(enabled bool) Option {
	return func(v *Processor) {
		v.StripMetadata = enabled
	}
}

// WithAvifSpeed with avif speed option
func WithAvifSpeed(avifSpeed int) Option {
	return func(v *Processor) {
		if avifSpeed >= 0 && avifSpeed <= 9 {
			v.AvifSpeed = avifSpeed
		}
	}
}

// WithMaxFilterOps with maximum number of filter operations option
func WithMaxFilterOps(num int) Option {
	return func(v *Processor) {
		if num != 0 {
			v.MaxFilterOps = num
		}
	}
}

// WithMaxAnimationFrames with maximum count of animation frames option
func WithMaxAnimationFrames(num int) Option {
	return func(v *Processor) {
		if num != 0 {
			v.MaxAnimationFrames = num
		}
	}
}

// WithConcurrency with libvips concurrency option
func WithConcurrency(num int) Option {
	return func(v *Processor) {
		if num != 0 {
			v.Concurrency = num
		}
	}
}

// WithMaxCacheFiles with libvips max cache files option
func WithMaxCacheFiles(num int) Option {
	return func(v *Processor) {
		if num > 0 {
			v.MaxCacheFiles = num
		}
	}
}

// WithMaxCacheSize with libvips max cache size option
func WithMaxCacheSize(num int) Option {
	return func(v *Processor) {
		if num > 0 {
			v.MaxCacheSize = num
		}
	}
}

// WithMaxCacheMem with libvips max cache mem option
func WithMaxCacheMem(num int) Option {
	return func(v *Processor) {
		if num > 0 {
			v.MaxCacheMem = num
		}
	}
}

// WithLogger with logger option
func WithLogger(logger *zap.Logger) Option {
	return func(v *Processor) {
		if logger != nil {
			v.Logger = logger
		}
	}
}

// WithDebug with debug option
func WithDebug(debug bool) Option {
	return func(v *Processor) {
		v.Debug = debug
	}
}

// WithMaxWidth with maximum width option
func WithMaxWidth(width int) Option {
	return func(v *Processor) {
		if width > 0 {
			v.MaxWidth = width
		}
	}
}

// WithMaxHeight with maximum height option
func WithMaxHeight(height int) Option {
	return func(v *Processor) {
		if height > 0 {
			v.MaxHeight = height
		}
	}
}

// WithMaxResolution with maximum resolution option
func WithMaxResolution(res int) Option {
	return func(v *Processor) {
		if res > 0 {
			v.MaxResolution = res
		}
	}
}

// WithForceBmpFallback force with BMP fallback
func WithForceBmpFallback() Option {
	return func(v *Processor) {
		v.FallbackFunc = v.bmpFallbackFunc
	}
}

// WithUnlimited with unlimited option that remove all denial of service limits
func WithUnlimited(unlimited bool) Option {
	return func(v *Processor) {
		v.Unlimited = unlimited
	}
}
