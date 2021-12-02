package server

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"
)

func New(handler http.Handler, options ...Option) *http.Server {
	s := &http.Server{
		Addr: ":9000",
		Handler: route(
			handleGet("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				return
			}),
			handleGet("/health", func(w http.ResponseWriter, r *http.Request) {
				resJSON(w, GetHealthStats())
				return
			}),
		)(handler),
		ReadTimeout:    time.Second * 30,
		MaxHeaderBytes: 1 << 20,
	}
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
