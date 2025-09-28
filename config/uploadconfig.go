package config

import (
	"flag"

	"github.com/cshum/imagor"
	"github.com/cshum/imagor/loader/uploadloader"
	"go.uber.org/zap"
)

// withUploadLoader with Upload Loader config option
func withUploadLoader(fs *flag.FlagSet, cb func() (*zap.Logger, bool)) imagor.Option {
	var (
		uploadLoaderMaxAllowedSize = fs.Int("upload-loader-max-allowed-size", 32<<20,
			"Upload Loader maximum allowed size in bytes for uploaded images")
		uploadLoaderAccept = fs.String("upload-loader-accept", "image/*",
			"Upload Loader accepted Content-Type for uploads")
		uploadLoaderFormFieldName = fs.String("upload-loader-form-field-name", "image",
			"Upload Loader form field name for multipart uploads")
		uploadLoaderDisable = fs.Bool("upload-loader-disable", false,
			"Disable Upload Loader")
	)
	_, _ = cb()
	return func(app *imagor.Imagor) {
		if !*uploadLoaderDisable {
			// Add Upload Loader for POST uploads when unsafe mode is enabled
			app.Loaders = append(app.Loaders,
				uploadloader.New(
					uploadloader.WithMaxAllowedSize(*uploadLoaderMaxAllowedSize),
					uploadloader.WithAccept(*uploadLoaderAccept),
					uploadloader.WithFormFieldName(*uploadLoaderFormFieldName),
				),
			)
		}
	}
}
