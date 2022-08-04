package vipsprocessor

import (
	"context"
	"github.com/cshum/imagor"
	"github.com/cshum/imagor/imagorpath"
	"github.com/davidbyttow/govips/v2/vips"
	"go.uber.org/zap"
	"runtime"
	"strconv"
	"strings"
	"sync"
)

type FilterFunc func(ctx context.Context, img *vips.ImageRef, load imagor.LoadFunc, args ...string) (err error)

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
		vips.LoggingSettings(func(domain string, level vips.LogLevel, msg string) {
			switch level {
			case vips.LogLevelDebug:
				v.Logger.Debug(domain, zap.String("log", msg))
			case vips.LogLevelMessage, vips.LogLevelInfo:
				v.Logger.Info(domain, zap.String("log", msg))
			case vips.LogLevelWarning, vips.LogLevelCritical:
				v.Logger.Warn(domain, zap.String("log", msg))
			case vips.LogLevelError:
				v.Logger.Error(domain, zap.String("log", msg))
			}
		}, vips.LogLevelDebug)
	} else {
		vips.LoggingSettings(func(domain string, level vips.LogLevel, msg string) {
			v.Logger.Error(domain, zap.String("log", msg))
		}, vips.LogLevelError)
	}
	vips.Startup(&vips.Config{
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
		vips.Shutdown()
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
		img                   *vips.ImageRef
		format                = vips.ImageTypeUnknown
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
				if format != vips.ImageTypeGIF && format != vips.ImageTypeWEBP {
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
				size := vips.SizeDown
				if upscale {
					size = vips.SizeBoth
				}
				if img, err = v.newThumbnail(
					blob, w, h, vips.InterestingNone, size, maxN,
				); err != nil {
					return nil, err
				}
				thumbnail = true
			}
		} else if stretch {
			if p.Width > 0 && p.Height > 0 {
				if img, err = v.newThumbnail(
					blob, p.Width, p.Height,
					vips.InterestingNone, vips.SizeForce, maxN,
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
					if img, err = v.newThumbnail(
						blob, p.Width, p.Height,
						interest, vips.SizeBoth, maxN,
					); err != nil {
						return nil, err
					}
				}
			} else if p.Width > 0 && p.Height == 0 {
				if img, err = v.newThumbnail(
					blob, p.Width, v.MaxHeight,
					vips.InterestingNone, vips.SizeBoth, maxN,
				); err != nil {
					return nil, err
				}
				thumbnail = true
			} else if p.Height > 0 && p.Width == 0 {
				if img, err = v.newThumbnail(
					blob, v.MaxWidth, p.Height,
					vips.InterestingNone, vips.SizeBoth, maxN,
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
				vips.InterestingNone, vips.SizeDown, maxN,
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
	if format == vips.ImageTypeUnknown {
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
			format = vips.ImageTypeJPEG
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
		if maxBytes > 0 && (quality > 10 || quality == 0) && format != vips.ImageTypePNG {
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

func getMeta(meta *vips.ImageMetadata) *imagor.Meta {
	format, ok := vips.ImageTypes[meta.Format]
	contentType, ok2 := imageMimeTypeMap[format]

	// govips returns "image/heif" for avif image content types
	if meta.Format == vips.ImageTypeAVIF {
		format = "avif"
		contentType = "image/avif"
	}

	if !ok || !ok2 {
		format = "jpeg"
		contentType = "image/jpeg"
	}

	pages := meta.Pages
	if pages < 1 {
		pages = 1
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

var imageTypeMap = map[string]vips.ImageType{
	"gif":    vips.ImageTypeGIF,
	"jpeg":   vips.ImageTypeJPEG,
	"jpg":    vips.ImageTypeJPEG,
	"magick": vips.ImageTypeMagick,
	"pdf":    vips.ImageTypePDF,
	"png":    vips.ImageTypePNG,
	"svg":    vips.ImageTypeSVG,
	"tiff":   vips.ImageTypeTIFF,
	"webp":   vips.ImageTypeWEBP,
	"heif":   vips.ImageTypeHEIF,
	"bmp":    vips.ImageTypeBMP,
	"avif":   vips.ImageTypeAVIF,
	"jp2":    vips.ImageTypeJP2K,
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

func (v *VipsProcessor) export(image *vips.ImageRef, format vips.ImageType, quality int) ([]byte, *vips.ImageMetadata, error) {
	switch format {
	case vips.ImageTypePNG:
		opts := vips.NewPngExportParams()
		return image.ExportPng(opts)
	case vips.ImageTypeWEBP:
		opts := vips.NewWebpExportParams()
		if quality > 0 {
			opts.Quality = quality
		}
		return image.ExportWebp(opts)
	case vips.ImageTypeTIFF:
		opts := vips.NewTiffExportParams()
		if quality > 0 {
			opts.Quality = quality
		}
		return image.ExportTiff(opts)
	case vips.ImageTypeGIF:
		opts := vips.NewGifExportParams()
		if quality > 0 {
			opts.Quality = quality
		}
		return image.ExportGIF(opts)
	case vips.ImageTypeAVIF:
		opts := vips.NewAvifExportParams()
		if quality > 0 {
			opts.Quality = quality
		}
		return image.ExportAvif(opts)
	case vips.ImageTypeJP2K:
		opts := vips.NewJp2kExportParams()
		if quality > 0 {
			opts.Quality = quality
		}
		return image.ExportJp2k(opts)
	default:
		opts := vips.NewJpegExportParams()
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
	if err == vips.ErrUnsupportedImageFormat {
		return imagor.ErrUnsupportedFormat
	}
	msg := err.Error()
	if strings.HasPrefix(msg, "VipsForeignLoad: buffer is not in a known format") {
		return imagor.ErrUnsupportedFormat
	}
	if idx := strings.Index(msg, "Stack:"); idx > -1 {
		msg = strings.TrimSpace(msg[:idx]) // neglect govips stacks from err msg
		return imagor.NewError(msg, 406)
	}
	return err
}
