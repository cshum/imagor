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
	"mime"
	"net/http"
	"path/filepath"
	"strings"
)

type S3Storage struct {
	S3       *s3.S3
	Uploader *s3manager.Uploader
	Bucket   string

	BaseDir    string
	PathPrefix string
	ACL        string
	SafeChars  string

	safeChars map[byte]bool
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

		safeChars: map[byte]bool{},
	}
	for _, option := range options {
		option(s)
	}
	for _, c := range s.SafeChars {
		s.safeChars[byte(c)] = true
	}
	return s
}

func (s *S3Storage) shouldEscape(c byte) bool {
	// alphanum
	if 'a' <= c && c <= 'z' || 'A' <= c && c <= 'Z' || '0' <= c && c <= '9' {
		return false
	}
	switch c {
	case '/': // should not escape path segment
		return false
	case '-', '_', '.', '~': // Unreserved characters
		return false
	case '!', '\'', '(', ')', '*':
		// https://docs.aws.amazon.com/AmazonS3/latest/userguide/object-keys.html#object-key-guidelines-safe-characters
		return false
	}
	if len(s.safeChars) > 0 && s.safeChars[c] {
		// safe chars from config
		return false
	}
	// Everything else must be escaped.
	return true
}

func (s *S3Storage) Path(image string) (string, bool) {
	image = "/" + imagorpath.Normalize(image, s.shouldEscape)
	if !strings.HasPrefix(image, s.PathPrefix) {
		return "", false
	}
	return filepath.Join(s.BaseDir, strings.TrimPrefix(image, s.PathPrefix)), true
}

func (s *S3Storage) Load(r *http.Request, image string) (*imagor.Blob, error) {
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
	buf, err := io.ReadAll(out.Body)
	if err != nil {
		return nil, err
	}
	return imagor.NewBlobBytes(buf), err
}

func (s *S3Storage) Save(ctx context.Context, image string, blob *imagor.Blob) error {
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
		ContentType: aws.String(mime.TypeByExtension(filepath.Ext(image))),
		Key:         aws.String(image),
	}
	_, err = s.Uploader.UploadWithContext(ctx, input)
	return err
}
