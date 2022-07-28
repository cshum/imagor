package main

import (
	"github.com/cshum/imagor/module"
	"github.com/cshum/imagor/module/awsmodule"
	"github.com/cshum/imagor/module/gcloudmodule"
	"github.com/cshum/imagor/module/vipsmodule"
	"os"
)

func main() {
	var server = module.Do(os.Args[1:], vipsmodule.WithVips, awsmodule.WithAWS, gcloudmodule.WithGCloud)
	if server != nil {
		server.Run()
	}
}
