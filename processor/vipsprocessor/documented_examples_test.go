package vipsprocessor

import (
	"context"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/cshum/imagor"
	"github.com/cshum/imagor/imagorpath"
	"github.com/cshum/imagor/storage/filestorage"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// The image every documented example is served against. Examples name files that
// exist on someone else's server, or placeholders such as IMAGE, so what gets
// checked is the shape of the path rather than the file.
const exampleImage = "gopher-front.png"

var (
	codeSpan = regexp.MustCompile(`<code>(.*?)</code>`)
	backtick = regexp.MustCompile("`([^`\n]+)`")
	// A path needs a separator and something marking it as a transformation,
	// which keeps bare dimensions and bare filter names out.
	pathMarker = regexp.MustCompile(`(^|/)(fit-in|stretch|\d+x\d+|filters:)`)
	// Elisions, placeholders, URLs and prose are documentation shorthand.
	notRunnable = regexp.MustCompile(`(\.\.\.|\bNAME\(|\bARGS\b|https?://|\s)`)
	// The docs also give examples as a bare filter, to be placed in a path.
	filterFragment = regexp.MustCompile(`^[a-z_]+\(.*\)$`)
)

// Examples that cannot be served against a local PNG, with the reason. Anything
// listed here is checked by hand instead, so keep it short; the test fails if an
// entry no longer matches an example, which stops the list going stale.
var exampleSkips = map[string]string{
	"filters:format(svg)/gopher-front.png": "documented to return 400 unless the source is an SVG",
}

type documentedExample struct {
	file string
	line int
	path string
}

// documentedExamples collects the example paths written in the docs, in file
// order, deduplicated by path.
func documentedExamples(t *testing.T) []documentedExample {
	t.Helper()
	files, err := filepath.Glob(filepath.Join("..", "..", "docs", "docs", "*.md"))
	require.NoError(t, err)
	require.NotEmpty(t, files, "no docs found to check")

	var out []documentedExample
	seen := map[string]bool{}
	for _, file := range files {
		src, err := os.ReadFile(file)
		require.NoError(t, err)
		for i, line := range strings.Split(string(src), "\n") {
			for _, re := range []*regexp.Regexp{codeSpan, backtick} {
				for _, m := range re.FindAllStringSubmatch(line, -1) {
					path, ok := runnablePath(html.UnescapeString(strings.TrimSpace(m[1])))
					if !ok || seen[path] {
						continue
					}
					seen[path] = true
					out = append(out, documentedExample{
						file: filepath.Base(file), line: i + 1, path: path,
					})
				}
			}
		}
	}
	return out
}

// runnablePath turns a documented example into a path that can be served, or
// reports that the example is shorthand rather than a path.
//
// The image is replaced textually rather than by parsing and re-generating,
// because generating normalises the path: parameters are always written before
// filters, so a round trip would quietly repair the ordering mistakes these
// examples are checked for.
func runnablePath(raw string) (string, bool) {
	if notRunnable.MatchString(raw) {
		return "", false
	}
	// A bare filter is given as the thing to put in a path, so put it in one.
	if filterFragment.MatchString(raw) {
		raw = "filters:" + raw + "/" + exampleImage
	}
	if !strings.Contains(raw, "/") || !pathMarker.MatchString(raw) {
		return "", false
	}
	return rewriteImages(strings.TrimPrefix(strings.TrimPrefix(raw, "/"), "unsafe/"), 0), true
}

// rewriteImages points the image named by a path, and the image named inside any
// path a filter loads, at the test image. Everything else is left where it was,
// so an example is served in the shape the docs wrote it.
func rewriteImages(path string, depth int) string {
	if depth > 4 {
		return path
	}
	p := imagorpath.Parse(path)
	if p.Image == "" {
		// A snippet, such as a filter reference with no image of its own.
		path = strings.TrimSuffix(path, "/") + "/" + exampleImage
	} else if i := strings.LastIndex(path, p.Image); i >= 0 {
		path = path[:i] + exampleImage + path[i+len(p.Image):]
	}
	for _, f := range p.Filters {
		if f.Name != "image" && f.Name != "watermark" {
			continue
		}
		args := imagorpath.SplitArgs(f.Args)
		if len(args) == 0 || !strings.Contains(path, args[0]) {
			continue
		}
		path = strings.Replace(path, args[0], rewriteImages(args[0], depth+1), 1)
	}
	return path
}

// Every example path written in the docs must process. Readers copy them, so an
// example that returns an error is a documentation bug: it states the wrong
// shape, or the wrong order, and nothing else in the suite would notice.
func TestDocumentedExamplesProcess(t *testing.T) {
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
		imagor.WithProcessors(NewProcessor(WithLogger(zap.NewNop()))),
	)
	require.NoError(t, app.Startup(context.Background()))

	examples := documentedExamples(t)
	require.NotEmpty(t, examples, "no documented examples were found, so the extractor is broken")

	skipped := map[string]bool{}
	for _, ex := range examples {
		t.Run(fmt.Sprintf("%s:%d", ex.file, ex.line), func(t *testing.T) {
			if reason, ok := exampleSkips[ex.path]; ok {
				skipped[ex.path] = true
				t.Skipf("skipped: %s", reason)
			}
			w := httptest.NewRecorder()
			app.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/unsafe/"+ex.path, nil))
			require.Equal(t, http.StatusOK, w.Code,
				"%s:%d documents a path that does not process\n  path: %s\n  says: %s",
				ex.file, ex.line, ex.path, strings.TrimSpace(w.Body.String()))
		})
	}
	for path := range exampleSkips {
		require.True(t, skipped[path],
			"exampleSkips lists %q, which no example produces any more", path)
	}
}

