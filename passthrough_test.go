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
		{
			name: "still formats are not configurable", in: []string{"png", "jpg"},
			wantErr: "not configurable",
		},
		{
			name: "one still format among valid", in: []string{"svg", "webp"},
			wantErr: "not configurable",
		},
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

// passthroughStubProcessor records the configuration the application hands down.
type passthroughStubProcessor struct {
	formats []BlobType
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
func (p *passthroughStubProcessor) SetPassthroughFormats(formats []BlobType) {
	p.formats = formats
}

func TestStartupForwardsPassthroughFormats(t *testing.T) {
	stub := &passthroughStubProcessor{}
	app := New(WithProcessors(stub), WithPassthroughFormats(BlobTypeSVG))
	require.NoError(t, app.Startup(context.Background()))
	assert.True(t, stub.started)
	assert.Equal(t, []BlobType{BlobTypeSVG}, stub.formats)

	// A processor that does not support passthrough must still start.
	plain := &passthroughStubProcessor{}
	require.NoError(t, New(WithProcessors(plain)).Startup(context.Background()))
	assert.Nil(t, plain.formats)
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
