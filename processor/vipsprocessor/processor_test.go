package vipsprocessor

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/cshum/imagor"
	"github.com/cshum/imagor/imagorpath"
	"github.com/cshum/imagor/storage/filestorage"
	"github.com/cshum/vipsgen/vips"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
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
	arm64Golden   bool
}

func TestMain(m *testing.M) {
	vips.Startup(&vips.Config{
		ReportLeaks: true,
	})

	// Get initial memory stats
	var initialStats vips.MemoryStats
	vips.ReadVipsMemStats(&initialStats)

	// Force garbage collection before running tests
	runtime.GC()

	// Run the tests
	code := m.Run()

	runtime.GC()

	// Give some time for cleanup
	time.Sleep(100 * time.Millisecond)

	// Get final memory stats
	var finalStats vips.MemoryStats
	vips.ReadVipsMemStats(&finalStats)

	// Check for memory leaks
	memLeaked := finalStats.Mem > initialStats.Mem
	filesLeaked := finalStats.Files > initialStats.Files
	allocsLeaked := finalStats.Allocs > initialStats.Allocs

	if memLeaked || filesLeaked || allocsLeaked {
		fmt.Printf("MEMORY LEAK DETECTED!\n")
		fmt.Printf("Initial stats - Mem: %d, Files: %d, Allocs: %d\n",
			initialStats.Mem, initialStats.Files, initialStats.Allocs)
		fmt.Printf("Final stats   - Mem: %d, Files: %d, Allocs: %d\n",
			finalStats.Mem, finalStats.Files, finalStats.Allocs)
		fmt.Printf("Differences   - Mem: %+d, Files: %+d, Allocs: %+d\n",
			finalStats.Mem-initialStats.Mem,
			finalStats.Files-initialStats.Files,
			finalStats.Allocs-initialStats.Allocs)

		vips.Shutdown()
		os.Exit(1) // Exit with error code
	}

	fmt.Printf("No memory leaks detected.\n")
	fmt.Printf("Final stats - Mem: %d, Files: %d, Allocs: %d\n",
		finalStats.Mem, finalStats.Files, finalStats.Allocs)

	vips.Shutdown()
	os.Exit(code) // Exit with the test result code
}

