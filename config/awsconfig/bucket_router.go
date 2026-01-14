package awsconfig

import (
	"os"
	"strings"

	"github.com/cshum/imagor/storage/s3storage"
	"gopkg.in/yaml.v3"
)

type bucketRouterConfig struct {
	DefaultBucket string `yaml:"default_bucket"`
	Rules         []struct {
		Prefix string `yaml:"prefix"`
		Bucket string `yaml:"bucket"`
	} `yaml:"rules"`
}

// LoadBucketRouterFromYAML loads bucket routing configuration from a YAML file
func LoadBucketRouterFromYAML(path string) (*s3storage.PrefixRouter, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg bucketRouterConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	rules := make([]s3storage.PrefixRule, 0, len(cfg.Rules))
	for _, r := range cfg.Rules {
		rules = append(rules, s3storage.PrefixRule{
			Prefix: strings.TrimLeft(r.Prefix, "/"),
			Bucket: r.Bucket,
		})
	}

	return s3storage.NewPrefixRouter(rules, cfg.DefaultBucket), nil
}
