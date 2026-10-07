package imagor_test

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
	"github.com/cshum/imagor/loader/httploader"
	"github.com/cshum/imagor/processor/vipsprocessor"
	"github.com/cshum/imagor/storage/filestorage"
	"github.com/cshum/vipsgen/vips"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// ptTestDataDir is the source imagor's own testdata, resolved from the package
// directory the test binary runs in.
const ptTestDataDir = "testdata"

// ptSVG is a document exercising the constructs that must be dropped, next to
// presentation that must survive.
//
// It contains external references, which librsvg tries to fetch when the
// document is rasterized - keep it out of tests that rasterize more than once
// (see ptSimpleSVG).
const ptSVG = `<svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" width="64" height="64" viewBox="0 0 64 64" onload="alert(1)">
  <script>alert('xss')</script>
  <foreignObject><iframe src="https://evil.test/"></iframe></foreignObject>
  <use href="#ok"/>
  <rect width="64" height="64" fill="#0af" onclick="alert(2)"/>
  <rect id="ok" x="4" y="4" width="8" height="8" fill="#f90"/>
  <text x="8" y="56" font-size="10" fill="#333">kept</text>
</svg>`

// ptExternalSVG references a resource outside the document, so serving it would
// mean serving a document that renders without what it points at.
const ptExternalSVG = `<svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" width="64" height="64" viewBox="0 0 64 64">
  <image xlink:href="https://evil.test/track.png" width="8" height="8"/>
  <rect width="64" height="64" fill="#0af"/>
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
	return filestorage.New(ptTestDataDir).Get(r, key)
}

func (l *ptLoader) Stat(_ context.Context, _ string) (*imagor.Stat, error) {
	return nil, imagor.ErrNotFound
}

func (l *ptLoader) Put(_ context.Context, _ string, _ *imagor.Blob) error { return nil }

func (l *ptLoader) Delete(_ context.Context, _ string) error { return nil }

// ptKeepAliveOnce holds a processor that is never shut down, so the vips
// refcount inside the processor never reaches zero during the test binary:
// vips.Shutdown() followed by vips.Startup() leaves the next rasterization to
// segfault on some libvips builds (8.18.2 locally; CI pins 8.18.6).
var ptKeepAliveOnce sync.Once

func ptKeepVipsAlive() {
	ptKeepAliveOnce.Do(func() {
		_ = vipsprocessor.NewProcessor().Startup(context.Background())
	})
}

// ptApp builds an app whose loader serves the given documents plus testdata.
func ptApp(t *testing.T, mem map[string][]byte, appOpts ...imagor.Option) *imagor.Imagor {
	t.Helper()
	return ptAppWith(t, mem, nil, appOpts...)
}

func ptAppWith(
	t *testing.T, mem map[string][]byte, procOpts []vipsprocessor.Option, appOpts ...imagor.Option,
) *imagor.Imagor {
	t.Helper()
	ptKeepVipsAlive()
	processor := vipsprocessor.NewProcessor(procOpts...)
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
		"default-src 'none'; style-src 'unsafe-inline'; img-src data:; sandbox",
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
		nil, imagor.WithSanitizeSVG(false),
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
// "convert it". Passthrough wins, because the marker only ever accompanies a
// format filter that content negotiation injected - if a named format beat the
// marker, auto WebP/AVIF would disable passthrough for exactly the clients it
// was enabled for. A request that wants a raster format must not carry it.
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

// TestPassthroughExplicitFormatSVG covers format(svg) as an explicit request:
// it is honoured with sanitization whether or not the operator enabled
// passthrough for SVG sources, because the safe path needs no configuration.
func TestPassthroughExplicitFormatSVG(t *testing.T) {
	mem := map[string][]byte{"hostile.svg": []byte(ptSVG)}

	t.Run("passthrough enabled", func(t *testing.T) {
		app := ptApp(t, mem, imagor.WithPassthroughFormats(imagor.BlobTypeSVG))
		res := ptGet(t, app, "filters:format(svg)/hostile.svg", nil)
		require.Equal(t, http.StatusOK, res.Code)
		assert.Equal(t, "image/svg+xml", res.Header().Get("Content-Type"))
		assert.NotContains(t, res.Body.String(), "script")
	})

	t.Run("passthrough disabled still sanitizes", func(t *testing.T) {
		app := ptApp(t, mem)
		res := ptGet(t, app, "filters:format(svg)/hostile.svg", nil)
		require.Equal(t, http.StatusOK, res.Code, res.Body.String())
		assert.Equal(t, "image/svg+xml", res.Header().Get("Content-Type"))

		body := res.Body.String()
		for _, leaked := range []string{"script", "onload", "evil.test", "foreignObject"} {
			assert.NotContains(t, body, leaked, "explicit svg request leaked %q", leaked)
		}
		assert.Contains(t, body, `fill="#0af"`)
	})

	t.Run("a raster source cannot become a vector", func(t *testing.T) {
		app := ptApp(t, mem, imagor.WithPassthroughFormats(imagor.BlobTypeSVG))
		res := ptGet(t, app, "filters:format(svg)/gopher-front.png", nil)
		assert.Equal(t, http.StatusBadRequest, res.Code, res.Body.String())
	})

	t.Run("sanitization off needs the operator opt-in", func(t *testing.T) {
		// Without the opt-in, honouring the request would mean serving upstream
		// markup untouched - a way around the operator's decision, not a use of
		// it.
		app := ptAppWith(t, mem, nil, imagor.WithSanitizeSVG(false))
		res := ptGet(t, app, "filters:format(svg)/hostile.svg", nil)
		assert.Equal(t, http.StatusBadRequest, res.Code, res.Body.String())

		// With it, the bytes are served as they are.
		app = ptAppWith(t, mem, nil, imagor.WithSanitizeSVG(false),
			imagor.WithPassthroughFormats(imagor.BlobTypeSVG))
		res = ptGet(t, app, "filters:format(svg)/hostile.svg", nil)
		require.Equal(t, http.StatusOK, res.Code)
		assert.Equal(t, "image/svg+xml", res.Header().Get("Content-Type"))
		assert.Equal(t, ptSVG, res.Body.String())
	})
}

func TestPassthroughPassiveFormatUnchanged(t *testing.T) {
	source, err := os.ReadFile(filepath.Join(ptTestDataDir, "gopher-front.png"))
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

// TestPassthroughWithTheHTTPLoader covers the shape a deployment actually has:
// the source comes from the HTTP loader, which sets the content type on the
// blob, so the type has to be recognised from the bytes or no document would be
// seen as an SVG.
func TestPassthroughWithTheHTTPLoader(t *testing.T) {
	ptKeepVipsAlive()
	const doc = `<svg xmlns="http://www.w3.org/2000/svg" width="8" height="8">` +
		`<script>alert(1)</script><rect width="8" height="8"/></svg>`
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/svg+xml")
		_, _ = w.Write([]byte(doc))
	}))
	defer upstream.Close()

	app := imagor.New(
		imagor.WithUnsafe(true),
		imagor.WithPassthroughFormats(imagor.BlobTypeSVG),
		imagor.WithLoaders(httploader.New()),
		imagor.WithProcessors(vipsprocessor.NewProcessor()),
		imagor.WithLogger(zap.NewNop()),
	)
	require.NoError(t, app.Startup(context.Background()))
	defer func() { _ = app.Shutdown(context.Background()) }()

	w := httptest.NewRecorder()
	app.ServeHTTP(w, httptest.NewRequest(
		http.MethodGet, "/unsafe/"+upstream.URL+"/x.svg", nil))
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, imagor.SVGContentType, w.Header().Get("Content-Type"),
		"a document from the HTTP loader must be served as markup, not rasterized")
	assert.NotContains(t, w.Body.String(), "<script>")
}

// TestPassthroughServesOverResolutionLimits pins the decision that serving is not
// rendering: the document is handed over as it came, so the limits that bound
// what imagor renders - and the decoder that would measure them - are not
// involved.
func TestPassthroughServesOverResolutionLimits(t *testing.T) {
	mem := map[string][]byte{"simple.svg": []byte(ptSimpleSVG)}
	procOpts := []vipsprocessor.Option{vipsprocessor.WithMaxResolution(100)}

	rasterized := ptGet(t, ptAppWith(t, mem, procOpts), "simple.svg", nil)
	require.Equal(t, http.StatusUnprocessableEntity, rasterized.Code, rasterized.Body.String())

	served := ptGet(t, ptAppWith(t, mem, procOpts,
		imagor.WithPassthroughFormats(imagor.BlobTypeSVG)), "simple.svg", nil)
	require.Equal(t, http.StatusOK, served.Code, served.Body.String())
	assert.Equal(t, imagor.SVGContentType, served.Header().Get("Content-Type"))
}

// TestPassthroughRefusesExternalReference covers the source that points at
// something outside the document: it is rasterized rather than served, because
// the served document would render without what it points at.
func TestPassthroughRefusesExternalReference(t *testing.T) {
	app := ptApp(t, map[string][]byte{"external.svg": []byte(ptExternalSVG)},
		imagor.WithPassthroughFormats(imagor.BlobTypeSVG))

	res := ptGet(t, app, "external.svg", nil)
	require.Equal(t, http.StatusOK, res.Code)
	assert.NotEqual(t, imagor.SVGContentType, res.Header().Get("Content-Type"),
		"a document referencing an external resource must be rasterized")

	explicit := ptGet(t, app, "filters:format(svg)/external.svg", nil)
	assert.GreaterOrEqual(t, explicit.Code, 400,
		"an explicit vector request must not answer with a raster")
	assert.Contains(t, explicit.Body.String(), "would change how it looks")
}

// TestPassthroughRefusesDocumentStyling covers the source that carries its own
// CSS: it is rasterized rather than served with the styling stripped, so the
// response still looks like the document the author drew.
func TestPassthroughRefusesDocumentStyling(t *testing.T) {
	const styled = `<svg xmlns="http://www.w3.org/2000/svg" width="64" height="64">` +
		`<style>.a{fill:#f90}</style><rect class="a" width="64" height="64"/></svg>`
	app := ptApp(t, map[string][]byte{"styled.svg": []byte(styled)},
		imagor.WithPassthroughFormats(imagor.BlobTypeSVG))

	res := ptGet(t, app, "styled.svg", nil)
	require.Equal(t, http.StatusOK, res.Code)
	assert.NotEqual(t, imagor.SVGContentType, res.Header().Get("Content-Type"),
		"a document with its own CSS must be rasterized, not served with the CSS stripped")

	explicit := ptGet(t, app, "filters:format(svg)/styled.svg", nil)
	assert.Equal(t, http.StatusBadRequest, explicit.Code,
		"an explicit vector request must not answer with a raster, and a document that cannot be served is the caller's to act on")
}
