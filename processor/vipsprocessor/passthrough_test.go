package vipsprocessor

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
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

// ptSVG is a document exercising the constructs that must be dropped, next to
// presentation that must survive.
//
// It contains external references, which librsvg tries to fetch when the
// document is rasterized - keep it out of tests that rasterize more than once
// (see ptSimpleSVG).
const ptSVG = `<svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" width="64" height="64" viewBox="0 0 64 64" onload="alert(1)">
  <script>alert('xss')</script>
  <style>@import url(https://evil.test/x.css);</style>
  <foreignObject><iframe src="https://evil.test/"></iframe></foreignObject>
  <image xlink:href="https://evil.test/track.png" width="8" height="8"/>
  <use href="javascript:alert(3)"/>
  <rect width="64" height="64" fill="url(https://evil.test/p.svg#g)" onclick="alert(2)"/>
  <rect id="ok" x="4" y="4" width="8" height="8" fill="#0af"/>
  <text x="8" y="56" font-size="10" fill="#333">kept</text>
</svg>`

// ptSimpleSVG is a self-contained document with nothing external to fetch.
const ptSimpleSVG = `<svg xmlns="http://www.w3.org/2000/svg" width="64" height="64" viewBox="0 0 64 64">
  <rect width="64" height="64" fill="#0af"/>
  <circle cx="32" cy="32" r="12" fill="#fff"/>
</svg>`

// ptLoader serves in-memory documents, falling back to the testdata directory.
type ptLoader struct {
	mem map[string][]byte
}

func (l *ptLoader) Get(r *http.Request, key string) (*imagor.Blob, error) {
	if b, ok := l.mem[key]; ok {
		return imagor.NewBlobFromBytes(b), nil
	}
	return filestorage.New(testDataDir).Get(r, key)
}

func (l *ptLoader) Stat(_ context.Context, _ string) (*imagor.Stat, error) {
	return nil, imagor.ErrNotFound
}

func (l *ptLoader) Put(_ context.Context, _ string, _ *imagor.Blob) error { return nil }

func (l *ptLoader) Delete(_ context.Context, _ string) error { return nil }

// ptKeepAliveOnce holds a processor that is never shut down, so the vips
// refcount inside the processor never reaches zero during the test binary.
//
// vips.Shutdown() followed by vips.Startup() leaves the next rasterization to
// segfault on some libvips builds (reproduced locally on 8.18.2; the repo
// targets 8.18.6, where the existing suites cycle apps without trouble). Tests
// are not the right place to depend on that cycle.
var ptKeepAliveOnce sync.Once

func ptKeepVipsAlive() {
	ptKeepAliveOnce.Do(func() {
		_ = NewProcessor().Startup(context.Background())
	})
}

// ptApp builds an app whose loader serves the given documents plus testdata.
func ptApp(t *testing.T, mem map[string][]byte, appOpts ...imagor.Option) *imagor.Imagor {
	t.Helper()
	return ptAppWith(t, mem, nil, appOpts...)
}

func ptAppWith(
	t *testing.T, mem map[string][]byte, procOpts []Option, appOpts ...imagor.Option,
) *imagor.Imagor {
	t.Helper()
	ptKeepVipsAlive()
	processor := NewProcessor(procOpts...)
	app := imagor.New(append([]imagor.Option{
		imagor.WithLoaders(&ptLoader{mem: mem}),
		imagor.WithUnsafe(true),
		imagor.WithProcessors(processor),
		imagor.WithLogger(zap.NewNop()),
	}, appOpts...)...)
	require.NoError(t, app.Startup(context.Background()))
	t.Cleanup(func() {
		_ = app.Shutdown(context.Background())
	})
	return app
}

func ptGet(t *testing.T, app *imagor.Imagor, path string, header http.Header) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/unsafe/"+path, nil)
	for k, v := range header {
		req.Header[k] = v
	}
	app.ServeHTTP(w, req)
	return w
}

