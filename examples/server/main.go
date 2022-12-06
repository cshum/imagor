package main

import (
	"github.com/cshum/imagor"
	"github.com/cshum/imagor/imagorpath"
	"github.com/cshum/imagor/loader/httploader"
	"github.com/cshum/imagor/server"
	"github.com/cshum/imagor/storage/filestorage"
	"github.com/cshum/imagor/vips"
	"go.uber.org/zap"
)

func main() {
	logger := zap.Must(zap.NewProduction())

	// create and run imagor server programmatically
	server.New(
		imagor.New(
			imagor.WithLogger(logger),
			imagor.WithUnsafe(true),
			imagor.WithProcessors(vips.NewProcessor()),
			imagor.WithLoaders(httploader.New()),
			imagor.WithStorages(filestorage.New("./")),
			imagor.WithResultStorages(filestorage.New("./")),
			imagor.WithResultStoragePathStyle(imagorpath.SuffixResultStorageHasher),
		),
		server.WithPort(8000),
		server.WithLogger(logger),
	).Run()
}
