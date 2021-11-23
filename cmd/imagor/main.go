package main

import (
	"github.com/cshum/imagor"
	"github.com/cshum/imagor/source/httpsource"
	"go.uber.org/zap"
)

func main() {
	var (
		logger *zap.Logger
		err    error
		srv    *imagor.Server
	)
	if logger, err = zap.NewProduction(); err != nil {
		panic(err)
	}
	srv = &imagor.Server{
		Port: 3000,
		Sources: []imagor.Source{
			httpsource.HTTPSource{},
		},
		Logger: logger,
	}
	panic(srv.Run())
}
