package main

import (
	"image"
	_ "image/png"
	"log"
	"net/http"

	"github.com/cshum/vipsgen/vips817"
)

func main() {
	// Create a Go image
	resp, err := http.Get("https://raw.githubusercontent.com/cshum/imagor/master/testdata/gopher.png")
	if err != nil {
		log.Fatalf("Failed to fetch image: %v", err)
	}
	defer resp.Body.Close()
	img, _, err := image.Decode(resp.Body)
	if err != nil {
		log.Fatalf("Failed to create Go image: %v", err)
	}
	nrgba := img.(*image.NRGBA)
	size := nrgba.Rect.Size()

	// Create vips.Image from Go image
	image, err := vips.NewImageFromMemory(nrgba.Pix, size.X, size.Y, 4)
	if err != nil {
		log.Fatalf("Failed to load image: %v", err)
	}
	defer image.Close()
	log.Printf("Loaded image: %dx%d\n", image.Width(), image.Height())
	err = image.Resize(0.5, nil)
	if err != nil {
		log.Fatalf("Failed to resize image: %v", err)
	}
	err = image.Flatten(&vips.FlattenOptions{
		Background: []float64{255, 255, 0}, // yellow background
	})
	if err != nil {
		log.Fatalf("Failed to flatten image: %v", err)
	}
	log.Printf("Processed image: %dx%d\n", image.Width(), image.Height())
	err = image.Jpegsave("gopher.jpg", nil)
	if err != nil {
		log.Fatalf("Failed to save image: %v", err)
	}
	log.Println("Successfully saved processed images")
}
