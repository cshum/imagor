package config

import (
	"net/http"
	"testing"
	"time"

	"github.com/cshum/imagor"
	"github.com/cshum/imagor/imagorpath"
	"github.com/cshum/imagor/loader/httploader"
	"github.com/cshum/imagor/loader/uploadloader"
	"github.com/cshum/imagor/metrics/prometheusmetrics"
	"github.com/cshum/imagor/storage/filestorage"
	"github.com/stretchr/testify/assert"
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
	assert.False(t, app.AutoJPEG)
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
		"-imagor-auto-jpeg",
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
	assert.True(t, app.AutoAVIF)
	assert.True(t, app.AutoJPEG)
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

func TestUploadLoader(t *testing.T) {
	// Test default (upload loader disabled)
	srv := CreateServer([]string{})
	app := srv.App.(*imagor.Imagor)
	assert.False(t, app.EnablePostRequests)

	// Verify no upload loader is present by default
	hasUploadLoader := false
	for _, loader := range app.Loaders {
		if _, ok := loader.(*uploadloader.UploadLoader); ok {
			hasUploadLoader = true
			break
		}
	}
	assert.False(t, hasUploadLoader)

	// Test upload loader enabled with defaults
	srv = CreateServer([]string{
		"-upload-loader-enable",
	})
	app = srv.App.(*imagor.Imagor)
	assert.True(t, app.EnablePostRequests)

	// Verify upload loader is present
	hasUploadLoader = false
	for _, loader := range app.Loaders {
		if _, ok := loader.(*uploadloader.UploadLoader); ok {
			hasUploadLoader = true
			break
		}
	}
	assert.True(t, hasUploadLoader)

	// Test upload loader with custom configuration
	srv = CreateServer([]string{
		"-upload-loader-enable",
		"-upload-loader-max-allowed-size", "16777216", // 16MB
		"-upload-loader-accept", "image/jpeg,image/png",
		"-upload-loader-form-field-name", "file",
	})
	app = srv.App.(*imagor.Imagor)
	assert.True(t, app.EnablePostRequests)

	// Verify upload loader is present with custom config
	hasUploadLoader = false
	for _, loader := range app.Loaders {
		if _, ok := loader.(*uploadloader.UploadLoader); ok {
			hasUploadLoader = true
			break
		}
	}
	assert.True(t, hasUploadLoader)

	// Test integration with other options
	srv = CreateServer([]string{
		"-imagor-unsafe",
		"-debug",
		"-upload-loader-enable",
		"-upload-loader-max-allowed-size", "33554432", // 32MB
	})
	app = srv.App.(*imagor.Imagor)
	assert.True(t, app.Unsafe)
	assert.True(t, app.Debug)
	assert.True(t, app.EnablePostRequests)

	// Should have both HTTP loader and upload loader
	httpLoaderCount := 0
	uploadLoaderCount := 0
	for _, loader := range app.Loaders {
		switch loader.(type) {
		case *httploader.HTTPLoader:
			httpLoaderCount++
		case *uploadloader.UploadLoader:
			uploadLoaderCount++
		}
	}
	assert.Equal(t, 1, httpLoaderCount)
	assert.Equal(t, 1, uploadLoaderCount)
}

func TestLoaderPriority(t *testing.T) {
	// Test that file loader comes before HTTP loader
	srv := CreateServer([]string{
		"-file-loader-base-dir", "./testdata",
	})
	app := srv.App.(*imagor.Imagor)

	// Should have file loader first, then HTTP loader
	assert.Equal(t, 2, len(app.Loaders))
	_, isFileLoader := app.Loaders[0].(*filestorage.FileStorage)
	_, isHTTPLoader := app.Loaders[1].(*httploader.HTTPLoader)
	assert.True(t, isFileLoader, "File loader should be first")
	assert.True(t, isHTTPLoader, "HTTP loader should be second")
}

func TestLoaderPriorityWithMultipleLoaders(t *testing.T) {
	// Test that all specific loaders come before HTTP loader (fallback)
	srv := CreateServer([]string{
		"-file-loader-base-dir", "./testdata",
		"-upload-loader-enable",
	})
	app := srv.App.(*imagor.Imagor)

	// Should have: file loader, upload loader, then HTTP loader
	assert.Equal(t, 3, len(app.Loaders))

	_, isFileLoader := app.Loaders[0].(*filestorage.FileStorage)
	_, isUploadLoader := app.Loaders[1].(*uploadloader.UploadLoader)
	_, isHTTPLoader := app.Loaders[2].(*httploader.HTTPLoader)

	assert.True(t, isFileLoader, "File loader should be first")
	assert.True(t, isUploadLoader, "Upload loader should be second")
	assert.True(t, isHTTPLoader, "HTTP loader should be last (fallback)")
}

func TestHTTPLoaderDisabledDoesNotAffectOtherLoaders(t *testing.T) {
	// Test that disabling HTTP loader doesn't affect other loaders
	srv := CreateServer([]string{
		"-file-loader-base-dir", "./testdata",
		"-upload-loader-enable",
		"-http-loader-disable",
	})
	app := srv.App.(*imagor.Imagor)

	// Should have only file and upload loaders, no HTTP loader
	assert.Equal(t, 2, len(app.Loaders))

	_, isFileLoader := app.Loaders[0].(*filestorage.FileStorage)
	_, isUploadLoader := app.Loaders[1].(*uploadloader.UploadLoader)

	assert.True(t, isFileLoader, "File loader should be first")
	assert.True(t, isUploadLoader, "Upload loader should be second")

	// Verify no HTTP loader
	for _, loader := range app.Loaders {
		_, isHTTP := loader.(*httploader.HTTPLoader)
		assert.False(t, isHTTP, "HTTP loader should not be present when disabled")
	}
}

func TestCloudLoadersBeforeHTTP(t *testing.T) {
	// Test that when file and upload loaders are enabled with cloud loaders,
	// HTTP loader is last. This test verifies the loader priority order.
	srv := CreateServer([]string{
		"-file-loader-base-dir", "./testdata",
		"-upload-loader-enable",
	})
	app := srv.App.(*imagor.Imagor)

	// HTTP loader should be the last one
	assert.GreaterOrEqual(t, len(app.Loaders), 2, "Should have multiple loaders")

	// Last loader should be HTTP loader
	lastLoader := app.Loaders[len(app.Loaders)-1]
	_, isHTTPLoader := lastLoader.(*httploader.HTTPLoader)
	assert.True(t, isHTTPLoader, "HTTP loader should be the last loader (fallback)")

	// All loaders before the last should NOT be HTTP loaders
	for i := 0; i < len(app.Loaders)-1; i++ {
		_, isHTTP := app.Loaders[i].(*httploader.HTTPLoader)
		assert.False(t, isHTTP, "Non-HTTP loaders should come before HTTP loader at index %d", i)
	}
}

func TestResponseRawOnError(t *testing.T) {
	srv := CreateServer([]string{
		"-imagor-response-raw-on-error",
	})
	app := srv.App.(*imagor.Imagor)
	assert.True(t, app.ResponseRawOnError)
}