func TestPassthroughServesSanitizedSVG(t *testing.T) {
	app := ptApp(t, map[string][]byte{"hostile.svg": []byte(ptSVG)},
		imagor.WithPassthroughFormats(imagor.BlobTypeSVG))

	res := ptGet(t, app, "hostile.svg", nil)
	body := res.Body.String()

	require.Equal(t, http.StatusOK, res.Code)
	assert.Equal(t, "image/svg+xml", res.Header().Get("Content-Type"))
	assert.Equal(t,
		"default-src 'none'; style-src 'unsafe-inline'; img-src data:",
		res.Header().Get("Content-Security-Policy"))
	assert.Equal(t, "nosniff", res.Header().Get("X-Content-Type-Options"))

	for _, leaked := range []string{
		"script", "alert(", "onload", "onclick", "evil.test", "javascript:",
		"foreignObject", "<style", "@import",
	} {
		assert.NotContains(t, body, leaked, "passthrough leaked %q:\n%s", leaked, body)
	}
	for _, kept := range []string{
		`viewBox="0 0 64 64"`, `fill="#0af"`, `>kept<`,
	} {
		assert.Contains(t, body, kept, "passthrough lost %q:\n%s", kept, body)
	}

	// The sanitized document must still be something librsvg renders, at the
	// original size - a sanitizer that breaks valid SVG is not shippable.
	img, err := vips.NewSvgloadBuffer([]byte(body), nil)
	require.NoError(t, err, "sanitized svg does not load: %s", body)
	defer img.Close()
	assert.Equal(t, 64, img.Width())
	assert.Equal(t, 64, img.Height())
}

func TestPassthroughDisabledRasterizes(t *testing.T) {
	app := ptApp(t, map[string][]byte{"simple.svg": []byte(ptSimpleSVG)})

	res := ptGet(t, app, "simple.svg", nil)
	require.Equal(t, http.StatusOK, res.Code)
	// Unchanged default behaviour: an SVG source with no format requested is
	// re-encoded to JPEG.
	assert.Equal(t, "image/jpeg", res.Header().Get("Content-Type"))
	assert.Equal(t, "", res.Header().Get("Content-Security-Policy"))
}

func TestPassthroughSanitizeDisabledPreservesBytes(t *testing.T) {
	app := ptAppWith(t, map[string][]byte{"hostile.svg": []byte(ptSVG)},
		[]Option{WithSanitizeSVG(false)},
		imagor.WithPassthroughFormats(imagor.BlobTypeSVG))

	res := ptGet(t, app, "hostile.svg", nil)
	require.Equal(t, http.StatusOK, res.Code)
	assert.Equal(t, "image/svg+xml", res.Header().Get("Content-Type"))
	assert.Equal(t, ptSVG, res.Body.String(), "expected the source document byte for byte")
	// Even unsanitized, the response headers still have to be set.
	assert.Equal(t, "nosniff", res.Header().Get("X-Content-Type-Options"))
}

// TestPassthroughRequiresNoOpRequest is the guard against silently dropping a
// transformation: only a request that asks for nothing may pass through.
func TestPassthroughRequiresNoOpRequest(t *testing.T) {
	for _, tc := range []struct{ name, path string }{
		{"resize", "100x100/simple.svg"},
		{"resize width only", "100x/simple.svg"},
		{"fit-in", "fit-in/100x100/simple.svg"},
		{"stretch", "stretch/100x100/simple.svg"},
		{"smart", "smart/simple.svg"},
		{"trim", "trim/simple.svg"},
		{"crop", "100x100:100x100/simple.svg"},
		{"rotate filter", "filters:rotate(90)/simple.svg"},
		{"crop filter", "filters:crop(0,0,10,10)/simple.svg"},
		{"format png", "filters:format(png)/simple.svg"},
		{"format webp", "filters:format(webp)/simple.svg"},
		{"quality filter", "filters:quality(10)/simple.svg"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			app := ptApp(t, map[string][]byte{"simple.svg": []byte(ptSimpleSVG)},
				imagor.WithPassthroughFormats(imagor.BlobTypeSVG))
			res := ptGet(t, app, tc.path, nil)
			require.Equal(t, http.StatusOK, res.Code, res.Body.String())
			assert.NotEqual(t, "image/svg+xml", res.Header().Get("Content-Type"),
				"%s must not pass through untouched", tc.name)
		})
	}
}

