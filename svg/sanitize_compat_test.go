package svg

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
	"unicode"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSerializationCompatibility(t *testing.T) {
	data, err := os.ReadFile("testdata/compatibility.json")
	require.NoError(t, err)
	var cases []struct {
		Name   string
		Input  string
		Output string
		Tags   []string
	}
	require.NoError(t, json.Unmarshal(data, &cases))
	require.NotEmpty(t, cases)
	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			var out []byte
			var err error
			if tc.Tags == nil {
				out, err = Sanitize(strings.NewReader(tc.Input))
			} else {
				out, err = SelectLanguage(strings.NewReader(tc.Input), tc.Tags)
			}
			require.NoError(t, err)
			assert.Equal(t, []byte(tc.Output), out)
		})
	}
}

func TestDecodeEntitiesCompatibility(t *testing.T) {
	for _, position := range []int{11, 12, 13} {
		t.Run(fmt.Sprintf("semicolon-at-%d", position), func(t *testing.T) {
			input := "&#" + strings.Repeat("0", position-4) + "65;"
			want := "A"
			if position > 12 {
				want = input
			}
			assert.Equal(t, want, decodeEntities(input))
		})
	}
	for _, input := range []string{
		strings.Repeat("&", 8192),
		strings.Repeat("&", 8192) + ";",
		"&&&&&&&&&&&&&amp;",
		"&amp;amp;#35;", "&unknown;", "&#x;", "&#X;", "&;", "&#0;", "&#xD800;", "&#x110000;",
		"&#9;&#10;&#13;&#32;&#xD7FF;&#xE000;&#xFFFD;&#x10000;&#x10FFFF;",
	} {
		assert.Equal(t, legacyDecodeEntities(input), decodeEntities(input))
	}
	for _, size := range []int{1024, 8192} {
		input := `<svg><rect fill="` + strings.Repeat("&amp;", size) + `"/></svg>`
		out := sanitize(t, input)
		assert.Equal(t, `<?xml version="1.0" encoding="UTF-8"?><svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink"><rect fill="`+
			strings.Repeat("&amp;", size)+`"/></svg>`, out)
	}
}

func TestRasterDataURICompatibility(t *testing.T) {
	for _, mime := range imageRasterTypes {
		for _, separator := range []string{";base64,", ","} {
			for _, header := range []string{"data:image/" + mime, strings.ToUpper("data:image/" + mime), "data:İmage/" + mime} {
				for _, payload := range []string{"", "Ab09", "世界İ", strings.Repeat("Ab09", 1<<16)} {
					input := header + separator + payload
					require.True(t, isRasterDataURI(input), "header %q", header)
					value, kept := imageHrefValue(input)
					require.True(t, kept)
					require.Equal(t, input, value)
				}
			}
		}
	}
	for _, input := range []string{
		"", "data:image/", "data:image/png", "data:image/pngx,", "data:image/svg+xml,",
		"data:image/png ,", "data:image/pnğ,", "data:ımage/png,", "data:image/ſvg,", "data:image/tıff,",
		"data:image/PNG\x00,", "data:image/png\xff,", "data:İmage/PNG;世界", "DATA:IMAGE/PNG,\xff",
	} {
		assert.Equal(t, legacyRasterDataURI(input), isRasterDataURI(input), "%q", input)
	}
	// Exercise every non-ASCII rune which the old lowercase mapping accepts as
	// a character in a supported header, including mappings that shrink UTF-8.
	const header = "data:image/png,"
	for r := rune(utf8.RuneSelf); r <= utf8.MaxRune; r++ {
		lower := unicode.ToLower(r)
		if lower >= utf8.RuneSelf {
			continue
		}
		for i := range header {
			if rune(header[i]) == lower {
				input := header[:i] + string(r) + header[i+1:]
				require.Equal(t, legacyRasterDataURI(input), isRasterDataURI(input), "rune %U", r)
			}
		}
	}
}

