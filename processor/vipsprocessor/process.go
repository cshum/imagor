package vipsprocessor

import (
	"context"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/cshum/imagor"
	"github.com/cshum/imagor/imagorpath"
	"github.com/cshum/vipsgen/vips"
	"go.uber.org/zap"
)

var imageTypeMap = map[string]vips.ImageType{
	"gif":  vips.ImageTypeGif,
	"jpeg": vips.ImageTypeJpeg,
	"jpg":  vips.ImageTypeJpeg,
	"pdf":  vips.ImageTypePdf,
	"png":  vips.ImageTypePng,
	"svg":  vips.ImageTypeSvg,
	"tiff": vips.ImageTypeTiff,
	"webp": vips.ImageTypeWebp,
	"heif": vips.ImageTypeHeif,
	"bmp":  vips.ImageTypeBmp,
	"avif": vips.ImageTypeAvif,
	"jp2":  vips.ImageTypeJp2k,
	"jxl":  vips.ImageTypeJxl,
}

// IsAnimationSupported indicates if image type supports animation
func IsAnimationSupported(imageType vips.ImageType) bool {
	return imageType == vips.ImageTypeGif || imageType == vips.ImageTypeWebp
}

// exportParams holds parameters needed for image export
type exportParams struct {
	format        vips.ImageType
	quality       int
	compression   int
	bitdepth      int
	palette       bool
	stripMetadata bool
	maxBytes      int
}

// Process implements imagor.Processor interface
func (v *Processor) Process(
	ctx context.Context, blob *imagor.Blob, p imagorpath.Params, load imagor.LoadFunc,
) (*imagor.Blob, error) {
	ctx = withContext(ctx)
	defer contextDone(ctx)

	// Use image cache for preview() requests: known-size, within cache max dims, no bypass conditions.
	// preview() opts in to base image caching for interactive editing workflows.
	// Skip for crop/focal/page/dpi: cache stores a downscaled single-page copy at default DPI.
	if p.Image != "" && imagorpath.HasFilter(p, "preview") {
		if _, isColor := parseColorImage(p.Image); !isColor {
			sizeKnown := p.Width > 0 && p.Height > 0
			if sizeKnown && p.Width <= v.CacheMaxWidth && p.Height <= v.CacheMaxHeight &&
				!imagorpath.HasCacheBypass(p) &&
				blob.BlobType() != imagor.BlobTypeMemory {
				if memBlob, _, cacheErr := v.loadOrCache(blob, p.Image, 1, nil); cacheErr == nil && memBlob != nil {
					blob = memBlob
				}
			}
		}
	}

	img, err := v.loadAndProcess(ctx, blob, p, load)
	if err != nil {
		return nil, err
	}
	defer img.Close()

	params := v.extractExportParams(p, blob, img)

	// Handle metadata response
	if p.Meta {
		stripExif := imagorpath.HasFilter(p, "strip_exif")
		var metaRegions []imagor.Region
		if v.Detector != nil {
			metaRegions = v.detectRegions(ctx, img)
		}
		return imagor.NewBlobFromJsonMarshal(metadata(img, params.format, stripExif, metaRegions)), nil
	}

	// Strip ICC profile before export when strip_metadata is requested.
	// This ensures proper colour conversion to sRGB before the ICC profile
	// is removed, matching the behaviour of the strip_icc filter.
	if params.stripMetadata {
		if err := stripIcc(ctx, img, load); err != nil {
			return nil, WrapErr(err)
		}
	}

	// Export with max_bytes retry loop
	params.format = supportedSaveFormat(params.format)
	for {
		buf, err := v.export(img, params.format, params.compression, params.quality, params.palette, params.bitdepth, params.stripMetadata)
		if err != nil {
			return nil, WrapErr(err)
		}
		if params.maxBytes > 0 && (params.quality > 10 || params.quality == 0) && params.format != vips.ImageTypePng {
			ln := len(buf)
			if v.Debug {
				v.Logger.Debug("max_bytes",
					zap.Int("bytes", ln),
					zap.Int("quality", params.quality),
				)
			}
			if ln > params.maxBytes {
				if params.quality == 0 {
					params.quality = 80
				}
				delta := float64(ln) / float64(params.maxBytes)
				switch {
				case delta > 3:
					params.quality = params.quality * 25 / 100
				case delta > 1.5:
					params.quality = params.quality * 50 / 100
				default:
					params.quality = params.quality * 75 / 100
				}
				if err := ctx.Err(); err != nil {
					return nil, WrapErr(err)
				}
				continue
			}
		}
		blob := imagor.NewBlobFromBytes(buf)
		if typ, ok := params.format.MimeType(); ok {
			blob.SetContentType(typ)
		}
		return blob, nil
	}
}

