package svg

import (
	"fmt"
	"strings"
	"testing"
)

var benchmarkBytes []byte
var benchmarkString string
var benchmarkBool bool

func BenchmarkSanitize(b *testing.B) {
	cases := []struct {
		name string
		body string
	}{
		{"Small", `<path d="M0 0h24v24H0z" fill="#ABC"/><use href="#icon"/>`},
		{"ManyPaths", strings.Repeat(`<path d="M0 0 L10 10 Z" fill="#ABC"/>`, 1000)},
		{"ManyAttrs", strings.Repeat(`<rect x="1" y="2" width="24" height="24" rx="2" fill="red" stroke="blue" stroke-width="2" transform="translate(1 2)" opacity="0.5" class="icon"/>`, 500)},
		{"Text", strings.Repeat(`<text xml:space="preserve">Привет &amp; hello &lt;世界&gt; &#9; &#10;</text>`, 500)},
		{"Styles", strings.Repeat(`<path style="fill:url(#g);stroke:url('#s');opacity:0.5" d="M0 0 L10 10 Z"/>`, 500)},
		{"DiscardedSubtree", `<metadata>` + strings.Repeat(`<item value="unused">discarded text</item>`, 1000) + `</metadata><rect width="1"/>`},
	}
	for _, tc := range cases {
		b.Run(tc.name, func(b *testing.B) {
			benchmarkSanitize(b, `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24">`+tc.body+`</svg>`)
		})
	}
	for _, size := range []int{1 << 10, 1 << 20} {
		b.Run(fmt.Sprintf("EmbeddedImage/%d", size), func(b *testing.B) {
			benchmarkSanitize(b, `<svg><image href="data:image/png;base64,`+strings.Repeat("Ab09", size/4)+`"/></svg>`)
		})
	}
	for _, size := range []int{1024, 8192, 65536} {
		b.Run(fmt.Sprintf("Ampersands/%d", size), func(b *testing.B) {
			benchmarkSanitize(b, `<svg><rect fill="`+strings.Repeat("&amp;", size)+`"/></svg>`)
		})
	}
}

func benchmarkSanitize(b *testing.B, input string) {
	b.Helper()
	if _, err := Sanitize(strings.NewReader(input)); err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.SetBytes(int64(len(input)))
	b.ResetTimer()
	for b.Loop() {
		out, err := Sanitize(strings.NewReader(input))
		if err != nil {
			b.Fatal(err)
		}
		benchmarkBytes = out
	}
}

func BenchmarkSelectLanguage(b *testing.B) {
	input := `<svg xml:lang="en"><style>.a { fill: red }</style>` +
		strings.Repeat(`<text systemLanguage="en" class="a&quot;b">Hello &amp; 世界</text><text systemLanguage="fr">Bonjour</text>`, 500) + `</svg>`
	tags := []string{"en-US"}
	b.ReportAllocs()
	b.SetBytes(int64(len(input)))
	b.ResetTimer()
	for b.Loop() {
		out, err := SelectLanguage(strings.NewReader(input), tags)
		if err != nil {
			b.Fatal(err)
		}
		benchmarkBytes = out
	}
}

func BenchmarkDecodeEntities(b *testing.B) {
	for _, size := range []int{1024, 8192, 65536} {
		b.Run(fmt.Sprintf("NoSemicolon/%d", size), func(b *testing.B) {
			input := strings.Repeat("&", size)
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				benchmarkString = decodeEntities(input)
			}
		})
	}
}

func BenchmarkRasterDataURI(b *testing.B) {
	for _, size := range []int{1024, 1 << 20} {
		b.Run(fmt.Sprint(size), func(b *testing.B) {
			input := "data:image/png;base64," + strings.Repeat("Ab09", size/4)
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				benchmarkBool = isRasterDataURI(input)
			}
		})
	}
}

func BenchmarkImageHrefValue(b *testing.B) {
	for _, tc := range []struct{ name, input string }{
		{"Fragment", "#icon"},
		{"External", "https://example.test/image.png"},
		{"RejectedData", "data:image/svg+xml;base64," + strings.Repeat("Ab09", 1<<18)},
		{"RasterData", "data:image/png;base64," + strings.Repeat("Ab09", 1<<18)},
	} {
		b.Run(tc.name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				benchmarkString, benchmarkBool = imageHrefValue(tc.input)
			}
		})
	}
}

func BenchmarkHasScheme(b *testing.B) {
	for _, tc := range []struct{ name, input string }{
		{"Color", "rgb(1,2,3)"},
		{"URL", "HTTPS://example.test/" + strings.Repeat("Ab09", 256)},
		{"Controls", "java\nscript:alert(1)"},
		{"Unicode", "HİS:payload"},
	} {
		b.Run(tc.name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				benchmarkString, benchmarkBool = hasScheme(tc.input)
			}
		})
	}
}

func BenchmarkStyleValue(b *testing.B) {
	for _, tc := range []struct{ name, input string }{
		{"Plain", "fill:red;stroke:blue;opacity:0.5"},
		{"OneURL", "fill:url(#g)"},
		{"ManyURLs", strings.Repeat("fill:url(#g);", 64)},
		{"EarlyRejection", "fill:url(https://example.test/g);" + strings.Repeat("fill:url(#g);", 64)},
	} {
		b.Run(tc.name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				benchmarkString, benchmarkBool = styleValue(tc.input)
			}
		})
	}
}
