package svg

import (
	"bytes"
	"encoding/xml"
	"errors"
	"io"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func sanitize(t *testing.T, in string) string {
	t.Helper()
	out, err := Sanitize(strings.NewReader(in))
	require.NoError(t, err)
	return string(out)
}

// TestDropsExecutableAndExternalContent covers the constructs that make
// serving an upstream SVG dangerous.
func TestDropsExecutableAndExternalContent(t *testing.T) {
	for _, tc := range []struct {
		name    string
		in      string
		dropped []string
		// refused is the expected reason when the document cannot be served at
		// all, because removing what it names would change how it looks.
		refused string
	}{
		{
			name: "script element",
			in:   `<svg xmlns="http://www.w3.org/2000/svg"><script>alert(1)</script><rect width="1" height="1"/></svg>`,
			dropped: []string{
				"script", "alert(1)",
			},
		},
		{
			name: "script inside nested group",
			in:   `<svg xmlns="http://www.w3.org/2000/svg"><g><defs><script href="x">1</script></defs><rect width="1" height="1"/></g></svg>`,
			dropped: []string{
				"script", "href=\"x\"",
			},
		},
		{
			name: "event handler attributes",
			in:   `<svg xmlns="http://www.w3.org/2000/svg" onload="alert(1)"><rect width="1" height="1" onclick="alert(2)" onmouseover="alert(3)"/></svg>`,
			dropped: []string{
				"onload", "onclick", "onmouseover", "alert",
			},
		},
		{
			name: "foreignObject with html",
			in:   `<svg xmlns="http://www.w3.org/2000/svg"><foreignObject><iframe src="https://evil.test/"></iframe><body xmlns="http://www.w3.org/1999/xhtml"><p>hi</p></body></foreignObject><rect width="1" height="1"/></svg>`,
			dropped: []string{
				"foreignObject", "iframe", "evil.test", "<p>",
			},
		},
		{
			name:    "image and feImage external references",
			in:      `<svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink"><image xlink:href="https://evil.test/a.png" href="https://evil.test/b.png" width="1" height="1"/><filter id="f"><feImage href="https://evil.test/c.png"/></filter></svg>`,
			refused: "a reference to a resource outside the document",
		},
		{
			name:    "javascript and data hrefs on use",
			in:      `<svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink"><use href="javascript:alert(1)"/><use xlink:href="data:text/html;base64,PHNjcmlwdD4="/></svg>`,
			refused: "a reference to a resource outside the document",
		},
		{
			name: "entity obfuscated scheme in a link",
			in:   `<svg xmlns="http://www.w3.org/2000/svg"><a href="&#x6a;avascript:alert(1)"><text>x</text></a></svg>`,
			dropped: []string{
				"javascript", "alert(1)",
			},
		},
		{
			name:    "entity obfuscated scheme in a use",
			in:      `<svg xmlns="http://www.w3.org/2000/svg"><use href="java&#115;cript:alert(1)"/></svg>`,
			refused: "a reference to a resource outside the document",
		},
		{
			name: "animate targeting href",
			in:   `<svg xmlns="http://www.w3.org/2000/svg"><use href="#a"><animate attributeName="href" values="javascript:alert(1)"/></use></svg>`,
			dropped: []string{
				"animate", "javascript",
			},
		},
		{
			name:    "relative and same-origin references",
			in:      `<svg xmlns="http://www.w3.org/2000/svg"><image href="photo.png" width="1" height="1"/><image href="/admin/delete" width="1" height="1"/><rect fill="url(/style.css)" width="1" height="1"/></svg>`,
			refused: "a reference to a resource outside the document",
		},
		{
			name:    "external url in presentation attribute",
			in:      `<svg xmlns="http://www.w3.org/2000/svg"><rect fill="url(https://evil.test/a.svg#x)" stroke="url(http://evil.test/b)" width="1" height="1"/></svg>`,
			refused: "a reference to a resource outside the document",
		},
		{
			name:    "external url in style attribute",
			in:      `<svg xmlns="http://www.w3.org/2000/svg"><rect style="fill:url(https://evil.test/a);stroke:url(//evil.test/b)" width="1" height="1"/></svg>`,
			refused: "an inline style declaration",
		},
		{
			name: "declarations and comments",
			in:   "<?xml version=\"1.0\"?><!DOCTYPE svg PUBLIC \"-//W3C//DTD SVG 1.1//EN\" \"http://www.w3.org/Graphics/SVG/1.1/DTD/svg11.dtd\"><!-- <script>alert(1)</script> --><svg xmlns=\"http://www.w3.org/2000/svg\"><rect width=\"1\" height=\"1\"/></svg>",
			dropped: []string{
				"DOCTYPE", "Graphics/SVG", "<!--", "script",
			},
		},
		{
			name: "unknown elements and foreign namespaces",
			in:   `<svg xmlns="http://www.w3.org/2000/svg" xmlns:ink="http://ink.test/ns"><ink:customelement><ink:child foo="bar"/></ink:customelement><rect width="1" height="1"/></svg>`,
			dropped: []string{
				"customelement", "ink:", "foo",
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			out, err := Sanitize(strings.NewReader(tc.in))
			if tc.refused != "" {
				require.ErrorIs(t, err, ErrRenderingChanged)
				assert.Contains(t, err.Error(), tc.refused)
				assert.Empty(t, out, "a refused document must not be served")
				return
			}
			require.NoError(t, err)
			doc := string(out)
			for _, s := range tc.dropped {
				assert.NotContains(t, doc, s, "sanitized output leaked %q:\n%s", s, doc)
			}
			// The result must always be parseable XML.
			require.NoError(t, xml.Unmarshal(out, new(struct {
				XMLName xml.Name
			})), "not well-formed: %s", doc)
		})
	}
}

// TestPreservesSafePresentation covers the elements and attributes a real
// icon, logo or illustration relies on.
func TestPreservesSafePresentation(t *testing.T) {
	in := `<svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" viewBox="0 0 24 24" width="24" height="24" class="icon" preserveAspectRatio="xMidYMid meet">
	<title>Icon</title>
	<desc>A description</desc>
	<defs>
		<linearGradient id="g" x1="0" y1="0" x2="1" y2="1"><stop offset="0" stop-color="#fff" stop-opacity="0.5"/><stop offset="1" stop-color="#000"/></linearGradient>
		<symbol id="sym"><path d="M0 0 L10 10" fill="none" stroke="#333" stroke-width="2" stroke-linecap="round"/></symbol>
		<clipPath id="c"><circle cx="12" cy="12" r="10"/></clipPath>
		<mask id="m"><rect width="24" height="24" fill="#fff"/></mask>
		<filter id="f"><feGaussianBlur in="SourceGraphic" stdDeviation="2"/></filter>
	</defs>
	<g transform="translate(1 1)" clip-path="url(#c)" mask="url(#m)" filter="url(#f)" opacity="0.9">
		<rect x="1" y="1" width="22" height="22" rx="4" fill="url(#g)" fill-rule="evenodd"/>
		<use xlink:href="#sym" href="#sym"/>
		<text x="10" y="10" font-family="sans-serif" font-size="8" text-anchor="middle" systemLanguage="en">Hi &amp; bye &lt;3</text>
		<image xlink:href="data:image/png;base64,iVBORw0KGgo=" width="4" height="4" preserveAspectRatio="none"/>
		<switch><g systemLanguage="en"><path d="M0 0h1v1H0z"/></g></switch>
	</g>
</svg>`

	out := sanitize(t, in)

	for _, want := range []string{
		`viewBox="0 0 24 24"`, // attribute case survives (HTML tokenizers lowercase it)
		`preserveAspectRatio="xMidYMid meet"`,
		`class="icon"`,
		`<title>Icon</title>`,
		`<desc>A description</desc>`,
		`<linearGradient id="g"`,
		`<stop offset="0" stop-color="#fff" stop-opacity="0.5"/>`,
		`<symbol id="sym">`,
		`d="M0 0 L10 10"`,
		`fill="none"`,
		`stroke-linecap="round"`,
		`<clipPath id="c">`,
		`<mask id="m">`,
		`<filter id="f">`,
		`<feGaussianBlur in="SourceGraphic" stdDeviation="2"/>`,
		`transform="translate(1 1)"`,
		`clip-path="url(#c)"`,
		`mask="url(#m)"`,
		`filter="url(#f)"`,
		`fill="url(#g)"`,
		`fill-rule="evenodd"`,
		`xlink:href="#sym"`,
		`href="#sym"`,
		`font-family="sans-serif"`,
		`text-anchor="middle"`,
		`systemLanguage="en"`,
		`Hi &amp; bye &lt;3`,
		`xlink:href="data:image/png;base64,iVBORw0KGgo="`,
		`<switch>`,
	} {
		assert.Contains(t, out, want, "sanitized output lost %q:\n%s", want, out)
	}
	assert.Contains(t, out, `xmlns="http://www.w3.org/2000/svg"`)
	assert.Contains(t, out, `xmlns:xlink="http://www.w3.org/1999/xlink"`)
	assert.NotContains(t, out, "xmlns:ink")
}

func TestSanitizeIsIdempotent(t *testing.T) {
	in := `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 4 4"><g onload="x()"><script>1</script><rect width="4" height="4" fill="url(#g)"/><use href="#g"/></g></svg>`
	once := sanitize(t, in)
	twice := sanitize(t, once)
	assert.Equal(t, once, twice, "second pass changed the document")
}

func TestStyleAttribute(t *testing.T) {
	for _, tc := range []struct {
		style string
		kept  bool
	}{
		{style: "fill:red", kept: true},
		{style: "fill:red;stroke:blue;stroke-width:2", kept: true},
		{style: "fill:url(#g)", kept: true},
	} {
		t.Run(tc.style, func(t *testing.T) {
			in := `<svg xmlns="http://www.w3.org/2000/svg"><rect style="` + tc.style + `" width="1" height="1"/></svg>`
			assert.Contains(t, sanitize(t, in), `style="`+tc.style+`"`)
		})
	}

	// A declaration that cannot be kept is not simply dropped: the document would
	// then be styled differently from the one that was authored, so it is refused
	// and the caller rasterizes it.
	for _, style := range []string{
		"fill:url(https://evil.test/a)",
		"background:url(//evil.test/b)",
		"background:url(data:image/svg+xml;base64,PHN2Zz4=)",
		"width:expression(alert(1))",
		"background:@import 'x.css'",
	} {
		t.Run(style, func(t *testing.T) {
			in := `<svg xmlns="http://www.w3.org/2000/svg"><rect style="` + style + `" width="1" height="1"/></svg>`
			_, err := Sanitize(strings.NewReader(in))
			assert.ErrorIs(t, err, ErrRenderingChanged)
		})
	}
}

func TestRejectsNonSVGInput(t *testing.T) {
	for _, tc := range []struct{ name, in string }{
		{"empty", ""},
		{"html", `<html><body>hi</body></html>`},
		{"xml non svg root", `<?xml version="1.0"?><rss><channel/></rss>`},
		{"malformed", `<svg xmlns="http://www.w3.org/2000/svg"><g></svg>`},
		{"unclosed", `<svg xmlns="http://www.w3.org/2000/svg"><rect width="1"`},
		{"trailing text", `<svg xmlns="http://www.w3.org/2000/svg"/>garbage`},
		{"code", `package main; func main() {}`},
		{"jpeg bytes", string([]byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 0x4A, 0x46})},
		{"two roots", `<svg xmlns="http://www.w3.org/2000/svg"/><svg xmlns="http://www.w3.org/2000/svg"/>`},
		{"unknown entity", `<svg xmlns="http://www.w3.org/2000/svg"><text>&xxe;</text></svg>`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			out, err := Sanitize(strings.NewReader(tc.in))
			assert.ErrorIs(t, err, ErrInvalidSVG, "expected rejection, got %s", out)
			assert.Nil(t, out)
		})
	}
}