// extractExportParams extracts export-related parameters from filters
func (v *Processor) extractExportParams(p imagorpath.Params, blob *imagor.Blob, img *vips.Image) *exportParams {
	var (
		quality       int
		bitdepth      int
		compression   int
		palette       bool
		stripMetadata = v.StripMetadata
		maxBytes      int
		format        = vips.ImageTypeUnknown
	)

	for _, f := range p.Filters {
		if v.disableFilters[f.Name] {
			continue
		}
		switch f.Name {
		case "format":
			if imageType, ok := imageTypeMap[f.Args]; ok {
				format = supportedSaveFormat(imageType)
			}
		case "quality":
			quality, _ = strconv.Atoi(f.Args)
		case "autojpg":
			format = vips.ImageTypeJpeg
		case "palette":
			palette = true
		case "bitdepth":
			bitdepth, _ = strconv.Atoi(f.Args)
		case "compression":
			compression, _ = strconv.Atoi(f.Args)
		case "max_bytes":
			if n, _ := strconv.Atoi(f.Args); n > 0 {
				maxBytes = n
			}
		case "strip_metadata":
			stripMetadata = true
		}
	}

	// Default format from blob/image type
	if format == vips.ImageTypeUnknown {
		if blob != nil && blob.BlobType() == imagor.BlobTypeAVIF {
			format = vips.ImageTypeAvif
		} else {
			format = img.Format()
		}
	}

	return &exportParams{
		format:        format,
		quality:       quality,
		compression:   compression,
		bitdepth:      bitdepth,
		palette:       palette,
		stripMetadata: stripMetadata,
		maxBytes:      maxBytes,
	}
}

