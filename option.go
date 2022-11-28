package imagor

import (
	"github.com/cshum/imagor/imagorpath"
	"go.uber.org/zap"
	"time"
)

type Option func(app *Imagor)

func WithOptions(options ...Option) Option {
	return func(app *Imagor) {
		for _, option := range options {
			if option != nil {
				option(app)
			}
		}
	}
}

func WithLogger(logger *zap.Logger) Option {
	return func(app *Imagor) {
		if logger != nil {
			app.Logger = logger
		}
	}
}

func WithLoaders(loaders ...Loader) Option {
	return func(app *Imagor) {
		app.Loaders = append(app.Loaders, loaders...)
	}
}

func WithStorages(savers ...Storage) Option {
	return func(app *Imagor) {
		app.Storages = append(app.Storages, savers...)
	}
}

func WithResultStorages(savers ...Storage) Option {
	return func(app *Imagor) {
		app.ResultStorages = append(app.ResultStorages, savers...)
	}
}

func WithProcessors(processors ...Processor) Option {
	return func(app *Imagor) {
		app.Processors = append(app.Processors, processors...)
	}
}

func WithRequestTimeout(timeout time.Duration) Option {
	return func(app *Imagor) {
		if timeout > 0 {
			app.RequestTimeout = timeout
		}
	}
}

func WithCacheHeaderTTL(ttl time.Duration) Option {
	return func(app *Imagor) {
		if ttl > 0 {
			app.CacheHeaderTTL = ttl
		}
	}
}

func WithCacheHeaderSWR(swr time.Duration) Option {
	return func(app *Imagor) {
		if swr > 0 {
			app.CacheHeaderSWR = swr
		}
	}
}

func WithCacheHeaderNoCache(nocache bool) Option {
	return func(app *Imagor) {
		if nocache {
			app.CacheHeaderTTL = 0
		}
	}
}

func WithLoadTimeout(timeout time.Duration) Option {
	return func(app *Imagor) {
		if timeout > 0 {
			app.LoadTimeout = timeout
		}
	}
}

func WithSaveTimeout(timeout time.Duration) Option {
	return func(app *Imagor) {
		if timeout > 0 {
			app.SaveTimeout = timeout
		}
	}
}

func WithProcessTimeout(timeout time.Duration) Option {
	return func(app *Imagor) {
		if timeout > 0 {
			app.ProcessTimeout = timeout
		}
	}
}

func WithProcessConcurrency(concurrency int64) Option {
	return func(app *Imagor) {
		if concurrency > 0 {
			app.ProcessConcurrency = concurrency
		}
	}
}

func WithProcessQueueSize(size int64) Option {
	return func(app *Imagor) {
		if size > 0 {
			app.ProcessQueueSize = size
		}
	}
}

func WithUnsafe(unsafe bool) Option {
	return func(app *Imagor) {
		app.Unsafe = unsafe
	}
}

func WithAutoWebP(enable bool) Option {
	return func(app *Imagor) {
		app.AutoWebP = enable
	}
}

func WithAutoAVIF(enable bool) Option {
	return func(app *Imagor) {
		app.AutoAVIF = enable
	}
}

func WithRetryQueryUnescape(enable bool) Option {
	return func(app *Imagor) {
		app.RetryQueryUnescape = enable
	}
}

func WithBasePathRedirect(url string) Option {
	return func(app *Imagor) {
		app.BasePathRedirect = url
	}
}

func WithBaseParams(params string) Option {
	return func(app *Imagor) {
		app.BaseParams = params
	}
}

func WithModifiedTimeCheck(enabled bool) Option {
	return func(app *Imagor) {
		app.ModifiedTimeCheck = enabled
	}
}

func WithDisableErrorBody(disabled bool) Option {
	return func(app *Imagor) {
		app.DisableErrorBody = disabled
	}
}

func WithDisableParamsEndpoint(disabled bool) Option {
	return func(app *Imagor) {
		app.DisableParamsEndpoint = disabled
	}
}

func WithDebug(debug bool) Option {
	return func(app *Imagor) {
		app.Debug = debug
	}
}

func WithResultStoragePathStyle(hasher imagorpath.ResultStorageHasher) Option {
	return func(app *Imagor) {
		if hasher != nil {
			app.ResultStoragePathStyle = hasher
		}
	}
}

func WithStoragePathStyle(hasher imagorpath.StorageHasher) Option {
	return func(app *Imagor) {
		if hasher != nil {
			app.StoragePathStyle = hasher
		}
	}
}

func WithSigner(signer imagorpath.Signer) Option {
	return func(app *Imagor) {
		if signer != nil {
			app.Signer = signer
		}
	}
}
