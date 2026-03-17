package vipsprocessor

import (
	"context"
	"encoding/base64"
	"net/url"
	"strconv"
	"strings"

	"github.com/cshum/imagor"
	"github.com/cshum/imagor/imagorpath"
	"github.com/cshum/vipsgen/vips"
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
	// Resolve f-token dimension placeholders (e.g. fxf, f-20xf-20) against the
	// parent image's pixel dimensions before handing the path to the parser.
	imagorPath = resolveFullDimensions(imagorPath, img.Width(), img.PageHeight())
	params := imagorpath.Parse(imagorPath)
	var blob *imagor.Blob
	// Skip loader for color: image paths — they are generated in-process
	if _, isColor := parseColorImage(params.Image); !isColor {
		if blob, err = load(params.Image); err != nil {
			return
		}
	}
	var overlay *vips.Image
	// create fresh context for this processing level
	// while preserving parent resource tracking context
	ctx = withContext(ctx)
	if overlay, err = v.loadFilterImage(ctx, blob, params, load, params.Image); err != nil || overlay == nil {
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
		if overlay, err = v.loadOverlayImage(ctx, load, image, w, h, n, vips.SizeBoth); err != nil {
			return
		}
	} else {
		if overlay, err = v.loadOverlayImage(ctx, load, image, 0, 0, n, vips.SizeDown); err != nil {
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

func label(_ context.Context, img *vips.Image, _ imagor.LoadFunc, args ...string) (err error) {
	ln := len(args)
	if ln == 0 {
		return
	}
	text := decodeTextArg(args[0])
	if strings.TrimSpace(text) == "" {
		return
	}
	font := "sans"
	size := 20
	c := []float64{0, 0, 0}
	var alpha float64
	var xArg, yArg string
	if ln > 1 {
		xArg = args[1]
	}
	if ln > 2 {
		yArg = args[2]
	}
	if ln > 3 {
		size, _ = strconv.Atoi(args[3])
	}
	if ln > 4 {
		c = getColor(img, args[4])
	}
	if ln > 5 {
		alpha, _ = strconv.ParseFloat(args[5], 64)
	}
	if ln > 6 {
		font = parseFontArg(args[6])
	}
	// Render text as RGBA: white text on transparent background.
	// rgba:true makes libvips/Pango emit a proper 4-band sRGB image so
	// HasAlpha() is true with no extra fixups needed.
	textImg, err := vips.NewText(text, &vips.TextOptions{Font: font, Width: 9999, Height: size, Rgba: true})
	if err != nil {
		return
	}
	defer textImg.Close()
	// Colorize: replace RGB channels with the target color (multiplier=0 zeros out
	// the original black text color, offset=c sets the target color), while
	// preserving the alpha channel exactly (coverage stays as Pango rendered it).
	if err = textImg.Linear(
		[]float64{0, 0, 0, 1},
		[]float64{c[0], c[1], c[2], 0},
		nil,
	); err != nil {
		return
	}
	if err = textImg.Cast(vips.BandFormatUchar, nil); err != nil {
		return
	}
	if textImg.Height() < size {
		if err = textImg.Embed(0, 0, textImg.Width(), size, nil); err != nil {
			return
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
	return compositeOverlay(img, textImg, xArg, yArg, alpha, vips.BlendModeOver)
}

func text(_ context.Context, img *vips.Image, _ imagor.LoadFunc, args ...string) (err error) {
	// text(text,x,y,font,color,alpha,blend_mode,width,align,justify,wrap,spacing,dpi)
	// font includes size e.g. "sans bold 24", "monospace 18"
	ln := len(args)
	if ln == 0 {
		return
	}
	var (
		textStr = decodeTextArg(args[0])
		xArg    string
	)
	if strings.TrimSpace(textStr) == "" {
		return
	}
	var (
		yArg      string
		font      = "sans 20"
		c         = []float64{0, 0, 0}
		alpha     float64
		blendMode = vips.BlendModeOver
		width     int
		align     = vips.AlignLow
		justy     = false
		wrap      = vips.TextWrapWord
		spacing   int
		dpi       int
	)
	if ln > 1 {
		xArg = args[1]
	}
	if ln > 2 {
		yArg = args[2]
	}
	if ln > 3 {
		font = parseFontArg(args[3])
	}
	if ln > 4 {
		c = getColor(img, args[4])
	}
	if ln > 5 {
		alpha, _ = strconv.ParseFloat(args[5], 64)
	}
	if ln > 6 {
		blendMode = getBlendMode(args[6])
	}
	if ln > 7 {
		width = parseTextWidth(args[7], img.Width())
	}
	if ln > 8 {
		switch strings.ToLower(args[8]) {
		case "centre", "center":
			align = vips.AlignCentre
		case "high", "right":
			align = vips.AlignHigh
		default:
			align = vips.AlignLow
		}
	}
	if ln > 9 {
		justy = args[9] == "true" || args[9] == "1"
	}
	if ln > 10 {
		switch strings.ToLower(args[10]) {
		case "char":
			wrap = vips.TextWrapChar
		case "wordchar", "word_char":
			wrap = vips.TextWrapWordChar
		case "none":
			wrap = vips.TextWrapNone
		default:
			wrap = vips.TextWrapWord
		}
	}
	if ln > 11 {
		spacing, _ = strconv.Atoi(args[11])
	}
	if ln > 12 {
		dpi, _ = strconv.Atoi(args[12])
	}

	opts := &vips.TextOptions{
		Font:    font,
		Width:   width,
		Align:   align,
		Justify: justy,
		Wrap:    wrap,
		Rgba:    true,
	}
	if spacing > 0 {
		opts.Spacing = spacing
	}
	if dpi > 0 {
		opts.Dpi = dpi
	}

	textImg, err := vips.NewText(textStr, opts)
	if err != nil {
		return
	}
	defer textImg.Close()

	// Colorize: Pango renders black [0,0,0,A]; zero the RGB and offset to target color
	if err = textImg.Linear(
		[]float64{0, 0, 0, 1},
		[]float64{c[0], c[1], c[2], 0},
		nil,
	); err != nil {
		return
	}
	if err = textImg.Cast(vips.BandFormatUchar, nil); err != nil {
		return
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
	return compositeOverlay(img, textImg, xArg, yArg, alpha, blendMode)
}
