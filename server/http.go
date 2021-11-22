package server

import (
	"fmt"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
	"net/http"
)

type HTTP struct {
	Port   int
	Logger *zap.Logger
}

func (s *HTTP) Router() *chi.Mux {
	r := chi.NewRouter()
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hi"))
	})
	return r
}

func (s *HTTP) Run() error {
	s.Logger.Info("start", zap.Int("port", s.Port))
	return http.ListenAndServe(fmt.Sprintf(":%d", s.Port), s.Router())
}
