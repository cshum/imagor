package server

import (
	"fmt"
	"github.com/cshum/imagor"
	"net/http"
)

func (s *Server) ImageHandler(w http.ResponseWriter, r *http.Request) {
	params, key, ok := imagor.ParseRequest(r)
	if !ok {
		s.NotFoundHandler(w, r)
		return
	}
	fmt.Println(params)
	w.Write([]byte(key))
	return
}

func (s *Server) NotFoundHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("not found"))
}
