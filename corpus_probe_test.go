package imagor_test

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"image/png"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/cshum/imagor"
	"github.com/cshum/vipsgen/vips"
	"github.com/stretchr/testify/require"
)

// TestRealWorldSVGCorpus is opt-in: SVG_CORPUS=/path/to/svgs. It drives real
// files through the server end to end - no-op request, resize, explicit
// format(svg) - and measures whether sanitization changes what librsvg renders,
// by rasterizing the original and the served document and diffing the pixels.
//
// The corpus used for it was the svgo test suite
// (https://svg.github.io/svgo-test-suite/svgo-test-suite.tar.gz: the W3C SVG 1.1
// test suite, oxygen and charm icons, ~5100 files) plus hand-picked files from
// icon sets and Wikimedia. A document the sanitizer refuses is expected to be
// rasterized and is reported separately; a document that is served but renders
// differently from the original fails the test.
func TestRealWorldSVGCorpus(t *testing.T) {
	dir := os.Getenv("SVG_CORPUS")
	if dir == "" {
		t.Skip("set SVG_CORPUS to a directory of real SVG files")
	}
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)

	mem := map[string][]byte{}
	var names []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".svg") {
			continue
		}
		b, err := os.ReadFile(filepath.Join(dir, e.Name()))
		require.NoError(t, err)
		if imagor.NewBlobFromBytes(b).BlobType() != imagor.BlobTypeSVG {
			continue
		}
		mem[e.Name()] = b
		names = append(names, e.Name())
	}
	require.NotEmpty(t, names)
	sort.Strings(names)

	app := ptApp(t, mem, imagor.WithPassthroughFormats(imagor.BlobTypeSVG))

	var problems, served, rasterized, rendersDiffer int
	refusedNames := map[string]bool{}
	t.Logf("%-20s %-26s %-16s %-16s %-10s %s", "file", "no-op", "100x100", "format(svg)", "render", "notes")
	for _, name := range names {
		original := mem[name]
		notes := []string{}
		if bytes.Contains(original, []byte("<style")) {
			notes = append(notes, "source has <style>")
		}
		if bytes.Contains(original, []byte("<text")) {
			notes = append(notes, "source has <text>")
		}
		if bytes.Contains(original, []byte("<image")) {
			notes = append(notes, "source has <image>")
		}

		// 1. No-op request: the document comes back, sanitized. A source the
		// sanitizer refuses is rasterized instead, which is a fallback rather
		// than a failure.
		noop := ptGet(t, app, name, nil)
		noopType := noop.Header().Get("Content-Type")
		noopCell := fmt.Sprintf("%d %s %dB", noop.Code, shortType(noopType), noop.Body.Len())
		renderCell := "-"
		refused := noop.Code == http.StatusOK && noopType != imagor.SVGContentType
		switch {
		case noop.Code != http.StatusOK:
			problems++
			notes = append(notes, fmt.Sprintf("NO-OP REQUEST FAILED (%d)", noop.Code))
		case refused:
			rasterized++
			refusedNames[name] = true
			renderCell = "rasterized"
		default:
			served++
			body := noop.Body.Bytes()
			for _, p := range ptReferenceProblems(body) {
				problems++
				notes = append(notes, p)
			}
			diff, sizeChanged, err := ptRenderDiff(original, body)
			switch {
			case err != nil:
				problems++
				renderCell = "render error"
				notes = append(notes, err.Error())
			case sizeChanged:
				problems++
				renderCell = "size changed"
			default:
				renderCell = fmt.Sprintf("%.3f%%", diff*100)
				if diff > 0.001 {
					notes = append(notes, "RENDER DIFFERS")
				}
				if diff > 0.01 {
					rendersDiffer++
				}
			}
		}

		// 2. A transformation must still rasterize.
		resized := ptGet(t, app, "100x100/"+name, nil)
		resizeCell := fmt.Sprintf("%d %s", resized.Code, shortType(resized.Header().Get("Content-Type")))
		if resized.Code != http.StatusOK || resized.Header().Get("Content-Type") == imagor.SVGContentType {
			problems++
			notes = append(notes, "RESIZE DID NOT RASTERIZE")
		}

		// 3. Explicit format(svg) answers with the document, or refuses when the
		// document cannot be sanitized - never with a raster.
		explicit := ptGet(t, app, "filters:format(svg)/"+name, nil)
		explicitCell := fmt.Sprintf("%d %s", explicit.Code, shortType(explicit.Header().Get("Content-Type")))
		explicitOK := explicit.Code == http.StatusOK && explicit.Header().Get("Content-Type") == imagor.SVGContentType
		if !explicitOK && !(refused && explicit.Code >= 400) {
			problems++
			notes = append(notes, "EXPLICIT SVG FAILED")
		}

		if len(notes) > 0 {
			t.Logf("%-20s %-26s %-16s %-16s %-10s %s", name, noopCell, resizeCell, explicitCell, renderCell, strings.Join(notes, ", "))
		}
	}
	// Auto format negotiation must not override a decided pass-through. A source
	// the sanitizer refuses was never eligible, so it is not part of this check.
	autoApp := ptApp(t, mem, imagor.WithPassthroughFormats(imagor.BlobTypeSVG), imagor.WithAutoWebP(true))
	for _, name := range names {
		if refusedNames[name] {
			continue
		}
		res := ptGet(t, autoApp, name, http.Header{"Accept": []string{"image/webp,*/*"}})
		if ct := res.Header().Get("Content-Type"); ct != imagor.SVGContentType {
			problems++
			t.Logf("auto-webp overrode pass-through: %s -> %s", name, ct)
		}
	}

	// A still format passes through byte-exact rather than through a codec, and a
	// request that asks for work still goes through one.
	stillMem := map[string][]byte{}
	var stillNames []string
	des, err := os.ReadDir(ptTestDataDir)
	require.NoError(t, err)
	for _, e := range des {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".png") {
			continue
		}
		b, err := os.ReadFile(filepath.Join(ptTestDataDir, e.Name()))
		require.NoError(t, err)
		stillMem[e.Name()] = b
		stillNames = append(stillNames, e.Name())
	}
	require.NotEmpty(t, stillNames)
	stillApp := ptApp(t, stillMem, imagor.WithPassthroughFormats(imagor.BlobTypeSVG, imagor.BlobTypePNG))
	for _, name := range stillNames {
		res := ptGet(t, stillApp, name, nil)
		exact := bytes.Equal(res.Body.Bytes(), stillMem[name])
		resized := ptGet(t, stillApp, "100x100/"+name, nil)
		t.Logf("%-24s %d %-18s byte-exact=%v  resize=%d %s %dB", name,
			res.Code, shortType(res.Header().Get("Content-Type")), exact,
			resized.Code, shortType(resized.Header().Get("Content-Type")), resized.Body.Len())
		if res.Code != http.StatusOK || res.Header().Get("Content-Type") != "image/png" || !exact {
			problems++
			t.Logf("still format not served as-is: %s (%d %s byte-exact=%v)",
				name, res.Code, shortType(res.Header().Get("Content-Type")), exact)
		}
		if resized.Code != http.StatusOK || bytes.Equal(resized.Body.Bytes(), stillMem[name]) {
			problems++
			t.Logf("still format was not processed on demand: %s (%d)", name, resized.Code)
		}
	}

	t.Logf("files=%d served=%d rasterized-instead=%d renders-differ=%d problems=%d",
		len(names), served, rasterized, rendersDiffer, problems)
	if problems > 0 || rendersDiffer > 0 {
		// A document that is served but no longer looks like the one that was
		// authored is the failure this harness exists to catch.
		t.Fatalf("%d problems and %d documents that no longer render as authored", problems, rendersDiffer)
	}
}

