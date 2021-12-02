package server

import (
	"fmt"
	"github.com/rs/cors"
	"go.uber.org/zap"
)

type Option func(s *Server)

func WithAddress(address string) Option {
	return func(s *Server) {
		s.Address = address
	}
}

func WithPort(port int) Option {
	return func(s *Server) {
		s.Addr = fmt.Sprintf(":%d", port)
	}
}

func WithLogger(logger *zap.Logger) Option {
	return func(s *Server) {
		s.Logger = logger
	}
}

func WithMiddleware(handler Middleware) Option {
	return func(s *Server) {
		s.Handler = handler(s.Handler)
	}
}

func WithPathPrefix(prefix string) Option {
	return func(s *Server) {
		s.PathPrefix = prefix
	}
}

func WithCORS() Option {
	return func(s *Server) {
		s.Handler = cors.Default().Handler(s.Handler)
	}
}