func TestRejectsDeepNesting(t *testing.T) {
	var b strings.Builder
	b.WriteString(`<svg xmlns="http://www.w3.org/2000/svg">`)
	for i := 0; i < maxDepth+2; i++ {
		b.WriteString("<g>")
	}
	for i := 0; i < maxDepth+2; i++ {
		b.WriteString("</g>")
	}
	b.WriteString(`</svg>`)
	_, err := Sanitize(strings.NewReader(b.String()))
	assert.ErrorIs(t, err, ErrInvalidSVG)
}

// TestWhitespaceAndTextEdges checks text handling: characters that are special
// in XML are escaped on output, and input the XML parser refuses (illegal
// characters) is a rejection rather than a pass-through.
func TestWhitespaceAndTextEdges(t *testing.T) {
	out := sanitize(t, "<svg xmlns=\"http://www.w3.org/2000/svg\">\n  <text>a &amp; b &lt; c</text>\n</svg>")
	assert.Contains(t, out, "<text>a &amp; b &lt; c</text>")

	_, err := Sanitize(strings.NewReader("<svg xmlns=\"http://www.w3.org/2000/svg\"><text>a\x00b</text></svg>"))
	assert.ErrorIs(t, err, ErrInvalidSVG, "illegal control character must be rejected")
}

