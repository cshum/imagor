package vipsprocessor

import (
	"context"
	"encoding/base64"
	"fmt"
	"image/color"
	"math"
	"net/url"
	"strconv"
	"strings"

	"github.com/cshum/vipsgen/vips"

	"github.com/cshum/imagor"
	"github.com/cshum/imagor/imagorpath"
	"golang.org/x/image/colornames"
)

func (v *Processor) watermark(ctx context.Context, img *vips.Image, load imagor.LoadFunc, args ...string) (err error) {
	ln := len(args)
	if ln < 1 {
		return
	}
	image := args[0]

	if unescape, e := url.QueryUnescape(args[0]); e == nil {
		image = unescape
	}

	if strings.HasPrefix(image, "b64:") {
		// if image URL starts with b64: prefix, Base64 decode it according to "base64url" in RFC 4648 (Section 5).
		result := make([]byte, base64.RawURLEncoding.DecodedLen(len(image[4:])))
		// in case decoding fails, use original image URL (possible that filename starts with b64: prefix, but as part of the file name)
		if _, e := base64.RawURLEncoding.Decode(result, []byte(image[4:])); e == nil {
			image = string(result)
		}
	}

	var blob *imagor.Blob
	if blob, err = load(image); err != nil {
		return
	}
	var x, y, w, h int
	var across = 1
	var down = 1
	var overlay *vips.Image
	var n = 1
	if isAnimated(img) {
		n = -1
	}
	// w_ratio h_ratio
	if ln >= 6 {
		w = img.Width()
		h = img.PageHeight()
		if args[4] != "none" {
			w, _ = strconv.Atoi(args[4])
			w = img.Width() * w / 100
		}
		if args[5] != "none" {
			h, _ = strconv.Atoi(args[5])
			h = img.PageHeight() * h / 100
		}
		if overlay, err = v.NewThumbnail(
			ctx, blob, w, h, vips.InterestingNone, vips.SizeBoth, n, 1, 0,
		); err != nil {
			return
		}
	} else {
		if overlay, err = v.NewThumbnail(
			ctx, blob, v.MaxWidth, v.MaxHeight, vips.InterestingNone, vips.SizeDown, n, 1, 0,
		); err != nil {
			return
		}
	}
	var overlayN = overlay.Height() / overlay.PageHeight()
	contextDefer(ctx, overlay.Close)
	if overlay.Bands() < 3 {
		if err = overlay.Colourspace(vips.InterpretationSrgb, nil); err != nil {
			return
		}
	}
	if !overlay.HasAlpha() {
		if err = overlay.Addalpha(); err != nil {
			return
		}
	}
	w = overlay.Width()
	h = overlay.PageHeight()
	// alpha
	if ln >= 4 {
		alpha, _ := strconv.ParseFloat(args[3], 64)
		alpha = 1 - alpha/100
		if alpha != 1 {
			if err = overlay.Linear([]float64{1, 1, 1, alpha}, []float64{0, 0, 0, 0}, nil); err != nil {
				return
			}
		}
	}
	// x y
	if ln >= 3 {
		if args[1] == "center" {
			x = (img.Width() - overlay.Width()) / 2
		} else if args[1] == imagorpath.HAlignLeft {
			x = 0
		} else if args[1] == imagorpath.HAlignRight {
			x = img.Width() - overlay.Width()
		} else if args[1] == "repeat" {
			x = 0
			across = img.Width()/overlay.Width() + 1
		} else if strings.HasPrefix(strings.TrimPrefix(args[1], "-"), "0.") {
			pec, _ := strconv.ParseFloat(args[1], 64)
			x = int(pec * float64(img.Width()))
		} else if strings.HasSuffix(args[1], "p") {
			x, _ = strconv.Atoi(strings.TrimSuffix(args[1], "p"))
			x = x * img.Width() / 100
		} else {
			x, _ = strconv.Atoi(args[1])
		}
		if args[2] == "center" {
			y = (img.PageHeight() - overlay.PageHeight()) / 2
		} else if args[2] == imagorpath.VAlignTop {
			y = 0
		} else if args[2] == imagorpath.VAlignBottom {
			y = img.PageHeight() - overlay.PageHeight()
		} else if args[2] == "repeat" {
			y = 0
			down = img.PageHeight()/overlay.PageHeight() + 1
		} else if strings.HasPrefix(strings.TrimPrefix(args[2], "-"), "0.") {
			pec, _ := strconv.ParseFloat(args[2], 64)
			y = int(pec * float64(img.PageHeight()))
		} else if strings.HasSuffix(args[2], "p") {
			y, _ = strconv.Atoi(strings.TrimSuffix(args[2], "p"))
			y = y * img.PageHeight() / 100
		} else {
			y, _ = strconv.Atoi(args[2])
		}
		if x < 0 {
			x += img.Width() - overlay.Width()
		}
		if y < 0 {
			y += img.PageHeight() - overlay.PageHeight()
		}
	}
	if across*down > 1 {
		if err = overlay.EmbedMultiPage(0, 0, across*w, down*h,
			&vips.EmbedMultiPageOptions{Extend: vips.ExtendRepeat}); err != nil {
			return
		}
	}
	if err = overlay.EmbedMultiPage(
		x, y, img.Width(), img.PageHeight(), nil,
	); err != nil {
		return
	}
	if n := img.Height() / img.PageHeight(); n > overlayN {
		cnt := n / overlayN
		if n%overlayN > 0 {
			cnt++
		}
		if err = overlay.Replicate(1, cnt); err != nil {
			return
		}
	}
	if err = img.Composite2(overlay, vips.BlendModeOver, nil); err != nil {
		return
	}
	return
}

