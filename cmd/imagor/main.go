package main

import (
	"github.com/cshum/imagor/config"
	"github.com/cshum/imagor/config/awsconfig"
	"github.com/cshum/imagor/config/gcloudconfig"
	"github.com/cshum/imagor/config/vipsconfig"
	"os"
)

func main() {
	var server = config.NewServer(
		os.Args[1:],
		vipsconfig.WithVips,
		awsconfig.WithAWS,
		gcloudconfig.WithGCloud,
	)
	if server != nil {
		server.Run()
	}
}
