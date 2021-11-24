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

type Loader interface {
	Match(r *http.Request, key string) bool
	Do(r *http.Request, key string) ([]byte, error)
}

type Processor interface {
	Do(ctx context.Context, buf []byte, params Params) ([]byte, error)
}

type Storage interface {
	Do(ctx context.Context, buf []byte, params Params) error
}

type Imagor struct {
	Logger     *zap.Logger
	Unsafe     bool
	Secret     string
	Loaders    []Loader
	Processors []Processor
	Storages   []Storage
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
	for _, processor := range o.Processors {
		b, e := processor.Do(ctx, buf, params)
		if e == nil {
			buf = b
			break
		} else {
			o.Logger.Error("process", zapParams, zap.Error(err))
		}
	}
	o.doStore(ctx, buf, params)
	return buf, nil
}

func (o *Imagor) doLoad(r *http.Request, image string) (buf []byte, err error) {
	for _, loader := range o.Loaders {
		if loader.Match(r, image) {
			if buf, err = loader.Do(r, image); err == nil {
				return
			}
		}
	}
	if err == nil {
		err = errors.New("unknown loader")
	}
	return
}

func (o *Imagor) doStore(ctx context.Context, buf []byte, params Params) {
	for _, storage := range o.Storages {
		var cancel func()
		sCtx := DetachContext(ctx)
		if o.Timeout > 0 {
			sCtx, cancel = context.WithTimeout(sCtx, o.Timeout)
		}
		go func(s Storage) {
			defer cancel()
			if err := s.Do(sCtx, buf, params); err != nil {
				o.Logger.Error("storage", zap.Any("params", params), zap.Error(err))
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