func (v *Processor) fill(ctx context.Context, img *vips.Image, w, h int, pLeft, pTop, pRight, pBottom int, colour string) (err error) {
	if isRotate90(ctx) {
		tmpW := w
		w = h
		h = tmpW
		tmpPLeft := pLeft
		pLeft = pTop
		pTop = tmpPLeft
		tmpPRight := pRight
		pRight = pBottom
		pBottom = tmpPRight
	}
	c := getColor(img, colour)
	left := (w-img.Width())/2 + pLeft
	top := (h-img.PageHeight())/2 + pTop
	width := w + pLeft + pRight
	height := h + pTop + pBottom
	if colour != "blur" || v.DisableBlur || isAnimated(img) {
		// fill color
		isTransparent := colour == "none" || colour == "transparent"
		if img.HasAlpha() && !isTransparent {
			c := getColor(img, colour)
			if err = img.Flatten(&vips.FlattenOptions{Background: c}); err != nil {
				return
			}
		}
		if isTransparent {
			if img.Bands() < 3 {
				if err = img.Colourspace(vips.InterpretationSrgb, nil); err != nil {
					return
				}
			}
			if !img.HasAlpha() {
				if err = img.Addalpha(); err != nil {
					return
				}
			}
			if err = img.EmbedMultiPage(left, top, width, height, &vips.EmbedMultiPageOptions{Extend: vips.ExtendBlack}); err != nil {
				return
			}
		} else if isBlack(c) {
			if err = img.EmbedMultiPage(left, top, width, height, &vips.EmbedMultiPageOptions{Extend: vips.ExtendBlack}); err != nil {
				return
			}
		} else if isWhite(c) {
			if err = img.EmbedMultiPage(left, top, width, height, &vips.EmbedMultiPageOptions{Extend: vips.ExtendWhite}); err != nil {
				return
			}
		} else {
			if err = img.EmbedMultiPage(left, top, width, height, &vips.EmbedMultiPageOptions{
				Extend:     vips.ExtendBackground,
				Background: c,
			}); err != nil {
				return
			}
		}
	} else {
		// fill blur
		var cp *vips.Image
		if cp, err = img.Copy(nil); err != nil {
			return
		}
		contextDefer(ctx, cp.Close)
		if err = img.ThumbnailImage(
			width, &vips.ThumbnailImageOptions{
				Height: height,
				Crop:   vips.InterestingNone,
				Size:   vips.SizeForce,
			},
		); err != nil {
			return
		}
		if err = img.Gaussblur(50, nil); err != nil {
			return
		}
		if err = img.Composite2(
			cp, vips.BlendModeOver,
			&vips.Composite2Options{X: left, Y: top}); err != nil {
			return
		}
	}
	return
}

