package server

import "net/http"

var handleFavicon = get("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	return
})

var handleHealth = get("/health", func(w http.ResponseWriter, r *http.Request) {
	resJSON(w, GetHealthStats())
	return
})
