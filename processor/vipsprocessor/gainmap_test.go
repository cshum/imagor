package vipsprocessor

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/cshum/imagor"
	"github.com/cshum/imagor/imagorpath"
	"github.com/cshum/vipsgen/v2/vips"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func gainmapSize(t *testing.T, img *vips.Image) (int, int) {
	t.Helper()
	gainmap, ok := img.Gainmap()
	require.True(t, ok)
	defer gainmap.Close()
	return gainmap.Width(), gainmap.PageHeight()
}

func TestRotateGainmap(t *testing.T) {
	if !vips.HasOperation("uhdrload") {
		t.Skip("libvips is built without libultrahdr")
	}
	img, err := vips.NewImageFromFile(filepath.Join(testDataDir, "ultra-hdr.jpg"), nil)
	require.NoError(t, err)
	defer img.Close()

	width, height := gainmapSize(t, img)
	require.NoError(t, rotateGainmap(img, 90))
	rotatedWidth, rotatedHeight := gainmapSize(t, img)
	assert.Equal(t, height, rotatedWidth)
	assert.Equal(t, width, rotatedHeight)
}

func TestCropGainmap(t *testing.T) {
	if !vips.HasOperation("uhdrload") {
		t.Skip("libvips is built without libultrahdr")
	}
	img, err := vips.NewImageFromFile(filepath.Join(testDataDir, "ultra-hdr.jpg"), nil)
	require.NoError(t, err)
	defer img.Close()

	width, height := gainmapSize(t, img)
	cropWidth, cropHeight := img.Width()/2, img.PageHeight()/2
	require.NoError(t, cropGainmap(img, img.Width()/4, img.PageHeight()/4, cropWidth, cropHeight))
	gotWidth, gotHeight := gainmapSize(t, img)
	assert.InDelta(t, width/2, gotWidth, 2)
	assert.InDelta(t, height/2, gotHeight, 2)
}

func TestFlipGainmap(t *testing.T) {
	if !vips.HasOperation("uhdrload") {
		t.Skip("libvips is built without libultrahdr")
	}
	img, err := vips.NewImageFromFile(filepath.Join(testDataDir, "ultra-hdr.jpg"), nil)
	require.NoError(t, err)
	defer img.Close()

	width, height := gainmapSize(t, img)
	require.NoError(t, flipGainmap(img, vips.DirectionHorizontal))
	gotWidth, gotHeight := gainmapSize(t, img)
	assert.Equal(t, width, gotWidth)
	assert.Equal(t, height, gotHeight)
}

func TestAttachGainmap(t *testing.T) {
	if !vips.HasOperation("uhdrload") {
		t.Skip("libvips is built without libultrahdr")
	}
	img, err := vips.NewImageFromFile(filepath.Join(testDataDir, "ultra-hdr.jpg"), nil)
	require.NoError(t, err)
	defer img.Close()

	require.False(t, img.HasField("gainmap"))
	require.NoError(t, attachGainmap(img))
	assert.True(t, img.HasField("gainmap"))
}

func TestEmbedGainmap(t *testing.T) {
	if !vips.HasOperation("uhdrload") {
		t.Skip("libvips is built without libultrahdr")
	}
	img, err := vips.NewImageFromFile(filepath.Join(testDataDir, "ultra-hdr.jpg"), nil)
	require.NoError(t, err)
	defer img.Close()

	width, height := img.Width(), img.PageHeight()
	gainmapWidth, gainmapHeight := gainmapSize(t, img)
	require.NoError(t, embedGainmap(img, 10, 20, width+20, height+40,
		&vips.EmbedOptions{Extend: vips.ExtendBlack}))
	require.NoError(t, img.EmbedMultiPage(10, 20, width+20, height+40,
		&vips.EmbedMultiPageOptions{Extend: vips.ExtendBlack}))
	gotWidth, gotHeight := gainmapSize(t, img)
	assert.InDelta(t, img.Width()*gainmapWidth/width, gotWidth, 2)
	assert.InDelta(t, img.PageHeight()*gainmapHeight/height, gotHeight, 2)
}

