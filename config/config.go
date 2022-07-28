package config

import (
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"flag"
	"fmt"
	"github.com/cshum/imagor/imagorpath"
	"runtime"
	"strings"
	"time"

	"github.com/cshum/imagor"
	"github.com/cshum/imagor/server"
	"github.com/peterbourgon/ff/v3"
	"go.uber.org/zap"
)

type Callback func() (logger *zap.Logger, isDebug bool)

type Setter func(fs *flag.FlagSet, cb Callback) imagor.Option

func Do(args []string, setters ...Setter) (srv *server.Server) {
	// base setters
	setters = append(setters, withFile, withHTTPLoader)

	var (
		fs      = flag.NewFlagSet("imagor", flag.ExitOnError)
		logger  *zap.Logger
		err     error
		options []imagor.Option
		alg     = sha1.New

		debug        = fs.Bool("debug", false, "Debug mode")
		version      = fs.Bool("version", false, "Imagor version")
		port         = fs.Int("port", 8000, "Sever port")
		goMaxProcess = fs.Int("gomaxprocs", 0, "GOMAXPROCS")

		_ = fs.String("config", ".env", "Retrieve configuration from the given file")

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

		serverAddress = fs.String("server-address", "",
			"Server address")
		serverPathPrefix = fs.String("server-path-prefix", "",
			"Server path prefix")
		serverCORS = fs.Bool("server-cors", false,
			"Enable CORS")
		serverStripQueryString = fs.Bool("server-strip-query-string", false,
			"Enable strip query string redirection")
		serverAccessLog = fs.Bool("server-access-log", false,
			"Enable server access log")
	)

	options = doSetters(fs, setters, func() (*zap.Logger, bool) {
		if err = ff.Parse(fs, args,
			ff.WithEnvVars(),
			ff.WithConfigFileFlag("config"),
			ff.WithIgnoreUndefined(true),
			ff.WithAllowMissingConfigFile(true),
			ff.WithConfigFileParser(ff.EnvParser),
		); err != nil {
			panic(err)
		}
		if *debug {
			if logger, err = zap.NewDevelopment(); err != nil {
				panic(err)
			}
		} else {
			if logger, err = zap.NewProduction(); err != nil {
				panic(err)
			}
		}
		return logger, *debug
	})

	if *version {
		fmt.Println(imagor.Version)
		return
	}

	if *goMaxProcess > 0 {
		logger.Debug("GOMAXPROCS", zap.Int("count", *goMaxProcess))
		runtime.GOMAXPROCS(*goMaxProcess)
	}

	if strings.ToLower(*imagorSignerType) == "sha256" {
		alg = sha256.New
	} else if strings.ToLower(*imagorSignerType) == "sha512" {
		alg = sha512.New
	}

	return server.New(
		imagor.New(append(
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
			imagor.WithDebug(*debug),
		)...),
		server.WithAddress(*serverAddress),
		server.WithPort(*port),
		server.WithPathPrefix(*serverPathPrefix),
		server.WithCORS(*serverCORS),
		server.WithStripQueryString(*serverStripQueryString),
		server.WithAccessLog(*serverAccessLog),
		server.WithLogger(logger),
		server.WithDebug(*debug),
	)
}

func doSetters(fs *flag.FlagSet, setters []Setter, cb Callback) (options []imagor.Option) {
	var logger *zap.Logger
	var isDebug bool
	if len(setters) > 0 {
		var last = len(setters) - 1
		options = append(options, setters[last](fs, func() (*zap.Logger, bool) {
			options = append(options, doSetters(fs, setters[:last], cb)...)
			return logger, isDebug
		}))
		return
	}
	logger, isDebug = cb()
	return
}
