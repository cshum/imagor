package main

import (
	"flag"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/cshum/imagor"
	"github.com/cshum/imagor/loader/httploader"
	"github.com/cshum/imagor/processor/vipsprocessor"
	"github.com/cshum/imagor/server"
	"github.com/cshum/imagor/store/filestore"
	"github.com/cshum/imagor/store/s3store"
	"github.com/peterbourgon/ff/v3"
	"go.uber.org/zap"
	"os"
)

func main() {
	var (
		fs       = flag.NewFlagSet("imagor", flag.ExitOnError)
		logger   *zap.Logger
		err      error
		loaders  []imagor.Loader
		storages []imagor.Storage
	)

	var (
		debug = fs.Bool("debug", false, "debug mode")
		port  = fs.Int("port", 9000, "sever port")

		imagorSecret = fs.String("imagor-secret", "",
			"")
		imagorUnsafe = fs.Bool("imagor-unsafe", false,
			"")
		imagorRequestTimeout = fs.Duration("imagor-request-timeout", 0,
			"")
		imagorSaveTimeout = fs.Duration("imagor-save-timeout", 0,
			"")

		serverAddress = fs.String("server-address", "",
			"")
		serverPathPrefix = fs.String("server-path-prefix", "",
			"")
		serverCORS = fs.Bool("server-cors", false,
			"")

		vipsDisableBlur = fs.Bool("vips-disable-blur", false,
			"")
		vipsDisableFilters = fs.String("vips-disable-filters", "",
			"")

		httpLoaderForwardHeaders = fs.String(
			"http-loader-forward-headers", "",
			"")
		httpLoaderForwardUserAgent = fs.Bool(
			"http-loader-forward-user-agent", false,
			"")
		httpLoaderForwardAllHeaders = fs.Bool(
			"http-loader-forward-all-headers", false,
			"")
		httpLoaderAllowedSources = fs.String(
			"http-loader-allowed-sources", "",
			"")
		httpLoaderMaxAllowedSize = fs.Int(
			"http-loader-max-allowed-size", 0,
			"")
		httpLoaderInsecureSkipVerifyTransport = fs.Bool(
			"http-loader-insecure-skip-verify-transport", false,
			"")

		awsRegion = fs.String("aws-region", "",
			"")
		awsAccessKeyId = fs.String("aws-access-key-id", "",
			"")
		awsSecretAccessKey = fs.String("aws-secret-access-key", "",
			"")

		s3LoaderBucket = fs.String("s3-loader-bucket", "",
			"")
		s3LoaderBaseDir = fs.String("s3-loader-base-dir", "",
			"")
		s3LoaderPathPrefix = fs.String("s3-loader-path-prefix", "",
			"")

		s3StorageBucket = fs.String("s3-storage-bucket", "",
			"")
		s3StorageBaseDir = fs.String("s3-storage-base-dir", "",
			"")
		s3StoragePathPrefix = fs.String("s3-storage-path-prefix", "",
			"")

		fileLoaderBaseDir = fs.String("file-loader-base-dir", "",
			"")
		fileLoaderPathPrefix = fs.String("file-loader-path-prefix", "",
			"")

		fileStorageBaseDir = fs.String("file-storage-base-dir", "",
			"")
		fileStoragePathPrefix = fs.String("file-storage-path-prefix", "",
			"")
	)

	if err = ff.Parse(fs, os.Args[1:], ff.WithEnvVarNoPrefix()); err != nil {
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

	if *awsRegion != "" && *awsAccessKeyId != "" && *awsSecretAccessKey != "" {
		sess, err := session.NewSession(&aws.Config{
			Region: awsRegion,
			Credentials: credentials.NewStaticCredentials(
				*awsAccessKeyId, *awsSecretAccessKey, ""),
		})
		if err != nil {
			panic(err)
		}
		if *s3LoaderBucket != "" {
			loaders = append(loaders,
				s3store.New(sess, *s3LoaderBucket,
					s3store.WithPathPrefix(*s3LoaderPathPrefix),
					s3store.WithBaseDir(*s3LoaderBaseDir),
				),
			)
		}
		if *s3StorageBucket != "" {
			loaders = append(loaders,
				s3store.New(sess, *s3StorageBucket,
					s3store.WithPathPrefix(*s3StoragePathPrefix),
					s3store.WithBaseDir(*s3StorageBaseDir),
				),
			)
		}
	}

	if *fileLoaderBaseDir != "" {
		loaders = append(loaders,
			filestore.New(
				*fileLoaderBaseDir,
				filestore.WithPathPrefix(*fileLoaderPathPrefix),
			),
		)
	}

	if *fileStorageBaseDir != "" {
		storages = append(storages,
			filestore.New(
				*fileLoaderBaseDir,
				filestore.WithPathPrefix(*fileStoragePathPrefix),
			),
		)
	}

	loaders = append(loaders,
		httploader.New(
			httploader.WithForwardUserAgent(*httpLoaderForwardUserAgent),
			httploader.WithForwardAllHeaders(*httpLoaderForwardAllHeaders),
			httploader.WithForwardHeaders(*httpLoaderForwardHeaders),
			httploader.WithAllowedSources(*httpLoaderAllowedSources),
			httploader.WithMaxAllowedSize(*httpLoaderMaxAllowedSize),
			httploader.WithInsecureSkipVerifyTransport(*httpLoaderInsecureSkipVerifyTransport),
		),
	)

	server.New(
		imagor.New(
			imagor.WithLoaders(loaders...),
			imagor.WithStorages(storages...),
			imagor.WithProcessors(
				vipsprocessor.New(
					vipsprocessor.WithDisableBlur(*vipsDisableBlur),
					vipsprocessor.WithDisableFilters(*vipsDisableFilters),
					vipsprocessor.WithLogger(logger),
					vipsprocessor.WithDebug(*debug),
				),
			),
			imagor.WithSecret(*imagorSecret),
			imagor.WithRequestTimeout(*imagorRequestTimeout),
			imagor.WithSaveTimeout(*imagorSaveTimeout),
			imagor.WithUnsafe(*imagorUnsafe),
			imagor.WithLogger(logger),
			imagor.WithDebug(*debug),
		),
		server.WithAddress(*serverAddress),
		server.WithPort(*port),
		server.WithPathPrefix(*serverPathPrefix),
		server.WithCORS(*serverCORS),
		server.WithLogger(logger),
		server.WithDebug(*debug),
	).Run()
}
