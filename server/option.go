package server

import (
	"fmt"
	"go.uber.org/zap"
)

type Option func(s *Server)

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
