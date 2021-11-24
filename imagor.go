package imagor

import (
	"encoding/json"
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
	Unsafe  bool
	Secret  string
	Loaders []Loader
}

func (o *Imagor) Do(r *http.Request) ([]byte, error) {
	params, err := ParseParams(r.URL.RawPath)
	if err != nil {
		return nil, err
	}
	if !o.Unsafe && !params.Verify(o.Secret) {
		return nil, errors.New("hash mismatch")
	}
	b, err := json.MarshalIndent(params, "", "  ")
	fmt.Println(string(b))
	buf, err := o.DoLoad(r, params.Image)
	if err != nil {
		return nil, err
	}
	return buf, nil
}

func (o *Imagor) DoLoad(r *http.Request, image string) (buf []byte, err error) {
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

func (o *Imagor) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	buf, err := o.Do(r)
	if err != nil {
		w.Write([]byte(fmt.Sprintf("%v", err)))
		return
	}
	w.Write(buf)
	return
}
