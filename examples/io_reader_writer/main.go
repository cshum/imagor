package main

import (
	"log"
	"net/http"
	"os"

	"github.com/cshum/vipsgen/vips"
)

func main() {
	// Fetch an image from http.Get
	resp, err := http.Get("https://raw.githubusercontent.com/cshum/imagor/master/testdata/gopher.png")
	if err != nil {
		log.Fatalf("Failed to fetch image: %v", err)
	}
	defer resp.Body.Close()

	// Create source from io.ReadCloser
	source := vips.NewSource(resp.Body)
	defer source.Close() // source needs to remain available during image lifetime

	// Shrink-on-load via creating image from thumbnail source with options
	image, err := vips.NewThumbnailSource(source, 800, &vips.ThumbnailSourceOptions{
		Height: 1000,
		FailOn: vips.FailOnError, // Fail on first error
	})
	if err != nil {
		log.Fatalf("Failed to load image: %v", err)
	}
	defer image.Close() // always close images to free memory

	log.Printf("Loaded image: %s %dx%d\n", image.Format(), image.Width(), image.Height())

	// Add a yellow border using vips_embed
	border := 10
	if err := image.Embed(
		border, border,
		image.Width()+border*2,
		image.Height()+border*2,
		&vips.EmbedOptions{
			Extend:     vips.ExtendBackground,       // extend with colour from the background property
			Background: []float64{255, 255, 0, 255}, // Yellow border
		},
	); err != nil {
		log.Fatalf("Failed to add border: %v", err)
	}

	log.Printf("Processed image: %dx%d\n", image.Width(), image.Height())

	// Create io.WriteCloser from create file
	file, err := os.Create("resized-gopher.webp")
	defer file.Close()

	// Create target from io.WriteCloser
	target := vips.NewTarget(file)
	defer target.Close()

	// Save the result as WebP target with options
	err = image.WebpsaveTarget(target, &vips.WebpsaveTargetOptions{
		Q:              85,   // Quality factor (0-100)
		Effort:         4,    // Compression effort (0-6)
		SmartSubsample: true, // Better chroma subsampling
	})
	if err != nil {
		log.Fatalf("Failed to save image as WebP: %v", err)
	}
	log.Println("Successfully saved processed images")
}
