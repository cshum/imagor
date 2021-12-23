package vipsprocessor

import (
	"context"
	"fmt"
	"github.com/cshum/imagor"
	"github.com/cshum/imagor/store/filestore"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"runtime"
	"testing"
)

var testDataDir string

func init() {
	_, b, _, _ := runtime.Caller(0)
	testDataDir = filepath.Join(filepath.Dir(b), "../../testdata")
}

func TestVipsProcessor(t *testing.T) {
	app := imagor.New(
		imagor.WithLoaders(filestore.New(testDataDir)),
		imagor.WithUnsafe(true),
		imagor.WithDebug(true),
		imagor.WithLogger(zap.NewExample()),
		imagor.WithProcessors(New(
			WithDebug(true),
			WithLogger(zap.NewExample()),
		)),
		imagor.WithResultStorages(filestore.New(
			filepath.Join(testDataDir, "result"),
			filestore.WithSaveErrIfExists(true),
		)),
	)
	assert.NoError(t, app.Startup(context.Background()))
	t.Cleanup(func() {
		assert.NoError(t, app.Shutdown(context.Background()))
	})
	tests := []struct {
		name string
		path string
	}{
		{"fit-in", "fit-in/200x210/gopher.png"},
		{"resize top", "200x210/top/gopher.png"},
		{"fill and format", "fit-in/200x210/filters:fill(yellow):format(jpeg)/gopher.png"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			app.ServeHTTP(w, httptest.NewRequest(
				http.MethodGet, fmt.Sprintf("/unsafe/%s", tt.path), nil))
			assert.Equal(t, 200, w.Code)
			buf, err := ioutil.ReadFile(filepath.Join(testDataDir, "result", tt.path))
			assert.NoError(t, err)
			assert.Equal(t, buf, w.Body.Bytes())
		})
	}
}
