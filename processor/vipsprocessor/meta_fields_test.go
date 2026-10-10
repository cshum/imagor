package vipsprocessor

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/cshum/imagor"
	"github.com/cshum/imagor/storage/filestorage"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// The filter report is always present, empty when there was nothing to report.
// A client that needs to know whether this server reports outcomes has no other
// way to tell: an older server and a server with nothing to report look the same
// when the field is omitted.
func TestMetaAlwaysReportsFilters(t *testing.T) {
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

	w := httptest.NewRecorder()
	app.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/unsafe/meta/100x100/gopher-front.png", nil))
	require.Equal(t, http.StatusOK, w.Code)

	var meta struct {
		Format  string            `json:"format"`
		Filters []json.RawMessage `json:"filters"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &meta))
	require.NotEmpty(t, meta.Format, "the response still describes the image")
	require.NotNil(t, meta.Filters, "the filters field must be present even when empty")
	require.Empty(t, meta.Filters, "this request asked for no filters")
}
