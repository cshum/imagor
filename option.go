package imagor

import (
	cache "github.com/cshum/hybridcache"
	"go.uber.org/zap"
	"time"
)

type Option func(o *Imagor)

func WithLogger(logger *zap.Logger) Option {
	return func(o *Imagor) {
		o.Logger = logger
	}
}

func WithCache(c cache.Cache, ttl time.Duration) Option {
	return func(o *Imagor) {
		o.Cache = c
		o.CacheTTL = ttl
	}
}

func WithLoaders(loaders ...Loader) Option {
	return func(o *Imagor) {
		o.Loaders = append(o.Loaders, loaders...)
	}
}

func WithProcessors(processors ...Processor) Option {
	return func(o *Imagor) {
		o.Processors = append(o.Processors, processors...)
	}
}

func WithStorages(storages ...Storage) Option {
	return func(o *Imagor) {
		o.Storages = append(o.Storages, storages...)
	}
}

func WithRequestTimeout(timeout time.Duration) Option {
	return func(o *Imagor) {
		if timeout > 0 {
			o.RequestTimeout = timeout
		}
	}
}

func WithUnsafe(unsafe bool) Option {
	return func(o *Imagor) {
		o.Unsafe = unsafe
	}
}

func WithSecret(secret string) Option {
	return func(o *Imagor) {
		o.Secret = secret
	}
}
