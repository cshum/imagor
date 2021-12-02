package server

import (
	"context"
	"github.com/cshum/imagor"
	"go.uber.org/zap"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
)

type Middleware func(http.Handler) http.Handler

type Server struct {
	http.Server
	Imagor     *imagor.Imagor
	Logger     *zap.Logger
	Debug      bool
	Address    string
	Port       int
	CertFile   string
	KeyFile    string
	PathPrefix string
}

func New(app *imagor.Imagor, options ...Option) *Server {
	s := &Server{}
	s.Imagor = app
	s.Port = 9000
	s.ReadTimeout = time.Second * 30
	s.MaxHeaderBytes = 1 << 20
	s.Logger = zap.NewNop()

	s.Handler = pathHandler(http.MethodGet, map[string]http.HandlerFunc{
		"/":            handleDefault,
		"/favicon.ico": handleFavicon,
		"/health":      handleHealth,
	})(s.Imagor)

	for _, option := range options {
		option(s)
	}
	if s.PathPrefix != "" {
		s.Handler = http.StripPrefix(s.PathPrefix, s.Handler)
	}
	s.Handler = s.panicHandler(s.Handler)
	return s
}

func (s *Server) Run() {
	s.Addr = s.Address + ":" + strconv.Itoa(s.Port)

	if err := s.Imagor.Startup(context.Background()); err != nil {
		s.Logger.Fatal("imagor start", zap.Error(err))
	}
	done := make(chan os.Signal, 1)
	signal.Notify(done, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := s.listenAndServe(); err != nil && err != http.ErrServerClosed {
			s.Logger.Fatal("listen", zap.Error(err))
		}
	}()

	s.Logger.Info("server start", zap.String("address", s.Address), zap.Int("port", s.Port))
	<-done

	// graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := s.Shutdown(ctx); err != nil {
		s.Logger.Error("server shutdown", zap.Error(err))
	}
	if err := s.Imagor.Shutdown(ctx); err != nil {
		s.Logger.Error("imagor shutdown", zap.Error(err))
	}
	s.Logger.Info("exit")
	return
}

func (s *Server) listenAndServe() error {
	if s.CertFile != "" && s.KeyFile != "" {
		return s.ListenAndServeTLS(s.CertFile, s.KeyFile)
	}
	return s.ListenAndServe()
}