func TestRotateGainmapArbitraryAngle(t *testing.T) {
	if !vips.HasOperation("uhdrload") {
		t.Skip("libvips is built without libultrahdr")
	}
	img, err := vips.NewImageFromFile(filepath.Join(testDataDir, "ultra-hdr.jpg"), nil)
	require.NoError(t, err)
	defer img.Close()

	require.NoError(t, rotateGainmap(img, 45))
	width, height := gainmapSize(t, img)
	assert.InDelta(t, width, height, 2)
}

func TestFlipGainmapVertical(t *testing.T) {
	if !vips.HasOperation("uhdrload") {
		t.Skip("libvips is built without libultrahdr")
	}
	img, err := vips.NewImageFromFile(filepath.Join(testDataDir, "ultra-hdr.jpg"), nil)
	require.NoError(t, err)
	defer img.Close()

	width, height := gainmapSize(t, img)
	require.NoError(t, flipGainmap(img, vips.DirectionVertical))
	gotWidth, gotHeight := gainmapSize(t, img)
	assert.Equal(t, width, gotWidth)
	assert.Equal(t, height, gotHeight)
}

func TestRotateGainmapWithoutGainmap(t *testing.T) {
	img, err := vips.NewImageFromFile(filepath.Join(testDataDir, "demo1.jpg"), nil)
	require.NoError(t, err)
	defer img.Close()

	assert.False(t, img.HasField("gainmap-data"))
	assert.NoError(t, rotateGainmap(img, 90))
	assert.False(t, img.HasField("gainmap-data"))
}

func TestAngleDegrees(t *testing.T) {
	tests := []struct {
		angle vips.Angle
		want  float64
	}{
		{vips.AngleD0, 0},
		{vips.AngleD90, 90},
		{vips.AngleD180, 180},
		{vips.AngleD270, 270},
		{vips.Angle(9), 0},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, angleDegrees(tt.angle))
	}
}

func TestEmbedGainmapUsesNeutralGain(t *testing.T) {
	img := loadUltraHDR(t)
	defer img.Close()

	require.NoError(t, embedGainmap(img, 10, 10, img.Width()+20, img.PageHeight()+20,
		&vips.EmbedOptions{Extend: vips.ExtendWhite}))
	require.NoError(t, img.EmbedMultiPage(10, 10, img.Width()+20, img.PageHeight()+20,
		&vips.EmbedMultiPageOptions{Extend: vips.ExtendWhite}))

	gainmap, ok := img.Gainmap()
	require.True(t, ok)
	defer gainmap.Close()
	corner, err := gainmap.Getpoint(0, 0, nil)
	require.NoError(t, err)
	assert.Less(t, corner[0], 10.0, "padding carries no boost, not the fill colour")
}

func TestCropGainmapTinyArea(t *testing.T) {
	img := loadUltraHDR(t)
	defer img.Close()

	width, height := img.Width(), img.PageHeight()
	require.NoError(t, cropGainmap(img, width-1, height-1, width, height))

	gainWidth, gainHeight := gainmapSize(t, img)
	assert.GreaterOrEqual(t, gainWidth, 1)
	assert.GreaterOrEqual(t, gainHeight, 1)
}

func TestGainmapIPhoneColourSpace(t *testing.T) {
	// A copy of its own: libvips caches a file by name, and an earlier load
	// in this package would hand back an image that already carries the flag.
	data, err := os.ReadFile(filepath.Join(testDataDir, "iphone-ultrahdr.jpg"))
	require.NoError(t, err)
	path := filepath.Join(t.TempDir(), "iphone.jpg")
	require.NoError(t, os.WriteFile(path, data, 0o644))

	img, err := vips.NewImageFromFile(path, nil)
	require.NoError(t, err)
	defer img.Close()

	// The gain map of an iPhone photo carries no ICC profile, which
	// libultrahdr refuses to encode without the colour space flag.
	setGainmapColourSpace(img)
	value, err := img.GetInt("gainmap-use-base-cg")
	require.NoError(t, err)
	require.Equal(t, 1, value)

	buf, err := img.JpegsaveBuffer(nil)
	require.NoError(t, err)

	out, err := vips.NewImageFromBuffer(buf, nil)
	require.NoError(t, err)
	defer out.Close()
	gainmap, ok := out.Gainmap()
	require.True(t, ok, "the saved image should still be Ultra HDR")
	gainmap.Close()
}

