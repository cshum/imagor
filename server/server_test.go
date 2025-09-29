package server

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/cshum/imagor"
	"github.com/cshum/imagor/imagorpath"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

type testProcessor struct {
	StartupCnt  int
	ShutdownCnt int
}

func (app *testProcessor) Process(ctx context.Context, blob *imagor.Blob, p imagorpath.Params, load imagor.LoadFunc) (*imagor.Blob, error) {
	return nil, nil
}

func (app *testProcessor) Startup(ctx context.Context) error {
	app.StartupCnt++
	return nil
}

func (app *testProcessor) Shutdown(ctx context.Context) error {
	app.ShutdownCnt++
	return nil
}

type slowTestProcessor struct {
	StartupCnt  int
	ShutdownCnt int
}

func (app *slowTestProcessor) Process(ctx context.Context, blob *imagor.Blob, p imagorpath.Params, load imagor.LoadFunc) (*imagor.Blob, error) {
	return nil, nil
}

func (app *slowTestProcessor) Startup(ctx context.Context) error {
	app.StartupCnt++
	select {
	case <-time.After(100 * time.Millisecond):
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (app *slowTestProcessor) Shutdown(ctx context.Context) error {
	app.ShutdownCnt++
	return nil
}

type loaderFunc func(r *http.Request, image string) (blob *imagor.Blob, err error)

func (f loaderFunc) Get(r *http.Request, image string) (*imagor.Blob, error) {
	return f(r, image)
}

func TestServer_Run(t *testing.T) {
	ctx, done := context.WithCancel(context.Background())
	processor := &testProcessor{}
	app := imagor.New(imagor.WithProcessors(processor))
	s := New(app,
		WithDebug(true),
		WithAddr(":0"),
		WithStartupTimeout(time.Millisecond),
		WithShutdownTimeout(time.Millisecond),
		WithMetrics(nil),
		WithLogger(zap.NewExample()))
	go func() {
		time.Sleep(time.Millisecond)
		assert.Equal(t, 1, processor.StartupCnt)
		assert.Equal(t, 0, processor.ShutdownCnt)
		done()
	}()
	s.RunContext(ctx)
	assert.Equal(t, 1, processor.ShutdownCnt)
}

func TestServer(t *testing.T) {
	s := New(
		imagor.New(
			imagor.WithUnsafe(true),
			imagor.WithLoaders(loaderFunc(func(r *http.Request, image string) (*imagor.Blob, error) {
				return imagor.NewBlobFromBytes([]byte("foo")), nil
			})),
		),
		WithAccessLog(true),
		WithMiddleware(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("X-Foo", "Bar")
				if strings.Contains(r.URL.String(), "boom") {
					panic("booooom")
				}
				next.ServeHTTP(w, r)
			})
		}),
		WithCORS(true),
	)

	w := httptest.NewRecorder()
	s.Handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "https://example.com/favicon.ico", nil))
	assert.Equal(t, 200, w.Code)
	assert.NotEmpty(t, w.Header().Get("Vary"))
	assert.Equal(t, "Bar", w.Header().Get("X-Foo"))

	w = httptest.NewRecorder()
	s.Handler.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "https://example.com/favicon.ico", nil))
	assert.Equal(t, 405, w.Code)

	w = httptest.NewRecorder()
	s.Handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "https://example.com/healthcheck", nil))
	assert.Equal(t, 200, w.Code)
	assert.NotEmpty(t, w.Header().Get("Vary"))
	assert.Equal(t, "Bar", w.Header().Get("X-Foo"))

	w = httptest.NewRecorder()
	s.Handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "https://example.com/unsafe/foo.jpg", nil))
	assert.Equal(t, 200, w.Code)
	assert.NotEmpty(t, w.Header().Get("Vary"))
	assert.Equal(t, "Bar", w.Header().Get("X-Foo"))

	w = httptest.NewRecorder()
	s.Handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "https://example.com/unsafe/bar.jpg?boom", nil))
	assert.Equal(t, 500, w.Code)
	assert.NotEmpty(t, w.Header().Get("Vary"))
	assert.Equal(t, "Bar", w.Header().Get("X-Foo"))
	assert.Equal(t, `{"message":"booooom","status":500}`, w.Body.String())
}

