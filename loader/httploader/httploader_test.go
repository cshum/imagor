package httploader

import (
	"github.com/cshum/imagor"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type testTransport map[string]string

func (t testTransport) RoundTrip(r *http.Request) (w *http.Response, err error) {
	if res, ok := t[r.URL.String()]; ok {
		w = &http.Response{
			StatusCode: http.StatusOK,
			Body:       ioutil.NopCloser(strings.NewReader(res)),
		}
		return
	}
	w = &http.Response{
		StatusCode: http.StatusNotFound,
		Body:       ioutil.NopCloser(strings.NewReader("not found")),
	}
	return
}

type test struct {
	name   string
	target string
	result string
	err    string
}

func doTests(t *testing.T, loader imagor.Loader, tests []test) {
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, tt.target, nil)
			b, err := loader.Load(r, tt.target)
			assert.Equal(t, string(b), tt.result)
			if tt.err == "" {
				assert.NoError(t, err)
			} else {
				assert.EqualError(t, err, tt.err)
			}
		})
	}
}

func TestWithAllowedSources(t *testing.T) {
	doTests(t, New(
		WithTransport(testTransport{
			"https://foo.bar/baz":   "baz",
			"https://foo.boo/boooo": "boom",
			"https://def.def/boo":   "boo",
			"https://foo.abc/bar":   "foobar",
		}),
		WithAllowedSources("foo.bar", "*.abc", "def.def,ghi.ghi"),
	), []test{
		{
			name:   "allowed source",
			target: "https://foo.bar/baz",
			result: "baz",
		},
		{
			name:   "allowed not found",
			target: "https://foo.bar/boooo",
			result: "not found",
			err:    "imagor: 404 Not Found",
		},
		{
			name:   "not allowed source",
			target: "https://foo.boo/boooo",
			err:    "imagor: 400 pass",
		},
		{
			name:   "csv allowed source",
			target: "https://def.def/boo",
			result: "boo",
		},
		{
			name:   "glob allowed source",
			target: "https://foo.abc/bar",
			result: "foobar",
		},
	})
}
