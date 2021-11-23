package imagor

import (
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
	buf, err := DoSources(r, params.Image, o.Sources)
	if err != nil {
		return nil, err
	}
	return buf, nil
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
