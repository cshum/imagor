package main

import (
	"fmt"
	"github.com/cshum/govips/v2/vips"
	"github.com/cshum/imagor"
	"github.com/cshum/imagor/loader/httploader"
	"github.com/cshum/imagor/processor/vipsprocessor"
	"github.com/cshum/imagor/store/filestore"
	"go.uber.org/zap"
	"net/http"
)

func main() {
	var (
		logger *zap.Logger
		err    error
		port   = 9000
	)
	if logger, err = zap.NewDevelopment(); err != nil {
		panic(err)
	}
	logger.Info("start", zap.Int("port", port))

	vips.Startup(nil)
	defer vips.Shutdown()

	store := filestore.New("./images")

	panic(http.ListenAndServe(
		fmt.Sprintf(":%d", port),
		imagor.New(
			imagor.WithLogger(logger),
			imagor.WithLoaders(
				store,
				httploader.New(
					httploader.WithForwardHeaders("*"),
					httploader.WithAutoScheme(true),
				),
			),
			imagor.WithStorages(store),
			imagor.WithProcessors(vipsprocessor.New()),
			imagor.WithSecret(""),
			imagor.WithUnsafe(true),
		),
	))
}