func shortType(ct string) string {
	if ct == "" {
		return "(none)"
	}
	return strings.SplitN(ct, ";", 2)[0]
}

// ptReferenceProblems reports what a served document must not contain: an
// executable element, or an attribute that points outside the document. The
// namespace URIs every document carries are not references, which is why this
// reads attributes rather than searching for substrings.
func ptReferenceProblems(doc []byte) []string {
	var problems []string
	for _, element := range []string{"<script", "<foreignObject", "<style", "<iframe"} {
		if bytes.Contains(doc, []byte(element)) {
			problems = append(problems, "SURVIVED "+element)
		}
	}
	dec := xml.NewDecoder(bytes.NewReader(doc))
	for {
		tok, err := dec.Token()
		if err != nil {
			break
		}
		el, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}
		for _, a := range el.Attr {
			if a.Name.Space == "xmlns" || a.Name.Local == "xmlns" {
				continue
			}
			v := strings.ToLower(a.Value)
			if strings.HasPrefix(a.Name.Local, "on") {
				problems = append(problems, "SURVIVED handler "+a.Name.Local)
			}
			if strings.Contains(v, "javascript:") {
				problems = append(problems, "SURVIVED javascript:")
			}
			if strings.Contains(v, "url(") {
				for _, m := range ptURLRe.FindAllStringSubmatch(v, -1) {
					if t := strings.TrimSpace(m[1]); !strings.HasPrefix(t, "#") {
						problems = append(problems, "external url() "+t)
					}
				}
			}
			if a.Name.Local == "href" && v != "" && !strings.HasPrefix(v, "#") &&
				!(el.Name.Local == "image" && strings.HasPrefix(v, "data:image/")) &&
				!(el.Name.Local == "a" && (strings.HasPrefix(v, "http://") ||
					strings.HasPrefix(v, "https://") || strings.HasPrefix(v, "//"))) {
				problems = append(problems, "external href on <"+el.Name.Local+"> "+a.Value)
			}
		}
	}
	return problems
}