func roundCorner(ctx context.Context, img *vips.Image, _ imagor.LoadFunc, args ...string) (err error) {
	var rx, ry int
	var c []float64
	if len(args) == 0 {
		return
	}
	if a, e := url.QueryUnescape(args[0]); e == nil {
		args[0] = a
	}
	if len(args) == 3 {
		// rx,ry,color
		c = getColor(img, args[2])
		args = args[:2]
	}
	rx, _ = strconv.Atoi(args[0])
	ry = rx
	if len(args) > 1 {
		ry, _ = strconv.Atoi(args[1])
	}

	var rounded *vips.Image
	var w = img.Width()
	var h = img.PageHeight()
	if rounded, err = vips.NewSvgloadBuffer([]byte(fmt.Sprintf(`
		<svg viewBox="0 0 %d %d">
			<rect rx="%d" ry="%d" 
			 x="0" y="0" width="%d" height="%d" 
			 fill="#fff"/>
		</svg>
	`, w, h, rx, ry, w, h)), nil); err != nil {
		return
	}
	contextDefer(ctx, rounded.Close)
	if n := img.Height() / img.PageHeight(); n > 1 {
		if err = rounded.Replicate(1, n); err != nil {
			return
		}
	}
	if err = img.Composite2(rounded, vips.BlendModeDestIn, nil); err != nil {
		return
	}
	if c != nil {
		if err = img.Flatten(&vips.FlattenOptions{Background: c}); err != nil {
			return
		}
	}
	return nil
}

func label(_ context.Context, img *vips.Image, _ imagor.LoadFunc, args ...string) (err error) {
	ln := len(args)
	if ln == 0 {
		return
	}
	if a, e := url.QueryUnescape(args[0]); e == nil {
		args[0] = a
	}
	var text = args[0]
	var font = "tahoma"
	var x, y int
	var c []float64
	var alpha float64
	var align = vips.AlignLow
	var size = 20
	var width = img.Width()
	if ln > 3 {
		size, _ = strconv.Atoi(args[3])
	}
	if ln > 1 {
		if args[1] == "center" {
			align = vips.AlignCentre
			x = width / 2
		} else if args[1] == imagorpath.HAlignRight {
			align = vips.AlignHigh
			x = width
		} else if strings.HasPrefix(strings.TrimPrefix(args[1], "-"), "0.") {
			pec, _ := strconv.ParseFloat(args[1], 64)
			x = int(pec * float64(width))
		} else if strings.HasSuffix(args[1], "p") {
			x, _ = strconv.Atoi(strings.TrimSuffix(args[1], "p"))
			x = x * width / 100
		} else {
			x, _ = strconv.Atoi(args[1])
		}
		if x < 0 {
			align = vips.AlignHigh
			x += width
		}
	}
	if ln > 2 {
		if args[2] == "center" {
			y = (img.PageHeight() - size) / 2
		} else if args[2] == imagorpath.VAlignTop {
			y = 0
		} else if args[2] == imagorpath.VAlignBottom {
			y = img.PageHeight() - size
		} else if strings.HasPrefix(strings.TrimPrefix(args[2], "-"), "0.") {
			pec, _ := strconv.ParseFloat(args[2], 64)
			y = int(pec * float64(img.PageHeight()))
		} else if strings.HasSuffix(args[2], "p") {
			y, _ = strconv.Atoi(strings.TrimSuffix(args[2], "p"))
			y = y * img.PageHeight() / 100
		} else {
			y, _ = strconv.Atoi(args[2])
		}
		if y < 0 {
			y += img.PageHeight() - size
		}
	}
	if ln > 4 {
		c = getColor(img, args[4])
	}
	if ln > 5 {
		alpha, _ = strconv.ParseFloat(args[5], 64)
		alpha /= 100
	}
	if ln > 6 {
		if a, e := url.QueryUnescape(args[6]); e == nil {
			font = a
		} else {
			font = args[6]
		}
	}
	if img.Bands() < 3 {
		if err = img.Colourspace(vips.InterpretationSrgb, nil); err != nil {
			return
		}
	}
	if !img.HasAlpha() {
		if err = img.Addalpha(); err != nil {
			return
		}
	}
	return img.Label(text, x, y, &vips.LabelOptions{
		Font:    font,
		Size:    size,
		Align:   align,
		Opacity: 1 - alpha,
		Color:   c,
	})
}

func (v *Processor) padding(ctx context.Context, img *vips.Image, _ imagor.LoadFunc, args ...string) error {
	ln := len(args)
	if ln < 2 {
		return nil
	}
	var (
		c       = args[0]
		left, _ = strconv.Atoi(args[1])
		top     = left
		right   = left
		bottom  = left
	)
	if ln > 2 {
		top, _ = strconv.Atoi(args[2])
		bottom = top
	}
	if ln > 4 {
		right, _ = strconv.Atoi(args[3])
		bottom, _ = strconv.Atoi(args[4])
	}
	return v.fill(ctx, img, img.Width(), img.PageHeight(), left, top, right, bottom, c)
}

