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
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
	"github.com/cshum/imagor"
	"github.com/cshum/imagor/imagorpath"
)

// S3Storage AWS S3 Storage implements imagor.Storage interface
type S3Storage struct {
	Client *s3.Client
	Bucket string

	BaseDir        string
	PathPrefix     string
	ACL            string
	SafeChars      string
	StorageClass   string
	Expiration     time.Duration
	Endpoint       string
	ForcePathStyle bool

	safeChars imagorpath.SafeChars
}

// New creates S3Storage
func New(cfg aws.Config, bucket string, options ...Option) *S3Storage {
	baseDir := "/"
	if idx := strings.Index(bucket, "/"); idx > -1 {
		baseDir = bucket[idx:]
		bucket = bucket[:idx]
	}
	s := &S3Storage{
		Bucket: bucket,

		BaseDir:    baseDir,
		PathPrefix: "/",
		ACL:        string(types.ObjectCannedACLPublicRead),
	}
	for _, option := range options {
		option(s)
	}

	// Create S3 client with endpoint and path style options
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
		// https://docs.aws.amazon.com/AmazonS3/latest/userguide/object-keys.html#object-key-guidelines-safe-characters
	}

	return s
}

// Path transforms and validates image key for storage path
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

// Get implements imagor.Storage interface
func (s *S3Storage) Get(r *http.Request, image string) (*imagor.Blob, error) {
	ctx := r.Context()
	image, ok := s.Path(image)
	if !ok {
		return nil, imagor.ErrInvalid
	}
	var blob *imagor.Blob
	var once sync.Once
	blob = imagor.NewBlob(func() (io.ReadCloser, int64, error) {
		input := &s3.GetObjectInput{
			Bucket: aws.String(s.Bucket),
			Key:    aws.String(image),
		}
		out, err := s.Client.GetObject(ctx, input)
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

// Put implements imagor.Storage interface
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

// Delete implements imagor.Storage interface
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

// Stat implements imagor.Storage interface
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

// Helper function for not found errors
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
