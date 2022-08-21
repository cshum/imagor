package gcloudstorage

import (
	"cloud.google.com/go/storage"
	"context"
	"errors"
	"github.com/cshum/imagor"
	"github.com/cshum/imagor/imagorpath"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"time"
)

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

type statKey struct {
	Key string
}

func New(client *storage.Client, bucket string, options ...Option) *GCloudStorage {
	s := &GCloudStorage{client: client, Bucket: bucket}
	for _, option := range options {
		option(s)
	}
	s.safeChars = imagorpath.NewSafeChars(s.SafeChars)
	return s
}

func (s *GCloudStorage) Get(r *http.Request, image string) (imageData *imagor.Blob, err error) {
	image, ok := s.Path(image)
	if !ok {
		return nil, imagor.ErrInvalid
	}
	object := s.client.Bucket(s.Bucket).Object(image)
	attrs, err := object.Attrs(r.Context())
	if err != nil {
		if errors.Is(err, storage.ErrObjectNotExist) {
			return nil, imagor.ErrNotFound
		}
		return nil, err
	}
	imagor.ContextCachePut(r.Context(), statKey{image}, imagor.Stat{
		Size:         attrs.Size,
		ModifiedTime: attrs.Updated,
	})
	if s.Expiration > 0 {
		if attrs != nil && time.Now().Sub(attrs.Updated) > s.Expiration {
			return nil, imagor.ErrExpired
		}
	}
	return imagor.NewBlob(func() (reader io.ReadCloser, size int64, err error) {
		if attrs != nil {
			size = attrs.Size
		}
		reader, err = object.NewReader(r.Context())
		return
	}), err
}

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
		_ = writer.Close()
	}()
	if s.ACL != "" {
		writer.PredefinedACL = s.ACL
	}
	writer.ContentType = blob.ContentType()
	if _, err := io.Copy(writer, reader); err != nil {
		return err
	}
	return writer.Close()
}

func (s *GCloudStorage) Delete(ctx context.Context, image string) error {
	image, ok := s.Path(image)
	if !ok {
		return imagor.ErrInvalid
	}
	return s.client.Bucket(s.Bucket).Object(image).Delete(ctx)
}

func (s *GCloudStorage) Path(image string) (string, bool) {
	image = "/" + imagorpath.Normalize(image, s.safeChars)

	if !strings.HasPrefix(image, s.PathPrefix) {
		return "", false
	}
	joinedPath := filepath.Join(s.BaseDir, strings.TrimPrefix(image, s.PathPrefix))
	// Google cloud paths don't need to start with "/"
	return strings.Trim(joinedPath, "/"), true
}

func (s *GCloudStorage) attrs(ctx context.Context, image string) (attrs *storage.ObjectAttrs, err error) {
	image, ok := s.Path(image)
	if !ok {
		return nil, imagor.ErrInvalid
	}
	object := s.client.Bucket(s.Bucket).Object(image)
	attrs, err = object.Attrs(ctx)
	if err != nil {
		if errors.Is(err, storage.ErrObjectNotExist) {
			return nil, imagor.ErrNotFound
		}
		return nil, err
	}
	return attrs, err
}

func (s *GCloudStorage) Stat(ctx context.Context, image string) (stat *imagor.Stat, err error) {
	if s, ok := imagor.ContextCacheGet(ctx, statKey{image}); ok && s != nil {
		if stat, ok2 := s.(imagor.Stat); ok2 {
			return &stat, nil
		}
	}
	attrs, err := s.attrs(ctx, image)
	if err != nil {
		return nil, err
	}
	return &imagor.Stat{
		Size:         attrs.Size,
		ModifiedTime: attrs.Updated,
	}, nil
}
