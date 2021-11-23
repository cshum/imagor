package source

import (
	"context"
	"net/http"
)

type Source interface {
	Match(r *http.Request) (ok bool)
	Do(ctx context.Context) ([]byte, error)
}
