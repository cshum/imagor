package server

import (
	"github.com/rs/cors"
	"go.uber.org/zap"
	"time"
)

type Option func(s *Server)

func WithAddress(address string) Option {
	return func(s *Server) {
		s.Address = address
	}
}

func WithPort(port int) Option {
	return func(s *Server) {
		s.Port = port
	}
}

func WithLogger(logger *zap.Logger) Option {
	return func(s *Server) {
		s.Logger = logger
	}
}

func WithMiddleware(middleware Middleware) Option {
	return func(s *Server) {
		s.Handler = middleware(s.Handler)
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

func WithReadTimeout(timeout time.Duration) Option {
	return func(s *Server) {
		s.ReadTimeout = timeout
	}
}

func WithWriteTimeout(timeout time.Duration) Option {
	return func(s *Server) {
		s.WriteTimeout = timeout
	}
}
