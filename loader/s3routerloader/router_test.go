package s3routerloader

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPatternRouter_ConfigFor(t *testing.T) {
	defaultBucket := &BucketConfig{Name: "default-bucket", Region: "us-east-1"}
	sgBucket := &BucketConfig{Name: "sg-bucket", Region: "ap-southeast-1"}
	usBucket := &BucketConfig{Name: "us-bucket", Region: "us-west-2"}

	rules := []MatchRule{
		{Match: "SG", Config: sgBucket},
		{Match: "US", Config: usBucket},
	}

	router, err := NewPatternRouter(
		`^[a-f0-9]{4}-(?P<bucket>[A-Z]{2})-`,
		rules,
		defaultBucket,
		nil,
	)
	require.NoError(t, err)

	tests := []struct {
		name     string
		key      string
		expected *BucketConfig
	}{
		{
			name:     "routes to SG bucket",
			key:      "abc1-SG-image.jpg",
			expected: sgBucket,
		},
		{
			name:     "routes to US bucket",
			key:      "def2-US-photo.png",
			expected: usBucket,
		},
		{
			name:     "falls back to default for unknown region",
			key:      "1234-JP-test.jpg",
			expected: defaultBucket,
		},
		{
			name:     "falls back to default for non-matching pattern",
			key:      "regular-image.jpg",
			expected: defaultBucket,
		},
		{
			name:     "handles leading slash",
			key:      "/abc1-SG-image.jpg",
			expected: sgBucket,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := router.ConfigFor(tt.key)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestPatternRouter_RequiresBucketGroup(t *testing.T) {
	_, err := NewPatternRouter(
		`^[a-f0-9]{4}-([A-Z]{2})-`,
		nil,
		&BucketConfig{Name: "default"},
		nil,
	)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "bucket")
}

func TestPatternRouter_InvalidRegex(t *testing.T) {
	_, err := NewPatternRouter(
		`^[invalid(`,
		nil,
		&BucketConfig{Name: "default"},
		nil,
	)
	assert.Error(t, err)
}

func TestPatternRouter_AllConfigs(t *testing.T) {
	defaultBucket := &BucketConfig{Name: "default-bucket"}
	sgBucket := &BucketConfig{Name: "sg-bucket"}
	fallback := &BucketConfig{Name: "fallback-bucket"}

	rules := []MatchRule{
		{Match: "SG", Config: sgBucket},
	}

	router, err := NewPatternRouter(
		`^(?P<bucket>[A-Z]{2})-`,
		rules,
		defaultBucket,
		[]*BucketConfig{fallback},
	)
	require.NoError(t, err)

	configs := router.AllConfigs()
	assert.Len(t, configs, 3)

	names := make(map[string]bool)
	for _, cfg := range configs {
		names[cfg.Name] = true
	}
	assert.True(t, names["default-bucket"])
	assert.True(t, names["sg-bucket"])
	assert.True(t, names["fallback-bucket"])
}

func TestPatternRouter_Fallbacks(t *testing.T) {
	fb1 := &BucketConfig{Name: "fb1"}
	fb2 := &BucketConfig{Name: "fb2"}
	fb3 := &BucketConfig{Name: "fb3"}

	router, err := NewPatternRouter(
		`^(?P<bucket>[A-Z]+)-`,
		nil,
		&BucketConfig{Name: "default"},
		[]*BucketConfig{fb1, fb2, fb3},
	)
	require.NoError(t, err)

	fallbacks := router.Fallbacks()
	assert.Len(t, fallbacks, 2)
	assert.Equal(t, "fb1", fallbacks[0].Name)
	assert.Equal(t, "fb2", fallbacks[1].Name)
}

func TestPatternRouter_Fallback(t *testing.T) {
	router, err := NewPatternRouter(
		`^(?P<bucket>[A-Z]+)-`,
		nil,
		&BucketConfig{Name: "my-default"},
		nil,
	)
	require.NoError(t, err)

	assert.Equal(t, "my-default", router.Fallback())
}

func TestPatternRouter_PrefixPattern(t *testing.T) {
	bucket1 := &BucketConfig{Name: "bucket1"}
	bucket2 := &BucketConfig{Name: "bucket2"}

	router, err := NewPatternRouter(
		`^(?P<bucket>[^/]+)/`,
		[]MatchRule{
			{Match: "bucket1", Config: bucket1},
			{Match: "bucket2", Config: bucket2},
		},
		&BucketConfig{Name: "default"},
		nil,
	)
	require.NoError(t, err)

	assert.Equal(t, bucket1, router.ConfigFor("bucket1/image.jpg"))
	assert.Equal(t, bucket2, router.ConfigFor("bucket2/photo.png"))
}

func TestPatternRouter_KeyFor_WithPathGroup(t *testing.T) {
	router, err := NewPatternRouter(
		`^(?P<bucket>[^/]+)/(?P<path>.+)$`,
		[]MatchRule{
			{Match: "mysite-test", Config: &BucketConfig{Name: "mysite-test"}},
		},
		&BucketConfig{Name: "mysite-prod"},
		nil,
	)
	require.NoError(t, err)

	tests := []struct {
		name     string
		key      string
		expected string
	}{
		{
			name:     "strips bucket prefix from path",
			key:      "mysite-test/images/photo.jpg",
			expected: "images/photo.jpg",
		},
		{
			name:     "strips bucket prefix with leading slash",
			key:      "/mysite-test/images/photo.jpg",
			expected: "images/photo.jpg",
		},
		{
			name:     "strips prefix for default bucket",
			key:      "mysite-prod/assets/logo.png",
			expected: "assets/logo.png",
		},
		{
			name:     "returns original key when pattern does not match",
			key:      "no-slash-here",
			expected: "no-slash-here",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := router.KeyFor(tt.key)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestPatternRouter_KeyFor_WithoutPathGroup(t *testing.T) {
	router, err := NewPatternRouter(
		`^(?P<bucket>[^/]+)/`,
		[]MatchRule{
			{Match: "bucket1", Config: &BucketConfig{Name: "bucket1"}},
		},
		&BucketConfig{Name: "default"},
		nil,
	)
	require.NoError(t, err)

	// Without a path group, KeyFor returns the original key unchanged
	assert.Equal(t, "bucket1/image.jpg", router.KeyFor("bucket1/image.jpg"))
	assert.Equal(t, "/bucket1/image.jpg", router.KeyFor("/bucket1/image.jpg"))
	assert.Equal(t, "regular-image.jpg", router.KeyFor("regular-image.jpg"))
}

func TestPatternRouter_KeyFor_IssueExample(t *testing.T) {
	// Reproduces the exact scenario from GitHub issue #763:
	// routing_pattern: '^(?P<bucket>mysite-[a-z]+)\/(?P<path>.+)$'
	// Request: /mysite-test/images/photo.jpg
	// Expected S3 key: images/photo.jpg (not mysite-test/images/photo.jpg)
	testBucket := &BucketConfig{Name: "mysite-test", Region: "auto"}
	devBucket := &BucketConfig{Name: "mysite-dev", Region: "auto"}
	defaultBucket := &BucketConfig{Name: "mysite-prod"}

	router, err := NewPatternRouter(
		`^(?P<bucket>mysite-[a-z]+)\/(?P<path>.+)$`,
		[]MatchRule{
			{Match: "mysite-test", Config: testBucket},
			{Match: "mysite-dev", Config: devBucket},
		},
		defaultBucket,
		nil,
	)
	require.NoError(t, err)

	// Routing still works correctly
	assert.Equal(t, testBucket, router.ConfigFor("/mysite-test/images/photo.jpg"))
	assert.Equal(t, devBucket, router.ConfigFor("/mysite-dev/assets/logo.png"))
	assert.Equal(t, defaultBucket, router.ConfigFor("/mysite-prod/data/file.txt"))

	// Key is correctly stripped of the bucket prefix
	assert.Equal(t, "images/photo.jpg", router.KeyFor("/mysite-test/images/photo.jpg"))
	assert.Equal(t, "assets/logo.png", router.KeyFor("/mysite-dev/assets/logo.png"))
	assert.Equal(t, "data/file.txt", router.KeyFor("/mysite-prod/data/file.txt"))
}