func TestEmptyAndSelfClosingSerialization(t *testing.T) {
	out := sanitize(t, `<svg xmlns="http://www.w3.org/2000/svg"></svg>`)
	assert.Equal(t, `<?xml version="1.0" encoding="UTF-8"?><svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink"/>`, out)
}

// TestImgproxy1708VectorsDropped reproduces the vectors from
// imgproxy/imgproxy#1708 - external references surviving in <style>, <image>
// and <feImage> - plus the attribute forms of the same idea. imgproxy's denylist
// still let these through. Here the document is refused outright, which is the
// strongest answer: the caller rasterizes the source, so nothing that reaches
// outside is served at all.
func TestImgproxy1708VectorsDropped(t *testing.T) {
	const prefix = `<svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" width="8" height="8">`

	for _, tc := range []struct{ name, body string }{
		{"style import", `<style>@import url(https://attacker.example.com/x.css);</style>`},
		{"style background", `<style>.a{background:url(//attacker.example.com/p)}</style>`},
		{"style cdata", `<style><![CDATA[.a{background:url(https://attacker.example.com/p)}]]></style>`},
		{"style attribute", `<rect style="background:url(https://attacker.example.com/p)" width="1" height="1"/>`},
		{"presentation attribute", `<rect fill="url(https://attacker.example.com/s.svg#g)" stroke="url(//attacker.example.com/t)" width="1" height="1"/>`},
		{"image href", `<image href="https://attacker.example.com/i.png" width="8" height="8"/>`},
		{"image xlink href", `<image xlink:href="//attacker.example.com/j.png" width="8" height="8"/>`},
		{"feImage href", `<filter id="f"><feImage href="https://attacker.example.com/k.png"/></filter>`},
		{"feImage xlink href", `<filter id="f"><feImage xlink:href="attacker.example.com/l.svg"/></filter>`},
		{"marker and clip urls", `<clipPath id="c"><rect width="1" height="1"/></clipPath><rect clip-path="url(https://attacker.example.com/m.svg#c)" width="1" height="1"/>`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			out, err := Sanitize(strings.NewReader(prefix + tc.body + `</svg>`))
			require.ErrorIs(t, err, ErrRenderingChanged, "the document must not be served")
			assert.Empty(t, out, "a refused document must not be returned")
			assert.NotContains(t, string(out), "attacker.example.com")
		})
	}

	// A link is the exception: it is navigation rather than content, so the
	// element is kept and only the href goes.
	out := sanitize(t, prefix+`<a href="https://attacker.example.com/"><rect width="8" height="8"/></a></svg>`)
	assert.Contains(t, out, "<a><rect")
	assert.NotContains(t, out, "attacker.example.com")
	assert.NotContains(t, out, "href")
}

