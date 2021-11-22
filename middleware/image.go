package middleware

import (
	"fmt"
	"github.com/cshum/imagor"
	"net/http"
)

func ImageHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		params, key, ok := imagor.ParseRequest(r)
		if !ok {
			next.ServeHTTP(w, r) // not found
			return
		}
		fmt.Println(params)
		w.Write([]byte(key))
		return
	})
}
