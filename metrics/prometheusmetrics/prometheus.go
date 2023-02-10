package prometheusmetrics

import (
	"context"
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

// PrometheusMetrics wraps the Service with additional http and app lifecycle handling
type PrometheusMetrics struct {
	http.Server

	Path   string
	Logger *zap.Logger
}

// New create new metrics PrometheusMetrics
func New(options ...Option) *PrometheusMetrics {
	s := &PrometheusMetrics{
		Logger: zap.NewNop(),
	}
	for _, option := range options {
		option(s)
	}
	if s.Path != "" {
		mux := http.NewServeMux()
		mux.Handle(s.Path, promhttp.Handler())
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, s.Path, http.StatusPermanentRedirect)
		})
		s.Handler = mux
	} else {
		s.Handler = promhttp.Handler()
	}
	return s
}

// Startup prometheus metrics server
func (s *PrometheusMetrics) Startup(_ context.Context) error {
	go func() {
		if err := s.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.Logger.Fatal("prometheus listen", zap.Error(err))
		}
	}()
	s.Logger.Info("prometheus listen", zap.String("addr", s.Addr), zap.String("path", s.Path))
	return nil
}

// Option PrometheusMetrics option
type Option func(s *PrometheusMetrics)

// WithAddr with server and port option
func WithAddr(addr string) Option {
	return func(s *PrometheusMetrics) {
		s.Addr = addr
	}
}

// WithPath with path option
func WithPath(path string) Option {
	return func(s *PrometheusMetrics) {
		s.Path = path
	}
}

// WithLogger with logger option
func WithLogger(logger *zap.Logger) Option {
	return func(s *PrometheusMetrics) {
		if logger != nil {
			s.Logger = logger
		}
	}
}
