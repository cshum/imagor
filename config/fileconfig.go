package config

import (
	"flag"
	"github.com/cshum/imagor"
	"github.com/cshum/imagor/storage/filestorage"
	"go.uber.org/zap"
)

// withFileSystem with File Loader, Storage, Result Storage based config option
func withFileSystem(fs *flag.FlagSet, cb func() (*zap.Logger, bool)) imagor.Option {
	var (
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

		_, _ = cb()
	)
	return func(o *imagor.Imagor) {
		if *fileStorageBaseDir != "" {
			// activate File Storage only if base dir config presents
			o.Storages = append(o.Storages,
				filestorage.New(
					*fileStorageBaseDir,
					filestorage.WithPathPrefix(*fileStoragePathPrefix),
					filestorage.WithMkdirPermission(*fileStorageMkdirPermission),
					filestorage.WithWritePermission(*fileStorageWritePermission),
					filestorage.WithSafeChars(*fileSafeChars),
					filestorage.WithExpiration(*fileStorageExpiration),
				),
			)
		}
		if *fileLoaderBaseDir != "" {
			// activate File Loader only if base dir config presents
			o.Loaders = append(o.Loaders,
				filestorage.New(
					*fileLoaderBaseDir,
					filestorage.WithPathPrefix(*fileLoaderPathPrefix),
					filestorage.WithSafeChars(*fileSafeChars),
				),
			)
		}
		if *fileResultStorageBaseDir != "" {
			// activate File Result Storage only if base dir config presents
			o.ResultStorages = append(o.ResultStorages,
				filestorage.New(
					*fileResultStorageBaseDir,
					filestorage.WithPathPrefix(*fileResultStoragePathPrefix),
					filestorage.WithMkdirPermission(*fileResultStorageMkdirPermission),
					filestorage.WithWritePermission(*fileResultStorageWritePermission),
					filestorage.WithSafeChars(*fileSafeChars),
					filestorage.WithExpiration(*fileResultStorageExpiration),
				),
			)
		}

	}
}
