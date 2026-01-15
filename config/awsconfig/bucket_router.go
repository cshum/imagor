package awsconfig

import (
	"os"
	"strings"

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
	DefaultBucket   bucketConfigYAML   `yaml:"default_bucket"`
	FallbackBuckets []bucketConfigYAML `yaml:"fallback_buckets"`
	Rules           []struct {
		Prefix string           `yaml:"prefix"`
		Bucket bucketConfigYAML `yaml:"bucket"`
	} `yaml:"rules"`
}

func LoadBucketRouterFromYAML(path string) (*s3storage.PrefixRouter, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg bucketRouterConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	defaultConfig := toBucketConfig(&cfg.DefaultBucket)

	var fallbacks []*s3storage.BucketConfig
	for _, fb := range cfg.FallbackBuckets {
		fallbacks = append(fallbacks, toBucketConfig(&fb))
	}

	rules := make([]s3storage.PrefixRule, 0, len(cfg.Rules))
	for _, r := range cfg.Rules {
		rules = append(rules, s3storage.PrefixRule{
			Prefix: strings.TrimLeft(r.Prefix, "/"),
			Config: toBucketConfig(&r.Bucket),
		})
	}

	return s3storage.NewPrefixRouter(rules, defaultConfig, fallbacks), nil
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
