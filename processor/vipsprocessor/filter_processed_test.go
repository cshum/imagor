package vipsprocessor

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cshum/imagor"
	"github.com/cshum/imagor/storage/filestorage"
	"github.com/cshum/vipsgen/vips"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func openTestImage(t *testing.T) *vips.Image {
	t.Helper()
	img, err := vips.NewImageFromFile(filepath.Join("..", "..", "testdata", "gopher-front.png"), nil)
	require.NoError(t, err)
	t.Cleanup(func() { img.Close() })
	return img
}

// The processed flag is what lets the dispatcher tell a filter that ran from one
// that declined. Every case below returned 200 with an unchanged image before it.
func TestFilterProcessedFlag(t *testing.T) {
	for _, tc := range []struct {
		name string
		fn   FilterFunc
		args []string
		want bool
		why  string
	}{
		{"blur/no-args", blur, nil, false, "blur() has no sigma and is silently ignored"},
		{"blur/sigma", blur, []string{"5"}, true, "blur(5) applies"},
		{"brightness/no-args", brightness, nil, false, "brightness() has no amount"},
		{"brightness/amount", brightness, []string{"40"}, true, "brightness(40) applies"},
		{"contrast/no-args", contrast, nil, false, "contrast() has no amount"},
		{"contrast/amount", contrast, []string{"40"}, true, "contrast(40) applies"},
		{"hue/no-args", hue, nil, false, "hue() has no angle"},
		{"hue/angle", hue, []string{"90"}, true, "hue(90) applies"},
		{"saturation/no-args", saturation, nil, false, "saturation() has no amount"},
		{"saturation/amount", saturation, []string{"50"}, true, "saturation(50) applies"},
		{"rotate/no-args", rotate, nil, false, "rotate() has no angle"},
		{"grayscale", grayscale, nil, true, "grayscale always applies"},
		{"invert", invert, nil, true, "invert always applies"},
		{"strip_exif", stripExif, nil, true, "strip_exif always applies"},
		{"strip_icc", stripIcc, nil, true, "strip_icc always applies"},
		{"pixelate/no-args", pixelate, nil, true, "pixelate defaults its block size"},
		// to_colorspace declines on the image rather than the arguments: with no
		// embedded ICC profile there is nothing to transform, and the gopher-front
		// fixture carries none.
		{"to_colorspace/no-profile", toColorspace, nil, false, "declines when the image carries no ICC profile"},
		{"rgb/short-args", rgb, []string{"1", "2"}, false, "rgb needs three components"},
		{"rgb/three-args", rgb, []string{"1", "2", "3"}, true, "rgb(1,2,3) applies"},
		{"modulate/short-args", modulate, []string{"1", "2"}, false, "modulate needs three components"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			img := openTestImage(t)
			got, err := tc.fn(context.Background(), img, nil, tc.args...)
			require.NoError(t, err, "declining must not be reported as an error")
			require.Equal(t, tc.want, got, tc.why)
		})
	}
}

// A filter that declines must leave the image untouched, so the flag cannot
// disagree with what actually happened.
func TestDeclinedFilterLeavesImageUnchanged(t *testing.T) {
	src := openTestImage(t)
	before, err := src.Copy(nil)
	require.NoError(t, err)
	defer before.Close()

	processed, err := blur(context.Background(), src, nil)
	require.NoError(t, err)
	require.False(t, processed, "blur() must decline")

	// Subtract the copy; a zero maximum difference means nothing changed.
	diff, err := src.Copy(nil)
	require.NoError(t, err)
	defer diff.Close()
	require.NoError(t, diff.Subtract(before))
	require.NoError(t, diff.Abs())
	max, err := diff.Max(nil)
	require.NoError(t, err)
	require.Zero(t, max, "a declined filter must not modify the image")
}

// A filter registered through WithFilter must report its own decline, and the
// flag must survive registration.
func TestWithFilterRespectsProcessedFlag(t *testing.T) {
	p := NewProcessor(WithFilter("declines", func(
		_ context.Context, _ *vips.Image, _ imagor.LoadFunc, _ ...string,
	) (bool, error) {
		return false, nil
	}))
	require.NotNil(t, p)
	require.NotNil(t, p.Filters["declines"], "WithFilter must register the handler")

	img := openTestImage(t)
	processed, err := p.Filters["declines"](context.Background(), img, nil)
	require.NoError(t, err)
	require.False(t, processed, "the flag must survive registration through WithFilter")
}

