package awsconfig

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadBucketRouterFromYAML(t *testing.T) {
	t.Run("full config with all fields", func(t *testing.T) {
		yaml := `
default_bucket:
  name: default-bucket
  region: us-east-1
  endpoint: https://s3.us-east-1.amazonaws.com

fallback_buckets:
  - name: fallback-1
    region: us-west-1
  - name: fallback-2
    region: eu-west-1
    endpoint: https://s3.eu-west-1.amazonaws.com

rules:
  - prefix: users
    bucket:
      name: users-bucket
      region: eu-west-1
  - prefix: products
    bucket:
      name: products-bucket
      region: ap-southeast-1
      access_key_id: AKIAIOSFODNN7EXAMPLE
      secret_access_key: wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
`
		tmpFile := createTempYAML(t, yaml)
		defer os.Remove(tmpFile)

		router, err := LoadBucketRouterFromYAML(tmpFile)
		require.NoError(t, err)

		defaultCfg := router.DefaultConfig()
		assert.Equal(t, "default-bucket", defaultCfg.Name)
		assert.Equal(t, "us-east-1", defaultCfg.Region)
		assert.Equal(t, "https://s3.us-east-1.amazonaws.com", defaultCfg.Endpoint)

		fallbacks := router.Fallbacks()
		assert.Len(t, fallbacks, 2)
		assert.Equal(t, "fallback-1", fallbacks[0].Name)
		assert.Equal(t, "us-west-1", fallbacks[0].Region)
		assert.Equal(t, "fallback-2", fallbacks[1].Name)
		assert.Equal(t, "eu-west-1", fallbacks[1].Region)

		usersCfg := router.ConfigFor("users/123/image.jpg")
		assert.Equal(t, "users-bucket", usersCfg.Name)
		assert.Equal(t, "eu-west-1", usersCfg.Region)

		productsCfg := router.ConfigFor("products/456/image.jpg")
		assert.Equal(t, "products-bucket", productsCfg.Name)
		assert.Equal(t, "ap-southeast-1", productsCfg.Region)
		assert.Equal(t, "AKIAIOSFODNN7EXAMPLE", productsCfg.AccessKeyID)
		assert.Equal(t, "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY", productsCfg.SecretAccessKey)

		otherCfg := router.ConfigFor("other/image.jpg")
		assert.Equal(t, "default-bucket", otherCfg.Name)
	})

	t.Run("minimal config", func(t *testing.T) {
		yaml := `
default_bucket:
  name: my-bucket
  region: us-east-1
`
		tmpFile := createTempYAML(t, yaml)
		defer os.Remove(tmpFile)

		router, err := LoadBucketRouterFromYAML(tmpFile)
		require.NoError(t, err)

		assert.Equal(t, "my-bucket", router.DefaultConfig().Name)
		assert.Empty(t, router.Fallbacks())

		cfg := router.ConfigFor("any/path")
		assert.Equal(t, "my-bucket", cfg.Name)
	})

	t.Run("fallback limited to 2", func(t *testing.T) {
		yaml := `
default_bucket:
  name: default
  region: us-east-1

fallback_buckets:
  - name: fb1
    region: us-east-1
  - name: fb2
    region: us-east-1
  - name: fb3
    region: us-east-1
  - name: fb4
    region: us-east-1
`
		tmpFile := createTempYAML(t, yaml)
		defer os.Remove(tmpFile)

		router, err := LoadBucketRouterFromYAML(tmpFile)
		require.NoError(t, err)

		fallbacks := router.Fallbacks()
		assert.Len(t, fallbacks, 2)
		assert.Equal(t, "fb1", fallbacks[0].Name)
		assert.Equal(t, "fb2", fallbacks[1].Name)
	})

	t.Run("longest prefix wins", func(t *testing.T) {
		yaml := `
default_bucket:
  name: default
  region: us-east-1

rules:
  - prefix: media
    bucket:
      name: media-bucket
      region: us-east-1
  - prefix: media/images
    bucket:
      name: images-bucket
      region: us-east-1
  - prefix: media/images/thumbnails
    bucket:
      name: thumbnails-bucket
      region: us-east-1
`
		tmpFile := createTempYAML(t, yaml)
		defer os.Remove(tmpFile)

		router, err := LoadBucketRouterFromYAML(tmpFile)
		require.NoError(t, err)

		assert.Equal(t, "thumbnails-bucket", router.ConfigFor("media/images/thumbnails/123.jpg").Name)
		assert.Equal(t, "images-bucket", router.ConfigFor("media/images/photo.jpg").Name)
		assert.Equal(t, "media-bucket", router.ConfigFor("media/video.mp4").Name)
		assert.Equal(t, "default", router.ConfigFor("other/file.jpg").Name)
	})

	t.Run("file not found", func(t *testing.T) {
		_, err := LoadBucketRouterFromYAML("/nonexistent/path.yaml")
		assert.Error(t, err)
	})

	t.Run("invalid yaml", func(t *testing.T) {
		tmpFile := createTempYAML(t, "invalid: yaml: content: [")
		defer os.Remove(tmpFile)

		_, err := LoadBucketRouterFromYAML(tmpFile)
		assert.Error(t, err)
	})

	t.Run("backward compat fallback method", func(t *testing.T) {
		yaml := `
default_bucket:
  name: my-fallback
  region: us-east-1
`
		tmpFile := createTempYAML(t, yaml)
		defer os.Remove(tmpFile)

		router, err := LoadBucketRouterFromYAML(tmpFile)
		require.NoError(t, err)

		assert.Equal(t, "my-fallback", router.Fallback())
	})
}

func createTempYAML(t *testing.T, content string) string {
	t.Helper()
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "config.yaml")
	err := os.WriteFile(tmpFile, []byte(content), 0644)
	require.NoError(t, err)
	return tmpFile
}
