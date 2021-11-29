package main

import (
	"fmt"
	"github.com/cshum/hybridcache"
	"github.com/cshum/imagor"
	"github.com/cshum/imagor/loader/httploader"
	"github.com/cshum/imagor/processor/vipsprocessor"
	"go.uber.org/zap"
	"net/http"
	"time"
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

	panic(http.ListenAndServe(
		fmt.Sprintf(":%d", port),
		&imagor.Imagor{
			Cache: cache.NewMemory(1000, 1<<28, time.Minute*5),
			Loaders: []imagor.Loader{
				&httploader.HTTPLoader{
					ForwardHeaders: []string{"*"},
				},
			},
			Processors: []imagor.Processor{
				vipsprocessor.New(),
			},
			Unsafe:  true,
			Logger:  logger,
			Timeout: time.Second * 30,
		},
	))
}
