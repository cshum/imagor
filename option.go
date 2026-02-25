package imagor

import (
	"time"

	"github.com/cshum/imagor/imagorpath"
	"go.uber.org/zap"
)

// Option imagor option
type Option func(app *Imagor)

// WithOptions with nested options
func WithOptions(options ...Option) Option {
	return func(app *Imagor) {
		for _, option := range options {
			if option != nil {
				option(app)
			}
		}
	}
}

// WithLogger with logger option
func WithLogger(logger *zap.Logger) Option {
	return func(app *Imagor) {
		if logger != nil {
			app.Logger = logger
		}
	}
}

// WithLoaders with loaders option
func WithLoaders(loaders ...Loader) Option {
	return func(app *Imagor) {
		app.Loaders = append(app.Loaders, loaders...)
	}
}

// WithStorages with storages option
func WithStorages(savers ...Storage) Option {
	return func(app *Imagor) {
		app.Storages = append(app.Storages, savers...)
	}
}

// WithResultStorages with result storages option
func WithResultStorages(savers ...Storage) Option {
	return func(app *Imagor) {
		app.ResultStorages = append(app.ResultStorages, savers...)
	}
}

// WithProcessors with processor option
func WithProcessors(processors ...Processor) Option {
	return func(app *Imagor) {
		app.Processors = append(app.Processors, processors...)
	}
}

// WithRequestTimeout with request timeout option
func WithRequestTimeout(timeout time.Duration) Option {
	return func(app *Imagor) {
		if timeout > 0 {
			app.RequestTimeout = timeout
		}
	}
}

// WithCacheHeaderTTL with browser cache header ttl option
func WithCacheHeaderTTL(ttl time.Duration) Option {
	return func(app *Imagor) {
		if ttl > 0 {
			app.CacheHeaderTTL = ttl
		}
	}
}

// WithCacheHeaderSWR with browser cache header swr option
func WithCacheHeaderSWR(swr time.Duration) Option {
	return func(app *Imagor) {
		if swr > 0 {
			app.CacheHeaderSWR = swr
		}
	}
}

// WithCacheHeaderNoCache with browser cache header no-cache option
func WithCacheHeaderNoCache(nocache bool) Option {
	return func(app *Imagor) {
		if nocache {
			app.CacheHeaderTTL = 0
		}
	}
}

// WithLoadTimeout with load timeout option for loader and storage
func WithLoadTimeout(timeout time.Duration) Option {
	return func(app *Imagor) {
		if timeout > 0 {
			app.LoadTimeout = timeout
		}
	}
}

// WithSaveTimeout with save timeout option for storage
func WithSaveTimeout(timeout time.Duration) Option {
	return func(app *Imagor) {
		if timeout > 0 {
			app.SaveTimeout = timeout
		}
	}
}

// WithProcessTimeout with process timeout option for processor
func WithProcessTimeout(timeout time.Duration) Option {
	return func(app *Imagor) {
		if timeout > 0 {
			app.ProcessTimeout = timeout
		}
	}
}

// WithProcessConcurrency maximum number of processor call to be executed simultaneously.
func WithProcessConcurrency(concurrency int64) Option {
	return func(app *Imagor) {
		if concurrency > 0 {
			app.ProcessConcurrency = concurrency
		}
	}
}

// WithProcessQueueSize maximum number of processor call that can be put to a queue
func WithProcessQueueSize(size int64) Option {
	return func(app *Imagor) {
		if size > 0 {
			app.ProcessQueueSize = size
		}
	}
}

// WithUnsafe with unsafe option
func WithUnsafe(unsafe bool) Option {
	return func(app *Imagor) {
		app.Unsafe = unsafe
	}
}

// WithAutoWebP with auto WebP option based on browser Accept header
func WithAutoWebP(enable bool) Option {
	return func(app *Imagor) {
		app.AutoWebP = enable
	}
}

// WithAutoAVIF experimental with auto AVIF option based on browser Accept header
func WithAutoAVIF(enable bool) Option {
	return func(app *Imagor) {
		app.AutoAVIF = enable
	}
}

// WithAutoJPEG with auto JPEG option when JPEG or no specific format is requested
func WithAutoJPEG(enable bool) Option {
	return func(app *Imagor) {
		app.AutoJPEG = enable
	}
}

// WithBasePathRedirect with base path redirect option
func WithBasePathRedirect(url string) Option {
	return func(app *Imagor) {
		app.BasePathRedirect = url
	}
}

// WithBaseParams with base params string option
func WithBaseParams(params string) Option {
	return func(app *Imagor) {
		app.BaseParams = params
	}
}

// WithModifiedTimeCheck with option for modified time check of storage against result storage
func WithModifiedTimeCheck(enabled bool) Option {
	return func(app *Imagor) {
		app.ModifiedTimeCheck = enabled
	}
}

// WithDisableErrorBody with disable error body option, resulting empty response on error
func WithDisableErrorBody(disabled bool) Option {
	return func(app *Imagor) {
		app.DisableErrorBody = disabled
	}
}

// WithDisableParamsEndpoint with disable imagor /params endpoint
func WithDisableParamsEndpoint(disabled bool) Option {
	return func(app *Imagor) {
		app.DisableParamsEndpoint = disabled
	}
}

// WithDebug with debug option
func WithDebug(debug bool) Option {
	return func(app *Imagor) {
		app.Debug = debug
	}
}

// WithResultStoragePathStyle with result storage path style hasher option
func WithResultStoragePathStyle(hasher imagorpath.ResultStorageHasher) Option {
	return func(app *Imagor) {
		if hasher != nil {
			app.ResultStoragePathStyle = hasher
		}
	}
}

// WithStoragePathStyle with storage path style hasher option
func WithStoragePathStyle(hasher imagorpath.StorageHasher) Option {
	return func(app *Imagor) {
		if hasher != nil {
			app.StoragePathStyle = hasher
		}
	}
}

// WithSigner with URL signature signer option
func WithSigner(signer imagorpath.Signer) Option {
	return func(app *Imagor) {
		if signer != nil {
			app.Signer = signer
		}
	}
}

// WithEnablePostRequests with enable POST requests option
func WithEnablePostRequests(enable bool) Option {
	return func(app *Imagor) {
		app.EnablePostRequests = enable
	}
}

// WithResponseRawOnError with response raw on error option
func WithResponseRawOnError(enabled bool) Option {
	return func(app *Imagor) {
		app.ResponseRawOnError = enabled
	}
}
