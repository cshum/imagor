package vipsprocessor

import (
	"context"
	"github.com/cshum/imagor"
	"github.com/cshum/imagor/imagorpath"
	"go.uber.org/zap"
	"golang.org/x/image/colornames"
	"image/color"
	"math"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

type FilterFunc func(ctx context.Context, img *ImageRef, load imagor.LoadFunc, args ...string) (err error)

type FilterMap map[string]FilterFunc

var l sync.RWMutex
var cnt int

type VipsProcessor struct {
	Filters            FilterMap
	DisableBlur        bool
	DisableFilters     []string
	MaxFilterOps       int
	Logger             *zap.Logger
	Concurrency        int
	MaxCacheFiles      int
	MaxCacheMem        int
	MaxCacheSize       int
	MaxWidth           int
	MaxHeight          int
	MaxResolution      int
	MaxAnimationFrames int
	MozJPEG            bool
	Debug              bool
}

func New(options ...Option) *VipsProcessor {
	v := &VipsProcessor{
		MaxWidth:           9999,
		MaxHeight:          9999,
		MaxResolution:      16800000,
		Concurrency:        1,
		MaxFilterOps:       -1,
		MaxAnimationFrames: -1,
		Logger:             zap.NewNop(),
	}
	v.Filters = FilterMap{
		"watermark":        v.watermark,
		"round_corner":     roundCorner,
		"rotate":           rotate,
		"grayscale":        grayscale,
		"brightness":       brightness,
		"background_color": backgroundColor,
		"contrast":         contrast,
		"modulate":         modulate,
		"hue":              hue,
		"saturation":       saturation,
		"rgb":              rgb,
		"blur":             blur,
		"sharpen":          sharpen,
		"strip_icc":        stripIcc,
		"strip_exif":       stripIcc,
		"trim":             trim,
		"frames":           frames,
		"padding":          v.padding,
		"proportion":       proportion,
	}
	for _, option := range options {
		option(v)
	}
	if v.DisableBlur {
		v.DisableFilters = append(v.DisableFilters, "blur", "sharpen")
	}
	for _, name := range v.DisableFilters {
		delete(v.Filters, name)
	}
	if v.Concurrency == -1 {
		v.Concurrency = runtime.NumCPU()
	}
	return v
}

func (v *VipsProcessor) Startup(_ context.Context) error {
	l.Lock()
	defer l.Unlock()
	cnt++
	if cnt > 1 {
		return nil
	}
	if v.Debug {
		loggingSettings(func(domain string, level LogLevel, msg string) {
			switch level {
			case LogLevelDebug:
				v.Logger.Debug(domain, zap.String("log", msg))
			case LogLevelMessage, LogLevelInfo:
				v.Logger.Info(domain, zap.String("log", msg))
			case LogLevelWarning, LogLevelCritical:
				v.Logger.Warn(domain, zap.String("log", msg))
			case LogLevelError:
				v.Logger.Error(domain, zap.String("log", msg))
			}
		}, LogLevelDebug)
	} else {
		loggingSettings(func(domain string, level LogLevel, msg string) {
			v.Logger.Error(domain, zap.String("log", msg))
		}, LogLevelError)
	}
	startup(&config{
		MaxCacheFiles:    v.MaxCacheFiles,
		MaxCacheMem:      v.MaxCacheMem,
		MaxCacheSize:     v.MaxCacheSize,
		ConcurrencyLevel: v.Concurrency,
	})
	return nil
}

func (v *VipsProcessor) Shutdown(_ context.Context) error {
	l.Lock()
	defer l.Unlock()
	if cnt <= 0 {
		return nil
	}
	cnt--
	if cnt == 0 {
		shutdown()
	}
	return nil
}

func focalSplit(r rune) bool {
	return r == 'x' || r == ',' || r == ':'
}

func (v *VipsProcessor) Process(
	ctx context.Context, blob *imagor.Blob, p imagorpath.Params, load imagor.LoadFunc,
) (*imagor.Blob, error) {
	var (
		thumbnailNotSupported bool
		upscale               = true
		stretch               = p.Stretch
		thumbnail             = false
		img                   *ImageRef
		format                = ImageTypeUnknown
		maxN                  = v.MaxAnimationFrames
		maxBytes              int
		focalRects            []focal
		err                   error
	)
	ctx = withInitImageRefs(ctx)
	defer closeImageRefs(ctx)
	if p.Trim {
		thumbnailNotSupported = true
	}
	if p.FitIn {
		upscale = false
	}
	if maxN == 0 || maxN < -1 {
		maxN = 1
	}
	for _, p := range p.Filters {
		switch p.Name {
		case "format":
			if typ, ok := imageTypeMap[p.Args]; ok {
				format = typ
				if format != ImageTypeGIF && format != ImageTypeWEBP {
					// no frames if export format not support animation
					maxN = 1
				}
			}
			break
		case "stretch":
			stretch = true
			break
		case "upscale":
			upscale = true
			break
		case "no_upscale":
			upscale = false
			break
		case "fill", "background_color":
			if args := strings.Split(p.Args, ","); args[0] == "auto" {
				thumbnailNotSupported = true
			}
			break
		case "max_bytes":
			if n, _ := strconv.Atoi(p.Args); n > 0 {
				maxBytes = n
				thumbnailNotSupported = true
			}
			break
		case "focal":
			thumbnailNotSupported = true
			break
		case "trim":
			thumbnailNotSupported = true
			break
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
				size := SizeDown
				if upscale {
					size = SizeBoth
				}
				if img, err = v.newThumbnail(
					blob, w, h, InterestingNone, size, maxN,
				); err != nil {
					return nil, err
				}
				thumbnail = true
			}
		} else if stretch {
			if p.Width > 0 && p.Height > 0 {
				if img, err = v.newThumbnail(
					blob, p.Width, p.Height,
					InterestingNone, SizeForce, maxN,
				); err != nil {
					return nil, err
				}
				thumbnail = true
			}
		} else {
			if p.Width > 0 && p.Height > 0 {
				interest := InterestingNone
				if p.Smart {
					interest = InterestingAttention
					thumbnail = true
				} else if (p.VAlign == imagorpath.VAlignTop && p.HAlign == "") ||
					(p.HAlign == imagorpath.HAlignLeft && p.VAlign == "") {
					interest = InterestingLow
					thumbnail = true
				} else if (p.VAlign == imagorpath.VAlignBottom && p.HAlign == "") ||
					(p.HAlign == imagorpath.HAlignRight && p.VAlign == "") {
					interest = InterestingHigh
					thumbnail = true
				} else if (p.VAlign == "" || p.VAlign == "middle") &&
					(p.HAlign == "" || p.HAlign == "center") {
					interest = InterestingCentre
					thumbnail = true
				}
				if thumbnail {
					if img, err = v.newThumbnail(
						blob, p.Width, p.Height,
						interest, SizeBoth, maxN,
					); err != nil {
						return nil, err
					}
				}
			} else if p.Width > 0 && p.Height == 0 {
				if img, err = v.newThumbnail(
					blob, p.Width, v.MaxHeight,
					InterestingNone, SizeBoth, maxN,
				); err != nil {
					return nil, err
				}
				thumbnail = true
			} else if p.Height > 0 && p.Width == 0 {
				if img, err = v.newThumbnail(
					blob, v.MaxWidth, p.Height,
					InterestingNone, SizeBoth, maxN,
				); err != nil {
					return nil, err
				}
				thumbnail = true
			}
		}
	}
	if !thumbnail {
		if thumbnailNotSupported {
			if img, err = v.newImage(blob, maxN); err != nil {
				return nil, err
			}
		} else {
			if img, err = v.newThumbnail(
				blob, v.MaxWidth, v.MaxHeight,
				InterestingNone, SizeDown, maxN,
			); err != nil {
				return nil, err
			}
		}
	}
	AddImageRef(ctx, img)
	var (
		quality    int
		pageN      = img.Height() / img.PageHeight()
		origWidth  = float64(img.Width())
		origHeight = float64(img.PageHeight())
	)
	if format == ImageTypeUnknown {
		format = img.Format()
	}
	SetPageN(ctx, pageN)
	if v.Debug {
		v.Logger.Debug("image",
			zap.Int("width", img.Width()),
			zap.Int("height", img.Height()),
			zap.Int("page_height", img.PageHeight()),
			zap.Int("page_n", pageN))
	}
	for _, p := range p.Filters {
		switch p.Name {
		case "quality":
			quality, _ = strconv.Atoi(p.Args)
			break
		case "autojpg":
			format = ImageTypeJPEG
			break
		case "focal":
			if args := strings.FieldsFunc(p.Args, focalSplit); len(args) == 4 {
				f := focal{}
				f.Left, _ = strconv.ParseFloat(args[0], 64)
				f.Top, _ = strconv.ParseFloat(args[1], 64)
				f.Right, _ = strconv.ParseFloat(args[2], 64)
				f.Bottom, _ = strconv.ParseFloat(args[3], 64)
				if f.Left < 1 && f.Top < 1 && f.Right <= 1 && f.Bottom <= 1 {
					f.Left *= origWidth
					f.Right *= origWidth
					f.Top *= origHeight
					f.Bottom *= origHeight
				}
				if f.Right > f.Left && f.Bottom > f.Top {
					focalRects = append(focalRects, f)
				}
			}
			break
		}
	}
	if err := v.process(ctx, img, p, load, thumbnail, stretch, upscale, focalRects); err != nil {
		return nil, wrapErr(err)
	}
	for {
		buf, meta, err := v.export(img, format, quality)
		if err != nil {
			return nil, wrapErr(err)
		}
		if maxBytes > 0 && (quality > 10 || quality == 0) && format != ImageTypePNG {
			ln := len(buf)
			if v.Debug {
				v.Logger.Debug("max_bytes",
					zap.Int("bytes", ln),
					zap.Int("quality", quality),
				)
			}
			if ln > maxBytes {
				if quality == 0 {
					quality = 80
				}
				delta := float64(ln) / float64(maxBytes)
				switch {
				case delta > 3:
					quality = quality * 25 / 100
				case delta > 1.5:
					quality = quality * 50 / 100
				default:
					quality = quality * 75 / 100
				}
				if err := ctx.Err(); err != nil {
					return nil, wrapErr(err)
				}
				continue
			}
		}
		b := imagor.NewBlobFromBytes(buf)
		if meta != nil {
			b.Meta = getMeta(meta)
		}
		return b, nil
	}
}