func TestSetGainmapColourSpaceDropsNonRGB(t *testing.T) {
	img := loadUltraHDR(t)
	defer img.Close()
	require.NoError(t, img.ExtractBand(0, nil))
	require.True(t, img.HasField("gainmap-data"))

	setGainmapColourSpace(img)
	assert.False(t, img.HasField("gainmap-data"))
	assert.False(t, img.HasField("gainmap"))
}

func TestBlurGainmap(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(testDataDir, "iphone-ultrahdr.jpg"))
	require.NoError(t, err)
	path := filepath.Join(t.TempDir(), "iphone.jpg")
	require.NoError(t, os.WriteFile(path, data, 0o644))
	img, err := vips.NewImageFromFile(path, nil)
	require.NoError(t, err)
	defer img.Close()

	gainmap, ok := img.Gainmap()
	require.True(t, ok)
	width, height := gainmap.Width(), gainmap.PageHeight()
	before, err := sumGainmap(gainmap)
	require.NoError(t, err)
	gainmap.Close()

	require.NoError(t, blurGainmap(img, 40, img.Width()))

	gainmap, ok = img.Gainmap()
	require.True(t, ok)
	defer gainmap.Close()
	after, err := sumGainmap(gainmap)
	require.NoError(t, err)
	assert.Equal(t, width, gainmap.Width())
	assert.Equal(t, height, gainmap.PageHeight())
	assert.NotEqual(t, before, after, "blurring changes the gain map")
}

func TestNeutralGainWithoutMetadata(t *testing.T) {
	img, err := vips.NewImageFromFile(filepath.Join(testDataDir, "demo1.jpg"), nil)
	require.NoError(t, err)
	defer img.Close()

	assert.Equal(t, []float64{0, 0, 0}, neutralGain(img, 3))
}

// sumGainmap adds a sample of the pixels, enough to tell whether a blur ran.
func sumGainmap(img *vips.Image) (float64, error) {
	var sum float64
	for y := 0; y < img.PageHeight(); y += 64 {
		for x := 0; x < img.Width(); x += 64 {
			pixel, err := img.Getpoint(x, y, nil)
			if err != nil {
				return 0, err
			}
			sum += pixel[0]
		}
	}
	return sum, nil
}

func TestGainmapHelpersWithoutGainmap(t *testing.T) {
	img, err := vips.NewImageFromFile(filepath.Join(testDataDir, "demo1.jpg"), nil)
	require.NoError(t, err)
	defer img.Close()

	setGainmapColourSpace(img)
	assert.False(t, img.HasField("gainmap-use-base-cg"))
	assert.NoError(t, blurGainmap(img, 5, img.Width()))
}

func loadUltraHDR(t *testing.T) *vips.Image {
	t.Helper()
	if !vips.HasOperation("uhdrload") {
		t.Skip("libvips is built without libultrahdr")
	}
	img, err := vips.NewImageFromFile(filepath.Join(testDataDir, "ultra-hdr.jpg"), nil)
	require.NoError(t, err)
	return img
}

