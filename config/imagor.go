package config

import (
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"flag"
	"github.com/cshum/imagor"
	"github.com/cshum/imagor/imagorpath"
	"go.uber.org/zap"
	"strings"
	"time"
)

type Callback func() (logger *zap.Logger, isDebug bool)

type Func func(fs *flag.FlagSet, cb Callback) imagor.Option

func NewImagor(fs *flag.FlagSet, cb Callback, funcs ...Func) *imagor.Imagor {
	var (
		imagorSecret = fs.String("imagor-secret", "",
			"Secret key for signing Imagor URL")
		imagorUnsafe = fs.Bool("imagor-unsafe", false,
			"Unsafe Imagor that does not require URL signature. Prone to URL tampering")
		imagorAutoWebP = fs.Bool("imagor-auto-webp", false,
			"Output WebP format automatically if browser supports")
		imagorAutoAVIF = fs.Bool("imagor-auto-avif", false,
			"Output AVIF format automatically if browser supports (experimental)")
		imagorRequestTimeout = fs.Duration("imagor-request-timeout",
			time.Second*30, "Timeout for performing Imagor request")
		imagorLoadTimeout = fs.Duration("imagor-load-timeout",
			time.Second*20, "Timeout for Imagor Loader request, should be smaller than imagor-request-timeout")
		imagorSaveTimeout = fs.Duration("imagor-save-timeout",
			time.Second*20, "Timeout for saving image to Imagor Storage")
		imagorProcessTimeout = fs.Duration("imagor-process-timeout",
			time.Second*20, "Timeout for image processing")
		imagorBasePathRedirect = fs.String("imagor-base-path-redirect", "",
			"URL to redirect for Imagor / base path e.g. https://www.google.com")
		imagorBaseParams = fs.String("imagor-base-params", "",
			"Imagor endpoint base params that applies to all resulting images e.g. fitlers:watermark(example.jpg)")
		imagorProcessConcurrency = fs.Int64("imagor-process-concurrency",
			-1, "Imagor semaphore size for process concurrency control. Set -1 for no limit")
		imagorCacheHeaderTTL = fs.Duration("imagor-cache-header-ttl",
			time.Hour*24*7, "Imagor HTTP Cache-Control header TTL for successful image response")
		imagorCacheHeaderSWR = fs.Duration("imagor-cache-header-swr",
			time.Hour*24, "Imagor HTTP Cache-Control header stale-while-revalidate for successful image response")
		imagorCacheHeaderNoCache = fs.Bool("imagor-cache-header-no-cache",
			false, "Imagor HTTP Cache-Control header no-cache for successful image response")
		imagorModifiedTimeCheck = fs.Bool("imagor-modified-time-check", false,
			"Check modified time of result image against the source image. This eliminates stale result but require more lookups")
		imagorDisableErrorBody      = fs.Bool("imagor-disable-error-body", false, "Imagor disable response body on error")
		imagorDisableParamsEndpoint = fs.Bool("imagor-disable-params-endpoint", false, "Imagor disable /params endpoint")
		imagorSignerType            = fs.String("imagor-signer-type", "sha1", "Imagor URL signature hasher type sha1 or sha256")
		imagorSignerTruncate        = fs.Int("imagor-signer-truncate", 0, "Imagor URL signature truncate at length")
	)

	var options, logger, isDebug = applyFuncs(fs, cb, funcs...)

	var alg = sha1.New
	if strings.ToLower(*imagorSignerType) == "sha256" {
		alg = sha256.New
	} else if strings.ToLower(*imagorSignerType) == "sha512" {
		alg = sha512.New
	}

	return imagor.New(append(
		options,
		imagor.WithSigner(imagorpath.NewHMACSigner(
			alg, *imagorSignerTruncate, *imagorSecret,
		)),
		imagor.WithBasePathRedirect(*imagorBasePathRedirect),
		imagor.WithBaseParams(*imagorBaseParams),
		imagor.WithRequestTimeout(*imagorRequestTimeout),
		imagor.WithLoadTimeout(*imagorLoadTimeout),
		imagor.WithSaveTimeout(*imagorSaveTimeout),
		imagor.WithProcessTimeout(*imagorProcessTimeout),
		imagor.WithProcessConcurrency(*imagorProcessConcurrency),
		imagor.WithCacheHeaderTTL(*imagorCacheHeaderTTL),
		imagor.WithCacheHeaderSWR(*imagorCacheHeaderSWR),
		imagor.WithCacheHeaderNoCache(*imagorCacheHeaderNoCache),
		imagor.WithAutoWebP(*imagorAutoWebP),
		imagor.WithAutoAVIF(*imagorAutoAVIF),
		imagor.WithModifiedTimeCheck(*imagorModifiedTimeCheck),
		imagor.WithDisableErrorBody(*imagorDisableErrorBody),
		imagor.WithDisableParamsEndpoint(*imagorDisableParamsEndpoint),
		imagor.WithUnsafe(*imagorUnsafe),
		imagor.WithLogger(logger),
		imagor.WithDebug(isDebug),
	)...)
}

func applyFuncs(
	fs *flag.FlagSet, cb Callback, funcs ...Func,
) (options []imagor.Option, logger *zap.Logger, isDebug bool) {
	if len(funcs) == 0 {
		logger, isDebug = cb()
		return
	} else {
		var last = len(funcs) - 1
		var called bool
		if funcs[last] == nil {
			return applyFuncs(fs, cb, funcs[:last]...)
		}
		options = append(options, funcs[last](fs, func() (*zap.Logger, bool) {
			options, logger, isDebug = applyFuncs(fs, cb, funcs[:last]...)
			called = true
			return logger, isDebug
		}))
		if !called {
			var opts []imagor.Option
			opts, logger, isDebug = applyFuncs(fs, cb, funcs[:last]...)
			options = append(opts, options...)
			return options, logger, isDebug
		}
		return
	}
}
