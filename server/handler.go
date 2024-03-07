package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"
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

func noopHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			next.ServeHTTP(w, r)
			return
		}
		if r.URL.Path == "/healthcheck" || r.URL.Path == "/favicon.ico" {
			handleOk(w, r)
			return
		}
		next.ServeHTTP(w, r)
		return
	})
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
		if r.URL.Path == "/healthcheck" || r.URL.Path == "/favicon.ico" {
			return // skip healthcheck routes
		}
		s.Logger.Info("access",
			zap.Int("status", wr.Status),
			zap.String("method", r.Method),
			zap.String("uri", r.URL.RequestURI()),
			zap.String("ip", RealIP(r)),
			zap.String("user-agent", r.UserAgent()),
			zap.Duration("took", time.Since(start)),
		)
	})
}

type serverErrorLogWriter struct {
	Logger *zap.Logger
}

func (s *serverErrorLogWriter) Write(p []byte) (int, error) {
	m := string(p)
	if strings.HasPrefix(m, "http: TLS handshake error") && strings.HasSuffix(m, ": EOF\n") {
		s.Logger.Debug("server", zap.String("log", m)) // https://github.com/golang/go/issues/26918
	} else if strings.HasPrefix(m, "http: URL query contains semicolon") {
		s.Logger.Debug("server", zap.String("log", m)) // https://github.com/golang/go/issues/25192
	} else {
		s.Logger.Warn("server", zap.String("log", m))
	}
	return len(p), nil
}

func newServerErrorLog(logger *zap.Logger) *log.Logger {
	return log.New(&serverErrorLogWriter{logger}, "", 0)
}
