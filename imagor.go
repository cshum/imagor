package imagor

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"go.uber.org/zap"
	"net/http"
	"time"
)

const (
	Version = "0.1.0"
)

// Loader load image from image source
type Loader interface {
	Match(r *http.Request, image string) bool
	Load(r *http.Request, image string) ([]byte, error)
}

// Storage store image buffer
type Storage interface {
	Store(ctx context.Context, image string, buf []byte) error
}

// Store both a Loader and Storage
type Store interface {
	Loader
	Storage
}

// Processor process image buffer
type Processor interface {
	Process(ctx context.Context, buf []byte, params Params) ([]byte, *Meta, error)
}

// Imagor image resize HTTP handler
type Imagor struct {
	Logger     *zap.Logger
	Unsafe     bool
	Secret     string
	Loaders    []Loader
	Storages   []Storage
	Processors []Processor
	Timeout    time.Duration
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
	path := r.URL.RawPath
	if path == "" {
		path = r.URL.Path
	}
	params := ParseParams(path)
	var cancel func()
	ctx := r.Context()
	if o.Timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, o.Timeout)
		defer cancel()
	}
	if !(o.Unsafe && params.Unsafe) && !params.Verify(o.Secret) {
		err = errors.New("hash mismatch")
		return
	}
	if buf, err = o.doLoad(r, params.Image); err != nil {
		return
	}
	if len(o.Storages) > 0 {
		o.doStore(ctx, o.Storages, params.Image, buf)
	}
	for _, processor := range o.Processors {
		b, meta, e := processor.Process(ctx, buf, params)
		if e == nil {
			buf = b
			if params.Meta {
				if b, e := json.Marshal(meta); e == nil {
					buf = b
				}
			}
			o.Logger.Debug("process", zap.Any("params", params), zap.Any("meta", meta))
			break
		} else {
			o.Logger.Error("process", zap.Any("params", params), zap.Error(err))
		}
	}
	return
}

func (o *Imagor) doLoad(r *http.Request, image string) (buf []byte, err error) {
	for _, loader := range o.Loaders {
		if loader.Match(r, image) {
			if buf, err = loader.Load(r, image); err == nil {
				return
			}
		}
	}
	if err == nil {
		err = errors.New("unknown loader")
	}
	return
}

func (o *Imagor) doStore(ctx context.Context, storages []Storage, image string, buf []byte) {
	for _, storage := range storages {
		var cancel func()
		sCtx := DetachContext(ctx)
		if o.Timeout > 0 {
			sCtx, cancel = context.WithTimeout(sCtx, o.Timeout)
		}
		go func(s Storage) {
			defer cancel()
			if err := s.Store(sCtx, image, buf); err != nil {
				o.Logger.Error("storage", zap.Any("image", image), zap.Error(err))
			}
		}(storage)
	}
}
