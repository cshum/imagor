package main

import (
	"github.com/cshum/imagor/server"
	"go.uber.org/zap"
)

func main() {
	var (
		logger *zap.Logger
		err    error
		srv    *server.Server
	)
	if logger, err = zap.NewProduction(); err != nil {
		panic(err)
	}
	srv = &server.Server{
		Port:   3000,
		Logger: logger,
	}
	panic(srv.Run())
}
