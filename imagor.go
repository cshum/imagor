package imagor

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/cshum/hybridcache"
	"go.uber.org/zap"
	"net/http"
	"strconv"
	"time"
)

const Version = "0.1.0"

type LoadFunc func(string) ([]byte, error)

// Loader Load image from image source
type Loader interface {
	Load(r *http.Request, image string) ([]byte, error)
}

// Storage save image buffer
type Storage interface {
	Save(ctx context.Context, image string, buf []byte) error
}

// Store both a Loader and Storage
type Store interface {
	Loader
	Storage
}

// Processor process image buffer
type Processor interface {
	Process(ctx context.Context, buf []byte, params Params, load LoadFunc) ([]byte, *Meta, error)
}

// Imagor image resize HTTP handler
type Imagor struct {
	// Cache is meant to be a short-lived buffer and call suppression.
	// For actual caching please place this under a reverse-proxy and CDN
	Cache          cache.Cache
	CacheTTL       time.Duration
	Unsafe         bool
	Secret         string
	Loaders        []Loader
	Storages       []Storage
	Processors     []Processor
	RequestTimeout time.Duration
	Logger         *zap.Logger
	Debug          bool
}

func New(options ...Option) *Imagor {
	o := &Imagor{
		Logger:         zap.NewNop(),
		Cache:          cache.NewMemory(1000, 1<<28, time.Minute),
		CacheTTL:       time.Minute,
		RequestTimeout: time.Second * 30,
	}
	for _, option := range options {
		option(o)
	}
	return o
}

func (o *Imagor) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	uri := r.URL.EscapedPath()
	params := ParseParams(uri)
	if o.Debug {
		o.Logger.Debug("params", zap.Any("params", params), zap.String("uri", uri))
	}
	buf, meta, err := o.Do(r, params)
	ln := len(buf)
	if meta != nil {
		if params.Meta {
			resJSON(w, meta)
			return
		} else {
			w.Header().Set("Content-Type", meta.ContentType)
		}
	} else if ln > 0 {
		w.Header().Set("Content-Type", http.DetectContentType(buf))
	}
	if err != nil {
		if e, ok := WrapError(err).(Error); ok {
			if e == ErrPass {
				// passed till the end means not found
				e = ErrNotFound
			}
			w.WriteHeader(e.Code)
		}
		if ln > 0 {
			w.Header().Set("Content-Length", strconv.Itoa(ln))
			w.Write(buf)
			return
		}
		resJSON(w, err)
		return
	}
	w.Header().Set("Content-Length", strconv.Itoa(ln))
	w.Write(buf)
	return
}

func (o *Imagor) Do(r *http.Request, params Params) (buf []byte, meta *Meta, err error) {
	var cancel func()
	ctx := r.Context()
	if o.RequestTimeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, o.RequestTimeout)
		defer cancel()
	}
	if !(o.Unsafe && params.Unsafe) && !params.Verify(o.Secret) {
		err = ErrHashMismatch
		if o.Debug {
			o.Logger.Debug("hash mismatch", zap.Any("params", params), zap.String("expected", Hash(params.Path, o.Secret)))
		}
		return
	}
	if buf, err = o.load(r, params.Image); err != nil {
		o.Logger.Info("load", zap.Any("params", params), zap.Error(err))
		return
	}
	load := func(image string) ([]byte, error) {
		return o.load(r, image)
	}
	for _, processor := range o.Processors {
		b, m, e := processor.Process(ctx, buf, params, load)
		if e == nil {
			buf = b
			meta = m
			if o.Debug {
				o.Logger.Debug("processed", zap.Any("params", params), zap.Any("meta", meta), zap.Int("size", len(buf)))
			}
			break
		} else {
			if e == ErrPass {
				if len(b) > 0 {
					// pass to next processor
					buf = b
				}
				if o.Debug {
					o.Logger.Debug("process", zap.Any("params", params), zap.Error(e))
				}
			} else {
				err = e
				o.Logger.Error("process", zap.Any("params", params), zap.Error(e))
			}
		}
	}
	return
}

func (o *Imagor) load(r *http.Request, image string) (buf []byte, err error) {
	buf, err = cache.NewFunc(o.Cache, o.RequestTimeout, o.CacheTTL, o.CacheTTL).
		DoBytes(r.Context(), image, func(ctx context.Context) (buf []byte, err error) {
			dr := r.WithContext(ctx)
			for _, loader := range o.Loaders {
				b, e := loader.Load(dr, image)
				if len(b) > 0 {
					buf = b
				}
				if e == nil {
					err = nil
					break
				}
				// should not log expected error as of now, as it has not reached the end
				if e != nil && e != ErrPass && e != ErrNotFound && !errors.Is(e, context.Canceled) {
					o.Logger.Error("load", zap.String("image", image), zap.Error(e))
				} else if o.Debug {
					o.Logger.Debug("load", zap.String("image", image), zap.Error(e))
				}
				err = e
			}
			if err == nil {
				if o.Debug {
					o.Logger.Debug("loaded", zap.String("image", image), zap.Int("size", len(buf)))
				}
				if len(o.Storages) > 0 {
					o.save(ctx, o.Storages, image, buf)
				}
			} else if !errors.Is(err, context.Canceled) {
				if err == ErrPass {
					err = ErrNotFound
				}
				// log non user-initiated error finally
				o.Logger.Error("load", zap.String("image", image), zap.Error(err))
			}
			return
		})
	err = WrapError(err)
	return
}

func (o *Imagor) save(
	ctx context.Context, storages []Storage, image string, buf []byte,
) {
	for _, storage := range storages {
		var cancel func()
		sCtx := DetachContext(ctx)
		if o.RequestTimeout > 0 {
			sCtx, cancel = context.WithTimeout(sCtx, o.RequestTimeout)
		}
		go func(s Storage) {
			defer cancel()
			if err := s.Save(sCtx, image, buf); err != nil {
				o.Logger.Error("save", zap.String("image", image), zap.Error(err))
			} else if o.Debug {
				o.Logger.Debug("saved", zap.String("image", image), zap.Int("size", len(buf)))
			}
		}(storage)
	}
}

func resJSON(w http.ResponseWriter, v interface{}) {
	buf, _ := json.Marshal(v)
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Length", strconv.Itoa(len(buf)))
	w.Write(buf)
	return
}