// TestImageDataURIRestrictedToRaster keeps embedded raster data and drops an
// embedded document, which this allowlist cannot inspect.
func TestImageDataURIRestrictedToRaster(t *testing.T) {
	for _, tc := range []struct {
		name string
		href string
		kept bool
	}{
		{"png", "data:image/png;base64,iVBORw0KGgo=", true},
		{"jpeg", "data:image/jpeg;base64,/9j/4AAQ", true},
		{"gif", "data:image/gif;base64,R0lGOD", true},
		{"webp", "data:image/webp;base64,UklGRg", true},
		{"uppercase mime", "DATA:IMAGE/PNG;base64,AAAA", true},
		{"fragment", "#icon", true},
		{"svg document", "data:image/svg+xml;base64,PHN2Zz48L3N2Zz4=", false},
		{"svg document unencoded", "data:image/svg+xml,%3Csvg%3E", false},
		{"html", "data:text/html;base64,PHNjcmlwdD4=", false},
		{"mime prefix trick", "data:image/pngx;base64,AAAA", false},
		{"bare type without separator", "data:image/png", false},
		{"external", "https://evil.test/a.png", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			in := `<svg xmlns="http://www.w3.org/2000/svg"><image href="` + tc.href + `" width="8" height="8"/></svg>`
			if tc.kept {
				assert.Contains(t, sanitize(t, in), `href="`+tc.href+`"`)
				return
			}
			// An <image> whose reference cannot be kept would render without the
			// image, so the document is refused and rasterized instead.
			_, err := Sanitize(strings.NewReader(in))
			assert.ErrorIs(t, err, ErrRenderingChanged)
		})
	}
}

// TestAttributeNamespaceHandling covers namespace-qualified attributes: xml:*
// survives, a foreign namespace cannot smuggle in an allowed local name, and an
// xlink attribute other than href is dropped.
func TestAttributeNamespaceHandling(t *testing.T) {
	out := sanitize(t, `<svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" xmlns:evil="http://evil.test/ns" xml:space="preserve">`+
		`<text xml:lang="en" evil:fill="url(http://evil.test/a)" xlink:title="x" xlink:href="#a">x</text></svg>`)

	assert.Contains(t, out, `xml:space="preserve"`)
	assert.Contains(t, out, `xml:lang="en"`)
	assert.Contains(t, out, `xlink:href="#a"`)
	assert.NotContains(t, out, "evil")
	assert.NotContains(t, out, "xlink:title")
}

// TestReferenceValueEdges walks the accept/reject boundary for href and the
// attributes that may hold a url() reference.
func TestReferenceValueEdges(t *testing.T) {
	const prefix = `<svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink">`

	for _, tc := range []struct {
		name string
		body string
		kept string
		// refused marks a value whose removal would change how the document
		// looks: what it referenced would no longer be drawn, so the document
		// is not served at all.
		refused bool
	}{
		{"fragment", `<use href="#sym"/>`, `href="#sym"`, false},
		{"fragment with xlink", `<use xlink:href="#sym"/>`, `xlink:href="#sym"`, false},
		{"bare hash", `<use href="#"/>`, "", true},
		{"fragment with space", `<use href="# a"/>`, "", true},
		{"fragment with quote", `<use href="#a&quot;b"/>`, "", true},
		{"hash not first", `<use href="a#b"/>`, "", true},
		{"javascript", `<use href="javascript:alert(1)"/>`, "", true},
		{"javascript uppercase", `<use href="JavaScript:alert(1)"/>`, "", true},
		{"vbscript", `<use href="vbscript:msgbox(1)"/>`, "", true},
		{"data on use", `<use href="data:image/png;base64,AAAA"/>`, "", true},
		{"url empty", `<rect fill="url()" width="1" height="1"/>`, "", true},
		{"url fragment", `<rect fill="url(#g)" width="1" height="1"/>`, `fill="url(#g)"`, false},
		{"url empty fragment", `<rect fill="url(#)" width="1" height="1"/>`, "", true},
		{"url external", `<rect fill="url(https://evil.test/a)" width="1" height="1"/>`, "", true},
		{"url relative", `<rect fill="url(a.svg#g)" width="1" height="1"/>`, "", true},
		{"bare colour", `<rect fill="#fff" width="1" height="1"/>`, `fill="#fff"`, false},
		{"bare keyword", `<rect fill="none" width="1" height="1"/>`, `fill="none"`, false},
		{"bare rgb", `<rect fill="rgb(1,2,3)" width="1" height="1"/>`, `fill="rgb(1,2,3)"`, false},
		{"bare path", `<rect fill="a/b.png" width="1" height="1"/>`, "", false},
		{"bare scheme", `<rect fill="javascript:alert(1)" width="1" height="1"/>`, "", false},
		{"scheme with control char", `<rect fill="java&#10;script:alert(1)" width="1" height="1"/>`, "", false},
		{"style url fragment", `<rect style="fill:url(#g)" width="1" height="1"/>`, `style="fill:url(#g)"`, false},
		{"style url external", `<rect style="fill:url(https://evil.test/a)" width="1" height="1"/>`, "", true},
		{"style expression", `<rect style="width:expression(alert(1))" width="1" height="1"/>`, "", true},
		{"style import", `<rect style="background:@import url(x.css)" width="1" height="1"/>`, "", true},
		{"style escape", `<rect style="fill:\75 rl(https://evil.test/a)" width="1" height="1"/>`, "", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			in := prefix + tc.body + `</svg>`
			if tc.refused {
				_, err := Sanitize(strings.NewReader(in))
				assert.ErrorIs(t, err, ErrRenderingChanged)
				return
			}
			out := sanitize(t, in)
			if tc.kept != "" {
				assert.Contains(t, out, tc.kept, out)
				return
			}
			// A value that is simply not a reference is dropped: the attribute
			// is invalid rather than pointing somewhere.
			assert.NotContains(t, out, "evil.test", out)
			assert.NotContains(t, out, "javascript", out)
			assert.NotContains(t, out, "expression", out)
			assert.NotContains(t, out, "url(", out)
			assert.NotContains(t, out, "href", out)
		})
	}
}

