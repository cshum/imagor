package s3storage

import (
	"bytes"
	"context"
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
	S3       *s3.S3
	Uploader *s3manager.Uploader
	Bucket   string

	BaseDir    string
	PathPrefix string
	ACL        string
	SafeChars  string
	Expiration time.Duration

	safeChars imagorpath.SafeChars
}

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

func (s *S3Storage) Get(r *http.Request, image string) (*imagor.Bytes, error) {
	image, ok := s.Path(image)
	if !ok {
		return nil, imagor.ErrPass
	}
	input := &s3.GetObjectInput{
		Bucket: aws.String(s.Bucket),
		Key:    aws.String(image),
	}
	out, err := s.S3.GetObjectWithContext(r.Context(), input)
	if e, ok := err.(awserr.Error); ok && e.Code() == s3.ErrCodeNoSuchKey {
		return nil, imagor.ErrNotFound
	} else if err != nil {
		return nil, err
	}
	if s.Expiration > 0 && out.LastModified != nil {
		if time.Now().Sub(*out.LastModified) > s.Expiration {
			return nil, imagor.ErrExpired
		}
	}
	buf, err := io.ReadAll(out.Body)
	if err != nil {
		return nil, err
	}
	return imagor.NewBytes(buf), err
}

func (s *S3Storage) Put(ctx context.Context, image string, blob *imagor.Bytes) error {
	image, ok := s.Path(image)
	if !ok {
		return imagor.ErrPass
	}
	buf, err := blob.ReadAll()
	if err != nil {
		return err
	}
	input := &s3manager.UploadInput{
		ACL:         aws.String(s.ACL),
		Body:        bytes.NewReader(buf),
		Bucket:      aws.String(s.Bucket),
		ContentType: aws.String(blob.ContentType()),
		Key:         aws.String(image),
	}
	_, err = s.Uploader.UploadWithContext(ctx, input)
	return err
}

func (s *S3Storage) Stat(ctx context.Context, image string) (stat *imagor.Stat, err error) {
	image, ok := s.Path(image)
	if !ok {
		return nil, imagor.ErrPass
	}
	input := &s3.HeadObjectInput{
		Bucket: aws.String(s.Bucket),
		Key:    aws.String(image),
	}
	out, err := s.S3.HeadObjectWithContext(ctx, input)
	if e, ok := err.(awserr.Error); ok && e.Code() == s3.ErrCodeNoSuchKey {
		return nil, imagor.ErrNotFound
	} else if err != nil {
		return nil, err
	}
	return &imagor.Stat{
		Size:         *out.ContentLength,
		ModifiedTime: *out.LastModified,
	}, nil
}
