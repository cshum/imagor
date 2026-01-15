package awsconfig

import (
	"fmt"
	"os"

	"github.com/cshum/imagor/storage/s3storage"
	"gopkg.in/yaml.v3"
)

type bucketConfigYAML struct {
	Name            string `yaml:"name"`
	Region          string `yaml:"region"`
	Endpoint        string `yaml:"endpoint"`
	AccessKeyID     string `yaml:"access_key_id"`
	SecretAccessKey string `yaml:"secret_access_key"`
	SessionToken    string `yaml:"session_token"`
}

type bucketRouterConfig struct {
	RoutingPattern  string             `yaml:"routing_pattern"`
	DefaultBucket   bucketConfigYAML   `yaml:"default_bucket"`
	FallbackBuckets []bucketConfigYAML `yaml:"fallback_buckets"`
	Rules           []struct {
		Match  string           `yaml:"match"`
		Bucket bucketConfigYAML `yaml:"bucket"`
	} `yaml:"rules"`
}

func LoadBucketRouterFromYAML(path string) (*s3storage.PatternRouter, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg bucketRouterConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	if cfg.RoutingPattern == "" {
		return nil, fmt.Errorf("routing_pattern is required")
	}

	defaultConfig := toBucketConfig(&cfg.DefaultBucket)

	var fallbacks []*s3storage.BucketConfig
	for _, fb := range cfg.FallbackBuckets {
		fallbacks = append(fallbacks, toBucketConfig(&fb))
	}

	rules := make([]s3storage.MatchRule, 0, len(cfg.Rules))
	for _, r := range cfg.Rules {
		rules = append(rules, s3storage.MatchRule{
			Match:  r.Match,
			Config: toBucketConfig(&r.Bucket),
		})
	}

	return s3storage.NewPatternRouter(cfg.RoutingPattern, rules, defaultConfig, fallbacks)
}

func toBucketConfig(y *bucketConfigYAML) *s3storage.BucketConfig {
	if y == nil || y.Name == "" {
		return nil
	}
	return &s3storage.BucketConfig{
		Name:            y.Name,
		Region:          y.Region,
		Endpoint:        y.Endpoint,
		AccessKeyID:     y.AccessKeyID,
		SecretAccessKey: y.SecretAccessKey,
		SessionToken:    y.SessionToken,
	}
}
