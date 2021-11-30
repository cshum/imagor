package main

import (
	"fmt"
	"github.com/cshum/govips/v2/vips"
	"github.com/cshum/imagor"
	"github.com/cshum/imagor/loader/httploader"
	"github.com/cshum/imagor/processor/vipsprocessor"
	"go.uber.org/zap"
	"net/http"
)

func main() {
	var (
		logger   *zap.Logger
		err      error
		port     = 9000
		loaders  []imagor.Loader
		storages []imagor.Storage
	)
	if logger, err = zap.NewDevelopment(); err != nil {
		panic(err)
	}
	logger.Info("start", zap.Int("port", port))

	vips.Startup(nil)
	defer vips.Shutdown()

	loaders = append(loaders,
		httploader.New(
			httploader.WithForwardHeaders("*"),
			httploader.WithAutoScheme(true),
		),
	)

	panic(http.ListenAndServe(
		fmt.Sprintf(":%d", port),
		imagor.New(
			imagor.WithLogger(logger),
			imagor.WithLoaders(loaders...),
			imagor.WithStorages(storages...),
			imagor.WithProcessors(vipsprocessor.New()),
			imagor.WithSecret(""),
			imagor.WithUnsafe(true),
			imagor.WithDebug(true),
		),
	))
}
