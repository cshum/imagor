package s3store

import (
	"bytes"
	"context"
	"fmt"
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

type S3Store struct {
	S3       *s3.S3
	Uploader *s3manager.Uploader
	Bucket   string

	BaseDir    string
	PathPrefix string
	ACL        string
}

func New(sess *session.Session, bucket string, options ...Option) *S3Store {
	baseDir := "/"
	if idx := strings.Index(bucket, "/"); idx > -1 {
		baseDir = bucket[idx:]
		bucket = bucket[:idx]
	}
	s := &S3Store{
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
	return s
}

func (s *S3Store) Path(image string) (string, bool) {
	image = "/" + imagorpath.Clean(image)
	if !strings.HasPrefix(image, s.PathPrefix) {
		return "", false
	}
	return filepath.Join(s.BaseDir, strings.TrimPrefix(image, s.PathPrefix)), true
}

func (s *S3Store) Load(r *http.Request, image string) (*imagor.File, error) {
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
	return imagor.NewFileBytes(buf), err
}

func (s *S3Store) Save(ctx context.Context, image string, file *imagor.File) error {
	image, ok := s.Path(image)
	if !ok {
		return imagor.ErrPass
	}
	fmt.Println(image)
	buf, err := file.Bytes()
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
