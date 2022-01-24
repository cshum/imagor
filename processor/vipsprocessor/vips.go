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
)

type FilterFunc func(ctx context.Context, img *vips.ImageRef, load imagor.LoadFunc, args ...string) (err error)

type FilterMap map[string]FilterFunc

type VipsProcessor struct {
	Filters        FilterMap
	DisableBlur    bool
	DisableFilters []string
	MaxFilterOps   int
	Logger         *zap.Logger
	Concurrency    int
	MaxCacheFiles  int
	MaxCacheMem    int
	MaxCacheSize   int
	MaxWidth       int
	MaxHeight      int
	Debug          bool
}

func New(options ...Option) *VipsProcessor {
	v := &VipsProcessor{
		MaxWidth:     9999,
		MaxHeight:    9999,
		MaxFilterOps: 10,
		Concurrency:  1,
		Logger:       zap.NewNop(),
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
		"strip_exif":       stripExif,
		"trim":             trimFilter,
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
		vips.Startup(&vips.Config{
			ReportLeaks:      true,
			MaxCacheFiles:    v.MaxCacheFiles,
			MaxCacheMem:      v.MaxCacheMem,
			MaxCacheSize:     v.MaxCacheSize,
			ConcurrencyLevel: v.Concurrency,
		})
	} else {
		vips.LoggingSettings(func(domain string, level vips.LogLevel, msg string) {
			v.Logger.Error(domain, zap.String("log", msg))
		}, vips.LogLevelError)
		vips.Startup(&vips.Config{
			MaxCacheFiles:    v.MaxCacheFiles,
			MaxCacheMem:      v.MaxCacheMem,
			MaxCacheSize:     v.MaxCacheSize,
			ConcurrencyLevel: v.Concurrency,
		})
	}
	return nil
}

func (v *VipsProcessor) Shutdown(_ context.Context) error {
	vips.Shutdown()
	return nil
}

func (v *VipsProcessor) newThumbnail(
	blob *imagor.Blob, width, height int, crop vips.Interesting, size vips.Size,
) (*vips.ImageRef, error) {
	if imagor.IsBlobEmpty(blob) {
		return nil, imagor.ErrNotFound
	}
	buf, err := blob.ReadAll()
	if err != nil {
		return nil, err
	}
	img, err := vips.NewThumbnailWithSizeFromBuffer(buf, width, height, crop, size)
	return img, wrapErr(err)
}

func (v *VipsProcessor) newImage(blob *imagor.Blob) (*vips.ImageRef, error) {
	if imagor.IsBlobEmpty(blob) {
		return nil, imagor.ErrNotFound
	}
	buf, err := blob.ReadAll()
	if err != nil {
		return nil, err
	}
	img, err := vips.NewImageFromBuffer(buf)
	return img, wrapErr(err)
}

func (v *VipsProcessor) Process(
	ctx context.Context, blob *imagor.Blob, p imagorpath.Params, load imagor.LoadFunc,
) (*imagor.Blob, error) {
	var (
		special   = false
		upscale   = true
		stretch   = p.Stretch
		thumbnail = false
		img       *vips.ImageRef
		err       error
	)
	ctx = WithInitImageRefs(ctx)
	defer CloseImageRefs(ctx)
	if p.Trim {
		special = true
	}
	if p.FitIn {
		upscale = false
	}
	for _, p := range p.Filters {
		switch p.Name {
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
				special = true
			}
			break
		case "trim":
			special = true
			break
		}
	}
	if !special && p.CropBottom == 0 && p.CropTop == 0 && p.CropLeft == 0 && p.CropRight == 0 {
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
					blob, w-p.HPadding*2, h-p.VPadding*2, vips.InterestingNone, size,
				); err != nil {
					return nil, err
				}
				thumbnail = true
			}
		} else if stretch {
			if p.Width > 0 && p.Height > 0 {
				if img, err = v.newThumbnail(
					blob, p.Width-p.HPadding*2, p.Height-p.VPadding*2,
					vips.InterestingNone, vips.SizeForce,
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
						blob, p.Width-p.HPadding*2, p.Height-p.VPadding*2, interest, vips.SizeBoth,
					); err != nil {
						return nil, err
					}
				}
			} else if p.Width > 0 && p.Height == 0 {
				if img, err = v.newThumbnail(
					blob, p.Width-p.HPadding*2, v.MaxHeight, vips.InterestingNone, vips.SizeBoth,
				); err != nil {
					return nil, err
				}
				thumbnail = true
			} else if p.Height > 0 && p.Width == 0 {
				if img, err = v.newThumbnail(
					blob, v.MaxWidth, p.Height-p.VPadding*2, vips.InterestingNone, vips.SizeBoth,
				); err != nil {
					return nil, err
				}
				thumbnail = true
			}
		}
	}
	if !thumbnail {
		if special {
			// special ops does not support create by thumbnail
			if img, err = v.newImage(blob); err != nil {
				return nil, err
			}
		} else {
			if img, err = v.newThumbnail(
				blob, v.MaxWidth, v.MaxHeight, vips.InterestingNone, vips.SizeDown,
			); err != nil {
				return nil, err
			}
		}
	}
	AddImageRef(ctx, img)
	var (
		format  = img.Format()
		quality int
	)
	for _, p := range p.Filters {
		switch p.Name {
		case "format":
			if typ, ok := imageTypeMap[p.Args]; ok {
				format = typ
			}
			break
		case "quality":
			quality, _ = strconv.Atoi(p.Args)
			break
		case "autojpg":
			format = vips.ImageTypeJPEG
			break
		}
	}
	if err := v.process(ctx, img, p, load, thumbnail, stretch, upscale); err != nil {
		return nil, wrapErr(err)
	}
	buf, meta, err := export(img, format, quality)
	if err != nil {
		return nil, wrapErr(err)
	}
	return imagor.NewBlobBytesWithMeta(buf, getMeta(meta)), nil
}

func getMeta(meta *vips.ImageMetadata) *imagor.Meta {
	format, ok := vips.ImageTypes[meta.Format]
	contentType, ok2 := imageMimeTypeMap[format]
	if !ok || !ok2 {
		format = "jpeg"
		contentType = "image/jpeg"
	}
	return &imagor.Meta{
		Format:      format,
		ContentType: contentType,
		Width:       meta.Width,
		Height:      meta.Height,
		Orientation: meta.Orientation,
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

func export(image *vips.ImageRef, format vips.ImageType, quality int) ([]byte, *vips.ImageMetadata, error) {
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
	case vips.ImageTypeHEIF:
		opts := vips.NewHeifExportParams()
		if quality > 0 {
			opts.Quality = quality
		}
		return image.ExportHeif(opts)
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
