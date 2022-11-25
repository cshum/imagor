package main

import (
	"context"
	"github.com/cshum/imagor"
	"github.com/cshum/imagor/imagorpath"
	"github.com/cshum/imagor/loader/httploader"
	"github.com/cshum/imagor/vips"
	"io"
	"net/http"
	"os"
)

func main() {
	app := imagor.New(
		imagor.WithUnsafe(true),
		imagor.WithLoaders(httploader.New()),
		imagor.WithProcessors(vips.NewProcessor()),
	)
	ctx := context.Background()
	if err := app.Startup(ctx); err != nil {
		panic(err)
	}
	defer app.Shutdown(ctx)

	downloadFile("https://raw.githubusercontent.com/cshum/imagor/master/testdata/gopher.png", "gopher.png")

	in := imagor.NewBlobFromFile("gopher.png")

	// serve via image path
	out, err := app.ServeBlob(ctx, in, imagorpath.Params{
		Unsafe: true,
		Width:  500,
		Height: 500,
		Smart:  true,
		Filters: []imagorpath.Filter{
			{"fill", "yellow"},
			{"format", "jpg"},
		},
	})
	if err != nil {
		panic(err)
	}
	reader, _, err := out.NewReader()
	if err != nil {
		panic(err)
	}
	defer reader.Close()
	file, err := os.Create("gopher.jpg")
	if err != nil {
		panic(err)
	}
	defer file.Close()
	if _, err := io.Copy(file, reader); err != nil {
		panic(err)
	}
}

func downloadFile(urlpath, filepath string) {
	resp, err := http.Get(urlpath)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	file, err := os.Create(filepath)
	if err != nil {
		panic(err)
	}
	defer file.Close()
	if _, err := io.Copy(file, resp.Body); err != nil {
		panic(err)
	}
}