func backgroundColor(_ context.Context, img *vips.Image, _ imagor.LoadFunc, args ...string) (err error) {
	if len(args) == 0 {
		return
	}
	if !img.HasAlpha() {
		return
	}
	c := getColor(img, args[0])
	return img.Flatten(&vips.FlattenOptions{
		Background: c,
	})
}

func rotate(ctx context.Context, img *vips.Image, _ imagor.LoadFunc, args ...string) (err error) {
	if len(args) == 0 {
		return
	}
	if angle, _ := strconv.Atoi(args[0]); angle > 0 {
		switch angle {
		case 90, 270:
			setRotate90(ctx)
		}
		if err = img.RotMultiPage(getAngle(angle)); err != nil {
			return err
		}
	}
	return
}

func getAngle(angle int) vips.Angle {
	switch angle {
	case 90:
		return vips.AngleD270
	case 180:
		return vips.AngleD180
	case 270:
		return vips.AngleD90
	default:
		return vips.AngleD0
	}
}

func proportion(_ context.Context, img *vips.Image, _ imagor.LoadFunc, args ...string) (err error) {
	if len(args) == 0 {
		return
	}
	scale, _ := strconv.ParseFloat(args[0], 64)
	if scale <= 0 {
		return // no ops
	}
	if scale > 100 {
		scale = 100
	}
	if scale > 1 {
		scale /= 100
	}
	width := int(float64(img.Width()) * scale)
	height := int(float64(img.PageHeight()) * scale)
	if width <= 0 || height <= 0 {
		return // op ops
	}
	return img.ThumbnailImage(width, &vips.ThumbnailImageOptions{
		Height: height,
		Crop:   vips.InterestingNone,
	})
}

func grayscale(_ context.Context, img *vips.Image, _ imagor.LoadFunc, _ ...string) (err error) {
	return img.Colourspace(vips.InterpretationBW, nil)
}

func brightness(_ context.Context, img *vips.Image, _ imagor.LoadFunc, args ...string) (err error) {
	if len(args) == 0 {
		return
	}
	b, _ := strconv.ParseFloat(args[0], 64)
	b = b * 255 / 100
	return linearRGB(img, []float64{1, 1, 1}, []float64{b, b, b})
}

func contrast(_ context.Context, img *vips.Image, _ imagor.LoadFunc, args ...string) (err error) {
	if len(args) == 0 {
		return
	}
	a, _ := strconv.ParseFloat(args[0], 64)
	a = a * 255 / 100
	a = math.Min(math.Max(a, -255), 255)
	a = (259 * (a + 255)) / (255 * (259 - a))
	b := 128 - a*128
	return linearRGB(img, []float64{a, a, a}, []float64{b, b, b})
}

func hue(_ context.Context, img *vips.Image, _ imagor.LoadFunc, args ...string) (err error) {
	if len(args) == 0 {
		return
	}
	h, _ := strconv.ParseFloat(args[0], 64)
	return img.Modulate(1, 1, h)
}

func saturation(_ context.Context, img *vips.Image, _ imagor.LoadFunc, args ...string) (err error) {
	if len(args) == 0 {
		return
	}
	s, _ := strconv.ParseFloat(args[0], 64)
	s = 1 + s/100
	return img.Modulate(1, s, 0)
}

func rgb(_ context.Context, img *vips.Image, _ imagor.LoadFunc, args ...string) (err error) {
	if len(args) != 3 {
		return
	}
	r, _ := strconv.ParseFloat(args[0], 64)
	g, _ := strconv.ParseFloat(args[1], 64)
	b, _ := strconv.ParseFloat(args[2], 64)
	r = r * 255 / 100
	g = g * 255 / 100
	b = b * 255 / 100
	return linearRGB(img, []float64{1, 1, 1}, []float64{r, g, b})
}

func modulate(_ context.Context, img *vips.Image, _ imagor.LoadFunc, args ...string) (err error) {
	if len(args) != 3 {
		return
	}
	b, _ := strconv.ParseFloat(args[0], 64)
	s, _ := strconv.ParseFloat(args[1], 64)
	h, _ := strconv.ParseFloat(args[2], 64)
	b = 1 + b/100
	s = 1 + s/100
	return img.Modulate(b, s, h)
}

