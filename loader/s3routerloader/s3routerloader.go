package s3routerloader

import (
	"net/http"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/cshum/imagor"
	"github.com/cshum/imagor/storage/s3storage"
)

type S3RouterLoader struct {
	router    BucketRouter
	loaders   map[string]*s3storage.S3Storage
	fallbacks []*s3storage.S3Storage
}

type S3StorageFactory func(cfg aws.Config, bucket string) *s3storage.S3Storage

func New(
	baseCfg aws.Config,
	router BucketRouter,
	storageFactory S3StorageFactory,
) *S3RouterLoader {
	l := &S3RouterLoader{
		router:  router,
		loaders: make(map[string]*s3storage.S3Storage),
	}

	for _, bucketCfg := range router.AllConfigs() {
		awsCfg := createAWSConfig(baseCfg, bucketCfg)
		l.loaders[bucketCfg.Name] = storageFactory(awsCfg, bucketCfg.Name)
	}

	for _, fb := range router.Fallbacks() {
		if loader, ok := l.loaders[fb.Name]; ok {
			l.fallbacks = append(l.fallbacks, loader)
		}
	}

	return l
}

func createAWSConfig(baseCfg aws.Config, bucketCfg *BucketConfig) aws.Config {
	cfg := baseCfg.Copy()

	if bucketCfg.Region != "" {
		cfg.Region = bucketCfg.Region
	}

	return cfg
}

func (l *S3RouterLoader) Get(r *http.Request, image string) (*imagor.Blob, error) {
	cfg := l.router.ConfigFor(image)
	if cfg == nil {
		return nil, imagor.ErrNotFound
	}

	loader, ok := l.loaders[cfg.Name]
	if !ok {
		return nil, imagor.ErrNotFound
	}

	blob, err := loader.Get(r, image)
	if err == nil {
		return blob, nil
	}

	if err != imagor.ErrNotFound {
		return nil, err
	}

	for _, fb := range l.fallbacks {
		if fb == loader {
			continue
		}
		blob, err = fb.Get(r, image)
		if err == nil {
			return blob, nil
		}
		if err != imagor.ErrNotFound {
			return nil, err
		}
	}

	return nil, imagor.ErrNotFound
}
