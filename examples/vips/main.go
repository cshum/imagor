package main

import (
	"github.com/cshum/imagor/vips"
	"net/http"
	"os"
)

func main() {
	vips.Startup(nil)
	defer vips.Shutdown()

	resp, err := http.Get("https://raw.githubusercontent.com/cshum/imagor/master/testdata/dancing-banana.gif")
	if err != nil {
		panic(err)
	}
	source := vips.NewSource(resp.Body)
	defer source.Close()

	params := vips.NewImportParams()
	params.NumPages.Set(-1) // enable animation
	image, err := source.LoadImage(params)
	if err != nil {
		panic(err)
	}
	if err = image.ExtractArea(30, 40, 50, 70); err != nil {
		panic(err)
	}
	if err = image.Flatten(&vips.Color{
		R: 0, G: 255, B: 255,
	}); err != nil {
		panic(err)
	}
	buf, err := image.ExportGIF(nil)
	if err != nil {
		panic(err)
	}
	if err = os.WriteFile("dancing-banana.gif", buf, 0666); err != nil {
		panic(err)
	}
}
