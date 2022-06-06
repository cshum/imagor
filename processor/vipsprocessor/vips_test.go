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
	"image"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"

	_ "golang.org/x/image/tiff"
	_ "golang.org/x/image/webp"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
)

var testDataDir string

func init() {
	_, b, _, _ := runtime.Caller(0)
	testDataDir = filepath.Join(filepath.Dir(b), "../../testdata")
}

func TestVipsProcessor(t *testing.T) {
	doTests(t, "parent", []test{}, WithDebug(true), WithLogger(zap.NewExample()))
	t.Parallel()
	t.Run("vips", func(t *testing.T) {
		var resultDir = filepath.Join(testDataDir, "result")
		var tests = []test{
			{"original", "gopher-front.png"},
			{"resize center", "100x100/filters:quality(70):format(jpeg)/gopher.png"},
			{"resize smart", "100x100/smart/filters:autojpg()/gopher.png"},
			{"resize smart focal", "300x100/smart/filters:fill(white):format(jpeg):focal(589x401:1000x814)/gopher.png"},
			{"resize smart focal vertical", "100x300/smart/filters:fill(white):format(jpeg):focal(589x401:1000x814)/gopher.png"},
			{"resize smart focal with crop", "0x100:9999x9999/300x100/smart/filters:fill(white):format(jpeg):focal(589x401:1000x814)/gopher.png"},
			{"resize smart focal float", "300x100/smart/filters:fill(white):format(jpeg):focal(0.35x0.25:0.6x0.3)/gopher.png"},
			{"resize top", "200x100/top/filters:quality(70):format(tiff)/gopher.png"},
			{"resize top", "200x100/right/top/gopher.png"},
			{"resize bottom", "200x100/bottom/gopher.png"},
			{"resize bottom", "200x100/left/bottom/gopher.png"},
			{"resize left", "100x200/left/gopher.png"},
			{"resize left", "100x200/left/bottom/gopher.png"},
			{"resize right", "100x200/right/gopher.png"},
			{"resize right", "100x200/right/top/gopher.png"},
			{"proportion", "filters:proportion(10)/gopher.png"},
			{"proportion float", "filters:proportion(0.1)/gopher.png"},
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
			{"max_bytes", "filters:max_bytes(60000):format(jpg):fill(white)/gopher.png"},
			{"fill auto", "fit-in/400x400/filters:fill(auto)/find_trim.png"},
			{"fill auto bottom-right", "fit-in/400x400/filters:fill(auto,bottom-right)/find_trim.png"},
			{"resize top flip blur", "200x-210/top/filters:blur(5):sharpen(5):background_color(ffff00):format(jpeg):quality(70)/gopher.png"},
			{"crop stretch top flip", "10x20:3000x5000/stretch/100x200/filters:brightness(-20):contrast(50):rgb(10,-50,30):fill(black)/gopher.png"},
			{"crop-percent stretch top flip", "0.006120x0.008993:1.0x1.0/stretch/100x200/filters:brightness(-20):contrast(50):rgb(10,-50,30):fill(black)/gopher.png"},
			{"padding rotation fill blur grayscale", "/fit-in/200x210/20x20/filters:rotate(90):rotate(270):rotate(180):fill(blur):grayscale()/gopher.png"},
			{"fill round_corner", "fit-in/0x210/filters:fill(yellow):round_corner(40,60,green)/gopher.png"},
			{"trim with crop", "trim:bottom-right/50x50:0x0/find_trim.png"},
			{"trim right", "trim:bottom-right/500x500/filters:strip_exif():upscale():no_upscale()/find_trim.png"},
			{"trim upscale", "trim/fit-in/1000x1000/filters:upscale():strip_icc()/find_trim.png"},
			{"trim tolerance", "trim:50/500x500/filters:stretch()/find_trim.png"},
			{"trim filter", "/fit-in/100x100/filters:fill(auto):trim(50)/find_trim.png"},
			{"watermark", "fit-in/500x500/filters:fill(white):watermark(gopher.png,10p,repeat,30,20,20):watermark(gopher.png,repeat,bottom,30,30,30):watermark(gopher-front.png,center,-10p)/gopher.png"},
			{"watermark float", "fit-in/500x500/filters:fill(white):watermark(gopher.png,0.1,repeat,30,20,20):watermark(gopher.png,repeat,bottom,30,30,30):watermark(gopher-front.png,center,-0.1)/gopher.png"},
			{"watermark align", "fit-in/500x500/filters:fill(white):watermark(gopher.png,left,top,30,20,20):watermark(gopher.png,right,center,30,30,30):watermark(gopher-front.png,-20,-10)/gopher.png"},

			{"original no animate", "filters:fill(white):format(jpeg)/dancing-banana.gif"},
			{"original animated", "dancing-banana.gif"},
			{"crop animated", "30x20:100x150/dancing-banana.gif"},
			{"crop-percent animated", "0.1x0.2:0.89x0.72/dancing-banana.gif"},
			{"smart focal animated", "100x30/smart/filters:focal(0.1x0:0.89x0.72)/dancing-banana.gif"},
			{"watermark frames static", "fit-in/200x200/filters:fill(white):frames(3):watermark(dancing-banana.gif):format(jpeg)/gopher.png"},
			{"padding", "fit-in/-180x180/10x10/filters:fill(yellow):padding(white,10,20,30,40):format(jpeg)/gopher.png"},
		}
		doTests(t, resultDir, tests, WithDebug(true), WithLogger(zap.NewExample()))
	})
	t.Run("max frames", func(t *testing.T) {
		var resultDir = filepath.Join(testDataDir, "result/max-frames")
		var tests = []test{
			{"original", "gopher-front.png"},
			{"original no animate", "filters:fill(white):format(jpeg)/dancing-banana.gif"},
			{"original animated", "dancing-banana.gif"},
			{"crop animated", "30x20:100x150/dancing-banana.gif"},
		}
		doTests(t, resultDir, tests, WithDebug(true), WithDisableBlur(true), WithMaxAnimationFrames(100))
	})
	t.Run("max frames limited", func(t *testing.T) {
		var resultDir = filepath.Join(testDataDir, "result/max-frames-limited")
		var tests = []test{
			{"original", "gopher-front.png"},
			{"original no animate", "filters:fill(white):format(jpeg)/dancing-banana.gif"},
			{"original animated", "dancing-banana.gif"},
			{"crop animated", "30x20:100x150/dancing-banana.gif"},
		}
		doTests(t, resultDir, tests, WithDebug(true), WithDisableBlur(true), WithMaxAnimationFrames(3))
	})
}

