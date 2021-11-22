package main

import (
	"github.com/cshum/imagor/server"
	"go.uber.org/zap"
)

func main() {
	var (
		logger *zap.Logger
		err    error
		srv    *server.HTTP
	)
	if logger, err = zap.NewProduction(); err != nil {
		panic(err)
	}
	srv = &server.HTTP{
		Port:   3000,
		Logger: logger,
	}
	panic(srv.Run())
}