func TestGainmapFilters(t *testing.T) {
	src := loadUltraHDR(t)
	srcWidth, srcHeight := src.Width(), src.PageHeight()
	src.Close()

	t.Run("crop", func(t *testing.T) {
		img := loadUltraHDR(t)
		defer img.Close()
		processed, err := crop(context.Background(), img, nil, "0.25", "0.25", "0.5", "0.5")
		require.NoError(t, err)
		assert.True(t, processed)
		assert.InDelta(t, srcWidth/2, img.Width(), 2)
		assert.InDelta(t, srcHeight/2, img.PageHeight(), 2)
		width, height := gainmapSize(t, img)
		assert.InDelta(t, img.Width()/2, width, 2)
		assert.InDelta(t, img.PageHeight()/2, height, 2)
	})

	t.Run("proportion", func(t *testing.T) {
		img := loadUltraHDR(t)
		defer img.Close()
		processed, err := proportion(context.Background(), img, nil, "50")
		require.NoError(t, err)
		assert.True(t, processed)
		assert.InDelta(t, srcWidth/2, img.Width(), 2)
		assert.InDelta(t, srcHeight/2, img.PageHeight(), 2)
		width, height := gainmapSize(t, img)
		assert.InDelta(t, img.Width()/2, width, 2)
		assert.InDelta(t, img.PageHeight()/2, height, 2)
	})

	t.Run("strip exif", func(t *testing.T) {
		img := loadUltraHDR(t)
		defer img.Close()

		processed, err := stripExif(context.Background(), img, nil)
		require.NoError(t, err)
		assert.True(t, processed)
		width, height := gainmapSize(t, img)
		assert.NotZero(t, width, "the gain map should survive strip_exif")
		assert.NotZero(t, height)
	})

	t.Run("trim", func(t *testing.T) {
		img := loadUltraHDR(t)
		defer img.Close()
		require.NoError(t, embedGainmap(img, 60, 60, srcWidth+120, srcHeight+120,
			&vips.EmbedOptions{Extend: vips.ExtendWhite}))
		require.NoError(t, img.EmbedMultiPage(60, 60, srcWidth+120, srcHeight+120,
			&vips.EmbedMultiPageOptions{Extend: vips.ExtendWhite}))

		processed, err := trim(context.Background(), img, nil, "10")
		require.NoError(t, err)
		assert.True(t, processed)
		assert.InDelta(t, srcWidth, img.Width(), 4)
		assert.InDelta(t, srcHeight, img.PageHeight(), 4)
		gainmapWidth, gainmapHeight := gainmapSize(t, img)
		assert.InDelta(t, img.Width()/2, gainmapWidth, 4)
		assert.InDelta(t, img.PageHeight()/2, gainmapHeight, 4)
	})
}

func TestGainmapHelperEdges(t *testing.T) {
	t.Run("crop clamps to the gain map", func(t *testing.T) {
		img := loadUltraHDR(t)
		defer img.Close()
		width, height := gainmapSize(t, img)
		require.NoError(t, cropGainmap(img, -10, -10, 100000, 100000))
		gotWidth, gotHeight := gainmapSize(t, img)
		assert.Equal(t, width, gotWidth)
		assert.Equal(t, height, gotHeight)
	})

	t.Run("crop with no area is a no-op", func(t *testing.T) {
		img := loadUltraHDR(t)
		defer img.Close()
		require.NoError(t, cropGainmap(img, 10, 10, 0, 0))
	})

	t.Run("images without a gain map are skipped", func(t *testing.T) {
		img, err := vips.NewImageFromFile(filepath.Join(testDataDir, "demo1.jpg"), nil)
		require.NoError(t, err)
		defer img.Close()
		require.NoError(t, attachGainmap(img))
		require.NoError(t, cropGainmap(img, 10, 10, 100, 100))
		require.NoError(t, embedGainmap(img, 10, 10, img.Width()+20, img.PageHeight()+20,
			&vips.EmbedOptions{Extend: vips.ExtendBlack}))
		require.NoError(t, rotateGainmap(img, 90))
		require.NoError(t, flipGainmap(img, vips.DirectionHorizontal))
	})
}

func TestUltraHDRPipeline(t *testing.T) {
	src := loadUltraHDR(t)
	srcWidth, srcHeight := src.Width(), src.PageHeight()
	gainWidth, gainHeight := gainmapSize(t, src)
	scaleX := float64(gainWidth) / float64(srcWidth)
	scaleY := float64(gainHeight) / float64(srcHeight)
	src.Close()

	v := NewProcessor()
	load := func(string) (*imagor.Blob, error) {
		return imagor.NewBlobFromFile(filepath.Join(testDataDir, "ultra-hdr.jpg")), nil
	}

	tests := []struct {
		path          string
		width, height int
	}{
		{"100x100/ultra-hdr.jpg", 100, 100},
		{"-100x100/ultra-hdr.jpg", 100, 100},
		{"100x100/10x10/filters:fill(white)/ultra-hdr.jpg", 120, 120},
		{"filters:proportion(50)/ultra-hdr.jpg", int(float64(srcWidth) * 0.5), int(float64(srcHeight) * 0.5)},
		{"filters:crop(0.1,0.1,0.8,0.8)/ultra-hdr.jpg", 0, 0},
		{"filters:rotate(90)/ultra-hdr.jpg", srcHeight, srcWidth},
		{"filters:orient(90)/ultra-hdr.jpg", srcHeight, srcWidth},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			blob := imagor.NewBlobFromFile(filepath.Join(testDataDir, "ultra-hdr.jpg"))
			out, err := v.Process(context.Background(), blob, imagorpath.Parse(tt.path), load)
			require.NoError(t, err)
			data, err := out.ReadAll()
			require.NoError(t, err)

			img, err := vips.NewImageFromBuffer(data, nil)
			require.NoError(t, err)
			defer img.Close()
			if tt.width > 0 {
				assert.InDelta(t, tt.width, img.Width(), 2)
				assert.InDelta(t, tt.height, img.PageHeight(), 2)
			}
			gotWidth, gotHeight := gainmapSize(t, img)
			assert.InDelta(t, float64(img.Width())*scaleX, gotWidth, 3)
			assert.InDelta(t, float64(img.PageHeight())*scaleY, gotHeight, 3)
		})
	}
}

