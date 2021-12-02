package server

import (
	"encoding/json"
	"go.uber.org/zap"
	"net/http"
	"strconv"
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

	s.Handler = route(
		handleFavicon,
		handleHealth,
	)(handler)

	for _, option := range options {
		option(s)
	}
	return s
}

func resJSON(w http.ResponseWriter, v interface{}) {
	buf, _ := json.Marshal(v)
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Length", strconv.Itoa(len(buf)))
	w.Write(buf)
	return
}

func route(middlewares ...Middleware) Middleware {
	return func(handler http.Handler) http.Handler {
		ln := len(middlewares)
		for i := ln - 1; i >= 0; i-- {
			handler = middlewares[i](handler)
		}
		return handler
	}
}

func handleMethod(method, path string, handler http.HandlerFunc) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != method || path != r.URL.Path {
				next.ServeHTTP(w, r)
				return
			}
			handler(w, r)
		})
	}
}

func get(path string, handler http.HandlerFunc) Middleware {
	return handleMethod(http.MethodGet, path, handler)
}
