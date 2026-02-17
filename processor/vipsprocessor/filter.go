package vipsprocessor

import (
	"context"
	"encoding/base64"
	"fmt"
	"math"
	"net/url"
	"strconv"
	"strings"

	"github.com/cshum/vipsgen/vips"

	"github.com/cshum/imagor"
	"github.com/cshum/imagor/imagorpath"
)

func (v *Processor) image(ctx context.Context, img *vips.Image, load imagor.LoadFunc, args ...string) (err error) {
	ln := len(args)
	if ln < 1 {
		return
	}
	imagorPath := args[0]
	if unescape, e := url.QueryUnescape(args[0]); e == nil {
		imagorPath = unescape
	}
	params := imagorpath.Parse(imagorPath)
	var blob *imagor.Blob
	if blob, err = load(params.Image); err != nil {
		return
	}
	var overlay *vips.Image
	// create fresh context for this processing level
	// while preserving parent resource tracking context
	ctx = withContext(ctx)
	if overlay, err = v.loadAndProcess(ctx, blob, params, load); err != nil || overlay == nil {
		return
	}
	contextDefer(ctx, overlay.Close)

	var xArg, yArg string
	var alpha float64
	var blendMode = vips.BlendModeOver // default to normal

	if ln >= 2 {
		xArg = args[1]
	}
	if ln >= 3 {
		yArg = args[2]
	}
	if ln >= 4 {
		alpha, _ = strconv.ParseFloat(args[3], 64)
	}
	if ln >= 5 {
		// Parse blend mode (5th parameter)
		blendMode = getBlendMode(args[4])
	}

	// Transform and composite overlay onto image
	return compositeOverlay(img, overlay, xArg, yArg, alpha, blendMode)
}

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
	var w, h int
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
	contextDefer(ctx, overlay.Close)

	// Parse arguments
	var xArg, yArg string
	var alpha float64
	if ln >= 3 {
		xArg = args[1]
		yArg = args[2]
	}
	if ln >= 4 {
		alpha, _ = strconv.ParseFloat(args[3], 64)
	}

	// Transform and composite overlay onto image
	return compositeOverlay(img, overlay, xArg, yArg, alpha, vips.BlendModeOver)
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
		// Check for alignment keyword with negative offset (e.g., left-20, l-20, right-30, r-30)
		if strings.HasPrefix(args[1], "left-") || strings.HasPrefix(args[1], "l-") {
			offset, _ := strconv.Atoi(strings.TrimPrefix(strings.TrimPrefix(args[1], "left-"), "l-"))
			x = -offset
		} else if strings.HasPrefix(args[1], "right-") || strings.HasPrefix(args[1], "r-") {
			offset, _ := strconv.Atoi(strings.TrimPrefix(strings.TrimPrefix(args[1], "right-"), "r-"))
			align = vips.AlignHigh
			x = width + offset
		} else if args[1] == "center" {
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
		// Apply negative adjustment for plain numeric values only (not prefixed keywords)
		if x < 0 &&
			!strings.HasPrefix(args[1], "left-") && !strings.HasPrefix(args[1], "l-") &&
			!strings.HasPrefix(args[1], "right-") && !strings.HasPrefix(args[1], "r-") {
			align = vips.AlignHigh
			x += width
		}
	}
	if ln > 2 {
		// Check for alignment keyword with negative offset (e.g., top-20, t-20, bottom-20, b-20)
		if strings.HasPrefix(args[2], "top-") || strings.HasPrefix(args[2], "t-") {
			offset, _ := strconv.Atoi(strings.TrimPrefix(strings.TrimPrefix(args[2], "top-"), "t-"))
			y = -offset
		} else if strings.HasPrefix(args[2], "bottom-") || strings.HasPrefix(args[2], "b-") {
			offset, _ := strconv.Atoi(strings.TrimPrefix(strings.TrimPrefix(args[2], "bottom-"), "b-"))
			y = img.PageHeight() - size + offset
		} else if args[2] == "center" {
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
		// Apply negative adjustment for plain numeric values only (not prefixed keywords)
		if y < 0 &&
			!strings.HasPrefix(args[2], "top-") && !strings.HasPrefix(args[2], "t-") &&
			!strings.HasPrefix(args[2], "bottom-") && !strings.HasPrefix(args[2], "b-") {
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
