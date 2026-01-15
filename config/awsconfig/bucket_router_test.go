package awsconfig

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadBucketRouterFromYAML(t *testing.T) {
	t.Run("random prefix pattern with bucket codes", func(t *testing.T) {
		yaml := `
routing_pattern: "^[a-f0-9]{4}-(?P<bucket>[A-Za-z0-9]+)-"

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
  - match: B1
    bucket:
      name: singapore-bucket
      region: ap-southeast-1
  - match: US
    bucket:
      name: us-bucket
      region: us-east-1
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

		b1Cfg := router.ConfigFor("f7a3-B1-project-123-image.jpg")
		assert.Equal(t, "singapore-bucket", b1Cfg.Name)
		assert.Equal(t, "ap-southeast-1", b1Cfg.Region)

		usCfg := router.ConfigFor("9bc2-US-project-456-image.jpg")
		assert.Equal(t, "us-bucket", usCfg.Name)
		assert.Equal(t, "us-east-1", usCfg.Region)
		assert.Equal(t, "AKIAIOSFODNN7EXAMPLE", usCfg.AccessKeyID)
		assert.Equal(t, "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY", usCfg.SecretAccessKey)

		unknownCfg := router.ConfigFor("abcd-XX-unknown.jpg")
		assert.Equal(t, "default-bucket", unknownCfg.Name)

		noMatchCfg := router.ConfigFor("no-match.jpg")
		assert.Equal(t, "default-bucket", noMatchCfg.Name)
	})

	t.Run("simple prefix-like pattern", func(t *testing.T) {
		yaml := `
routing_pattern: "^(?P<bucket>[^/]+)/"

default_bucket:
  name: default-bucket
  region: us-east-1

rules:
  - match: users
    bucket:
      name: users-bucket
      region: eu-west-1
  - match: products
    bucket:
      name: products-bucket
      region: ap-southeast-1
`
		tmpFile := createTempYAML(t, yaml)
		defer os.Remove(tmpFile)

		router, err := LoadBucketRouterFromYAML(tmpFile)
		require.NoError(t, err)

		assert.Equal(t, "users-bucket", router.ConfigFor("users/123/image.jpg").Name)
		assert.Equal(t, "products-bucket", router.ConfigFor("products/456/image.jpg").Name)
		assert.Equal(t, "default-bucket", router.ConfigFor("other/image.jpg").Name)
	})

	t.Run("region-based pattern", func(t *testing.T) {
		yaml := `
routing_pattern: "^(?P<bucket>us|eu|ap)/"

default_bucket:
  name: default-bucket
  region: us-east-1

rules:
  - match: us
    bucket:
      name: us-bucket
      region: us-east-1
  - match: eu
    bucket:
      name: eu-bucket
      region: eu-west-1
  - match: ap
    bucket:
      name: ap-bucket
      region: ap-southeast-1
`
		tmpFile := createTempYAML(t, yaml)
		defer os.Remove(tmpFile)

		router, err := LoadBucketRouterFromYAML(tmpFile)
		require.NoError(t, err)

		assert.Equal(t, "us-bucket", router.ConfigFor("us/image.jpg").Name)
		assert.Equal(t, "eu-bucket", router.ConfigFor("eu/image.jpg").Name)
		assert.Equal(t, "ap-bucket", router.ConfigFor("ap/image.jpg").Name)
		assert.Equal(t, "default-bucket", router.ConfigFor("other/image.jpg").Name)
	})

	t.Run("fallback limited to 2", func(t *testing.T) {
		yaml := `
routing_pattern: "^(?P<bucket>[^/]+)/"

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

	t.Run("missing routing_pattern", func(t *testing.T) {
		yaml := `
default_bucket:
  name: my-bucket
  region: us-east-1
`
		tmpFile := createTempYAML(t, yaml)
		defer os.Remove(tmpFile)

		_, err := LoadBucketRouterFromYAML(tmpFile)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "routing_pattern")
	})

	t.Run("invalid routing_pattern regex", func(t *testing.T) {
		yaml := `
routing_pattern: "^[invalid("

default_bucket:
  name: my-bucket
  region: us-east-1
`
		tmpFile := createTempYAML(t, yaml)
		defer os.Remove(tmpFile)

		_, err := LoadBucketRouterFromYAML(tmpFile)
		assert.Error(t, err)
	})

	t.Run("missing bucket capture group", func(t *testing.T) {
		yaml := `
routing_pattern: "^([^/]+)/"

default_bucket:
  name: my-bucket
  region: us-east-1
`
		tmpFile := createTempYAML(t, yaml)
		defer os.Remove(tmpFile)

		_, err := LoadBucketRouterFromYAML(tmpFile)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "(?P<bucket>...)")
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
routing_pattern: "^(?P<bucket>[^/]+)/"

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
