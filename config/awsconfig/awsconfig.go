package awsconfig

import (
	"flag"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
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
			"S3 safe characters to be excluded from image key escape")

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

		_, _ = cb()
	)
	return func(app *imagor.Imagor) {
		if *s3StorageBucket == "" && *s3LoaderBucket == "" && *s3ResultStorageBucket == "" {
			return
		}
		var loaderSess, storageSess, resultStorageSess *session.Session
		var cred = credentials.NewStaticCredentials(
			*awsAccessKeyID, *awsSecretAccessKey, *awsSessionToken)
		var options = session.Options{
			SharedConfigState: session.SharedConfigEnable,
		}
		if _, err := cred.Get(); err != nil {
			cred = credentials.NewSharedCredentials("", "")
		}
		if _, err := cred.Get(); err == nil {
			options.Config = aws.Config{
				Endpoint:         s3Endpoint,
				Region:           awsRegion,
				Credentials:      cred,
				S3ForcePathStyle: s3ForcePathStyle,
			}
		}
		var sess = session.Must(session.NewSessionWithOptions(options))
		loaderSess = sess
		storageSess = sess
		resultStorageSess = sess
		if *awsLoaderRegion != "" && *awsLoaderAccessKeyID != "" && *awsLoaderSecretAccessKey != "" {
			cfg := &aws.Config{
				Endpoint: s3LoaderEndpoint,
				Region:   awsLoaderRegion,
				Credentials: credentials.NewStaticCredentials(
					*awsLoaderAccessKeyID, *awsLoaderSecretAccessKey, *awsLoaderSessionToken),
				S3ForcePathStyle: s3ForcePathStyle,
			}
			// activate AWS Session only if credentials present
			loaderSess = session.Must(session.NewSession(cfg))
		}
		if *awsStorageRegion != "" && *awsStorageAccessKeyID != "" && *awsStorageSecretAccessKey != "" {
			cfg := &aws.Config{
				Endpoint: s3StorageEndpoint,
				Region:   awsStorageRegion,
				Credentials: credentials.NewStaticCredentials(
					*awsStorageAccessKeyID, *awsStorageSecretAccessKey, *awsStorageSessionToken),
				S3ForcePathStyle: s3ForcePathStyle,
			}
			// activate AWS Session only if credentials present
			storageSess = session.Must(session.NewSession(cfg))
		}
		if *awsResultStorageRegion != "" && *awsResultStorageAccessKeyID != "" && *awsResultStorageSecretAccessKey != "" {
			cfg := &aws.Config{
				Endpoint: s3ResultStorageEndpoint,
				Region:   awsResultStorageRegion,
				Credentials: credentials.NewStaticCredentials(
					*awsResultStorageAccessKeyID, *awsResultStorageSecretAccessKey, *awsResultStorageSessionToken),
				S3ForcePathStyle: s3ForcePathStyle,
			}
			// activate AWS Session only if credentials present
			resultStorageSess = session.Must(session.NewSession(cfg))
		}
		if storageSess != nil && *s3StorageBucket != "" {
			// activate S3 Storage only if bucket config presents
			app.Storages = append(app.Storages,
				s3storage.New(storageSess, *s3StorageBucket,
					s3storage.WithPathPrefix(*s3StoragePathPrefix),
					s3storage.WithBaseDir(*s3StorageBaseDir),
					s3storage.WithACL(*s3StorageACL),
					s3storage.WithSafeChars(*s3SafeChars),
					s3storage.WithExpiration(*s3StorageExpiration),
				),
			)
		}
		if loaderSess != nil && *s3LoaderBucket != "" {
			// activate S3 Loader only if bucket config presents
			app.Loaders = append(app.Loaders,
				s3storage.New(loaderSess, *s3LoaderBucket,
					s3storage.WithPathPrefix(*s3LoaderPathPrefix),
					s3storage.WithBaseDir(*s3LoaderBaseDir),
					s3storage.WithSafeChars(*s3SafeChars),
				),
			)
		}
		if resultStorageSess != nil && *s3ResultStorageBucket != "" {
			// activate S3 ResultStorage only if bucket config presents
			app.ResultStorages = append(app.ResultStorages,
				s3storage.New(resultStorageSess, *s3ResultStorageBucket,
					s3storage.WithPathPrefix(*s3ResultStoragePathPrefix),
					s3storage.WithBaseDir(*s3ResultStorageBaseDir),
					s3storage.WithACL(*s3ResultStorageACL),
					s3storage.WithSafeChars(*s3SafeChars),
					s3storage.WithExpiration(*s3ResultStorageExpiration),
				),
			)
		}
	}
}
