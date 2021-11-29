package vipsprocessor

type Option func(h *vipsProcessor)

func New(options ...Option) *vipsProcessor {
	v := &vipsProcessor{
		Filters: map[string]FilterFunc{
			"watermark":    watermark,
			"round_corner": roundCorner,
			"rotate":       rotate,
			"grayscale":    grayscale,
			"brightness":   brightness,
			"contrast":     contrast,
			"hue":          hue,
			"saturation":   saturation,
			"rgb":          rgb,
			"blur":         blur,
			"sharpen":      sharpen,
			"strip_icc":    stripIcc,
			"strip_exif":   stripExif,
		},
	}
	for _, option := range options {
		option(v)
	}
	return v
}

func WithFilter(name string, filter FilterFunc) Option {
	return func(h *vipsProcessor) {
		h.Filters[name] = filter
	}
}
