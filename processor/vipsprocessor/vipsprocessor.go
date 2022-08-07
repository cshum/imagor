package vipsprocessor

import (
	"context"
	"github.com/cshum/imagor"
	"go.uber.org/zap"
	"math"
	"runtime"
	"strings"
	"sync"
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

func LoadImageFromBlob(blob *imagor.Blob, params *ImportParams) (*ImageRef, error) {
	if blob == nil || blob.IsEmpty() {
		return nil, imagor.ErrNotFound
	}
	if filepath := blob.FilePath(); filepath != "" {
		if err := blob.Err(); err != nil {
			return nil, err
		}
		return LoadImageFromFile(filepath, params)
	} else {
		buf, err := blob.ReadAll()
		if err != nil {
			return nil, err
		}
		return LoadImageFromBuffer(buf, params)
	}
}

func LoadThumbnailFromBlob(
	blob *imagor.Blob, width, height int, crop Interesting, size Size, params *ImportParams,
) (*ImageRef, error) {
	if blob == nil || blob.IsEmpty() {
		return nil, imagor.ErrNotFound
	}
	if filepath := blob.FilePath(); filepath != "" {
		if err := blob.Err(); err != nil {
			return nil, err
		}
		return LoadThumbnailFromFile(filepath, width, height, crop, size, params)
	} else {
		buf, err := blob.ReadAll()
		if err != nil {
			return nil, err
		}
		return LoadThumbnailFromBuffer(buf, width, height, crop, size, params)
	}
}

func (v *VipsProcessor) newThumbnail(
	blob *imagor.Blob, width, height int, crop Interesting, size Size, n int,
) (*ImageRef, error) {
	var params *ImportParams
	var err error
	var img *ImageRef
	if isBlobAnimated(blob, n) {
		params = NewImportParams()
		if n < -1 {
			params.NumPages.Set(-n)
		} else {
			params.NumPages.Set(-1)
		}
		if crop == InterestingNone || size == SizeForce {
			if img, err = v.checkResolution(
				LoadThumbnailFromBlob(blob, width, height, crop, size, params),
			); err != nil {
				return nil, wrapErr(err)
			}
			if n > 1 && img.Pages() > n {
				// reload image to restrict frames loaded
				img.Close()
				return v.newThumbnail(blob, width, height, crop, size, -n)
			}
		} else {
			if img, err = v.checkResolution(LoadImageFromBlob(blob, params)); err != nil {
				return nil, wrapErr(err)
			}
			if n > 1 && img.Pages() > n {
				// reload image to restrict frames loaded
				img.Close()
				return v.newThumbnail(blob, width, height, crop, size, -n)
			}
			if err = v.animatedThumbnailWithCrop(img, width, height, crop, size); err != nil {
				img.Close()
				return nil, wrapErr(err)
			}
		}
	} else if blob.BlobType() == imagor.BlobTypePNG {
		return v.newThumbnailPNG(blob, width, height, crop, size)
	} else {
		img, err = LoadThumbnailFromBlob(blob, width, height, crop, size, nil)
	}
	return v.checkResolution(img, wrapErr(err))
}

func (v *VipsProcessor) newThumbnailPNG(
	blob *imagor.Blob, width, height int, crop Interesting, size Size,
) (img *ImageRef, err error) {
	if img, err = v.checkResolution(LoadImageFromBlob(blob, nil)); err != nil {
		return
	}
	if err = img.ThumbnailWithSize(width, height, crop, size); err != nil {
		img.Close()
		return
	}
	return v.checkResolution(img, wrapErr(err))
}

func (v *VipsProcessor) newImage(blob *imagor.Blob, n int) (*ImageRef, error) {
	var params *ImportParams
	if isBlobAnimated(blob, n) {
		params = NewImportParams()
		if n < -1 {
			params.NumPages.Set(-n)
		} else {
			params.NumPages.Set(-1)
		}
		img, err := v.checkResolution(LoadImageFromBlob(blob, params))
		if err != nil {
			return nil, wrapErr(err)
		}
		// reload image to restrict frames loaded
		if n > 1 && img.Pages() > n {
			img.Close()
			return v.newImage(blob, -n)
		} else {
			return img, nil
		}
	} else {
		img, err := v.checkResolution(LoadImageFromBlob(blob, params))
		if err != nil {
			return nil, wrapErr(err)
		}
		return img, nil
	}
}

func (v *VipsProcessor) thumbnail(
	img *ImageRef, width, height int, crop Interesting, size Size,
) error {
	if crop == InterestingNone || size == SizeForce || img.Height() == img.PageHeight() {
		return img.ThumbnailWithSize(width, height, crop, size)
	}
	return v.animatedThumbnailWithCrop(img, width, height, crop, size)
}

func (v *VipsProcessor) focalThumbnail(img *ImageRef, w, h int, fx, fy float64) (err error) {
	if float64(w)/float64(h) > float64(img.Width())/float64(img.PageHeight()) {
		if err = img.Thumbnail(w, v.MaxHeight, InterestingNone); err != nil {
			return
		}
	} else {
		if err = img.Thumbnail(v.MaxWidth, h, InterestingNone); err != nil {
			return
		}
	}
	var top, left float64
	left = float64(img.Width())*fx - float64(w)/2
	top = float64(img.PageHeight())*fy - float64(h)/2
	left = math.Max(0, math.Min(left, float64(img.Width()-w)))
	top = math.Max(0, math.Min(top, float64(img.PageHeight()-h)))
	return img.ExtractArea(int(left), int(top), w, h)
}

func (v *VipsProcessor) animatedThumbnailWithCrop(
	img *ImageRef, w, h int, crop Interesting, size Size,
) (err error) {
	if size == SizeDown && img.Width() < w && img.PageHeight() < h {
		return
	}
	var top, left int
	if float64(w)/float64(h) > float64(img.Width())/float64(img.PageHeight()) {
		if err = img.ThumbnailWithSize(w, v.MaxHeight, InterestingNone, size); err != nil {
			return
		}
	} else {
		if err = img.ThumbnailWithSize(v.MaxWidth, h, InterestingNone, size); err != nil {
			return
		}
	}
	if crop == InterestingHigh {
		left = img.Width() - w
		top = img.PageHeight() - h
	} else if crop == InterestingCentre || crop == InterestingAttention {
		left = (img.Width() - w) / 2
		top = (img.PageHeight() - h) / 2
	}
	return img.ExtractArea(left, top, w, h)
}

func (v *VipsProcessor) checkResolution(img *ImageRef, err error) (*ImageRef, error) {
	if err != nil || img == nil {
		return img, err
	}
	if img.Width() > v.MaxWidth || img.PageHeight() > v.MaxHeight ||
		(img.Width()*img.PageHeight()) > v.MaxResolution {
		img.Close()
		return nil, imagor.ErrMaxResolutionExceeded
	}
	return img, nil
}

func isBlobAnimated(blob *imagor.Blob, n int) bool {
	return blob != nil && blob.SupportsAnimation() && n != 1 && n != 0
}

func wrapErr(err error) error {
	if err == nil {
		return nil
	}
	if e, ok := err.(imagor.Error); ok {
		return e
	}
	msg := strings.TrimSpace(err.Error())
	if strings.HasPrefix(msg, "VipsForeignLoad: buffer is not in a known format") {
		return imagor.ErrUnsupportedFormat
	}
	return imagor.NewError(msg, 406)
}