// loadAndProcess loads the image from blob and applies all transformations
func (v *Processor) loadAndProcess(
	ctx context.Context, blob *imagor.Blob, p imagorpath.Params, load imagor.LoadFunc,
) (*vips.Image, error) {
	if c, ok := parseColorImage(p.Image); ok {
		w, h := p.Width, p.Height
		if w <= 0 && h <= 0 {
			w, h = 1, 1
		} else if w <= 0 {
			w = h
		} else if h <= 0 {
			h = w
		}
		if !v.Unlimited && w*h > v.MaxResolution {
			return nil, imagor.ErrMaxResolutionExceeded
		}
		if w > v.MaxWidth {
			w = v.MaxWidth
		}
		if h > v.MaxHeight {
			h = v.MaxHeight
		}
		img, err := newColorImage(w, h, c)
		if err != nil {
			return nil, WrapErr(err)
		}
		if v.Debug {
			v.Logger.Debug("color-image",
				zap.Int("width", w),
				zap.Int("height", h),
				zap.Any("color", c))
		}
		// thumbnail=true: image is already at target size, skip resize/crop
		if err := v.applyTransformations(ctx, img, p, load, true, false, false, nil); err != nil {
			img.Close()
			return nil, WrapErr(err)
		}
		return img, nil
	}

	var (
		thumbnailNotSupported bool
		upscale               = true
		stretch               = p.Stretch
		thumbnail             = false
		orient                int
		img                   *vips.Image
		maxN                  = v.MaxAnimationFrames
		page                  = 1
		dpi                   = 0
		err                   error
	)
	if p.Trim || p.VFlip || p.FullFitIn || p.AdaptiveFitIn {
		thumbnailNotSupported = true
	}
	// When a detector is configured, load the full-resolution source so detection
	// runs before any crop. Without this, Smart=true triggers NewThumbnail with
	// InterestingAttention which decodes + attention-crops in one libvips call,
	// leaving only the already-cropped thumbnail for detection.
	if p.Smart && v.Detector != nil {
		thumbnailNotSupported = true
	}
	if p.FitIn && !p.FullFitIn {
		upscale = false
	}
	if maxN == 0 || maxN < -1 {
		maxN = 1
	}
	if blob != nil && !blob.SupportsAnimation() {
		maxN = 1
	}
	for _, f := range p.Filters {
		if v.disableFilters[f.Name] {
			continue
		}
		switch f.Name {
		case "format":
			if imageType, ok := imageTypeMap[f.Args]; ok {
				format := supportedSaveFormat(imageType)
				if !IsAnimationSupported(format) {
					// no frames if export format not support animation
					maxN = 1
				}
			}
		case "max_frames":
			if n, _ := strconv.Atoi(f.Args); n > 0 && (maxN == -1 || n < maxN) {
				maxN = n
			}
		case "stretch":
			stretch = true
		case "upscale":
			upscale = true
		case "no_upscale":
			upscale = false
		case "fill", "background_color":
			if args := strings.Split(f.Args, ","); args[0] == "auto" {
				thumbnailNotSupported = true
			}
		case "page":
			if n, _ := strconv.Atoi(f.Args); n > 0 {
				page = n
			}
		case "dpi":
			if n, _ := strconv.Atoi(f.Args); n > 0 {
				dpi = n
			}
		case "orient":
			if n, _ := strconv.Atoi(f.Args); n > 0 {
				orient = n
				thumbnailNotSupported = true
			}
		case "max_bytes":
			if n, _ := strconv.Atoi(f.Args); n > 0 {
				thumbnailNotSupported = true
			}
		case "trim", "focal", "rotate", "annotate_detection":
			thumbnailNotSupported = true
		}
	}

	if !thumbnailNotSupported &&
		p.CropBottom == 0.0 && p.CropTop == 0.0 && p.CropLeft == 0.0 && p.CropRight == 0.0 {
		// apply shrink-on-load where possible
		if p.FitIn {
			if p.Width > 0 || p.Height > 0 {
				w := p.Width
				h := p.Height
				if w == 0 {
					w = v.MaxWidth
				}
				if h == 0 {
					h = v.MaxHeight
				}
				size := vips.SizeDown
				if upscale {
					size = vips.SizeBoth
				}
				if img, err = v.NewThumbnail(
					ctx, blob, w, h, vips.InterestingNone, size, maxN, page, dpi,
				); err != nil {
					return nil, err
				}
				thumbnail = true
			}
		} else if stretch {
			if p.Width > 0 && p.Height > 0 {
				if img, err = v.NewThumbnail(
					ctx, blob, p.Width, p.Height,
					vips.InterestingNone, vips.SizeForce, maxN, page, dpi,
				); err != nil {
					return nil, err
				}
				thumbnail = true
			}
		} else {
			if p.Width > 0 && p.Height > 0 {
				interest := vips.InterestingNone
				if p.Smart {
					interest = vips.InterestingAttention
					thumbnail = true
				} else if (p.VAlign == imagorpath.VAlignTop && p.HAlign == "") ||
					(p.HAlign == imagorpath.HAlignLeft && p.VAlign == "") {
					interest = vips.InterestingLow
					thumbnail = true
				} else if (p.VAlign == imagorpath.VAlignBottom && p.HAlign == "") ||
					(p.HAlign == imagorpath.HAlignRight && p.VAlign == "") {
					interest = vips.InterestingHigh
					thumbnail = true
				} else if (p.VAlign == "" || p.VAlign == "middle") &&
					(p.HAlign == "" || p.HAlign == "center") {
					interest = vips.InterestingCentre
					thumbnail = true
				}
				if thumbnail {
					size := vips.SizeBoth
					if !upscale {
						size = vips.SizeDown
					}
					if img, err = v.NewThumbnail(
						ctx, blob, p.Width, p.Height,
						interest, size, maxN, page, dpi,
					); err != nil {
						return nil, err
					}
				}
			} else if p.Width > 0 && p.Height == 0 {
				size := vips.SizeBoth
				if !upscale {
					size = vips.SizeDown
				}
				if img, err = v.NewThumbnail(
					ctx, blob, p.Width, v.MaxHeight,
					vips.InterestingNone, size, maxN, page, dpi,
				); err != nil {
					return nil, err
				}
				thumbnail = true
			} else if p.Height > 0 && p.Width == 0 {
				size := vips.SizeBoth
				if !upscale {
					size = vips.SizeDown
				}
				if img, err = v.NewThumbnail(
					ctx, blob, v.MaxWidth, p.Height,
					vips.InterestingNone, size, maxN, page, dpi,
				); err != nil {
					return nil, err
				}
				thumbnail = true
			}
		}
	}
	if !thumbnail {
		if thumbnailNotSupported {
			if img, err = v.NewImage(ctx, blob, maxN, page, dpi); err != nil {
				return nil, err
			}
		} else {
			if img, err = v.NewThumbnail(
				ctx, blob, v.MaxWidth, v.MaxHeight,
				vips.InterestingNone, vips.SizeDown, maxN, page, dpi,
			); err != nil {
				return nil, err
			}
		}
	}

	if orient > 0 {
		// orient rotate before resize
		if err = img.RotMultiPage(getAngle(orient)); err != nil {
			return nil, err
		}
	}

	var (
		origWidth  = float64(img.Width())
		origHeight = float64(img.PageHeight())
	)
	if v.Debug {
		v.Logger.Debug("image",
			zap.Int("width", img.Width()),
			zap.Int("height", img.Height()),
			zap.Int("page_height", img.PageHeight()))
	}

	var focalRects []focal
	for _, f := range p.Filters {
		if v.disableFilters[f.Name] {
			continue
		}
		if f.Name == "focal" {
			args := strings.FieldsFunc(f.Args, argSplit)
			switch len(args) {
			case 4:
				rect := focal{}
				rect.Left, _ = strconv.ParseFloat(args[0], 64)
				rect.Top, _ = strconv.ParseFloat(args[1], 64)
				rect.Right, _ = strconv.ParseFloat(args[2], 64)
				rect.Bottom, _ = strconv.ParseFloat(args[3], 64)
				if rect.Left < 1 && rect.Top < 1 && rect.Right <= 1 && rect.Bottom <= 1 {
					rect.Left *= origWidth
					rect.Right *= origWidth
					rect.Top *= origHeight
					rect.Bottom *= origHeight
				}
				if rect.Right > rect.Left && rect.Bottom > rect.Top {
					focalRects = append(focalRects, rect)
				}
			case 2:
				rect := focal{}
				rect.Left, _ = strconv.ParseFloat(args[0], 64)
				rect.Top, _ = strconv.ParseFloat(args[1], 64)
				if rect.Left < 1 && rect.Top < 1 {
					rect.Left *= origWidth
					rect.Top *= origHeight
				}
				rect.Right = rect.Left + 1
				rect.Bottom = rect.Top + 1
				focalRects = append(focalRects, rect)
			}
		}
	}
	// Run detector when smart mode is active, a detector is configured, and no
	// explicit focal() rects were provided by the caller.  Detection results are
	// normalised ratios; multiply by original dimensions to obtain absolute focal
	// rects that FocalThumbnail / parseFocalPoint already know how to consume.
	if p.Smart && v.Detector != nil && len(focalRects) == 0 {
		detected := v.detectRegions(ctx, img)
		if v.Debug {
			v.Logger.Debug("detector",
				zap.Int("regions", len(detected)),
				zap.Float64("orig_width", origWidth),
				zap.Float64("orig_height", origHeight))
		}
		for _, r := range detected {
			focalRects = append(focalRects, focal{
				Left:   r.Left * origWidth,
				Top:    r.Top * origHeight,
				Right:  r.Right * origWidth,
				Bottom: r.Bottom * origHeight,
			})
		}
	}
	if err := v.applyTransformations(ctx, img, p, load, thumbnail, stretch, upscale, focalRects); err != nil {
		return nil, WrapErr(err)
	}

	return img, nil
}

