package imagor

import (
	"context"
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
	Process(ctx context.Context, buf []byte, params Params) ([]byte, error)
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

func (o *Imagor) Do(r *http.Request) ([]byte, error) {
	var cancel func()
	ctx := r.Context()
	if o.Timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, o.Timeout)
		defer cancel()
	}
	params, err := ParseParams(r.URL.RawPath)
	if err != nil {
		return nil, err
	}
	if !o.Unsafe && !params.Verify(o.Secret) {
		return nil, errors.New("hash mismatch")
	}
	zapParams := zap.Any("params", params)
	o.Logger.Debug("access", zapParams)
	buf, err := o.doLoad(r, params.Image)
	if err != nil {
		return nil, err
	}
	if len(o.Storages) > 0 {
		o.doStore(ctx, o.Storages, params.Image, buf)
	}
	for _, processor := range o.Processors {
		b, e := processor.Process(ctx, buf, params)
		if e == nil {
			buf = b
			break
		} else {
			o.Logger.Error("process", zapParams, zap.Error(err))
		}
	}
	return buf, nil
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

func (o *Imagor) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	buf, err := o.Do(r)
	if err != nil {
		w.Write([]byte(fmt.Sprintf("%v", err)))
		return
	}
	w.Write(buf)
	return
}
