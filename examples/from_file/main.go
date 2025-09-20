package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/cshum/vipsgen/vips"
)

// downloadFile helper function to download file from url
func downloadFile(url string, filepath string) error {
	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}
	return nil
}

func main() {
	err := downloadFile("https://raw.githubusercontent.com/cshum/imagor/master/testdata/dancing-banana.gif", "dancing-banana.gif")
	if err != nil {
		log.Fatalf("Failed to fetch image: %v", err)
	}
	// Create vips Image from file
	image, err := vips.NewImageFromFile("dancing-banana.gif", &vips.LoadOptions{
		N: -1, // load all pages a.k.a animation frames from gif
	})
	if err != nil {
		log.Fatalf("Failed to load image: %v", err)
	}
	defer image.Close()
	log.Printf("Loaded image: %s %dx%d\n", image.Format(), image.Width(), image.Height())
	// crop with animation support
	if err = image.ExtractAreaMultiPage(30, 40, 50, 70); err != nil {
		log.Fatalf("Failed to crop image: %v", err)
	}
	// Flatten image with cyan background
	if err = image.Flatten(&vips.FlattenOptions{Background: []float64{0, 255, 255}}); err != nil {
		log.Fatalf("Failed to flatten image: %v", err)
	}
	log.Printf("Processed image: %dx%d\n", image.Width(), image.Height())
	err = image.Gifsave("dancing-banana-cropped.gif", nil)
	if err != nil {
		log.Fatalf("Failed to save image: %v", err)
	}
	log.Println("Successfully saved processed images")
}