func getMeta(meta *ImageMetadata) *imagor.Meta {
	format := ImageTypes[meta.Format]
	contentType := imageMimeTypeMap[format]
	pages := 1
	if p := meta.Pages; p > 1 {
		pages = p
	}
	return &imagor.Meta{
		Format:      format,
		ContentType: contentType,
		Width:       meta.Width,
		Height:      meta.Height / pages,
		Orientation: meta.Orientation,
		Pages:       pages,
	}
}

var imageTypeMap = map[string]ImageType{
	"gif":    ImageTypeGIF,
	"jpeg":   ImageTypeJPEG,
	"jpg":    ImageTypeJPEG,
	"magick": ImageTypeMagick,
	"pdf":    ImageTypePDF,
	"png":    ImageTypePNG,
	"svg":    ImageTypeSVG,
	"tiff":   ImageTypeTIFF,
	"webp":   ImageTypeWEBP,
	"heif":   ImageTypeHEIF,
	"bmp":    ImageTypeBMP,
	"avif":   ImageTypeAVIF,
	"jp2":    ImageTypeJP2K,
}

var imageMimeTypeMap = map[string]string{
	"gif":  "image/gif",
	"jpeg": "image/jpeg",
	"jpg":  "image/jpeg",
	"pdf":  "application/pdf",
	"png":  "image/png",
	"svg":  "image/svg+xml",
	"tiff": "image/tiff",
	"webp": "image/webp",
	"heif": "image/heif",
	"bmp":  "image/bmp",
	"avif": "image/avif",
	"jp2":  "image/jp2",
}