type test struct {
	name string
	path string
}

func doTests(t *testing.T, resultDir string, tests []test, opts ...Option) {
	resStorage := filestorage.New(
		resultDir,
		filestorage.WithSaveErrIfExists(true),
	)
	app := imagor.New(
		imagor.WithLoaders(filestorage.New(testDataDir)),
		imagor.WithUnsafe(true),
		imagor.WithDebug(true),
		imagor.WithLogger(zap.NewExample()),
		imagor.WithProcessors(New(opts...)),
	)
	require.NoError(t, app.Startup(context.Background()))
	t.Cleanup(func() {
		assert.NoError(t, app.Shutdown(context.Background()))
	})
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			app.ServeHTTP(w, httptest.NewRequest(
				http.MethodGet, fmt.Sprintf("/unsafe/%s", tt.path), nil))
			assert.Equal(t, 200, w.Code)
			_ = resStorage.Put(context.Background(), tt.path, imagor.NewBytes(w.Body.Bytes()))
			path := filepath.Join(resultDir, imagorpath.Normalize(tt.path, nil))

			buf, err := ioutil.ReadFile(path)
			require.NoError(t, err)
			if b := w.Body.Bytes(); reflect.DeepEqual(buf, b) {
				return
			}

			existingImageFile, err := os.Open(path)
			require.NoError(t, err)
			defer existingImageFile.Close()
			img1, imageType, err := image.Decode(existingImageFile)
			require.NoError(t, err)
			img2, imageType2, err := image.Decode(w.Body)
			require.NoError(t, err)
			require.Equal(t, imageType, imageType2, "%s %s", imageType, imageType2)
			require.Equalf(t, img1.Bounds(), img2.Bounds(), "image bounds not equal: %+v, %+v", img1.Bounds(), img2.Bounds())
			require.True(t, pixelCompare(img1, img2) < 10, "image pixel mismatch")
		})
	}
}

func pixelCompare(img1, img2 image.Image) (accuErr int64) {
	b := img1.Bounds()
	for i := 0; i < b.Dx(); i++ {
		for j := 0; j < b.Dy(); j++ {
			r1, g1, b1, a1 := img1.At(i, j).RGBA()
			r2, g2, b2, a2 := img2.At(i, j).RGBA()
			dr, dg, db, da := r1-r2, g1-g2, b1-b2, a1-a2
			if dr < 0 {
				dr = -dr
			}
			if dg < 0 {
				dg = -dg
			}
			if db < 0 {
				db = -db
			}
			if da < 0 {
				da = -da
			}
			accuErr += int64(dr + dg + db + da)
		}
	}
	return
}
