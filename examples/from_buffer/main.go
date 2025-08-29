package main

import (
	"fmt"
	_ "image/png"
	"io"
	"log"
	"net/http"

	"github.com/cshum/vipsgen/vips"
)

func getBytesFromURL(url string) ([]byte, error) {
	// Make HTTP GET request
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %v", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bad status: %s", resp.Status)
	}

	// Read entire response body into bytes
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %v", err)
	}

	return data, nil
}

func main() {
	buf, err := getBytesFromURL("https://raw.githubusercontent.com/cshum/imagor/master/testdata/gopher.png")
	if err != nil {
		log.Fatalf("Failed to fetch image: %v", err)
	}

	// Create vips.Image from Go image
	image, err := vips.NewImageFromBuffer(buf, nil)
	if err != nil {
		log.Fatalf("Failed to load image: %v", err)
	}
	defer image.Close()
	log.Printf("Loaded image: %s %dx%d\n", image.Format(), image.Width(), image.Height())
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