func TestProcessor(t *testing.T) {
	v := NewProcessor(WithDebug(true))
	require.NoError(t, v.Startup(context.Background()))
	t.Cleanup(func() {
		stats := &vips.MemoryStats{}
		vips.ReadVipsMemStats(stats)
		fmt.Println(stats)
		require.NoError(t, v.Shutdown(context.Background()))
	})
	t.Run("vips basic", func(t *testing.T) {
		var resultDir = filepath.Join(testDataDir, "golden")
		doGoldenTests(t, resultDir, []test{
			{name: "png", path: "gopher-front.png"},
			{name: "jpeg", path: "fit-in/100x100/demo1.jpg"},
			{name: "webp", path: "fit-in/100x100/demo3.webp", arm64Golden: true},
			{name: "tiff", path: "fit-in/100x100/gopher.tiff"},
			{name: "avif", path: "fit-in/100x100/gopher-front.avif", checkTypeOnly: true},
			{name: "jxl", path: "fit-in/100x100/jxl-isobmff.jxl", checkTypeOnly: true},
			{name: "export gif", path: "filters:format(gif):quality(70)/gopher-front.png"},
			{name: "export webp", path: "filters:format(webp):quality(70)/gopher-front.png", arm64Golden: true},
			{name: "export tiff", path: "filters:format(tiff):quality(70)/gopher-front.png"},
			{name: "export jxl", path: "filters:format(jxl):quality(70)/gopher-front.png", checkTypeOnly: true},
			{name: "export avif", path: "filters:format(avif):quality(70)/gopher-front.png", checkTypeOnly: true},
			{name: "export heif", path: "filters:format(heif):quality(70)/gopher-front.png", checkTypeOnly: true},
		}, WithDebug(true), WithLogger(zap.NewExample()))
	})
	t.Run("meta", func(t *testing.T) {
		var resultDir = filepath.Join(testDataDir, "golden")
		doGoldenTests(t, resultDir, []test{
			{name: "meta jpeg", path: "meta/fit-in/100x100/demo1.jpg"},
			{name: "meta gif", path: "meta/fit-in/100x100/dancing-banana.gif"},
			{name: "base meta svg", path: "meta/test.svg"},
			{name: "base meta jp2", path: "meta/gopher.jp2"},
			{name: "base meta pdf", path: "meta/sample.pdf"},
			{name: "base meta heif", path: "meta/gopher-front.heif"},
			{name: "base meta tiff", path: "meta/gopher.tiff"},
			{name: "meta format no animate", path: "meta/fit-in/100x100/filters:format(jpg)/dancing-banana.gif"},
			{name: "meta exif", path: "meta/Canon_40D.jpg"},
			{name: "meta strip exif", path: "meta/filters:strip_exif()/Canon_40D.jpg"},
		}, WithDebug(true), WithLogger(zap.NewExample()))
	})
	t.Run("vips strip metadata config", func(t *testing.T) {
		var resultDir = filepath.Join(testDataDir, "golden")
		doGoldenTests(t, resultDir, []test{
			{name: "png", path: "fit-in/67x67/gopher-front.png"},
			{name: "jpeg", path: "fit-in/67x67/demo1.jpg"},
			{name: "webp", path: "fit-in/67x67/demo3.webp", arm64Golden: true},
			{name: "tiff", path: "fit-in/67x67/gopher.tiff", arm64Golden: true},
			{name: "tiff", path: "fit-in/67x67/dancing-banana.gif", arm64Golden: true},
			{name: "avif", path: "fit-in/67x67/gopher-front.avif", checkTypeOnly: true},
		}, WithDebug(true), WithStripMetadata(true), WithLogger(zap.NewExample()))
	})
	t.Run("vips strip_metadata filter", func(t *testing.T) {
		var resultDir = filepath.Join(testDataDir, "golden")
		doGoldenTests(t, resultDir, []test{
			{name: "png", path: "gopher-front.png"},
			{name: "jpeg", path: "fit-in/67x67/filters:strip_metadata()/demo1.jpg"},
			{name: "webp", path: "fit-in/67x67/filters:strip_metadata()/demo3.webp", arm64Golden: true},
			{name: "tiff", path: "fit-in/67x67/filters:strip_metadata()/gopher.tiff"},
			{name: "gif", path: "fit-in/67x67/filters:strip_metadata()/dancing-banana.gif", arm64Golden: true},
			{name: "avif", path: "fit-in/67x67/filters:strip_metadata()/gopher-front.avif", checkTypeOnly: true},
		}, WithDebug(true), WithLogger(zap.NewExample()))
	})
	t.Run("vips operations", func(t *testing.T) {
		var resultDir = filepath.Join(testDataDir, "golden")
		doGoldenTests(t, resultDir, []test{
			{name: "no-ops", path: "filters:background_color():round_corner():padding():rotate():proportion():proportion(9999):proportion(0.0000000001):proportion(-10)/gopher-front.png"},
			{name: "no-ops 2", path: "trim/filters:watermark():blur(2):sharpen(2):brightness():contrast():hue():saturation():rgb():modulate()/dancing-banana.gif"},
			{name: "no-ops 3", path: "filters:proportion():proportion(9999):proportion(0.0000000001):proportion(-10):sharpen(-1)/gopher-front.png"},
			{name: "resize center", path: "100x100/filters:quality(70):format(jpeg)/gopher.png"},
			{name: "resize smart", path: "100x100/smart/filters:autojpg()/gopher.png"},
			{name: "resize focal", path: "300x100/filters:fill(white):format(jpeg):focal(589x401:1000x814)/gopher.png"},
			{name: "resize focal vertical", path: "100x300/filters:fill(white):format(jpeg):focal(589x401:1000x814)/gopher.png"},
			{name: "resize focal with crop", path: "0x100:9999x9999/300x100/filters:fill(white):format(jpeg):focal(589x401:1000x814)/gopher.png"},
			{name: "resize focal float", path: "300x100/filters:fill(white):format(jpeg):focal(0.35x0.25:0.6x0.3)/gopher.png"},
			{name: "resize focal point", path: "300x100/filters:fill(white):format(jpeg):focal(589x401):focal(1000x814)/gopher.png"},
			{name: "resize focal point edge", path: "300x100/filters:fill(white):format(jpeg):focal(9999x9999)/gopher.png"},
			{name: "resize focal point exif orientation cw90", path: "300x300/filters:format(jpeg):focal(150:150)/gopher-exif-orientation-cw90.png"},
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
			{name: "resize orient", path: "100x200/left/filters:orient(90)/gopher.png"},
			{name: "png params", path: "200x200/filters:format(png):palette():bitdepth(4):compression(8)/gopher.png", arm64Golden: true},
			{name: "fit-in unspecified height", path: "fit-in/50x0/filters:fill(white):format(jpg)/Canon_40D.jpg"},
			{name: "resize unspecified height", path: "50x0/filters:fill(white):format(jpg)/Canon_40D.jpg"},
			{name: "fit-in unspecified width", path: "fit-in/0x50/filters:fill(white):format(jpg)/Canon_40D.jpg"},
			{name: "resize unspecified width", path: "0x50/filters:fill(white):format(jpg)/Canon_40D.jpg"},
			{name: "resize with no_upscale", path: "500x400/filters:no_upscale()/gopher-front.png"},
			{name: "resize with no_upscale unspecified height", path: "500x0/filters:no_upscale()/gopher-front.png"},
			{name: "resize with no_upscale cropped", path: "500x200/filters:no_upscale()/gopher-front.png"},
			{name: "adaptive-fit-in landscape to portrait", path: "adaptive-fit-in/100x200/gopher.png"},
			{name: "adaptive-fit-in portrait to landscape", path: "adaptive-fit-in/200x100/gopher-front.png"},
			{name: "adaptive-fit-in same orientation", path: "adaptive-fit-in/200x100/gopher.png"},
			{name: "adaptive-fit-in with filters", path: "adaptive-fit-in/200x100/filters:fill(white):format(jpeg)/gopher-front.png"},
			{name: "full-fit-in basic", path: "full-fit-in/300x200/gopher.png"},
			{name: "full-fit-in vertical", path: "full-fit-in/200x300/gopher-front.png"},
			{name: "full-fit-in with smart", path: "full-fit-in/300x200/smart/gopher.png"},
			{name: "full-fit-in upscale", path: "full-fit-in/500x400/gopher-front.png"},
			{name: "full-fit-in rounding precision", path: "full-fit-in/74x11/gopher-front.png"},
			{name: "adaptive-full-fit-in combined", path: "adaptive-full-fit-in/100x200/gopher.png"},
			{name: "adaptive-full-fit-in with filters", path: "adaptive-full-fit-in/200x100/filters:fill(yellow):format(jpeg)/gopher-front.png"},
			{name: "adaptive-full-fit-in upscale", path: "adaptive-full-fit-in/500x400/gopher-front.png"},
			{name: "stretch", path: "stretch/100x100/filters:modulate(-10,30,20)/gopher.png"},
			{name: "fit-in flip hue", path: "fit-in/-200x0/filters:hue(290):saturation(100):fill(FFO):upscale()/gopher.png"},
			{name: "fit-in padding", path: "fit-in/100x100/10x5/filters:fill(white)/gopher.png"},
			{name: "fit-in padding transparent", path: "fit-in/100x100/10x5/filters:fill(none)/gopher.png"},
			{name: "fit-in padding transparent non-alpha", path: "fit-in/100x120/10x5/filters:fill(none):format(png)/demo1.jpg"},
			{name: "fit-in stretch padding transparent", path: "fit-in/stretch/100x100/10x10/filters:fill(transparent)/gopher.png"},
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
			{name: "grayscale fill none", path: "fit-in/100x100/filters:fill(none)/2bands.png", checkTypeOnly: true},
			{name: "trim alpha", path: "trim/find_trim_alpha.png"},
			{name: "trim with crop", path: "trim:bottom-right/50x50:0x0/find_trim.png"},
			{name: "trim right", path: "trim:bottom-right/500x500/filters:strip_exif():upscale():no_upscale()/find_trim.png"},
			{name: "trim upscale", path: "trim/fit-in/1000x1000/filters:upscale():strip_icc()/find_trim.png"},
			{name: "trim tolerance", path: "trim:50/500x500/filters:stretch()/find_trim.png"},
			{name: "trim position tolerance filter", path: "50x50:0x0/filters:trim(50,bottom-right)/find_trim.png"},
			{name: "trim filter", path: "/fit-in/100x100/filters:fill(auto):trim(50)/find_trim.png"},
			{name: "watermark", path: "fit-in/500x500/filters:fill(white):watermark(gopher.png,10p,repeat,30,20,20):watermark(gopher.png,repeat,bottom,30,30,30):watermark(gopher-front.png,center,-10p)/gopher.png", arm64Golden: true},
			{name: "watermark base64encoded", path: "fit-in/500x500/filters:fill(white):watermark(b64:Z29waGVyLnBuZw,10p,repeat,30,20,20):watermark(gopher.png,repeat,bottom,30,30,30):watermark(gopher-front.png,center,-10p)/gopher.png", arm64Golden: true},
			{name: "watermark non alpha", path: "filters:watermark(demo1.jpg,repeat,repeat,40,25,50)/demo1.jpg", arm64Golden: true},
			{name: "background color non alpha", path: "filters:background_color(yellow)/demo1.jpg"},
			{name: "watermark 2 bands", path: "filters:watermark(2bands.png,repeat,bottom,40,25,50)/demo1.jpg", arm64Golden: true},
			{name: "watermark float", path: "fit-in/500x500/filters:fill(white):watermark(gopher.png,0.1,repeat,30,20,20):watermark(gopher.png,repeat,bottom,30,30,30):watermark(gopher-front.png,center,-0.1)/gopher.png", arm64Golden: true},
			{name: "watermark align", path: "fit-in/500x500/filters:fill(white):watermark(gopher.png,left,top,30,20,20):watermark(gopher.png,right,center,30,30,30):watermark(gopher-front.png,-20,-10)/gopher.png"},
			{name: "image left offset", path: "fit-in/500x500/filters:fill(white):image(gopher-front.png,left-20,top-10)/gopher.png"},
			{name: "image right offset", path: "fit-in/500x500/filters:fill(white):image(gopher-front.png,right-30,bottom-20)/gopher.png"},
			{name: "image shorthand l t", path: "fit-in/500x500/filters:fill(white):image(gopher-front.png,l-20,t-10)/gopher.png"},
			{name: "image shorthand r b", path: "fit-in/500x500/filters:fill(white):image(gopher-front.png,r-30,b-20)/gopher.png"},

			{name: "image default position", path: "fit-in/500x500/filters:image(/100x100/gopher-front.png)/gopher.png"},
			{name: "image center", path: "fit-in/500x500/filters:image(/100x100/gopher-front.png,center,center)/gopher.png"},
			{name: "image outside rotate", path: "fit-in/500x500/filters:rotate(90):image(/100x100/gopher-front.png,center,center)/gopher.png"},
			{name: "image inside rotate", path: "fit-in/500x500/filters:image(/100x100/filters:rotate(90)/gopher-front.png,center,center)/gopher.png"},
			{name: "image with alpha", path: "fit-in/500x500/filters:image(/100x100/gopher-front.png,center,center,50)/gopher.png"},
			{name: "image with mask blend mode", path: "fit-in/500x500/filters:image(/100x100/gopher-front.png,center,center,0,mask)/gopher.png"},
			{name: "image with invalid blend mode fallback", path: "fit-in/500x500/filters:image(/100x100/gopher-front.png,center,center,50,invalid-mode)/gopher.png"},
			{name: "image with multiply blend mode", path: "fit-in/500x500/filters:image(/100x100/gopher-front.png,center,center,30,multiply)/gopher.png"},
			{name: "image negative position", path: "fit-in/500x500/filters:image(/100x100/gopher-front.png,-10,-10)/gopher.png"},
			{name: "image repeat", path: "fit-in/300x300/filters:image(/50x50/gopher-front.png,repeat,repeat)/gopher.png", arm64Golden: true},
			{name: "image nested single", path: "fit-in/500x500/filters:image(/150x150/filters:image(/50x50/gopher-front.png,center,center)/gopher.png,10,10)/demo1.jpg", arm64Golden: true},
			{name: "image nested double", path: "fit-in/500x500/filters:image(/200x200/filters:image(/100x100/filters:image(/50x50/gopher-front.png,center,center)/gopher.png,center,center)/demo1.jpg,center,center)/gopher.png", arm64Golden: true},
			{name: "image nested with transforms", path: "filters:image(/150x150/filters:grayscale():image(/50x50/filters:rotate(90)/gopher-front.png,center,center)/gopher.png,center,center)/demo1.jpg", arm64Golden: true},

			// Overlay cropping edge cases - tests transformOverlay boundary logic
			{name: "image overlay crop right edge", path: "fit-in/300x300/filters:image(/100x100/gopher-front.png,250,50)/gopher.png"},
			{name: "image overlay crop bottom edge", path: "fit-in/300x300/filters:image(/100x100/gopher-front.png,50,250)/gopher.png"},
			{name: "image overlay crop left edge", path: "fit-in/300x300/filters:image(/100x100/gopher-front.png,-50,50)/gopher.png"},
			{name: "image overlay crop top edge", path: "fit-in/300x300/filters:image(/100x100/gopher-front.png,50,-50)/gopher.png"},
			{name: "image overlay outside bounds", path: "fit-in/300x300/filters:image(/100x100/gopher-front.png,400,50)/gopher.png"},
			{name: "image overlay outside bounds far right", path: "fit-in/300x300/filters:image(/100x100/gopher-front.png,5000,0)/gopher.png"},
			{name: "image overlay outside bounds far below", path: "fit-in/300x300/filters:image(/100x100/gopher-front.png,0,5000)/gopher.png"},
			{name: "image overlay outside bounds far left", path: "fit-in/300x300/filters:image(/100x100/gopher-front.png,-5000,0)/gopher.png"},
			{name: "image overlay outside bounds far above", path: "fit-in/300x300/filters:image(/100x100/gopher-front.png,0,-5000)/gopher.png"},
			{name: "image overlay center child larger than parent", path: "fit-in/100x100/filters:fill(yellow):image(/fit-in/150x150/filters:grayscale()/gopher-front.png,center,center)/dancing-banana.gif"},

			{name: "original no animate", path: "filters:fill(white):format(jpeg)/dancing-banana.gif"},
			{name: "original animated", path: "dancing-banana.gif"},
			{name: "original animated quality", path: "filters:quality(60)/dancing-banana.gif"},
			{name: "original animated max_frames", path: "filters:max_frames(3)/dancing-banana.gif"},
			{name: "original animated page", path: "filters:page(5)/dancing-banana.gif"},
			{name: "original animated page exceeded", path: "filters:page(999)/dancing-banana.gif"},
			{name: "original animated strip_exif retain metadata", path: "filters:strip_exif()/dancing-banana.gif"},
			{name: "rotate animated", path: "fit-in/100x150/filters:rotate(90):fill(yellow)/dancing-banana.gif", arm64Golden: true},
			{name: "crop animated", path: "30x20:100x150/dancing-banana.gif"},
			{name: "crop-percent animated", path: "0.1x0.2:0.89x0.72/dancing-banana.gif"},
			{name: "focal region animated", path: "100x30/filters:focal(0.1x0:0.89x0.72)/dancing-banana.gif"},
			{name: "focal point animated", path: "100x30/filters:focal(0.89x0.72)/dancing-banana.gif", arm64Golden: true},
			{name: "padding", path: "fit-in/-180x180/10x10/filters:fill(yellow):padding(white,10,20,30,40):format(jpeg)/gopher.png"},
			{name: "rotate fill", path: "fit-in/100x210/10x20:15x3/filters:rotate(90):fill(yellow)/gopher-front.png"},
			{name: "resize center animated", path: "100x100/dancing-banana.gif", arm64Golden: true},
			{name: "resize top animated", path: "200x100/top/dancing-banana.gif", arm64Golden: true},
			{name: "resize top animated", path: "200x100/right/top/dancing-banana.gif", arm64Golden: true},
			{name: "resize bottom animated", path: "200x100/bottom/dancing-banana.gif", arm64Golden: true},
			{name: "resize bottom animated", path: "200x100/left/bottom/dancing-banana.gif", arm64Golden: true},
			{name: "resize left animated", path: "100x200/left/dancing-banana.gif", arm64Golden: true},
			{name: "resize left animated", path: "100x200/left/bottom/dancing-banana.gif", arm64Golden: true},
			{name: "resize right animated", path: "100x200/right/dancing-banana.gif", arm64Golden: true},
			{name: "resize right animated", path: "100x200/right/top/dancing-banana.gif", arm64Golden: true},
			{name: "stretch animated", path: "stretch/100x200/dancing-banana.gif", arm64Golden: true},
			{name: "resize padding animated", path: "100x100/10x5/top/filters:fill(yellow)/dancing-banana.gif", arm64Golden: true},
			{name: "watermark animated", path: "fit-in/200x150/filters:fill(yellow):watermark(gopher-front.png,repeat,bottom,0,30,30)/dancing-banana.gif", arm64Golden: true},
			{name: "watermark animated align bottom right", path: "fit-in/200x150/filters:fill(yellow):watermark(gopher-front.png,-20,-10,0,30,30)/dancing-banana.gif", arm64Golden: true},
			{name: "watermark double animated", path: "fit-in/200x150/filters:fill(yellow):watermark(dancing-banana.gif,-20,-10,0,30,30):watermark(nyan-cat.gif,0,10,0,40,30)/dancing-banana.gif", arm64Golden: true},
			{name: "watermark double animated 2", path: "fit-in/200x150/filters:fill(yellow):watermark(dancing-banana.gif,30,-10,0,40,40):watermark(dancing-banana.gif,0,10,0,40,40)/nyan-cat.gif", arm64Golden: true},
			{name: "padding with watermark double animated", path: "200x0/20x20:100x20/filters:fill(yellow):watermark(dancing-banana.gif,-10,-10,0,50,50):watermark(dancing-banana.gif,-30,10,0,50,50)/nyan-cat.gif", arm64Golden: true},
			{name: "watermark repeated animated", path: "fit-in/200x150/filters:fill(cyan):watermark(dancing-banana.gif,repeat,bottom,0,50,50)/dancing-banana.gif", arm64Golden: true},
			{name: "animated fill round_corner", path: "filters:fill(cyan):round_corner(60)/dancing-banana.gif"},
			{name: "label", path: "fit-in/300x200/10x10/filters:fill(yellow):label(IMAGOR,15,10,30,blue,30)/gopher-front.png", arm64Golden: true},
			{name: "label top left", path: "fit-in/300x200/10x10/filters:fill(yellow):label(IMAGOR,left,top,30,red,30)/gopher-front.png", arm64Golden: true},
			{name: "label right center", path: "fit-in/300x200/10x10/filters:fill(yellow):label(IMAGOR,right,center,30,red,30)/gopher-front.png", arm64Golden: true},
			{name: "label center bottom", path: "fit-in/300x200/10x10/filters:fill(yellow):label(IMAGOR,center,bottom,30,red,30)/gopher-front.png", arm64Golden: true},
			{name: "label negative", path: "fit-in/300x200/10x10/filters:fill(yellow):label(IMAGOR,-15,-10,30,red,30)/gopher-front.png", arm64Golden: true},
			{name: "label percentage", path: "fit-in/300x200/10x10/filters:fill(yellow):label(IMAGOR,-15p,10p,30,red,30)/gopher-front.png", arm64Golden: true},
			{name: "label float", path: "fit-in/300x200/10x10/filters:fill(yellow):label(IMAGOR,-0.15,0.1,30,red,30)/gopher-front.png", arm64Golden: true},
			{name: "label left offset", path: "fit-in/300x200/10x10/filters:fill(yellow):label(IMAGOR,left-20,top-10,30,blue,30)/gopher-front.png", arm64Golden: true},
			{name: "label right offset", path: "fit-in/300x200/10x10/filters:fill(yellow):label(IMAGOR,right-30,bottom-20,30,green,30)/gopher-front.png", arm64Golden: true},
			{name: "label shorthand l t", path: "fit-in/300x200/10x10/filters:fill(yellow):label(IMAGOR,l-20,t-10,30,blue,30)/gopher-front.png", arm64Golden: true},
			{name: "label shorthand r b", path: "fit-in/300x200/10x10/filters:fill(yellow):label(IMAGOR,r-30,b-20,30,green,30)/gopher-front.png", arm64Golden: true},
			{name: "label animated", path: "fit-in/150x200/10x00:10x50/filters:fill(yellow):label(IMAGOR,center,-30,25,black)/dancing-banana.gif", arm64Golden: true},
			{name: "label animated with font", path: "fit-in/150x200/10x00:10x50/filters:fill(cyan):label(IMAGOR,center,-30,25,white,0,monospace)/dancing-banana.gif", arm64Golden: true},
			{name: "label grayscale", path: "fit-in/filters:label(imagor,-1,0,50)/2bands.png", checkTypeOnly: true},
			{name: "strip exif", path: "filters:strip_exif()/Canon_40D.jpg"},
			{name: "bmp 24bit", path: "100x100/bmp_24.bmp"},
			{name: "bmp 8bit", path: "100x100/lena_gray.bmp"},
			{name: "svg", path: "test.svg", checkTypeOnly: true},
			{name: "crop absolute", path: "300x300/filters:crop(50,50,200,200)/gopher.png"},
			{name: "crop relative", path: "300x300/filters:crop(0.1,0.1,0.8,0.8)/gopher.png"},
			{name: "crop overflow", path: "300x300/filters:crop(250,250,200,200)/gopher.png"},
			{name: "crop animated", path: "200x200/filters:crop(20,20,160,160)/dancing-banana.gif", arm64Golden: true},
			{name: "crop with fill", path: "400x400/filters:fill(yellow):crop(50,50,300,300)/gopher.png"},
			{name: "strip icc", path: "200x200/filters:strip_icc():to_colorspace()/jpg-24bit-icc-adobe-rgb.jpg"},
			{name: "to colorspace", path: "200x200/filters:to_colorspace(cmyk)/jpg-24bit-icc-adobe-rgb.jpg"},
		}, WithDebug(true), WithLogger(zap.NewExample()), WithForceBmpFallback())
	})
	t.Run("max frames", func(t *testing.T) {
		var resultDir = filepath.Join(testDataDir, "golden/max-frames")
		doGoldenTests(t, resultDir, []test{
			{name: "original", path: "gopher-front.png"},
			{name: "original no animate", path: "filters:fill(white):format(jpeg)/dancing-banana.gif"},
			{name: "original animated", path: "dancing-banana.gif"},
			{name: "original animated trim no-op", path: "trim/dancing-banana.gif"},
			{name: "crop animated", path: "30x20:100x150/dancing-banana.gif"},
			{name: "resize top animated", path: "200x100/top/dancing-banana.gif", arm64Golden: true},
			{name: "watermark repeated animated", path: "fit-in/200x150/filters:fill(cyan):watermark(dancing-banana.gif,repeat,bottom,0,50,50)/dancing-banana.gif", arm64Golden: true},
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
			{name: "resize top animated", path: "200x100/top/dancing-banana.gif", arm64Golden: true},
			{name: "watermark repeated animated", path: "fit-in/200x150/filters:fill(cyan):watermark(dancing-banana.gif,repeat,bottom,0,50,50)/dancing-banana.gif", arm64Golden: true},
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
			http.MethodGet, "/unsafe/1000x1000/gopher-front.png", nil))
		assert.Equal(t, 422, w.Code)

		w = httptest.NewRecorder()
		app.ServeHTTP(w, httptest.NewRequest(
			http.MethodGet, "/unsafe/gopher.png", nil))
		assert.Equal(t, 422, w.Code)

		w = httptest.NewRecorder()
		app.ServeHTTP(w, httptest.NewRequest(
			http.MethodGet, "/unsafe/1000x0/gopher-front.png", nil))
		assert.Equal(t, 422, w.Code)
	})

	t.Run("resolution exceeded bmp", func(t *testing.T) {
		app := imagor.New(
			imagor.WithLoaders(filestorage.New(testDataDir)),
			imagor.WithUnsafe(true),
			imagor.WithDebug(true),
			imagor.WithLogger(zap.NewExample()),
			imagor.WithProcessors(NewProcessor(
				WithMaxResolution(150*150),
				WithDebug(true),
			)),
		)
		require.NoError(t, app.Startup(context.Background()))
		t.Cleanup(func() {
			assert.NoError(t, app.Shutdown(context.Background()))
		})
		w := httptest.NewRecorder()
		app.ServeHTTP(w, httptest.NewRequest(
			http.MethodGet, "/unsafe/100x100/bmp_24.bmp", nil))
		assert.Equal(t, 422, w.Code)
	})
	t.Run("resolution exceeded bmp 2", func(t *testing.T) {
		app := imagor.New(
			imagor.WithLoaders(filestorage.New(testDataDir)),
			imagor.WithUnsafe(true),
			imagor.WithDebug(true),
			imagor.WithLogger(zap.NewExample()),
			imagor.WithProcessors(NewProcessor(
				WithMaxHeight(199),
				WithDebug(true),
			)),
		)
		require.NoError(t, app.Startup(context.Background()))
		t.Cleanup(func() {
			assert.NoError(t, app.Shutdown(context.Background()))
		})
		w := httptest.NewRecorder()
		app.ServeHTTP(w, httptest.NewRequest(
			http.MethodGet, "/unsafe/100x100/bmp_24.bmp", nil))
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
	t.Run("invalid BMP", func(t *testing.T) {
		ctx := context.Background()
		blob := imagor.NewBlobFromBytes([]byte("BMabcdasdfasdfasdfasdfasdfasdfasdfasdfasdfasdf"))
		assert.Equal(t, imagor.BlobTypeBMP, blob.BlobType())
		p := NewProcessor(
			WithDebug(true),
		)
		img, err := p.Process(ctx, blob, imagorpath.Params{}, nil)
		assert.Empty(t, img)
		assert.Error(t, err)
	})
}

func doGoldenTests(t *testing.T, resultDir string, tests []test, opts ...Option) {
	resStorage := filestorage.New(resultDir,
		filestorage.WithSaveErrIfExists(true))
	resultDirArm64 := strings.ReplaceAll(resultDir, "/golden", "/golden_arm64")
	resStorageArm64 := filestorage.New(resultDirArm64, filestorage.WithSaveErrIfExists(true))
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
			var path string
			path = tt.path
			if strings.HasPrefix(path, "meta/") {
				path += ".json"
			}
			if tt.arm64Golden && runtime.GOARCH == "arm64" {
				_ = resStorageArm64.Put(context.Background(), path, b)
				path = filepath.Join(resultDirArm64, imagorpath.Normalize(path, nil))
			} else {
				_ = resStorage.Put(context.Background(), path, b)
				path = filepath.Join(resultDir, imagorpath.Normalize(path, nil))
			}
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
			img1, err := vips.NewImageFromBuffer(buf, nil)
			require.NoError(t, err)
			defer img1.Close()
			img2, err := vips.NewImageFromBuffer(w.Body.Bytes(), nil)
			require.NoError(t, err)
			defer img2.Close()
			require.Equal(t, img1.Width(), img2.Width(), "width mismatch")
			require.Equal(t, img1.Height(), img2.Height(), "height mismatch")
			buf1, err := img1.WebpsaveBuffer(nil)
			require.NoError(t, err)
			buf2, err := img2.WebpsaveBuffer(nil)
			require.NoError(t, err)
			require.True(t, reflect.DeepEqual(buf1, buf2), "image mismatch")
		})
	}
}

type loaderFunc func(r *http.Request, image string) (blob *imagor.Blob, err error)

func (f loaderFunc) Get(r *http.Request, image string) (*imagor.Blob, error) {
	return f(r, image)
}
