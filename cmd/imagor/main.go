package main

import (
	"fmt"
	"github.com/cshum/imagor"
	"github.com/cshum/imagor/store/httpstore"
	"go.uber.org/zap"
	"net/http"
	"time"
)

func main() {
	var (
		logger *zap.Logger
		err    error
		port   = 3000
	)
	if logger, err = zap.NewDevelopment(); err != nil {
		panic(err)
	}
	logger.Info("start", zap.Int("port", port))
	panic(http.ListenAndServe(
		fmt.Sprintf(":%d", port),
		&imagor.Imagor{
			Loaders: []imagor.Loader{
				httpstore.HTTPStore{
					ForwardHeaders: []string{"*"},
				},
			},
			Unsafe:  true,
			Logger:  logger,
			Timeout: time.Second * 30,
		},
	))
}
