package server

import (
	"context"
	"net/http"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/cshum/imagor/metrics/prometheusmetrics"
	"go.uber.org/zap"
)

// Service is a http.Handler with Startup and Shutdown lifecycle
type Service interface {
	http.Handler

	// Startup controls app startup
	Startup(ctx context.Context) error

	// Shutdown controls app shutdown
	Shutdown(ctx context.Context) error
}

// Server wraps the Service with additional http and app lifecycle handling
type Server struct {
	http.Server
	App               Service
	Address           string
	Port              int
	CertFile          string
	KeyFile           string
	PathPrefix        string
	StartupTimeout    time.Duration
	ShutdownTimeout   time.Duration
	Logger            *zap.Logger
	Debug             bool
	PrometheusMetrics *prometheusmetrics.PrometheusMetrics
}

// New create new Server
func New(app Service, options ...Option) *Server {
	s := &Server{}
	s.App = app
	s.Port = 8000
	s.MaxHeaderBytes = 1 << 20
	s.StartupTimeout = time.Second * 10
	s.ShutdownTimeout = time.Second * 10
	s.Logger = zap.NewNop()
	s.Handler = pathHandler(http.MethodGet, map[string]http.HandlerFunc{
		"/favicon.ico": handleOk,
		"/healthcheck": handleOk,
	})(s.App)

	for _, option := range options {
		option(s)
	}
	if s.PathPrefix != "" {
		s.Handler = http.StripPrefix(s.PathPrefix, s.Handler)
	}
	s.Handler = s.panicHandler(s.Handler)
	if s.Addr == "" {
		s.Addr = s.Address + ":" + strconv.Itoa(s.Port)
	}
	s.ErrorLog = newServerErrorLog(s.Logger)
	return s
}

// Run server that terminates on SIGINT, SIGTERM signals
func (s *Server) Run() {
	ctx, cancel := signal.NotifyContext(
		context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()
	s.RunContext(ctx)
}

// RunContext run server with context
func (s *Server) RunContext(ctx context.Context) {
	s.startup(ctx)

	go func() {
		if err := s.listenAndServe(); err != nil && err != http.ErrServerClosed {
			s.Logger.Fatal("listen", zap.Error(err))
		}
	}()
	s.Logger.Info("listen", zap.String("addr", s.Addr))

	if s.PrometheusMetrics != nil {
		s.PrometheusMetrics.Run()
	}

	<-ctx.Done()

	s.shutdown(context.Background())
}

func (s *Server) startup(ctx context.Context) {
	ctx, cancel := context.WithTimeout(ctx, s.StartupTimeout)
	defer cancel()
	if err := s.App.Startup(ctx); err != nil {
		s.Logger.Fatal("app-startup", zap.Error(err))
	}
}

func (s *Server) shutdown(ctx context.Context) {
	ctx, cancel := context.WithTimeout(ctx, s.ShutdownTimeout)
	defer cancel()
	s.Logger.Info("shutdown")
	if err := s.Shutdown(ctx); err != nil {
		s.Logger.Error("server-shutdown", zap.Error(err))
	}
	if err := s.App.Shutdown(ctx); err != nil {
		s.Logger.Error("app-shutdown", zap.Error(err))
	}
	if s.PrometheusMetrics != nil {
		if err := s.PrometheusMetrics.Shutdown(ctx); err != nil {
			s.Logger.Error("prometheus-shutdown", zap.Error(err))
		}
	}
}

func (s *Server) listenAndServe() error {
	if s.CertFile != "" && s.KeyFile != "" {
		return s.ListenAndServeTLS(s.CertFile, s.KeyFile)
	}
	return s.ListenAndServe()
}
