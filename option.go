package imagor

import (
	cache "github.com/cshum/hybridcache"
	"go.uber.org/zap"
	"time"
)

type Option func(o *imagor)

func WithLogger(logger *zap.Logger) Option {
	return func(o *imagor) {
		o.Logger = logger
	}
}

func WithCache(c cache.Cache, ttl time.Duration) Option {
	return func(o *imagor) {
		o.Cache = c
		o.CacheTTL = ttl
	}
}

func WithLoaders(loaders ...Loader) Option {
	return func(o *imagor) {
		o.Loaders = append(o.Loaders, loaders...)
	}
}

func WithProcessors(processors ...Processor) Option {
	return func(o *imagor) {
		o.Processors = append(o.Processors, processors...)
	}
}

func WithStorages(storages ...Storage) Option {
	return func(o *imagor) {
		o.Storages = append(o.Storages, storages...)
	}
}

func WithTimeout(timeout time.Duration) Option {
	return func(o *imagor) {
		o.Timeout = timeout
	}
}

func WithUnsafe(unsafe bool) Option {
	return func(o *imagor) {
		o.Unsafe = unsafe
	}
}

func WithSecret(secret string) Option {
	return func(o *imagor) {
		o.Secret = secret
	}
}
