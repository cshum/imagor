package vips

import (
	"context"
	"github.com/cshum/imagor"
	"github.com/cshum/imagor/vips/vipscontext"
	"go.uber.org/zap"
	"math"
	"runtime"
	"strings"
	"sync"
)

type FilterFunc func(ctx context.Context, img *Image, load imagor.LoadFunc, args ...string) (err error)

type FilterMap map[string]FilterFunc

var processorLock sync.RWMutex
var processorCount int

type Processor struct {
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

func NewProcessor(options ...Option) *Processor {
	v := &Processor{
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
		"label":            label,
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

func (v *Processor) Startup(_ context.Context) error {
	processorLock.Lock()
	defer processorLock.Unlock()
	processorCount++
	if processorCount > 1 {
		return nil
	}
	if v.Debug {
		SetLogging(func(domain string, level LogLevel, msg string) {
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
		SetLogging(func(domain string, level LogLevel, msg string) {
			v.Logger.Error(domain, zap.String("log", msg))
		}, LogLevelError)
	}
	Startup(&config{
		MaxCacheFiles:    v.MaxCacheFiles,
		MaxCacheMem:      v.MaxCacheMem,
		MaxCacheSize:     v.MaxCacheSize,
		ConcurrencyLevel: v.Concurrency,
	})
	return nil
}

func (v *Processor) Shutdown(_ context.Context) error {
	processorLock.Lock()
	defer processorLock.Unlock()
	if processorCount <= 0 {
		return nil
	}
	processorCount--
	if processorCount == 0 {
		Shutdown()
	}
	return nil
}

func newImageFromBlob(
	ctx context.Context, blob *imagor.Blob, params *ImportParams,
) (*Image, error) {
	if blob == nil || blob.IsEmpty() {
		return nil, imagor.ErrNotFound
	}
	if blob.BlobType() == imagor.BlobTypeMemory {
		buf, width, height, bands, _ := blob.Memory()
		return LoadImageFromMemory(buf, width, height, bands)
	} else if filepath := blob.FilePath(); filepath != "" {
		if err := blob.Err(); err != nil {
			return nil, err
		}
		return LoadImageFromFile(filepath, params)
	} else {
		reader, _, err := blob.NewReader()
		if err != nil {
			return nil, err
		}
		src := NewSource(reader)
		vipscontext.Defer(ctx, src.Close)
		return src.LoadImage(params)
	}
}

func newThumbnailFromBlob(
	ctx context.Context, blob *imagor.Blob,
	width, height int, crop Interesting, size Size, params *ImportParams,
) (*Image, error) {
	if blob == nil || blob.IsEmpty() {
		return nil, imagor.ErrNotFound
	}
	if filepath := blob.FilePath(); filepath != "" {
		if err := blob.Err(); err != nil {
			return nil, err
		}
		return LoadThumbnailFromFile(filepath, width, height, crop, size, params)
	} else {
		reader, _, err := blob.NewReader()
		if err != nil {
			return nil, err
		}
		src := NewSource(reader)
		vipscontext.Defer(ctx, src.Close)
		return src.LoadThumbnail(width, height, crop, size, params)
	}
}

func (v *Processor) NewThumbnail(
	ctx context.Context, blob *imagor.Blob, width, height int, crop Interesting, size Size, n int,
) (*Image, error) {
	var params *ImportParams
	var err error
	var img *Image
	if isBlobAnimated(blob, n) {
		params = NewImportParams()
		if n < -1 {
			params.NumPages.Set(-n)
		} else {
			params.NumPages.Set(-1)
		}
		if crop == InterestingNone || size == SizeForce {
			if img, err = v.CheckResolution(
				newThumbnailFromBlob(ctx, blob, width, height, crop, size, params),
			); err != nil {
				return nil, WrapErr(err)
			}
			if n > 1 && img.Pages() > n {
				// reload image to restrict frames loaded
				img.Close()
				return v.NewThumbnail(ctx, blob, width, height, crop, size, -n)
			}
		} else {
			if img, err = v.CheckResolution(newImageFromBlob(ctx, blob, params)); err != nil {
				return nil, WrapErr(err)
			}
			if n > 1 && img.Pages() > n {
				// reload image to restrict frames loaded
				img.Close()
				return v.NewThumbnail(ctx, blob, width, height, crop, size, -n)
			}
			if err = v.animatedThumbnailWithCrop(img, width, height, crop, size); err != nil {
				img.Close()
				return nil, WrapErr(err)
			}
		}
	} else {
		img, err = newThumbnailFromBlob(ctx, blob, width, height, crop, size, nil)
	}
	return v.CheckResolution(img, WrapErr(err))
}

func (v *Processor) NewImage(ctx context.Context, blob *imagor.Blob, n int) (*Image, error) {
	var params *ImportParams
	if isBlobAnimated(blob, n) {
		params = NewImportParams()
		if n < -1 {
			params.NumPages.Set(-n)
		} else {
			params.NumPages.Set(-1)
		}
		img, err := v.CheckResolution(newImageFromBlob(ctx, blob, params))
		if err != nil {
			return nil, WrapErr(err)
		}
		// reload image to restrict frames loaded
		if n > 1 && img.Pages() > n {
			img.Close()
			return v.NewImage(ctx, blob, -n)
		} else {
			return img, nil
		}
	} else {
		img, err := v.CheckResolution(newImageFromBlob(ctx, blob, params))
		if err != nil {
			return nil, WrapErr(err)
		}
		return img, nil
	}
}

func (v *Processor) Thumbnail(
	img *Image, width, height int, crop Interesting, size Size,
) error {
	if crop == InterestingNone || size == SizeForce || img.Height() == img.PageHeight() {
		return img.ThumbnailWithSize(width, height, crop, size)
	}
	return v.animatedThumbnailWithCrop(img, width, height, crop, size)
}

func (v *Processor) FocalThumbnail(img *Image, w, h int, fx, fy float64) (err error) {
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

func (v *Processor) animatedThumbnailWithCrop(
	img *Image, w, h int, crop Interesting, size Size,
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

func (v *Processor) CheckResolution(img *Image, err error) (*Image, error) {
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

func WrapErr(err error) error {
	if err == nil {
		return nil
	}
	if e, ok := err.(imagor.Error); ok {
		return e
	}
	msg := strings.TrimSpace(err.Error())
	if strings.HasPrefix(msg, "VipsForeignLoad:") &&
		strings.HasSuffix(msg, "is not in a known format") {
		return imagor.ErrUnsupportedFormat
	}
	return imagor.NewError(msg, 406)
}