var ptURLRe = regexp.MustCompile(`(?i)url\(\s*['"]?([^'")]*)`)

// ptRenderDiff rasterizes both documents and reports the mean absolute
// per-channel difference, 0 when librsvg renders them the same.
func ptRenderDiff(a, b []byte) (diff float64, sizeChanged bool, err error) {
	ia, err := vips.NewSvgloadBuffer(a, nil)
	if err != nil {
		return 0, false, fmt.Errorf("original does not load: %w", err)
	}
	defer ia.Close()
	ib, err := vips.NewSvgloadBuffer(b, nil)
	if err != nil {
		return 0, false, fmt.Errorf("served document does not load: %w", err)
	}
	defer ib.Close()
	if ia.Width() != ib.Width() || ia.Height() != ib.Height() {
		return 0, true, nil
	}
	pa, err := ia.PngsaveBuffer(nil)
	if err != nil {
		return 0, false, err
	}
	pb, err := ib.PngsaveBuffer(nil)
	if err != nil {
		return 0, false, err
	}
	imgA, err := png.Decode(bytes.NewReader(pa))
	if err != nil {
		return 0, false, err
	}
	imgB, err := png.Decode(bytes.NewReader(pb))
	if err != nil {
		return 0, false, err
	}
	bad := imgA.Bounds()
	var total, n float64
	for y := bad.Min.Y; y < bad.Max.Y; y++ {
		for x := bad.Min.X; x < bad.Max.X; x++ {
			r1, g1, b1, a1 := imgA.At(x, y).RGBA()
			r2, g2, b2, a2 := imgB.At(x, y).RGBA()
			for _, d := range []float64{
				float64(r1) - float64(r2), float64(g1) - float64(g2),
				float64(b1) - float64(b2), float64(a1) - float64(a2),
			} {
				if d < 0 {
					d = -d
				}
				total += d
				n++
			}
		}
	}
	return total / n / 65535, false, nil
}
