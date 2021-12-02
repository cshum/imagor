package server

import "net/http"

type Middleware func(http.Handler) http.Handler

func route(middlewares ...Middleware) Middleware {
	return func(handler http.Handler) http.Handler {
		ln := len(middlewares)
		for i := ln - 1; i >= 0; i-- {
			handler = middlewares[i](handler)
		}
		return handler
	}
}

func handleGet(path string, handler http.HandlerFunc) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet || path != r.URL.Path {
				next.ServeHTTP(w, r)
				return
			}
			handler(w, r)
		})
	}
}