// TestEntityEncodedValues covers values that arrive character-referenced: the
// XML parser resolves most of them, and decodeEntities handles a second layer,
// so an encoded scheme cannot slip past the check.
func TestEntityEncodedValues(t *testing.T) {
	const prefix = `<svg xmlns="http://www.w3.org/2000/svg">`

	// A reference that is safe once decoded still decodes.
	out := sanitize(t, prefix+`<rect fill="&#35;fff" width="1" height="1"/></svg>`)
	assert.Contains(t, out, `fill="#fff"`, out)

	for _, tc := range []struct {
		name    string
		body    string
		refused bool
	}{
		{"decimal scheme in bare value", `<rect fill="&#106;avascript:alert(1)" width="1" height="1"/>`, false},
		{"hex scheme in bare value", `<rect fill="&#x6a;avascript:alert(1)" width="1" height="1"/>`, false},
		{"encoded colon in url", `<rect fill="url(&#35;x)" width="1" height="1"/>`, false},
		{"double encoded href", `<use href="&amp;#106;avascript:alert(1)"/>`, true},
		{"double encoded hex href", `<use href="&amp;#x6a;avascript:alert(1)"/>`, true},
		{"encoded quote in style", `<rect style="fill:url(&quot;https://evil.test/a&quot;)" width="1" height="1"/>`, true},
		{"double encoded control character", `<rect fill="&amp;#1;x" width="1" height="1"/>`, false},
		{"double encoded surrogate", `<rect fill="&amp;#xD800;x" width="1" height="1"/>`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			in := prefix + tc.body + `</svg>`
			if tc.refused {
				_, err := Sanitize(strings.NewReader(in))
				assert.ErrorIs(t, err, ErrRenderingChanged)
				return
			}
			out := sanitize(t, in)
			require.NoError(t, xml.Unmarshal([]byte(out), new(struct{ XMLName xml.Name })), "output must stay well-formed: %s", out)
			if tc.name == "encoded colon in url" {
				// &#35; is "#": the url() form decodes to url(#x), allowed.
				assert.Contains(t, out, `fill="url(#x)"`, out)
				return
			}
			assert.NotContains(t, out, "javascript", out)
			assert.NotContains(t, out, "evil.test", out)
			// A reference the parser resolved once and decodeEntities resolved
			// again must not reach the output as a control character.
			assert.NotContains(t, out, "\x01")
			assert.NotContains(t, out, "\uFFFD")
		})
	}
}

// TestDecodeEntities covers the second decoding layer directly, including forms
// the XML parser refuses in input and so can only arrive double-encoded.
func TestDecodeEntities(t *testing.T) {
	for _, tc := range []struct{ name, in, want string }{
		{"no ampersand", "red", "red"},
		{"predefined", "a&amp;b&lt;c&gt;d&quot;e&apos;f", `a&b<c>d"e'f`},
		{"decimal", "&#106;avascript:", "javascript:"},
		{"hex", "&#x6a;avascript:", "javascript:"},
		{"uppercase hex marker", "&#X6A;avascript:", "javascript:"},
		{"hash", "&#35;fff", "#fff"},
		{"max rune", "&#x10FFFF;", "\U0010FFFF"},
		{"surrogate", "&#xD800;x", "x"},
		{"control", "&#1;x", "x"},
		{"beyond max rune", "&#x110000;x", "x"},
		{"unknown entity", "&foo;x", "x"},
		{"unterminated", "&#106avascript:", "&#106avascript:"},
		{"no digits", "&#;x", "x"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, decodeEntities(tc.in))
		})
	}
}

// TestCharsetReader covers the declared charsets directly: ISO-8859-1 is mapped,
// UTF-8 and ASCII pass through, and anything else is refused so the caller can
// rasterize instead of guessing.
func TestCharsetReader(t *testing.T) {
	for _, charset := range []string{"utf-8", "UTF-8", "us-ascii", "ascii", "ascii "} {
		r, err := charsetReader(charset, strings.NewReader("x"))
		require.NoError(t, err, charset)
		data, err := io.ReadAll(r)
		require.NoError(t, err)
		assert.Equal(t, "x", string(data))
	}

	for _, charset := range []string{"iso-8859-1", "ISO-8859-1", "latin1", "latin-1"} {
		r, err := charsetReader(charset, strings.NewReader("caf\xe9"))
		require.NoError(t, err, charset)
		data, err := io.ReadAll(r)
		require.NoError(t, err)
		assert.Equal(t, "café", string(data), charset)
	}

	for _, charset := range []string{"windows-1252", "shift_jis", "utf-16", ""} {
		_, err := charsetReader(charset, strings.NewReader("x"))
		assert.ErrorIs(t, err, ErrInvalidSVG, charset)
	}

	// A read failure while converting is reported, not swallowed: the caller
	// must not treat a truncated document as a sanitized one.
	_, err := charsetReader("iso-8859-1", errReader{})
	assert.ErrorIs(t, err, errReaderErr)
}