func (v *VipsProcessor) export(image *ImageRef, format ImageType, quality int) ([]byte, *ImageMetadata, error) {
	switch format {
	case ImageTypePNG:
		opts := NewPngExportParams()
		return image.ExportPng(opts)
	case ImageTypeWEBP:
		opts := NewWebpExportParams()
		if quality > 0 {
			opts.Quality = quality
		}
		return image.ExportWebp(opts)
	case ImageTypeTIFF:
		opts := NewTiffExportParams()
		if quality > 0 {
			opts.Quality = quality
		}
		return image.ExportTiff(opts)
	case ImageTypeGIF:
		opts := NewGifExportParams()
		if quality > 0 {
			opts.Quality = quality
		}
		return image.ExportGIF(opts)
	case ImageTypeAVIF:
		opts := NewAvifExportParams()
		if quality > 0 {
			opts.Quality = quality
		}
		return image.ExportAvif(opts)
	case ImageTypeJP2K:
		opts := NewJp2kExportParams()
		if quality > 0 {
			opts.Quality = quality
		}
		return image.ExportJp2k(opts)
	default:
		opts := NewJpegExportParams()
		if v.MozJPEG {
			opts.Quality = 75
			opts.StripMetadata = true
			opts.OptimizeCoding = true
			opts.Interlace = true
			opts.OptimizeScans = true
			opts.TrellisQuant = true
			opts.QuantTable = 3
		}
		if quality > 0 {
			opts.Quality = quality
		}
		return image.ExportJpeg(opts)
	}
}

