package gcloudconfig

import (
	"github.com/cshum/imagor"
	"github.com/cshum/imagor/config"
	"github.com/cshum/imagor/storage/gcloudstorage"
	"github.com/fsouza/fake-gcs-server/fakestorage"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

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

	srv := config.CreateServer([]string{
		"-gcloud-safe-chars", "!",

		"-gcloud-loader-bucket", "a",
		"-gcloud-loader-base-dir", "foo",
		"-gcloud-loader-path-prefix", "abcd",
	}, WithGCloud)
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

	srv := config.CreateServer([]string{
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
	}, WithGCloud)
	app := srv.App.(*imagor.Imagor)
	assert.Equal(t, 1, len(app.Loaders))
	storage := app.Storages[0].(*gcloudstorage.GCloudStorage)
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