var errReaderErr = errors.New("read failed")

type errReader struct{}

func (errReader) Read([]byte) (int, error) { return 0, errReaderErr }

// TestLatin1DocumentIsSanitized checks the one charset imagor converts: libvips
// renders these documents, so refusing them would mean refusing to pass through
// something the rasterizer handles.
func TestLatin1DocumentIsSanitized(t *testing.T) {
	out := sanitize(t, "<?xml version=\"1.0\" encoding=\"ISO-8859-1\"?><svg xmlns=\"http://www.w3.org/2000/svg\" width=\"8\" height=\"8\"><text>caf\xe9</text><script>alert(1)</script></svg>")
	assert.Contains(t, out, "café")
	assert.Contains(t, out, `encoding="UTF-8"`)
	assert.NotContains(t, out, "script")

	// A charset that cannot be mapped is a rejection, never a guess.
	_, err := Sanitize(strings.NewReader("<?xml version=\"1.0\" encoding=\"windows-1252\"?><svg xmlns=\"http://www.w3.org/2000/svg\"><text>\x93x\x94</text></svg>"))
	assert.ErrorIs(t, err, ErrInvalidSVG)

	// The converted stream is what the parser validates, so a latin-1 control
	// byte is rejected there rather than reaching serialization.
	_, err = Sanitize(strings.NewReader("<?xml version=\"1.0\" encoding=\"ISO-8859-1\"?><svg xmlns=\"http://www.w3.org/2000/svg\"><text>a\x01b</text></svg>"))
	assert.ErrorIs(t, err, ErrInvalidSVG)
}

// TestEscapingOnOutput covers the serialization of values that must be escaped
// to stay well-formed, and the guard that drops characters XML 1.0 forbids.
func TestEscapingOnOutput(t *testing.T) {
	for _, tc := range []struct{ name, in, want string }{
		{"quote in attribute", `<rect class="a&quot;b" width="1" height="1"/>`, `class="a&quot;b"`},
		{"less than in attribute", `<rect class="a&lt;b" width="1" height="1"/>`, `class="a&lt;b"`},
		{"ampersand in attribute", `<rect class="a&amp;b" width="1" height="1"/>`, `class="a&amp;b"`},
		{"tab in attribute", `<rect class="a&#9;b" width="1" height="1"/>`, `class="a&#9;b"`},
		{"newline in attribute", `<rect class="a&#10;b" width="1" height="1"/>`, `class="a&#10;b"`},
		{"carriage return in attribute", `<rect class="a&#13;b" width="1" height="1"/>`, `class="a&#13;b"`},
		{"greater than in text", `<text>a > b</text>`, `a &gt; b`},
		{"ampersand in text", `<text>a &amp; b</text>`, `a &amp; b`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			out := sanitize(t, `<svg xmlns="http://www.w3.org/2000/svg">`+tc.in+`</svg>`)
			assert.Contains(t, out, tc.want, out)
			require.NoError(t, xml.Unmarshal([]byte(out), new(struct{ XMLName xml.Name })))
		})
	}

	// decodeEntities can produce a control character the parser never vetted,
	// and the serializer must not pass one through.
	out := sanitize(t, `<svg xmlns="http://www.w3.org/2000/svg"><text>a&amp;#1;b</text></svg>`)
	assert.NotContains(t, out, "\x01")
	require.NoError(t, xml.Unmarshal([]byte(out), new(struct{ XMLName xml.Name })))
	assert.False(t, isValidXMLChar(0x1))
	assert.False(t, isValidXMLChar(rune(0xD800)))
	assert.False(t, isValidXMLChar(0x110000))
	assert.True(t, isValidXMLChar(0x20))
	assert.True(t, isValidXMLChar(0x10FFFF))
}

func TestParseErrorEdges(t *testing.T) {
	for _, tc := range []struct{ name, in string }{
		{"stray end tag", `</svg>`},
		{"only whitespace", "\n	  "},
		{"comment only", `<!-- nothing -->`},
		{"end before start", `<g></g><svg xmlns="http://www.w3.org/2000/svg"/>`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Sanitize(strings.NewReader(tc.in))
			assert.ErrorIs(t, err, ErrInvalidSVG)
		})
	}
}

// TestNestedSvgPreserved keeps a legitimate nested viewport.
func TestNestedSvgPreserved(t *testing.T) {
	out := sanitize(t, `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 8 8"><svg x="1" y="1" width="4" height="4" viewBox="0 0 4 4"><rect width="4" height="4"/></svg></svg>`)
	assert.Contains(t, out, `<svg x="1" y="1" width="4" height="4" viewBox="0 0 4 4">`)
}