func wrapErr(err error) error {
	if err == nil {
		return nil
	}
	msg := strings.TrimSpace(err.Error())
	if strings.HasPrefix(msg, "VipsForeignLoad: buffer is not in a known format") {
		return imagor.ErrUnsupportedFormat
	}
	return imagor.NewError(msg, 406)
}

func (v *VipsProcessor) process(
	ctx context.Context, img *ImageRef, p imagorpath.Params, load imagor.LoadFunc, thumbnail, stretch, upscale bool, focalRects []focal,
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
		// percentage
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
		if err := img.ExtractArea(
			int(cropLeft), int(cropTop), int(cropRight-cropLeft), int(cropBottom-cropTop),
		); err != nil {
			return err
		}
	}
	var (
		w = p.Width
		h = p.Height
	)
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
			if upscale || w < img.Width() || h < img.PageHeight() {
				if err := img.Thumbnail(w, h, InterestingNone); err != nil {
					return err
				}
			}
		} else if stretch {
			if upscale || (w < img.Width() && h < img.PageHeight()) {
				if err := img.ThumbnailWithSize(
					w, h, InterestingNone, SizeForce,
				); err != nil {
					return err
				}
			}
		} else if upscale || w < img.Width() || h < img.PageHeight() {
			interest := InterestingCentre
			if p.Smart {
				interest = InterestingAttention
			} else if float64(w)/float64(h) > float64(img.Width())/float64(img.PageHeight()) {
				if p.VAlign == imagorpath.VAlignTop {
					interest = InterestingLow
				} else if p.VAlign == imagorpath.VAlignBottom {
					interest = InterestingHigh
				}
			} else {
				if p.HAlign == imagorpath.HAlignLeft {
					interest = InterestingLow
				} else if p.HAlign == imagorpath.HAlignRight {
					interest = InterestingHigh
				}
			}
			if p.Smart && len(focalRects) > 0 {
				focalX, focalY := parseFocalPoint(focalRects...)
				if err := v.focalThumbnail(
					img, w, h,
					(focalX-cropLeft)/float64(img.Width()),
					(focalY-cropTop)/float64(img.PageHeight()),
				); err != nil {
					return err
				}
			} else {
				if err := v.thumbnail(img, w, h, interest, SizeBoth); err != nil {
					return err
				}
			}
			if _, err := v.checkResolution(img, nil); err != nil {
				return err
			}
		}
	}
	if p.HFlip {
		if err := img.Flip(DirectionHorizontal); err != nil {
			return err
		}
	}
	if p.VFlip {
		if err := img.Flip(DirectionVertical); err != nil {
			return err
		}
	}
	for i, filter := range p.Filters {
		if err := ctx.Err(); err != nil {
			return err
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
			args = strings.Split(filter.Args, ",")
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
	ctx context.Context, img *ImageRef, pos string, tolerance int,
) (l, t, w, h int, err error) {
	if IsAnimated(ctx) {
		// skip animation support
		return
	}
	var x, y int
	if pos == imagorpath.TrimByBottomRight {
		x = img.Width() - 1
		y = img.PageHeight() - 1
	}
	if tolerance == 0 {
		tolerance = 1
	}
	p, err := img.GetPoint(x, y)
	if err != nil {
		return
	}
	l, t, w, h, err = img.FindTrim(float64(tolerance), &Color{
		R: uint8(p[0]), G: uint8(p[1]), B: uint8(p[2]),
	})
	return
}

func isBlack(c *Color) bool {
	return c.R == 0x00 && c.G == 0x00 && c.B == 0x00
}

func isWhite(c *Color) bool {
	return c.R == 0xff && c.G == 0xff && c.B == 0xff
}

func getColor(img *ImageRef, color string) *Color {
	vc := &Color{}
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
			p, _ := img.GetPoint(x, y)
			if len(p) >= 3 {
				vc.R = uint8(p[0])
				vc.G = uint8(p[1])
				vc.B = uint8(p[2])
			}
		}
	} else if c, ok := colornames.Map[name]; ok {
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
