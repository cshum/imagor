package vips

// #include "vips.h"
import "C"

import (
	"errors"
	"fmt"
	"runtime"
	"strconv"
	"strings"
	"sync"
)

// Image contains a libvips image and manages its lifecycle.
type Image struct {
	// NOTE: We keep a reference to this so that the input buffer is
	// never garbage collected during processing. Some image loaders use random
	// access transcoding and therefore need the original buffer to be in memory.
	buf                 []byte
	image               *C.VipsImage
	format              ImageType
	lock                sync.Mutex
	optimizedIccProfile string
}

// ImageMetadata is a data structure holding the width, height, orientation and other metadata of the picture.
type ImageMetadata struct {
	Format      ImageType
	Width       int
	Height      int
	Colorspace  Interpretation
	Orientation int
	Pages       int
}

type Parameter struct {
	value interface{}
	isSet bool
}

func (p *Parameter) IsSet() bool {
	return p.isSet
}

func (p *Parameter) set(v interface{}) {
	p.value = v
	p.isSet = true
}

type BoolParameter struct {
	Parameter
}

func (p *BoolParameter) Set(v bool) {
	p.set(v)
}

func (p *BoolParameter) Get() bool {
	return p.value.(bool)
}

type IntParameter struct {
	Parameter
}

func (p *IntParameter) Set(v int) {
	p.set(v)
}

func (p *IntParameter) Get() int {
	return p.value.(int)
}

// ImportParams are options for loading an image. Some are type-specific.
// For default loading, use NewImportParams() or specify nil
type ImportParams struct {
	AutoRotate  BoolParameter
	FailOnError BoolParameter
	Page        IntParameter
	NumPages    IntParameter
	Density     IntParameter

	JpegShrinkFactor IntParameter
	HeifThumbnail    BoolParameter
	SvgUnlimited     BoolParameter
}

// NewImportParams creates default ImportParams
func NewImportParams() *ImportParams {
	p := &ImportParams{}
	p.FailOnError.Set(true)
	return p
}

// OptionString convert import params to option_string
func (i *ImportParams) OptionString() string {
	var values []string
	if v := i.NumPages; v.IsSet() {
		values = append(values, "n="+strconv.Itoa(v.Get()))
	}
	if v := i.Page; v.IsSet() {
		values = append(values, "page="+strconv.Itoa(v.Get()))
	}
	if v := i.Density; v.IsSet() {
		values = append(values, "dpi="+strconv.Itoa(v.Get()))
	}
	if v := i.FailOnError; v.IsSet() {
		values = append(values, "fail="+boolToStr(v.Get()))
	}
	if v := i.JpegShrinkFactor; v.IsSet() {
		values = append(values, "shrink="+strconv.Itoa(v.Get()))
	}
	if v := i.AutoRotate; v.IsSet() {
		values = append(values, "autorotate="+boolToStr(v.Get()))
	}
	if v := i.SvgUnlimited; v.IsSet() {
		values = append(values, "unlimited="+boolToStr(v.Get()))
	}
	if v := i.HeifThumbnail; v.IsSet() {
		values = append(values, "thumbnail="+boolToStr(v.Get()))
	}
	return strings.Join(values, ",")
}

// LoadImageFromFile loads an image from file and creates a new Image
func LoadImageFromFile(file string, params *ImportParams) (*Image, error) {
	startupIfNeeded()

	if params == nil {
		params = NewImportParams()
	}

	vipsImage, format, err := vipsImageFromFile(file, params)
	if err != nil {
		return nil, err
	}

	ref := newImageRef(vipsImage, format, nil)
	log("govips", LogLevelDebug, fmt.Sprintf("creating imageRef from file %s", file))
	return ref, nil
}

// LoadImageFromBuffer loads an image buffer and creates a new Image
func LoadImageFromBuffer(buf []byte, params *ImportParams) (*Image, error) {
	startupIfNeeded()

	if params == nil {
		params = NewImportParams()
	}

	vipsImage, format, err := vipsImageFromBuffer(buf, params)
	if err != nil {
		return nil, err
	}

	ref := newImageRef(vipsImage, format, buf)

	log("govips", LogLevelDebug, fmt.Sprintf("created imageRef %p", ref))
	return ref, nil
}

