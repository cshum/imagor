package main

import (
	"github.com/cshum/imagor"
	"github.com/cshum/imagor/loader/httploader"
	"github.com/cshum/imagor/processor/vipsprocessor"
	"github.com/cshum/imagor/server"
	"go.uber.org/zap"
)

func main() {
	var (
		logger   *zap.Logger
		err      error
		loaders  []imagor.Loader
		storages []imagor.Storage
	)
	if logger, err = zap.NewDevelopment(); err != nil {
		panic(err)
	}

	loaders = append(loaders,
		httploader.New(
			httploader.WithForwardHeaders("*"),
			httploader.WithAutoScheme(true),
		),
	)

	server.New(
		imagor.New(
			imagor.WithLogger(logger),
			imagor.WithLoaders(loaders...),
			imagor.WithStorages(storages...),
			imagor.WithProcessors(vipsprocessor.New()),
			imagor.WithSecret(""),
			imagor.WithUnsafe(true),
			imagor.WithDebug(true),
		),
		server.WithPort(9000),
		server.WithLogger(logger),
		server.WithCORS(true),
	).Run()
}