func TestImageHrefValuePayloadValidation(t *testing.T) {
	input := "data:image/png;base64," + strings.Repeat("Ab09", 1<<16)
	for _, suffix := range []string{"", "世界", "<", ">", `"`, "'", `\`, "\n"} {
		value := input + suffix
		want, wantOK := legacyImageHrefValue(value)
		got, gotOK := imageHrefValue(value)
		assert.Equal(t, wantOK, gotOK, "suffix %q", suffix)
		assert.Equal(t, want, got, "suffix %q", suffix)
	}
}

func TestUnservableReasonCompatibility(t *testing.T) {
	const resource = "a reference to a resource outside the document"
	const inline = "an inline style declaration"
	attr := func(name, value string) xml.Attr {
		return xml.Attr{Name: xml.Name{Local: name}, Value: value}
	}
	for _, tc := range []struct {
		name   string
		el     string
		attrs  []xml.Attr
		reason string
	}{
		{"style before attributes", "style", []xml.Attr{attr("href", "external")}, "the document's own CSS"},
		{"font before attributes", "font-face-uri", []xml.Attr{attr("style", "expression(1)")}, "font data outside the document"},
		{"ordinary attributes", "rect", []xml.Attr{attr("d", "url(external)"), attr("onclick", "expression(1)")}, ""},
		{"navigation", "a", []xml.Attr{attr("href", "javascript:alert(1)")}, ""},
		{"non-resource href", "rect", []xml.Attr{attr("href", "external")}, ""},
		{"safe style", "rect", []xml.Attr{attr("style", "fill:url('#g')")}, ""},
		{"unsafe style", "rect", []xml.Attr{attr("style", "fill:URL(https://example.test/g)")}, inline},
		{"style escapes", "rect", []xml.Attr{attr("style", `fill:\75rl(#g)`)}, inline},
		{"style then href", "image", []xml.Attr{attr("style", "expression(1)"), attr("href", "external")}, inline},
		{"href then style", "image", []xml.Attr{attr("href", "external"), attr("style", "expression(1)")}, resource},
		{"foreign namespace", "image", []xml.Attr{{Name: xml.Name{Space: "urn:foreign", Local: "href"}, Value: "external"}}, ""},
		{"xml namespace", "image", []xml.Attr{{Name: xml.Name{Space: xmlNamespace, Local: "href"}, Value: "external"}}, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.reason, unservableReason(xml.StartElement{Name: xml.Name{Local: tc.el}, Attr: tc.attrs}))
		})
	}
	for el := range resourceElements {
		if el == "font-face-uri" {
			continue
		}
		for _, space := range []string{"", xlinkNamespace} {
			for _, value := range []string{"external", "#", "javascript:alert(1)"} {
				a := xml.Attr{Name: xml.Name{Space: space, Local: "href"}, Value: value}
				assert.Equal(t, resource, unservableReason(xml.StartElement{Name: xml.Name{Local: el}, Attr: []xml.Attr{a}}))
			}
		}
	}
	for name := range urlAttrs {
		for _, tc := range []struct{ value, reason string }{
			{"url(https://example.test/g)", resource},
			{"URL(https://example.test/g)", ""},
			{"url (https://example.test/g)", ""},
			{"&#117;rl(https://example.test/g)", ""},
			{"url(#g)", ""},
			{"javascript:alert(1)", ""},
		} {
			assert.Equal(t, tc.reason, unservableReason(xml.StartElement{Name: xml.Name{Local: "rect"}, Attr: []xml.Attr{attr(name, tc.value)}}), "%s=%q", name, tc.value)
		}
	}
}

