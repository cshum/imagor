package vipsprocessor

import (
	"fmt"
	"github.com/cshum/govips/v2/vips"
	"github.com/cshum/imagor"
	"golang.org/x/image/colornames"
	"image/color"
	"net/url"
	"strconv"
	"strings"
)

func watermark(img *vips.ImageRef, load imagor.LoadFunc, args ...string) (err error) {
	ln := len(args)
	if ln < 1 {
		return
	}
	image := args[0]
	if unescape, e := url.QueryUnescape(args[0]); e == nil {
		image = unescape
	}
	var buf []byte
	if buf, err = load(image); err != nil {
		return
	}
	var overlay *vips.ImageRef
	if overlay, err = vips.NewImageFromBuffer(buf); err != nil {
		return
	}
	defer overlay.Close()
	var x, y, w, h int
	var repeatX = 1
	var repeatY = 1

	// w_ratio h_ratio
	if ln >= 6 {
		w = img.Width()
		h = img.Height()
		if args[4] != "none" {
			w, _ = strconv.Atoi(args[4])
			w = img.Width() * w / 100
		}
		if args[5] != "none" {
			h, _ = strconv.Atoi(args[5])
			h = img.Height() * h / 100
		}
		if w < overlay.Width() || h < overlay.Height() {
			if err = overlay.Thumbnail(w, h, vips.InterestingNone); err != nil {
				return
			}
		}
	}
	w = overlay.Width()
	h = overlay.Height()
	// alpha
	if ln >= 4 {
		alpha, _ := strconv.ParseFloat(args[3], 64)
		alpha = 1 - alpha/100
		if err = overlay.AddAlpha(); err != nil {
			return
		}
		if err = overlay.Linear([]float64{1, 1, 1, alpha}, []float64{0, 0, 0, 0}); err != nil {
			return
		}
	}
	// x y
	if ln >= 3 {
		if args[1] == "center" {
			x = (img.Width() - overlay.Width()) / 2
		} else if args[1] == "repeat" {
			x = 0
			repeatX = img.Width()/overlay.Width() + 1
		} else if strings.HasSuffix(args[1], "p") {
			x, _ = strconv.Atoi(strings.TrimSuffix(args[1], "p"))
			x = x * img.Width() / 100
		} else {
			x, _ = strconv.Atoi(args[1])
		}
		if args[2] == "center" {
			y = (img.Height() - overlay.Height()) / 2
		} else if args[2] == "repeat" {
			y = 0
			repeatY = img.Height()/overlay.Height() + 1
		} else if strings.HasSuffix(args[2], "p") {
			y, _ = strconv.Atoi(strings.TrimSuffix(args[2], "p"))
			y = y * img.Height() / 100
		} else {
			y, _ = strconv.Atoi(args[2])
		}
		if x < 0 {
			x += img.Width() - overlay.Width()
		}
		if y < 0 {
			y += img.Height() - overlay.Height()
		}
	}
	for i := 0; i < repeatX; i++ {
		for j := 0; j < repeatY; j++ {
			if err = img.Composite(
				overlay, vips.BlendModeOver, x+w*i, y+h*j); err != nil {
				return
			}
		}
	}

	return
}

func roundCorner(img *vips.ImageRef, _ imagor.LoadFunc, args ...string) (err error) {
	var rx, ry int
	var c *vips.Color
	if len(args) == 0 {
		return
	}
	if a, e := url.QueryUnescape(args[0]); e == nil {
		args[0] = a
	}
	if len(args) == 3 {
		// rx,ry,color
		c = getColor(args[2])
		args = args[:2]
	} else if len(args) >= 4 {
		// rx|ry,r,g,b
		r, _ := strconv.Atoi(args[1])
		g, _ := strconv.Atoi(args[2])
		b, _ := strconv.Atoi(args[3])
		c = &vips.Color{R: uint8(r), G: uint8(g), B: uint8(b)}
		args = args[:1]
	}
	// rx|ry
	if strings.Contains(args[0], "|") {
		args = append(strings.Split(args[0], "|"), args[1:]...)
	}
	rx, _ = strconv.Atoi(args[0])
	ry = rx
	if len(args) > 1 {
		rx, _ = strconv.Atoi(args[1])
	}

	var rounded *vips.ImageRef
	var w = img.Width()
	var h = img.Height()
	if rounded, err = vips.NewImageFromBuffer([]byte(fmt.Sprintf(`
		<svg viewBox="0 0 %d %d">
			<rect rx="%d" ry="%d" 
			 x="0" y="0" width="%d" height="%d" 
			 fill="#fff"/>
		</svg>
	`, w, h, rx, ry, w, h))); err != nil {
		return
	}
	defer rounded.Close()
	if err = img.Composite(rounded, vips.BlendModeDestIn, 0, 0); err != nil {
		return
	}
	if c != nil {
		if err = applyBackgroundColor(img, c); err != nil {
			return
		}
	}
	return nil
}

