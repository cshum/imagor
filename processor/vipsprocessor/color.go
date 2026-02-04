package vipsprocessor

import (
	"image/color"
	"strings"

	"github.com/cshum/vipsgen/vips"
	"golang.org/x/image/colornames"
)

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

// parseHexColor parses a hex color string (3 or 6 characters) into RGBA
func parseHexColor(s string) (c color.RGBA, ok bool) {
	c.A = 0xff
	switch len(s) {
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
