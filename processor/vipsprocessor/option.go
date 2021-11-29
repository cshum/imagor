package vipsprocessor

type Option func(h *vipsProcessor)

func WithFilter(name string, filter FilterFunc) Option {
	return func(h *vipsProcessor) {
		h.Filters[name] = filter
	}
}