func TestSetGainmapColourSpace(t *testing.T) {
	img := loadUltraHDR(t)
	defer img.Close()

	setGainmapColourSpace(img)
	value, err := img.GetInt("gainmap-use-base-cg")
	require.NoError(t, err)
	assert.Equal(t, 1, value)
}

func TestStripMetadataKeepsGainmap(t *testing.T) {
	v := NewProcessor()
	load := func(string) (*imagor.Blob, error) {
		return imagor.NewBlobFromFile(filepath.Join(testDataDir, "ultra-hdr.jpg")), nil
	}
	blob := imagor.NewBlobFromFile(filepath.Join(testDataDir, "ultra-hdr.jpg"))
	out, err := v.Process(context.Background(), blob,
		imagorpath.Parse("400x400/filters:strip_metadata()/ultra-hdr.jpg"), load)
	require.NoError(t, err)
	data, err := out.ReadAll()
	require.NoError(t, err)

	img, err := vips.NewImageFromBuffer(data, nil)
	require.NoError(t, err)
	defer img.Close()
	_, ok := img.Gainmap()
	assert.True(t, ok, "the gain map is part of the image, not metadata")
}

// TestGainmapPaths holds the invariant the per-op replay relies on: whatever
// a request does to an Ultra HDR image, a JPEG result keeps its gain map.
var gainmapPathTests = []struct {
	path string
	kept bool
}{
	{"400x400/ultra-hdr.jpg", true},
	{"fit-in/400x400/ultra-hdr.jpg", true},
	{"-400x400/ultra-hdr.jpg", true},
	{"400x-400/ultra-hdr.jpg", true},
	{"300x200:500x400/400x400/ultra-hdr.jpg", true},
	{"400x400/20x20/filters:fill(white)/ultra-hdr.jpg", true},
	{"400x400/filters:orient(90)/ultra-hdr.jpg", true},
	{"400x400/filters:rotate(45)/ultra-hdr.jpg", true},
	{"400x400/filters:crop(0.1,0.1,0.8,0.8)/ultra-hdr.jpg", true},
	{"400x400/filters:proportion(50)/ultra-hdr.jpg", true},
	{"400x400/filters:fill(white)/ultra-hdr.jpg", true},
	{"400x400/filters:blur(2)/ultra-hdr.jpg", true},
	{"400x400/filters:sharpen(2)/ultra-hdr.jpg", true},
	{"400x400/filters:saturation(50)/ultra-hdr.jpg", true},
	{"400x400/filters:brightness(10)/ultra-hdr.jpg", true},
	{"400x400/filters:contrast(20)/ultra-hdr.jpg", true},
	{"400x400/filters:hue(90)/ultra-hdr.jpg", true},
	{"400x400/filters:invert()/ultra-hdr.jpg", true},
	{"400x400/filters:pixelate(5)/ultra-hdr.jpg", true},
	{"400x400/filters:round_corner(10)/ultra-hdr.jpg", true},
	{"400x400/filters:round_corner(10,10,white)/ultra-hdr.jpg", true},
	{"400x400/filters:background_color(white)/ultra-hdr.jpg", true},
	{"400x400/filters:strip_exif()/ultra-hdr.jpg", true},
	{"400x400/filters:strip_metadata()/ultra-hdr.jpg", true},
	{"400x400/filters:strip_icc()/ultra-hdr.jpg", true},
	{"400x400/filters:to_colorspace(srgb)/ultra-hdr.jpg", true},
	{"400x400/filters:quality(80)/ultra-hdr.jpg", true},
	{"400x400/filters:max_bytes(100000)/ultra-hdr.jpg", true},
	{"400x400/filters:watermark(gopher-front.png,10,10,50)/ultra-hdr.jpg", true},
	{"400x400/filters:text(hello,10,10)/ultra-hdr.jpg", true},
	{"400x400/filters:image(/fit-in/50x50/gopher-front.png,10,10)/ultra-hdr.jpg", true},
	{"400x400/filters:trim(10)/ultra-hdr.jpg", true},
	{"400x400/filters:label(hello,10,10)/ultra-hdr.jpg", true},
	{"400x400/filters:modulate(50,50,50)/ultra-hdr.jpg", true},
	{"400x400/filters:rgb(10,0,0)/ultra-hdr.jpg", true},
	{"400x400/filters:padding(white,10,10,10,10)/ultra-hdr.jpg", true},
	{"stretch/400x400/ultra-hdr.jpg", true},
	{"fit-in/400x400/filters:fill(blur)/ultra-hdr.jpg", true},
	{"fit-in/400x400/filters:fill(auto)/ultra-hdr.jpg", true},
	{"fit-in/400x400/filters:fill(white)/ultra-hdr.jpg", true},
	{"400x400/filters:focal(0.5x0.5)/ultra-hdr.jpg", true},
	{"filters:focal(0.5x0.5)/200x200/ultra-hdr.jpg", true},
	{"400x400/filters:watermark(gopher-front.png,repeat,bottom,10)/ultra-hdr.jpg", true},
	{"400x400/filters:format(jpeg)/ultra-hdr.jpg", true},
	{"400x400/filters:raw()/ultra-hdr.jpg", true},
	{"400x400/filters:preview()/ultra-hdr.jpg", true},
	{"400x400/filters:no_upscale()/ultra-hdr.jpg", true},
	{"400x400/filters:lossless()/ultra-hdr.jpg", true},
	{"400x400/filters:max_frames(1)/ultra-hdr.jpg", true},
	{"400x400/filters:grayscale()/ultra-hdr.jpg", false},
	{"400x400/filters:to_colorspace(cmyk)/ultra-hdr.jpg", false},
	{"400x400/filters:redact(blur)/ultra-hdr.jpg", true},
	{"400x400/filters:redact_oval(blur)/ultra-hdr.jpg", true},
	{"400x400/filters:draw_detections()/ultra-hdr.jpg", true},
	{"400x400/filters:format(webp)/ultra-hdr.jpg", false},
}

