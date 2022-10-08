package vips

import (
	"context"
	"fmt"
	"github.com/cshum/imagor"
	"github.com/cshum/imagor/imagorpath"
	"github.com/cshum/imagor/storage/filestorage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
)

var testDataDir string

func init() {
	_, b, _, _ := runtime.Caller(0)
	testDataDir = filepath.Join(filepath.Dir(b), "../testdata")
}

type test struct {
	name          string
	path          string
	checkTypeOnly bool
}

func TestProcessor(t *testing.T) {
	v := NewProcessor(WithDebug(true))
	require.NoError(t, v.Startup(context.Background()))
	t.Cleanup(func() {
		stats := &MemoryStats{}
		ReadVipsMemStats(stats)
		fmt.Println(stats)
		require.NoError(t, v.Shutdown(context.Background()))
	})
	t.Run("vips basic", func(t *testing.T) {
		var resultDir = filepath.Join(testDataDir, "golden")
		doGoldenTests(t, resultDir, []test{
			{name: "png", path: "gopher-front.png"},
			{name: "jpeg", path: "fit-in/100x100/demo1.jpg"},
			{name: "webp", path: "fit-in/100x100/demo3.webp"},
			{name: "tiff", path: "fit-in/100x100/gopher.tiff"},
			{name: "avif", path: "fit-in/100x100/gopher-front.avif", checkTypeOnly: true},
			{name: "export gif", path: "filters:format(gif):quality(70)/gopher-front.png"},
			{name: "export webp", path: "filters:format(webp):quality(70)/gopher-front.png"},
			{name: "export tiff", path: "filters:format(tiff):quality(70)/gopher-front.png"},
			{name: "export avif", path: "filters:format(avif):quality(70)/gopher-front.png", checkTypeOnly: true},
			{name: "export heif", path: "filters:format(heif):quality(70)/gopher-front.png", checkTypeOnly: true},
		}, WithDebug(true), WithLogger(zap.NewExample()))
	})
	t.Run("meta", func(t *testing.T) {
		var resultDir = filepath.Join(testDataDir, "golden")
		doGoldenTests(t, resultDir, []test{
			{name: "meta jpeg", path: "meta/fit-in/100x100/demo1.jpg"},
			{name: "meta gif", path: "meta/fit-in/100x100/dancing-banana.gif"},
			{name: "meta svg", path: "meta/test.svg"},
			{name: "meta format no animate", path: "meta/fit-in/100x100/filters:format(jpg)/dancing-banana.gif"},
			{name: "meta exif", path: "meta/Canon_40D.jpg"},
			{name: "meta strip exif", path: "meta/filters:strip_exif()/Canon_40D.jpg"},
		}, WithDebug(true), WithLogger(zap.NewExample()))
	})
	t.Run("vips operations", func(t *testing.T) {
		var resultDir = filepath.Join(testDataDir, "golden")
		doGoldenTests(t, resultDir, []test{
			{name: "no-ops", path: "filters:background_color():frames():frames(0):round_corner():padding():rotate():proportion():proportion(9999):proportion(0.0000000001):proportion(-10)/gopher-front.png"},
			{name: "no-ops 2", path: "trim/filters:watermark():blur(2):sharpen(2):brightness():contrast():hue():saturation():rgb():modulate()/dancing-banana.gif"},
			{name: "no-ops 3", path: "filters:proportion():proportion(9999):proportion(0.0000000001):proportion(-10)/gopher-front.png"},
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
			{name: "resize orient", path: "100x200/filters:orient(90)/gopher.png"},
			{name: "fit-in unspecified height", path: "fit-in/50x0/filters:fill(white):format(jpg)/Canon_40D.jpg"},
			{name: "resize unspecified height", path: "50x0/filters:fill(white):format(jpg)/Canon_40D.jpg"},
			{name: "fit-in unspecified width", path: "fit-in/0x50/filters:fill(white):format(jpg)/Canon_40D.jpg"},
			{name: "resize unspecified width", path: "0x50/filters:fill(white):format(jpg)/Canon_40D.jpg"},
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
			{name: "blur sharpen 2", path: "200x-210/top/filters:blur(1,2):sharpen(1,2):background_color(ff0):format(jpeg):quality(70)/gopher.png"},
			{name: "crop stretch top flip", path: "10x20:3000x5000/stretch/100x200/filters:brightness(-20):contrast(50):rgb(10,-50,30):fill(black)/gopher.png"},
			{name: "crop-percent stretch top flip", path: "0.006120x0.008993:1.0x1.0/stretch/100x200/filters:brightness(-20):contrast(50):rgb(10,-50,30):fill(black)/gopher.png"},
			{name: "padding rotation fill blur grayscale", path: "/fit-in/200x210/20x20/filters:rotate(90):rotate(270):rotate(180):fill(blur):grayscale()/gopher.png"},
			{name: "fill round_corner", path: "fit-in/0x210/filters:fill(yellow):round_corner(40,60,green)/gopher.png"},
			{name: "trim with crop", path: "trim:bottom-right/50x50:0x0/find_trim.png"},
			{name: "trim right", path: "trim:bottom-right/500x500/filters:strip_exif():upscale():no_upscale()/find_trim.png"},
			{name: "trim upscale", path: "trim/fit-in/1000x1000/filters:upscale():strip_icc()/find_trim.png"},
			{name: "trim tolerance", path: "trim:50/500x500/filters:stretch()/find_trim.png"},
			{name: "trim position tolerance filter", path: "50x50:0x0/filters:trim(50,bottom-right)/find_trim.png"},
			{name: "trim filter", path: "/fit-in/100x100/filters:fill(auto):trim(50)/find_trim.png"},
			{name: "watermark", path: "fit-in/500x500/filters:fill(white):watermark(gopher.png,10p,repeat,30,20,20):watermark(gopher.png,repeat,bottom,30,30,30):watermark(gopher-front.png,center,-10p)/gopher.png"},
			{name: "watermark non alpha", path: "filters:watermark(demo1.jpg,repeat,repeat,40,25,50)/demo1.jpg"},
			{name: "background color non alpha", path: "filters:background_color(yellow)/demo1.jpg"},
			{name: "watermark float", path: "fit-in/500x500/filters:fill(white):watermark(gopher.png,0.1,repeat,30,20,20):watermark(gopher.png,repeat,bottom,30,30,30):watermark(gopher-front.png,center,-0.1)/gopher.png"},
			{name: "watermark align", path: "fit-in/500x500/filters:fill(white):watermark(gopher.png,left,top,30,20,20):watermark(gopher.png,right,center,30,30,30):watermark(gopher-front.png,-20,-10)/gopher.png"},

			{name: "original no animate", path: "filters:fill(white):format(jpeg)/dancing-banana.gif"},
			{name: "original animated", path: "dancing-banana.gif"},
			{name: "original animated quality", path: "filters:quality(60)/dancing-banana.gif"},
			{name: "original animated max_frames", path: "filters:max_frames(3)/dancing-banana.gif"},
			{name: "rotate animated", path: "fit-in/100x150/filters:rotate(90):fill(yellow)/dancing-banana.gif"},
			{name: "crop animated", path: "30x20:100x150/dancing-banana.gif"},
			{name: "crop-percent animated", path: "0.1x0.2:0.89x0.72/dancing-banana.gif"},
			{name: "smart focal animated", path: "100x30/smart/filters:focal(0.1x0:0.89x0.72)/dancing-banana.gif"},
			{name: "watermark frames static", path: "fit-in/200x200/filters:fill(white):frames(3):watermark(dancing-banana.gif):format(jpeg)/gopher.png"},
			{name: "padding", path: "fit-in/-180x180/10x10/filters:fill(yellow):padding(white,10,20,30,40):format(jpeg)/gopher.png"},
			{name: "rotate fill", path: "fit-in/100x210/10x20:15x3/filters:rotate(90):fill(yellow)/gopher-front.png"},
			{name: "resize center animated", path: "100x100/dancing-banana.gif"},
			{name: "resize top animated", path: "200x100/top/dancing-banana.gif"},
			{name: "resize top animated", path: "200x100/right/top/dancing-banana.gif"},
			{name: "resize bottom animated", path: "200x100/bottom/dancing-banana.gif"},
			{name: "resize bottom animated", path: "200x100/left/bottom/dancing-banana.gif"},
			{name: "resize left animated", path: "100x200/left/dancing-banana.gif"},
			{name: "resize left animated", path: "100x200/left/bottom/dancing-banana.gif"},
			{name: "resize right animated", path: "100x200/right/dancing-banana.gif"},
			{name: "resize right animated", path: "100x200/right/top/dancing-banana.gif"},
			{name: "stretch animated", path: "stretch/100x200/dancing-banana.gif"},
			{name: "resize padding animated", path: "100x100/10x5/top/filters:fill(yellow)/dancing-banana.gif"},
			{name: "watermark animated", path: "fit-in/200x150/filters:fill(yellow):watermark(gopher-front.png,repeat,bottom,0,30,30)/dancing-banana.gif"},
			{name: "watermark animated align bottom right", path: "fit-in/200x150/filters:fill(yellow):watermark(gopher-front.png,-20,-10,0,30,30)/dancing-banana.gif"},
			{name: "watermark double animated", path: "fit-in/200x150/filters:fill(yellow):watermark(dancing-banana.gif,-20,-10,0,30,30):watermark(nyan-cat.gif,0,10,0,40,30)/dancing-banana.gif"},
			{name: "watermark double animated 2", path: "fit-in/200x150/filters:fill(yellow):watermark(dancing-banana.gif,30,-10,0,40,40):watermark(dancing-banana.gif,0,10,0,40,40)/nyan-cat.gif"},
			{name: "padding with watermark double animated", path: "200x0/20x20:100x20/filters:fill(yellow):watermark(dancing-banana.gif,-10,-10,0,50,50):watermark(dancing-banana.gif,-30,10,0,50,50)/nyan-cat.gif"},
			{name: "watermark frames animated", path: "fit-in/200x200/filters:fill(white):frames(3,200):watermark(dancing-banana.gif):format(gif)/gopher.png"},
			{name: "watermark frames animated repeated", path: "fit-in/200x200/filters:fill(white):frames(3,200):watermark(dancing-banana.gif,repeat,repeat,0,33,33):format(gif)/gopher.png"},
			{name: "watermark repeated animated", path: "fit-in/200x150/filters:fill(cyan):watermark(dancing-banana.gif,repeat,bottom,0,50,50)/dancing-banana.gif"},
			{name: "animated fill round_corner", path: "filters:fill(cyan):round_corner(60)/dancing-banana.gif"},
			{name: "label", path: "fit-in/300x200/10x10/filters:fill(yellow):label(IMAGOR,15,10,30,blue,30)/gopher-front.png"},
			{name: "label top left", path: "fit-in/300x200/10x10/filters:fill(yellow):label(IMAGOR,left,top,30,red,30)/gopher-front.png"},
			{name: "label right center", path: "fit-in/300x200/10x10/filters:fill(yellow):label(IMAGOR,right,center,30,red,30)/gopher-front.png"},
			{name: "label center bottom", path: "fit-in/300x200/10x10/filters:fill(yellow):label(IMAGOR,center,bottom,30,red,30)/gopher-front.png"},
			{name: "label negative", path: "fit-in/300x200/10x10/filters:fill(yellow):label(IMAGOR,-15,-10,30,red,30)/gopher-front.png"},
			{name: "label percentage", path: "fit-in/300x200/10x10/filters:fill(yellow):label(IMAGOR,-15p,10p,30,red,30)/gopher-front.png"},
			{name: "label float", path: "fit-in/300x200/10x10/filters:fill(yellow):label(IMAGOR,-0.15,0.1,30,red,30)/gopher-front.png"},
			{name: "label animated", path: "fit-in/150x200/10x00:10x50/filters:fill(yellow):label(IMAGOR,center,-30,25,black)/dancing-banana.gif"},
			{name: "label animated with font", path: "fit-in/150x200/10x00:10x50/filters:fill(cyan):label(IMAGOR,center,-30,25,white,0,monospace)/dancing-banana.gif"},
			{name: "strip exif", path: "filters:strip_exif()/Canon_40D.jpg"},
			{name: "svg", path: "test.svg"},
		}, WithDebug(true), WithLogger(zap.NewExample()))
	})
	t.Run("max frames", func(t *testing.T) {
		var resultDir = filepath.Join(testDataDir, "golden/max-frames")
		doGoldenTests(t, resultDir, []test{
			{name: "original", path: "gopher-front.png"},
			{name: "original no animate", path: "filters:fill(white):format(jpeg)/dancing-banana.gif"},
			{name: "original animated", path: "dancing-banana.gif"},
			{name: "original animated trim no-op", path: "trim/dancing-banana.gif"},
			{name: "crop animated", path: "30x20:100x150/dancing-banana.gif"},
			{name: "resize top animated", path: "200x100/top/dancing-banana.gif"},
			{name: "watermark repeated animated", path: "fit-in/200x150/filters:fill(cyan):watermark(dancing-banana.gif,repeat,bottom,0,50,50)/dancing-banana.gif"},
		}, WithDebug(true), WithDisableBlur(true), WithMaxAnimationFrames(100))
	})
	t.Run("max frames limited", func(t *testing.T) {
		var resultDir = filepath.Join(testDataDir, "golden/max-frames-limited")
		doGoldenTests(t, resultDir, []test{
			{name: "original", path: "gopher-front.png"},
			{name: "original no animate", path: "filters:fill(white):format(jpeg)/dancing-banana.gif"},
			{name: "original animated", path: "dancing-banana.gif"},
			{name: "original animated trim no-op", path: "trim/dancing-banana.gif"},
			{name: "original animated no-ops", path: "filters:max_frames(6)/dancing-banana.gif"},
			{name: "crop animated", path: "30x20:100x150/dancing-banana.gif"},
			{name: "resize top animated", path: "200x100/top/dancing-banana.gif"},
			{name: "watermark repeated animated", path: "fit-in/200x150/filters:fill(cyan):watermark(dancing-banana.gif,repeat,bottom,0,50,50)/dancing-banana.gif"},
		}, WithDebug(true), WithDisableBlur(true), WithMaxAnimationFrames(3))
	})
	t.Run("disable filters", func(t *testing.T) {
		var resultDir = filepath.Join(testDataDir, "golden/disable-filters")
		doGoldenTests(t, resultDir, []test{
			{name: "original no animate", path: "filters:fill(white):format(jpeg)/dancing-banana.gif"},
			{name: "watermark fill disabled", path: "fit-in/200x150/filters:fill(cyan):watermark(dancing-banana.gif,repeat,bottom,0,50,50)/dancing-banana.gif"},
		}, WithDebug(true), WithDisableFilters("fill", "watermark", "format"))
	})
	t.Run("no animation", func(t *testing.T) {
		var resultDir = filepath.Join(testDataDir, "golden/no-animation")
		doGoldenTests(t, resultDir, []test{
			{name: "png", path: "gopher-front.png"},
			{name: "gif", path: "dancing-banana.gif"},
		}, WithDebug(true), WithMaxAnimationFrames(1))
	})
	t.Run("max-filter-ops", func(t *testing.T) {
		var resultDir = filepath.Join(testDataDir, "golden/max-filter-ops")
		doGoldenTests(t, resultDir, []test{
			{name: "max-filter-ops within", path: "fit-in/200x150/filters:fill(yellow)/dancing-banana.gif"},
			{name: "max-filter-ops exceeded no ops", path: "fit-in/200x150/filters:fill(yellow):watermark(dancing-banana.gif,-20,-10,0,30,30):watermark(nyan-cat.gif,0,10,0,40,30)/dancing-banana.gif"},
		}, WithDebug(true), WithMaxFilterOps(1))
	})
	t.Run("image from memory", func(t *testing.T) {
		var resultDir = filepath.Join(testDataDir, "golden/memory")
		doGoldenTests(t, resultDir, []test{
			{name: "memory", path: "filters:format(png)/memory-test.png"},
			{name: "memory resize", path: "30x0/filters:format(png)/memory-test.png"},
		}, WithDebug(true), WithMaxAnimationFrames(-167))
	})
	t.Run("unsupported", func(t *testing.T) {
		loader := filestorage.New(testDataDir + "/../")
		app := imagor.New(
			imagor.WithLoaders(loader),
			imagor.WithUnsafe(true),
			imagor.WithDebug(true),
			imagor.WithLogger(zap.NewExample()),
			imagor.WithProcessors(NewProcessor(WithDebug(true))),
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

		w = httptest.NewRecorder()
		app.ServeHTTP(w, httptest.NewRequest(
			http.MethodGet, "/unsafe/meta/README.md", nil))
		assert.Equal(t, 406, w.Code)
		assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
	})
	t.Run("resolution exceeded", func(t *testing.T) {
		app := imagor.New(
			imagor.WithLoaders(filestorage.New(testDataDir)),
			imagor.WithUnsafe(true),
			imagor.WithDebug(true),
			imagor.WithLogger(zap.NewExample()),
			imagor.WithProcessors(NewProcessor(
				WithMaxResolution(300*300),
				WithDebug(true),
			)),
		)
		require.NoError(t, app.Startup(context.Background()))
		t.Cleanup(func() {
			assert.NoError(t, app.Shutdown(context.Background()))
		})
		w := httptest.NewRecorder()
		app.ServeHTTP(w, httptest.NewRequest(
			http.MethodGet, "/unsafe/gopher-front.png", nil))
		assert.Equal(t, 200, w.Code)

		w = httptest.NewRecorder()
		app.ServeHTTP(w, httptest.NewRequest(
			http.MethodGet, "/unsafe/gopher.png", nil))
		assert.Equal(t, 422, w.Code)

		w = httptest.NewRecorder()
		app.ServeHTTP(w, httptest.NewRequest(
			http.MethodGet, "/unsafe/trim/1000x0/gopher-front.png", nil))
		assert.Equal(t, 422, w.Code)
	})
	t.Run("resolution exceeded max frames within", func(t *testing.T) {
		app := imagor.New(
			imagor.WithLoaders(filestorage.New(testDataDir)),
			imagor.WithUnsafe(true),
			imagor.WithDebug(true),
			imagor.WithLogger(zap.NewExample()),
			imagor.WithProcessors(NewProcessor(
				WithMaxResolution(300*300),
				WithMaxAnimationFrames(3),
				WithDebug(true),
			)),
		)
		require.NoError(t, app.Startup(context.Background()))
		t.Cleanup(func() {
			assert.NoError(t, app.Shutdown(context.Background()))
		})
		w := httptest.NewRecorder()
		app.ServeHTTP(w, httptest.NewRequest(
			http.MethodGet, "/unsafe/dancing-banana.gif", nil))
		assert.Equal(t, 200, w.Code)
	})
	t.Run("resolution exceeded max frames", func(t *testing.T) {
		app := imagor.New(
			imagor.WithLoaders(filestorage.New(testDataDir)),
			imagor.WithUnsafe(true),
			imagor.WithDebug(true),
			imagor.WithLogger(zap.NewExample()),
			imagor.WithProcessors(NewProcessor(
				WithMaxResolution(300*300),
				WithMaxAnimationFrames(6),
				WithDebug(true),
			)),
		)
		require.NoError(t, app.Startup(context.Background()))
		t.Cleanup(func() {
			assert.NoError(t, app.Shutdown(context.Background()))
		})
		w := httptest.NewRecorder()
		app.ServeHTTP(w, httptest.NewRequest(
			http.MethodGet, "/unsafe/dancing-banana.gif", nil))
		assert.Equal(t, 422, w.Code)
	})
}

func doGoldenTests(t *testing.T, resultDir string, tests []test, opts ...Option) {
	resStorage := filestorage.New(resultDir,
		filestorage.WithSaveErrIfExists(true))
	fileLoader := filestorage.New(testDataDir)
	processor := NewProcessor(opts...)

	loader := loaderFunc(func(r *http.Request, image string) (blob *imagor.Blob, err error) {
		image, _ = fileLoader.Path(image)
		return imagor.NewBlob(func() (reader io.ReadCloser, size int64, err error) {
			// unknown size to force enable seek
			reader, err = os.Open(image)
			return
		}), nil
	})
	app := imagor.New(
		imagor.WithLoaders(loader, loaderFunc(func(r *http.Request, image string) (blob *imagor.Blob, err error) {
			if strings.HasPrefix(image, "memory-test") {
				return imagor.NewBlobFromMemory([]byte{
					255, 0, 0,
					0, 255, 0,
					0, 0, 255,
				}, 3, 1, 3), nil
			}
			return nil, imagor.ErrNotFound
		})),
		imagor.WithUnsafe(true),
		imagor.WithDebug(true),
		imagor.WithLogger(zap.NewExample()),
		imagor.WithProcessors(processor),
	)
	require.NoError(t, app.Startup(context.Background()))
	t.Cleanup(func() {
		assert.NoError(t, app.Shutdown(context.Background()))
	})
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			ctx, cancel := context.WithCancel(context.Background())
			req := httptest.NewRequest(
				http.MethodGet, fmt.Sprintf("/unsafe/%s", tt.path), nil).WithContext(ctx)
			app.ServeHTTP(w, req)
			cancel()
			assert.Equal(t, 200, w.Code)
			b := imagor.NewBlobFromBytes(w.Body.Bytes())
			_ = resStorage.Put(context.Background(), tt.path, b)
			path := filepath.Join(resultDir, imagorpath.Normalize(tt.path, nil))

			bc := imagor.NewBlobFromFile(path)
			buf, err := bc.ReadAll()
			require.NoError(t, err)
			if tt.checkTypeOnly {
				require.NotEqual(t, imagor.BlobTypeUnknown, b.BlobType())
				assert.Equal(t, bc.ContentType(), b.ContentType())
				assert.Equal(t, bc.BlobType(), b.BlobType())
				return
			}
			if reflect.DeepEqual(buf, w.Body.Bytes()) {
				return
			}
			img1, err := LoadImageFromBuffer(buf, nil)
			require.NoError(t, err)
			img2, err := LoadImageFromBuffer(w.Body.Bytes(), nil)
			require.NoError(t, err)
			require.Equal(t, img1.Width(), img2.Width(), "width mismatch")
			require.Equal(t, img1.Height(), img2.Height(), "height mismatch")
			buf1, err := img1.ExportWebp(nil)
			require.NoError(t, err)
			buf2, err := img2.ExportWebp(nil)
			require.NoError(t, err)
			require.True(t, reflect.DeepEqual(buf1, buf2), "image mismatch")
		})
	}
}

type loaderFunc func(r *http.Request, image string) (blob *imagor.Blob, err error)

func (f loaderFunc) Get(r *http.Request, image string) (*imagor.Blob, error) {
	return f(r, image)
}
