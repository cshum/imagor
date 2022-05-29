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
	bucket     string
	safeChars  map[byte]bool
}

func New(client *storage.Client, bucket string, options ...Option) *GCloudStorage {
	s := &GCloudStorage{client: client, bucket: bucket, safeChars: map[byte]bool{}}
	for _, option := range options {
		option(s)
	}
	for _, c := range s.SafeChars {
		s.safeChars[byte(c)] = true
	}
	return s
}

func (s *GCloudStorage) Get(r *http.Request, image string) (imageData *imagor.Bytes, err error) {
	image, ok := s.Path(image)
	if !ok {
		return nil, imagor.ErrPass
	}
	object := s.client.Bucket(s.bucket).Object(image)

	// Verify attributes only if expiration is set to avoid additional requests
	if s.Expiration > 0 {
		attrs, err := object.Attrs(r.Context())
		if err != nil {
			if errors.Is(err, storage.ErrObjectNotExist) {
				return nil, imagor.ErrNotFound
			}
			return nil, err
		}
		if attrs != nil && time.Now().Sub(attrs.Updated) > s.Expiration {
			return nil, imagor.ErrExpired
		}
	}

	reader, err := object.NewReader(r.Context())
	if err != nil {
		if errors.Is(err, storage.ErrObjectNotExist) {
			return nil, imagor.ErrNotFound
		}
		return nil, err
	}
	defer func() {
		if readerErr := reader.Close(); err == nil && readerErr != nil {
			err = readerErr
		}
	}()

	buf, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	return imagor.NewBytes(buf), err
}

func (s *GCloudStorage) Put(ctx context.Context, image string, blob *imagor.Bytes) (err error) {
	image, ok := s.Path(image)
	if !ok {
		return imagor.ErrPass
	}
	buf, err := blob.ReadAll()
	if err != nil {
		return err
	}

	objectHandle := s.client.Bucket(s.bucket).Object(image)
	writer := objectHandle.NewWriter(ctx)
	defer func() {
		if writerErr := writer.Close(); err == nil && writerErr != nil {
			err = writerErr
		}
	}()
	if s.ACL != "" {
		writer.PredefinedACL = s.ACL
	}

	if _, err := writer.Write(buf); err != nil {
		return err
	}

	return writer.Close()
}

func (s *GCloudStorage) Path(image string) (string, bool) {
	image = "/" + imagorpath.Normalize(image, s.escapeByte)

	if !strings.HasPrefix(image, s.PathPrefix) {
		return "", false
	}
	joinedPath := filepath.Join(s.BaseDir, strings.TrimPrefix(image, s.PathPrefix))
	// Google cloud paths don't need to start with "/"
	return strings.Trim(joinedPath, "/"), true
}

func (s *GCloudStorage) escapeByte(c byte) bool {
	switch c {
	// Escape google recommendation: https://cloud.google.com/storage/docs/naming-objects
	case '#', '[', ']', '*', '?':
		return true
	}
	if len(s.safeChars) > 0 && s.safeChars[c] {
		// safe chars from config
		return false
	}
	// Anything else - use defaults
	return imagorpath.DefaultEscapeByte(c)
}

func (s *GCloudStorage) Stat(ctx context.Context, image string) (stat *imagor.Stat, err error) {
	image, ok := s.Path(image)
	if !ok {
		return nil, imagor.ErrPass
	}
	object := s.client.Bucket(s.bucket).Object(image)

	attrs, err := object.Attrs(ctx)
	if err != nil {
		if errors.Is(err, storage.ErrObjectNotExist) {
			return nil, imagor.ErrNotFound
		}
		return nil, err
	}
	return &imagor.Stat{
		Size:         attrs.Size,
		ModifiedTime: attrs.Updated,
	}, nil
}