func TestServerErrorLog(t *testing.T) {
	expectLogged := []string{"panic", "server", "server"}
	var logged []string
	logger := zap.NewExample(zap.Hooks(func(entry zapcore.Entry) error {
		logged = append(logged, entry.Message)
		return nil
	}))
	s := New(
		imagor.New(
			imagor.WithUnsafe(true),
			imagor.WithLoaders(loaderFunc(func(r *http.Request, image string) (*imagor.Blob, error) {
				return imagor.NewBlobFromBytes([]byte("foo")), nil
			})),
		),
		WithAccessLog(true),
		WithDebug(true),
		WithLogger(logger),
		WithMiddleware(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("X-Foo", "Bar")
				if strings.Contains(r.URL.String(), "boom") {
					panic("booooom")
				}
				next.ServeHTTP(w, r)
			})
		}),
		WithCORS(true),
	)

	ts := httptest.NewServer(s.Handler)
	ts.Config = &s.Server
	defer ts.Close()

	w, err := http.Get(ts.URL + "/unsafe/bar.jpg?boom")
	assert.NoError(t, err)
	assert.Equal(t, 500, w.StatusCode)
	assert.NotEmpty(t, w.Header.Get("Vary"))
	assert.Equal(t, "Bar", w.Header.Get("X-Foo"))
	resp, err := io.ReadAll(w.Body)
	assert.NoError(t, err)
	assert.Equal(t, `{"message":"booooom","status":500}`, string(resp))

	_, err = ts.Config.ErrorLog.Writer().Write([]byte("http: TLS handshake error from 172.16.0.3:42672: EOF"))
	assert.NoError(t, err)
	_, err = ts.Config.ErrorLog.Writer().Write([]byte("foobar"))
	assert.NoError(t, err)

	assert.Equal(t, expectLogged, logged)
}

func TestWithStripQueryString(t *testing.T) {
	s := New(imagor.New(),
		WithAddr("https://example.com:1667"), WithPort(1234))
	assert.Equal(t, "https://example.com:1667", s.Addr)

	w := httptest.NewRecorder()
	s.Handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "https://example.com/?a=1&b=2", nil))
	assert.Equal(t, http.StatusOK, w.Code)

	s = New(imagor.New(),
		WithStripQueryString(true), WithAddress("https://foo.com"), WithPort(1234))
	assert.Equal(t, "https://foo.com:1234", s.Addr)

	w = httptest.NewRecorder()
	s.Handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "https://example.com/?a=1&b=2", nil))
	assert.Equal(t, http.StatusTemporaryRedirect, w.Code)
	assert.Equal(t, "https://example.com/", w.Header().Get("Location"))

	w = httptest.NewRecorder()
	s.Handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "https://example.com/", nil))
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestWithPathPrefix(t *testing.T) {
	s := New(imagor.New())

	w := httptest.NewRecorder()
	s.Handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "https://example.com/", nil))
	assert.Equal(t, http.StatusOK, w.Code)

	s = New(imagor.New(), WithPathPrefix("/imagor"))

	w = httptest.NewRecorder()
	s.Handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "https://example.com/", nil))
	assert.Equal(t, http.StatusNotFound, w.Code)

	w = httptest.NewRecorder()
	s.Handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "https://example.com/imagor", nil))
	assert.Equal(t, http.StatusOK, w.Code)
	fmt.Println(w.Body.String())
}

func TestWithSentry(t *testing.T) {
	s := New(imagor.New(), WithSentry("https://12345@sentry.com/123"))
	assert.Equal(t, "https://12345@sentry.com/123", s.SentryDsn)
}