// applyTransformations applies all image transformations (crop, resize, flip, filters)
func (v *Processor) applyTransformations(
	ctx context.Context, img *vips.Image, p imagorpath.Params, load imagor.LoadFunc, thumbnail, stretch, upscale bool, focalRects []focal,
) error {
	var (
		origWidth  = float64(img.Width())
		origHeight = float64(img.PageHeight())
		cropLeft,
		cropTop,
		cropRight,
		cropBottom float64
	)
	if p.CropRight > 0 || p.CropLeft > 0 || p.CropBottom > 0 || p.CropTop > 0 {
		cropLeft = math.Max(p.CropLeft, 0)
		cropTop = math.Max(p.CropTop, 0)
		cropRight = p.CropRight
		cropBottom = p.CropBottom
		if p.CropLeft < 1 && p.CropTop < 1 && p.CropRight <= 1 && p.CropBottom <= 1 {
			cropLeft = math.Round(cropLeft * origWidth)
			cropTop = math.Round(cropTop * origHeight)
			cropRight = math.Round(cropRight * origWidth)
			cropBottom = math.Round(cropBottom * origHeight)
		}
		if cropRight == 0 {
			cropRight = origWidth - 1
		}
		if cropBottom == 0 {
			cropBottom = origHeight - 1
		}
		cropRight = math.Min(cropRight, origWidth-1)
		cropBottom = math.Min(cropBottom, origHeight-1)
	}
	if p.Trim {
		if l, t, w, h, err := findTrim(ctx, img, p.TrimBy, p.TrimTolerance); err == nil {
			cropLeft = math.Max(cropLeft, float64(l))
			cropTop = math.Max(cropTop, float64(t))
			if cropRight > 0 {
				cropRight = math.Min(cropRight, float64(l+w))
			} else {
				cropRight = float64(l + w)
			}
			if cropBottom > 0 {
				cropBottom = math.Min(cropBottom, float64(t+h))
			} else {
				cropBottom = float64(t + h)
			}
		}
	}
	if cropRight > cropLeft && cropBottom > cropTop {
		if err := img.ExtractAreaMultiPage(
			int(cropLeft), int(cropTop), int(cropRight-cropLeft), int(cropBottom-cropTop),
		); err != nil {
			return err
		}
	}
	var (
		w = p.Width
		h = p.Height
	)

	// Apply adaptive fit-in: swap dimensions if it would get better image definition
	if p.AdaptiveFitIn && w > 0 && h > 0 {
		imgAspect := float64(img.Width()) / float64(img.PageHeight())
		boxAspect := float64(w) / float64(h)
		// If orientations differ (one portrait, one landscape), swap dimensions
		if (imgAspect > 1) != (boxAspect > 1) {
			w, h = h, w
		}
	}

	if w == 0 && h == 0 {
		w = img.Width()
		h = img.PageHeight()
	} else if w == 0 {
		w = img.Width() * h / img.PageHeight()
		if !upscale && w > img.Width() {
			w = img.Width()
		}
	} else if h == 0 {
		h = img.PageHeight() * w / img.Width()
		if !upscale && h > img.PageHeight() {
			h = img.PageHeight()
		}
	}
	if !thumbnail {
		if p.FitIn {
			if p.FullFitIn && w > 0 && h > 0 {
				imgAspect := float64(img.Width()) / float64(img.PageHeight())
				boxAspect := float64(w) / float64(h)

				if imgAspect < boxAspect {
					// Image is taller (portrait) - use width as constraint, height will exceed box
					// Use Ceil to ensure output is always >= requested box (full-fit-in guarantee)
					h = int(math.Ceil(float64(w) / imgAspect))
				} else {
					// Image is wider (landscape) - use height as constraint, width will exceed box
					// Use Ceil to ensure output is always >= requested box (full-fit-in guarantee)
					w = int(math.Ceil(float64(h) * imgAspect))
				}
			}

			if upscale || w < img.Width() || h < img.PageHeight() {
				opts := &vips.ThumbnailImageOptions{Height: h, Crop: vips.InterestingNone}
				if err := img.ThumbnailImage(w, opts); err != nil {
					return err
				}
			}
		} else if stretch {
			if upscale || (w < img.Width() && h < img.PageHeight()) {
				if err := img.ThumbnailImage(
					w, &vips.ThumbnailImageOptions{Height: h, Crop: vips.InterestingNone, Size: vips.SizeForce},
				); err != nil {
					return err
				}
			}
		} else if upscale || w < img.Width() || h < img.PageHeight() {
			interest := vips.InterestingCentre
			if p.Smart {
				interest = vips.InterestingAttention
			} else if float64(w)/float64(h) > float64(img.Width())/float64(img.PageHeight()) {
				if p.VAlign == imagorpath.VAlignTop {
					interest = vips.InterestingLow
				} else if p.VAlign == imagorpath.VAlignBottom {
					interest = vips.InterestingHigh
				}
			} else {
				if p.HAlign == imagorpath.HAlignLeft {
					interest = vips.InterestingLow
				} else if p.HAlign == imagorpath.HAlignRight {
					interest = vips.InterestingHigh
				}
			}
			if len(focalRects) > 0 {
				focalX, focalY := parseFocalPoint(focalRects...)
				if err := v.FocalThumbnail(
					img, w, h,
					(focalX-cropLeft)/float64(img.Width()),
					(focalY-cropTop)/float64(img.PageHeight()),
				); err != nil {
					return err
				}
			} else {
				if err := v.Thumbnail(img, w, h, interest, vips.SizeBoth); err != nil {
					return err
				}
			}
			if _, err := v.CheckResolution(img, nil); err != nil {
				return err
			}
		}
	}
	if p.HFlip {
		if err := img.Flip(vips.DirectionHorizontal); err != nil {
			return err
		}
	}
	if p.VFlip {
		if err := img.Flip(vips.DirectionVertical); err != nil {
			return err
		}
	}
	for i, filter := range p.Filters {
		if err := ctx.Err(); err != nil {
			return err
		}
		if v.disableFilters[filter.Name] {
			continue
		}
		if v.MaxFilterOps > 0 && i >= v.MaxFilterOps {
			if v.Debug {
				v.Logger.Debug("max-filter-ops-exceeded",
					zap.String("name", filter.Name), zap.String("args", filter.Args))
			}
			break
		}
		start := time.Now()
		var args []string
		if filter.Args != "" {
			args = imagorpath.SplitArgs(filter.Args)
		}
		if fn := v.Filters[filter.Name]; fn != nil {
			if err := fn(ctx, img, load, args...); err != nil {
				return err
			}
		} else if filter.Name == "fill" {
			if err := v.fill(ctx, img, w, h,
				p.PaddingLeft, p.PaddingTop, p.PaddingRight, p.PaddingBottom,
				filter.Args); err != nil {
				return err
			}
		}
		if v.Debug {
			v.Logger.Debug("filter",
				zap.String("name", filter.Name), zap.String("args", filter.Args),
				zap.Duration("took", time.Since(start)))
		}
	}
	return nil
}

