package s3storage

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPatternRouter_ConfigFor(t *testing.T) {
	defaultCfg := &BucketConfig{Name: "default-bucket", Region: "us-east-1"}
	b1Cfg := &BucketConfig{Name: "singapore-bucket", Region: "ap-southeast-1"}
	usCfg := &BucketConfig{Name: "us-bucket", Region: "us-east-1"}
	euCfg := &BucketConfig{Name: "eu-bucket", Region: "eu-west-1"}

	rules := []MatchRule{
		{Match: "B1", Config: b1Cfg},
		{Match: "US", Config: usCfg},
		{Match: "EU", Config: euCfg},
	}

	t.Run("random prefix pattern", func(t *testing.T) {
		router, err := NewPatternRouter(
			`^[a-f0-9]{4}-(?P<bucket>[A-Za-z0-9]+)-`,
			rules,
			defaultCfg,
			nil,
		)
		require.NoError(t, err)

		tests := []struct {
			key            string
			expectedBucket string
		}{
			{"f7a3-B1-project-123-image.jpg", "singapore-bucket"},
			{"9bc2-US-project-456-image.png", "us-bucket"},
			{"a4d1-EU-project-789-image.webp", "eu-bucket"},
			{"abcd-XX-unknown-code.jpg", "default-bucket"},
			{"no-match.jpg", "default-bucket"},
			{"", "default-bucket"},
		}

		for _, tt := range tests {
			t.Run(tt.key, func(t *testing.T) {
				cfg := router.ConfigFor(tt.key)
				assert.Equal(t, tt.expectedBucket, cfg.Name)
			})
		}
	})

	t.Run("simple prefix-like pattern", func(t *testing.T) {
		prefixRules := []MatchRule{
			{Match: "users", Config: &BucketConfig{Name: "users-bucket"}},
			{Match: "products", Config: &BucketConfig{Name: "products-bucket"}},
		}

		router, err := NewPatternRouter(
			`^(?P<bucket>[^/]+)/`,
			prefixRules,
			defaultCfg,
			nil,
		)
		require.NoError(t, err)

		tests := []struct {
			key            string
			expectedBucket string
		}{
			{"users/123/image.jpg", "users-bucket"},
			{"products/456/photo.png", "products-bucket"},
			{"other/789/file.jpg", "default-bucket"},
			{"/users/123/image.jpg", "users-bucket"},
		}

		for _, tt := range tests {
			t.Run(tt.key, func(t *testing.T) {
				cfg := router.ConfigFor(tt.key)
				assert.Equal(t, tt.expectedBucket, cfg.Name)
			})
		}
	})

	t.Run("region-based pattern", func(t *testing.T) {
		regionRules := []MatchRule{
			{Match: "us", Config: &BucketConfig{Name: "us-bucket"}},
			{Match: "eu", Config: &BucketConfig{Name: "eu-bucket"}},
			{Match: "ap", Config: &BucketConfig{Name: "ap-bucket"}},
		}

		router, err := NewPatternRouter(
			`^(?P<bucket>us|eu|ap)/`,
			regionRules,
			defaultCfg,
			nil,
		)
		require.NoError(t, err)

		tests := []struct {
			key            string
			expectedBucket string
		}{
			{"us/image.jpg", "us-bucket"},
			{"eu/photo.png", "eu-bucket"},
			{"ap/file.webp", "ap-bucket"},
			{"other/image.jpg", "default-bucket"},
		}

		for _, tt := range tests {
			t.Run(tt.key, func(t *testing.T) {
				cfg := router.ConfigFor(tt.key)
				assert.Equal(t, tt.expectedBucket, cfg.Name)
			})
		}
	})
}

func TestPatternRouter_InvalidPattern(t *testing.T) {
	t.Run("invalid regex", func(t *testing.T) {
		_, err := NewPatternRouter(
			`^[invalid(`,
			nil,
			nil,
			nil,
		)
		assert.Error(t, err)
	})

	t.Run("missing bucket capture group", func(t *testing.T) {
		_, err := NewPatternRouter(
			`^[a-f0-9]{4}-([A-Za-z0-9]+)-`,
			nil,
			nil,
			nil,
		)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "(?P<bucket>...)")
	})
}

