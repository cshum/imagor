package gcloudstorage

import (
	"context"
	"errors"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"cloud.google.com/go/storage"
	"github.com/cshum/imagor"
	"github.com/cshum/imagor/imagorpath"
)

// GCloudStorage Google Cloud Storage implements imagor.Storage interface
type GCloudStorage struct {
	BaseDir    string
	PathPrefix string
	ACL        string
	SafeChars  string
	Expiration time.Duration
	client     *storage.Client
	Bucket     string

	safeChars imagorpath.SafeChars
}

// New creates GCloudStorage
func New(client *storage.Client, bucket string, options ...Option) *GCloudStorage {
	s := &GCloudStorage{client: client, Bucket: bucket}
	for _, option := range options {
		option(s)
	}
	s.safeChars = imagorpath.NewSafeChars(s.SafeChars)
	return s
}

// Get implements imagor.Storage interface
func (s *GCloudStorage) Get(r *http.Request, image string) (imageData *imagor.Blob, err error) {
	ctx := r.Context()
	image, ok := s.Path(image)
	if !ok {
		return nil, imagor.ErrInvalid
	}
	object := s.client.Bucket(s.Bucket).Object(image)
	attrs, err := object.Attrs(ctx)
	if err != nil {
		if errors.Is(err, storage.ErrObjectNotExist) {
			return nil, imagor.ErrNotFound
		}
		return nil, err
	}
	if s.Expiration > 0 {
		if attrs != nil && time.Now().Sub(attrs.Updated) > s.Expiration {
			return nil, imagor.ErrExpired
		}
	}
	blob := imagor.NewBlob(func() (reader io.ReadCloser, size int64, err error) {
		if err = ctx.Err(); err != nil {
			return
		}
		if attrs != nil {
			size = attrs.Size
		}
		reader, err = object.NewReader(context.Background())
		return
	})
	if attrs != nil {
		blob.SetContentType(attrs.ContentType)
		blob.Stat = &imagor.Stat{
			Size:         attrs.Size,
			ETag:         attrs.Etag,
			ModifiedTime: attrs.Updated,
		}
	}
	return blob, err
}

// Put implements imagor.Storage interface
func (s *GCloudStorage) Put(ctx context.Context, image string, blob *imagor.Blob) (err error) {
	image, ok := s.Path(image)
	if !ok {
		return imagor.ErrInvalid
	}
	reader, _, err := blob.NewReader()
	if err != nil {
		return err
	}
	objectHandle := s.client.Bucket(s.Bucket).Object(image)
	writer := objectHandle.NewWriter(ctx)
	defer func() {
		_ = reader.Close()
		_ = writer.Close()
	}()
	if s.ACL != "" {
		writer.PredefinedACL = s.ACL
	}
	writer.ContentType = blob.ContentType()
	if _, err = io.Copy(writer, reader); err != nil {
		return err
	}
	return
}

// Delete implements imagor.Storage interface
func (s *GCloudStorage) Delete(ctx context.Context, image string) error {
	image, ok := s.Path(image)
	if !ok {
		return imagor.ErrInvalid
	}
	return s.client.Bucket(s.Bucket).Object(image).Delete(ctx)
}

// Path transforms and validates image key for storage path
func (s *GCloudStorage) Path(image string) (string, bool) {
	image = "/" + imagorpath.Normalize(image, s.safeChars)

	if !strings.HasPrefix(image, s.PathPrefix) {
		return "", false
	}
	joinedPath := filepath.Join(s.BaseDir, strings.TrimPrefix(image, s.PathPrefix))
	// Google cloud paths don't need to start with "/"
	return strings.Trim(joinedPath, "/"), true
}

// Stat implements imagor.Storage interface
func (s *GCloudStorage) Stat(ctx context.Context, image string) (stat *imagor.Stat, err error) {
	image, ok := s.Path(image)
	if !ok {
		return nil, imagor.ErrInvalid
	}
	object := s.client.Bucket(s.Bucket).Object(image)
	attrs, err := object.Attrs(ctx)
	if err != nil {
		if errors.Is(err, storage.ErrObjectNotExist) {
			return nil, imagor.ErrNotFound
		}
		return nil, err
	}
	return &imagor.Stat{
		Size:         attrs.Size,
		ETag:         attrs.Etag,
		ModifiedTime: attrs.Updated,
	}, nil
}