// MetaRegion is a detected region of interest in the output image, in absolute pixels.
type MetaRegion struct {
	Left   int `json:"left"`
	Top    int `json:"top"`
	Right  int `json:"right"`
	Bottom int `json:"bottom"`
}

// Metadata image attributes
type Metadata struct {
	Format          string            `json:"format"`
	ContentType     string            `json:"content_type"`
	Width           int               `json:"width"`
	Height          int               `json:"height"`
	Orientation     int               `json:"orientation"`
	Pages           int               `json:"pages"`
	Bands           int               `json:"bands"`
	Exif            map[string]string `json:"exif"`
	DetectedRegions []MetaRegion      `json:"detected_regions,omitempty"`
}

func metadata(img *vips.Image, format vips.ImageType, stripExif bool, regions []imagor.Region) *Metadata {
	pages := 1
	if IsAnimationSupported(format) {
		pages = img.Height() / img.PageHeight()
	}
	if format == vips.ImageTypePdf {
		pages = img.Pages()
	}
	exif := map[string]string{}
	if !stripExif {
		exif = extractExif(img.Exif())
	}
	var metaRegions []MetaRegion
	for _, r := range regions {
		metaRegions = append(metaRegions, MetaRegion{
			Left:   int(math.Round(r.Left * float64(img.Width()))),
			Top:    int(math.Round(r.Top * float64(img.PageHeight()))),
			Right:  int(math.Round(r.Right * float64(img.Width()))),
			Bottom: int(math.Round(r.Bottom * float64(img.PageHeight()))),
		})
	}
	mimeType, _ := format.MimeType()
	return &Metadata{
		Format:          string(format),
		ContentType:     mimeType,
		Width:           img.Width(),
		Height:          img.PageHeight(),
		Pages:           pages,
		Bands:           img.Bands(),
		Orientation:     img.Orientation(),
		Exif:            exif,
		DetectedRegions: metaRegions,
	}
}

