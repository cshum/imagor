package prometheusmetrics

import (
	"net/http"
	"strconv"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

// Server wraps the Service with additional http and app lifecycle handling
type Server struct {
	http.Server

	Host   string
	Port   int
	Path   string
	Logger *zap.Logger
}

// New create new metrics Server
func New(options ...Option) *Server {
	s := &Server{
		Port:   9000,
		Path:   "/metrics",
		Logger: zap.NewNop(),
	}
	for _, option := range options {
		option(s)
	}

	s.Addr = s.Host + ":" + strconv.Itoa(s.Port)

	mux := http.NewServeMux()
	mux.Handle(s.Path, promhttp.Handler())
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, s.Path, http.StatusPermanentRedirect)
	})
	s.Handler = mux

	return s
}

// Run http metrics server
func (s *Server) Run() {
	go func() {
		if err := s.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.Logger.Fatal("prometheus listen", zap.Error(err))
		}
	}()
	s.Logger.Info("prometheus listen", zap.String("addr", s.Addr), zap.String("path", s.Path))
}

// Option Server option
type Option func(s *Server)

// WithHost with server address option
func WithHost(address string) Option {
	return func(s *Server) {
		s.Host = address
	}
}

// WithPort with port option
func WithPort(port int) Option {
	return func(s *Server) {
		s.Port = port
	}
}

// WithPath with path option
func WithPath(path string) Option {
	return func(s *Server) {
		s.Path = path
	}
}

// WithLogger with logger option
func WithLogger(logger *zap.Logger) Option {
	return func(s *Server) {
		if logger != nil {
			s.Logger = logger
		}
	}
}