// LoadThumbnailFromFile loads an image from file and creates a new Image with thumbnail crop and size
func LoadThumbnailFromFile(file string, width, height int, crop Interesting, size Size, params *ImportParams) (*Image, error) {
	startupIfNeeded()

	if params == nil {
		params = NewImportParams()
	}

	vipsImage, format, err := vipsThumbnailFromFile(file, width, height, crop, size, params)
	if err != nil {
		return nil, err
	}

	ref := newImageRef(vipsImage, format, nil)

	log("govips", LogLevelDebug, fmt.Sprintf("created imageref %p", ref))
	return ref, nil
}

// Metadata returns the metadata (ImageMetadata struct) of the associated Image
func (r *Image) Metadata() *ImageMetadata {
	return r.metadata(r.format)
}

func (r *Image) metadata(format ImageType) *ImageMetadata {
	return &ImageMetadata{
		Format:      format,
		Width:       r.Width(),
		Height:      r.Height(),
		Colorspace:  r.ColorSpace(),
		Orientation: r.Orientation(),
		Pages:       r.Pages(),
	}
}

// Copy creates a new copy of the given image.
func (r *Image) Copy() (*Image, error) {
	out, err := vipsCopyImage(r.image)
	if err != nil {
		return nil, err
	}

	return newImageRef(out, r.format, r.buf), nil
}

func newImageRef(vipsImage *C.VipsImage, format ImageType, buf []byte) *Image {
	imageRef := &Image{
		image:  vipsImage,
		format: format,
		buf:    buf,
	}
	runtime.SetFinalizer(imageRef, finalizeImage)
	return imageRef
}

func finalizeImage(ref *Image) {
	log("govips", LogLevelDebug, fmt.Sprintf("closing image %p", ref))
	ref.Close()
}

// Close manually closes the image and frees the memory. Calling Close() is optional.
// Images are automatically closed by GC. However, in high volume applications the GC
// can't keep up with the amount of memory, so you might want to manually close the images.
func (r *Image) Close() {
	r.lock.Lock()

	if r.image != nil {
		clearImage(r.image)
		r.image = nil
	}

	r.buf = nil

	r.lock.Unlock()
}

// Format returns the initial format of the vips image when loaded.
func (r *Image) Format() ImageType {
	return r.format
}

// Width returns the width of this image.
func (r *Image) Width() int {
	return int(r.image.Xsize)
}

// Height returns the height of this image.
func (r *Image) Height() int {
	return int(r.image.Ysize)
}

// HasAlpha returns if the image has an alpha layer.
func (r *Image) HasAlpha() bool {
	return vipsHasAlpha(r.image)
}

// Orientation returns the orientation number as it appears in the EXIF, if present
func (r *Image) Orientation() int {
	return vipsGetMetaOrientation(r.image)
}

// Interpretation returns the current interpretation of the color space of the image.
func (r *Image) Interpretation() Interpretation {
	return Interpretation(int(r.image.Type))
}

// ColorSpace returns the interpretation of the current color space. Alias to Interpretation().
func (r *Image) ColorSpace() Interpretation {
	return r.Interpretation()
}

// Pages returns the number of pages in the Image
// For animated images this corresponds to the number of frames
func (r *Image) Pages() int {
	// libvips uses the same attribute (n_pages) to represent the number of pyramid layers in JP2K
	// as we interpret the attribute as frames and JP2K does not support animation we override this with 1
	if r.format == ImageTypeJP2K {
		return 1
	}

	return vipsGetImageNPages(r.image)
}

// PageHeight return the height of a single page
func (r *Image) PageHeight() int {
	return vipsGetPageHeight(r.image)
}

// SetPageHeight set the height of a page
// For animated images this is used when "unrolling" back to frames
func (r *Image) SetPageHeight(height int) error {
	out, err := vipsCopyImage(r.image)
	if err != nil {
		return err
	}

	vipsSetPageHeight(out, height)

	r.setImage(out)
	return nil
}

