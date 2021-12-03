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
		httploader.New(
			httploader.WithForwardUserAgent(true),
			httploader.WithForwardAllHeaders(false),
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
		server.WithReadTimeout(0),
		server.WithWriteTimeout(0),
		server.WithCORS(true),
		server.WithLogger(logger),
		server.WithDebug(debug),
	).Run()
}