// Test the isNil utility function
func TestIsNil(t *testing.T) {
	t.Run("nil interface", func(t *testing.T) {
		var i interface{}
		assert.True(t, isNil(i))
	})

	t.Run("nil pointer", func(t *testing.T) {
		var p *testProcessor
		assert.True(t, isNil(p))
	})

	t.Run("nil slice", func(t *testing.T) {
		var s []string
		assert.False(t, isNil(s))
	})

	t.Run("nil map", func(t *testing.T) {
		var m map[string]string
		assert.False(t, isNil(m))
	})

	t.Run("nil channel", func(t *testing.T) {
		var c chan string
		assert.False(t, isNil(c))
	})

	t.Run("nil function", func(t *testing.T) {
		var f func()
		assert.False(t, isNil(f))
	})

	t.Run("non-nil values", func(t *testing.T) {
		assert.False(t, isNil("string"))
		assert.False(t, isNil(42))
		assert.False(t, isNil([]string{"test"}))
		assert.False(t, isNil(map[string]string{"key": "value"}))
		assert.False(t, isNil(&testProcessor{}))
	})

	t.Run("nil interface with non-nil pointer", func(t *testing.T) {
		var i interface{} = (*testProcessor)(nil)
		assert.True(t, isNil(i))
	})

	t.Run("non-nil interface with nil value", func(t *testing.T) {
		var p *testProcessor
		var i interface{} = p
		assert.True(t, isNil(i))
	})
}

// Test startup method directly
func TestServerStartup(t *testing.T) {
	t.Run("successful startup", func(t *testing.T) {
		processor := &testProcessor{}
		app := imagor.New(imagor.WithProcessors(processor))
		s := New(app, WithStartupTimeout(time.Second))

		ctx := context.Background()
		s.startup(ctx)

		assert.Equal(t, 1, processor.StartupCnt)
	})
}

func TestServerShutdown(t *testing.T) {
	t.Run("successful shutdown", func(t *testing.T) {
		processor := &testProcessor{}
		app := imagor.New(imagor.WithProcessors(processor))
		s := New(app, WithShutdownTimeout(time.Second))

		ctx := context.Background()
		s.shutdown(ctx)

		assert.Equal(t, 1, processor.ShutdownCnt)
	})

	t.Run("shutdown with metrics", func(t *testing.T) {
		processor := &testProcessor{}
		app := imagor.New(imagor.WithProcessors(processor))

		mockMetrics := &testMetrics{}
		s := New(app, WithMetrics(mockMetrics), WithShutdownTimeout(time.Second))

		ctx := context.Background()
		s.shutdown(ctx)

		assert.Equal(t, 1, processor.ShutdownCnt)
		assert.Equal(t, 1, mockMetrics.ShutdownCnt)
	})
}

// Test listenAndServe method
func TestServerListenAndServe(t *testing.T) {
	t.Run("HTTP server", func(t *testing.T) {
		processor := &testProcessor{}
		app := imagor.New(imagor.WithProcessors(processor))
		s := New(app, WithAddr(":0"))

		// Test that it would start HTTP server (not TLS)
		assert.Empty(t, s.CertFile)
		assert.Empty(t, s.KeyFile)
	})

	t.Run("HTTPS server", func(t *testing.T) {
		processor := &testProcessor{}
		app := imagor.New(imagor.WithProcessors(processor))
		s := New(app, WithAddr(":0"))
		s.CertFile = "cert.pem"
		s.KeyFile = "key.pem"

		// Test that it would start HTTPS server
		assert.NotEmpty(t, s.CertFile)
		assert.NotEmpty(t, s.KeyFile)
	})
}

