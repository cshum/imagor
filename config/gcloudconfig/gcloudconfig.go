package gcloudconfig

import (
	"cloud.google.com/go/storage"
	"context"
	"flag"
	"github.com/cshum/imagor"
	"github.com/cshum/imagor/config"
	"github.com/cshum/imagor/storage/gcloudstorage"
)

func WithGCloud(fs *flag.FlagSet, cb config.Callback) imagor.Option {
	var (
		gcloudSafeChars = fs.String("gcloud-safe-chars", "",
			"Google Cloud safe characters to be excluded from image key escape")

		gcloudLoaderBucket = fs.String("gcloud-loader-bucket", "",
			"Bucket name for Google Cloud Storage Loader. Enable Google Cloud Loader only if this value present")
		gcloudLoaderBaseDir = fs.String("gcloud-loader-base-dir", "",
			"Base directory for Google Cloud Loader")
		gcloudLoaderPathPrefix = fs.String("gcloud-loader-path-prefix", "",
			"Base path prefix for Google Cloud Loader")

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

		_, _ = cb()
	)
	return func(app *imagor.Imagor) {
		if *gcloudStorageBucket != "" || *gcloudLoaderBucket != "" || *gcloudResultStorageBucket != "" {
			// Activate the session, will panic if credentials are missing
			// Google cloud uses credentials from GOOGLE_APPLICATION_CREDENTIALS env file
			gcloudClient, err := storage.NewClient(context.Background())
			if err != nil {
				panic(err)
			}
			if *gcloudStorageBucket != "" {
				// activate Google Cloud Storage only if bucket config presents
				app.Storages = append(app.Storages,
					gcloudstorage.New(gcloudClient, *gcloudStorageBucket,
						gcloudstorage.WithPathPrefix(*gcloudStoragePathPrefix),
						gcloudstorage.WithBaseDir(*gcloudStorageBaseDir),
						gcloudstorage.WithACL(*gcloudStorageACL),
						gcloudstorage.WithSafeChars(*gcloudSafeChars),
						gcloudstorage.WithExpiration(*gcloudStorageExpiration),
					),
				)
			}

			if *gcloudLoaderBucket != "" {
				// activate Google Cloud Loader only if bucket config presents
				if *gcloudLoaderPathPrefix != *gcloudStoragePathPrefix ||
					*gcloudLoaderBucket != *gcloudStorageBucket ||
					*gcloudLoaderBaseDir != *gcloudStorageBaseDir {
					// create another loader if different from storage
					app.Loaders = append(app.Loaders,
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
				app.ResultStorages = append(app.ResultStorages,
					gcloudstorage.New(gcloudClient, *gcloudResultStorageBucket,
						gcloudstorage.WithPathPrefix(*gcloudResultStoragePathPrefix),
						gcloudstorage.WithBaseDir(*gcloudResultStorageBaseDir),
						gcloudstorage.WithACL(*gcloudResultStorageACL),
						gcloudstorage.WithSafeChars(*gcloudSafeChars),
						gcloudstorage.WithExpiration(*gcloudResultStorageExpiration),
					),
				)
			}
		}
	}
}
