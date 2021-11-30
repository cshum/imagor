package vipsprocessor

type Option func(h *VipsProcessor)

func WithFilter(name string, filter FilterFunc) Option {
	return func(h *VipsProcessor) {
		h.Filters[name] = filter
	}
}
