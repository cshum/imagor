package imagor

import (
	"net/http"
)

type Source interface {
	Match(r *http.Request, key string) bool
	Do(r *http.Request, key string) ([]byte, error)
}
