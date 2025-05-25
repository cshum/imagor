package main

import (
	"github.com/cshum/vipsgen/vips"
	"net/http"
)

func main() {
	// manipulate images using libvips C bindings
	vips.Startup(nil)
	defer vips.Shutdown()

	// create source from io.ReadCloser
	resp, err := http.Get("https://raw.githubusercontent.com/cshum/imagor/master/testdata/dancing-banana.gif")
	if err != nil {
		panic(err)
	}
	source := vips.NewSource(resp.Body)
	defer source.Close() // source needs to remain available during the lifetime of image

	image, err := vips.NewImageFromSource(source, &vips.LoadOptions{N: -1})
	if err != nil {
		panic(err)
	}
	defer image.Close()
	if err = image.ExtractAreaMultiPage(30, 40, 50, 70); err != nil {
		panic(err)
	}
	if err = image.Flatten(&vips.FlattenOptions{Background: []float64{0, 255, 255}}); err != nil {
		panic(err)
	}
	err = image.Gifsave("dancing-banana.gif", nil)
	if err != nil {
		panic(err)
	}
}
