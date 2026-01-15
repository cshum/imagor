package s3storage

import (
	"context"
	"errors"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
	"github.com/cshum/imagor"
	"github.com/cshum/imagor/imagorpath"
)

type S3Storage struct {
	Client       *s3.Client
	Bucket       string
	BucketRouter BucketRouter

	BaseDir        string
	PathPrefix     string
	ACL            string
	SafeChars      string
	StorageClass   string
	Expiration     time.Duration
	Endpoint       string
	ForcePathStyle bool

	safeChars imagorpath.SafeChars

	baseConfig aws.Config
	clients    map[string]*s3.Client
	clientsMu  sync.RWMutex
}

func New(cfg aws.Config, bucket string, options ...Option) *S3Storage {
	baseDir := "/"
	if idx := strings.Index(bucket, "/"); idx > -1 {
		baseDir = bucket[idx:]
		bucket = bucket[:idx]
	}
	s := &S3Storage{
		Bucket:     bucket,
		baseConfig: cfg,
		clients:    make(map[string]*s3.Client),

		BaseDir:    baseDir,
		PathPrefix: "/",
		ACL:        string(types.ObjectCannedACLPublicRead),
	}
	for _, option := range options {
		option(s)
	}

	var s3Options []func(*s3.Options)
	if s.Endpoint != "" {
		s3Options = append(s3Options, func(o *s3.Options) {
			o.BaseEndpoint = aws.String(s.Endpoint)
			o.DisableLogOutputChecksumValidationSkipped = true
		})
	}
	if s.ForcePathStyle {
		s3Options = append(s3Options, func(o *s3.Options) {
			o.UsePathStyle = true
		})
	}
	s.Client = s3.NewFromConfig(cfg, s3Options...)

	if s.SafeChars == "--" {
		s.safeChars = imagorpath.NewNoopSafeChars()
	} else {
		s.safeChars = imagorpath.NewSafeChars("!\"()*" + s.SafeChars)
	}

	if s.BucketRouter != nil {
		s.initRouterClients()
	}

	return s
}

func (s *S3Storage) initRouterClients() {
	for _, cfg := range s.BucketRouter.AllConfigs() {
		s.getOrCreateClient(cfg)
	}
}

func (s *S3Storage) getOrCreateClient(cfg *BucketConfig) *s3.Client {
	if cfg == nil {
		return s.Client
	}

	key := s.clientKey(cfg)

	s.clientsMu.RLock()
	if client, ok := s.clients[key]; ok {
		s.clientsMu.RUnlock()
		return client
	}
	s.clientsMu.RUnlock()

	s.clientsMu.Lock()
	defer s.clientsMu.Unlock()

	if client, ok := s.clients[key]; ok {
		return client
	}

	client := s.createClient(cfg)
	s.clients[key] = client
	return client
}

func (s *S3Storage) clientKey(cfg *BucketConfig) string {
	return cfg.Region + "|" + cfg.Endpoint + "|" + cfg.AccessKeyID
}

func (s *S3Storage) createClient(cfg *BucketConfig) *s3.Client {
	awsCfg := s.baseConfig

	if cfg.Region != "" {
		awsCfg.Region = cfg.Region
	}

	if cfg.AccessKeyID != "" && cfg.SecretAccessKey != "" {
		awsCfg.Credentials = credentials.NewStaticCredentialsProvider(
			cfg.AccessKeyID, cfg.SecretAccessKey, cfg.SessionToken)
	} else if cfg.Region != "" && cfg.Region != s.baseConfig.Region {
		ctx := context.Background()
		newCfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(cfg.Region))
		if err == nil {
			awsCfg = newCfg
		}
	}

	var s3Options []func(*s3.Options)

	endpoint := cfg.Endpoint
	if endpoint == "" {
		endpoint = s.Endpoint
	}
	if endpoint != "" {
		s3Options = append(s3Options, func(o *s3.Options) {
			o.BaseEndpoint = aws.String(endpoint)
			o.DisableLogOutputChecksumValidationSkipped = true
		})
	}

	if s.ForcePathStyle {
		s3Options = append(s3Options, func(o *s3.Options) {
			o.UsePathStyle = true
		})
	}

	return s3.NewFromConfig(awsCfg, s3Options...)
}

func (s *S3Storage) Path(image string) (string, bool) {
	image = "/" + imagorpath.Normalize(image, s.safeChars)
	if !strings.HasPrefix(image, s.PathPrefix) {
		return "", false
	}
	result := filepath.Join(s.BaseDir, strings.TrimPrefix(image, s.PathPrefix))
	if len(result) > 0 && result[0] == '/' {
		result = result[1:]
	}
	return result, true
}