func blur(ctx context.Context, img *vips.Image, _ imagor.LoadFunc, args ...string) (err error) {
	if isAnimated(img) {
		// skip animation support
		return
	}
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
		return img.Gaussblur(sigma, nil)
	}
	return
}

func sharpen(ctx context.Context, img *vips.Image, _ imagor.LoadFunc, args ...string) (err error) {
	if isAnimated(img) {
		// skip animation support
		return
	}
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
	if sigma > 0 {
		return img.Sharpen(&vips.SharpenOptions{
			Sigma: sigma,
			X1:    1,
			M2:    2,
		})
	}
	return
}

func stripIcc(_ context.Context, img *vips.Image, _ imagor.LoadFunc, _ ...string) (err error) {
	if img.HasICCProfile() {
		opts := vips.DefaultIccTransformOptions()
		opts.Embedded = true
		opts.Intent = vips.IntentPerceptual
		if img.Interpretation() == vips.InterpretationRgb16 {
			opts.Depth = 16
		}
		_ = img.IccTransform("srgb", opts)
	}
	return img.RemoveICCProfile()
}

func toColorspace(_ context.Context, img *vips.Image, _ imagor.LoadFunc, args ...string) (err error) {
	profile := "srgb"
	if len(args) > 0 && args[0] != "" {
		profile = strings.ToLower(args[0])
	}
	if !img.HasICCProfile() {
		return nil
	}
	opts := vips.DefaultIccTransformOptions()
	opts.Embedded = true
	opts.Intent = vips.IntentPerceptual
	if img.Interpretation() == vips.InterpretationRgb16 {
		opts.Depth = 16
	}
	return img.IccTransform(profile, opts)
}

func stripExif(_ context.Context, img *vips.Image, _ imagor.LoadFunc, _ ...string) (err error) {
	return img.RemoveExif()
}

func trim(ctx context.Context, img *vips.Image, _ imagor.LoadFunc, args ...string) error {
	var (
		ln        = len(args)
		pos       string
		tolerance int
	)
	if ln > 0 {
		tolerance, _ = strconv.Atoi(args[0])
	}
	if ln > 1 {
		pos = args[1]
	}
	if l, t, w, h, err := findTrim(ctx, img, pos, tolerance); err == nil {
		return img.ExtractAreaMultiPage(l, t, w, h)
	}
	return nil
}

func linearRGB(img *vips.Image, a, b []float64) error {
	if img.HasAlpha() {
		a = append(a, 1)
		b = append(b, 0)
	}
	return img.Linear(a, b, nil)
}

func isBlack(c []float64) bool {
	if len(c) < 3 {
		return false
	}
	return c[0] == 0x00 && c[1] == 0x00 && c[2] == 0x00
}

func isWhite(c []float64) bool {
	if len(c) < 3 {
		return false
	}
	return c[0] == 0xff && c[1] == 0xff && c[2] == 0xff
}

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
	}
	return 0
}

func crop(_ context.Context, img *vips.Image, _ imagor.LoadFunc, args ...string) error {
	if len(args) < 4 {
		return nil
	}

	// Parse arguments
	left, _ := strconv.ParseFloat(args[0], 64)
	top, _ := strconv.ParseFloat(args[1], 64)
	width, _ := strconv.ParseFloat(args[2], 64)
	height, _ := strconv.ParseFloat(args[3], 64)

	imgWidth := float64(img.Width())
	imgHeight := float64(img.PageHeight())

	// Convert relative (0-1) to absolute pixels
	if left > 0 && left < 1 {
		left = left * imgWidth
	}
	if top > 0 && top < 1 {
		top = top * imgHeight
	}
	if width > 0 && width < 1 {
		width = width * imgWidth
	}
	if height > 0 && height < 1 {
		height = height * imgHeight
	}

	// Clamp left and top to image bounds
	left = math.Max(0, math.Min(left, imgWidth))
	top = math.Max(0, math.Min(top, imgHeight))

	// Adjust width and height to not exceed image bounds
	width = math.Min(width, imgWidth-left)
	height = math.Min(height, imgHeight-top)

	// Skip if invalid crop area
	if width <= 0 || height <= 0 {
		return nil
	}

	return img.ExtractAreaMultiPage(int(left), int(top), int(width), int(height))
}

func isAnimated(img *vips.Image) bool {
	return img.Height() > img.PageHeight()
}