// What the docs say the path is, and what imagor does with it.
//
// Everything after filters: is read as the image, so parameters written after the
// filters are swallowed into the image name and the load fails. The canonical
// form, parameters then filters then image, is what the docs teach; these cases
// pin both halves so the docs and the parser cannot drift apart silently.
//
// The failures are asserted as "not OK" rather than a status code, because the
// code depends on the loader: a local file gives 500 or 406, a remote one gives
// 404.
func TestDocumentedPathOrder(t *testing.T) {
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
		imagor.WithProcessors(NewProcessor(WithLogger(zap.NewNop()))),
	)
	require.NoError(t, app.Startup(context.Background()))

	const img = "gopher-front.png"
	for _, tc := range []struct {
		name   string
		path   string
		wantOK bool
	}{
		{"parameters, then filters", "200x200/filters:fill(yellow)/" + img, true},
		{"filters, then image, no parameters", "filters:fill(yellow)/" + img, true},
		{"nested: parameters, then filters", "filters:image(fit-in/50x50/filters:fill(yellow)/" + img + ",10,10)/" + img, true},
		{"nested: filters, then image, no parameters", "filters:image(filters:fill(yellow)/" + img + ",10,10)/" + img, true},
		{"nested: crop, then parameters", "filters:image(0x0:50x50/fit-in/50x50/" + img + ",10,10)/" + img, true},
		{"parameters written after the filters", "filters:fill(yellow)/200x200/" + img, false},
		{"nested: parameters written after the filters", "filters:image(filters:fill(yellow)/fit-in/50x50/" + img + ",10,10)/" + img, false},
		{"nested: crop written after the filters", "filters:image(filters:fill(yellow)/0x0:50x50/" + img + ",10,10)/" + img, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			app.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/unsafe/"+tc.path, nil))
			if tc.wantOK {
				require.Equal(t, http.StatusOK, w.Code,
					"%s should process\n  path: %s\n  says: %s",
					tc.name, tc.path, strings.TrimSpace(w.Body.String()))
				return
			}
			require.NotEqual(t, http.StatusOK, w.Code,
				"%s now processes, so the docs should stop saying it does not\n  path: %s",
				tc.name, tc.path)
		})
	}
}
