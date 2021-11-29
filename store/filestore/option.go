package filestore

type Option func(h *fileStore)

func WithBaseURI(baseURI string) Option {
	return func(s *fileStore) {
		s.BaseURI = baseURI
	}
}

func WithBlacklists(pattern ...string) Option {
	return func(s *fileStore) {
		s.Blacklists = append(s.Blacklists, pattern...)
	}
}
