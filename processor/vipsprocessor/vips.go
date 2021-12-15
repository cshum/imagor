package vipsprocessor

import (
	"context"
	"encoding/json"
	"github.com/cshum/govips/v2/vips"
	"github.com/cshum/imagor"
	"github.com/cshum/imagor/imagorpath"
	"go.uber.org/zap"
	"strconv"
)

type FilterFunc func(img *vips.ImageRef, load imagor.LoadFunc, args ...string) (err error)

type FilterMap map[string]FilterFunc

func (m FilterMap) MarshalJSON() ([]byte, error) {
	var names []string
	for name := range m {
		names = append(names, name)
	}
	return json.Marshal(names)
}

type VipsProcessor struct {
	Filters          FilterMap
	DisableBlur      bool
	DisableFilters   []string
	MaxFilterOps     int
	Logger           *zap.Logger
	ConcurrencyLevel int
	MaxCacheFiles    int
	MaxCacheMem      int
	MaxCacheSize     int
	Debug            bool
}

func New(options ...Option) *VipsProcessor {
	v := &VipsProcessor{
		Filters: FilterMap{
			"watermark":        watermark,
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
		},
		MaxFilterOps: 10,
		Logger:       zap.NewNop(),
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
			ConcurrencyLevel: v.ConcurrencyLevel,
		})
	} else {
		vips.LoggingSettings(func(domain string, level vips.LogLevel, msg string) {
			v.Logger.Error(domain, zap.String("log", msg))
		}, vips.LogLevelError)
		vips.Startup(&vips.Config{
			MaxCacheFiles:    v.MaxCacheFiles,
			MaxCacheMem:      v.MaxCacheMem,
			MaxCacheSize:     v.MaxCacheSize,
			ConcurrencyLevel: v.ConcurrencyLevel,
		})
	}
	return nil
}

func (v *VipsProcessor) Shutdown(_ context.Context) error {
	vips.Shutdown()
	return nil
}

func (v *VipsProcessor) Process(
	ctx context.Context, buf []byte, p imagorpath.Params, load imagor.LoadFunc,
) ([]byte, *imagor.Meta, error) {
	img, err := vips.NewImageFromBuffer(buf)
	if err != nil {
		if err == vips.ErrUnsupportedImageFormat {
			err = imagor.ErrUnsupportedFormat
		}
		return nil, nil, err
	}
	defer img.Close()
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
	if err := v.process(ctx, img, p, load); err != nil {
		return nil, nil, err
	}
	buf, meta, err := export(img, format, quality)
	if err != nil {
		return nil, nil, err
	}
	return buf, getMeta(meta), nil
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
