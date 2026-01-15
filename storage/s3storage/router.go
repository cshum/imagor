package s3storage

import (
	"fmt"
	"regexp"
	"strings"
)

type BucketConfig struct {
	Name            string
	Region          string
	Endpoint        string
	AccessKeyID     string
	SecretAccessKey string
	SessionToken    string
}

type BucketRouter interface {
	ConfigFor(key string) *BucketConfig
	Fallbacks() []*BucketConfig
	DefaultConfig() *BucketConfig
	AllConfigs() []*BucketConfig
}

type MatchRule struct {
	Match  string
	Config *BucketConfig
}

type PatternRouter struct {
	pattern       *regexp.Regexp
	bucketGroup   int
	rules         map[string]*BucketConfig
	defaultConfig *BucketConfig
	fallbacks     []*BucketConfig
}

func NewPatternRouter(pattern string, rules []MatchRule, defaultConfig *BucketConfig, fallbacks []*BucketConfig) (*PatternRouter, error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, err
	}

	bucketGroup := -1
	for i, name := range re.SubexpNames() {
		if name == "bucket" {
			bucketGroup = i
			break
		}
	}
	if bucketGroup == -1 {
		return nil, fmt.Errorf("pattern must contain named capture group (?P<bucket>...)")
	}

	if len(fallbacks) > 2 {
		fallbacks = fallbacks[:2]
	}

	rulesMap := make(map[string]*BucketConfig, len(rules))
	for _, r := range rules {
		rulesMap[r.Match] = r.Config
	}

	return &PatternRouter{
		pattern:       re,
		bucketGroup:   bucketGroup,
		rules:         rulesMap,
		defaultConfig: defaultConfig,
		fallbacks:     fallbacks,
	}, nil
}

func (r *PatternRouter) ConfigFor(key string) *BucketConfig {
	key = strings.TrimPrefix(key, "/")

	matches := r.pattern.FindStringSubmatch(key)
	if matches == nil || len(matches) <= r.bucketGroup {
		return r.defaultConfig
	}

	bucketID := matches[r.bucketGroup]
	if cfg, ok := r.rules[bucketID]; ok {
		return cfg
	}

	return r.defaultConfig
}

func (r *PatternRouter) Fallbacks() []*BucketConfig {
	return r.fallbacks
}

func (r *PatternRouter) DefaultConfig() *BucketConfig {
	return r.defaultConfig
}

func (r *PatternRouter) AllConfigs() []*BucketConfig {
	seen := make(map[string]bool)
	var configs []*BucketConfig

	addConfig := func(cfg *BucketConfig) {
		if cfg != nil && !seen[cfg.Name] {
			seen[cfg.Name] = true
			configs = append(configs, cfg)
		}
	}

	addConfig(r.defaultConfig)
	for _, cfg := range r.rules {
		addConfig(cfg)
	}
	for _, fb := range r.fallbacks {
		addConfig(fb)
	}

	return configs
}

func (r *PatternRouter) Fallback() string {
	if r.defaultConfig != nil {
		return r.defaultConfig.Name
	}
	return ""
}
