package server

import (
	"context"
	"fmt"
	"github.com/cshum/imagor"
	"github.com/cshum/imagor/imagorpath"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
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

func TestServer_Run(t *testing.T) {
	ctx, done := context.WithCancel(context.Background())
	processor := &testProcessor{}
	app := imagor.New(imagor.WithProcessors(processor))
	s := New(app,
		WithDebug(true),
		WithAddr(":0"),
		WithStartupTimeout(time.Millisecond),
		WithShutdownTimeout(time.Millisecond),
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
	s := New(imagor.New(imagor.WithUnsafe(true)),
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
		WithCORS(true))

	w := httptest.NewRecorder()
	s.Handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "https://example.com/favicon.ico", nil))
	assert.Equal(t, 200, w.Code)
	assert.NotEmpty(t, w.Header().Get("Vary"))
	assert.Equal(t, "Bar", w.Header().Get("X-Foo"))

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
	assert.Equal(t, http.StatusPermanentRedirect, w.Code)
	assert.Equal(t, "https://example.com/", w.Header().Get("Location"))
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
