package vipsprocessor

import (
	"context"
	"github.com/cshum/imagor"
)

type Vips struct {
}

func (v *Vips) Process(ctx context.Context, buf []byte, params imagor.Params) ([]byte, error) {
	return nil, nil
}
