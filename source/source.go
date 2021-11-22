package source

import (
	"net/http"
)

type Source interface {
	Match(*http.Request) bool
	Do(*http.Request) ([]byte, error)
}