// TestAnchorKeepsItsContents covers what a real document looks like: a graphic
// wrapped in a link. The container must survive - dropping it would delete the
// graphic - while the link itself goes unless it points inside the document.
func TestAnchorKeepsItsContents(t *testing.T) {
	out := sanitize(t, `<svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" viewBox="0 0 100 100">`+
		`<a xlink:href="http://www.w3.org/Graphics/SVG/" xlink:title="home" target="_parent">`+
		`<title>SVG logo</title><rect width="100" height="100" fill="#FF9900" rx="4" ry="4"/><circle cx="50" cy="18" r="18"/>`+
		`</a></svg>`)
	assert.Contains(t, out, "<rect")
	assert.Contains(t, out, "<circle")
	assert.Contains(t, out, "<title>SVG logo</title>")
	assert.NotContains(t, out, "href")
	assert.NotContains(t, out, "Graphics/SVG")
	assert.NotContains(t, out, "target")
	assert.Contains(t, out, "<a><title>", "the anchor survives as a container without attributes")

	// A same-document link is not a way out of the document, so it stays.
	out = sanitize(t, `<svg xmlns="http://www.w3.org/2000/svg"><defs><rect id="r" width="1" height="1"/></defs><a href="#r"><text>x</text></a></svg>`)
	assert.Contains(t, out, `<a href="#r">`)
	assert.Contains(t, out, "<text>x</text>")

	// Without a link the element is still a container.
	out = sanitize(t, `<svg xmlns="http://www.w3.org/2000/svg"><a><rect width="1" height="1"/></a></svg>`)
	assert.Contains(t, out, "<a><rect")
}

// TestDocumentStylingIsRefused covers the one drop that would change what a
// document looks like: a <style> block is the document's own CSS, and serving
// the document without it would show the wrong thing. An inline declaration on
// an element is not document styling and is kept.
func TestDocumentStylingIsRefused(t *testing.T) {
	for _, in := range []string{
		`<svg xmlns="http://www.w3.org/2000/svg"><style>@import url(https://evil.test/x.css); .a{background:url(//evil.test/p)}</style><rect class="a" width="1" height="1"/></svg>`,
		`<svg xmlns="http://www.w3.org/2000/svg"><style>.a{fill:red}</style><rect class="a" width="1" height="1"/></svg>`,
		`<svg xmlns="http://www.w3.org/2000/svg"><defs><style>.a{fill:red}</style></defs><rect width="1" height="1"/></svg>`,
		`<svg xmlns="http://www.w3.org/2000/svg"><foreignObject><style>p{color:red}</style></foreignObject></svg>`,
	} {
		_, err := Sanitize(strings.NewReader(in))
		assert.ErrorIs(t, err, ErrRenderingChanged)
	}

	out := sanitize(t, `<svg xmlns="http://www.w3.org/2000/svg"><rect style="fill:#fff" width="1" height="1"/></svg>`)
	assert.Contains(t, out, `style="fill:#fff"`)
}

// TestGradientAndBackgroundAttributesPreserved keeps the attributes that decide
// how a document looks and cannot carry a reference: the radial gradient focal
// point, and the SVG 1.1 background and pointer attributes.
func TestGradientAndBackgroundAttributesPreserved(t *testing.T) {
	out := sanitize(t, `<svg xmlns="http://www.w3.org/2000/svg">`+
		`<defs><radialGradient id="g" cx="0.5" cy="0.5" fx="0.3" fy="0.7" fr="0.1" spreadMethod="pad">`+
		`<stop offset="0" stop-color="#fff"/></radialGradient></defs>`+
		`<rect width="1" height="1" fill="url(#g)" enable-background="new" pointer-events="none" cursor="crosshair"/></svg>`)
	for _, want := range []string{
		`fx="0.3"`, `fy="0.7"`, `fr="0.1"`, `spreadMethod="pad"`,
		`enable-background="new"`, `pointer-events="none"`, `cursor="crosshair"`,
	} {
		assert.Contains(t, out, want, out)
	}
}

// TestFilterParametersPreserved keeps the parameters a filter primitive is
// configured with. Dropping them does not remove the filter, it applies it with
// default values, and the result differs from the document that was authored.
func TestFilterParametersPreserved(t *testing.T) {
	out := sanitize(t, `<svg xmlns="http://www.w3.org/2000/svg"><filter id="f">`+
		`<feTurbulence type="fractalNoise" baseFrequency="0.05" numOctaves="3" seed="7" stitchTiles="stitch" result="n"/>`+
		`<feSpecularLighting specularConstant="0.8" specularExponent="12" surfaceScale="3" lighting-color="#fff">`+
		`<fePointLight x="1" y="2" z="3"/></feSpecularLighting>`+
		`<feMorphology radius="2" operator="erode" preserveAlpha="true"/>`+
		`<feComposite in="n" in2="SourceGraphic" operator="in" k1="0.1"/>`+
		`<feImage href="#n" bottomLeftOrigin="true"/>`+
		`</filter></svg>`)
	for _, want := range []string{
		`type="fractalNoise"`, `baseFrequency="0.05"`, `numOctaves="3"`, `seed="7"`,
		`stitchTiles="stitch"`, `result="n"`, `specularConstant="0.8"`,
		`specularExponent="12"`, `surfaceScale="3"`, `z="3"`, `radius="2"`,
		`preserveAlpha="true"`, `in="n"`, `in2="SourceGraphic"`, `k1="0.1"`,
		`bottomLeftOrigin="true"`, `href="#n"`,
	} {
		assert.Contains(t, out, want, out)
	}
}

