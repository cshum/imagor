package server

import (
	"context"
	"github.com/cshum/imagor"
	"go.uber.org/zap"
	"net/http"
	"time"
)

type Middleware func(http.Handler) http.Handler

type Server struct {
	http.Server
	Imagor *imagor.Imagor
	Logger *zap.Logger
}

func New(handler *imagor.Imagor, options ...Option) *Server {
	s := &Server{}
	s.Addr = ":9000"
	s.ReadTimeout = time.Second * 30
	s.MaxHeaderBytes = 1 << 20
	s.Logger = zap.NewNop()
	s.Imagor = handler

	s.Handler = pathHandler(http.MethodGet, map[string]http.HandlerFunc{
		"/favicon.ico": handleFavicon,
		"/health":      handleHealth,
	})(s.Imagor)

	for _, option := range options {
		option(s)
	}

	s.Handler = s.panicHandler(s.Handler)
	return s
}

func (s *Server) Start(ctx context.Context) (err error) {
	if err = s.Imagor.Start(ctx); err != nil {
		return
	}
	return s.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) (err error) {
	if err = s.Imagor.Shutdown(ctx); err != nil {
		return
	}
	return
}
