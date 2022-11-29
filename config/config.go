package config

import (
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"flag"
	"fmt"
	"github.com/cshum/imagor"
	"github.com/cshum/imagor/imagorpath"
	"github.com/cshum/imagor/server"
	"github.com/peterbourgon/ff/v3"
	"go.uber.org/zap"
	"runtime"
	"strings"
	"time"
)

var baseConfig = []Func{
	withFileSystem,
	withHTTPLoader,
}

func NewImagor(
	fs *flag.FlagSet, cb func() (*zap.Logger, bool), funcs ...Func,
) *imagor.Imagor {
	var (
		imagorSecret = fs.String("imagor-secret", "",
			"Secret key for signing imagor URL")
		imagorUnsafe = fs.Bool("imagor-unsafe", false,
			"Unsafe imagor that does not require URL signature. Prone to URL tampering")
		imagorAutoWebP = fs.Bool("imagor-auto-webp", false,
			"Output WebP format automatically if browser supports")
		imagorAutoAVIF = fs.Bool("imagor-auto-avif", false,
			"Output AVIF format automatically if browser supports (experimental)")
		imagorRequestTimeout = fs.Duration("imagor-request-timeout",
			time.Second*30, "Timeout for performing imagor request")
		imagorLoadTimeout = fs.Duration("imagor-load-timeout",
			0, "Timeout for imagor Loader request, should be smaller than imagor-request-timeout")
		imagorSaveTimeout = fs.Duration("imagor-save-timeout",
			0, "Timeout for saving image to imagor Storage")
		imagorProcessTimeout = fs.Duration("imagor-process-timeout",
			0, "Timeout for image processing")
		imagorBasePathRedirect = fs.String("imagor-base-path-redirect", "",
			"URL to redirect for imagor / base path e.g. https://www.google.com")
		imagorBaseParams = fs.String("imagor-base-params", "",
			"imagor endpoint base params that applies to all resulting images e.g. fitlers:watermark(example.jpg)")
		imagorProcessConcurrency = fs.Int64("imagor-process-concurrency",
			-1, "Maximum number of image process to be executed simultaneously. Requests that exceed this limit are put in the queue. Set -1 for no limit")
		imagorProcessQueueSize = fs.Int64("imagor-process-queue-size",
			-1, "Maximum number of image process that can be put in the queue. Requests that exceed this limit are rejected with HTTP status 429. Set -1 for no limit")
		imagorCacheHeaderTTL = fs.Duration("imagor-cache-header-ttl",
			time.Hour*24*7, "imagor HTTP Cache-Control header TTL for successful image response")
		imagorCacheHeaderSWR = fs.Duration("imagor-cache-header-swr",
			time.Hour*24, "imagor HTTP Cache-Control header stale-while-revalidate for successful image response")
		imagorCacheHeaderNoCache = fs.Bool("imagor-cache-header-no-cache",
			false, "imagor HTTP Cache-Control header no-cache for successful image response")
		imagorModifiedTimeCheck = fs.Bool("imagor-modified-time-check", false,
			"Check modified time of result image against the source image. This eliminates stale result but require more lookups")
		imagorDisableErrorBody       = fs.Bool("imagor-disable-error-body", false, "imagor disable response body on error")
		imagorDisableParamsEndpoint  = fs.Bool("imagor-disable-params-endpoint", false, "imagor disable /params endpoint")
		imagorSignerType             = fs.String("imagor-signer-type", "sha1", "imagor URL signature hasher type: sha1, sha256, sha512")
		imagorSignerTruncate         = fs.Int("imagor-signer-truncate", 0, "imagor URL signature truncate at length")
		imagorStoragePathStyle       = fs.String("imagor-storage-path-style", "original", "imagor storage path style: original, digest")
		imagorResultStoragePathStyle = fs.String("imagor-result-storage-path-style", "original", "imagor result storage path style: original, digest, suffix")

		options, logger, isDebug = applyFuncs(fs, cb, append(funcs, baseConfig...)...)

		alg          = sha1.New
		hasher       imagorpath.StorageHasher
		resultHasher imagorpath.ResultStorageHasher
	)

	if strings.ToLower(*imagorSignerType) == "sha256" {
		alg = sha256.New
	} else if strings.ToLower(*imagorSignerType) == "sha512" {
		alg = sha512.New
	}

	if strings.ToLower(*imagorStoragePathStyle) == "digest" {
		hasher = imagorpath.DigestStorageHasher
	}

	if strings.ToLower(*imagorResultStoragePathStyle) == "digest" {
		resultHasher = imagorpath.DigestResultStorageHasher
	} else if strings.ToLower(*imagorResultStoragePathStyle) == "suffix" {
		resultHasher = imagorpath.SuffixResultStorageHasher
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
		imagor.WithProcessQueueSize(*imagorProcessQueueSize),
		imagor.WithCacheHeaderTTL(*imagorCacheHeaderTTL),
		imagor.WithCacheHeaderSWR(*imagorCacheHeaderSWR),
		imagor.WithCacheHeaderNoCache(*imagorCacheHeaderNoCache),
		imagor.WithAutoWebP(*imagorAutoWebP),
		imagor.WithAutoAVIF(*imagorAutoAVIF),
		imagor.WithModifiedTimeCheck(*imagorModifiedTimeCheck),
		imagor.WithDisableErrorBody(*imagorDisableErrorBody),
		imagor.WithDisableParamsEndpoint(*imagorDisableParamsEndpoint),
		imagor.WithStoragePathStyle(hasher),
		imagor.WithResultStoragePathStyle(resultHasher),
		imagor.WithUnsafe(*imagorUnsafe),
		imagor.WithLogger(logger),
		imagor.WithDebug(isDebug),
	)...)
}

func CreateServer(args []string, funcs ...Func) (srv *server.Server) {
	var (
		fs     = flag.NewFlagSet("imagor", flag.ExitOnError)
		logger *zap.Logger
		err    error
		app    *imagor.Imagor

		debug        = fs.Bool("debug", false, "Debug mode")
		version      = fs.Bool("version", false, "imagor version")
		port         = fs.Int("port", 8000, "Sever port")
		goMaxProcess = fs.Int("gomaxprocs", 0, "GOMAXPROCS")

		_ = fs.String("config", ".env", "Retrieve configuration from the given file")

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

	app = NewImagor(fs, func() (*zap.Logger, bool) {
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
			logger = zap.Must(zap.NewDevelopment())
		} else {
			logger = zap.Must(zap.NewProduction())
		}
		return logger, *debug
	}, funcs...)

	if *version {
		fmt.Println(imagor.Version)
		return
	}

	if *goMaxProcess > 0 {
		logger.Debug("GOMAXPROCS", zap.Int("count", *goMaxProcess))
		runtime.GOMAXPROCS(*goMaxProcess)
	}

	return server.New(app,
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
