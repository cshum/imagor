package main

import (
	"cloud.google.com/go/storage"
	"context"
	"flag"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/cshum/imagor"
	"github.com/cshum/imagor/loader/httploader"
	"github.com/cshum/imagor/processor/vipsprocessor"
	"github.com/cshum/imagor/server"
	"github.com/cshum/imagor/storage/filestorage"
	"github.com/cshum/imagor/storage/gcloudstorage"
	"github.com/cshum/imagor/storage/s3storage"
	"github.com/joho/godotenv"
	"github.com/peterbourgon/ff/v3"
	"go.uber.org/zap"
	"os"
	"runtime"
	"time"
)

func main() {
	var (
		fs            = flag.NewFlagSet("imagor", flag.ExitOnError)
		logger        *zap.Logger
		err           error
		loaders       []imagor.Loader
		savers        []imagor.Saver
		resultLoaders []imagor.Loader
		resultSavers  []imagor.Saver
	)

	_ = godotenv.Load()

	var (
		debug        = fs.Bool("debug", false, "Debug mode")
		version      = fs.Bool("version", false, "Imagor version")
		port         = fs.Int("port", 8000, "Sever port")
		goMaxProcess = fs.Int("gomaxprocs", 0, "GOMAXPROCS")

		imagorSecret = fs.String("imagor-secret", "",
			"Secret key for signing Imagor URL")
		imagorUnsafe = fs.Bool("imagor-unsafe", false,
			"Unsafe Imagor that does not require URL signature. Prone to URL tampering")
		imagorAutoWebP = fs.Bool("imagor-auto-webp", false,
			"Output WebP format automatically if browser supports")
		imagorAutoAvif = fs.Bool("imagor-auto-avif", false,
			"Output Avif format automatically if browser supports")
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
		imagorProcessConcurrency = fs.Int64("imagor-process-concurrency",
			-1, "Imagor semaphore size for process concurrency control. Set -1 for no limit")
		imagorCacheHeaderTTL = fs.Duration("imagor-cache-header-ttl",
			time.Hour*24, "Imagor HTTP cache header ttl for successful image response. Set -1 for no-cache")

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

		vipsDisableBlur = fs.Bool("vips-disable-blur", false,
			"VIPS disable blur operations for vips processor")
		vipsMaxAnimationFrames = fs.Int("vips-max-animation-frames", -1,
			"VIPS maximum number of animation frames to be loaded. Set 1 to disable animation, -1 for unlimited.")
		vipsDisableFilters = fs.String("vips-disable-filters", "",
			"VIPS disable filters by csv e.g. blur,watermark,rgb")
		vipsMaxFilterOps = fs.Int("vips-max-filter-ops", 10,
			"VIPS maximum number of filter operations allowed")
		vipsConcurrency = fs.Int("vips-concurrency", 1,
			"VIPS concurrency. Set -1 to be the number of CPU cores")
		vipsMaxCacheFiles = fs.Int("vips-max-cache-files", 0,
			"VIPS max cache files")
		vipsMaxCacheSize = fs.Int("vips-max-cache-size", 0,
			"VIPS max cache size")
		vipsMaxCacheMem = fs.Int("vips-max-cache-mem", 0,
			"VIPS max cache mem")
		vipsMaxWidth = fs.Int("vips-max-width", 0,
			"VIPS max image width")
		vipsMaxHeight = fs.Int("vips-max-height", 0,
			"VIPS max image height")
		vipsMozJPEG = fs.Bool("vips-mozjpeg", false,
			"VIPS enable maximum compression with MozJPEG for JPEG export")

		httpLoaderForwardHeaders = fs.String("http-loader-forward-headers", "",
			"Forward request header to HTTP Loader request by csv e.g. User-Agent,Accept")
		httpLoaderForwardClientHeaders = fs.Bool("http-loader-forward-client-headers", false,
			"Forward browser client request headers to HTTP Loader request")
		httpLoaderForwardAllHeaders = fs.Bool("http-loader-forward-all-headers", false,
			"Deprecated in flavour of -http-loader-forward-client-headers")
		httpLoaderAllowedSources = fs.String("http-loader-allowed-sources", "",
			"HTTP Loader allowed hosts whitelist to load images from if set. Accept csv wth glob pattern e.g. *.google.com,*.github.com.")
		httpLoaderMaxAllowedSize = fs.Int("http-loader-max-allowed-size", 0,
			"HTTP Loader maximum allowed size in bytes for loading images if set")
		httpLoaderInsecureSkipVerifyTransport = fs.Bool("http-loader-insecure-skip-verify-transport", false,
			"HTTP Loader to use HTTP transport with InsecureSkipVerify true")
		httpLoaderDefaultScheme = fs.String("http-loader-default-scheme", "https",
			"HTTP Loader default scheme if not specified by image path. Set \"nil\" to disable default scheme.")
		httpLoaderAccept = fs.String("http-loader-accept", "image/*",
			"HTTP Loader set request Accept header and validate response Content-Type header")
		httpLoaderProxyURLs = fs.String("http-loader-proxy-urls", "",
			"HTTP Loader Proxy URLs. Enable HTTP Loader proxy only if this value present. Accept csv of proxy urls e.g. http://user:pass@host:port,http://user:pass@host:port")
		httpLoaderProxyAllowedSources = fs.String("http-loader-proxy-allowed-sources", "",
			"HTTP Loader Proxy allowed hosts that enable proxy transport, if proxy URLs are set. Accept csv wth glob pattern e.g. *.google.com,*.github.com.")
		httpLoaderDisable = fs.Bool("http-loader-disable", false,
			"Disable HTTP Loader")

		awsRegion = fs.String("aws-region", "",
			"AWS Region. Required if using S3 Loader or storage")
		awsAccessKeyId = fs.String("aws-access-key-id", "",
			"AWS Access Key ID. Required if using S3 Loader or Storage")
		awsSecretAccessKey = fs.String("aws-secret-access-key", "",
			"AWS Secret Access Key. Required if using S3 Loader or Storage")
		s3Endpoint = fs.String("s3-endpoint", "",
			"Optional S3 Endpoint to override default")
		s3ForcePathStyle = fs.Bool("s3-force-path-style", false,
			"S3 force the request to use path-style addressing s3.amazonaws.com/bucket/key, instead of bucket.s3.amazonaws.com/key")
		s3SafeChars = fs.String("s3-safe-chars", "",
			"S3 safe characters to be excluded from image key escape")

		gcloudSafeChars = fs.String("gcloud-safe-chars", "",
			"Google Cloud safe characters to be excluded from image key escape")

		s3LoaderBucket = fs.String("s3-loader-bucket", "",
			"S3 Bucket for S3 Loader. Enable S3 Loader only if this value present")
		s3LoaderBaseDir = fs.String("s3-loader-base-dir", "",
			"Base directory for S3 Loader")
		s3LoaderPathPrefix = fs.String("s3-loader-path-prefix", "",
			"Base path prefix for S3 Loader")

		gcloudLoaderBucket = fs.String("gcloud-loader-bucket", "",
			"Bucket name for Google Cloud Storage Loader. Enable Google Cloud Loader only if this value present")
		gcloudLoaderBaseDir = fs.String("gcloud-loader-base-dir", "",
			"Base directory for Google Cloud Loader")
		gcloudLoaderPathPrefix = fs.String("gcloud-loader-path-prefix", "",
			"Base path prefix for Google Cloud Loader")

		s3StorageBucket = fs.String("s3-storage-bucket", "",
			"S3 Bucket for S3 Storage. Enable S3 Storage only if this value present")
		s3StorageBaseDir = fs.String("s3-storage-base-dir", "",
			"Base directory for S3 Storage")
		s3StoragePathPrefix = fs.String("s3-storage-path-prefix", "",
			"Base path prefix for S3 Storage")
		s3StorageACL = fs.String("s3-storage-acl", "public-read",
			"Upload ACL for S3 Storage")
		s3StorageExpiration = fs.Duration("s3-storage-expiration", 0,
			"S3 Storage expiration duration e.g. 24h. Default no expiration")

		gcloudStorageBucket = fs.String("gcloud-storage-bucket", "",
			"Bucket name for Google Cloud Storage. Enable Google Cloud Storage only if this value present")
		gcloudStorageBaseDir = fs.String("gcloud-storage-base-dir", "",
			"Base directory for Google Cloud")
		gcloudStoragePathPrefix = fs.String("gcloud-storage-path-prefix", "",
			"Base path prefix for Google Cloud Storage")
		gcloudStorageACL = fs.String("gcloud-storage-acl", "",
			"Upload ACL for Google Cloud Storage")
		gcloudStorageExpiration = fs.Duration("gcloud-storage-expiration", 0,
			"Google Cloud Storage expiration duration e.g. 24h. Default no expiration")

		fileSafeChars = fs.String("file-safe-chars", "",
			"File safe characters to be excluded from image key escape")
		fileLoaderBaseDir = fs.String("file-loader-base-dir", "",
			"Base directory for File Loader. Enable File Loader only if this value present")
		fileLoaderPathPrefix = fs.String("file-loader-path-prefix", "",
			"Base path prefix for File Loader")

		fileStorageBaseDir = fs.String("file-storage-base-dir", "",
			"Base directory for File Storage. Enable File Storage only if this value present")
		fileStoragePathPrefix = fs.String("file-storage-path-prefix", "",
			"Base path prefix for File Storage")
		fileStorageMkdirPermission = fs.String("file-storage-mkdir-permission", "0755",
			"File Storage mkdir permission")
		fileStorageWritePermission = fs.String("file-storage-write-permission", "0666",
			"File Storage write permission")
		fileStorageExpiration = fs.Duration("file-storage-expiration", 0,
			"File Storage expiration duration e.g. 24h. Default no expiration")

		s3ResultStorageBucket = fs.String("s3-result-storage-bucket", "",
			"S3 Bucket for S3 Result Storage. Enable S3 Result Storage only if this value present")
		s3ResultStorageBaseDir = fs.String("s3-result-storage-base-dir", "",
			"Base directory for S3 Result Storage")
		s3ResultStoragePathPrefix = fs.String("s3-result-storage-path-prefix", "",
			"Base path prefix for S3 Result Storage")
		s3ResultStorageACL = fs.String("s3-result-storage-acl", "public-read",
			"Upload ACL for S3 Result Storage")
		s3ResultStorageExpiration = fs.Duration("s3-result-storage-expiration", 0,
			"S3 Result Storage expiration duration e.g. 24h. Default no expiration")

		gcloudResultStorageBucket = fs.String("gcloud-result-storage-bucket", "",
			"Bucket name for Google Cloud Result Storage. Enable Google Cloud Result Storage only if this value present")
		gcloudResultStorageBaseDir = fs.String("gcloud-result-storage-base-dir", "",
			"Base directory for Google Cloud Result Storage")
		gcloudResultStoragePathPrefix = fs.String("gcloud-result-storage-path-prefix", "",
			"Base path prefix for Google Cloud Result Storage")
		gcloudResultStorageACL = fs.String("gcloud-result-storage-acl", "",
			"Upload ACL for Google Cloud Result Storage")
		gcloudResultStorageExpiration = fs.Duration("gcloud-result-storage-expiration", 0,
			"Google Cloud Result Storage expiration duration e.g. 24h. Default no expiration")

		fileResultStorageBaseDir = fs.String("file-result-storage-base-dir", "",
			"Base directory for File Result Storage. Enable File Result Storage only if this value present")
		fileResultStoragePathPrefix = fs.String("file-result-storage-path-prefix", "",
			"Base path prefix for File Result Storage")
		fileResultStorageMkdirPermission = fs.String("file-result-storage-mkdir-permission", "0755",
			"File Result Storage mkdir permission")
		fileResultStorageWritePermission = fs.String("file-result-storage-write-permission", "0666",
			"File Storage write permission")
		fileResultStorageExpiration = fs.Duration("file-result-storage-expiration", 0,
			"File Result Storage expiration duration e.g. 24h. Default no expiration")
	)

	if err = ff.Parse(fs, os.Args[1:], ff.WithEnvVarNoPrefix()); err != nil {
		panic(err)
	}

	if *version {
		fmt.Println(imagor.Version)
		return
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

	if *goMaxProcess > 0 {
		logger.Debug("GOMAXPROCS", zap.Int("count", *goMaxProcess))
		runtime.GOMAXPROCS(*goMaxProcess)
	}

	if *fileStorageBaseDir != "" {
		// activate File Storage only if base dir config presents
		s := filestorage.New(
			*fileStorageBaseDir,
			filestorage.WithPathPrefix(*fileStoragePathPrefix),
			filestorage.WithMkdirPermission(*fileStorageMkdirPermission),
			filestorage.WithWritePermission(*fileStorageWritePermission),
			filestorage.WithSafeChars(*fileSafeChars),
			filestorage.WithExpiration(*fileStorageExpiration),
		)
		loaders = append(loaders, s)
		savers = append(savers, s)
	}
	if *fileLoaderBaseDir != "" {
		// activate File Loader only if base dir config presents
		if *fileStorageBaseDir != *fileLoaderBaseDir ||
			*fileStoragePathPrefix != *fileLoaderPathPrefix {
			// create another loader if different from storage
			loaders = append(loaders,
				filestorage.New(
					*fileLoaderBaseDir,
					filestorage.WithPathPrefix(*fileLoaderPathPrefix),
					filestorage.WithSafeChars(*fileSafeChars),
				),
			)
		}
	}
	if *fileResultStorageBaseDir != "" {
		// activate File Result Storage only if base dir config presents
		resultStorage := filestorage.New(
			*fileResultStorageBaseDir,
			filestorage.WithPathPrefix(*fileResultStoragePathPrefix),
			filestorage.WithMkdirPermission(*fileResultStorageMkdirPermission),
			filestorage.WithWritePermission(*fileResultStorageWritePermission),
			filestorage.WithSafeChars(*fileSafeChars),
			filestorage.WithExpiration(*fileResultStorageExpiration),
		)
		resultLoaders = append(resultLoaders, resultStorage)
		resultSavers = append(resultSavers, resultStorage)
	}

	if *gcloudStorageBucket != "" || *gcloudLoaderBucket != "" || *gcloudResultStorageBucket != "" {
		// Activate the session, will panic if credentials are missing
		// Google cloud uses credentials from GOOGLE_APPLICATION_CREDENTIALS env file
		gcloudClient, err := storage.NewClient(context.Background())
		if err != nil {
			panic(err)
		}
		if *gcloudStorageBucket != "" {
			// activate Google Cloud Storage only if bucket config presents
			s := gcloudstorage.New(gcloudClient, *gcloudStorageBucket,
				gcloudstorage.WithPathPrefix(*gcloudStoragePathPrefix),
				gcloudstorage.WithBaseDir(*gcloudStorageBaseDir),
				gcloudstorage.WithACL(*gcloudStorageACL),
				gcloudstorage.WithSafeChars(*gcloudSafeChars),
				gcloudstorage.WithExpiration(*gcloudStorageExpiration),
			)
			loaders = append(loaders, s)
			savers = append(savers, s)
		}

		if *gcloudLoaderBucket != "" {
			// activate Google Cloud Loader only if bucket config presents
			if *gcloudLoaderPathPrefix != *gcloudStoragePathPrefix ||
				*gcloudLoaderBucket != *gcloudStorageBucket ||
				*gcloudLoaderBaseDir != *gcloudStorageBaseDir {
				// create another loader if different from storage
				loaders = append(loaders,
					gcloudstorage.New(gcloudClient, *gcloudLoaderBucket,
						gcloudstorage.WithPathPrefix(*gcloudLoaderPathPrefix),
						gcloudstorage.WithBaseDir(*gcloudLoaderBaseDir),
						gcloudstorage.WithSafeChars(*gcloudSafeChars),
					),
				)
			}
		}

		if *gcloudResultStorageBucket != "" {
			// activate Google Cloud ResultStorage only if bucket config presents
			resultStorage := gcloudstorage.New(gcloudClient, *gcloudResultStorageBucket,
				gcloudstorage.WithPathPrefix(*gcloudResultStoragePathPrefix),
				gcloudstorage.WithBaseDir(*gcloudResultStorageBaseDir),
				gcloudstorage.WithACL(*gcloudResultStorageACL),
				gcloudstorage.WithSafeChars(*gcloudSafeChars),
				gcloudstorage.WithExpiration(*gcloudResultStorageExpiration),
			)
			resultLoaders = append(resultLoaders, resultStorage)
			resultSavers = append(resultSavers, resultStorage)
		}
	}

	if *awsRegion != "" && *awsAccessKeyId != "" && *awsSecretAccessKey != "" {
		config := &aws.Config{
			Endpoint: s3Endpoint,
			Region:   awsRegion,
			Credentials: credentials.NewStaticCredentials(
				*awsAccessKeyId, *awsSecretAccessKey, ""),
		}
		if *s3ForcePathStyle {
			config.WithS3ForcePathStyle(true)
		}
		// activate AWS Session only if credentials present
		sess, err := session.NewSession(config)
		if err != nil {
			panic(err)
		}
		if *s3StorageBucket != "" {
			// activate S3 Storage only if bucket config presents
			s := s3storage.New(sess, *s3StorageBucket,
				s3storage.WithPathPrefix(*s3StoragePathPrefix),
				s3storage.WithBaseDir(*s3StorageBaseDir),
				s3storage.WithACL(*s3StorageACL),
				s3storage.WithSafeChars(*s3SafeChars),
				s3storage.WithExpiration(*s3StorageExpiration),
			)
			loaders = append(loaders, s)
			savers = append(savers, s)
		}
		if *s3LoaderBucket != "" {
			// activate S3 Loader only if bucket config presents
			if *s3LoaderPathPrefix != *s3StoragePathPrefix ||
				*s3LoaderBucket != *s3StorageBucket ||
				*s3LoaderBaseDir != *s3StorageBaseDir {
				// create another loader if different from storage
				loaders = append(loaders,
					s3storage.New(sess, *s3LoaderBucket,
						s3storage.WithPathPrefix(*s3LoaderPathPrefix),
						s3storage.WithBaseDir(*s3LoaderBaseDir),
						s3storage.WithSafeChars(*s3SafeChars),
					),
				)
			}
		}
		if *s3ResultStorageBucket != "" {
			// activate S3 ResultStorage only if bucket config presents
			resultStorage := s3storage.New(sess, *s3ResultStorageBucket,
				s3storage.WithPathPrefix(*s3ResultStoragePathPrefix),
				s3storage.WithBaseDir(*s3ResultStorageBaseDir),
				s3storage.WithACL(*s3ResultStorageACL),
				s3storage.WithSafeChars(*s3SafeChars),
				s3storage.WithExpiration(*s3ResultStorageExpiration),
			)
			resultLoaders = append(resultLoaders, resultStorage)
			resultSavers = append(resultSavers, resultStorage)
		}
	}

	if !*httpLoaderDisable {
		// fallback with HTTP Loader unless explicitly disabled
		loaders = append(loaders,
			httploader.New(
				httploader.WithForwardClientHeaders(
					*httpLoaderForwardClientHeaders || *httpLoaderForwardAllHeaders),
				httploader.WithAccept(*httpLoaderAccept),
				httploader.WithForwardHeaders(*httpLoaderForwardHeaders),
				httploader.WithAllowedSources(*httpLoaderAllowedSources),
				httploader.WithMaxAllowedSize(*httpLoaderMaxAllowedSize),
				httploader.WithInsecureSkipVerifyTransport(*httpLoaderInsecureSkipVerifyTransport),
				httploader.WithDefaultScheme(*httpLoaderDefaultScheme),
				httploader.WithProxyTransport(*httpLoaderProxyURLs, *httpLoaderProxyAllowedSources),
			),
		)
	}

	// run server with Imagor app
	server.New(
		imagor.New(
			imagor.WithLoaders(loaders...),
			imagor.WithSavers(savers...),
			imagor.WithProcessors(
				vipsprocessor.New(
					vipsprocessor.WithMaxAnimationFrames(*vipsMaxAnimationFrames),
					vipsprocessor.WithDisableBlur(*vipsDisableBlur),
					vipsprocessor.WithDisableFilters(*vipsDisableFilters),
					vipsprocessor.WithConcurrency(*vipsConcurrency),
					vipsprocessor.WithMaxCacheFiles(*vipsMaxCacheFiles),
					vipsprocessor.WithMaxCacheMem(*vipsMaxCacheMem),
					vipsprocessor.WithMaxCacheSize(*vipsMaxCacheSize),
					vipsprocessor.WithMaxFilterOps(*vipsMaxFilterOps),
					vipsprocessor.WithMaxWidth(*vipsMaxWidth),
					vipsprocessor.WithMaxHeight(*vipsMaxHeight),
					vipsprocessor.WithMozJPEG(*vipsMozJPEG),
					vipsprocessor.WithLogger(logger),
					vipsprocessor.WithDebug(*debug),
				),
			),
			imagor.WithResultLoaders(resultLoaders...),
			imagor.WithResultSavers(resultSavers...),
			imagor.WithSecret(*imagorSecret),
			imagor.WithBasePathRedirect(*imagorBasePathRedirect),
			imagor.WithRequestTimeout(*imagorRequestTimeout),
			imagor.WithLoadTimeout(*imagorLoadTimeout),
			imagor.WithSaveTimeout(*imagorSaveTimeout),
			imagor.WithProcessTimeout(*imagorProcessTimeout),
			imagor.WithProcessConcurrency(*imagorProcessConcurrency),
			imagor.WithCacheHeaderTTL(*imagorCacheHeaderTTL),
			imagor.WithAutoWebP(*imagorAutoWebP),
			imagor.WithAutoAvif(*imagorAutoAvif),
			imagor.WithUnsafe(*imagorUnsafe),
			imagor.WithLogger(logger),
			imagor.WithDebug(*debug),
		),
		server.WithAddress(*serverAddress),
		server.WithPort(*port),
		server.WithPathPrefix(*serverPathPrefix),
		server.WithCORS(*serverCORS),
		server.WithStripQueryString(*serverStripQueryString),
		server.WithAccessLog(*serverAccessLog),
		server.WithLogger(logger),
		server.WithDebug(*debug),
	).Run()
}
