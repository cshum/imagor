package s3storage

import (
	"sort"
	"strings"
)

// BucketRouter determines which bucket to use based on the image key
type BucketRouter interface {
	BucketFor(key string) string
}

// PrefixRule maps a path prefix to a bucket
type PrefixRule struct {
	Prefix string
	Bucket string
}

// PrefixRouter routes requests to buckets based on longest-prefix-first matching
type PrefixRouter struct {
	rules    []PrefixRule
	fallback string
}

// NewPrefixRouter creates a PrefixRouter, sorting rules by prefix length descending
func NewPrefixRouter(rules []PrefixRule, fallback string) *PrefixRouter {
	sorted := make([]PrefixRule, len(rules))
	copy(sorted, rules)

	sort.Slice(sorted, func(i, j int) bool {
		return len(sorted[i].Prefix) > len(sorted[j].Prefix)
	})

	return &PrefixRouter{
		rules:    sorted,
		fallback: fallback,
	}
}

// BucketFor returns the bucket for the given key, or fallback if no prefix matches
func (r *PrefixRouter) BucketFor(key string) string {
	key = strings.TrimLeft(key, "/")

	for _, rule := range r.rules {
		if strings.HasPrefix(key, rule.Prefix) {
			return rule.Bucket
		}
	}
	return r.fallback
}

// Fallback returns the fallback bucket name
func (r *PrefixRouter) Fallback() string {
	return r.fallback
}
