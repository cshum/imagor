package server

import (
	"go.uber.org/zap"
	"net/http"
	"time"
)

type Middleware func(http.Handler) http.Handler

type Server struct {
	http.Server
	Logger *zap.Logger
}

func New(handler http.Handler, options ...Option) *Server {
	s := &Server{}
	s.Addr = ":9000"
	s.ReadTimeout = time.Second * 30
	s.MaxHeaderBytes = 1 << 20
	s.Logger = zap.NewNop()

	s.Handler = pathHandler(http.MethodGet, map[string]http.HandlerFunc{
		"/favicon.ico": handleFavicon,
		"/health":      handleHealth,
	})(handler)
	for _, option := range options {
		option(s)
	}
	s.Handler = s.panicHandler(s.Handler)

	return s
}
