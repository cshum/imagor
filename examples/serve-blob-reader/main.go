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
	"strconv"
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
	// serve via io.ReadCloser Blob
	in := imagor.NewBlob(func() (reader io.ReadCloser, size int64, err error) {
		var resp *http.Response
		resp, err = http.Get("https://raw.githubusercontent.com/cshum/imagor/master/testdata/dancing-banana.gif")
		reader = resp.Body
		size, _ = strconv.ParseInt(resp.Header.Get("Content-Length"), 10, 64)
		// known size via Content-Length header
		// size is optional; providing size enables better throughput and memory allocations
		return
	})
	out, err := app.ServeBlob(ctx, in, imagorpath.Params{
		Unsafe: true,
		Width:  200,
		Height: 150,
		FitIn:  true,
		Filters: []imagorpath.Filter{
			{"fill", "yellow"},
			{"watermark", "https://raw.githubusercontent.com/cshum/imagor/master/testdata/gopher-front.png,repeat,bottom,0,40,40"},
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
	file, err := os.Create("dancing-banana.gif")
	if err != nil {
		panic(err)
	}
	defer file.Close()
	if _, err := io.Copy(file, reader); err != nil {
		panic(err)
	}
}
