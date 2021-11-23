package imagor

import (
	"errors"
	"fmt"
	"go.uber.org/zap"
	"net/http"
)

const (
	Version = "0.1.0"
)

type Imagor struct {
	Logger  *zap.Logger
	Sources []Source
}

func (o *Imagor) Do(r *http.Request) ([]byte, error) {
	params, err := ParseParams(r.URL.Path)
	if err != nil {
		return nil, err
	}
	buf, err := o.doSources(r, params.Image)
	if err != nil {
		return nil, err
	}
	return buf, nil
}

func (o *Imagor) doSources(r *http.Request, image string) (buf []byte, err error) {
	for _, source := range o.Sources {
		if source.Match(r, image) {
			return source.Do(r, image)
		}
	}
	err = errors.New("unknown source")
	return
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
