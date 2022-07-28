package awsconfig

import (
	"flag"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/cshum/imagor"
	"github.com/cshum/imagor/config"
	"github.com/cshum/imagor/storage/s3storage"
)

func WithAWS(fs *flag.FlagSet, cb config.Callback) imagor.Option {
	var (
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
		if *awsRegion != "" && *awsAccessKeyId != "" && *awsSecretAccessKey != "" {
			cfg := &aws.Config{
				Endpoint: s3Endpoint,
				Region:   awsRegion,
				Credentials: credentials.NewStaticCredentials(
					*awsAccessKeyId, *awsSecretAccessKey, ""),
			}
			if *s3ForcePathStyle {
				cfg.WithS3ForcePathStyle(true)
			}
			// activate AWS Session only if credentials present
			sess, err := session.NewSession(cfg)
			if err != nil {
				panic(err)
			}
			if *s3StorageBucket != "" {
				// activate S3 Storage only if bucket config presents
				app.Storages = append(app.Storages,
					s3storage.New(sess, *s3StorageBucket,
						s3storage.WithPathPrefix(*s3StoragePathPrefix),
						s3storage.WithBaseDir(*s3StorageBaseDir),
						s3storage.WithACL(*s3StorageACL),
						s3storage.WithSafeChars(*s3SafeChars),
						s3storage.WithExpiration(*s3StorageExpiration),
					),
				)
			}
			if *s3LoaderBucket != "" {
				// activate S3 Loader only if bucket config presents
				if *s3LoaderPathPrefix != *s3StoragePathPrefix ||
					*s3LoaderBucket != *s3StorageBucket ||
					*s3LoaderBaseDir != *s3StorageBaseDir {
					// create another loader if different from storage
					app.Loaders = append(app.Loaders,
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
				app.ResultStorages = append(app.ResultStorages,
					s3storage.New(sess, *s3ResultStorageBucket,
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
}
