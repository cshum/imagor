package imagor

import (
	"fmt"
	"go.uber.org/zap"
	"net/http"
)

type Server struct {
	Port    int
	Logger  *zap.Logger
	Sources []Source
}

func (s *Server) Handler() http.Handler {
	return http.HandlerFunc(s.ImageHandler)
}

func (s *Server) Run() error {
	s.Logger.Info("start", zap.Int("port", s.Port))
	return http.ListenAndServe(fmt.Sprintf(":%d", s.Port), s.Handler())
}
