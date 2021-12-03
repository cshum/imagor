package main

import (
	"github.com/cshum/imagor"
	"github.com/cshum/imagor/loader/httploader"
	"github.com/cshum/imagor/processor/vipsprocessor"
	"github.com/cshum/imagor/server"
	"github.com/cshum/imagor/store/filestore"
	"go.uber.org/zap"
)

func main() {
	var (
		debug    = true
		logger   *zap.Logger
		err      error
		loaders  []imagor.Loader
		storages []imagor.Storage
	)
	if debug {
		if logger, err = zap.NewDevelopment(); err != nil {
			panic(err)
		}
	} else {
		if logger, err = zap.NewProduction(); err != nil {
			panic(err)
		}
	}

	loaders = append(loaders,
		filestore.New("./"))

	storages = append(storages,
		filestore.New("./"))

	loaders = append(loaders,
		httploader.New(
			httploader.WithForwardUserAgent(true),
			httploader.WithForwardAllHeaders(false),
			httploader.WithForwardHeaders(""),
			httploader.WithAllowedSources(""),
			httploader.WithMaxAllowedSize(0),
			httploader.WithInsecureSkipVerifyTransport(false),
		),
	)

	server.New(
		imagor.New(
			imagor.WithLoaders(loaders...),
			imagor.WithStorages(storages...),
			imagor.WithProcessors(vipsprocessor.New(
				vipsprocessor.WithLogger(logger),
				vipsprocessor.WithDebug(debug),
			)),
			imagor.WithSecret(""),
			imagor.WithRequestTimeout(0),
			imagor.WithUnsafe(true),
			imagor.WithLogger(logger),
			imagor.WithDebug(debug),
		),
		server.WithAddress(""),
		server.WithPort(9000),
		server.WithPathPrefix(""),
		server.WithCORS(true),
		server.WithLogger(logger),
		server.WithDebug(debug),
	).Run()
}
