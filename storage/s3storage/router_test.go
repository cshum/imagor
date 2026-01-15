package s3storage

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPrefixRouter_ConfigFor(t *testing.T) {
	defaultCfg := &BucketConfig{Name: "default-bucket", Region: "us-east-1"}
	usersCfg := &BucketConfig{Name: "users-bucket", Region: "eu-west-1"}
	productsCfg := &BucketConfig{Name: "products-bucket", Region: "ap-southeast-1"}
	vipCfg := &BucketConfig{Name: "vip-bucket", Region: "us-west-2"}

	tests := []struct {
		name           string
		rules          []PrefixRule
		defaultConfig  *BucketConfig
		key            string
		expectedBucket string
	}{
		{
			name:           "empty rules returns default",
			rules:          []PrefixRule{},
			defaultConfig:  defaultCfg,
			key:            "users/123/image.jpg",
			expectedBucket: "default-bucket",
		},
		{
			name: "exact prefix match",
			rules: []PrefixRule{
				{Prefix: "users/", Config: usersCfg},
				{Prefix: "products/", Config: productsCfg},
			},
			defaultConfig:  defaultCfg,
			key:            "users/123/image.jpg",
			expectedBucket: "users-bucket",
		},
		{
			name: "no match returns default",
			rules: []PrefixRule{
				{Prefix: "users/", Config: usersCfg},
				{Prefix: "products/", Config: productsCfg},
			},
			defaultConfig:  defaultCfg,
			key:            "other/123/image.jpg",
			expectedBucket: "default-bucket",
		},
		{
			name: "longest prefix wins",
			rules: []PrefixRule{
				{Prefix: "users/", Config: usersCfg},
				{Prefix: "users/vip/", Config: vipCfg},
			},
			defaultConfig:  defaultCfg,
			key:            "users/vip/123/image.jpg",
			expectedBucket: "vip-bucket",
		},
		{
			name: "strips leading slash from key",
			rules: []PrefixRule{
				{Prefix: "users/", Config: usersCfg},
			},
			defaultConfig:  defaultCfg,
			key:            "/users/123/image.jpg",
			expectedBucket: "users-bucket",
		},
		{
			name: "deep nested prefix",
			rules: []PrefixRule{
				{Prefix: "media/images/thumbnails/", Config: &BucketConfig{Name: "thumbnails-bucket"}},
				{Prefix: "media/images/", Config: &BucketConfig{Name: "images-bucket"}},
				{Prefix: "media/", Config: &BucketConfig{Name: "media-bucket"}},
			},
			defaultConfig:  defaultCfg,
			key:            "media/images/thumbnails/123.jpg",
			expectedBucket: "thumbnails-bucket",
		},
		{
			name: "empty key returns default",
			rules: []PrefixRule{
				{Prefix: "users/", Config: usersCfg},
			},
			defaultConfig:  defaultCfg,
			key:            "",
			expectedBucket: "default-bucket",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := NewPrefixRouter(tt.rules, tt.defaultConfig, nil)
			cfg := router.ConfigFor(tt.key)
			assert.Equal(t, tt.expectedBucket, cfg.Name)
		})
	}
}

func TestPrefixRouter_Fallbacks(t *testing.T) {
	defaultCfg := &BucketConfig{Name: "default-bucket", Region: "us-east-1"}
	fb1 := &BucketConfig{Name: "fallback-1", Region: "us-west-1"}
	fb2 := &BucketConfig{Name: "fallback-2", Region: "eu-west-1"}
	fb3 := &BucketConfig{Name: "fallback-3", Region: "ap-southeast-1"}

	t.Run("returns fallbacks", func(t *testing.T) {
		router := NewPrefixRouter(nil, defaultCfg, []*BucketConfig{fb1, fb2})
		fallbacks := router.Fallbacks()
		assert.Len(t, fallbacks, 2)
		assert.Equal(t, "fallback-1", fallbacks[0].Name)
		assert.Equal(t, "fallback-2", fallbacks[1].Name)
	})

	t.Run("limits to 2 fallbacks", func(t *testing.T) {
		router := NewPrefixRouter(nil, defaultCfg, []*BucketConfig{fb1, fb2, fb3})
		fallbacks := router.Fallbacks()
		assert.Len(t, fallbacks, 2)
	})

	t.Run("empty fallbacks", func(t *testing.T) {
		router := NewPrefixRouter(nil, defaultCfg, nil)
		fallbacks := router.Fallbacks()
		assert.Len(t, fallbacks, 0)
	})
}

