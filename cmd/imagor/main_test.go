package main

import (
	"github.com/cshum/imagor"
	"github.com/cshum/imagor/loader/httploader"
	"github.com/cshum/imagor/storage/filestorage"
	"github.com/cshum/imagor/storage/gcloudstorage"
	"github.com/cshum/imagor/storage/s3storage"
	"github.com/fsouza/fake-gcs-server/fakestorage"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
	"time"
)

func TestDefault(t *testing.T) {
	srv := newServer()
	app := srv.App.(*imagor.Imagor)

	assert.False(t, app.Debug)
	assert.False(t, app.Unsafe)
	assert.Equal(t, time.Second*30, app.RequestTimeout)
	assert.Equal(t, time.Second*20, app.LoadTimeout)
	assert.Equal(t, time.Second*20, app.SaveTimeout)
	assert.Equal(t, time.Second*20, app.ProcessTimeout)
	assert.Equal(t, time.Second*20, app.ProcessTimeout)
	assert.Empty(t, app.BasePathRedirect)
	assert.Empty(t, app.ProcessConcurrency)
	assert.False(t, app.ModifiedTimeCheck)
	assert.False(t, app.AutoWebP)
	assert.False(t, app.AutoAVIF)
	assert.Equal(t, time.Hour*24, app.CacheHeaderTTL)
	assert.Empty(t, app.ResultStorages)
	assert.Empty(t, app.ResultLoaders)
	assert.Empty(t, app.Storages)
	assert.IsType(t, &httploader.HTTPLoader{}, app.Loaders[0])
}

func TestVersion(t *testing.T) {
	assert.Empty(t, newServer("-version"))
}

func TestBasic(t *testing.T) {
	srv := newServer(
		"-debug",
		"-port", "2345",
		"-imagor-secret", "foo",
		"-imagor-unsafe",
		"-imagor-auto-webp",
		"-imagor-auto-avif",
		"-imagor-request-timeout", "16s",
		"-imagor-load-timeout", "7s",
		"-imagor-process-timeout", "19s",
		"-imagor-process-concurrency", "199",
		"-imagor-base-path-redirect", "https://www.google.com",
		"-imagor-cache-header-ttl", "167h",
	)
	app := srv.App.(*imagor.Imagor)

	assert.Equal(t, 2345, srv.Port)
	assert.True(t, app.Debug)
	assert.True(t, app.Unsafe)
	assert.True(t, app.AutoWebP)
	assert.Equal(t, "RrTsWGEXFU2s1J1mTl1j_ciO-1E=", app.Signer.Sign("bar"))
	assert.Equal(t, time.Second*16, app.RequestTimeout)
	assert.Equal(t, time.Second*7, app.LoadTimeout)
	assert.Equal(t, time.Second*19, app.ProcessTimeout)
	assert.Equal(t, int64(199), app.ProcessConcurrency)
	assert.Equal(t, "https://www.google.com", app.BasePathRedirect)
	assert.Equal(t, time.Hour*167, app.CacheHeaderTTL)
}

func TestCacheHeaderNoCache(t *testing.T) {
	srv := newServer("-imagor-cache-header-no-cache")
	app := srv.App.(*imagor.Imagor)
	assert.Empty(t, app.CacheHeaderTTL)
}

func TestDisableHTTPLoader(t *testing.T) {
	srv := newServer("-http-loader-disable")
	app := srv.App.(*imagor.Imagor)
	assert.Empty(t, app.Loaders)
}

func TestFileLoader(t *testing.T) {
	srv := newServer(
		"-file-safe-chars", "!",

		"-file-loader-base-dir", "./foo",
		"-file-loader-path-prefix", "abcd",
	)
	app := srv.App.(*imagor.Imagor)
	fileLoader := app.Loaders[0].(*filestorage.FileStorage)
	assert.Equal(t, "./foo", fileLoader.BaseDir)
	assert.Equal(t, "/abcd/", fileLoader.PathPrefix)
	assert.Equal(t, "!", fileLoader.SafeChars)
}

func TestFileStorage(t *testing.T) {
	srv := newServer(
		"-file-safe-chars", "!",

		"-file-storage-base-dir", "./foo",
		"-file-storage-path-prefix", "abcd",
		"-file-loader-base-dir", "./foo",
		"-file-loader-path-prefix", "abcd",

		"-file-result-storage-base-dir", "./bar",
		"-file-result-storage-path-prefix", "bcda",
	)
	app := srv.App.(*imagor.Imagor)
	loader := app.Loaders[0].(*filestorage.FileStorage)
	storage := app.Storages[0].(*filestorage.FileStorage)
	assert.Equal(t, loader, storage)
	assert.Equal(t, "./foo", storage.BaseDir)
	assert.Equal(t, "/abcd/", storage.PathPrefix)
	assert.Equal(t, "!", storage.SafeChars)

	resultStorage := app.ResultStorages[0].(*filestorage.FileStorage)
	assert.Equal(t, "./bar", resultStorage.BaseDir)
	assert.Equal(t, "/bcda/", resultStorage.PathPrefix)
	assert.Equal(t, "!", resultStorage.SafeChars)
}

