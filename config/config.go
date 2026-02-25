package config

import (
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"flag"
	"fmt"
	"runtime"
	"strings"
	"time"

	"github.com/TheZeroSlave/zapsentry"
	"github.com/cshum/imagor"
	"github.com/cshum/imagor/imagorpath"
	"github.com/cshum/imagor/metrics/prometheusmetrics"
	"github.com/cshum/imagor/server"
	"github.com/getsentry/sentry-go"
	"github.com/peterbourgon/ff/v3"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var baseConfig = []Option{
	withFileSystem,
	withUploadLoader,
	withHTTPLoader, // HTTP loader should be last as a fallback
}

// NewImagor create imagor from config flags
func NewImagor(
	fs *flag.FlagSet, cb func() (*zap.Logger, bool), funcs ...Option,
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
		imagorAutoJPEG = fs.Bool("imagor-auto-jpeg", false,
			"Output JPEG format automatically if JPEG or no specific format is requested")
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
			"imagor endpoint base params that applies to all resulting images e.g. filters:watermark(example.jpg)")
		imagorProcessConcurrency = fs.Int64("imagor-process-concurrency",
			-1, "Maximum number of image process to be executed simultaneously. Requests that exceed this limit are put in the queue. Set -1 for no limit")
		imagorProcessQueueSize = fs.Int64("imagor-process-queue-size",
			0, "Maximum number of image process that can be put in the queue. Requests that exceed this limit are rejected with HTTP status 429")
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
		imagorResponseRawOnError     = fs.Bool("imagor-response-raw-on-error", false, "imagor response with a raw unprocessed and unchecked source image on error")
		imagorSignerType             = fs.String("imagor-signer-type", "sha1", "imagor URL signature hasher type: sha1, sha256, sha512")
		imagorSignerTruncate         = fs.Int("imagor-signer-truncate", 0, "imagor URL signature truncate at length")
		imagorStoragePathStyle       = fs.String("imagor-storage-path-style", "original", "imagor storage path style: original, digest")
		imagorResultStoragePathStyle = fs.String("imagor-result-storage-path-style", "original", "imagor result storage path style: original, digest, suffix")

		options, logger, isDebug = applyOptions(fs, cb, append(funcs, baseConfig...)...)

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
	} else if strings.ToLower(*imagorResultStoragePathStyle) == "size" {
		resultHasher = imagorpath.SizeSuffixResultStorageHasher
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
		imagor.WithAutoJPEG(*imagorAutoJPEG),
		imagor.WithModifiedTimeCheck(*imagorModifiedTimeCheck),
		imagor.WithDisableErrorBody(*imagorDisableErrorBody),
		imagor.WithDisableParamsEndpoint(*imagorDisableParamsEndpoint),
		imagor.WithResponseRawOnError(*imagorResponseRawOnError),
		imagor.WithStoragePathStyle(hasher),
		imagor.WithResultStoragePathStyle(resultHasher),
		imagor.WithUnsafe(*imagorUnsafe),
		imagor.WithLogger(logger),
		imagor.WithDebug(isDebug),
	)...)
}

// CreateServer create server from config flags. Returns nil on version or help command
func CreateServer(args []string, funcs ...Option) (srv *server.Server) {
	var (
		fs     = flag.NewFlagSet("imagor", flag.ExitOnError)
		logger *zap.Logger
		err    error
		app    *imagor.Imagor

		debug        = fs.Bool("debug", false, "Debug mode")
		version      = fs.Bool("version", false, "imagor version")
		port         = fs.Int("port", 8000, "Server port")
		goMaxProcess = fs.Int("gomaxprocs", 0, "GOMAXPROCS")

		bind = fs.String("bind", "",
			"Server address and port to bind .e.g. myhost:8888. This overrides server address and port config")

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
		sentryDsn = fs.String("sentry-dsn", "",
			"Sentry DSN config")

		prometheusBind = fs.String("prometheus-bind", "", "Specify address and port to enable Prometheus metrics, e.g. :5000, prom:7000")
		prometheusPath = fs.String("prometheus-path", "/", "Prometheus metrics path")
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

		if len(*sentryDsn) > 0 {
			err = sentry.Init(sentry.ClientOptions{
				Dsn: *sentryDsn,
			})
			if err != nil {
				fmt.Printf("sentry.Init: %s", err)
			}
			defer sentry.Flush(2 * time.Second)

			// Add Sentry integration to zap logger
			core, err := zapsentry.NewCore(zapsentry.Configuration{
				Level:             zapcore.ErrorLevel, // only log errors or higher levels to Sentry
				EnableBreadcrumbs: true,               // enable sending breadcrumbs to Sentry
				BreadcrumbLevel:   zapcore.InfoLevel,  // at what level should we sent breadcrumbs to sentry, this level can't be higher than `Level`
			}, zapsentry.NewSentryClientFromClient(sentry.CurrentHub().Client()))
			if err != nil {
				fmt.Printf("zapsentry integration error: %s", err)
			}
			logger = zapsentry.AttachCoreToLogger(core, logger)
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

	var pm *prometheusmetrics.PrometheusMetrics
	if *prometheusBind != "" {
		pm = prometheusmetrics.New(
			prometheusmetrics.WithAddr(*prometheusBind),
			prometheusmetrics.WithPath(*prometheusPath),
			prometheusmetrics.WithLogger(logger),
		)
	}

	return server.New(app,
		server.WithAddr(*bind),
		server.WithPort(*port),
		server.WithAddress(*serverAddress),
		server.WithPathPrefix(*serverPathPrefix),
		server.WithCORS(*serverCORS),
		server.WithStripQueryString(*serverStripQueryString),
		server.WithAccessLog(*serverAccessLog),
		server.WithLogger(logger),
		server.WithDebug(*debug),
		server.WithMetrics(pm),
		server.WithSentry(*sentryDsn),
	)
}
