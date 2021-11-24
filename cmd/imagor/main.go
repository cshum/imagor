package main

import (
	"fmt"
	"github.com/cshum/imagor"
	"github.com/cshum/imagor/loader/httploader"
	"go.uber.org/zap"
	"net/http"
)

func main() {
	var (
		logger *zap.Logger
		err    error
		port   = 3000
	)
	if logger, err = zap.NewProduction(); err != nil {
		panic(err)
	}
	logger.Info("start", zap.Int("port", port))
	panic(http.ListenAndServe(
		fmt.Sprintf(":%d", port),
		&imagor.Imagor{
			Loaders: []imagor.Loader{
				httploader.HTTPLoader{
					ForwardHeaders: []string{"*"},
				},
			},
			Unsafe: true,
			Logger: logger,
		},
	))
}