// TestPassthroughMarkerCannotDropAnOperation covers a crafted path: the marker
// is only appended to no-op requests, but nothing stops a client or a URL
// generator from writing it next to an operation. The operation wins.
func TestPassthroughMarkerCannotDropAnOperation(t *testing.T) {
	app := ptApp(t, map[string][]byte{"simple.svg": []byte(ptSimpleSVG)},
		imagor.WithPassthroughFormats(imagor.BlobTypeSVG))

	for _, tc := range []struct{ name, path string }{
		{"resize", "100x100/filters:passthrough()/simple.svg"},
		{"rotate", "filters:passthrough():rotate(90)/simple.svg"},
		{"crop", "filters:passthrough():crop(0,0,10,10)/simple.svg"},
		{"quality", "filters:passthrough():quality(10)/simple.svg"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			res := ptGet(t, app, tc.path, nil)
			require.Equal(t, http.StatusOK, res.Code, res.Body.String())
			assert.NotEqual(t, "image/svg+xml", res.Header().Get("Content-Type"),
				"a crafted marker must not drop the operation")
		})
	}
}

// TestPassthroughMarkerWithFormatFilter documents the one contradictory
// combination: the marker says "serve the source", a format filter says
// "convert it".
//
// Passthrough wins, because the marker only ever accompanies a request whose
// format filter was injected by content negotiation - if a named format beat the
// marker, auto WebP/AVIF would disable passthrough for exactly the clients it
// was enabled for. A request that wants a raster format must not carry the
// marker, and the application never adds one when the client names a format.
func TestPassthroughMarkerWithFormatFilter(t *testing.T) {
	app := ptApp(t, map[string][]byte{"simple.svg": []byte(ptSimpleSVG)},
		imagor.WithPassthroughFormats(imagor.BlobTypeSVG))

	res := ptGet(t, app, "filters:passthrough():format(webp)/simple.svg", nil)
	require.Equal(t, http.StatusOK, res.Code, res.Body.String())
	assert.Equal(t, "image/svg+xml", res.Header().Get("Content-Type"))

	// Without the marker the format is honoured, which is what a client that
	// wants WebP gets for naming it.
	res = ptGet(t, app, "filters:format(webp)/simple.svg", nil)
	require.Equal(t, http.StatusOK, res.Code)
	assert.Equal(t, "image/webp", res.Header().Get("Content-Type"))
}

// TestPassthroughExplicitSVGWithTransformation checks that naming svg together
// with a transformation fails loudly rather than serving the untouched vector.
func TestPassthroughExplicitSVGWithTransformation(t *testing.T) {
	app := ptApp(t, map[string][]byte{"simple.svg": []byte(ptSimpleSVG)},
		imagor.WithPassthroughFormats(imagor.BlobTypeSVG))

	res := ptGet(t, app, "100x100/filters:format(svg)/simple.svg", nil)
	require.Equal(t, http.StatusBadRequest, res.Code, res.Body.String())

	// Naming svg while passthrough is off is equally unanswerable.
	app = ptApp(t, map[string][]byte{"simple.svg": []byte(ptSimpleSVG)})
	res = ptGet(t, app, "filters:format(svg)/simple.svg", nil)
	require.Equal(t, http.StatusBadRequest, res.Code, res.Body.String())
}

// TestPassthroughResizeStillApplies proves the resize is actually performed,
// not just that the response is no longer an SVG.
func TestPassthroughResizeStillApplies(t *testing.T) {
	app := ptApp(t, map[string][]byte{"simple.svg": []byte(ptSimpleSVG)},
		imagor.WithPassthroughFormats(imagor.BlobTypeSVG))

	res := ptGet(t, app, "24x24/simple.svg", nil)
	require.Equal(t, http.StatusOK, res.Code)
	require.Equal(t, "image/jpeg", res.Header().Get("Content-Type"))

	img, err := vips.NewJpegloadBuffer(res.Body.Bytes(), nil)
	require.NoError(t, err)
	defer img.Close()
	assert.Equal(t, 24, img.Width())
	assert.Equal(t, 24, img.Height())
}

func TestPassthroughWinsOverAutoWebP(t *testing.T) {
	app := ptApp(t, map[string][]byte{"simple.svg": []byte(ptSimpleSVG)},
		imagor.WithPassthroughFormats(imagor.BlobTypeSVG),
		imagor.WithAutoWebP(true))

	acceptWebP := http.Header{"Accept": {"image/webp,*/*"}}

	res := ptGet(t, app, "simple.svg", acceptWebP)
	require.Equal(t, http.StatusOK, res.Code)
	assert.Equal(t, "image/svg+xml", res.Header().Get("Content-Type"),
		"a negotiated raster format must not defeat passthrough")

	// ...while raster sources keep the negotiation they had before.
	res = ptGet(t, app, "gopher-front.png", acceptWebP)
	require.Equal(t, http.StatusOK, res.Code)
	assert.Equal(t, "image/webp", res.Header().Get("Content-Type"))
}

