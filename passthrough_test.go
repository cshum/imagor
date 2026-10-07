package imagor

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/cshum/imagor/imagorpath"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParsePassthroughFormats(t *testing.T) {
	for _, tc := range []struct {
		name    string
		in      []string
		want    []BlobType
		wantErr string
	}{
		{name: "empty", in: nil, want: nil},
		{name: "blank entries", in: []string{"", "  "}, want: nil},
		{name: "svg", in: []string{"svg"}, want: []BlobType{BlobTypeSVG}},
		{name: "case and space", in: []string{" SVG "}, want: []BlobType{BlobTypeSVG}},
		{name: "duplicates collapse", in: []string{"svg", "svg"}, want: []BlobType{BlobTypeSVG}},
		{name: "passive formats", in: []string{"png", "jpg"}, want: []BlobType{BlobTypePNG, BlobTypeJPEG}},
		{name: "jpeg alias", in: []string{"jpeg"}, want: []BlobType{BlobTypeJPEG}},
		{
			name: "pdf is refused", in: []string{"pdf"},
			wantErr: "refused",
		},
		{
			name: "svg plus pdf is refused as a whole", in: []string{"svg", "pdf"},
			wantErr: "refused",
		},
		{name: "unknown name", in: []string{"exe"}, wantErr: "unknown passthrough format"},
		{name: "raw camera format", in: []string{"cr2"}, wantErr: "unknown passthrough format"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParsePassthroughFormats(tc.in)
			if tc.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestPassthroughPolicyOf(t *testing.T) {
	assert.Equal(t, PassthroughSanitize, PassthroughPolicyOf(BlobTypeSVG))
	for _, p := range []BlobType{
		BlobTypePNG, BlobTypeJPEG, BlobTypeGIF, BlobTypeWEBP, BlobTypeAVIF,
		BlobTypeHEIF, BlobTypeTIFF, BlobTypeBMP, BlobTypeJXL, BlobTypeJP2,
	} {
		assert.Equal(t, PassthroughPassive, PassthroughPolicyOf(p), "format %v", p)
	}
	// Active content and everything unconsidered is refused rather than passed
	// through by default.
	for _, p := range []BlobType{
		BlobTypePDF, BlobTypeUnknown, BlobTypeEmpty, BlobTypeJSON, BlobTypeMemory,
		BlobTypeCR2, BlobTypeRAF, BlobTypeORF, BlobTypeRW2, BlobTypeX3F, BlobTypeCR3,
	} {
		assert.Equal(t, PassthroughRefused, PassthroughPolicyOf(p), "format %v", p)
	}
}

func TestPassthroughFormatNames(t *testing.T) {
	assert.ElementsMatch(t,
		[]string{"svg"},
		PassthroughFormatNames([]BlobType{BlobTypeSVG}))
	assert.ElementsMatch(t,
		[]string{"svg", "png"},
		PassthroughFormatNames([]BlobType{BlobTypePNG, BlobTypeSVG}))
	assert.Empty(t, PassthroughFormatNames(nil))
}

// TestPassthroughEligible pins the application-side rule: the marker is only
// appended when the request asks for nothing at all. The field-by-field sweep
// that guards this against new Params fields lives next to the predicate in
// imagorpath (TestHasTransformationsCoversEveryParamsField).
func TestPassthroughEligible(t *testing.T) {
	assert.True(t, passthroughEligible(imagorpath.Params{Image: "plain/photo.svg"}))

	for _, tc := range []struct {
		name   string
		params imagorpath.Params
	}{
		{"meta", imagorpath.Params{Image: "x.svg", Meta: true}},
		{"width", imagorpath.Params{Image: "x.svg", Width: 100}},
		{"fit-in", imagorpath.Params{Image: "x.svg", FitIn: true}},
		{"smart", imagorpath.Params{Image: "x.svg", Smart: true}},
		{"trim", imagorpath.Params{Image: "x.svg", Trim: true}},
		{"crop", imagorpath.Params{Image: "x.svg", CropLeft: 0.5}},
		{"padding", imagorpath.Params{Image: "x.svg", PaddingLeft: 1}},
		{
			// A filter the client wrote is not a no-op, including a format:
			// asking for an output format is asking for work.
			"format filter",
			imagorpath.Params{Image: "x.svg", Filters: imagorpath.Filters{{Name: "format", Args: "webp"}}},
		},
		{
			"raw filter",
			imagorpath.Params{Image: "x.svg", Filters: imagorpath.Filters{{Name: "raw"}}},
		},
		{
			"processing filter",
			imagorpath.Params{Image: "x.svg", Filters: imagorpath.Filters{{Name: "rotate", Args: "90"}}},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.False(t, passthroughEligible(tc.params))
		})
	}
}

// passthroughStubProcessor is a minimal processor: it takes part without knowing
// that passthrough exists.
type passthroughStubProcessor struct {
	started bool
}

func (p *passthroughStubProcessor) Startup(context.Context) error { p.started = true; return nil }
func (p *passthroughStubProcessor) Shutdown(context.Context) error {
	return nil
}
func (p *passthroughStubProcessor) Process(
	_ context.Context, blob *Blob, _ imagorpath.Params, _ LoadFunc,
) (*Blob, error) {
	return blob, nil
}
func TestStartupStartsProcessors(t *testing.T) {
	// A processor takes part without implementing anything for passthrough: the
	// application serves it itself.
	stub := &passthroughStubProcessor{}
	app := New(WithProcessors(stub), WithPassthroughFormats(BlobTypeSVG))
	require.NoError(t, app.Startup(context.Background()))
	assert.True(t, stub.started)
}

// ptStubLoader serves one blob for any key.
type ptStubLoader struct{ blob *Blob }

func (l ptStubLoader) Get(*http.Request, string) (*Blob, error) { return l.blob, nil }

// TestPassthroughMarkerInRequestPath checks the marker is appended only for
// no-op requests, and that it lands in the path the result storage keys on.
func TestPassthroughMarkerInRequestPath(t *testing.T) {
	newApp := func(t *testing.T, opts ...Option) (*Imagor, *[]string) {
		t.Helper()
		var paths []string
		stub := &passthroughStubProcessor{}
		app := New(append([]Option{
			WithProcessors(stub),
			WithUnsafe(true),
			WithLoaders(ptStubLoader{blob: NewBlobFromBytes([]byte("x"))}),
			WithGetResultKey(func(_ *http.Request, p imagorpath.Params) string {
				paths = append(paths, p.Path)
				return ""
			}),
		}, opts...)...)
		require.NoError(t, app.Startup(context.Background()))
		t.Cleanup(func() { _ = app.Shutdown(context.Background()) })
		return app, &paths
	}
	serve := func(app *Imagor, path string) {
		app.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/unsafe/"+path, nil))
	}

	t.Run("no-op request is marked", func(t *testing.T) {
		app, paths := newApp(t, WithPassthroughFormats(BlobTypeSVG))
		serve(app, "plain/photo.svg")
		require.Len(t, *paths, 1)
		assert.Equal(t, "filters:"+PassthroughFilterName+"()/plain/photo.svg", (*paths)[0])
	})

	t.Run("disabled is not marked", func(t *testing.T) {
		app, paths := newApp(t)
		serve(app, "plain/photo.svg")
		require.Len(t, *paths, 1)
		assert.Equal(t, "plain/photo.svg", (*paths)[0])
	})

	t.Run("a transformation is not marked", func(t *testing.T) {
		app, paths := newApp(t, WithPassthroughFormats(BlobTypeSVG))
		serve(app, "100x100/plain/photo.svg")
		require.Len(t, *paths, 1)
		assert.Equal(t, "100x100/plain/photo.svg", (*paths)[0])
	})

	t.Run("an explicit format is not marked", func(t *testing.T) {
		app, paths := newApp(t, WithPassthroughFormats(BlobTypeSVG))
		serve(app, "filters:format(webp)/plain/photo.svg")
		require.Len(t, *paths, 1)
		assert.NotContains(t, (*paths)[0], PassthroughFilterName)
	})

	t.Run("raw is not marked", func(t *testing.T) {
		app, paths := newApp(t, WithPassthroughFormats(BlobTypeSVG))
		serve(app, "filters:raw()/plain/photo.svg")
		require.Len(t, *paths, 1)
		assert.NotContains(t, (*paths)[0], PassthroughFilterName)
	})

	t.Run("meta is not marked", func(t *testing.T) {
		app, paths := newApp(t, WithPassthroughFormats(BlobTypeSVG))
		serve(app, "meta/plain/photo.svg")
		require.Len(t, *paths, 1)
		assert.NotContains(t, (*paths)[0], PassthroughFilterName)
	})

	t.Run("auto format negotiation keeps the marker", func(t *testing.T) {
		// The marker is decided before negotiation appends its format filter, so
		// a negotiated representation does not defeat passthrough.
		app, paths := newApp(t, WithPassthroughFormats(BlobTypeSVG), WithAutoWebP(true))
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/unsafe/plain/photo.svg", nil)
		req.Header.Set("Accept", "image/webp,*/*")
		app.ServeHTTP(w, req)
		require.Len(t, *paths, 1)
		assert.Equal(t, "filters:"+PassthroughFilterName+"():format(webp)/plain/photo.svg", (*paths)[0])
		// The negotiated representation still varies on Accept, so a shared cache
		// cannot serve this response to a client that would get WebP.
		assert.Contains(t, w.Header().Values("Vary"), "Accept")
	})
}

// markedParams is the request the application marks when passthrough applies.
func markedParams(image string) imagorpath.Params {
	return imagorpath.Params{
		Image:   image,
		Filters: imagorpath.Filters{{Name: PassthroughFilterName}},
	}
}

// explicitSVGParams is a request that names the vector as its output format.
func explicitSVGParams(image string) imagorpath.Params {
	return imagorpath.Params{
		Image:   image,
		Filters: imagorpath.Filters{{Name: "format", Args: "svg"}},
	}
}

// TestServePassthroughSanitizeFallback covers the decision to degrade to
// rasterizing rather than fail, and to fail loudly when the client named svg.
func TestServePassthroughSanitizeFallback(t *testing.T) {
	const doc = `<svg xmlns="http://www.w3.org/2000/svg" width="64" height="64">` +
		`<rect width="64" height="64"/></svg>`
	restore := maxSanitizeBytes
	maxSanitizeBytes = 64 // force the sanitizer to refuse a valid document
	t.Cleanup(func() { maxSanitizeBytes = restore })

	app := New(WithPassthroughFormats(BlobTypeSVG))
	blob := NewBlobFromBytes([]byte(doc))
	require.Equal(t, BlobTypeSVG, blob.BlobType())

	out, handled, err := app.servePassthrough(markedParams("x.svg"), blob)
	require.NoError(t, err)
	assert.False(t, handled, "an unsanitizable source must fall back to rasterizing")
	assert.Nil(t, out)

	_, handled, err = app.servePassthrough(explicitSVGParams("x.svg"), blob)
	assert.Error(t, err, "an explicit svg request must not answer with a raster")
	assert.False(t, handled)
}

// TestServePassthroughLatin1SourceServed covers the one declared charset imagor
// maps: the document is converted rather than refused, and the source bytes are
// not re-emitted.
func TestServePassthroughLatin1SourceServed(t *testing.T) {
	const latin1 = "<?xml version=\"1.0\" encoding=\"ISO-8859-1\"?>" +
		"<svg xmlns=\"http://www.w3.org/2000/svg\" width=\"64\" height=\"64\">" +
		"<text x=\"4\" y=\"32\">caf\xe9</text><script>alert(1)</script></svg>"
	app := New(WithPassthroughFormats(BlobTypeSVG))
	blob := NewBlobFromBytes([]byte(latin1))
	require.Equal(t, BlobTypeSVG, blob.BlobType())

	out, handled, err := app.servePassthrough(markedParams("x.svg"), blob)
	require.NoError(t, err)
	require.True(t, handled)
	assert.Equal(t, SVGContentType, out.ContentType())

	data, err := out.ReadAll()
	require.NoError(t, err)
	assert.Contains(t, string(data), "café")
	assert.NotContains(t, string(data), "script")
	assert.NotContains(t, string(data), "\xe9", "the source bytes are converted, not re-emitted")
}

// TestServePassthroughUnsupportedCharsetFallsBack covers a document declaring a
// charset the sanitizer cannot map: the marked request rasterizes, the explicit
// request fails.
func TestServePassthroughUnsupportedCharsetFallsBack(t *testing.T) {
	const cp1252 = "<?xml version=\"1.0\" encoding=\"windows-1252\"?>" +
		"<svg xmlns=\"http://www.w3.org/2000/svg\" width=\"64\" height=\"64\">" +
		"<text x=\"4\" y=\"32\">\x93quoted\x94</text></svg>"
	app := New(WithPassthroughFormats(BlobTypeSVG))
	blob := NewBlobFromBytes([]byte(cp1252))
	require.Equal(t, BlobTypeSVG, blob.BlobType())

	out, handled, err := app.servePassthrough(markedParams("x.svg"), blob)
	require.NoError(t, err)
	assert.False(t, handled)
	assert.Nil(t, out)

	_, handled, err = app.servePassthrough(explicitSVGParams("x.svg"), blob)
	assert.Error(t, err)
	assert.False(t, handled)
}

// TestServePassthroughWithoutAProcessor is the point of serving from the
// application: the source is returned with no processor registered at all, so no
// processor has to know that passthrough exists.
func TestServePassthroughWithoutAProcessor(t *testing.T) {
	const doc = `<svg xmlns="http://www.w3.org/2000/svg" width="8" height="8">` +
		`<rect width="8" height="8"/></svg>`
	app := New(
		WithUnsafe(true),
		WithPassthroughFormats(BlobTypeSVG),
		WithLoaders(loaderFunc(func(r *http.Request, image string) (*Blob, error) {
			return NewBlobFromBytes([]byte(doc)), nil
		})),
	)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, httptest.NewRequest(
		http.MethodGet, "https://example.com/unsafe/photo.svg", nil))

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, SVGContentType, w.Header().Get("Content-Type"))
	assert.Equal(t, SVGContentSecurityPolicy, w.Header().Get("Content-Security-Policy"))
	assert.Contains(t, w.Body.String(), "<rect")
}

// TestBlobTypeSVGWithContentType covers a source that arrives with its content
// type already set, as the HTTP loader does: the bytes still decide the type, or
// nothing would recognise the document as an SVG.
func TestBlobTypeSVGWithContentType(t *testing.T) {
	const doc = `<svg xmlns="http://www.w3.org/2000/svg" width="8" height="8"><rect width="8" height="8"/></svg>`
	blob := NewBlobFromBytes([]byte(doc))
	blob.SetContentType("image/svg+xml")
	assert.Equal(t, BlobTypeSVG, blob.BlobType())
}
