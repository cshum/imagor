package server

import (
	"net/http"
	"time"

	"github.com/cshum/imagor/metrics/prometheusmetrics"
	"github.com/rs/cors"
	"go.uber.org/zap"
)

// Option Server option
type Option func(s *Server)

// WithAddr with server address with port option
func WithAddr(addr string) Option {
	return func(s *Server) {
		s.Addr = addr
	}
}

// WithAddress with server address option
func WithAddress(address string) Option {
	return func(s *Server) {
		s.Address = address
	}
}

// WithPort with port option
func WithPort(port int) Option {
	return func(s *Server) {
		s.Port = port
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

// WithMiddleware with HTTP middleware option
func WithMiddleware(middleware func(http.Handler) http.Handler) Option {
	return func(s *Server) {
		if middleware != nil {
			s.Handler = middleware(s.Handler)
		}
	}
}

// WithPathPrefix with path prefix option
func WithPathPrefix(prefix string) Option {
	return func(s *Server) {
		s.PathPrefix = prefix
	}
}

// WithCORS with CORS option
func WithCORS(enabled bool) Option {
	return func(s *Server) {
		if enabled {
			s.Handler = cors.Default().Handler(s.Handler)
		}
	}
}

// WithDebug with debug option
func WithDebug(debug bool) Option {
	return func(s *Server) {
		s.Debug = debug
	}
}

// WithStartupTimeout with server startup timeout option
func WithStartupTimeout(timeout time.Duration) Option {
	return func(s *Server) {
		if timeout > 0 {
			s.StartupTimeout = timeout
		}
	}
}

// WithShutdownTimeout with server shutdown timeout option
func WithShutdownTimeout(timeout time.Duration) Option {
	return func(s *Server) {
		if timeout > 0 {
			s.ShutdownTimeout = timeout
		}
	}
}

// WithStripQueryString with strip query string option
func WithStripQueryString(enabled bool) Option {
	return func(s *Server) {
		if enabled {
			s.Handler = stripQueryStringHandler(s.Handler)
		}
	}
}

// WithAccessLog with server access log option
func WithAccessLog(enabled bool) Option {
	return func(s *Server) {
		if enabled {
			s.Handler = s.accessLogHandler(s.Handler)
		}
	}
}

// WithAccessLog with server access log option
func WithPrometheusMetrics(pm *prometheusmetrics.PrometheusMetrics) Option {
	return func(s *Server) {
		s.PrometheusMetrics = pm
	}
}
