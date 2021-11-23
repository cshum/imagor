package imagor

import (
	"fmt"
	"net/http"
)

func (s *Server) ImageHandler(w http.ResponseWriter, r *http.Request) {
	params, key, ok := ParseRequest(r)
	if !ok {
		s.NotFoundHandler(w, r)
		return
	}
	for _, source := range s.Sources {
		if source.Match(r, key) {
			buf, err := source.Do(r, key)
			fmt.Println(params)
			if err != nil {
				w.Write([]byte("failed " + key))
				return
			}
			w.Write(buf)
			return
		}
	}
	w.Write([]byte("no available source"))
	return
}

func (s *Server) NotFoundHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("not found"))
}