func backgroundColor(img *vips.ImageRef, _ imagor.LoadFunc, args ...string) (err error) {
	if len(args) == 0 {
		return
	}
	return applyBackgroundColor(img, getColor(args[0]))
}

func rotate(img *vips.ImageRef, _ imagor.LoadFunc, args ...string) (err error) {
	if len(args) == 0 {
		return
	}
	if angle, _ := strconv.Atoi(args[0]); angle > 0 {
		vAngle := vips.Angle0
		switch angle {
		case 90:
			vAngle = vips.Angle270
		case 180:
			vAngle = vips.Angle180
		case 270:
			vAngle = vips.Angle90
		}
		if err = img.Rotate(vAngle); err != nil {
			return err
		}
	}
	return
}

func grayscale(img *vips.ImageRef, _ imagor.LoadFunc, _ ...string) (err error) {
	return img.Modulate(1, 0, 0)
}

func brightness(img *vips.ImageRef, _ imagor.LoadFunc, args ...string) (err error) {
	if len(args) == 0 {
		return
	}
	b, _ := strconv.ParseFloat(args[0], 64)
	b = b * 256 / 100
	return img.Linear([]float64{1, 1, 1}, []float64{b, b, b})
}

func contrast(img *vips.ImageRef, _ imagor.LoadFunc, args ...string) (err error) {
	if len(args) == 0 {
		return
	}
	a, _ := strconv.ParseFloat(args[0], 64)
	a = a * 256 / 100
	b := 128 - a*128
	return img.Linear([]float64{a, a, a}, []float64{b, b, b})
}

func hue(img *vips.ImageRef, _ imagor.LoadFunc, args ...string) (err error) {
	if len(args) == 0 {
		return
	}
	h, _ := strconv.ParseFloat(args[0], 64)
	return img.Modulate(1, 1, h)
}

func saturation(img *vips.ImageRef, _ imagor.LoadFunc, args ...string) (err error) {
	if len(args) == 0 {
		return
	}
	s, _ := strconv.ParseFloat(args[0], 64)
	s = 1 + s/100
	return img.Modulate(1, s, 0)
}

func rgb(img *vips.ImageRef, _ imagor.LoadFunc, args ...string) (err error) {
	if len(args) != 3 {
		return
	}
	r, _ := strconv.ParseFloat(args[0], 64)
	g, _ := strconv.ParseFloat(args[1], 64)
	b, _ := strconv.ParseFloat(args[2], 64)
	r = r * 256 / 100
	g = g * 256 / 100
	b = b * 256 / 100
	return img.Linear([]float64{1, 1, 1}, []float64{r, g, b})
}

func blur(img *vips.ImageRef, _ imagor.LoadFunc, args ...string) (err error) {
	var sigma float64
	switch len(args) {
	case 2:
		sigma, _ = strconv.ParseFloat(args[1], 64)
		break
	case 1:
		sigma, _ = strconv.ParseFloat(args[0], 64)
		break
	}
	sigma /= 2
	if sigma > 0 {
		return img.GaussianBlur(sigma)
	}
	return
}

func sharpen(img *vips.ImageRef, _ imagor.LoadFunc, args ...string) (err error) {
	var sigma float64
	switch len(args) {
	case 1:
		sigma, _ = strconv.ParseFloat(args[0], 64)
		break
	case 2, 3:
		sigma, _ = strconv.ParseFloat(args[1], 64)
		break
	}
	sigma = 1 + sigma*2
	return img.Sharpen(sigma, 1, 2)
}

func stripIcc(img *vips.ImageRef, _ imagor.LoadFunc, _ ...string) (err error) {
	return img.RemoveICCProfile()
}

func stripExif(img *vips.ImageRef, _ imagor.LoadFunc, _ ...string) (err error) {
	return img.RemoveICCProfile()
}

func applyBackgroundColor(img *vips.ImageRef, c *vips.Color) (err error) {
	// set background color if specified
	var rect *vips.ImageRef
	if rect, err = vips.Black(img.Width(), img.Height()); err != nil {
		return
	}
	defer rect.Close()
	if err = rect.Linear([]float64{1, 1, 1}, []float64{
		float64(c.R), float64(c.G), float64(c.B),
	}); err != nil {
		return
	}
	if err = img.Composite(rect, vips.BlendModeDestOver, 0, 0); err != nil {
		return
	}
	return
}

func getColor(name string) *vips.Color {
	vc := &vips.Color{}
	strings.TrimPrefix(strings.ToLower(name), "#")
	if c, ok := colornames.Map[strings.ToLower(name)]; ok {
		vc.R = c.R
		vc.G = c.G
		vc.B = c.B
	} else if c, ok := parseHexColor(name); ok {
		vc.R = c.R
		vc.G = c.G
		vc.B = c.B
	}
	return vc
}

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

func hexToByte(b byte) byte {
	switch {
	case b >= '0' && b <= '9':
		return b - '0'
	case b >= 'a' && b <= 'f':
		return b - 'a' + 10
	case b >= 'A' && b <= 'F':
		return b - 'A' + 10
	}
	return 0
}