// Test server options
func TestServerOptions(t *testing.T) {
	processor := &testProcessor{}
	app := imagor.New(imagor.WithProcessors(processor))

	t.Run("WithAddr", func(t *testing.T) {
		s := New(app, WithAddr("localhost:8080"))
		assert.Equal(t, "localhost:8080", s.Addr)
	})

	t.Run("WithAddress and WithPort", func(t *testing.T) {
		s := New(app, WithAddress("localhost"), WithPort(9090))
		assert.Equal(t, "localhost", s.Address)
		assert.Equal(t, 9090, s.Port)
		assert.Equal(t, "localhost:9090", s.Addr)
	})

	t.Run("WithLogger", func(t *testing.T) {
		logger := zap.NewExample()
		s := New(app, WithLogger(logger))
		assert.Equal(t, logger, s.Logger)
	})

	t.Run("WithLogger nil", func(t *testing.T) {
		s := New(app, WithLogger(nil))
		assert.NotNil(t, s.Logger)
	})

	t.Run("WithDebug", func(t *testing.T) {
		s := New(app, WithDebug(true))
		assert.True(t, s.Debug)
	})

	t.Run("WithStartupTimeout", func(t *testing.T) {
		timeout := 5 * time.Second
		s := New(app, WithStartupTimeout(timeout))
		assert.Equal(t, timeout, s.StartupTimeout)
	})

	t.Run("WithStartupTimeout zero", func(t *testing.T) {
		s := New(app, WithStartupTimeout(0))
		assert.Equal(t, time.Second*10, s.StartupTimeout) // Should keep default
	})

	t.Run("WithShutdownTimeout", func(t *testing.T) {
		timeout := 15 * time.Second
		s := New(app, WithShutdownTimeout(timeout))
		assert.Equal(t, timeout, s.ShutdownTimeout)
	})

	t.Run("WithShutdownTimeout zero", func(t *testing.T) {
		s := New(app, WithShutdownTimeout(0))
		assert.Equal(t, time.Second*10, s.ShutdownTimeout) // Should keep default
	})

	t.Run("WithMiddleware", func(t *testing.T) {
		middleware := func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("X-Test", "middleware")
				next.ServeHTTP(w, r)
			})
		}
		s := New(app, WithMiddleware(middleware))

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		s.Handler.ServeHTTP(w, r)

		assert.Equal(t, "middleware", w.Header().Get("X-Test"))
	})

	t.Run("WithMiddleware nil", func(t *testing.T) {
		s := New(app, WithMiddleware(nil))
		assert.NotNil(t, s.Handler)
	})
}

// Test server error log writer
func TestServerErrorLogWriter(t *testing.T) {
	core, logs := observer.New(zapcore.DebugLevel)
	logger := zap.New(core)
	writer := &serverErrorLogWriter{Logger: logger}

	t.Run("TLS handshake error", func(t *testing.T) {
		logs.TakeAll() // Clear previous logs

		msg := "http: TLS handshake error from 172.16.0.3:42672: EOF\n"
		n, err := writer.Write([]byte(msg))

		assert.NoError(t, err)
		assert.Equal(t, len(msg), n)

		logEntries := logs.All()
		assert.Len(t, logEntries, 1)
		assert.Equal(t, "server", logEntries[0].Message)
		assert.Equal(t, zapcore.DebugLevel, logEntries[0].Level)
	})

	t.Run("URL query semicolon error", func(t *testing.T) {
		logs.TakeAll() // Clear previous logs

		msg := "http: URL query contains semicolon, which is deprecated\n"
		n, err := writer.Write([]byte(msg))

		assert.NoError(t, err)
		assert.Equal(t, len(msg), n)

		logEntries := logs.All()
		assert.Len(t, logEntries, 1)
		assert.Equal(t, "server", logEntries[0].Message)
		assert.Equal(t, zapcore.DebugLevel, logEntries[0].Level)
	})

	t.Run("other server error", func(t *testing.T) {
		logs.TakeAll() // Clear previous logs

		msg := "some other server error\n"
		n, err := writer.Write([]byte(msg))

		assert.NoError(t, err)
		assert.Equal(t, len(msg), n)

		logEntries := logs.All()
		assert.Len(t, logEntries, 1)
		assert.Equal(t, "server", logEntries[0].Message)
		assert.Equal(t, zapcore.WarnLevel, logEntries[0].Level)
	})
}