// filterReports is what lets the dispatch loop judge the filters that Process and
// loadAndProcess applied before it ran.
func TestFilterReportsRecord(t *testing.T) {
	reports := newFilterReports()

	_, recognised := reports.accepted("format")
	require.False(t, recognised, "an unrecorded name must be unrecognised")

	reports.record("format", false)
	recognised, processed := reports.accepted("format")
	require.True(t, recognised, "a name recorded as declined is still recognised")
	require.False(t, processed, "a recognised filter that declined reports false")

	reports.record("format", true)
	_, processed = reports.accepted("format")
	require.True(t, processed, "processed wins over declined")

	reports.record("format", false)
	_, processed = reports.accepted("format")
	require.True(t, processed, "a later decline must not clear an earlier accept")
}

// report is what a /meta response carries: one entry per occurrence, in URL
// order, since a filter can repeat and its occurrences can disagree. Only handled
// filters reach it; see TestMetaFilterReport.
func TestFilterReportsReport(t *testing.T) {
	reports := newFilterReports()
	// A filter can repeat, and its occurrences can disagree.
	reports.add("blur", true)
	reports.add("rotate", false)
	reports.add("blur", false)
	require.Equal(t, []FilterReport{
		{Name: "blur", Processed: true},
		{Name: "rotate", Processed: false},
		{Name: "blur", Processed: false},
	}, reports.report())

	require.Nil(t, newFilterReports().report(), "nothing handled reports nothing")
}

// A client must be able to tell a working filter URL from an inert one: every case
// below returned 200 with an unchanged image and the same log line before this.
func TestFilterEventOutcomes(t *testing.T) {
	const img = "gopher-front.png"
	core, logs := observer.New(zap.DebugLevel)
	fileLoader := filestorage.New(testDataDir)
	app := imagor.New(
		imagor.WithLoaders(loaderFunc(func(r *http.Request, image string) (blob *imagor.Blob, err error) {
			image, _ = fileLoader.Path(image)
			return imagor.NewBlob(func() (reader io.ReadCloser, size int64, err error) {
				reader, err = os.Open(image)
				return
			}), nil
		})),
		imagor.WithUnsafe(true),
		imagor.WithLogger(zap.NewNop()),
		imagor.WithProcessors(NewProcessor(WithDebug(true), WithLogger(zap.New(core)))),
	)
	require.NoError(t, app.Startup(context.Background()))
	// Deliberately not shut down: Shutdown calls vips.Shutdown() at a processor
	// count of zero, and this test runs before processor_test.go.

	for _, tc := range []struct {
		name string
		path string
		want map[string]string
	}{
		{
			name: "unrecognised name",
			path: "200x200/filters:vintage(70)/" + img,
			want: map[string]string{"vintage": "filter-unhandled"},
		},
		{
			name: "handler declines",
			path: "200x200/filters:blur()/" + img,
			want: map[string]string{"blur": "filter-declined"},
		},
		{
			name: "handler processes",
			path: "200x200/filters:blur(5)/" + img,
			want: map[string]string{"blur": "filter"},
		},
		{
			name: "export filter with unparsable args",
			path: "200x200/filters:quality(abc)/" + img,
			want: map[string]string{"quality": "filter-declined"},
		},
		{
			name: "unrecognised export format",
			path: "200x200/filters:format(bogus)/" + img,
			want: map[string]string{"format": "filter-declined"},
		},
		{
			name: "export filters that apply",
			path: "200x200/filters:format(jpeg):quality(70)/" + img,
			want: map[string]string{"format": "filter", "quality": "filter"},
		},
		{
			name: "decode constraint with unparsable args",
			path: "200x200/filters:max_frames(abc)/" + img,
			want: map[string]string{"max_frames": "filter-declined"},
		},
		{
			name: "decode constraint that applies",
			path: "200x200/filters:stretch()/" + img,
			want: map[string]string{"stretch": "filter"},
		},
		{
			// avgcolor is recognised only under the /meta endpoint, so outside it
			// the filter genuinely does nothing.
			name: "meta-only filter outside meta",
			path: "200x200/filters:avgcolor()/" + img,
			want: map[string]string{"avgcolor": "filter-unhandled"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			before := logs.Len()
			w := httptest.NewRecorder()
			app.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/unsafe/"+tc.path, nil))
			require.Equal(t, 200, w.Code)

			events := map[string]string{}
			for _, e := range logs.All()[before:] {
				if !strings.HasPrefix(e.Message, "filter") {
					continue
				}
				if name, ok := e.ContextMap()["name"].(string); ok {
					events[name] = e.Message
				}
			}
			for name, want := range tc.want {
				require.Equal(t, want, events[name], "filter %q produced the wrong event", name)
			}
		})
	}
}

