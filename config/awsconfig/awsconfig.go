package awsconfig

import (
	"context"
	"flag"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/cshum/imagor"
	"github.com/cshum/imagor/storage/s3storage"
	"go.uber.org/zap"
)

// WithAWS with AWS S3 Loader, Storage and Result Storage config option
func WithAWS(fs *flag.FlagSet, cb func() (*zap.Logger, bool)) imagor.Option {
	var (
		awsRegion = fs.String("aws-region", "",
			"AWS Region. Required if using S3 Loader or storage")
		awsAccessKeyID = fs.String("aws-access-key-id", "",
			"AWS Access Key ID. Required if using S3 Loader or Storage")
		awsSecretAccessKey = fs.String("aws-secret-access-key", "",
			"AWS Secret Access Key. Required if using S3 Loader or Storage")
		awsSessionToken = fs.String("aws-session-token", "",
			"AWS Session Token. Optional temporary credentials token")
		s3Endpoint = fs.String("s3-endpoint", "",
			"Optional S3 Endpoint to override default")

		awsLoaderRegion = fs.String("aws-loader-region", "",
			"AWS Region for S3 Loader to override global config")
		awsLoaderAccessKeyID = fs.String("aws-loader-access-key-id", "",
			"AWS Access Key ID for S3 Loader to override global config")
		awsLoaderSecretAccessKey = fs.String("aws-loader-secret-access-key", "",
			"AWS Secret Access Key for S3 Loader to override global config")
		awsLoaderSessionToken = fs.String("aws-loader-session-token", "",
			"AWS Session Token for S3 Loader to override global config")
		s3LoaderEndpoint = fs.String("s3-loader-endpoint", "",
			"Optional S3 Loader Endpoint to override default")

		awsStorageRegion = fs.String("aws-storage-region", "",
			"AWS Region for S3 Storage to override global config")
		awsStorageAccessKeyID = fs.String("aws-storage-access-key-id", "",
			"AWS Access Key ID for S3 Storage to override global config")
		awsStorageSecretAccessKey = fs.String("aws-storage-secret-access-key", "",
			"AWS Secret Access Key for S3 Storage to override global config")
		awsStorageSessionToken = fs.String("aws-storage-session-token", "",
			"AWS Session Token for S3 Storage to override global config")
		s3StorageEndpoint = fs.String("s3-storage-endpoint", "",
			"Optional S3 Storage Endpoint to override default")

		awsResultStorageRegion = fs.String("aws-result-storage-region", "",
			"AWS Region for S3 Result Storage to override global config")
		awsResultStorageAccessKeyID = fs.String("aws-result-storage-access-key-id", "",
			"AWS Access Key ID for S3 Result Storage to override global config")
		awsResultStorageSecretAccessKey = fs.String("aws-result-storage-secret-access-key", "",
			"AWS Secret Access Key for S3 Result Storage to override global config")
		awsResultStorageSessionToken = fs.String("aws-result-storage-session-token", "",
			"AWS Session Token for S3 Result Storage to override global config")
		s3ResultStorageEndpoint = fs.String("s3-result-storage-endpoint", "",
			"Optional S3 Storage Endpoint to override default")

		s3ForcePathStyle = fs.Bool("s3-force-path-style", false,
			"S3 force the request to use path-style addressing s3.amazonaws.com/bucket/key, instead of bucket.s3.amazonaws.com/key")
		s3SafeChars = fs.String("s3-safe-chars", "",
			"S3 safe characters to be excluded from image key escape. Set -- for no-op")

		s3LoaderBucket = fs.String("s3-loader-bucket", "",
			"S3 Bucket for S3 Loader. Enable S3 Loader only if this value present")
		s3LoaderBaseDir = fs.String("s3-loader-base-dir", "",
			"Base directory for S3 Loader")
		s3LoaderPathPrefix = fs.String("s3-loader-path-prefix", "",
			"Base path prefix for S3 Loader")

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
		s3StorageClass = fs.String("s3-storage-class", "STANDARD",
			"S3 File Storage Class. Available values: REDUCED_REDUNDANCY, STANDARD_IA, ONEZONE_IA, INTELLIGENT_TIERING, GLACIER, DEEP_ARCHIVE. Default: STANDARD.")

		_, _ = cb()
	)
	return func(app *imagor.Imagor) {
		if *s3StorageBucket == "" && *s3LoaderBucket == "" && *s3ResultStorageBucket == "" {
			return
		}

		ctx := context.Background()

		// Helper function to create S3 client with custom endpoint
		createS3Client := func(cfg aws.Config, endpoint string) *s3.Client {
			var options []func(*s3.Options)

			if endpoint != "" {
				options = append(options, func(o *s3.Options) {
					o.BaseEndpoint = aws.String(endpoint)
				})
			}

			if *s3ForcePathStyle {
				options = append(options, func(o *s3.Options) {
					o.UsePathStyle = true
				})
			}

			return s3.NewFromConfig(cfg, options...)
		}

		// Create base configuration
		var loaderCfg, storageCfg, resultStorageCfg aws.Config
		var err error

		// Default configuration
		defaultCfg, err := config.LoadDefaultConfig(ctx)
		if err != nil {
			panic(err)
		}

		// Override with explicit credentials if provided
		if *awsAccessKeyID != "" && *awsSecretAccessKey != "" {
			defaultCfg.Credentials = credentials.NewStaticCredentialsProvider(
				*awsAccessKeyID, *awsSecretAccessKey, *awsSessionToken)
		}
		if *awsRegion != "" {
			defaultCfg.Region = *awsRegion
		}

		// Set default configurations
		loaderCfg = defaultCfg
		storageCfg = defaultCfg
		resultStorageCfg = defaultCfg

		// Override loader config if specific credentials provided
		if *awsLoaderRegion != "" && *awsLoaderAccessKeyID != "" && *awsLoaderSecretAccessKey != "" {
			loaderCfg = aws.Config{
				Region: *awsLoaderRegion,
				Credentials: credentials.NewStaticCredentialsProvider(
					*awsLoaderAccessKeyID, *awsLoaderSecretAccessKey, *awsLoaderSessionToken),
			}
		}

		// Override storage config if specific credentials provided
		if *awsStorageRegion != "" && *awsStorageAccessKeyID != "" && *awsStorageSecretAccessKey != "" {
			storageCfg = aws.Config{
				Region: *awsStorageRegion,
				Credentials: credentials.NewStaticCredentialsProvider(
					*awsStorageAccessKeyID, *awsStorageSecretAccessKey, *awsStorageSessionToken),
			}
		}

		// Override result storage config if specific credentials provided
		if *awsResultStorageRegion != "" && *awsResultStorageAccessKeyID != "" && *awsResultStorageSecretAccessKey != "" {
			resultStorageCfg = aws.Config{
				Region: *awsResultStorageRegion,
				Credentials: credentials.NewStaticCredentialsProvider(
					*awsResultStorageAccessKeyID, *awsResultStorageSecretAccessKey, *awsResultStorageSessionToken),
			}
		}

		// Create S3 Storage instances
		if *s3StorageBucket != "" {
			// Determine endpoint: service-specific takes priority over global
			endpoint := *s3StorageEndpoint
			if endpoint == "" {
				endpoint = *s3Endpoint
			}

			storage := s3storage.New(storageCfg, *s3StorageBucket,
				s3storage.WithPathPrefix(*s3StoragePathPrefix),
				s3storage.WithBaseDir(*s3StorageBaseDir),
				s3storage.WithACL(*s3StorageACL),
				s3storage.WithSafeChars(*s3SafeChars),
				s3storage.WithExpiration(*s3StorageExpiration),
				s3storage.WithStorageClass(*s3StorageClass),
			)
			// Override client with custom endpoint if needed
			storage.Client = createS3Client(storageCfg, endpoint)

			app.Storages = append(app.Storages, storage)
		}

		if *s3LoaderBucket != "" {
			// Determine endpoint: service-specific takes priority over global
			endpoint := *s3LoaderEndpoint
			if endpoint == "" {
				endpoint = *s3Endpoint
			}

			loader := s3storage.New(loaderCfg, *s3LoaderBucket,
				s3storage.WithPathPrefix(*s3LoaderPathPrefix),
				s3storage.WithBaseDir(*s3LoaderBaseDir),
				s3storage.WithSafeChars(*s3SafeChars),
			)
			// Override client with custom endpoint if needed
			loader.Client = createS3Client(loaderCfg, endpoint)

			app.Loaders = append(app.Loaders, loader)
		}

		if *s3ResultStorageBucket != "" {
			// Determine endpoint: service-specific takes priority over global
			endpoint := *s3ResultStorageEndpoint
			if endpoint == "" {
				endpoint = *s3Endpoint
			}

			resultStorage := s3storage.New(resultStorageCfg, *s3ResultStorageBucket,
				s3storage.WithPathPrefix(*s3ResultStoragePathPrefix),
				s3storage.WithBaseDir(*s3ResultStorageBaseDir),
				s3storage.WithACL(*s3ResultStorageACL),
				s3storage.WithSafeChars(*s3SafeChars),
				s3storage.WithExpiration(*s3ResultStorageExpiration),
				s3storage.WithStorageClass(*s3StorageClass),
			)
			// Override client with custom endpoint if needed
			resultStorage.Client = createS3Client(resultStorageCfg, endpoint)

			app.ResultStorages = append(app.ResultStorages, resultStorage)
		}
	}
}