// SetPageDelay set the page delay array for animation
func (r *Image) SetPageDelay(delay []int) error {
	var data []C.int
	for _, d := range delay {
		data = append(data, C.int(d))
	}
	return vipsImageSetDelay(r.image, data)
}

// ExportJpeg exports the image as JPEG to a buffer.
func (r *Image) ExportJpeg(params *JpegExportParams) ([]byte, *ImageMetadata, error) {
	if params == nil {
		params = NewJpegExportParams()
	}

	buf, err := vipsSaveJPEGToBuffer(r.image, *params)
	if err != nil {
		return nil, nil, err
	}

	return buf, r.metadata(ImageTypeJPEG), nil
}

// ExportPng exports the image as PNG to a buffer.
func (r *Image) ExportPng(params *PngExportParams) ([]byte, *ImageMetadata, error) {
	if params == nil {
		params = NewPngExportParams()
	}

	buf, err := vipsSavePNGToBuffer(r.image, *params)
	if err != nil {
		return nil, nil, err
	}

	return buf, r.metadata(ImageTypePNG), nil
}

// ExportWebp exports the image as WEBP to a buffer.
func (r *Image) ExportWebp(params *WebpExportParams) ([]byte, *ImageMetadata, error) {
	if params == nil {
		params = NewWebpExportParams()
	}

	paramsWithIccProfile := *params
	paramsWithIccProfile.IccProfile = r.optimizedIccProfile

	buf, err := vipsSaveWebPToBuffer(r.image, paramsWithIccProfile)
	if err != nil {
		return nil, nil, err
	}

	return buf, r.metadata(ImageTypeWEBP), nil
}

// ExportHeif exports the image as HEIF to a buffer.
func (r *Image) ExportHeif(params *HeifExportParams) ([]byte, *ImageMetadata, error) {
	if params == nil {
		params = NewHeifExportParams()
	}

	buf, err := vipsSaveHEIFToBuffer(r.image, *params)
	if err != nil {
		return nil, nil, err
	}

	return buf, r.metadata(ImageTypeHEIF), nil
}

// ExportTiff exports the image as TIFF to a buffer.
func (r *Image) ExportTiff(params *TiffExportParams) ([]byte, *ImageMetadata, error) {
	if params == nil {
		params = NewTiffExportParams()
	}

	buf, err := vipsSaveTIFFToBuffer(r.image, *params)
	if err != nil {
		return nil, nil, err
	}

	return buf, r.metadata(ImageTypeTIFF), nil
}

// ExportGIF exports the image as GIF to a buffer.
func (r *Image) ExportGIF(params *GifExportParams) ([]byte, *ImageMetadata, error) {
	if params == nil {
		params = NewGifExportParams()
	}

	buf, err := vipsSaveGIFToBuffer(r.image, *params)
	if err != nil {
		return nil, nil, err
	}

	return buf, r.metadata(ImageTypeGIF), nil
}

// ExportAvif exports the image as AVIF to a buffer.
func (r *Image) ExportAvif(params *AvifExportParams) ([]byte, *ImageMetadata, error) {
	if params == nil {
		params = NewAvifExportParams()
	}

	buf, err := vipsSaveAVIFToBuffer(r.image, *params)
	if err != nil {
		return nil, nil, err
	}

	return buf, r.metadata(ImageTypeAVIF), nil
}

// ExportJp2k exports the image as JPEG2000 to a buffer.
func (r *Image) ExportJp2k(params *Jp2kExportParams) ([]byte, *ImageMetadata, error) {
	if params == nil {
		params = NewJp2kExportParams()
	}

	buf, err := vipsSaveJP2KToBuffer(r.image, *params)
	if err != nil {
		return nil, nil, err
	}

	return buf, r.metadata(ImageTypeJP2K), nil
}

// Composite composites the given overlay image on top of the associated image with provided blending mode.
func (r *Image) Composite(overlay *Image, mode BlendMode, x, y int) error {
	out, err := vipsComposite2(r.image, overlay.image, mode, x, y)
	if err != nil {
		return err
	}
	r.setImage(out)
	return nil
}

