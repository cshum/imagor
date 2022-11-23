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
		imagor.WithProcessors(vips.NewProcessor()),
		imagor.WithUnsafe(true),
		imagor.WithLoaders(httploader.New()),
	)
	ctx := context.Background()
	if err := app.Startup(ctx); err != nil {
		panic(err)
	}
	defer app.Shutdown(ctx)
	r, err := http.NewRequestWithContext(ctx, http.MethodGet, "", nil)
	if err != nil {
		panic(err)
	}
	blob, err := app.Do(r, imagorpath.Params{
		Unsafe: true,
		Image:  "https://raw.githubusercontent.com/cshum/imagor/master/testdata/gopher.png",
		Width:  500,
		Height: 500,
		Smart:  true,
		Filters: []imagorpath.Filter{
			{"fill", "white"},
			{"format", "jpg"},
		},
	})
	if err != nil {
		panic(err)
	}
	reader, _, err := blob.NewReader()
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