func TestPassthroughExplicitFormatSVG(t *testing.T) {
	app := ptApp(t, map[string][]byte{"hostile.svg": []byte(ptSVG)},
		imagor.WithPassthroughFormats(imagor.BlobTypeSVG))

	res := ptGet(t, app, "filters:format(svg)/hostile.svg", nil)
	require.Equal(t, http.StatusOK, res.Code)
	assert.Equal(t, "image/svg+xml", res.Header().Get("Content-Type"))
	assert.NotContains(t, res.Body.String(), "script")

	// A raster source cannot become a vector: say so rather than answering with
	// another format under the requested name.
	res = ptGet(t, app, "filters:format(svg)/gopher-front.png", nil)
	assert.Equal(t, http.StatusBadRequest, res.Code, res.Body.String())
}

func TestPassthroughPassiveFormatUnchanged(t *testing.T) {
	source, err := os.ReadFile(filepath.Join(testDataDir, "gopher-front.png"))
	require.NoError(t, err)

	app := ptApp(t, nil, imagor.WithPassthroughFormats(imagor.BlobTypePNG))

	res := ptGet(t, app, "gopher-front.png", nil)
	require.Equal(t, http.StatusOK, res.Code)
	assert.Equal(t, "image/png", res.Header().Get("Content-Type"))
	assert.Equal(t, source, res.Body.Bytes(), "passive passthrough must be byte-exact")
	assert.Equal(t, "nosniff", res.Header().Get("X-Content-Type-Options"))
}

func TestPassthroughRefusedFormatIsNotServed(t *testing.T) {
	app := ptApp(t, nil, imagor.WithPassthroughFormats(imagor.BlobTypePDF))

	res := ptGet(t, app, "sample.pdf", nil)
	require.Equal(t, http.StatusOK, res.Code, res.Body.String())
	assert.NotEqual(t, "application/pdf", res.Header().Get("Content-Type"),
		"a refused format must never be streamed")
}

func TestPassthroughMarkerIgnoredWithoutConfiguration(t *testing.T) {
	app := ptApp(t, map[string][]byte{"simple.svg": []byte(ptSimpleSVG)})

	res := ptGet(t, app, "filters:passthrough()/simple.svg", nil)
	require.Equal(t, http.StatusOK, res.Code)
	assert.Equal(t, "image/jpeg", res.Header().Get("Content-Type"),
		"the marker alone must not enable passthrough")
}

func TestPassthroughResultKeyAndCache(t *testing.T) {
	resDir := t.TempDir()
	resStorage := filestorage.New(resDir)
	mem := map[string][]byte{"simple.svg": []byte(ptSimpleSVG)}

	keysFor := func(withPassthrough bool) []string {
		opts := []imagor.Option{imagor.WithResultStorages(resStorage)}
		if withPassthrough {
			opts = append(opts, imagor.WithPassthroughFormats(imagor.BlobTypeSVG))
		}
		app := ptApp(t, mem, opts...)
		require.Equal(t, http.StatusOK, ptGet(t, app, "simple.svg", nil).Code)
		time.Sleep(400 * time.Millisecond) // the result save is asynchronous
		var keys []string
		require.NoError(t, filepath.Walk(resDir, func(p string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() {
				return err
			}
			rel, _ := filepath.Rel(resDir, p)
			keys = append(keys, filepath.ToSlash(rel))
			return nil
		}))
		for _, k := range keys {
			require.NoError(t, os.RemoveAll(filepath.Join(resDir, k)))
		}
		return keys
	}

	withKeys := keysFor(true)
	require.Len(t, withKeys, 1)
	assert.Equal(t, "filters%3Apassthrough%28%29/simple.svg", withKeys[0],
		"the passthrough marker must be part of the result key")

	// Without passthrough the same request keys without the marker, so the two
	// representations can never be served for each other.
	withoutKeys := keysFor(false)
	require.Len(t, withoutKeys, 1)
	assert.Equal(t, "simple.svg", withoutKeys[0])
	assert.NotEqual(t, withKeys[0], withoutKeys[0])

	// A second request is served from the result cache and stays an SVG.
	app := ptApp(t, map[string][]byte{"simple.svg": []byte(ptSimpleSVG)},
		imagor.WithResultStorages(resStorage),
		imagor.WithPassthroughFormats(imagor.BlobTypeSVG))
	require.Equal(t, http.StatusOK, ptGet(t, app, "simple.svg", nil).Code)
	time.Sleep(400 * time.Millisecond)
	res := ptGet(t, app, "simple.svg", nil)
	require.Equal(t, http.StatusOK, res.Code)
	assert.Equal(t, "image/svg+xml", res.Header().Get("Content-Type"))
	assert.Equal(t, imagor.SVGContentSecurityPolicy,
		res.Header().Get("Content-Security-Policy"),
		"the policy must hold for a response served from the result cache too")
	assert.Equal(t, "nosniff", res.Header().Get("X-Content-Type-Options"))
}

