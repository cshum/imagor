package server

import (
	"context"
	"net/http"
	"os/signal"
	"reflect"
	"strconv"
	"syscall"
	"time"

	"github.com/cshum/imagor"
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

// Metrics represents metrics Startup and Shutdown lifecycle and Handle middleware
type Metrics interface {

	// Handle HTTP middleware metrics handler
	Handle(next http.Handler) http.Handler

	// Startup controls metrics startup
	Startup(ctx context.Context) error

	// Shutdown controls metrics shutdown
	Shutdown(ctx context.Context) error
}

// Server wraps the Service with additional http and app lifecycle handling
type Server struct {
	http.Server
	App             Service
	Address         string
	Port            int
	CertFile        string
	KeyFile         string
	PathPrefix      string
	SentryDsn       string
	StartupTimeout  time.Duration
	ShutdownTimeout time.Duration
	Logger          *zap.Logger
	Debug           bool
	Metrics         Metrics
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

	// build up middleware handlers in reverse order
	// Handler: application
	s.Handler = s.App

	// handle no-op routes /healthcheck, /favicon.ico
	s.Handler = noopHandler(s.Handler)

	for _, option := range options {
		option(s)
	}

	// Handler: request ID middleware (must be after options so it runs before accessLogHandler)
	s.Handler = requestIDHandler(s.Handler)

	// Handler: prefixes
	if s.PathPrefix != "" {
		s.Handler = http.StripPrefix(s.PathPrefix, s.Handler)
	}

	// Handler: recover from panics
	s.Handler = s.panicHandler(s.Handler)

	// Handler: observe metrics if enabled
	if !isNil(s.Metrics) {
		s.Handler = s.Metrics.Handle(s.Handler)
	}

	if s.Addr == "" {
		s.Addr = s.Address + ":" + strconv.Itoa(s.Port)
	}
	s.ErrorLog = newServerErrorLog(s.Logger)
	return s
}

// withContextLogger creates a logger that automatically includes request ID from context
func (s *Server) withContextLogger(ctx context.Context) *zap.Logger {
	if requestID := imagor.GetRequestID(ctx); requestID != "" {
		return s.Logger.With(zap.String("request_id", requestID))
	}
	return s.Logger
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

	if !isNil(s.Metrics) {
		if err := s.Metrics.Startup(ctx); err != nil {
			s.Logger.Fatal("metrics-startup", zap.Error(err))
		}
	}

	<-ctx.Done()

	s.shutdown(context.Background())
}

func isNil(c any) bool {
	return c == nil || (reflect.ValueOf(c).Kind() == reflect.Ptr && reflect.ValueOf(c).IsNil())
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
	if !isNil(s.Metrics) {
		if err := s.Metrics.Shutdown(ctx); err != nil {
			s.Logger.Error("metrics-shutdown", zap.Error(err))
		}
	}
	if err := s.Shutdown(ctx); err != nil {
		s.Logger.Error("server-shutdown", zap.Error(err))
	}
	if err := s.App.Shutdown(ctx); err != nil {
		s.Logger.Error("app-shutdown", zap.Error(err))
	}
}

func (s *Server) listenAndServe() error {
	if s.CertFile != "" && s.KeyFile != "" {
		return s.ListenAndServeTLS(s.CertFile, s.KeyFile)
	}
	return s.ListenAndServe()
}
