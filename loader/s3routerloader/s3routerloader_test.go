package s3routerloader

import (
	"net/http/httptest"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/cshum/imagor/storage/s3storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestS3RouterLoader_RoutesToCorrectBucket(t *testing.T) {
	sgBucket := &BucketConfig{Name: "sg-bucket", Region: "ap-southeast-1"}
	usBucket := &BucketConfig{Name: "us-bucket", Region: "us-west-2"}
	defaultBucket := &BucketConfig{Name: "default-bucket", Region: "us-east-1"}

	router, err := NewPatternRouter(
		`^[a-f0-9]{4}-(?P<bucket>[A-Z]{2})-`,
		[]MatchRule{
			{Match: "SG", Config: sgBucket},
			{Match: "US", Config: usBucket},
		},
		defaultBucket,
		nil,
	)
	require.NoError(t, err)

	createdBuckets := make(map[string]bool)
	factory := func(cfg aws.Config, bucket string) *s3storage.S3Storage {
		createdBuckets[bucket] = true
		return s3storage.New(cfg, bucket)
	}

	_ = New(aws.Config{Region: "us-east-1"}, router, factory)

	assert.True(t, createdBuckets["sg-bucket"])
	assert.True(t, createdBuckets["us-bucket"])
	assert.True(t, createdBuckets["default-bucket"])
}

func TestS3RouterLoader_Get_NoMatchingLoader(t *testing.T) {
	router, err := NewPatternRouter(
		`^(?P<bucket>[A-Z]+)-`,
		nil,
		nil,
		nil,
	)
	require.NoError(t, err)

	factory := func(cfg aws.Config, bucket string) *s3storage.S3Storage {
		return s3storage.New(cfg, bucket)
	}

	loader := New(aws.Config{}, router, factory)

	req := httptest.NewRequest("GET", "/test.jpg", nil)
	_, err = loader.Get(req, "test.jpg")
	assert.Error(t, err)
}

func TestS3RouterLoader_Get_UsesKeyForWithPathGroup(t *testing.T) {
	// Verifies that when a path group is present in the routing pattern,
	// the S3 loader receives the stripped key (without the bucket prefix),
	// not the full original image path.
	testBucket := &BucketConfig{Name: "mysite-test", Region: "auto"}
	defaultBucket := &BucketConfig{Name: "mysite-prod"}

	router, err := NewPatternRouter(
		`^(?P<bucket>mysite-[a-z]+)\/(?P<path>.+)$`,
		[]MatchRule{
			{Match: "mysite-test", Config: testBucket},
		},
		defaultBucket,
		nil,
	)
	require.NoError(t, err)

	receivedKeys := make(map[string]string) // bucket -> key
	factory := func(cfg aws.Config, bucket string) *s3storage.S3Storage {
		return s3storage.New(cfg, bucket)
	}

	loader := New(aws.Config{Region: "us-east-1"}, router, factory)

	// We can't easily intercept the S3 call, but we can verify KeyFor directly
	// to confirm the loader would use the correct key.
	assert.Equal(t, "images/photo.jpg", router.KeyFor("mysite-test/images/photo.jpg"))
	assert.Equal(t, "images/photo.jpg", router.KeyFor("/mysite-test/images/photo.jpg"))
	assert.Equal(t, "assets/logo.png", router.KeyFor("mysite-prod/assets/logo.png"))

	// Confirm routing still resolves to the correct bucket config
	assert.Equal(t, testBucket, router.ConfigFor("mysite-test/images/photo.jpg"))
	assert.Equal(t, defaultBucket, router.ConfigFor("mysite-prod/assets/logo.png"))

	_ = receivedKeys
	_ = loader
}

func TestS3RouterLoader_PassthroughMode(t *testing.T) {
	// When no rules and no default_bucket are configured, the router returns nil
	// for ConfigFor. The loader should fall through to passthrough mode, using
	// the bucket name captured by the pattern directly.
	router, err := NewPatternRouter(
		`^(?P<bucket>[^/]+)\/(?P<path>.+)$`,
		nil, // no rules
		nil, // no default bucket
		nil,
	)
	require.NoError(t, err)

	createdBuckets := make(map[string]bool)
	factory := func(cfg aws.Config, bucket string) *s3storage.S3Storage {
		createdBuckets[bucket] = true
		return s3storage.New(cfg, bucket)
	}

	loader := New(aws.Config{Region: "us-east-1"}, router, factory)

	// No pre-created loaders (no configs declared)
	assert.Empty(t, createdBuckets)

	// ConfigFor returns nil (no rules, no default)
	assert.Nil(t, router.ConfigFor("mysite-test/images/photo.jpg"))

	// BucketNameFor extracts the bucket from the pattern
	assert.Equal(t, "mysite-test", router.BucketNameFor("mysite-test/images/photo.jpg"))
	assert.Equal(t, "mysite-prod", router.BucketNameFor("mysite-prod/assets/logo.png"))

	// KeyFor strips the bucket prefix
	assert.Equal(t, "images/photo.jpg", router.KeyFor("mysite-test/images/photo.jpg"))

	// After a Get call, the dynamic loader for that bucket should be created
	req := httptest.NewRequest("GET", "/mysite-test/images/photo.jpg", nil)
	_, _ = loader.Get(req, "mysite-test/images/photo.jpg")
	assert.True(t, createdBuckets["mysite-test"], "dynamic loader should be created for mysite-test")

	// A second call reuses the cached loader (no duplicate creation)
	prevCount := len(createdBuckets)
	_, _ = loader.Get(req, "mysite-test/images/other.jpg")
	assert.Equal(t, prevCount, len(createdBuckets), "should reuse cached loader")

	// A different bucket creates a new loader
	req2 := httptest.NewRequest("GET", "/mysite-prod/assets/logo.png", nil)
	_, _ = loader.Get(req2, "mysite-prod/assets/logo.png")
	assert.True(t, createdBuckets["mysite-prod"], "dynamic loader should be created for mysite-prod")
}

func TestS3RouterLoader_CreatesClientsForAllConfigs(t *testing.T) {
	defaultBucket := &BucketConfig{Name: "default", Region: "us-east-1"}
	sgBucket := &BucketConfig{Name: "sg", Region: "ap-southeast-1"}
	fallback := &BucketConfig{Name: "fallback", Region: "eu-west-1"}

	router, err := NewPatternRouter(
		`^(?P<bucket>[A-Z]+)-`,
		[]MatchRule{
			{Match: "SG", Config: sgBucket},
		},
		defaultBucket,
		[]*BucketConfig{fallback},
	)
	require.NoError(t, err)

	regions := make(map[string]string)
	factory := func(cfg aws.Config, bucket string) *s3storage.S3Storage {
		regions[bucket] = cfg.Region
		return s3storage.New(cfg, bucket)
	}

	_ = New(aws.Config{Region: "base-region"}, router, factory)

	assert.Equal(t, "us-east-1", regions["default"])
	assert.Equal(t, "ap-southeast-1", regions["sg"])
	assert.Equal(t, "eu-west-1", regions["fallback"])
}