// Test metrics integration
func TestServerWithMetrics(t *testing.T) {
	processor := &testProcessor{}
	app := imagor.New(imagor.WithProcessors(processor))
	mockMetrics := &testMetrics{}

	t.Run("server with metrics", func(t *testing.T) {
		s := New(app, WithMetrics(mockMetrics))
		assert.Equal(t, mockMetrics, s.Metrics)
		assert.False(t, isNil(s.Metrics))
	})

	t.Run("server without metrics", func(t *testing.T) {
		s := New(app, WithMetrics(nil))
		assert.True(t, isNil(s.Metrics))
	})

	t.Run("metrics middleware integration", func(t *testing.T) {
		s := New(app, WithMetrics(mockMetrics))

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/test", nil)
		s.Handler.ServeHTTP(w, r)

		assert.Equal(t, 1, mockMetrics.HandleCnt)
	})
}

// Mock metrics for testing
type testMetrics struct {
	StartupCnt  int
	ShutdownCnt int
	HandleCnt   int
}

func (m *testMetrics) Handle(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		m.HandleCnt++
		next.ServeHTTP(w, r)
	})
}

func (m *testMetrics) Startup(ctx context.Context) error {
	m.StartupCnt++
	return nil
}

func (m *testMetrics) Shutdown(ctx context.Context) error {
	m.ShutdownCnt++
	return nil
}

// Test handler functions
func TestHandlerFunctions(t *testing.T) {
	t.Run("isNoopRequest", func(t *testing.T) {
		// Test healthcheck
		r := httptest.NewRequest(http.MethodGet, "/healthcheck", nil)
		assert.True(t, isNoopRequest(r))

		// Test favicon
		r = httptest.NewRequest(http.MethodGet, "/favicon.ico", nil)
		assert.True(t, isNoopRequest(r))

		// Test non-noop request
		r = httptest.NewRequest(http.MethodGet, "/api/test", nil)
		assert.False(t, isNoopRequest(r))

		// Test POST method (should not be noop even for healthcheck)
		r = httptest.NewRequest(http.MethodPost, "/healthcheck", nil)
		assert.False(t, isNoopRequest(r))
	})

	t.Run("handleOk", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/", nil)

		handleOk(w, r)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

// Test panic handler
func TestPanicHandler(t *testing.T) {
	core, logs := observer.New(zapcore.ErrorLevel)
	logger := zap.New(core)

	processor := &testProcessor{}
	app := imagor.New(imagor.WithProcessors(processor))
	s := New(app, WithLogger(logger))

	t.Run("panic with error", func(t *testing.T) {
		logs.TakeAll() // Clear previous logs

		handler := s.panicHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			panic(fmt.Errorf("test error"))
		}))

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/", nil)

		handler.ServeHTTP(w, r)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		assert.Contains(t, w.Body.String(), "test error")

		logEntries := logs.All()
		assert.Len(t, logEntries, 1)
		assert.Equal(t, "panic", logEntries[0].Message)
	})

	t.Run("panic with string", func(t *testing.T) {
		logs.TakeAll() // Clear previous logs

		handler := s.panicHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			panic("string panic")
		}))

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/", nil)

		handler.ServeHTTP(w, r)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		assert.Contains(t, w.Body.String(), "string panic")

		logEntries := logs.All()
		assert.Len(t, logEntries, 1)
		assert.Equal(t, "panic", logEntries[0].Message)
	})

	t.Run("no panic", func(t *testing.T) {
		logs.TakeAll() // Clear previous logs

		handler := s.panicHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("success"))
		}))

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/", nil)

		handler.ServeHTTP(w, r)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "success", w.Body.String())

		logEntries := logs.All()
		assert.Len(t, logEntries, 0)
	})
}
