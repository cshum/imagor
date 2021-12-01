package vipsprocessor

type Option func(h *VipsProcessor)

func WithFilter(name string, filter FilterFunc) Option {
	return func(h *VipsProcessor) {
		h.Filters[name] = filter
	}
}

func WithDisableBlur(disableBlur bool) Option {
	return func(h *VipsProcessor) {
		h.DisableBlur = disableBlur
	}
}

func WithoutFilter(names ...string) Option {
	return func(h *VipsProcessor) {
		for _, name := range names {
			delete(h.Filters, name)
		}
	}
}