func TestPrefixRouter_AllConfigs(t *testing.T) {
	defaultCfg := &BucketConfig{Name: "default-bucket", Region: "us-east-1"}
	usersCfg := &BucketConfig{Name: "users-bucket", Region: "eu-west-1"}
	fb1 := &BucketConfig{Name: "fallback-1", Region: "us-west-1"}

	rules := []PrefixRule{
		{Prefix: "users/", Config: usersCfg},
	}

	router := NewPrefixRouter(rules, defaultCfg, []*BucketConfig{fb1})
	configs := router.AllConfigs()

	assert.Len(t, configs, 3)

	names := make(map[string]bool)
	for _, cfg := range configs {
		names[cfg.Name] = true
	}
	assert.True(t, names["default-bucket"])
	assert.True(t, names["users-bucket"])
	assert.True(t, names["fallback-1"])
}

func TestPrefixRouter_AllConfigs_NoDuplicates(t *testing.T) {
	sharedCfg := &BucketConfig{Name: "shared-bucket", Region: "us-east-1"}

	rules := []PrefixRule{
		{Prefix: "users/", Config: sharedCfg},
		{Prefix: "products/", Config: sharedCfg},
	}

	router := NewPrefixRouter(rules, sharedCfg, []*BucketConfig{sharedCfg})
	configs := router.AllConfigs()

	assert.Len(t, configs, 1)
	assert.Equal(t, "shared-bucket", configs[0].Name)
}

func TestPrefixRouter_Fallback_BackwardCompat(t *testing.T) {
	defaultCfg := &BucketConfig{Name: "my-fallback", Region: "us-east-1"}
	router := NewPrefixRouter(nil, defaultCfg, nil)
	assert.Equal(t, "my-fallback", router.Fallback())
}

func TestPrefixRouter_Fallback_NilDefault(t *testing.T) {
	router := NewPrefixRouter(nil, nil, nil)
	assert.Equal(t, "", router.Fallback())
}

func TestPrefixRouter_ConfigPreservesRegion(t *testing.T) {
	euConfig := &BucketConfig{Name: "eu-bucket", Region: "eu-west-1", Endpoint: "https://s3.eu-west-1.amazonaws.com"}
	apConfig := &BucketConfig{Name: "ap-bucket", Region: "ap-southeast-1"}
	defaultCfg := &BucketConfig{Name: "default", Region: "us-east-1"}

	rules := []PrefixRule{
		{Prefix: "europe/", Config: euConfig},
		{Prefix: "asia/", Config: apConfig},
	}

	router := NewPrefixRouter(rules, defaultCfg, nil)

	cfg := router.ConfigFor("europe/image.jpg")
	assert.Equal(t, "eu-bucket", cfg.Name)
	assert.Equal(t, "eu-west-1", cfg.Region)
	assert.Equal(t, "https://s3.eu-west-1.amazonaws.com", cfg.Endpoint)

	cfg = router.ConfigFor("asia/image.jpg")
	assert.Equal(t, "ap-bucket", cfg.Name)
	assert.Equal(t, "ap-southeast-1", cfg.Region)
}

func TestPrefixRouter_DoesNotMutateInput(t *testing.T) {
	rules := []PrefixRule{
		{Prefix: "b/", Config: &BucketConfig{Name: "bucket-b"}},
		{Prefix: "a/", Config: &BucketConfig{Name: "bucket-a"}},
	}
	originalFirst := rules[0]

	NewPrefixRouter(rules, &BucketConfig{Name: "default"}, nil)

	assert.Equal(t, originalFirst.Prefix, rules[0].Prefix)
}