func TestPatternRouter_Fallbacks(t *testing.T) {
	defaultCfg := &BucketConfig{Name: "default-bucket", Region: "us-east-1"}
	fb1 := &BucketConfig{Name: "fallback-1", Region: "us-west-1"}
	fb2 := &BucketConfig{Name: "fallback-2", Region: "eu-west-1"}
	fb3 := &BucketConfig{Name: "fallback-3", Region: "ap-southeast-1"}

	t.Run("returns fallbacks", func(t *testing.T) {
		router, err := NewPatternRouter(
			`^(?P<bucket>[^/]+)/`,
			nil,
			defaultCfg,
			[]*BucketConfig{fb1, fb2},
		)
		require.NoError(t, err)

		fallbacks := router.Fallbacks()
		assert.Len(t, fallbacks, 2)
		assert.Equal(t, "fallback-1", fallbacks[0].Name)
		assert.Equal(t, "fallback-2", fallbacks[1].Name)
	})

	t.Run("limits to 2 fallbacks", func(t *testing.T) {
		router, err := NewPatternRouter(
			`^(?P<bucket>[^/]+)/`,
			nil,
			defaultCfg,
			[]*BucketConfig{fb1, fb2, fb3},
		)
		require.NoError(t, err)

		fallbacks := router.Fallbacks()
		assert.Len(t, fallbacks, 2)
	})

	t.Run("empty fallbacks", func(t *testing.T) {
		router, err := NewPatternRouter(
			`^(?P<bucket>[^/]+)/`,
			nil,
			defaultCfg,
			nil,
		)
		require.NoError(t, err)

		fallbacks := router.Fallbacks()
		assert.Len(t, fallbacks, 0)
	})
}

func TestPatternRouter_AllConfigs(t *testing.T) {
	defaultCfg := &BucketConfig{Name: "default-bucket", Region: "us-east-1"}
	usersCfg := &BucketConfig{Name: "users-bucket", Region: "eu-west-1"}
	fb1 := &BucketConfig{Name: "fallback-1", Region: "us-west-1"}

	rules := []MatchRule{
		{Match: "users", Config: usersCfg},
	}

	router, err := NewPatternRouter(
		`^(?P<bucket>[^/]+)/`,
		rules,
		defaultCfg,
		[]*BucketConfig{fb1},
	)
	require.NoError(t, err)

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

func TestPatternRouter_AllConfigs_NoDuplicates(t *testing.T) {
	sharedCfg := &BucketConfig{Name: "shared-bucket", Region: "us-east-1"}

	rules := []MatchRule{
		{Match: "users", Config: sharedCfg},
		{Match: "products", Config: sharedCfg},
	}

	router, err := NewPatternRouter(
		`^(?P<bucket>[^/]+)/`,
		rules,
		sharedCfg,
		[]*BucketConfig{sharedCfg},
	)
	require.NoError(t, err)

	configs := router.AllConfigs()
	assert.Len(t, configs, 1)
	assert.Equal(t, "shared-bucket", configs[0].Name)
}

func TestPatternRouter_Fallback_BackwardCompat(t *testing.T) {
	defaultCfg := &BucketConfig{Name: "my-fallback", Region: "us-east-1"}

	router, err := NewPatternRouter(
		`^(?P<bucket>[^/]+)/`,
		nil,
		defaultCfg,
		nil,
	)
	require.NoError(t, err)

	assert.Equal(t, "my-fallback", router.Fallback())
}

func TestPatternRouter_Fallback_NilDefault(t *testing.T) {
	router, err := NewPatternRouter(
		`^(?P<bucket>[^/]+)/`,
		nil,
		nil,
		nil,
	)
	require.NoError(t, err)

	assert.Equal(t, "", router.Fallback())
}

func TestPatternRouter_ConfigPreservesRegion(t *testing.T) {
	euConfig := &BucketConfig{Name: "eu-bucket", Region: "eu-west-1", Endpoint: "https://s3.eu-west-1.amazonaws.com"}
	apConfig := &BucketConfig{Name: "ap-bucket", Region: "ap-southeast-1"}
	defaultCfg := &BucketConfig{Name: "default", Region: "us-east-1"}

	rules := []MatchRule{
		{Match: "EU", Config: euConfig},
		{Match: "AP", Config: apConfig},
	}

	router, err := NewPatternRouter(
		`^[a-f0-9]{4}-(?P<bucket>[A-Z]+)-`,
		rules,
		defaultCfg,
		nil,
	)
	require.NoError(t, err)

	cfg := router.ConfigFor("abcd-EU-image.jpg")
	assert.Equal(t, "eu-bucket", cfg.Name)
	assert.Equal(t, "eu-west-1", cfg.Region)
	assert.Equal(t, "https://s3.eu-west-1.amazonaws.com", cfg.Endpoint)

	cfg = router.ConfigFor("1234-AP-image.jpg")
	assert.Equal(t, "ap-bucket", cfg.Name)
	assert.Equal(t, "ap-southeast-1", cfg.Region)
}
