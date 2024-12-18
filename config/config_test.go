package config

import (
	"github.com/cshum/imagor"
	"github.com/cshum/imagor/imagorpath"
	"github.com/cshum/imagor/loader/httploader"
	"github.com/cshum/imagor/metrics/prometheusmetrics"
	"github.com/cshum/imagor/storage/filestorage"
	"github.com/stretchr/testify/assert"
	"net/http"
	"testing"
	"time"
)

func TestDefault(t *testing.T) {
	srv := CreateServer(nil)
	assert.Equal(t, ":8000", srv.Addr)
	app := srv.App.(*imagor.Imagor)

	assert.False(t, app.Debug)
	assert.False(t, app.Unsafe)
	assert.Equal(t, time.Second*30, app.RequestTimeout)
	assert.Equal(t, time.Second*20, app.LoadTimeout)
	assert.Equal(t, time.Second*20, app.SaveTimeout)
	assert.Equal(t, time.Second*20, app.ProcessTimeout)
	assert.Empty(t, app.BasePathRedirect)
	assert.Empty(t, app.ProcessConcurrency)
	assert.Empty(t, app.BaseParams)
	assert.False(t, app.ModifiedTimeCheck)
	assert.False(t, app.AutoWebP)
	assert.False(t, app.AutoAVIF)
	assert.False(t, app.DisableErrorBody)
	assert.False(t, app.DisableParamsEndpoint)
	assert.Equal(t, time.Hour*24*7, app.CacheHeaderTTL)
	assert.Equal(t, time.Hour*24, app.CacheHeaderSWR)
	assert.Empty(t, app.ResultStorages)
	assert.Empty(t, app.Storages)
	loader := app.Loaders[0].(*httploader.HTTPLoader)
	assert.Empty(t, loader.BaseURL)
	assert.Equal(t, "https", loader.DefaultScheme)
}

func TestBasic(t *testing.T) {
	srv := CreateServer([]string{
		"-debug",
		"-port", "2345",
		"-imagor-secret", "foo",
		"-imagor-unsafe",
		"-imagor-auto-webp",
		"-imagor-auto-avif",
		"-imagor-disable-error-body",
		"-imagor-disable-params-endpoint",
		"-imagor-request-timeout", "16s",
		"-imagor-load-timeout", "7s",
		"-imagor-process-timeout", "19s",
		"-imagor-process-concurrency", "199",
		"-imagor-process-queue-size", "1999",
		"-imagor-base-path-redirect", "https://www.google.com",
		"-imagor-base-params", "filters:watermark(example.jpg)",
		"-imagor-cache-header-ttl", "169h",
		"-imagor-cache-header-swr", "167h",
		"-http-loader-insecure-skip-verify-transport",
		"-http-loader-override-response-headers", "cache-control,content-type",
		"-http-loader-base-url", "https://www.example.com/foo.org",
	})
	app := srv.App.(*imagor.Imagor)

	assert.Equal(t, 2345, srv.Port)
	assert.Equal(t, ":2345", srv.Addr)
	assert.True(t, app.Debug)
	assert.True(t, app.Unsafe)
	assert.True(t, app.AutoWebP)
	assert.True(t, app.DisableErrorBody)
	assert.True(t, app.DisableParamsEndpoint)
	assert.Equal(t, "RrTsWGEXFU2s1J1mTl1j_ciO-1E=", app.Signer.Sign("bar"))
	assert.Equal(t, time.Second*16, app.RequestTimeout)
	assert.Equal(t, time.Second*7, app.LoadTimeout)
	assert.Equal(t, time.Second*19, app.ProcessTimeout)
	assert.Equal(t, int64(199), app.ProcessConcurrency)
	assert.Equal(t, int64(1999), app.ProcessQueueSize)
	assert.Equal(t, "https://www.google.com", app.BasePathRedirect)
	assert.Equal(t, "filters:watermark(example.jpg)/", app.BaseParams)
	assert.Equal(t, time.Hour*169, app.CacheHeaderTTL)
	assert.Equal(t, time.Hour*167, app.CacheHeaderSWR)

	httpLoader := app.Loaders[0].(*httploader.HTTPLoader)
	assert.True(t, httpLoader.Transport.(*http.Transport).TLSClientConfig.InsecureSkipVerify)
	assert.Equal(t, "https://www.example.com/foo.org", httpLoader.BaseURL.String())
	assert.Equal(t, []string{"cache-control", "content-type"}, httpLoader.OverrideResponseHeaders)
}

func TestVersion(t *testing.T) {
	assert.Empty(t, CreateServer([]string{"-version"}))
}

func TestBind(t *testing.T) {
	srv := CreateServer([]string{
		"-debug",
		"-port", "2345",
		"-bind", ":4567",
	})
	assert.Equal(t, ":4567", srv.Addr)
}

