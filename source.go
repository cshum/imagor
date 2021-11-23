package imagor

import (
	"errors"
	"net/http"
)

type Source interface {
	Match(r *http.Request, key string) bool
	Do(r *http.Request, key string) ([]byte, error)
}

func DoSources(r *http.Request, key string, sources []Source) (buf []byte, err error) {
	for _, source := range sources {
		if source.Match(r, key) {
			return source.Do(r, key)
		}
	}
	err = errors.New("unknown source")
	return
}