func TestSanitizeErrorPriority(t *testing.T) {
	for _, tc := range []struct {
		name, input, reason string
		kind                error
	}{
		{"style before wrong root", `<html><style/></html>`, "the document's own CSS", ErrRenderingChanged},
		{"style before malformed tail", `<svg><style/><broken`, "the document's own CSS", ErrRenderingChanged},
		{"resource before style", `<svg><image href="external"/><style/></svg>`, "a reference to a resource outside the document", ErrRenderingChanged},
		{"inline style before fill", `<svg><rect style="expression(1)" fill="url(https://example.test/g)"/></svg>`, "an inline style declaration", ErrRenderingChanged},
		{"fill before inline style", `<svg><rect fill="url(https://example.test/g)" style="expression(1)"/></svg>`, "a reference to a resource outside the document", ErrRenderingChanged},
		{"depth before style", `<svg>` + strings.Repeat(`<g>`, maxDepth-1) + `<style/>`, fmt.Sprintf("exceeds max element depth %d", maxDepth), ErrInvalidSVG},
		{"style at maximum depth", `<svg>` + strings.Repeat(`<g>`, maxDepth-2) + `<style/>`, "the document's own CSS", ErrRenderingChanged},
		{"style in discarded tree", `<svg><metadata><style/></metadata></svg>`, "the document's own CSS", ErrRenderingChanged},
		{"resource in discarded tree", `<svg><metadata><use href="external"/></metadata></svg>`, "a reference to a resource outside the document", ErrRenderingChanged},
		{"syntax before style", `<svg><g></svg><style/>`, "XML syntax error", ErrInvalidSVG},
		{"unclosed safe element", `<svg><g>`, "unexpected EOF", ErrInvalidSVG},
		{"second root", `<svg/><svg/>`, "content after root element", ErrInvalidSVG},
		{"trailing text", `<svg/>text`, "content after root element", ErrInvalidSVG},
	} {
		t.Run(tc.name, func(t *testing.T) {
			out, err := Sanitize(strings.NewReader(tc.input))
			require.ErrorIs(t, err, tc.kind)
			assert.Contains(t, err.Error(), tc.reason)
			assert.Nil(t, out)
		})
	}
	for _, depth := range []int{maxDepth, maxDepth + 1} {
		input := `<svg>` + strings.Repeat(`<g>`, depth-1) + strings.Repeat(`</g>`, depth-1) + `</svg>`
		out, err := Sanitize(strings.NewReader(input))
		if depth == maxDepth {
			require.NoError(t, err)
			require.NotEmpty(t, out)
		} else {
			require.ErrorIs(t, err, ErrInvalidSVG)
			require.Nil(t, out)
		}
	}
}

func TestSanitizeEarlyRefusal(t *testing.T) {
	// MultiReader exposes the first chunk independently of the sentinel reader;
	// read-ahead within that chunk is allowed. No exact byte count is asserted.
	for _, tc := range []struct{ element, reason string }{
		{`<style/>`, "the document's own CSS"},
		{`<font-face-uri/>`, "font data outside the document"},
		{`<use href="external"/>`, "a reference to a resource outside the document"},
		{`<rect style="expression(1)"/>`, "an inline style declaration"},
	} {
		t.Run(tc.element, func(t *testing.T) {
			tail := &refusalTailReader{}
			out, err := Sanitize(io.MultiReader(strings.NewReader(`<svg>`+tc.element), tail))
			require.ErrorIs(t, err, ErrRenderingChanged)
			assert.Equal(t, ErrRenderingChanged.Error()+": "+tc.reason, err.Error())
			assert.Nil(t, out)
			assert.False(t, tail.read)
		})
	}
}

type refusalTailReader struct{ read bool }

func (r *refusalTailReader) Read([]byte) (int, error) {
	r.read = true
	return 0, errors.New("read past refusal")
}

