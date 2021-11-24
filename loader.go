package imagor

import (
	"net/http"
)

type Loader interface {
	Match(r *http.Request, key string) bool
	Do(r *http.Request, key string) ([]byte, error)
}
