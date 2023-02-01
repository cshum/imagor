package prometheusmetrics

import (
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
	"net/http"
)

// PrometheusMetrics wraps the Service with additional http and app lifecycle handling
type PrometheusMetrics struct {
	http.Server

	Namespace string
	Logger    *zap.Logger
}

// New create new metrics PrometheusMetrics
func New(options ...Option) *PrometheusMetrics {
	s := &PrometheusMetrics{
		Logger: zap.NewNop(),
	}
	for _, option := range options {
		option(s)
	}

	s.Handler = promhttp.Handler()

	return s
}

// Run http metrics server
func (s *PrometheusMetrics) Run() {
	go func() {
		if err := s.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.Logger.Fatal("prometheus listen", zap.Error(err))
		}
	}()
	s.Logger.Info("prometheus listen", zap.String("addr", s.Addr), zap.String("path", s.Namespace))
}

// Option PrometheusMetrics option
type Option func(s *PrometheusMetrics)

// WithAddr with server and port option
func WithAddr(addr string) Option {
	return func(s *PrometheusMetrics) {
		s.Addr = addr
	}
}

// WithNamespace with path option
func WithNamespace(path string) Option {
	return func(s *PrometheusMetrics) {
		s.Namespace = path
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