// AddAlpha adds an alpha channel to the associated image.
func (r *Image) AddAlpha() error {
	if vipsHasAlpha(r.image) {
		return nil
	}

	out, err := vipsAddAlpha(r.image)
	if err != nil {
		return err
	}
	r.setImage(out)
	return nil
}

// Linear passes an image through a linear transformation (i.e. output = input * a + b).
// See https://libvips.github.io/libvips/API/current/libvips-arithmetic.html#vips-linear
func (r *Image) Linear(a, b []float64) error {
	if len(a) != len(b) {
		return errors.New("a and b must be of same length")
	}

	out, err := vipsLinear(r.image, a, b, len(a))
	if err != nil {
		return err
	}
	r.setImage(out)
	return nil
}

// ExtractArea crops the image to a specified area
func (r *Image) ExtractArea(left, top, width, height int) error {
	if r.Height() > r.PageHeight() {
		// use animated extract area if more than 1 pages loaded
		out, err := vipsExtractAreaMultiPage(r.image, left, top, width, height)
		if err != nil {
			return err
		}
		r.setImage(out)
	} else {
		out, err := vipsExtractArea(r.image, left, top, width, height)
		if err != nil {
			return err
		}
		r.setImage(out)
	}
	return nil
}

// RemoveICCProfile removes the ICC Profile information from the image.
// Typically, browsers and other software assume images without profile to be in the sRGB color space.
func (r *Image) RemoveICCProfile() error {
	out, err := vipsCopyImage(r.image)
	if err != nil {
		return err
	}

	vipsRemoveICCProfile(out)

	r.setImage(out)
	return nil
}

// ToColorSpace changes the color space of the image to the interpretation supplied as the parameter.
func (r *Image) ToColorSpace(interpretation Interpretation) error {
	out, err := vipsToColorSpace(r.image, interpretation)
	if err != nil {
		return err
	}
	r.setImage(out)
	return nil
}

// Flatten removes the alpha channel from the image and replaces it with the background color
func (r *Image) Flatten(backgroundColor *Color) error {
	out, err := vipsFlatten(r.image, backgroundColor)
	if err != nil {
		return err
	}
	r.setImage(out)
	return nil
}

// GaussianBlur blurs the image
func (r *Image) GaussianBlur(sigma float64) error {
	out, err := vipsGaussianBlur(r.image, sigma)
	if err != nil {
		return err
	}
	r.setImage(out)
	return nil
}

// Sharpen sharpens the image
// sigma: sigma of the gaussian
// x1: flat/jaggy threshold
// m2: slope for jaggy areas
func (r *Image) Sharpen(sigma float64, x1 float64, m2 float64) error {
	out, err := vipsSharpen(r.image, sigma, x1, m2)
	if err != nil {
		return err
	}
	r.setImage(out)
	return nil
}

// Modulate the colors
func (r *Image) Modulate(brightness, saturation, hue float64) error {
	var err error
	var multiplications []float64
	var additions []float64

	colorspace := r.ColorSpace()
	if colorspace == InterpretationRGB {
		colorspace = InterpretationSRGB
	}

	multiplications = []float64{brightness, saturation, 1}
	additions = []float64{0, 0, hue}

	if r.HasAlpha() {
		multiplications = append(multiplications, 1)
		additions = append(additions, 0)
	}

	err = r.ToColorSpace(InterpretationLCH)
	if err != nil {
		return err
	}

	err = r.Linear(multiplications, additions)
	if err != nil {
		return err
	}

	err = r.ToColorSpace(colorspace)
	if err != nil {
		return err
	}

	return nil
}

// FindTrim returns the bounding box of the non-border part of the image
// Returned values are left, top, width, height
func (r *Image) FindTrim(threshold float64, backgroundColor *Color) (int, int, int, int, error) {
	return vipsFindTrim(r.image, threshold, backgroundColor)
}