func TestSentry(t *testing.T) {
	srv := CreateServer([]string{
		"-sentry-dsn", "https://12345@sentry.com/123",
	})
	assert.Equal(t, "https://12345@sentry.com/123", srv.SentryDsn)
}

func TestSignerAlgorithm(t *testing.T) {
	srv := CreateServer([]string{
		"-imagor-signer-type", "sha256",
	})
	app := srv.App.(*imagor.Imagor)
	assert.Equal(t, "WN6mgyl8pD4KTy5IDSBs0GcFPaV7-R970JLsd01pqAU=", app.Signer.Sign("bar"))

	srv = CreateServer([]string{
		"-imagor-signer-type", "sha512",
		"-imagor-signer-truncate", "32",
	})
	app = srv.App.(*imagor.Imagor)
	assert.Equal(t, "Kmml5ejnmsn7M7TszYkeM2j5G3bpI7mp", app.Signer.Sign("bar"))
}

func TestCacheHeaderNoCache(t *testing.T) {
	srv := CreateServer([]string{"-imagor-cache-header-no-cache"})
	app := srv.App.(*imagor.Imagor)
	assert.Empty(t, app.CacheHeaderTTL)
}

func TestDisableHTTPLoader(t *testing.T) {
	srv := CreateServer([]string{"-http-loader-disable"})
	app := srv.App.(*imagor.Imagor)
	assert.Empty(t, app.Loaders)
}

func TestFileLoader(t *testing.T) {
	srv := CreateServer([]string{
		"-file-safe-chars", "!",

		"-file-loader-base-dir", "./foo",
		"-file-loader-path-prefix", "abcd",
	})
	app := srv.App.(*imagor.Imagor)
	fileLoader := app.Loaders[0].(*filestorage.FileStorage)
	assert.Equal(t, "./foo", fileLoader.BaseDir)
	assert.Equal(t, "/abcd/", fileLoader.PathPrefix)
	assert.Equal(t, "!", fileLoader.SafeChars)
}

func TestFileStorage(t *testing.T) {
	srv := CreateServer([]string{
		"-file-safe-chars", "!",

		"-file-storage-base-dir", "./foo",
		"-file-storage-path-prefix", "abcd",

		"-file-result-storage-base-dir", "./bar",
		"-file-result-storage-path-prefix", "bcda",
	})
	app := srv.App.(*imagor.Imagor)
	assert.Equal(t, 1, len(app.Loaders))
	storage := app.Storages[0].(*filestorage.FileStorage)
	assert.Equal(t, "./foo", storage.BaseDir)
	assert.Equal(t, "/abcd/", storage.PathPrefix)
	assert.Equal(t, "!", storage.SafeChars)

	resultStorage := app.ResultStorages[0].(*filestorage.FileStorage)
	assert.Equal(t, "./bar", resultStorage.BaseDir)
	assert.Equal(t, "/bcda/", resultStorage.PathPrefix)
	assert.Equal(t, "!", resultStorage.SafeChars)
}

func TestPathStyle(t *testing.T) {
	srv := CreateServer([]string{
		"-imagor-storage-path-style", "digest",
		"-imagor-result-storage-path-style", "digest",
	})
	app := srv.App.(*imagor.Imagor)
	assert.Equal(t, "a9/99/3e364706816aba3e25717850c26c9cd0d89d", app.StoragePathStyle.Hash("abc"))
	assert.Equal(t, "30/fd/be2aa5086e0f0c50ea72dd3859a10d8071ad", app.ResultStoragePathStyle.HashResult(imagorpath.Parse("200x200/abc")))

	srv = CreateServer([]string{
		"-imagor-result-storage-path-style", "suffix",
	})
	app = srv.App.(*imagor.Imagor)
	assert.Equal(t, "abc.30fdbe2aa5086e0f0c50", app.ResultStoragePathStyle.HashResult(imagorpath.Parse("200x200/abc")))

	srv = CreateServer([]string{
		"-imagor-result-storage-path-style", "size",
	})
	app = srv.App.(*imagor.Imagor)
	assert.Equal(t, "abc.30fdbe2aa5086e0f0c50_200x200", app.ResultStoragePathStyle.HashResult(imagorpath.Parse("200x200/abc")))
}

func TestPrometheusBind(t *testing.T) {
	srv := CreateServer([]string{
		"-bind", ":2345",
		"-prometheus-bind", ":6789",
		"-prometheus-path", "/myprom",
	})
	assert.Equal(t, ":2345", srv.Addr)
	pm := srv.Metrics.(*prometheusmetrics.PrometheusMetrics)
	assert.Equal(t, pm.Path, "/myprom")
	assert.Equal(t, pm.Addr, ":6789")
}
