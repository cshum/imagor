package main

import (
	"context"
	"github.com/cshum/imagor"
	"github.com/cshum/imagor/imagorpath"
	"github.com/cshum/imagor/loader/httploader"
	"github.com/cshum/imagor/processor/vipsprocessor"
	"image"
	"io"
	"net/http"
	"os"

	_ "image/png"
)

func main() {
	app := imagor.New(
		imagor.WithLoaders(httploader.New()),
		imagor.WithProcessors(vipsprocessor.NewProcessor()),
	)
	ctx := context.Background()
	if err := app.Startup(ctx); err != nil {
		panic(err)
	}
	defer app.Shutdown(ctx)

	resp, err := http.Get("https://raw.githubusercontent.com/cshum/imagor/master/testdata/gopher.png")
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	img, _, err := image.Decode(resp.Body)
	if err != nil {
		panic(err)
	}
	nrgba := img.(*image.NRGBA)
	size := nrgba.Rect.Size()

	in := imagor.NewBlobFromMemory(nrgba.Pix, size.X, size.Y, 4)

	// serve via image path
	out, err := app.ServeBlob(ctx, in, imagorpath.Params{
		Width:  500,
		Height: 500,
		HFlip:  true,
		FitIn:  true,
		Filters: []imagorpath.Filter{
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
