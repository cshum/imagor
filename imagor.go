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

func (o *Imagor) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	params, err := Parse(r.URL)
	if err != nil {
		o.err(w, r, err)
		return
	}
	buf, err := DoSources(r, params.Image, o.Sources)
	if err != nil {
		o.err(w, r, err)
		return
	}
	fmt.Println(params)
	w.Write(buf)
	return
}

func (o *Imagor) err(w http.ResponseWriter, r *http.Request, err error) {
	w.Write([]byte(fmt.Sprintf("%v", err)))
}
