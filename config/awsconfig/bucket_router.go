package awsconfig

import (
	"fmt"
	"os"

	"github.com/cshum/imagor/loader/s3routerloader"
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

func LoadBucketRouterFromYAML(path string) (*s3routerloader.PatternRouter, error) {
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

	var fallbacks []*s3routerloader.BucketConfig
	for _, fb := range cfg.FallbackBuckets {
		fallbacks = append(fallbacks, toBucketConfig(&fb))
	}

	rules := make([]s3routerloader.MatchRule, 0, len(cfg.Rules))
	for _, r := range cfg.Rules {
		rules = append(rules, s3routerloader.MatchRule{
			Match:  r.Match,
			Config: toBucketConfig(&r.Bucket),
		})
	}

	return s3routerloader.NewPatternRouter(cfg.RoutingPattern, rules, defaultConfig, fallbacks)
}

func toBucketConfig(y *bucketConfigYAML) *s3routerloader.BucketConfig {
	if y == nil || y.Name == "" {
		return nil
	}
	return &s3routerloader.BucketConfig{
		Name:            y.Name,
		Region:          y.Region,
		Endpoint:        y.Endpoint,
		AccessKeyID:     y.AccessKeyID,
		SecretAccessKey: y.SecretAccessKey,
		SessionToken:    y.SessionToken,
	}
}
