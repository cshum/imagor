package imagor

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/cshum/hybridcache"
	"go.uber.org/zap"
	"net/http"
	"time"
)

const Version = "0.1.0"

var ErrPass = errors.New("Imagor: pass")
var ErrNotFound = errors.New("Imagor: not found")

type LoadFunc func(string) ([]byte, error)

// Loader Load image from image source
type Loader interface {
	Load(r *http.Request, image string) ([]byte, error)
}

// Storage store image buffer
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
	Logger         *zap.Logger
	Cache          cache.Cache
	CacheTTL       time.Duration
	Unsafe         bool
	Secret         string
	Loaders        []Loader
	Storages       []Storage
	Processors     []Processor
	RequestTimeout time.Duration
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
	buf, err := o.Do(r)
	if err != nil {
		w.Write([]byte(fmt.Sprintf("%v", err)))
		return
	}
	w.Write(buf)
	return
}

func (o *Imagor) Do(r *http.Request) (buf []byte, err error) {
	params := ParseParams(r.URL.EscapedPath())
	var cancel func()
	ctx := r.Context()
	if o.RequestTimeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, o.RequestTimeout)
		defer cancel()
	}
	if !(o.Unsafe && params.Unsafe) && !params.Verify(o.Secret) {
		err = errors.New("hash mismatch")
		return
	}
	if buf, err = o.load(r, params.Image); err != nil {
		return
	}
	load := func(image string) ([]byte, error) {
		return o.load(r, image)
	}
	for _, processor := range o.Processors {
		b, meta, e := processor.Process(ctx, buf, params, load)
		if e == nil {
			buf = b
			if params.Meta {
				if b, e := json.Marshal(meta); e == nil {
					buf = b
				}
			}
			o.Logger.Debug("process", zap.Any("params", params), zap.Any("meta", meta))
			break
		} else if e == ErrPass {
			if len(b) > 0 {
				buf = b
			}
		} else {
			o.Logger.Error("process", zap.Any("params", params), zap.Error(e))
		}
	}
	return
}

func (o *Imagor) load(r *http.Request, image string) (buf []byte, err error) {
	return cache.NewFunc(o.Cache, o.RequestTimeout, o.CacheTTL, o.CacheTTL).
		DoBytes(r.Context(), image, func(ctx context.Context) (buf []byte, err error) {
			dr := r.WithContext(ctx)
			for _, loader := range o.Loaders {
				buf, err = loader.Load(dr, image)
				if err == nil {
					break
				}
				if err != nil && err != ErrPass && err != ErrNotFound {
					o.Logger.Error("load", zap.String("image", image), zap.Error(err))
				}
			}
			if err == nil {
				if len(o.Storages) > 0 {
					o.store(ctx, o.Storages, image, buf)
				}
			} else if err == ErrPass {
				err = ErrNotFound
			}
			return
		})
}

func (o *Imagor) store(
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
				o.Logger.Error("storage", zap.Any("image", image), zap.Error(err))
			}
		}(storage)
	}
}
