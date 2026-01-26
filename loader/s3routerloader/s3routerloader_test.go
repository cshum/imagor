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
