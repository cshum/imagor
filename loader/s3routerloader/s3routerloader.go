package s3routerloader

import (
	"net/http"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/cshum/imagor"
	"github.com/cshum/imagor/storage/s3storage"
)

type S3RouterLoader struct {
	router         BucketRouter
	baseCfg        aws.Config
	storageFactory S3StorageFactory
	loaders        map[string]*s3storage.S3Storage
	fallbacks      []*s3storage.S3Storage
	dynamicLoaders sync.Map // bucket name → *s3storage.S3Storage, for passthrough mode
}

type S3StorageFactory func(cfg aws.Config, bucket string) *s3storage.S3Storage

func New(
	baseCfg aws.Config,
	router BucketRouter,
	storageFactory S3StorageFactory,
) *S3RouterLoader {
	l := &S3RouterLoader{
		router:         router,
		baseCfg:        baseCfg,
		storageFactory: storageFactory,
		loaders:        make(map[string]*s3storage.S3Storage),
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

// getOrCreateDynamicLoader returns a cached or newly created S3Storage for the
// given bucket name, using the base AWS config. This supports passthrough mode
// where no rules or default_bucket are configured in the YAML — the bucket name
// is extracted directly from the routing pattern's (?P<bucket>...) capture group.
func (l *S3RouterLoader) getOrCreateDynamicLoader(bucket string) *s3storage.S3Storage {
	if v, ok := l.dynamicLoaders.Load(bucket); ok {
		return v.(*s3storage.S3Storage)
	}
	loader := l.storageFactory(l.baseCfg, bucket)
	actual, _ := l.dynamicLoaders.LoadOrStore(bucket, loader)
	return actual.(*s3storage.S3Storage)
}

func (l *S3RouterLoader) Get(r *http.Request, image string) (*imagor.Blob, error) {
	cfg := l.router.ConfigFor(image)
	key := l.router.KeyFor(image)

	var loader *s3storage.S3Storage

	if cfg != nil {
		var ok bool
		loader, ok = l.loaders[cfg.Name]
		if !ok {
			return nil, imagor.ErrNotFound
		}
	} else {
		// Passthrough mode: no matching rule and no default_bucket configured.
		// Use the bucket name captured by the routing pattern directly.
		// KeyFor returns the stripped path; we need the bucket from ConfigFor's
		// pattern match. Extract it via the key difference or use a dedicated method.
		// Since cfg is nil, the pattern didn't match OR there's no default.
		// We use the bucket extracted from the image path via the pattern.
		bucketName := l.router.BucketNameFor(image)
		if bucketName == "" {
			return nil, imagor.ErrNotFound
		}
		loader = l.getOrCreateDynamicLoader(bucketName)
	}

	blob, err := loader.Get(r, key)
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
		blob, err = fb.Get(r, key)
		if err == nil {
			return blob, nil
		}
		if err != imagor.ErrNotFound {
			return nil, err
		}
	}

	return nil, imagor.ErrNotFound
}
