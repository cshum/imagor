package vipsprocessor

import (
	"context"
	"fmt"
	"github.com/cshum/imagor"
	"github.com/cshum/imagor/imagorpath"
	"github.com/cshum/imagor/storage/filestorage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

var tests = []struct {
	name string
	path string
}{
	{"original", "gopher-front.png"},
	{"resize center", "100x100/filters:quality(70):format(jpeg)/gopher.png"},
	{"resize smart", "100x100/smart/filters:autojpg()/gopher.png"},
	{"resize top", "200x100/top/filters:quality(70):format(tiff)/gopher.png"},
	{"resize top", "200x100/right/top/gopher.png"},
	{"resize bottom", "200x100/bottom/gopher.png"},
	{"resize bottom", "200x100/left/bottom/gopher.png"},
	{"resize left", "100x200/left/gopher.png"},
	{"resize left", "100x200/left/bottom/gopher.png"},
	{"resize right", "100x200/right/gopher.png"},
	{"resize right", "100x200/right/top/gopher.png"},
	{"fit-in unspecified height", "fit-in/500x0/filters:fill(white):format(jpg)/gopher-front.png"},
	{"resize unspecified height", "500x0/filters:fill(white):format(jpg)/gopher-front.png"},
	{"fit-in unspecified width", "fit-in/0x500/filters:fill(white):format(jpg)/gopher-front.png"},
	{"resize unspecified width", "0x500/filters:fill(white):format(jpg)/gopher-front.png"},
	{"stretch", "stretch/100x100/filters:modulate(-10,30,20)/gopher.png"},
	{"fit-in flip hue", "fit-in/-200x0/filters:hue(290):saturation(100):fill(FFO):upscale()/gopher.png"},
	{"fit-in padding", "fit-in/100x100/10x5/filters:fill(white)/gopher.png"},
	{"resize padding", "100x100/10x5/top/filters:fill(white)/gopher.png"},
	{"stretch padding", "stretch/100x100/10x5/filters:fill(white)/gopher.png"},
	{"padding", "0x0/40x50/filters:fill(white)/gopher-front.png"},
	{"fill auto", "fit-in/400x400/filters:fill(auto)/find_trim.png"},
	{"fill auto bottom-right", "fit-in/400x400/filters:fill(auto,bottom-right)/find_trim.png"},
	{"resize top flip blur", "200x-210/top/filters:blur(5):sharpen(5):background_color(ffff00):format(jpeg):quality(70)/gopher.png"},
	{"crop stretch top flip", "10x20:3000x5000/stretch/100x200/filters:brightness(-20):contrast(50):rgb(10,-50,30):fill(black)/gopher.png"},
	{"padding rotation fill blur grayscale", "/fit-in/200x210/20x20/filters:rotate(90):rotate(270):rotate(180):fill(blur):grayscale()/gopher.png"},
	{"fill round_corner", "fit-in/0x210/filters:fill(yellow):round_corner(40,60,green)/gopher.png"},
	{"trim right", "trim:bottom-right/500x500/filters:strip_exif():upscale():no_upscale()/find_trim.png"},
	{"trim upscale", "trim/fit-in/1000x1000/filters:upscale():strip_icc()/find_trim.png"},
	{"trim tolerance", "trim:50/500x500/filters:stretch()/find_trim.png"},
	{"trim filter", "/fit-in/100x100/filters:fill(auto):trim(50)/find_trim.png"},
	{"watermark", "fit-in/500x500/filters:fill(white):watermark(gopher.png,10p,repeat,30,20,20):watermark(gopher.png,repeat,bottom,30,30,30):watermark(gopher-front.png,center,-10p)/gopher.png"},
}

func doTest(t *testing.T, name string, app *imagor.Imagor, cleanup func(func())) {
	t.Run(name, func(t *testing.T) {
		require.NoError(t, app.Startup(context.Background()))
		t.Parallel()
		cleanup(func() {
			require.NoError(t, app.Shutdown(context.Background()))
		})

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				w := httptest.NewRecorder()
				app.ServeHTTP(w, httptest.NewRequest(
					http.MethodGet, fmt.Sprintf("/unsafe/%s", tt.path), nil))
				assert.Equal(t, 200, w.Code)
				path := filepath.Join(testDataDir, "result", imagorpath.Normalize(tt.path))
				buf, err := ioutil.ReadFile(path)
				require.NoError(t, err)
				if b := w.Body.Bytes(); !reflect.DeepEqual(buf, b) {
					if len(b) < 512 {
						t.Error(string(b))
					} else {
						t.Errorf("%s: not equal", path)
					}
				}
			})
		}
	})
}

func TestVipsProcessor(t *testing.T) {
	doTest(t, "from buffer", imagor.New(
		imagor.WithLoaders(filestorage.New(testDataDir)),
		imagor.WithUnsafe(true),
		imagor.WithDebug(true),
		imagor.WithLogger(zap.NewExample()),
		imagor.WithRequestTimeout(time.Second*3),
		imagor.WithProcessors(New(
			WithDebug(true),
		)),
		imagor.WithResultSavers(filestorage.New(
			filepath.Join(testDataDir, "result"),
			filestorage.WithSaveErrIfExists(true),
		)),
	), t.Cleanup)
	doTest(t, "from file", imagor.New(
		imagor.WithLoaders(filestorage.New(testDataDir)),
		imagor.WithUnsafe(true),
		imagor.WithDebug(true),
		imagor.WithLogger(zap.NewExample()),
		imagor.WithRequestTimeout(time.Second*3),
		imagor.WithProcessors(New(
			WithDebug(false),
			WithLogger(zap.NewExample()),
			WithLoadFromFile(true),
		)),
		imagor.WithResultSavers(filestorage.New(
			filepath.Join(testDataDir, "result"),
			filestorage.WithSaveErrIfExists(true),
		)),
	), t.Cleanup)
}