// GetPoint reads a single pixel on an image.
// The pixel values are returned in a slice of length n.
func (r *Image) GetPoint(x int, y int) ([]float64, error) {
	n := 3
	if vipsHasAlpha(r.image) {
		n = 4
	}
	return vipsGetPoint(r.image, n, x, y)
}

// Thumbnail resizes the image to the given width and height.
// crop decides algorithm vips uses to shrink and crop to fill target,
func (r *Image) Thumbnail(width, height int, crop Interesting) error {
	out, err := vipsThumbnail(r.image, width, height, crop, SizeBoth)
	if err != nil {
		return err
	}
	r.setImage(out)
	return nil
}

// ThumbnailWithSize resizes the image to the given width and height.
// crop decides algorithm vips uses to shrink and crop to fill target,
// size controls upsize, downsize, both or force
func (r *Image) ThumbnailWithSize(width, height int, crop Interesting, size Size) error {
	out, err := vipsThumbnail(r.image, width, height, crop, size)
	if err != nil {
		return err
	}
	r.setImage(out)
	return nil
}

// Embed embeds the given picture in a new one, i.e. the opposite of ExtractArea
func (r *Image) Embed(left, top, width, height int, extend ExtendStrategy) error {
	if r.Height() > r.PageHeight() {
		out, err := vipsEmbedMultiPage(r.image, left, top, width, height, extend)
		if err != nil {
			return err
		}
		r.setImage(out)
	} else {
		out, err := vipsEmbed(r.image, left, top, width, height, extend)
		if err != nil {
			return err
		}
		r.setImage(out)
	}
	return nil
}

// EmbedBackground embeds the given picture with a background color
func (r *Image) EmbedBackground(left, top, width, height int, backgroundColor *Color) error {
	c := &ColorRGBA{
		R: backgroundColor.R,
		G: backgroundColor.G,
		B: backgroundColor.B,
		A: 255,
	}
	if r.Height() > r.PageHeight() {
		out, err := vipsEmbedMultiPageBackground(r.image, left, top, width, height, c)
		if err != nil {
			return err
		}
		r.setImage(out)
	} else {
		out, err := vipsEmbedBackground(r.image, left, top, width, height, c)
		if err != nil {
			return err
		}
		r.setImage(out)
	}
	return nil
}

// EmbedBackgroundRGBA embeds the given picture with a background rgba color
func (r *Image) EmbedBackgroundRGBA(left, top, width, height int, backgroundColor *ColorRGBA) error {
	if r.Height() > r.PageHeight() {
		out, err := vipsEmbedMultiPageBackground(r.image, left, top, width, height, backgroundColor)
		if err != nil {
			return err
		}
		r.setImage(out)
	} else {
		out, err := vipsEmbedBackground(r.image, left, top, width, height, backgroundColor)
		if err != nil {
			return err
		}
		r.setImage(out)
	}
	return nil
}

// Flip flips the image either horizontally or vertically based on the parameter
func (r *Image) Flip(direction Direction) error {
	out, err := vipsFlip(r.image, direction)
	if err != nil {
		return err
	}
	r.setImage(out)
	return nil
}

// Rotate rotates the image by multiples of 90 degrees
func (r *Image) Rotate(angle Angle) error {
	if r.Height() > r.PageHeight() {
		out, err := vipsRotateMultiPage(r.image, angle)
		if err != nil {
			return err
		}
		r.setImage(out)
	} else {
		out, err := vipsRotate(r.image, angle)
		if err != nil {
			return err
		}
		r.setImage(out)
	}
	return nil
}

// Replicate repeats an image many times across and down
func (r *Image) Replicate(across int, down int) error {
	out, err := vipsReplicate(r.image, across, down)
	if err != nil {
		return err
	}
	r.setImage(out)
	return nil
}

// setImage resets the image for this image and frees the previous one
func (r *Image) setImage(image *C.VipsImage) {
	r.lock.Lock()
	defer r.lock.Unlock()

	if r.image == image {
		return
	}

	if r.image != nil {
		clearImage(r.image)
	}

	r.image = image
}