func TestGainmapPaths(t *testing.T) {
	v := NewProcessor(WithDetector(&stubDetector{regions: []imagor.DetectorRegion{
		{Name: "face", Left: 0.25, Top: 0.25, Right: 0.75, Bottom: 0.75},
	}}))
	for _, tt := range gainmapPathTests {
		t.Run(tt.path, func(t *testing.T) {
			load := func(name string) (*imagor.Blob, error) {
				return imagor.NewBlobFromFile(filepath.Join(testDataDir, name)), nil
			}
			blob := imagor.NewBlobFromFile(filepath.Join(testDataDir, "ultra-hdr.jpg"))
			out, err := v.Process(context.Background(), blob, imagorpath.Parse(tt.path), load)
			require.NoError(t, err)
			data, err := out.ReadAll()
			require.NoError(t, err)

			img, err := vips.NewImageFromBuffer(data, nil)
			require.NoError(t, err)
			defer img.Close()
			gainmap, ok := img.Gainmap()
			if ok {
				gainmap.Close()
			}
			assert.Equal(t, tt.kept, ok, tt.path)
		})
	}
}

// TestGainmapFilterRegistry fails when a filter is registered without a case
// above, so a new filter cannot skip the gain map silently.
func TestGainmapFilterRegistry(t *testing.T) {
	covered := map[string]bool{}
	for _, tt := range gainmapPathTests {
		for _, m := range regexp.MustCompile(`filters:([a-z_]+)`).FindAllStringSubmatch(tt.path, -1) {
			covered[m[1]] = true
		}
	}
	for name := range NewProcessor().Filters {
		if !covered[name] {
			t.Errorf("%s has no gain map case in TestGainmapPaths", name)
		}
	}
}
