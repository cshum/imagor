package vipsprocessor

import (
	"math"
	"strings"

	"github.com/cshum/vipsgen/v2/vips"
)

// libvips updates a gain map only in thumbnail; these mirror the rest.

func gainmapOf(img *vips.Image) (*vips.Image, error) {
	// Only UltraHDR images carry a gain map.
	if img.Format() != vips.ImageTypeUhdr {
		return nil, nil
	}
	gainmap, ok := img.Gainmap()
	if !ok {
		return nil, nil
	}
	return gainmap, nil
}

// rotateGainmap rotates the gain map with the image.
func rotateGainmap(img *vips.Image, angle float64) error {
	gainmap, err := gainmapOf(img)
	if err != nil || gainmap == nil {
		return err
	}
	defer gainmap.Close()
	if err = gainmap.Rotate(angle, nil); err != nil {
		return err
	}
	img.SetGainmap(gainmap)
	return nil
}

// flipGainmap flips the gain map with the image.
func flipGainmap(img *vips.Image, direction vips.Direction) error {
	gainmap, err := gainmapOf(img)
	if err != nil || gainmap == nil {
		return err
	}
	defer gainmap.Close()
	if err = gainmap.Flip(direction); err != nil {
		return err
	}
	img.SetGainmap(gainmap)
	return nil
}

// scaledRect maps a rectangle in image pixels onto the gain map.
func scaledRect(img, gainmap *vips.Image, left, top, width, height int) (x, y, w, h int) {
	imgWidth, imgHeight := img.Width(), img.PageHeight()
	if imgWidth == 0 || imgHeight == 0 {
		return 0, 0, 0, 0
	}
	gmWidth, gmHeight := gainmap.Width(), gainmap.PageHeight()
	scaleX := float64(gmWidth) / float64(imgWidth)
	scaleY := float64(gmHeight) / float64(imgHeight)
	x = int(math.Round(float64(left) * scaleX))
	y = int(math.Round(float64(top) * scaleY))
	w = int(math.Round(float64(width) * scaleX))
	h = int(math.Round(float64(height) * scaleY))
	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}
	if x > gmWidth-1 {
		x = gmWidth - 1
	}
	if y > gmHeight-1 {
		y = gmHeight - 1
	}
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	if w > gmWidth-x {
		w = gmWidth - x
	}
	if h > gmHeight-y {
		h = gmHeight - y
	}
	return
}

// cropGainmap crops the gain map to the region the image keeps. Call it
// before the crop.
func cropGainmap(img *vips.Image, left, top, width, height int) error {
	gainmap, err := gainmapOf(img)
	if err != nil || gainmap == nil {
		return err
	}
	defer gainmap.Close()
	x, y, w, h := scaledRect(img, gainmap, left, top, width, height)
	if err = gainmap.ExtractArea(x, y, w, h); err != nil {
		return err
	}
	img.SetGainmap(gainmap)
	return nil
}

// scaledCanvas maps a target canvas and offset onto the gain map. Unlike a
// crop, the result can extend past the gain map, so it is not clamped.
func scaledCanvas(img, gainmap *vips.Image, left, top, width, height int) (x, y, w, h int) {
	imgWidth, imgHeight := img.Width(), img.PageHeight()
	if imgWidth == 0 || imgHeight == 0 {
		return 0, 0, 0, 0
	}
	scaleX := float64(gainmap.Width()) / float64(imgWidth)
	scaleY := float64(gainmap.PageHeight()) / float64(imgHeight)
	x = int(math.Round(float64(left) * scaleX))
	y = int(math.Round(float64(top) * scaleY))
	w = int(math.Round(float64(width) * scaleX))
	h = int(math.Round(float64(height) * scaleY))
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	return
}

// embedGainmap pads the gain map like the image, with no boost in the padding.
func embedGainmap(img *vips.Image, left, top, width, height int, opts *vips.EmbedOptions) error {
	gainmap, err := gainmapOf(img)
	if err != nil || gainmap == nil {
		return err
	}
	defer gainmap.Close()
	x, y, w, h := scaledCanvas(img, gainmap, left, top, width, height)
	if opts != nil && opts.Extend != vips.ExtendCopy && opts.Extend != vips.ExtendRepeat {
		opts = &vips.EmbedOptions{
			Extend:     vips.ExtendBackground,
			Background: neutralGain(img, gainmap.Bands()),
		}
	}
	if err = gainmap.Embed(x, y, w, h, opts); err != nil {
		return err
	}
	img.SetGainmap(gainmap)
	return nil
}