func supportedSaveFormat(format vips.ImageType) vips.ImageType {
	switch format {
	case vips.ImageTypePng, vips.ImageTypeWebp, vips.ImageTypeTiff, vips.ImageTypeGif, vips.ImageTypeAvif, vips.ImageTypeHeif, vips.ImageTypeJp2k, vips.ImageTypeJxl:
		return format
	}
	return vips.ImageTypeJpeg
}

func (v *Processor) export(
	image *vips.Image, format vips.ImageType, compression int, quality int, palette bool, bitdepth int, stripMetadata bool,
) ([]byte, error) {
	if _, err := v.CheckResolution(image, nil); err != nil {
		return nil, err
	}
	switch format {
	case vips.ImageTypePng:
		opts := &vips.PngsaveBufferOptions{
			Q:           quality,
			Palette:     palette,
			Bitdepth:    bitdepth,
			Compression: compression,
		}
		if stripMetadata {
			opts.Keep = vips.KeepNone
		} else {
			opts.Keep = vips.KeepAll
		}
		return image.PngsaveBuffer(opts)
	case vips.ImageTypeWebp:
		opts := &vips.WebpsaveBufferOptions{
			Q: quality,
		}
		if stripMetadata {
			opts.Keep = vips.KeepNone
		} else {
			opts.Keep = vips.KeepAll
		}
		return image.WebpsaveBuffer(opts)
	case vips.ImageTypeJxl:
		opts := &vips.JxlsaveBufferOptions{
			Q: quality,
		}
		if stripMetadata {
			opts.Keep = vips.KeepNone
		} else {
			opts.Keep = vips.KeepAll
		}
		return image.JxlsaveBuffer(opts)
	case vips.ImageTypeTiff:
		opts := &vips.TiffsaveBufferOptions{
			Q: quality,
		}
		if stripMetadata {
			opts.Keep = vips.KeepNone
		} else {
			opts.Keep = vips.KeepAll
		}
		return image.TiffsaveBuffer(opts)
	case vips.ImageTypeGif:
		opts := &vips.GifsaveBufferOptions{}
		if stripMetadata {
			opts.Keep = vips.KeepNone
		} else {
			opts.Keep = vips.KeepAll
		}
		return image.GifsaveBuffer(opts)
	case vips.ImageTypeAvif:
		opts := &vips.HeifsaveBufferOptions{
			Q:           quality,
			Compression: vips.HeifCompressionAv1,
		}
		if stripMetadata {
			opts.Keep = vips.KeepNone
		} else {
			opts.Keep = vips.KeepAll
		}
		opts.Effort = 9 - v.AvifSpeed
		return image.HeifsaveBuffer(opts)
	case vips.ImageTypeHeif:
		opts := &vips.HeifsaveBufferOptions{
			Q: quality,
		}
		if stripMetadata {
			opts.Keep = vips.KeepNone
		} else {
			opts.Keep = vips.KeepAll
		}
		return image.HeifsaveBuffer(opts)
	case vips.ImageTypeJp2k:
		opts := &vips.Jp2ksaveBufferOptions{
			Q: quality,
		}
		if stripMetadata {
			opts.Keep = vips.KeepNone
		} else {
			opts.Keep = vips.KeepAll
		}
		return image.Jp2ksaveBuffer(opts)
	default:
		opts := &vips.JpegsaveBufferOptions{}
		if v.MozJPEG {
			opts.Q = 75
			opts.Keep = vips.KeepNone
			opts.OptimizeCoding = true
			opts.Interlace = true
			opts.OptimizeScans = true
			opts.TrellisQuant = true
			opts.QuantTable = 3
		}
		if quality > 0 {
			opts.Q = quality
		}
		if stripMetadata {
			opts.Keep = vips.KeepNone
		} else if !v.MozJPEG {
			opts.Keep = vips.KeepAll
		}
		return image.JpegsaveBuffer(opts)
	}
}

