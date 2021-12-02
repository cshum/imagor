package server

import (
	"context"
	"github.com/cshum/imagor"
	"go.uber.org/zap"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type Middleware func(http.Handler) http.Handler

type Server struct {
	http.Server
	Imagor *imagor.Imagor
	Logger *zap.Logger
}

func New(process *imagor.Imagor, options ...Option) *Server {
	s := &Server{}
	s.Addr = ":9000"
	s.ReadTimeout = time.Second * 30
	s.MaxHeaderBytes = 1 << 20
	s.Logger = zap.NewNop()
	s.Imagor = process

	s.Handler = pathHandler(http.MethodGet, map[string]http.HandlerFunc{
		"/favicon.ico": handleFavicon,
		"/health":      handleHealth,
	})(s.Imagor)

	for _, option := range options {
		option(s)
	}

	s.Handler = s.panicHandler(s.Handler)
	return s
}

func (s *Server) Run() (err error) {
	if err = s.Imagor.Start(context.Background()); err != nil {
		return
	}
	done := make(chan os.Signal, 1)
	signal.Notify(done, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err = s.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.Logger.Error("listen", zap.Error(err))
		}
	}()

	s.Logger.Info("start", zap.String("addr", s.Addr))
	<-done

	// graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err = s.Imagor.Shutdown(ctx); err != nil {
		s.Logger.Error("shutdown", zap.Error(err))
		return
	}
	s.Logger.Info("shutdown")
	return
}