// TestPassthroughResolutionGuard checks the passthrough path does not become a
// way around the image bomb limits: the same request fails the same way with
// passthrough on and off.
func TestPassthroughResolutionGuard(t *testing.T) {
	mem := map[string][]byte{"simple.svg": []byte(ptSimpleSVG)}
	procOpts := []Option{WithMaxResolution(100)}

	rasterized := ptGet(t, ptAppWith(t, mem, procOpts), "simple.svg", nil)
	require.Equal(t, http.StatusUnprocessableEntity, rasterized.Code, rasterized.Body.String())

	passed := ptGet(t, ptAppWith(t, mem, procOpts,
		imagor.WithPassthroughFormats(imagor.BlobTypeSVG)), "simple.svg", nil)
	assert.Equal(t, rasterized.Code, passed.Code,
		"passthrough must enforce the same resolution limit")
}

// TestPassthroughSanitizeFallback covers the decision to degrade to rasterizing
// rather than fail, and to fail loudly when the client named svg.
func TestPassthroughSanitizeFallback(t *testing.T) {
	restore := maxSanitizeBytes
	maxSanitizeBytes = 64 // force the sanitizer to refuse a valid document
	t.Cleanup(func() { maxSanitizeBytes = restore })

	v := NewProcessor()
	v.SetPassthroughFormats([]imagor.BlobType{imagor.BlobTypeSVG})
	blob := imagor.NewBlobFromBytes([]byte(ptSimpleSVG))
	require.Equal(t, imagor.BlobTypeSVG, blob.BlobType())
	// The document is loadable by libvips, so it passes the resolution guard and
	// only the sanitizer refuses it - exactly the case where falling back beats
	// failing.
	require.NoError(t, v.checkPassthroughResolution(context.Background(), blob))

	marked := imagorpath.Params{
		Image:   "x.svg",
		Filters: imagorpath.Filters{{Name: imagor.PassthroughFilterName}},
	}
	out, handled, err := v.passthroughBlob(context.Background(), blob, marked)
	require.NoError(t, err)
	assert.False(t, handled, "an unsanitizable source must fall back to rasterizing")
	assert.Nil(t, out)

	explicit := imagorpath.Params{
		Image:   "x.svg",
		Filters: imagorpath.Filters{{Name: "format", Args: "svg"}},
	}
	_, handled, err = v.passthroughBlob(context.Background(), blob, explicit)
	assert.Error(t, err, "an explicit svg request must not answer with a raster")
	assert.False(t, handled)
}

func TestPassthroughProcessorRegistration(t *testing.T) {
	// The application must hand its configuration to the processor at startup,
	// otherwise the key and the behaviour would disagree.
	processor := NewProcessor()
	app := imagor.New(
		imagor.WithProcessors(processor),
		imagor.WithPassthroughFormats(imagor.BlobTypeSVG, imagor.BlobTypePNG),
	)
	require.NoError(t, app.Startup(context.Background()))
	defer func() { _ = app.Shutdown(context.Background()) }()

	assert.Contains(t, processor.passthroughFormats, imagor.BlobTypeSVG)
	assert.Contains(t, processor.passthroughFormats, imagor.BlobTypePNG)
	assert.NotContains(t, processor.passthroughFormats, imagor.BlobTypePDF)
	assert.ElementsMatch(t,
		[]string{"png", "svg"},
		imagor.PassthroughFormatNames(app.PassthroughFormats))
}
