package vipsprocessor

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/cshum/imagor"
	"github.com/stretchr/testify/require"
)

func TestMetaReportsComputedFilters(t *testing.T) {
	for _, disabled := range []bool{false, true} {
		options := []Option{WithMaxFilterOps(1)}
		if disabled {
			options = append(options, WithDisableFilters("avgcolor", "blurhash", "thumbhash"))
		}
		app := imagor.New(imagor.WithUnsafe(true), imagor.WithProcessors(NewProcessor(options...)))
		w := httptest.NewRecorder()
		app.ServeHTTP(w, httptest.NewRequest(http.MethodGet,
			"/unsafe/meta/16x16/filters:blur(1):blur(2):avgcolor():quality(80):blurhash(2,2):thumbhash():avgcolor()/color:red", nil))
		require.Equal(t, http.StatusOK, w.Code)
		var meta Metadata
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &meta))
		want := []FilterReport{{Name: "blur", Processed: true}}
		if !disabled {
			want = append(want, FilterReport{Name: "avgcolor", Processed: true})
		}
		want = append(want, FilterReport{Name: "quality", Processed: true})
		if !disabled {
			want = append(want, FilterReport{Name: "blurhash", Processed: true}, FilterReport{Name: "thumbhash", Processed: true}, FilterReport{Name: "avgcolor", Processed: true})
		}
		require.Equal(t, want, meta.Filters)
		if !disabled {
			require.NotNil(t, meta.AverageColor)
			require.NotEmpty(t, meta.BlurHash)
			require.NotEmpty(t, meta.ThumbHash)
		} else {
			require.Nil(t, meta.AverageColor)
			require.Empty(t, meta.BlurHash)
			require.Empty(t, meta.ThumbHash)
		}
	}
}

func TestMetaReportsSelectedLanguage(t *testing.T) {
	app := imagor.New(imagor.WithUnsafe(true),
		imagor.WithLoaders(loaderFunc(func(_ *http.Request, _ string) (*imagor.Blob, error) {
			return imagor.NewBlobFromBytes([]byte(langFixture)), nil
		})), imagor.WithProcessors(NewProcessor()))
	w := httptest.NewRecorder()
	app.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/unsafe/meta/filters:lang(fr)/photo.svg", nil))
	require.Equal(t, http.StatusOK, w.Code)
	var meta Metadata
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &meta))
	require.Equal(t, []FilterReport{{Name: "lang", Processed: true}}, meta.Filters)
}
