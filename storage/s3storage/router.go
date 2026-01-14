package s3storage

import (
	"sort"
	"strings"
)

// BucketConfig contains S3 bucket configuration including region and credentials
type BucketConfig struct {
	Name            string
	Region          string
	Endpoint        string
	AccessKeyID     string
	SecretAccessKey string
	SessionToken    string
}

// BucketRouter determines which bucket configuration to use based on the image key
type BucketRouter interface {
	// ConfigFor returns the bucket config for the given key
	ConfigFor(key string) *BucketConfig
	// Fallbacks returns the list of fallback bucket configs to try if primary fails
	Fallbacks() []*BucketConfig
	// DefaultConfig returns the default bucket config
	DefaultConfig() *BucketConfig
	// AllConfigs returns all unique bucket configs for client initialization
	AllConfigs() []*BucketConfig
}

// PrefixRule maps a path prefix to a bucket config
type PrefixRule struct {
	Prefix string
	Config *BucketConfig
}

// PrefixRouter routes requests to buckets based on longest-prefix-first matching
type PrefixRouter struct {
	rules         []PrefixRule
	defaultConfig *BucketConfig
	fallbacks     []*BucketConfig
}

// NewPrefixRouter creates a PrefixRouter, sorting rules by prefix length descending
func NewPrefixRouter(rules []PrefixRule, defaultConfig *BucketConfig, fallbacks []*BucketConfig) *PrefixRouter {
	sorted := make([]PrefixRule, len(rules))
	copy(sorted, rules)

	sort.Slice(sorted, func(i, j int) bool {
		return len(sorted[i].Prefix) > len(sorted[j].Prefix)
	})

	// Limit fallbacks to 2
	if len(fallbacks) > 2 {
		fallbacks = fallbacks[:2]
	}

	return &PrefixRouter{
		rules:         sorted,
		defaultConfig: defaultConfig,
		fallbacks:     fallbacks,
	}
}

// ConfigFor returns the bucket config for the given key, or default if no prefix matches
func (r *PrefixRouter) ConfigFor(key string) *BucketConfig {
	key = strings.TrimLeft(key, "/")

	for _, rule := range r.rules {
		if strings.HasPrefix(key, rule.Prefix) {
			return rule.Config
		}
	}
	return r.defaultConfig
}

// Fallbacks returns the list of fallback bucket configs
func (r *PrefixRouter) Fallbacks() []*BucketConfig {
	return r.fallbacks
}

// DefaultConfig returns the default bucket config
func (r *PrefixRouter) DefaultConfig() *BucketConfig {
	return r.defaultConfig
}

// AllConfigs returns all unique bucket configs for client initialization
func (r *PrefixRouter) AllConfigs() []*BucketConfig {
	seen := make(map[string]bool)
	var configs []*BucketConfig

	addConfig := func(cfg *BucketConfig) {
		if cfg != nil && !seen[cfg.Name] {
			seen[cfg.Name] = true
			configs = append(configs, cfg)
		}
	}

	addConfig(r.defaultConfig)
	for _, rule := range r.rules {
		addConfig(rule.Config)
	}
	for _, fb := range r.fallbacks {
		addConfig(fb)
	}

	return configs
}

// Fallback returns the fallback bucket name (for backward compatibility)
func (r *PrefixRouter) Fallback() string {
	if r.defaultConfig != nil {
		return r.defaultConfig.Name
	}
	return ""
}