func TestS3Loader(t *testing.T) {
	srv := newServer(
		"-aws-region", "asdf",
		"-aws-access-key-id", "asdf",
		"-aws-secret-access-key", "asdf",
		"-s3-endpoint", "asdfasdf",
		"-s3-force-path-style",
		"-s3-safe-chars", "!",

		"-s3-loader-bucket", "a",
		"-s3-loader-base-dir", "foo",
		"-s3-loader-path-prefix", "abcd",
	)
	app := srv.App.(*imagor.Imagor)
	loader := app.Loaders[0].(*s3storage.S3Storage)
	assert.Equal(t, "a", loader.Bucket)
	assert.Equal(t, "/foo/", loader.BaseDir)
	assert.Equal(t, "/abcd/", loader.PathPrefix)
	assert.Equal(t, "!", loader.SafeChars)
}

func TestS3Storage(t *testing.T) {
	srv := newServer(
		"-aws-region", "asdf",
		"-aws-access-key-id", "asdf",
		"-aws-secret-access-key", "asdf",
		"-s3-endpoint", "asdfasdf",
		"-s3-force-path-style",
		"-s3-safe-chars", "!",

		"-s3-loader-bucket", "a",
		"-s3-loader-base-dir", "foo",
		"-s3-loader-path-prefix", "abcd",
		"-s3-storage-bucket", "a",
		"-s3-storage-base-dir", "foo",
		"-s3-storage-path-prefix", "abcd",

		"-s3-result-storage-bucket", "b",
		"-s3-result-storage-base-dir", "bar",
		"-s3-result-storage-path-prefix", "bcda",
	)
	app := srv.App.(*imagor.Imagor)
	loader := app.Loaders[0].(*s3storage.S3Storage)
	storage := app.Storages[0].(*s3storage.S3Storage)
	assert.Equal(t, loader, storage)
	assert.Equal(t, "a", storage.Bucket)
	assert.Equal(t, "/foo/", storage.BaseDir)
	assert.Equal(t, "/abcd/", storage.PathPrefix)
	assert.Equal(t, "!", storage.SafeChars)

	resultStorage := app.ResultStorages[0].(*s3storage.S3Storage)
	assert.Equal(t, "b", resultStorage.Bucket)
	assert.Equal(t, "/bar/", resultStorage.BaseDir)
	assert.Equal(t, "/bcda/", resultStorage.PathPrefix)
	assert.Equal(t, "!", resultStorage.SafeChars)
}

func fakeGCSServer() *fakestorage.Server {
	if err := os.Setenv("STORAGE_EMULATOR_HOST", "localhost:12345"); err != nil {
		panic(err)
	}
	svr, err := fakestorage.NewServerWithOptions(fakestorage.Options{
		Host: "localhost", Port: 12345,
	})
	if err != nil {
		panic(err)
	}
	return svr
}

func TestGCSLoader(t *testing.T) {
	svr := fakeGCSServer()
	defer svr.Stop()

	srv := newServer(
		"-gcloud-safe-chars", "!",

		"-gcloud-loader-bucket", "a",
		"-gcloud-loader-base-dir", "foo",
		"-gcloud-loader-path-prefix", "abcd",
	)
	app := srv.App.(*imagor.Imagor)
	loader := app.Loaders[0].(*gcloudstorage.GCloudStorage)
	assert.Equal(t, "a", loader.Bucket)
	assert.Equal(t, "foo", loader.BaseDir)
	assert.Equal(t, "/abcd/", loader.PathPrefix)
	assert.Equal(t, "!", loader.SafeChars)
}

func TestGCSStorage(t *testing.T) {
	svr := fakeGCSServer()
	defer svr.Stop()

	srv := newServer(
		"-gcloud-safe-chars", "!",

		"-gcloud-loader-bucket", "a",
		"-gcloud-loader-base-dir", "foo",
		"-gcloud-loader-path-prefix", "abcd",
		"-gcloud-storage-bucket", "a",
		"-gcloud-storage-base-dir", "foo",
		"-gcloud-storage-path-prefix", "abcd",

		"-gcloud-result-storage-bucket", "b",
		"-gcloud-result-storage-base-dir", "bar",
		"-gcloud-result-storage-path-prefix", "bcda",
	)
	app := srv.App.(*imagor.Imagor)
	loader := app.Loaders[0].(*gcloudstorage.GCloudStorage)
	storage := app.Storages[0].(*gcloudstorage.GCloudStorage)
	assert.Equal(t, loader, storage)
	assert.Equal(t, "a", storage.Bucket)
	assert.Equal(t, "foo", storage.BaseDir)
	assert.Equal(t, "/abcd/", storage.PathPrefix)
	assert.Equal(t, "!", storage.SafeChars)

	resultStorage := app.ResultStorages[0].(*gcloudstorage.GCloudStorage)
	assert.Equal(t, "b", resultStorage.Bucket)
	assert.Equal(t, "bar", resultStorage.BaseDir)
	assert.Equal(t, "/bcda/", resultStorage.PathPrefix)
	assert.Equal(t, "!", resultStorage.SafeChars)
}