// TestFilterImageAndTextReference covers the two primitives that carry a
// reference: feImage embeds an image into a filter and takes the same values as
// <image>, tref pulls in the text of another element.
func TestFilterImageAndTextReference(t *testing.T) {
	out := sanitize(t, `<svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink">`+
		`<defs><text id="t">label</text></defs>`+
		`<filter id="g"><feImage href="data:image/png;base64,iVBORw0KGgo=" width="4" height="4"/></filter>`+
		`<rect width="1" height="1" filter="url(#g)"/><text><tref href="#t"/></text></svg>`)
	assert.Contains(t, out, "<feImage")
	assert.Contains(t, out, `href="data:image/png;base64,iVBORw0KGgo="`)
	assert.Contains(t, out, "<tref")
	assert.Contains(t, out, `href="#t"`)
	assert.NotContains(t, out, `href="http`)

	// A reference either of them cannot keep means the document would render
	// without what it points at, so it is not served at all.
	for _, in := range []string{
		`<svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink"><filter id="f"><feImage xlink:href="https://evil.test/a.png" width="4" height="4"/></filter></svg>`,
		`<svg xmlns="http://www.w3.org/2000/svg"><text><tref href="https://evil.test/x"/></text></svg>`,
		`<svg xmlns="http://www.w3.org/2000/svg"><filter><feImage href="data:image/svg+xml;base64,PHN2Zz4="/></filter></svg>`,
		`<svg xmlns="http://www.w3.org/2000/svg"><image href="data:image/svg+xml;base64,PHN2Zz4=" width="4" height="4"/></svg>`,
	} {
		_, err := Sanitize(strings.NewReader(in))
		assert.ErrorIs(t, err, ErrRenderingChanged)
	}
}

// FuzzSanitize asserts the properties the sanitizer promises for arbitrary
// input: it never panics, and whatever it accepts is well-formed XML that
// carries no executable element and no reference reaching outside the document.
// Rejection is always an acceptable answer, so only accepted output is checked.
func FuzzSanitize(f *testing.F) {
	for _, s := range []string{
		`<svg xmlns="http://www.w3.org/2000/svg"><rect width="1" height="1"/></svg>`,
		`<svg xmlns="http://www.w3.org/2000/svg"><text>a &amp; b</text><use href="#a"/></svg>`,
		`<svg xmlns="http://www.w3.org/2000/svg" onload="alert(1)"><script>alert(1)</script></svg>`,
		`<svg xmlns="http://www.w3.org/2000/svg"><style>@import url(https://attacker.example.com/)</style></svg>`,
		`<svg xmlns="http://www.w3.org/2000/svg"><image href="https://attacker.example.com/i.png"/></svg>`,
		`<svg xmlns="http://www.w3.org/2000/svg"><filter><feImage href="https://attacker.example.com/k.png"/></filter></svg>`,
		`<svg xmlns="http://www.w3.org/2000/svg"><rect style="background:url(https://attacker.example.com/p)"/></svg>`,
		"<?xml version=\"1.0\" encoding=\"ISO-8859-1\"?><svg xmlns=\"http://www.w3.org/2000/svg\"><text>caf\xe9</text></svg>",
		`<svg xmlns="http://www.w3.org/2000/svg"><text>&amp;#1;</text><text>&amp;#xD800;</text></svg>`,
	} {
		f.Add([]byte(s))
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		out, err := Sanitize(bytes.NewReader(data))
		if err != nil {
			return
		}
		doc := string(out)
		require.NoError(t, xml.Unmarshal(out, new(struct{ XMLName xml.Name })),
			"accepted output must be well-formed XML: %s", doc)
		for _, element := range []string{"<script", "<foreignObject", "<style", "<iframe", "<animate"} {
			assert.NotContains(t, doc, element, "executable element survived: %s", doc)
		}
		// Every attribute of the accepted document is on the allowlist and
		// carries no reference that leaves the document. Text content is
		// escaped, so it cannot carry a reference that has any effect.
		dec := xml.NewDecoder(strings.NewReader(doc))
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
				// The serializer re-emits the namespace declarations.
				if a.Name.Space == "xmlns" || a.Name.Local == "xmlns" {
					continue
				}
				name := a.Name.Local
				if a.Name.Space == xmlNamespace {
					name = "xml:" + name
				} else if a.Name.Space == xlinkNamespace {
					name = "xlink:" + name
				}
				if _, ok := allowedAttrs[name]; !ok && !isHrefAttr(name) {
					t.Fatalf("attribute not on the allowlist: %s on <%s>: %s", name, el.Name.Local, doc)
				}
				// A value that names a reference must stay inside the document.
				v := strings.ToLower(a.Value)
				if strings.Contains(v, "url(") {
					for _, target := range testURLTargets(v) {
						if !strings.HasPrefix(target, "#") {
							t.Fatalf("url() outside the document: %s on <%s>: %s", a.Value, el.Name.Local, doc)
						}
					}
				}
				if strings.Contains(v, "javascript:") {
					t.Fatalf("javascript: survived in an attribute: %s on <%s>: %s", a.Value, el.Name.Local, doc)
				}
				if isHrefAttr(name) {
					if v == "" || strings.HasPrefix(v, "#") {
						continue
					}
					if el.Name.Local == "image" && isRasterDataURI(a.Value) {
						continue
					}
					t.Fatalf("href left the document: %s on <%s>: %s", a.Value, el.Name.Local, doc)
				}
			}
		}
	})
}

var testURLRe = regexp.MustCompile(`(?i)url\(\s*['"]?([^'")]*)`)

// testURLTargets lists the targets of the url() references in a value.
func testURLTargets(v string) []string {
	var targets []string
	for _, m := range testURLRe.FindAllStringSubmatch(v, -1) {
		targets = append(targets, strings.TrimSpace(m[1]))
	}
	return targets
}
