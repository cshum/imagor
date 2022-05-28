package imagor

import (
	"go.uber.org/zap"
	"time"
)

type Option func(o *Imagor)

func WithLogger(logger *zap.Logger) Option {
	return func(o *Imagor) {
		if logger != nil {
			o.Logger = logger
		}
	}
}

func WithLoaders(loaders ...Loader) Option {
	return func(o *Imagor) {
		o.Loaders = append(o.Loaders, loaders...)
	}
}

func WithStorages(savers ...Storage) Option {
	return func(o *Imagor) {
		o.Storages = append(o.Storages, savers...)
	}
}

func WithResultStorages(savers ...Storage) Option {
	return func(o *Imagor) {
		o.ResultStorages = append(o.ResultStorages, savers...)
	}
}

func WithProcessors(processors ...Processor) Option {
	return func(o *Imagor) {
		o.Processors = append(o.Processors, processors...)
	}
}

func WithRequestTimeout(timeout time.Duration) Option {
	return func(o *Imagor) {
		if timeout > 0 {
			o.RequestTimeout = timeout
		}
	}
}

func WithCacheHeaderTTL(ttl time.Duration) Option {
	return func(o *Imagor) {
		if ttl > 0 {
			o.CacheHeaderTTL = ttl
		} else if ttl == -1 {
			o.CacheHeaderTTL = 0
		}
	}
}

func WithLoadTimeout(timeout time.Duration) Option {
	return func(o *Imagor) {
		if timeout > 0 {
			o.LoadTimeout = timeout
		}
	}
}

func WithSaveTimeout(timeout time.Duration) Option {
	return func(o *Imagor) {
		if timeout > 0 {
			o.SaveTimeout = timeout
		}
	}
}

func WithProcessTimeout(timeout time.Duration) Option {
	return func(o *Imagor) {
		if timeout > 0 {
			o.ProcessTimeout = timeout
		}
	}
}

func WithProcessConcurrency(concurrency int64) Option {
	return func(o *Imagor) {
		if concurrency > 0 {
			o.ProcessConcurrency = concurrency
		}
	}
}

func WithUnsafe(unsafe bool) Option {
	return func(o *Imagor) {
		o.Unsafe = unsafe
	}
}

func WithAutoWebP(enable bool) Option {
	return func(o *Imagor) {
		o.AutoWebP = enable
	}
}

func WithAutoAVIF(enable bool) Option {
	return func(o *Imagor) {
		o.AutoAVIF = enable
	}
}

func WithSecret(secret string) Option {
	return func(o *Imagor) {
		o.Secret = secret
	}
}

func WithBasePathRedirect(url string) Option {
	return func(o *Imagor) {
		o.BasePathRedirect = url
	}
}

func WithModifiedTimeCheck(enabled bool) Option {
	return func(o *Imagor) {
		o.ModifiedTimeCheck = enabled
	}
}

func WithDebug(debug bool) Option {
	return func(o *Imagor) {
		o.Debug = debug
	}
}
