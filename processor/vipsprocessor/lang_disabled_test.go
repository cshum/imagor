package vipsprocessor

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/cshum/imagor"
	"github.com/stretchr/testify/require"
)

func TestDisabledLangLeavesSVGUnchanged(t *testing.T) {
	app := imagor.New(
		imagor.WithUnsafe(true),
		imagor.WithLoaders(loaderFunc(func(_ *http.Request, _ string) (*imagor.Blob, error) {
			return imagor.NewBlobFromBytes([]byte(langFixture)), nil
		})),
		imagor.WithProcessors(NewProcessor(WithDisableFilters("lang"))),
	)
	baseline := httptest.NewRecorder()
	app.ServeHTTP(baseline, httptest.NewRequest(http.MethodGet, "/unsafe/filters:format(png)/photo.svg", nil))
	require.Equal(t, http.StatusOK, baseline.Code)
	for _, tag := range []string{"fr", "en", "!!"} {
		t.Run(tag, func(t *testing.T) {
			w := httptest.NewRecorder()
			app.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/unsafe/filters:lang("+tag+"):format(png)/photo.svg", nil))
			require.Equal(t, http.StatusOK, w.Code)
			require.Equal(t, baseline.Body.Bytes(), w.Body.Bytes(), "a disabled lang filter must not rewrite the source")
		})
	}
}
