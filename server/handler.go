package server

import (
	"encoding/json"
	"fmt"
	"go.uber.org/zap"
	"html"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type errResp struct {
	Message string `json:"message,omitempty"`
	Code    int    `json:"status,omitempty"`
}

func handleOk(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	return
}

func (s *Server) panicHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rvr := recover(); rvr != nil {
				err, ok := rvr.(error)
				if !ok {
					err = fmt.Errorf("%v", rvr)
				}
				s.Logger.Error("panic", zap.Error(err))
				w.WriteHeader(http.StatusInternalServerError)
				writeJSON(w, r, errResp{
					Message: err.Error(),
					Code:    http.StatusInternalServerError,
				})
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func pathHandler(
	method string, handleFuncs map[string]http.HandlerFunc,
) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != method {
				next.ServeHTTP(w, r)
				return
			}
			if handle, ok := handleFuncs[r.URL.Path]; ok {
				handle(w, r)
				return
			}
			next.ServeHTTP(w, r)
			return
		})
	}
}

func stripQueryStringHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.RawQuery != "" {
			r.URL.RawQuery = ""
			http.Redirect(w, r, r.URL.String(), http.StatusTemporaryRedirect)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, r *http.Request, v interface{}) {
	buf, _ := json.Marshal(v)
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Length", strconv.Itoa(len(buf)))
	if r.Method != http.MethodHead {
		_, _ = w.Write(buf)
	}
	return
}

type statusRecorder struct {
	http.ResponseWriter
	Status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.Status = status
	r.ResponseWriter.WriteHeader(status)
}

func (s *Server) accessLogHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		wr := &statusRecorder{
			ResponseWriter: w,
			Status:         200,
		}
		next.ServeHTTP(wr, r)
		var (
			uri = sanitise(r.URL.RequestURI())
			ip  = sanitise(RealIP(r))
			ua  = sanitise(r.UserAgent())
		)
		s.Logger.Info("access",
			zap.Int("status", wr.Status),
			zap.String("method", r.Method),
			zap.String("uri", uri),
			zap.String("ip", ip),
			zap.String("user-agent", ua),
			zap.Duration("took", time.Since(start)),
		)
	})
}

var breaksCleaner = strings.NewReplacer(
	"\r\n", "",
	"\r", "",
	"\n", "",
	"\v", "",
	"\f", "",
	"\u0085", "",
	"\u2028", "",
	"\u2029", "",
)

func sanitise(s string) string {
	return html.EscapeString(breaksCleaner.Replace(s))
}
