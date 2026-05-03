package prometheusmetrics

import (
	"context"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

const DefaultNamespace = "imagor"

var httpRequestDuration *prometheus.HistogramVec

// PrometheusMetrics wraps the Service with additional http and app lifecycle handling
type PrometheusMetrics struct {
	http.Server

	Path      string
	Namespace string
	Logger    *zap.Logger
}

// New create new metrics PrometheusMetrics
func New(options ...Option) *PrometheusMetrics {
	s := &PrometheusMetrics{
		Path:      "/",
		Namespace: DefaultNamespace,
		Logger:    zap.NewNop(),
	}
	for _, option := range options {
		option(s)
	}
	if s.Path != "" && s.Path != "/" {
		mux := http.NewServeMux()
		mux.Handle(s.Path, promhttp.Handler())
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, s.Path, http.StatusPermanentRedirect)
		})
		s.Handler = mux
	} else {
		s.Handler = promhttp.Handler()
	}

	s.prepareMetrics()

	return s
}

func (s *PrometheusMetrics) prepareMetrics() {
	httpRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: s.Namespace,
			Name:      "http_request_duration_seconds",
			Help:      "A histogram of latencies for requests",
		},
		[]string{"code", "method"},
	)
}

// Startup prometheus metrics server
func (s *PrometheusMetrics) Startup(_ context.Context) error {
	if err := prometheus.Register(httpRequestDuration); err != nil {
		return err
	}

	go func() {
		if err := s.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.Logger.Fatal("prometheus listen", zap.Error(err))
		}
	}()
	s.Logger.Info("prometheus listen", zap.String("addr", s.Addr), zap.String("path", s.Path))
	return nil
}

// Handle prometheus http middleware handler
func (s *PrometheusMetrics) Handle(next http.Handler) http.Handler {
	return promhttp.InstrumentHandlerDuration(httpRequestDuration, next)
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

// WithNamespace with namespace option
func WithNamespace(namespace string) Option {
	return func(s *PrometheusMetrics) {
		s.Namespace = namespace
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
