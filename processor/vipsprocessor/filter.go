package vipsprocessor

import (
	"context"
	"fmt"
	"github.com/cshum/imagor"
	"github.com/cshum/imagor/imagorpath"
	"github.com/davidbyttow/govips/v2/vips"
	"math"
	"net/url"
	"strconv"
	"strings"
)

func (v *VipsProcessor) watermark(ctx context.Context, img *vips.ImageRef, load imagor.LoadFunc, args ...string) (err error) {
	ln := len(args)
	if ln < 1 {
		return
	}
	image := args[0]
	if unescape, e := url.QueryUnescape(args[0]); e == nil {
		image = unescape
	}
	var blob *imagor.Blob
	if blob, err = load(image); err != nil {
		return
	}
	var x, y, w, h int
	var across = 1
	var down = 1
	var overlay *vips.ImageRef
	var n = 1
	if isAnimated(ctx) {
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
		if overlay, err = v.newThumbnail(
			blob, w, h, vips.InterestingNone, vips.SizeDown, n,
		); err != nil {
			return
		}
	} else {
		if overlay, err = v.newThumbnail(
			blob, v.MaxWidth, v.MaxHeight, vips.InterestingNone, vips.SizeDown, n,
		); err != nil {
			return
		}
	}
	var overlayN = overlay.Height() / overlay.PageHeight()
	imagor.Defer(ctx, overlay.Close)
	if err = overlay.AddAlpha(); err != nil {
		return
	}
	w = overlay.Width()
	h = overlay.PageHeight()
	// alpha
	if ln >= 4 {
		alpha, _ := strconv.ParseFloat(args[3], 64)
		alpha = 1 - alpha/100
		if alpha != 1 {
			if err = overlay.Linear([]float64{1, 1, 1, alpha}, []float64{0, 0, 0, 0}); err != nil {
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
		if err = overlay.Embed(0, 0, across*w, down*h, vips.ExtendRepeat); err != nil {
			return
		}
	}
	if err = overlay.EmbedBackgroundRGBA(
		x, y, img.Width(), img.PageHeight(), &vips.ColorRGBA{},
	); err != nil {
		return
	}
	if n := getPageN(ctx); n > overlayN {
		cnt := n / overlayN
		if n%overlayN > 0 {
			cnt += 1
		}
		if err = overlay.Replicate(1, cnt); err != nil {
			return
		}
	}
	if err = img.Composite(overlay, vips.BlendModeOver, 0, 0); err != nil {
		return
	}
	return
}

func frames(ctx context.Context, img *vips.ImageRef, _ imagor.LoadFunc, args ...string) (err error) {
	ln := len(args)
	if ln == 0 {
		return
	}
	newN, _ := strconv.Atoi(args[0])
	if newN < 1 {
		return
	}
	if n := getPageN(ctx); n != newN {
		height := img.PageHeight()
		if err = img.SetPageHeight(img.Height()); err != nil {
			return
		}
		if err = img.Embed(0, 0, img.Width(), height*newN, vips.ExtendRepeat); err != nil {
			return
		}
		if err = img.SetPageHeight(height); err != nil {
			return
		}
		setPageN(ctx, newN)
	}
	var delay int
	if ln > 1 {
		delay, _ = strconv.Atoi(args[1])
	}
	if delay == 0 {
		delay = 100
	}
	delays := make([]int, newN)
	for i := 0; i < newN; i++ {
		delays[i] = delay
	}
	if err = img.SetPageDelay(delays); err != nil {
		return
	}
	return
}

func (v *VipsProcessor) fill(ctx context.Context, img *vips.ImageRef, w, h int, pLeft, pTop, pRight, pBottom int, colour string) (err error) {
	c := getColor(img, colour)
	left := (w-img.Width())/2 + pLeft
	top := (h-img.PageHeight())/2 + pTop
	width := w + pLeft + pRight
	height := h + pTop + pBottom
	if colour != "blur" || (colour == "blur" && v.DisableBlur) || isAnimated(ctx) {
		// fill color
		if img.HasAlpha() {
			if err = img.Flatten(getColor(img, colour)); err != nil {
				return
			}
		}
		if isBlack(c) {
			if err = img.Embed(left, top, width, height, vips.ExtendBlack); err != nil {
				return
			}
		} else if isWhite(c) {
			if err = img.Embed(left, top, width, height, vips.ExtendWhite); err != nil {
				return
			}
		} else {
			if err = img.EmbedBackground(left, top, width, height, c); err != nil {
				return
			}
		}
	} else {
		// fill blur
		var cp *vips.ImageRef
		if cp, err = img.Copy(); err != nil {
			return
		}
		imagor.Defer(ctx, cp.Close)
		if err = img.ThumbnailWithSize(
			width, height, vips.InterestingNone, vips.SizeForce,
		); err != nil {
			return
		}
		if err = img.GaussianBlur(50); err != nil {
			return
		}
		if err = img.Composite(
			cp, vips.BlendModeOver, left, top); err != nil {
			return
		}
	}
	return
}

func roundCorner(ctx context.Context, img *vips.ImageRef, _ imagor.LoadFunc, args ...string) (err error) {
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
		c = getColor(img, args[2])
		args = args[:2]
	}
	rx, _ = strconv.Atoi(args[0])
	ry = rx
	if len(args) > 1 {
		ry, _ = strconv.Atoi(args[1])
	}

	var rounded *vips.ImageRef
	var w = img.Width()
	var h = img.PageHeight()
	if rounded, err = vips.NewThumbnailFromBuffer([]byte(fmt.Sprintf(`
		<svg viewBox="0 0 %d %d">
			<rect rx="%d" ry="%d" 
			 x="0" y="0" width="%d" height="%d" 
			 fill="#fff"/>
		</svg>
	`, w, h, rx, ry, w, h)), w, h, vips.InterestingNone); err != nil {
		return
	}
	imagor.Defer(ctx, rounded.Close)
	if n := getPageN(ctx); n > 1 {
		if err = rounded.Replicate(1, n); err != nil {
			return
		}
	}
	if err = img.Composite(rounded, vips.BlendModeDestIn, 0, 0); err != nil {
		return
	}
	if c != nil {
		if err = img.Flatten(c); err != nil {
			return
		}
	}
	return nil
}

func (v *VipsProcessor) padding(ctx context.Context, img *vips.ImageRef, _ imagor.LoadFunc, args ...string) error {
	ln := len(args)
	if ln < 2 {
		return nil
	}
	var (
		color   = args[0]
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
	return v.fill(ctx, img, img.Width(), img.PageHeight(), left, top, right, bottom, color)
}

func backgroundColor(_ context.Context, img *vips.ImageRef, _ imagor.LoadFunc, args ...string) (err error) {
	if len(args) == 0 {
		return
	}
	if !img.HasAlpha() {
		return
	}
	return img.Flatten(getColor(img, args[0]))
}

func rotate(_ context.Context, img *vips.ImageRef, _ imagor.LoadFunc, args ...string) (err error) {
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

func proportion(_ context.Context, img *vips.ImageRef, _ imagor.LoadFunc, args ...string) (err error) {
	if len(args) == 0 {
		return
	}
	scale, _ := strconv.ParseFloat(args[0], 64)
	if scale <= 0 {
		return // no ops
	}
	if scale > 1 {
		scale /= 100
	}
	if scale > 100 {
		scale = 100
	}
	width := int(float64(img.Width()) * scale)
	height := int(float64(img.PageHeight()) * scale)
	if width <= 0 || height <= 0 {
		return // op ops
	}
	return img.Thumbnail(width, height, vips.InterestingNone)
}

func grayscale(_ context.Context, img *vips.ImageRef, _ imagor.LoadFunc, _ ...string) (err error) {
	return img.Modulate(1, 0, 0)
}

func brightness(_ context.Context, img *vips.ImageRef, _ imagor.LoadFunc, args ...string) (err error) {
	if len(args) == 0 {
		return
	}
	b, _ := strconv.ParseFloat(args[0], 64)
	b = b * 255 / 100
	return linearRGB(img, []float64{1, 1, 1}, []float64{b, b, b})
}

func contrast(_ context.Context, img *vips.ImageRef, _ imagor.LoadFunc, args ...string) (err error) {
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

func hue(_ context.Context, img *vips.ImageRef, _ imagor.LoadFunc, args ...string) (err error) {
	if len(args) == 0 {
		return
	}
	h, _ := strconv.ParseFloat(args[0], 64)
	return img.Modulate(1, 1, h)
}

func saturation(_ context.Context, img *vips.ImageRef, _ imagor.LoadFunc, args ...string) (err error) {
	if len(args) == 0 {
		return
	}
	s, _ := strconv.ParseFloat(args[0], 64)
	s = 1 + s/100
	return img.Modulate(1, s, 0)
}

func rgb(_ context.Context, img *vips.ImageRef, _ imagor.LoadFunc, args ...string) (err error) {
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

func modulate(_ context.Context, img *vips.ImageRef, _ imagor.LoadFunc, args ...string) (err error) {
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

func blur(ctx context.Context, img *vips.ImageRef, _ imagor.LoadFunc, args ...string) (err error) {
	if isAnimated(ctx) {
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
		return img.GaussianBlur(sigma)
	}
	return
}

func sharpen(ctx context.Context, img *vips.ImageRef, _ imagor.LoadFunc, args ...string) (err error) {
	if isAnimated(ctx) {
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
	return img.Sharpen(sigma, 1, 2)
}

func stripIcc(_ context.Context, img *vips.ImageRef, _ imagor.LoadFunc, _ ...string) (err error) {
	return img.RemoveICCProfile()
}

func trim(ctx context.Context, img *vips.ImageRef, _ imagor.LoadFunc, args ...string) error {
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
		return img.ExtractArea(l, t, w, h)
	}
	return nil
}

func linearRGB(img *vips.ImageRef, a, b []float64) error {
	if img.HasAlpha() {
		a = append(a, 1)
		b = append(b, 0)
	}
	return img.Linear(a, b)
}
