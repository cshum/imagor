package vipsprocessor

import (
	"image/color"
	"strings"

	"github.com/cshum/vipsgen/vips"
	"golang.org/x/image/colornames"
)

// newColorImage creates a solid color vips.Image with the given RGBA color and dimensions.
func newColorImage(width, height int, c []float64) (*vips.Image, error) {
	hasAlpha := len(c) >= 4 && c[3] < 255

	// Create a 3-band black image using vips native operations
	img, err := vips.NewBlack(width, height, &vips.BlackOptions{Bands: 3})
	if err != nil {
		return nil, err
	}

	// Cast to uchar interpretation sRGB
	if err := img.Cast(vips.BandFormatUchar, nil); err != nil {
		img.Close()
		return nil, err
	}

	// Add color using Linear: pixel = pixel * 1 + offset
	if err := img.Linear([]float64{1, 1, 1}, []float64{c[0], c[1], c[2]}, nil); err != nil {
		img.Close()
		return nil, err
	}

	// Cast back to uchar after Linear (which produces float output)
	if err := img.Cast(vips.BandFormatUchar, nil); err != nil {
		img.Close()
		return nil, err
	}

	// Copy with sRGB interpretation to ensure proper export
	copied, err := img.Copy(&vips.CopyOptions{Interpretation: vips.InterpretationSrgb})
	if err != nil {
		img.Close()
		return nil, err
	}
	img.Close()
	img = copied

	// Add alpha channel if needed
	if hasAlpha {
		alpha, err := vips.NewBlack(width, height, nil)
		if err != nil {
			img.Close()
			return nil, err
		}
		if err := alpha.Cast(vips.BandFormatUchar, nil); err != nil {
			img.Close()
			alpha.Close()
			return nil, err
		}
		if err := alpha.Linear([]float64{1}, []float64{c[3]}, nil); err != nil {
			img.Close()
			alpha.Close()
			return nil, err
		}
		joined, err := vips.NewBandjoin([]*vips.Image{img, alpha})
		if err != nil {
			img.Close()
			alpha.Close()
			return nil, err
		}
		img.Close()
		alpha.Close()
		img = joined
	}

	return img, nil
}

// getColor parses a color string and returns RGB values as float64 slice
// Supports: color names (e.g., "red"), hex codes (e.g., "#ff0000" or "ff0000"),
// and "auto" which samples the image at top-left or bottom-right
func getColor(img *vips.Image, color string) []float64 {
	var vc = make([]float64, 3)
	args := strings.Split(strings.ToLower(color), ",")
	mode := ""
	name := strings.TrimPrefix(args[0], "#")
	if len(args) > 1 {
		mode = args[1]
	}
	if name == "auto" {
		if img != nil {
			x := 0
			y := 0
			if mode == "bottom-right" {
				x = img.Width() - 1
				y = img.PageHeight() - 1
			}
			p, _ := img.Getpoint(x, y, nil)
			if len(p) >= 3 {
				vc[0] = p[0]
				vc[1] = p[1]
				vc[2] = p[2]
			}
		}
	} else if c, ok := colornames.Map[name]; ok {
		vc[0] = float64(c.R)
		vc[1] = float64(c.G)
		vc[2] = float64(c.B)
	} else if c, ok := parseHexColor(name); ok {
		vc[0] = float64(c.R)
		vc[1] = float64(c.G)
		vc[2] = float64(c.B)
	}
	return vc
}

// parseHexColor parses a hex color string (3, 6, or 8 characters) into RGBA
func parseHexColor(s string) (c color.RGBA, ok bool) {
	c.A = 0xff
	switch len(s) {
	case 8:
		c.R = hexToByte(s[0])<<4 + hexToByte(s[1])
		c.G = hexToByte(s[2])<<4 + hexToByte(s[3])
		c.B = hexToByte(s[4])<<4 + hexToByte(s[5])
		c.A = hexToByte(s[6])<<4 + hexToByte(s[7])
		ok = true
	case 6:
		c.R = hexToByte(s[0])<<4 + hexToByte(s[1])
		c.G = hexToByte(s[2])<<4 + hexToByte(s[3])
		c.B = hexToByte(s[4])<<4 + hexToByte(s[5])
		ok = true
	case 3:
		c.R = hexToByte(s[0]) * 17
		c.G = hexToByte(s[1]) * 17
		c.B = hexToByte(s[2]) * 17
		ok = true
	}
	return
}

// hexToByte converts a single hex character to its byte value
func hexToByte(b byte) byte {
	switch {
	case b >= '0' && b <= '9':
		return b - '0'
	case b >= 'a' && b <= 'f':
		return b - 'a' + 10
	}
	return 0
}

// getColorRGBA parses a color string and returns RGBA values as float64 slice [R, G, B, A].
// Supports: color names (e.g., "red"), hex codes (3, 6, or 8 characters),
// and "transparent" keyword for fully transparent.
func getColorRGBA(name string) (c []float64, ok bool) {
	name = strings.TrimPrefix(strings.ToLower(strings.TrimSpace(name)), "#")
	if name == "" {
		return
	}
	if name == "transparent" || name == "none" {
		return []float64{0, 0, 0, 0}, true
	}
	if nc, found := colornames.Map[name]; found {
		return []float64{float64(nc.R), float64(nc.G), float64(nc.B), float64(nc.A)}, true
	}
	if hc, found := parseHexColor(name); found {
		return []float64{float64(hc.R), float64(hc.G), float64(hc.B), float64(hc.A)}, true
	}
	return
}

const colorImagePrefix = "color:"

// parseColorImage checks if the image path is a color image specification (color:xxx)
// and returns the RGBA color values if matched.
func parseColorImage(image string) (c []float64, ok bool) {
	if !strings.HasPrefix(strings.ToLower(image), colorImagePrefix) {
		return
	}
	return getColorRGBA(image[len(colorImagePrefix):])
}

// isBlack checks if a color is pure black (0, 0, 0)
func isBlack(c []float64) bool {
	if len(c) < 3 {
		return false
	}
	return c[0] == 0x00 && c[1] == 0x00 && c[2] == 0x00
}

// isWhite checks if a color is pure white (255, 255, 255)
func isWhite(c []float64) bool {
	if len(c) < 3 {
		return false
	}
	return c[0] == 0xff && c[1] == 0xff && c[2] == 0xff
}