func TestHasSchemeCompatibility(t *testing.T) {
	for _, input := range []string{
		"", "#icon", "red", "rgb(1,2,3)", "HTTPS://example.test/", "a+b.c-d:tail",
		"1http:x", "h_ttp:x", ":", "java\nscript:alert(1)", " \x00\x1f\x7fHTTP\tS:\nvalue",
		"HİS:payload", "K:payload", "ſ:payload", "https://世界", "h\xff:x", "http\u00a0s:x",
	} {
		want, wantOK := legacyHasScheme(input)
		got, gotOK := hasScheme(input)
		assert.Equal(t, want, got, "%q", input)
		assert.Equal(t, wantOK, gotOK, "%q", input)
	}
}

func TestHasSchemeCharacterBoundaries(t *testing.T) {
	check := func(input string) {
		t.Helper()
		want, wantOK := legacyHasScheme(input)
		got, gotOK := hasScheme(input)
		require.Equal(t, want, got, "input %q", input)
		require.Equal(t, wantOK, gotOK, "input %q", input)
	}
	for c := 0; c <= 255; c++ {
		character := string([]byte{byte(c)})
		for _, input := range []string{
			character + ":payload",
			"a" + character + ":payload",
			"a" + character + "Z:payload",
			"http:" + character,
		} {
			check(input)
		}
		if c <= ' ' || c == 0x7f {
			for _, prefix := range []string{"", "ht", "https"} {
				input := prefix + character + "https"[len(prefix):] + ":payload"
				got, ok := hasScheme(input)
				require.True(t, ok, "input %q", input)
				require.Equal(t, "https", got, "input %q", input)
			}
		}
	}
	for _, character := range []string{"İ", "K", "ſ", "世界", "\u00a0", "\ufffd"} {
		for _, input := range []string{character + ":payload", "a" + character + ":payload", "HTTP:" + character} {
			check(input)
		}
	}
}

func TestMultipleURLCompatibility(t *testing.T) {
	for _, tc := range []struct {
		name, value string
		kept        bool
	}{
		{"local references", `url(#a) url('#b') URL( "#c" )`, true},
		{"external second", `url(#a) url(https://example.test/b)`, false},
		{"external middle", `url(#a) url(//example.test/b) url(#c)`, false},
		{"external last", strings.Repeat(`url(#a) `, 32) + `url(https://example.test/b)`, false},
		{"empty second", `url(#a) url()`, false},
		{"empty fragment second", `url(#a) url(#)`, false},
		{"unterminated local quote", `url(#a) url('#b`, true},
	} {
		for _, name := range []string{"fill", "style"} {
			t.Run(tc.name+"/"+name, func(t *testing.T) {
				value := tc.value
				check, reference := urlAttrValue, legacyURLAttrValue
				reason := "a reference to a resource outside the document"
				if name == "style" {
					value = "fill:" + value
					check, reference = styleValue, legacyStyleValue
					reason = "an inline style declaration"
				}
				want, wantOK := reference(value)
				got, gotOK := check(value)
				require.Equal(t, tc.kept, wantOK)
				require.Equal(t, want, got)
				require.Equal(t, wantOK, gotOK)
				var escaped bytes.Buffer
				require.NoError(t, xml.EscapeText(&escaped, []byte(value)))
				out, err := Sanitize(strings.NewReader(`<svg><rect ` + name + `="` + escaped.String() + `"/></svg>`))
				if !tc.kept {
					require.ErrorIs(t, err, ErrRenderingChanged)
					assert.Equal(t, ErrRenderingChanged.Error()+": "+reason, err.Error())
					assert.Nil(t, out)
					return
				}
				require.NoError(t, err)
				attrValue := strings.ReplaceAll(value, `"`, "&quot;")
				assert.Equal(t, []byte(`<?xml version="1.0" encoding="UTF-8"?><svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink"><rect `+name+`="`+attrValue+`"/></svg>`), out)
			})
		}
	}
}

