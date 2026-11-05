package svg

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

const compatibilityBaseline = "fdcc6eaf89531418b0555aa066219c64799ec266"
const generatedDocumentCount = 64

type generatedResult struct {
	Output string `json:"output,omitempty"`
	Error  string `json:"error,omitempty"`
	Kind   string `json:"kind,omitempty"`
}

type generatedDocumentCase struct {
	Seed           int             `json:"seed"`
	Sanitize       generatedResult `json:"sanitize"`
	SelectLanguage generatedResult `json:"select_language"`
}

type generatedDocumentCorpus struct {
	Baseline string                  `json:"baseline"`
	Cases    []generatedDocumentCase `json:"cases"`
}

func TestGeneratedDocumentsCompatibility(t *testing.T) {
	data, err := os.ReadFile("testdata/generated_compatibility.json")
	require.NoError(t, err)
	var corpus generatedDocumentCorpus
	require.NoError(t, json.Unmarshal(data, &corpus))
	require.Equal(t, compatibilityBaseline, corpus.Baseline)
	require.Len(t, corpus.Cases, generatedDocumentCount)
	for i, tc := range corpus.Cases {
		require.Equal(t, i, tc.Seed)
		t.Run(fmt.Sprint(tc.Seed), func(t *testing.T) {
			input, tags := generatedCompatibilityDocument(t, tc.Seed)
			t.Run("Sanitize", func(t *testing.T) {
				out, err := Sanitize(strings.NewReader(input))
				assertGeneratedResult(t, tc.Sanitize, out, err)
			})
			t.Run("SelectLanguage", func(t *testing.T) {
				out, err := SelectLanguage(strings.NewReader(input), tags)
				assertGeneratedResult(t, tc.SelectLanguage, out, err)
			})
		})
	}
}

func assertGeneratedResult(t *testing.T, want generatedResult, out []byte, err error) {
	t.Helper()
	if want.Error != "" {
		require.Error(t, err)
		require.Equal(t, want.Error, err.Error())
		require.Equal(t, want.Kind, generatedErrorKind(err))
		require.Nil(t, out)
		return
	}
	require.NoError(t, err)
	require.Equal(t, []byte(want.Output), out)
}

func generatedErrorKind(err error) string {
	switch {
	case err == nil:
		return ""
	case errors.Is(err, ErrInvalidLanguageTag):
		return "invalid-language-tag"
	case errors.Is(err, ErrRenderingChanged):
		return "rendering-changed"
	case errors.Is(err, ErrInvalidSVG):
		return "invalid-svg"
	default:
		return "other"
	}
}

// The corpus records the original public functions' results, not results
// reconstructed from the optimized helpers. Keep input generation deterministic.
func generatedCompatibilityDocument(t testing.TB, seed int) (string, []string) {
	t.Helper()
	variant := seed / 16
	tags := [][]string{{"en-US"}, {"fr"}, {"zh-CN", "en"}, {"en_US"}}[variant]
	if seed == generatedDocumentCount-1 {
		tags = nil
	}
	escape := func(value string) string {
		var b strings.Builder
		require.NoError(t, xml.EscapeText(&b, []byte(value)))
		return b.String()
	}
	root := `<svg xmlns="http://www.w3.org/2000/svg" xmlns:x="http://www.w3.org/1999/xlink" xmlns:f="urn:foreign" viewBox="0 0 24 24" class="` +
		escape([]string{"icon", "a\tb\nc\rd", `&<>"'`, "世界İ"}[variant]) + `">`
	body := `<defs><path id="icon" d="M0 0h1v1Z"/></defs>` +
		`<text xml:space="preserve">` + escape([]string{"Hello & 世界", "Привет < svg >", "\ttext\n", `"'&<>`}[variant]) + `</text>` +
		`<switch><text systemLanguage="en">English</text><text systemLanguage="fr">Français</text><text systemLanguage="zh">中文</text><text>Default</text></switch>`
	var addition string
	switch seed % 16 {
	case 0:
		addition = `<g><g/><g></g><g><!--comment--></g><g><![CDATA[]]></g></g>`
	case 1:
		addition = `<svg x="1"><use x:href="#icon" f:label="` + escape(`&<>"'`) + `" xml:lang="en"/></svg>`
	case 2:
		addition = `<image href="` + []string{"data:image/png,Ab09", "DATA:IMAGE/JPEG;base64,Ab09", "data:İmage/tiff,世界", "#icon"}[variant] + `"/>`
	case 3:
		addition = `<path style="` + escape(`fill:url('#icon');stroke:URL( "#icon" )`) + `" fill="url(#icon)"/>`
	case 4:
		addition = `<use href="&amp;#35;icon"/><image href="&amp;amp;#35;icon"/><a href="&amp;amp;#35;icon"/>`
	case 5:
		addition = `<metadata><unknown data="value">discard</unknown></metadata><g onclick="alert(1)" f:fill="url(external)"><rect width="2"/></g>`
	case 6:
		return "<?xml version=\"1.0\" encoding=\"ISO-8859-1\"?><svg><text>caf\xe9 &amp; bye</text><rect class=\"&#9;\"/></svg>", tags
	case 7:
		return "\ufeff" + root + body + `</svg>`, tags
	case 8:
		addition = `<metadata><style>.a { fill: red }</style></metadata>`
	case 9:
		addition = `<rect style="expression(1)" fill="url(https://example.test/g)"/>`
		if variant%2 != 0 {
			addition = `<rect fill="url(https://example.test/g)" style="expression(1)"/>`
		}
	case 10:
		return root + `<image href="external"/><broken`, tags
	case 11:
		return `<html>` + body + `</html>`, tags
	case 12:
		return root + `<g></svg>`, tags
	case 13:
		return root + body + `</svg><svg/>`, tags
	case 14:
		depth := maxDepth - 2 + variant%2
		return `<svg>` + strings.Repeat(`<g>`, depth) + `<style/>` + strings.Repeat(`</g>`, depth) + `</svg>`, tags
	case 15:
		return `<?xml version="1.0" encoding="windows-1252"?><svg/>`, tags
	}
	return root + body + addition + `</svg>`, tags
}
