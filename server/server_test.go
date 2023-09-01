package server

import (
	"context"
	"fmt"
	"github.com/cshum/imagor"
	"github.com/cshum/imagor/imagorpath"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
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