func argSplit(r rune) bool {
	return r == 'x' || r == ',' || r == ':'
}

type focal struct {
	Left   float64
	Right  float64
	Top    float64
	Bottom float64
}

func parseFocalPoint(focalRects ...focal) (focalX, focalY float64) {
	var sumWeight float64
	for _, f := range focalRects {
		sumWeight += (f.Right - f.Left) * (f.Bottom - f.Top)
	}
	for _, f := range focalRects {
		r := (f.Right - f.Left) * (f.Bottom - f.Top) / sumWeight
		focalX += (f.Left + f.Right) / 2 * r
		focalY += (f.Top + f.Bottom) / 2 * r
	}
	return
}

func findTrim(
	_ context.Context, img *vips.Image, pos string, tolerance int,
) (l, t, w, h int, err error) {
	if isAnimated(img) {
		// skip animation support
		return
	}
	tmp, err := img.Copy(&vips.CopyOptions{Interpretation: vips.InterpretationSrgb})
	if err != nil {
		return
	}
	defer tmp.Close()
	if tmp.HasAlpha() {
		if err = tmp.Flatten(&vips.FlattenOptions{Background: []float64{255, 0, 255}}); err != nil {
			return
		}
	}
	var x, y int
	if pos == imagorpath.TrimByBottomRight {
		x = tmp.Width() - 1
		y = tmp.PageHeight() - 1
	}
	if tolerance == 0 {
		tolerance = 1
	}
	background, err := tmp.Getpoint(x, y, nil)
	if err != nil {
		return
	}
	l, t, w, h, err = tmp.FindTrim(&vips.FindTrimOptions{
		Threshold:  float64(tolerance),
		Background: background,
	})
	return
}