// attachGainmap attaches the gain map so thumbnail updates it on resize.
// Call it before the resize.
func attachGainmap(img *vips.Image) error {
	gainmap, err := gainmapOf(img)
	if err != nil || gainmap == nil {
		return err
	}
	defer gainmap.Close()
	img.SetGainmap(gainmap)
	return nil
}

// stripExifKeepingGainmap removes the identifying metadata fields, keeping the
// gain map fields.
func stripExifKeepingGainmap(img *vips.Image) error {
	for _, field := range img.GetFields() {
		if strings.HasPrefix(field, "exif") ||
			strings.HasPrefix(field, "iptc") ||
			strings.HasPrefix(field, "xmp") ||
			field == "xml" {
			img.RemoveField(field)
		}
	}
	return nil
}

// setGainmapColourSpace marks the gain map as sharing the colour space of the
// image. libultrahdr fails to save the gain maps of iPhones without it, and
// refuses images that are no longer RGB.
func setGainmapColourSpace(img *vips.Image) {
	if img.Format() != vips.ImageTypeUhdr {
		return
	}
	if img.Bands() == 1 || img.Interpretation() == vips.InterpretationCmyk {
		img.RemoveField("gainmap")
		img.RemoveField("gainmap-data")
		return
	}
	img.SetInt("gainmap-use-base-cg", 1)
}

// neutralGain returns the value of a gain of one in every band, for new edges.
func neutralGain(img *vips.Image, bands int) []float64 {
	values := make([]float64, bands)
	minBoost, _ := img.GetArrayDouble("gainmap-min-content-boost")
	maxBoost, _ := img.GetArrayDouble("gainmap-max-content-boost")
	gamma, _ := img.GetArrayDouble("gainmap-gamma")
	for i := range values {
		lo, hi := log2At(minBoost, i), log2At(maxBoost, i)
		g := 1.0
		if i < len(gamma) && gamma[i] > 0 {
			g = gamma[i]
		}
		if hi > lo {
			values[i] = 255 * math.Pow(math.Min(math.Max(-lo/(hi-lo), 0), 1), g)
		}
	}
	return values
}

func log2At(values []float64, i int) float64 {
	if i >= len(values) || values[i] <= 0 {
		return 0
	}
	return math.Log2(values[i])
}

// blurGainmap blurs the gain map like the image, so the highlights keep their edges.
func blurGainmap(img *vips.Image, sigma float64, width int) error {
	gainmap, err := gainmapOf(img)
	if err != nil || gainmap == nil {
		return err
	}
	defer gainmap.Close()
	sigma *= float64(gainmap.Width()) / float64(width)
	if sigma <= 0 {
		return nil
	}
	if err = gainmap.Gaussblur(sigma, nil); err != nil {
		return err
	}
	img.SetGainmap(gainmap)
	return nil
}

// gainmapMeta is the part of the gain map that an operation rebuilding the
// image drops: the image and the fields uhdrsave reads.
type gainmapMeta struct {
	img     *vips.Image
	data    []byte
	arrays  map[string][]float64
	doubles map[string]float64
}

var gainmapArrays = []string{
	"gainmap-max-content-boost", "gainmap-min-content-boost", "gainmap-gamma",
	"gainmap-offset-sdr", "gainmap-offset-hdr",
}

var gainmapDoubles = []string{"gainmap-hdr-capacity-min", "gainmap-hdr-capacity-max"}

// saveGainmapMeta captures the gain map before an operation rebuilds the image.
func saveGainmapMeta(img *vips.Image) *gainmapMeta {
	gainmap, err := gainmapOf(img)
	if err != nil || gainmap == nil {
		return nil
	}
	meta := &gainmapMeta{img: gainmap, arrays: map[string][]float64{}, doubles: map[string]float64{}}
	if data, err := img.GetBlob("gainmap-data"); err == nil {
		meta.data = data
	}
	for _, name := range gainmapArrays {
		if values, err := img.GetArrayDouble(name); err == nil {
			meta.arrays[name] = values
		}
	}
	for _, name := range gainmapDoubles {
		if value, err := img.GetDouble(name); err == nil {
			meta.doubles[name] = value
		}
	}
	return meta
}

// restore sets the saved gain map on the rebuilt image.
func (m *gainmapMeta) restore(img *vips.Image) error {
	if m == nil {
		return nil
	}
	defer m.img.Close()
	if len(m.data) > 0 {
		img.SetBlob("gainmap-data", m.data)
	}
	img.SetGainmap(m.img)
	for name, values := range m.arrays {
		if err := img.SetArrayDouble(name, values); err != nil {
			return err
		}
	}
	for name, value := range m.doubles {
		img.SetDouble(name, value)
	}
	return nil
}