func FuzzSanitizeHelpersCompatibility(f *testing.F) {
	for _, input := range []string{
		"", "&&&&&", "&#0000000065;", "&amp;amp;#35;", "&#xD800;", "&#x110000;",
		"data:image/png;base64,Ab09", "data:İmage/png,世界", "pngx,", "tiff;", "HİS:payload",
		"java\nscript:x", "\x00HTTP\x7fS:x", "fill:url(#g);stroke:url('#s')",
		"fill:url(https://example.test/g);fill:url(#g)", "url()", `url("#g")`, "url(#g) @IMPORT", "url(#g)\\",
	} {
		f.Add(input)
	}
	f.Fuzz(func(t *testing.T, input string) {
		// The reference entity decoder is quadratic on some inputs. Large-input
		// scaling is measured separately, rather than exhausting fuzz workers.
		if len(input) > 2048 {
			t.Skip()
		}
		require.Equal(t, legacyDecodeEntities(input), decodeEntities(input))
		for _, candidate := range []string{input, "data:image/" + input} {
			require.Equal(t, legacyRasterDataURI(candidate), isRasterDataURI(candidate))
		}
		want, wantOK := legacyImageHrefValue(input)
		got, gotOK := imageHrefValue(input)
		require.Equal(t, want, got)
		require.Equal(t, wantOK, gotOK)
		want, wantOK = legacyHasScheme(input)
		got, gotOK = hasScheme(input)
		require.Equal(t, want, got)
		require.Equal(t, wantOK, gotOK)
		want, wantOK = legacyStyleValue(input)
		got, gotOK = styleValue(input)
		require.Equal(t, want, got)
		require.Equal(t, wantOK, gotOK)
		want, wantOK = legacyURLAttrValue(input)
		got, gotOK = urlAttrValue(input)
		require.Equal(t, want, got)
		require.Equal(t, wantOK, gotOK)
	})
}

func TestEscapingCompatibility(t *testing.T) {
	var all strings.Builder
	for r := rune(0); r <= utf8.MaxRune; r++ {
		if isValidXMLChar(r) {
			all.WriteRune(r)
		}
	}
	textReplacer := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;")
	attrReplacer := strings.NewReplacer("&", "&amp;", "<", "&lt;", `"`, "&quot;", "\t", "&#9;", "\n", "&#10;", "\r", "&#13;")
	for _, input := range []string{"", "plain ASCII", "Привет 世界", "&<>\"'\t\n\r", all.String()} {
		var text, attr bytes.Buffer
		text.WriteString("prefix")
		attr.WriteString("prefix")
		writeEscapedText(&text, []byte(input))
		writeEscapedAttr(&attr, input)
		text.WriteString("suffix")
		attr.WriteString("suffix")
		assert.Equal(t, "prefix"+textReplacer.Replace(input)+"suffix", text.String())
		assert.Equal(t, "prefix"+attrReplacer.Replace(input)+"suffix", attr.String())
	}
}

func TestHrefNameCompatibility(t *testing.T) {
	for _, name := range []string{"", "href", "xlink:href", "HREF", "xml:href", "xlink:", "xlink:xlink:href", "other:href"} {
		assert.Equal(t, name == "href" || name == "xlink:href", isHrefAttr(name), name)
	}
}

func TestSanitizeFragmentedInput(t *testing.T) {
	body := strings.Repeat("世界&amp;&lt;&gt;", 4096)
	input := "\ufeff<svg><text>" + body + "</text></svg>"
	want := `<?xml version="1.0" encoding="UTF-8"?><svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink"><text>` + body + `</text></svg>`
	for _, size := range []int{1, 2, 7, 4096} {
		t.Run(fmt.Sprint(size), func(t *testing.T) {
			out, err := Sanitize(fragmentReader{r: strings.NewReader(input), size: size})
			require.NoError(t, err)
			assert.Equal(t, []byte(want), out)
		})
	}
}

type fragmentReader struct {
	r    io.Reader
	size int
}

func (r fragmentReader) Read(p []byte) (int, error) {
	return r.r.Read(p[:min(len(p), r.size)])
}