func (s *S3Storage) Get(r *http.Request, image string) (*imagor.Blob, error) {
	ctx := r.Context()
	image, ok := s.Path(image)
	if !ok {
		return nil, imagor.ErrInvalid
	}

	if s.BucketRouter == nil {
		return s.getFromBucket(ctx, s.Client, s.Bucket, image)
	}

	cfg := s.BucketRouter.ConfigFor(image)
	client := s.getOrCreateClient(cfg)
	bucket := cfg.Name

	fallbacks := s.BucketRouter.Fallbacks()
	if len(fallbacks) == 0 {
		return s.getFromBucket(ctx, client, bucket, image)
	}

	blob, err := s.getFromBucketEager(ctx, client, bucket, image)
	if err == nil {
		return blob, nil
	}
	if err != imagor.ErrNotFound {
		return nil, err
	}

	for _, fallbackCfg := range fallbacks {
		fallbackClient := s.getOrCreateClient(fallbackCfg)
		blob, err = s.getFromBucketEager(ctx, fallbackClient, fallbackCfg.Name, image)
		if err == nil {
			return blob, nil
		}
		if err != imagor.ErrNotFound {
			return nil, err
		}
	}

	return nil, imagor.ErrNotFound
}

func (s *S3Storage) getFromBucketEager(ctx context.Context, client *s3.Client, bucket, image string) (*imagor.Blob, error) {
	input := &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(image),
	}
	out, err := client.GetObject(ctx, input)
	if err != nil {
		if isNotFoundError(err) {
			return nil, imagor.ErrNotFound
		}
		return nil, err
	}

	if s.Expiration > 0 && out.LastModified != nil {
		if time.Now().Sub(*out.LastModified) > s.Expiration {
			_ = out.Body.Close()
			return nil, imagor.ErrExpired
		}
	}

	var size int64
	if out.ContentLength != nil {
		size = *out.ContentLength
	}

	blob := imagor.NewBlob(func() (io.ReadCloser, int64, error) {
		return out.Body, size, nil
	})

	if out.ContentType != nil {
		blob.SetContentType(*out.ContentType)
	}
	if out.ContentLength != nil && out.ETag != nil && out.LastModified != nil {
		blob.Stat = &imagor.Stat{
			Size:         *out.ContentLength,
			ETag:         *out.ETag,
			ModifiedTime: *out.LastModified,
		}
	}

	return blob, nil
}

func (s *S3Storage) getFromBucket(ctx context.Context, client *s3.Client, bucket, image string) (*imagor.Blob, error) {
	var blob *imagor.Blob
	var once sync.Once
	blob = imagor.NewBlob(func() (io.ReadCloser, int64, error) {
		input := &s3.GetObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(image),
		}
		out, err := client.GetObject(ctx, input)
		if err != nil {
			if isNotFoundError(err) {
				return nil, 0, imagor.ErrNotFound
			}
			return nil, 0, err
		}
		once.Do(func() {
			if out.ContentType != nil {
				blob.SetContentType(*out.ContentType)
			}
			if out.ContentLength != nil && out.ETag != nil && out.LastModified != nil {
				blob.Stat = &imagor.Stat{
					Size:         *out.ContentLength,
					ETag:         *out.ETag,
					ModifiedTime: *out.LastModified,
				}
			}
		})
		if s.Expiration > 0 && out.LastModified != nil {
			if time.Now().Sub(*out.LastModified) > s.Expiration {
				return nil, 0, imagor.ErrExpired
			}
		}
		var size int64
		if out.ContentLength != nil {
			size = *out.ContentLength
		}
		return out.Body, size, nil
	})
	return blob, nil
}

func (s *S3Storage) Put(ctx context.Context, image string, blob *imagor.Blob) error {
	image, ok := s.Path(image)
	if !ok {
		return imagor.ErrInvalid
	}
	reader, size, err := blob.NewReader()
	if err != nil {
		return err
	}
	defer func() {
		_ = reader.Close()
	}()

	input := &s3.PutObjectInput{
		ACL:           types.ObjectCannedACL(s.ACL),
		Body:          reader,
		Bucket:        aws.String(s.Bucket),
		ContentType:   aws.String(blob.ContentType()),
		ContentLength: aws.Int64(size),
		Key:           aws.String(image),
		StorageClass:  types.StorageClass(s.StorageClass),
	}
	_, err = s.Client.PutObject(ctx, input)
	return err
}

func (s *S3Storage) Delete(ctx context.Context, image string) error {
	image, ok := s.Path(image)
	if !ok {
		return imagor.ErrInvalid
	}
	_, err := s.Client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.Bucket),
		Key:    aws.String(image),
	})
	return err
}

func (s *S3Storage) Stat(ctx context.Context, image string) (stat *imagor.Stat, err error) {
	image, ok := s.Path(image)
	if !ok {
		return nil, imagor.ErrInvalid
	}
	input := &s3.HeadObjectInput{
		Bucket: aws.String(s.Bucket),
		Key:    aws.String(image),
	}
	head, err := s.Client.HeadObject(ctx, input)
	if err != nil {
		if isNotFoundError(err) {
			return nil, imagor.ErrNotFound
		}
		return nil, err
	}
	return &imagor.Stat{
		Size:         *head.ContentLength,
		ETag:         *head.ETag,
		ModifiedTime: *head.LastModified,
	}, nil
}

func isNotFoundError(err error) bool {
	var nsk *types.NoSuchKey
	var nbf *types.NoSuchBucket
	if errors.As(err, &nsk) || errors.As(err, &nbf) {
		return true
	}
	var ae smithy.APIError
	if errors.As(err, &ae) {
		switch ae.ErrorCode() {
		case "NoSuchKey", "NotFound":
			return true
		}
	}
	return false
}
