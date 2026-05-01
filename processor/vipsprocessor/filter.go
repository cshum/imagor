package vipsprocessor

import (
	"context"
	"encoding/base64"
	"fmt"
	"image"
	"math"
	"net/url"
	"strconv"
	"strings"

	"github.com/bbrks/go-blurhash"
	"github.com/cshum/vipsgen/vips"
	"go.n16f.net/thumbhash"

	"github.com/cshum/imagor"
)

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
	if err = ensureCompositeSRGB(img); err != nil {
		return
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
		// explicit sigma provided — use directly
		sigma, _ = strconv.ParseFloat(args[1], 64)
	case 1:
		// only radius provided — convert to sigma
		sigma, _ = strconv.ParseFloat(args[0], 64)
		sigma /= 2
	}
	if sigma > 0 {
		return img.Gaussblur(sigma, nil)
	}
	return
}

// pixelateImage applies a pixelate effect to img in-place using integer-ratio
// operations with zero interpolation:
//   - Shrink: box-average downscale — each output pixel is the average of a
//     blockSize×blockSize input block (no interpolation kernel).
//   - Zoom: pixel replication upscale — each pixel is replicated exactly
//     blockSize times in both axes (pure nearest-neighbour, no blending).
//
// This produces perfectly sharp square blocks — the classic "lack of resolution"
// pixelate look — with no anti-aliasing at either step.
func pixelateImage(img *vips.Image, blockSize int) error {
	if blockSize <= 1 {
		return nil
	}
	// Shrink: integer box-average downscale (no interpolation)
	if err := img.Shrink(float64(blockSize), float64(blockSize), nil); err != nil {
		return err
	}
	// Zoom: integer pixel replication upscale (no interpolation)
	return img.Zoom(blockSize, blockSize)
}

func pixelate(_ context.Context, img *vips.Image, _ imagor.LoadFunc, args ...string) (err error) {
	if isAnimated(img) {
		return
	}
	blockSize := 10
	if len(args) > 0 {
		if b, e := strconv.Atoi(args[0]); e == nil && b > 0 {
			blockSize = b
		}
	}
	return pixelateImage(img, blockSize)
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
	normalizeSrgb(img)
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

// avgColorRGB computes the average RGB color of img by:
//  1. Flattening any alpha channel against a white background (255,255,255)
//     so transparent areas contribute white rather than black to the average.
//  2. Downscaling to at most 64px wide for fast stats.
//  3. Running vips Stats and reading the per-band mean values.
//
// The returned slice is [R, G, B] as float64 in the 0–255 range.
func avgColorRGB(img *vips.Image) ([]float64, error) {
	thumb, err := img.Copy(nil)
	if err != nil {
		return nil, err
	}
	defer thumb.Close()
	normalizeSrgb(thumb)
	if thumb.HasAlpha() {
		// Flatten against white so transparent pixels don't skew the average dark.
		if err := thumb.Flatten(&vips.FlattenOptions{Background: []float64{255, 255, 255}}); err != nil {
			return nil, err
		}
	}
	if err := thumb.ThumbnailImage(64, &vips.ThumbnailImageOptions{Size: vips.SizeDown}); err != nil {
		return nil, err
	}
	// Stats matrix. Column 4 = mean, bands are R,G,B.
	if err := thumb.Stats(); err != nil {
		return nil, err
	}
	rMean, err := thumb.Getpoint(4, 1, nil)
	if err != nil {
		return nil, err
	}
	gMean, err := thumb.Getpoint(4, 2, nil)
	if err != nil {
		return nil, err
	}
	bMean, err := thumb.Getpoint(4, 3, nil)
	if err != nil {
		return nil, err
	}
	return []float64{
		math.Round(rMean[0]),
		math.Round(gMean[0]),
		math.Round(bMean[0]),
	}, nil
}

// avgColor returns the average color of img as an AvgColor with RGB components.
func avgColor(_ context.Context, img *vips.Image) (*AvgColor, error) {
	rgb, err := avgColorRGB(img)
	if err != nil {
		return nil, err
	}
	return &AvgColor{
		R: uint8(rgb[0]),
		G: uint8(rgb[1]),
		B: uint8(rgb[2]),
	}, nil
}

// blurHash returns a Blurhash string for img using the given x and y components.
func blurHash(_ context.Context, img *vips.Image, xComponents, yComponents int) (string, error) {
	thumb, err := img.Copy(nil)
	if err != nil {
		return "", err
	}
	defer thumb.Close()
	normalizeSrgb(thumb)
	if thumb.HasAlpha() {
		if err := thumb.Flatten(vips.DefaultFlattenOptions()); err != nil {
			return "", err
		}
	}
	if err := thumb.ThumbnailImage(64, &vips.ThumbnailImageOptions{Size: vips.SizeDown}); err != nil {
		return "", err
	}
	w, h := thumb.Width(), thumb.Height()
	raw, err := thumb.WriteToMemory()
	if err != nil {
		return "", err
	}
	// raw is packed RGB (3 bytes/pixel), expand to RGBA with an alpha channel of 255
	if len(raw) != w*h*3 {
		return "", fmt.Errorf("unexpected raw pixel length %d (want %d)", len(raw), w*h*3)
	}
	nrgba := image.NewNRGBA(image.Rect(0, 0, w, h))
	for i, j := 0, 0; i < len(raw); i, j = i+3, j+4 {
		nrgba.Pix[j] = raw[i]
		nrgba.Pix[j+1] = raw[i+1]
		nrgba.Pix[j+2] = raw[i+2]
		nrgba.Pix[j+3] = 255
	}
	return blurhash.Encode(xComponents, yComponents, nrgba)
}

// thumbHash returns a base64-encoded ThumbHash string for img.
// Alpha is preserved when present.
func thumbHash(_ context.Context, img *vips.Image) (string, error) {
	thumb, err := img.Copy(nil)
	if err != nil {
		return "", err
	}
	defer thumb.Close()
	normalizeSrgb(thumb)
	if err := thumb.ThumbnailImage(100, &vips.ThumbnailImageOptions{Size: vips.SizeDown}); err != nil {
		return "", err
	}
	w, h := thumb.Width(), thumb.Height()
	raw, err := thumb.WriteToMemory()
	if err != nil {
		return "", err
	}
	nrgba := image.NewNRGBA(image.Rect(0, 0, w, h))
	if thumb.HasAlpha() {
		if len(raw) != w*h*4 {
			return "", fmt.Errorf("unexpected raw pixel length %d (want %d)", len(raw), w*h*4)
		}
		copy(nrgba.Pix, raw)
	} else {
		if len(raw) != w*h*3 {
			return "", fmt.Errorf("unexpected raw pixel length %d (want %d)", len(raw), w*h*3)
		}
		for i, j := 0, 0; i < len(raw); i, j = i+3, j+4 {
			nrgba.Pix[j] = raw[i]
			nrgba.Pix[j+1] = raw[i+1]
			nrgba.Pix[j+2] = raw[i+2]
			nrgba.Pix[j+3] = 255
		}
	}
	return base64.StdEncoding.EncodeToString(thumbhash.EncodeImage(nrgba)), nil
}
