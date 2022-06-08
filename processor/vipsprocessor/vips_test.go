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

type test struct {
	name          string
	path          string
	checkTypeOnly bool
}

func TestVipsProcessor(t *testing.T) {
	doTests(t, "parent", []test{}, WithDebug(true), WithLogger(zap.NewExample()))
	t.Parallel()
	t.Run("vips", func(t *testing.T) {
		var resultDir = filepath.Join(testDataDir, "result")
		var tests = []test{
			{name: "original", path: "gopher-front.png"},
			{name: "export gif", path: "filters:format(gif):quality(70)/gopher-front.png", checkTypeOnly: true},
			{name: "export webp", path: "filters:format(webp):quality(70)/gopher-front.png", checkTypeOnly: true},
			{name: "export avif", path: "filters:format(avif):quality(70)/gopher-front.png", checkTypeOnly: true},
			{name: "export tiff", path: "filters:format(tiff):quality(70)/gopher-front.png", checkTypeOnly: true},
			{name: "no-ops", path: "filters:frames():frames(0):round_corner():padding():rotate():proportion():proportion(-10):brightness():contrast():hue():saturation():rgb():modulate()/gopher-front.png"},
			{name: "resize center", path: "100x100/filters:quality(70):format(jpeg)/gopher.png"},
			{name: "resize smart", path: "100x100/smart/filters:autojpg()/gopher.png"},
			{name: "resize smart focal", path: "300x100/smart/filters:fill(white):format(jpeg):focal(589x401:1000x814)/gopher.png"},
			{name: "resize smart focal vertical", path: "100x300/smart/filters:fill(white):format(jpeg):focal(589x401:1000x814)/gopher.png"},
			{name: "resize smart focal with crop", path: "0x100:9999x9999/300x100/smart/filters:fill(white):format(jpeg):focal(589x401:1000x814)/gopher.png"},
			{name: "resize smart focal float", path: "300x100/smart/filters:fill(white):format(jpeg):focal(0.35x0.25:0.6x0.3)/gopher.png"},
			{name: "resize top", path: "200x100/top/filters:quality(70):format(tiff)/gopher.png"},
			{name: "resize top", path: "200x100/right/top/gopher.png"},
			{name: "resize bottom", path: "200x100/bottom/gopher.png"},
			{name: "resize bottom", path: "200x100/left/bottom/gopher.png"},
			{name: "resize left", path: "100x200/left/gopher.png"},
			{name: "resize left", path: "100x200/left/bottom/gopher.png"},
			{name: "resize right", path: "100x200/right/gopher.png"},
			{name: "resize right", path: "100x200/right/top/gopher.png"},
			{name: "proportion", path: "filters:proportion(10)/gopher.png"},
			{name: "proportion float", path: "filters:proportion(0.1)/gopher.png"},
			{name: "fit-in unspecified height", path: "fit-in/500x0/filters:fill(white):format(jpg)/gopher-front.png"},
			{name: "resize unspecified height", path: "500x0/filters:fill(white):format(jpg)/gopher-front.png"},
			{name: "fit-in unspecified width", path: "fit-in/0x500/filters:fill(white):format(jpg)/gopher-front.png"},
			{name: "resize unspecified width", path: "0x500/filters:fill(white):format(jpg)/gopher-front.png"},
			{name: "stretch", path: "stretch/100x100/filters:modulate(-10,30,20)/gopher.png"},
			{name: "fit-in flip hue", path: "fit-in/-200x0/filters:hue(290):saturation(100):fill(FFO):upscale()/gopher.png"},
			{name: "fit-in padding", path: "fit-in/100x100/10x5/filters:fill(white)/gopher.png"},
			{name: "resize padding", path: "100x100/10x5/top/filters:fill(white)/gopher.png"},
			{name: "stretch padding", path: "stretch/100x100/10x5/filters:fill(white)/gopher.png"},
			{name: "padding", path: "0x0/40x50/filters:fill(white)/gopher-front.png"},
			{name: "max_bytes", path: "filters:max_bytes(60000):format(jpg):fill(white)/gopher.png"},
			{name: "max_bytes 2", path: "filters:max_bytes(6000):format(jpg):fill(white)/gopher.png"},
			{name: "fill auto", path: "fit-in/400x400/filters:fill(auto)/find_trim.png"},
			{name: "fill auto bottom-right", path: "fit-in/400x400/filters:fill(auto,bottom-right)/find_trim.png"},
			{name: "resize top flip blur", path: "200x-210/top/filters:blur(5):sharpen(5):background_color(ffff00):format(jpeg):quality(70)/gopher.png"},
			{name: "crop stretch top flip", path: "10x20:3000x5000/stretch/100x200/filters:brightness(-20):contrast(50):rgb(10,-50,30):fill(black)/gopher.png"},
			{name: "crop-percent stretch top flip", path: "0.006120x0.008993:1.0x1.0/stretch/100x200/filters:brightness(-20):contrast(50):rgb(10,-50,30):fill(black)/gopher.png"},
			{name: "padding rotation fill blur grayscale", path: "/fit-in/200x210/20x20/filters:rotate(90):rotate(270):rotate(180):fill(blur):grayscale()/gopher.png"},
			{name: "fill round_corner", path: "fit-in/0x210/filters:fill(yellow):round_corner(40,60,green)/gopher.png"},
			{name: "trim with crop", path: "trim:bottom-right/50x50:0x0/find_trim.png"},
			{name: "trim right", path: "trim:bottom-right/500x500/filters:strip_exif():upscale():no_upscale()/find_trim.png"},
			{name: "trim upscale", path: "trim/fit-in/1000x1000/filters:upscale():strip_icc()/find_trim.png"},
			{name: "trim tolerance", path: "trim:50/500x500/filters:stretch()/find_trim.png"},
			{name: "trim filter", path: "/fit-in/100x100/filters:fill(auto):trim(50)/find_trim.png"},
			{name: "watermark", path: "fit-in/500x500/filters:fill(white):watermark(gopher.png,10p,repeat,30,20,20):watermark(gopher.png,repeat,bottom,30,30,30):watermark(gopher-front.png,center,-10p)/gopher.png"},
			{name: "watermark float", path: "fit-in/500x500/filters:fill(white):watermark(gopher.png,0.1,repeat,30,20,20):watermark(gopher.png,repeat,bottom,30,30,30):watermark(gopher-front.png,center,-0.1)/gopher.png"},
			{name: "watermark align", path: "fit-in/500x500/filters:fill(white):watermark(gopher.png,left,top,30,20,20):watermark(gopher.png,right,center,30,30,30):watermark(gopher-front.png,-20,-10)/gopher.png"},

			{name: "original no animate", path: "filters:fill(white):format(jpeg)/dancing-banana.gif"},
			{name: "original animated", path: "dancing-banana.gif"},
			{name: "original animated quality", path: "filters:quality(60)/dancing-banana.gif"},
			{name: "crop animated", path: "30x20:100x150/dancing-banana.gif"},
			{name: "crop-percent animated", path: "0.1x0.2:0.89x0.72/dancing-banana.gif"},
			{name: "smart focal animated", path: "100x30/smart/filters:focal(0.1x0:0.89x0.72)/dancing-banana.gif"},
			{name: "watermark frames static", path: "fit-in/200x200/filters:fill(white):frames(3):watermark(dancing-banana.gif):format(jpeg)/gopher.png"},
			{name: "padding", path: "fit-in/-180x180/10x10/filters:fill(yellow):padding(white,10,20,30,40):format(jpeg)/gopher.png"},
		}
		doTests(t, resultDir, tests, WithDebug(true), WithLogger(zap.NewExample()))
	})
	t.Run("max frames", func(t *testing.T) {
		var resultDir = filepath.Join(testDataDir, "result/max-frames")
		var tests = []test{
			{name: "original", path: "gopher-front.png"},
			{name: "original no animate", path: "filters:fill(white):format(jpeg)/dancing-banana.gif"},
			{name: "original animated", path: "dancing-banana.gif"},
			{name: "crop animated", path: "30x20:100x150/dancing-banana.gif"},
		}
		doTests(t, resultDir, tests, WithDebug(true), WithDisableBlur(true), WithMaxAnimationFrames(100))
	})
	t.Run("max frames limited", func(t *testing.T) {
		var resultDir = filepath.Join(testDataDir, "result/max-frames-limited")
		var tests = []test{
			{name: "original", path: "gopher-front.png"},
			{name: "original no animate", path: "filters:fill(white):format(jpeg)/dancing-banana.gif"},
			{name: "original animated", path: "dancing-banana.gif"},
			{name: "crop animated", path: "30x20:100x150/dancing-banana.gif"},
		}
		doTests(t, resultDir, tests, WithDebug(true), WithDisableBlur(true), WithMaxAnimationFrames(3))
	})
	t.Run("unsupported", func(t *testing.T) {
		loader := filestorage.New(testDataDir + "/../")
		app := imagor.New(
			imagor.WithLoaders(loader),
			imagor.WithUnsafe(true),
			imagor.WithDebug(true),
			imagor.WithLogger(zap.NewExample()),
			imagor.WithProcessors(New(WithDebug(true))),
		)
		require.NoError(t, app.Startup(context.Background()))
		t.Cleanup(func() {
			assert.NoError(t, app.Shutdown(context.Background()))
		})
		w := httptest.NewRecorder()
		app.ServeHTTP(w, httptest.NewRequest(
			http.MethodGet, "/unsafe/README.md", nil))
		assert.Equal(t, 406, w.Code)

		buf, err := ioutil.ReadFile(testDataDir + "/../README.md")
		require.NoError(t, err)
		assert.Equal(t, buf, w.Body.Bytes(), "should return original file")
	})
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
			b := imagor.NewBytes(w.Body.Bytes())
			require.NotEqual(t, imagor.BytesTypeUnknown, b.BytesType())
			_ = resStorage.Put(context.Background(), tt.path, b)
			path := filepath.Join(resultDir, imagorpath.Normalize(tt.path, nil))

			bc := imagor.NewBytesFilePath(path)
			buf, err := bc.ReadAll()
			require.NoError(t, err)
			if tt.checkTypeOnly {
				assert.Equal(t, bc.ContentType(), b.ContentType())
				assert.Equal(t, bc.BytesType(), b.BytesType())
			} else {
				if bb := w.Body.Bytes(); reflect.DeepEqual(buf, bb) {
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
			}
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
