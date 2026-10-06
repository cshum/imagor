package svg

import (
	"encoding/xml"
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
			name: "style element with css imports",
			in:   `<svg xmlns="http://www.w3.org/2000/svg"><style>@import url(https://evil.test/x.css); .a{background:url(//evil.test/p)}</style><rect class="a" width="1" height="1"/></svg>`,
			dropped: []string{
				"style", "@import", "evil.test",
			},
		},
		{
			name: "image and feImage external references",
			in:   `<svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink"><image xlink:href="https://evil.test/a.png" href="https://evil.test/b.png" width="1" height="1"/><filter id="f"><feImage href="https://evil.test/c.png"/></filter></svg>`,
			dropped: []string{
				"evil.test",
			},
		},
		{
			name: "javascript and data hrefs on use",
			in:   `<svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink"><use href="javascript:alert(1)"/><use xlink:href="data:text/html;base64,PHNjcmlwdD4="/></svg>`,
			dropped: []string{
				"javascript", "data:text/html",
			},
		},
		{
			name: "entity obfuscated scheme",
			in:   `<svg xmlns="http://www.w3.org/2000/svg"><a href="&#x6a;avascript:alert(1)"><text>x</text></a><use href="java&#115;cript:alert(1)"/></svg>`,
			dropped: []string{
				"javascript", "alert(1)",
			},
		},
		{
			name: "animate targeting href",
			in:   `<svg xmlns="http://www.w3.org/2000/svg"><use href="#a"><animate attributeName="href" values="javascript:alert(1)"/></use></svg>`,
			dropped: []string{
				"animate", "javascript",
			},
		},
		{
			name: "relative and same-origin references",
			in:   `<svg xmlns="http://www.w3.org/2000/svg"><image href="photo.png" width="1" height="1"/><image href="/admin/delete" width="1" height="1"/><rect fill="url(/style.css)" width="1" height="1"/></svg>`,
			dropped: []string{
				"photo.png", "/admin/delete", "url(/style.css)",
			},
		},
		{
			name: "external url in presentation attribute",
			in:   `<svg xmlns="http://www.w3.org/2000/svg"><rect fill="url(https://evil.test/a.svg#x)" stroke="url(http://evil.test/b)" width="1" height="1"/></svg>`,
			dropped: []string{
				"evil.test",
			},
		},
		{
			name: "external url in style attribute",
			in:   `<svg xmlns="http://www.w3.org/2000/svg"><rect style="fill:url(https://evil.test/a);stroke:url(//evil.test/b)" width="1" height="1"/></svg>`,
			dropped: []string{
				"evil.test",
			},
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
			out := sanitize(t, tc.in)
			for _, s := range tc.dropped {
				assert.NotContains(t, out, s, "sanitized output leaked %q:\n%s", s, out)
			}
			// The result must always be parseable XML.
			require.NoError(t, xml.Unmarshal([]byte(out), new(struct {
				XMLName xml.Name
			})), "not well-formed: %s", out)
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
		{style: "fill:url(https://evil.test/a)", kept: false},
		{style: "background:url(//evil.test/b)", kept: false},
		{style: "background:url(data:image/svg+xml;base64,PHN2Zz4=)", kept: false},
		{style: "width:expression(alert(1))", kept: false},
		{style: "background:@import 'x.css'", kept: false},
	} {
		t.Run(tc.style, func(t *testing.T) {
			in := `<svg xmlns="http://www.w3.org/2000/svg"><rect style="` + tc.style + `" width="1" height="1"/></svg>`
			out := sanitize(t, in)
			if tc.kept {
				assert.Contains(t, out, `style="`+tc.style+`"`, out)
			} else {
				assert.NotContains(t, out, "style=", out)
			}
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