// VIPS_DISABLE_FILTERS has to be honoured everywhere a filter's effect is
// applied, not only where it is dispatched. A region is returned only if
// detection actually ran, so it doubles as proof that redact() was inert.
func TestDisableFilters(t *testing.T) {
	fileLoader := filestorage.New(testDataDir)
	detector := &stubDetector{regions: []imagor.DetectorRegion{{
		Left: 0.1, Top: 0.1, Right: 0.4, Bottom: 0.6, Name: "face",
	}}}
	app := imagor.New(
		imagor.WithLoaders(loaderFunc(func(r *http.Request, image string) (blob *imagor.Blob, err error) {
			image, _ = fileLoader.Path(image)
			return imagor.NewBlob(func() (reader io.ReadCloser, size int64, err error) {
				reader, err = os.Open(image)
				return
			}), nil
		})),
		imagor.WithUnsafe(true),
		imagor.WithLogger(zap.NewNop()),
		imagor.WithProcessors(NewProcessor(
			WithDetector(detector),
			WithDisableFilters("avgcolor", "blurhash", "format", "strip_exif", "redact"))),
	)
	require.NoError(t, app.Startup(context.Background()))

	// The metadata switch used to apply these regardless.
	t.Run("metadata filters do not run", func(t *testing.T) {
		w := httptest.NewRecorder()
		app.ServeHTTP(w, httptest.NewRequest(http.MethodGet,
			"/unsafe/meta/filters:avgcolor():blurhash(4,3)/Canon_40D.jpg", nil))
		require.Equal(t, 200, w.Code)

		var meta struct {
			AverageColor *AvgColor `json:"average_color"`
			BlurHash     string    `json:"blurhash"`
		}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &meta))
		require.Nil(t, meta.AverageColor, "avgcolor is disabled")
		require.Empty(t, meta.BlurHash, "blurhash is disabled")
	})

	t.Run("export filters do not run", func(t *testing.T) {
		w := httptest.NewRecorder()
		app.ServeHTTP(w, httptest.NewRequest(http.MethodGet,
			"/unsafe/100x100/filters:format(png)/Canon_40D.jpg", nil))
		require.Equal(t, 200, w.Code)
		require.Equal(t, "image/jpeg", w.Header().Get("Content-Type"),
			"format is disabled, so the source format is kept")
	})

	// strip_exif used to hide exif from the metadata response whether or not it
	// was disabled, because the gate only asked whether the URL mentioned it.
	t.Run("strip_exif does not strip the metadata response", func(t *testing.T) {
		w := httptest.NewRecorder()
		app.ServeHTTP(w, httptest.NewRequest(http.MethodGet,
			"/unsafe/meta/filters:strip_exif()/Canon_40D.jpg", nil))
		require.Equal(t, 200, w.Code)

		var meta struct {
			Exif map[string]string `json:"exif"`
		}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &meta))
		require.NotEmpty(t, meta.Exif, "strip_exif is disabled, so exif is returned")
	})

	// redact used to trigger detection regardless, so a disabled filter still
	// ran the detector and still put regions in the response.
	t.Run("redact does not trigger detection", func(t *testing.T) {
		w := httptest.NewRecorder()
		app.ServeHTTP(w, httptest.NewRequest(http.MethodGet,
			"/unsafe/meta/filters:redact()/Canon_40D.jpg", nil))
		require.Equal(t, 200, w.Code)

		var meta struct {
			DetectedRegions []imagor.DetectorRegion `json:"detected_regions"`
		}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &meta))
		require.Empty(t, meta.DetectedRegions, "redact is disabled, so detection does not run")
	})
}

