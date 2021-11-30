package s3store

import (
	"bytes"
	"context"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/cshum/imagor"
	"io"
	"mime"
	"net/http"
	"path"
	"path/filepath"
	"strings"
)

type s3Store struct {
	S3       *s3.S3
	Uploader *s3manager.Uploader
	Bucket   string

	BaseDir string
	BaseURI string
}

func New(session *session.Session, bucket string) *s3Store {
	s := &s3Store{
		S3:       s3.New(session),
		Uploader: s3manager.NewUploader(session),
		Bucket:   bucket,

		BaseDir: "/",
		BaseURI: "/",
	}
	return s
}

func (s *s3Store) Path(image string) (string, bool) {
	image = "/" + strings.TrimPrefix(path.Clean(
		strings.ReplaceAll(image, ":/", "%3A"),
	), "/")
	if !strings.HasPrefix(image, s.BaseURI) {
		return "", false
	}
	return filepath.Join(s.BaseDir, strings.TrimPrefix(image, s.BaseURI)), true
}

func (s *s3Store) Load(r *http.Request, image string) ([]byte, error) {
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
	return io.ReadAll(out.Body)
}

func (s *s3Store) Store(ctx context.Context, image string, buf []byte) error {
	image, ok := s.Path(image)
	if !ok {
		return imagor.ErrPass
	}
	input := &s3manager.UploadInput{
		ACL:         aws.String(s3.ObjectCannedACLPublicRead),
		Body:        bytes.NewReader(buf),
		Bucket:      aws.String(s.Bucket),
		ContentType: aws.String(mime.TypeByExtension(filepath.Ext(image))),
		Key:         aws.String(image),
	}
	_, err := s.Uploader.UploadWithContext(ctx, input)
	return err
}
