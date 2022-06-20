package s3storage

import (
	"context"
	"encoding/json"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/cshum/imagor"
	"github.com/cshum/imagor/imagorpath"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"time"
)

type S3Storage struct {
	S3         *s3.S3
	Uploader   *s3manager.Uploader
	Downloader *s3manager.Downloader
	Bucket     string

	BaseDir    string
	PathPrefix string
	ACL        string
	SafeChars  string
	Expiration time.Duration

	safeChars imagorpath.SafeChars
}

const metaKey = "Imagor-Meta"

func New(sess *session.Session, bucket string, options ...Option) *S3Storage {
	baseDir := "/"
	if idx := strings.Index(bucket, "/"); idx > -1 {
		baseDir = bucket[idx:]
		bucket = bucket[:idx]
	}
	s := &S3Storage{
		S3:       s3.New(sess),
		Uploader: s3manager.NewUploader(sess),
		Bucket:   bucket,

		BaseDir:    baseDir,
		PathPrefix: "/",
		ACL:        s3.ObjectCannedACLPublicRead,
	}
	for _, option := range options {
		option(s)
	}
	s.safeChars = imagorpath.NewSafeChars("!\"()*" + s.SafeChars)
	// https://docs.aws.amazon.com/AmazonS3/latest/userguide/object-keys.html#object-key-guidelines-safe-characters

	return s
}

func (s *S3Storage) Path(image string) (string, bool) {
	image = "/" + imagorpath.Normalize(image, s.safeChars)
	if !strings.HasPrefix(image, s.PathPrefix) {
		return "", false
	}
	return filepath.Join(s.BaseDir, strings.TrimPrefix(image, s.PathPrefix)), true
}

func (s *S3Storage) Get(r *http.Request, image string) (*imagor.Blob, error) {
	image, ok := s.Path(image)
	if !ok {
		return nil, imagor.ErrPass
	}
	return imagor.NewBlob(func() (io.ReadCloser, int64, error) {
		input := &s3.GetObjectInput{
			Bucket: aws.String(s.Bucket),
			Key:    aws.String(image),
		}
		out, err := s.S3.GetObjectWithContext(r.Context(), input)
		if e, ok := err.(awserr.Error); ok && e.Code() == s3.ErrCodeNoSuchKey {
			return nil, 0, imagor.ErrNotFound
		} else if err != nil {
			return nil, 0, err
		}
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
	}), nil
}

func (s *S3Storage) Put(ctx context.Context, image string, blob *imagor.Blob) error {
	image, ok := s.Path(image)
	if !ok {
		return imagor.ErrPass
	}
	reader, _, err := blob.NewReader()
	if err != nil {
		return err
	}
	defer func() {
		_ = reader.Close()
	}()
	var metadata map[string]*string
	if blob.Meta != nil {
		if buf, _ := json.Marshal(blob.Meta); len(buf) > 0 {
			metadata = map[string]*string{
				metaKey: aws.String(string(buf)),
			}
		}
	}
	input := &s3manager.UploadInput{
		ACL:         aws.String(s.ACL),
		Body:        reader,
		Bucket:      aws.String(s.Bucket),
		ContentType: aws.String(blob.ContentType()),
		Metadata:    metadata,
		Key:         aws.String(image),
	}
	_, err = s.Uploader.UploadWithContext(ctx, input)
	return err
}

func (s *S3Storage) Del(ctx context.Context, image string) error {
	image, ok := s.Path(image)
	if !ok {
		return imagor.ErrPass
	}
	_, err := s.S3.DeleteObjectWithContext(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.Bucket),
		Key:    aws.String(image),
	})
	return err
}

func (s *S3Storage) head(ctx context.Context, image string) (*s3.HeadObjectOutput, error) {
	image, ok := s.Path(image)
	if !ok {
		return nil, imagor.ErrPass
	}
	input := &s3.HeadObjectInput{
		Bucket: aws.String(s.Bucket),
		Key:    aws.String(image),
	}
	head, err := s.S3.HeadObjectWithContext(ctx, input)
	if e, ok := err.(awserr.Error); ok && e.Code() == s3.ErrCodeNoSuchKey {
		return nil, imagor.ErrNotFound
	} else if err != nil {
		return nil, err
	}
	return head, nil
}

func (s *S3Storage) Stat(ctx context.Context, image string) (stat *imagor.Stat, err error) {
	head, err := s.head(ctx, image)
	if err != nil {
		return nil, err
	}
	return &imagor.Stat{
		Size:         *head.ContentLength,
		ModifiedTime: *head.LastModified,
	}, nil
}

func (s *S3Storage) Meta(ctx context.Context, image string) (meta *imagor.Meta, err error) {
	head, err := s.head(ctx, image)
	if err != nil {
		return nil, err
	}
	if head.Metadata == nil || head.Metadata[metaKey] == nil || *head.Metadata[metaKey] == "" {
		return nil, imagor.ErrNotFound
	}
	if s.Expiration > 0 && head.LastModified != nil {
		if time.Now().Sub(*head.LastModified) > s.Expiration {
			return nil, imagor.ErrExpired
		}
	}
	meta = &imagor.Meta{}
	if err := json.Unmarshal([]byte(*head.Metadata[metaKey]), meta); err != nil {
		return nil, err
	}
	return meta, nil
}
