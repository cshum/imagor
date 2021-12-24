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
	"reflect"
	"runtime"
	"testing"
	"time"
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
		imagor.WithRequestTimeout(time.Second),
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
	t.Parallel()
	tests := []struct {
		name string
		path string
	}{
		{"fit-in flip hue", "fit-in/-200x210/filters:hue(290):saturation(100):fill(FFO)/gopher.png"},
		{"resize top flip blur", "200x-210/top/filters:blur(5):fill(ffff00):format(jpeg):quality(70)/gopher.png"},
		{"crop stretch top flip", "10x20:300x500/stretch/100x200/filters:brightness(-20):contrast(50):rgb(10,-50,30):fill(black)/gopher.png"},
		{"fit-in padding bottom flip grayscale fill blur", "/fit-in/-200x-210/30x30/filters:rotate(90):fill(blur):grayscale()/gopher.png"},
		{"fill round_corner", "fit-in/200x210/filters:fill(yellow):round_corner(40,60)/gopher.png"},
		{"trim", "trim/find_trim.png"},
		{"trim bottom", "trim:bottom-right/find_trim.png"},
		{"trim tolerance", "trim:50/find_trim.png"},
		{"trim filter", "/fit-in/50x50/filters:trim(50)/find_trim.png"},

		{"watermark", "filters:fill(white):watermark(gopher.png,10p,repeat,30,20,20):watermark(gopher.png,repeat,bottom,30,30,30):watermark(gopher.png,repeat,repeat,70,10,10):watermark(gopher-front.png,center,-10p)/gopher.png"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			app.ServeHTTP(w, httptest.NewRequest(
				http.MethodGet, fmt.Sprintf("/unsafe/%s", tt.path), nil))
			assert.Equal(t, 200, w.Code)
			buf, err := ioutil.ReadFile(filepath.Join(testDataDir, "result", tt.path))
			assert.NoError(t, err)
			if b := w.Body.Bytes(); !reflect.DeepEqual(buf, b) {
				if len(b) < 512 {
					t.Error(string(b))
				} else {
					t.Error("result not equal")
				}
			}
		})
	}
}
