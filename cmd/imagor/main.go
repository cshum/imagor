package main

import (
	"flag"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/cshum/imagor"
	"github.com/cshum/imagor/loader/httploader"
	"github.com/cshum/imagor/processor/vipsprocessor"
	"github.com/cshum/imagor/server"
	"github.com/cshum/imagor/store/filestore"
	"github.com/cshum/imagor/store/s3store"
	"github.com/joho/godotenv"
	"github.com/peterbourgon/ff/v3"
	"go.uber.org/zap"
	"os"
	"runtime"
	"time"
)

var Version = "dev"

func main() {
	var (
		fs       = flag.NewFlagSet("imagor", flag.ExitOnError)
		logger   *zap.Logger
		err      error
		loaders  []imagor.Loader
		storages []imagor.Storage
	)

	_ = godotenv.Load()

	var (
		debug        = fs.Bool("debug", false, "Debug mode")
		port         = fs.Int("port", 8000, "Sever port")
		goMaxProcess = fs.Int("gomaxprocs", 0, "GOMAXPROCS")

		imagorSecret = fs.String("imagor-secret", "",
			"Secret key for signing Imagor URL")
		imagorUnsafe = fs.Bool("imagor-unsafe", false,
			"Unsafe Imagor that does not require URL signature. Prone to URL tampering")
		imagorRequestTimeout = fs.Duration("imagor-request-timeout",
			time.Second*30, "Timeout for performing Imagor request")
		imagorLoadTimeout = fs.Duration("imagor-load-timeout",
			time.Second*20, "Timeout for Imagor Loader request, should be smaller than imagor-request-timeout")
		imagorSaveTimeout = fs.Duration("imagor-save-timeout",
			time.Minute, "Timeout for saving image for storage")
		imagorCacheHeaderTTL = fs.Duration("imagor-cache-header-ttl",
			time.Hour*24, "Imagor HTTP cache header ttl for successful image response")
		imagorVersion = fs.Bool("imagor-version", false, "Imagor version")

		serverAddress = fs.String("server-address", "",
			"Server address")
		serverPathPrefix = fs.String("server-path-prefix", "",
			"Server path prefix")
		serverCORS = fs.Bool("server-cors", false,
			"Enable CORS")
		serverStripQueryString = fs.Bool("server-strip-query-string", false,
			"Enable strip query string redirection")

		vipsDisableBlur = fs.Bool("vips-disable-blur", false,
			"VIPS disable blur operations for vips processor")
		vipsDisableFilters = fs.String("vips-disable-filters", "",
			"VIPS disable filters by csv e.g. blur,watermark,rgb")
		vipsMaxFilterOps = fs.Int("vips-max-filter-ops", 10,
			"VIPS maximum number of filter operations allowed")
		vipsConcurrencyLevel = fs.Int("vips-concurrency-level", 1,
			"VIPS concurrency level. Set -1 to be the number of CPU cores")
		vipsMaxCacheFiles = fs.Int("vips-max-cache-files", 0,
			"VIPS max cache files")
		vipsMaxCacheSize = fs.Int("vips-max-cache-size", 0,
			"VIPS max cache size")
		vipsMaxCacheMem = fs.Int("vips-max-cache-mem", 0,
			"VIPS max cache mem")

		httpLoaderForwardHeaders = fs.String("http-loader-forward-headers", "",
			"Forward request header to HTTP Loader request by csv e.g. User-Agent,Accept")
		httpLoaderForwardAllHeaders = fs.Bool("http-loader-forward-all-headers", false,
			"Forward all request headers to HTTP Loader request")
		httpLoaderAllowedSources = fs.String("http-loader-allowed-sources", "",
			"HTTP Loader allowed hosts whitelist to load images from if set. Accept csv wth glob pattern e.g. *.google.com,*.github.com.")
		httpLoaderMaxAllowedSize = fs.Int("http-loader-max-allowed-size", 0,
			"HTTP Loader maximum allowed size in bytes for loading images if set")
		httpLoaderInsecureSkipVerifyTransport = fs.Bool("http-loader-insecure-skip-verify-transport", false,
			"HTTP Loader to use HTTP transport with InsecureSkipVerify true")
		httpLoaderDefaultScheme = fs.String("http-loader-default-scheme", "https",
			"HTTP Loader default scheme if not specified by image path. Set \"nil\" to disable default scheme.")
		httpLoaderDisable = fs.Bool("http-loader-disable", false,
			"Disable HTTP Loader")

		awsRegion = fs.String("aws-region", "",
			"AWS Region. Required if using S3 Loader or storage")
		awsAccessKeyId = fs.String("aws-access-key-id", "",
			"AWS Access Key ID. Required if using S3 Loader or storage")
		awsSecretAccessKey = fs.String("aws-secret-access-key", "",
			"AWS Secret Access Key. Required if using S3 Loader or storage")
		s3Endpoint = fs.String("s3-endpoint", "",
			"Optional S3 Endpoint to override default")

		s3LoaderBucket = fs.String("s3-loader-bucket", "",
			"S3 Bucket for S3 Loader. Will activate S3 Loader only if this value present")
		s3LoaderBaseDir = fs.String("s3-loader-base-dir", "",
			"Base directory for S3 Loader")
		s3LoaderPathPrefix = fs.String("s3-loader-path-prefix", "",
			"Base path prefix for S3 Loader")

		s3StorageBucket = fs.String("s3-storage-bucket", "",
			"S3 Bucket for S3 Storage. Will activate S3 Storage only if this value present")
		s3StorageBaseDir = fs.String("s3-storage-base-dir", "",
			"Base directory for S3 Storage")
		s3StoragePathPrefix = fs.String("s3-storage-path-prefix", "",
			"Base path prefix for S3 Storage")
		s3StorageACL = fs.String("s3-storage-acl", "public-read",
			"Upload ACL for S3 Storage")

		fileLoaderBaseDir = fs.String("file-loader-base-dir", "",
			"Base directory for File Loader. Will activate File Loader only if this value present")
		fileLoaderPathPrefix = fs.String("file-loader-path-prefix", "",
			"Base path prefix for File Loader")

		fileStorageBaseDir = fs.String("file-storage-base-dir", "",
			"Base directory for File Storage. Will activate File Storage only if this value present")
		fileStoragePathPrefix = fs.String("file-storage-path-prefix", "",
			"Base path prefix for File Storage")
		fileStorageMkdirPermission = fs.String("file-storage-mkdir-permission", "0755",
			"File Storage mkdir permission")
		fileStorageWritePermission = fs.String("file-storage-write-permission", "0666",
			"File Storage write permission")
	)

	if err = ff.Parse(fs, os.Args[1:], ff.WithEnvVarNoPrefix()); err != nil {
		panic(err)
	}

	if *imagorVersion {
		fmt.Println(Version)
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

	if *awsRegion != "" && *awsAccessKeyId != "" && *awsSecretAccessKey != "" {
		// activate AWS Session only if credentials present
		sess, err := session.NewSession(&aws.Config{
			Endpoint: s3Endpoint,
			Region:   awsRegion,
			Credentials: credentials.NewStaticCredentials(
				*awsAccessKeyId, *awsSecretAccessKey, ""),
		})
		if err != nil {
			panic(err)
		}
		var store *s3store.S3Store
		if *s3StorageBucket != "" {
			// activate S3 Storage only if bucket config presents
			store = s3store.New(sess, *s3StorageBucket,
				s3store.WithPathPrefix(*s3StoragePathPrefix),
				s3store.WithBaseDir(*s3StorageBaseDir),
				s3store.WithACL(*s3StorageACL),
			)
			storages = append(storages, store)
		}
		if *s3LoaderBucket != "" {
			// activate S3 Loader only if bucket config presents
			if store != nil &&
				*s3LoaderPathPrefix == *s3StoragePathPrefix &&
				*s3LoaderBucket == *s3StorageBucket &&
				*s3LoaderBaseDir == *s3StorageBaseDir {
				// reuse store if loader and storage are the same
				loaders = append(loaders, store)
			} else {
				// otherwise, create another loader
				loaders = append(loaders,
					s3store.New(sess, *s3LoaderBucket,
						s3store.WithPathPrefix(*s3LoaderPathPrefix),
						s3store.WithBaseDir(*s3LoaderBaseDir),
					),
				)
			}
		}
	}

	var store *filestore.FileStore
	if *fileStorageBaseDir != "" {
		// activate File Storage only if base dir config presents
		store = filestore.New(
			*fileStorageBaseDir,
			filestore.WithPathPrefix(*fileStoragePathPrefix),
			filestore.WithMkdirPermission(*fileStorageMkdirPermission),
			filestore.WithWritePermission(*fileStorageWritePermission),
		)
		storages = append(storages, store)
	}
	if *fileLoaderBaseDir != "" {
		// activate File Loader only if base dir config presents
		if store != nil &&
			*fileStorageBaseDir == *fileLoaderBaseDir &&
			*fileStoragePathPrefix == *fileLoaderPathPrefix {
			// reuse store if loader and storage are the same
			loaders = append(loaders, store)
		} else {
			// otherwise, create another loader
			loaders = append(loaders,
				filestore.New(
					*fileLoaderBaseDir,
					filestore.WithPathPrefix(*fileLoaderPathPrefix),
				),
			)
		}
	}

	if !*httpLoaderDisable {
		// fallback with HTTP Loader unless explicitly disabled
		loaders = append(loaders,
			httploader.New(
				httploader.WithForwardAllHeaders(*httpLoaderForwardAllHeaders),
				httploader.WithForwardHeaders(*httpLoaderForwardHeaders),
				httploader.WithAllowedSources(*httpLoaderAllowedSources),
				httploader.WithMaxAllowedSize(*httpLoaderMaxAllowedSize),
				httploader.WithInsecureSkipVerifyTransport(*httpLoaderInsecureSkipVerifyTransport),
				httploader.WithDefaultScheme(*httpLoaderDefaultScheme),
				httploader.WithUserAgent(fmt.Sprintf("Imagor/%s", Version)),
			),
		)
	}

	// run server with Imagor app
	server.New(
		imagor.New(
			imagor.WithVersion(Version),
			imagor.WithLoaders(loaders...),
			imagor.WithStorages(storages...),
			imagor.WithProcessors(
				vipsprocessor.New(
					vipsprocessor.WithDisableBlur(*vipsDisableBlur),
					vipsprocessor.WithDisableFilters(*vipsDisableFilters),
					vipsprocessor.WithConcurrencyLevel(*vipsConcurrencyLevel),
					vipsprocessor.WithMaxCacheFiles(*vipsMaxCacheFiles),
					vipsprocessor.WithMaxCacheMem(*vipsMaxCacheMem),
					vipsprocessor.WithMaxCacheSize(*vipsMaxCacheSize),
					vipsprocessor.WithMaxFilterOps(*vipsMaxFilterOps),
					vipsprocessor.WithLogger(logger),
					vipsprocessor.WithDebug(*debug),
				),
			),
			imagor.WithSecret(*imagorSecret),
			imagor.WithRequestTimeout(*imagorRequestTimeout),
			imagor.WithLoadTimeout(*imagorLoadTimeout),
			imagor.WithSaveTimeout(*imagorSaveTimeout),
			imagor.WithCacheHeaderTTL(*imagorCacheHeaderTTL),
			imagor.WithUnsafe(*imagorUnsafe),
			imagor.WithLogger(logger),
			imagor.WithDebug(*debug),
		),
		server.WithAddress(*serverAddress),
		server.WithPort(*port),
		server.WithPathPrefix(*serverPathPrefix),
		server.WithCORS(*serverCORS),
		server.WithStripQueryString(*serverStripQueryString),
		server.WithLogger(logger),
		server.WithDebug(*debug),
	).Run()
}
