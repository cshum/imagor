package server

import (
	"fmt"
	"github.com/cshum/imagor/middleware"
	"github.com/gorilla/mux"
	"go.uber.org/zap"
	"net/http"
)

type HTTP struct {
	Port   int
	Logger *zap.Logger
}

func (s *HTTP) Handler() http.Handler {
	r := mux.NewRouter()
	r.Use(middleware.ImageHandler)
	r.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("not found"))
	})
	return r
}

func (s *HTTP) Run() error {
	s.Logger.Info("start", zap.Int("port", s.Port))
	return http.ListenAndServe(fmt.Sprintf(":%d", s.Port), s.Handler())
}
