package s3storage

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPrefixRouter_BucketFor(t *testing.T) {
	tests := []struct {
		name           string
		rules          []PrefixRule
		fallback       string
		key            string
		expectedBucket string
	}{
		{
			name:           "empty rules returns fallback",
			rules:          []PrefixRule{},
			fallback:       "default-bucket",
			key:            "users/123/image.jpg",
			expectedBucket: "default-bucket",
		},
		{
			name: "exact prefix match",
			rules: []PrefixRule{
				{Prefix: "users/", Bucket: "users-bucket"},
				{Prefix: "products/", Bucket: "products-bucket"},
			},
			fallback:       "default-bucket",
			key:            "users/123/image.jpg",
			expectedBucket: "users-bucket",
		},
		{
			name: "no match returns fallback",
			rules: []PrefixRule{
				{Prefix: "users/", Bucket: "users-bucket"},
				{Prefix: "products/", Bucket: "products-bucket"},
			},
			fallback:       "default-bucket",
			key:            "other/123/image.jpg",
			expectedBucket: "default-bucket",
		},
		{
			name: "longest prefix wins",
			rules: []PrefixRule{
				{Prefix: "users/", Bucket: "users-bucket"},
				{Prefix: "users/vip/", Bucket: "vip-bucket"},
			},
			fallback:       "default-bucket",
			key:            "users/vip/123/image.jpg",
			expectedBucket: "vip-bucket",
		},
		{
			name: "longest prefix wins reverse order",
			rules: []PrefixRule{
				{Prefix: "users/vip/", Bucket: "vip-bucket"},
				{Prefix: "users/", Bucket: "users-bucket"},
			},
			fallback:       "default-bucket",
			key:            "users/vip/123/image.jpg",
			expectedBucket: "vip-bucket",
		},
		{
			name: "strips leading slash from key",
			rules: []PrefixRule{
				{Prefix: "users/", Bucket: "users-bucket"},
			},
			fallback:       "default-bucket",
			key:            "/users/123/image.jpg",
			expectedBucket: "users-bucket",
		},
		{
			name: "multiple leading slashes stripped",
			rules: []PrefixRule{
				{Prefix: "users/", Bucket: "users-bucket"},
			},
			fallback:       "default-bucket",
			key:            "///users/123/image.jpg",
			expectedBucket: "users-bucket",
		},
		{
			name: "deep nested prefix",
			rules: []PrefixRule{
				{Prefix: "media/images/thumbnails/", Bucket: "thumbnails-bucket"},
				{Prefix: "media/images/", Bucket: "images-bucket"},
				{Prefix: "media/", Bucket: "media-bucket"},
			},
			fallback:       "default-bucket",
			key:            "media/images/thumbnails/123.jpg",
			expectedBucket: "thumbnails-bucket",
		},
		{
			name: "empty key returns fallback",
			rules: []PrefixRule{
				{Prefix: "users/", Bucket: "users-bucket"},
			},
			fallback:       "default-bucket",
			key:            "",
			expectedBucket: "default-bucket",
		},
		{
			name: "key equals prefix",
			rules: []PrefixRule{
				{Prefix: "users/", Bucket: "users-bucket"},
			},
			fallback:       "default-bucket",
			key:            "users/",
			expectedBucket: "users-bucket",
		},
		{
			name: "prefix without trailing slash",
			rules: []PrefixRule{
				{Prefix: "users", Bucket: "users-bucket"},
			},
			fallback:       "default-bucket",
			key:            "users/123/image.jpg",
			expectedBucket: "users-bucket",
		},
		{
			name: "prefix without trailing slash matches similar names",
			rules: []PrefixRule{
				{Prefix: "user", Bucket: "user-bucket"},
			},
			fallback:       "default-bucket",
			key:            "users/123/image.jpg",
			expectedBucket: "user-bucket",
		},
		{
			name: "mixed trailing slash prefixes longest wins",
			rules: []PrefixRule{
				{Prefix: "img", Bucket: "img-bucket"},
				{Prefix: "images/", Bucket: "images-bucket"},
			},
			fallback:       "default-bucket",
			key:            "images/photo.jpg",
			expectedBucket: "images-bucket",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := NewPrefixRouter(tt.rules, tt.fallback)
			bucket := router.BucketFor(tt.key)
			assert.Equal(t, tt.expectedBucket, bucket)
		})
	}
}

func TestPrefixRouter_Fallback(t *testing.T) {
	router := NewPrefixRouter([]PrefixRule{}, "my-fallback")
	assert.Equal(t, "my-fallback", router.Fallback())
}

func TestPrefixRouter_RulesSortedByLength(t *testing.T) {
	rules := []PrefixRule{
		{Prefix: "a/", Bucket: "bucket-a"},
		{Prefix: "aaa/", Bucket: "bucket-aaa"},
		{Prefix: "aa/", Bucket: "bucket-aa"},
	}
	router := NewPrefixRouter(rules, "default")

	assert.Equal(t, "bucket-a", router.BucketFor("a/file.jpg"))
	assert.Equal(t, "bucket-aa", router.BucketFor("aa/file.jpg"))
	assert.Equal(t, "bucket-aaa", router.BucketFor("aaa/file.jpg"))
}

func TestPrefixRouter_DoesNotMutateInput(t *testing.T) {
	rules := []PrefixRule{
		{Prefix: "b/", Bucket: "bucket-b"},
		{Prefix: "a/", Bucket: "bucket-a"},
	}
	originalFirst := rules[0]

	NewPrefixRouter(rules, "default")

	assert.Equal(t, originalFirst, rules[0])
}