// VIPS_MAX_FILTER_OPS bounds the operations the processor performs, not the
// filters in the URL: one handled by an earlier switch costs nothing here and
// must not consume the budget of one that does.
func TestMaxFilterOpsCountsOperations(t *testing.T) {
	core, logs := observer.New(zap.DebugLevel)
	fileLoader := filestorage.New(testDataDir)
	app := imagor.New(
		imagor.WithLoaders(loaderFunc(func(r *http.Request, image string) (blob *imagor.Blob, err error) {
			image, _ = fileLoader.Path(image)
			return imagor.NewBlob(func() (reader io.ReadCloser, size int64, err error) {
				reader, err = os.Open(image)
				return
			}), nil
		})),
		imagor.WithUnsafe(true),
		imagor.WithLogger(zap.NewNop()),
		imagor.WithProcessors(NewProcessor(
			WithDebug(true), WithLogger(zap.New(core)), WithMaxFilterOps(1))),
	)
	require.NoError(t, app.Startup(context.Background()))

	// The reported filters are the ones the loop reached, so this is what the
	// budget was spent on.
	reported := func(path string) []string {
		before := logs.Len()
		w := httptest.NewRecorder()
		app.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/unsafe/"+path, nil))
		require.Equal(t, 200, w.Code)

		var names []string
		for _, e := range logs.All()[before:] {
			if !strings.HasPrefix(e.Message, "filter") {
				continue
			}
			if name, ok := e.ContextMap()["name"].(string); ok {
				names = append(names, name)
			}
		}
		return names
	}

	// format is handled before the loop, so it must not spend the one allowed
	// operation. The previous index-based guard broke here and blur never ran.
	require.Equal(t,
		[]string{"format", "blur"},
		reported("100x100/filters:format(jpeg):blur(5)/Canon_40D.jpg"))

	// The cap still applies to the operations the loop performs.
	require.Equal(t,
		[]string{"blur"},
		reported("100x100/filters:blur(5):blur(3)/Canon_40D.jpg"))
}

// A /meta response reports what this processor did, and stays silent about names
// it did not handle, since a later processor may handle them.
func TestMetaFilterReport(t *testing.T) {
	fileLoader := filestorage.New(testDataDir)
	app := imagor.New(
		imagor.WithLoaders(loaderFunc(func(r *http.Request, image string) (blob *imagor.Blob, err error) {
			image, _ = fileLoader.Path(image)
			return imagor.NewBlob(func() (reader io.ReadCloser, size int64, err error) {
				reader, err = os.Open(image)
				return
			}), nil
		})),
		imagor.WithUnsafe(true),
		imagor.WithLogger(zap.NewNop()),
		imagor.WithProcessors(NewProcessor()),
	)
	require.NoError(t, app.Startup(context.Background()))

	for _, tc := range []struct {
		name string
		path string
		want []FilterReport
	}{
		{
			name: "processed and declined",
			path: "meta/200x200/filters:blur(5):rotate()/gopher-front.png",
			want: []FilterReport{
				{Name: "blur", Processed: true},
				{Name: "rotate", Processed: false},
			},
		},
		{
			name: "a name this processor did not handle is left out",
			path: "meta/200x200/filters:vintage(70):blur(5)/gopher-front.png",
			want: []FilterReport{{Name: "blur", Processed: true}},
		},
		{
			name: "nothing to report omits the field",
			path: "meta/200x200/filters:vintage(70)/gopher-front.png",
			want: nil,
		},
		{
			name: "no filters omits the field",
			path: "meta/200x200/gopher-front.png",
			want: nil,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			app.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/unsafe/"+tc.path, nil))
			require.Equal(t, 200, w.Code)

			var meta struct {
				Filters []FilterReport `json:"filters"`
			}
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &meta))
			require.Equal(t, tc.want, meta.Filters)
		})
	}
}