// detectorProbeSize is the maximum dimension (width or height) of the
// downscaled probe image passed to the Detector.  Keeping the probe small
// (~400 px) gives roughly a 10× speed-up over full-resolution detection with
// negligible loss in crop precision.
const detectorProbeSize = 400

// detectRegions creates a cheap downscaled probe from img, exports its raw
// sRGB pixels, and asks the configured Detector to locate regions of interest.
// The returned regions are in normalised [0.0, 1.0] coordinates relative to
// the probe — callers must multiply by original image dimensions before use.
// All errors are treated as non-fatal: an empty slice is returned so the
// caller falls back to the default InterestingAttention crop.
func (v *Processor) detectRegions(ctx context.Context, img *vips.Image) []imagor.Region {
	probe, err := img.Copy(nil)
	if err != nil {
		return nil
	}
	defer probe.Close()

	if err := probe.ThumbnailImage(detectorProbeSize, &vips.ThumbnailImageOptions{
		Height: detectorProbeSize,
		Crop:   vips.InterestingNone,
		Size:   vips.SizeDown,
	}); err != nil {
		return nil
	}
	normalizeSrgb(probe)

	buf, err := probe.WriteToMemory()
	if err != nil {
		return nil
	}
	regions, err := v.Detector.Detect(ctx, buf, probe.Width(), probe.PageHeight(), probe.Bands())
	if err != nil {
		if v.Debug {
			v.Logger.Debug("detector error", zap.Error(err))
		}
		return nil
	}
	return regions
}
