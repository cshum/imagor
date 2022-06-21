package gcloudstorage

import (
	"context"
	"github.com/cshum/imagor"
	"github.com/fsouza/fake-gcs-server/fakestorage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net/http"
	"testing"
	"time"
)

func TestGCloudStorage_Path(t *testing.T) {

	tests := []struct {
		name           string
		bucket         string
		baseDir        string
		baseURI        string
		image          string
		safeChars      string
		expectedPath   string
		expectedBucket string
		expectedOk     bool
	}{
		{
			name:         "defaults ok",
			image:        "/foo/bar",
			expectedPath: "foo/bar",
			expectedOk:   true,
		},
		{
			name:         "escape unsafe chars",
			image:        "/foo/b{:}ar",
			expectedPath: "foo/b%7B%3A%7Dar",
			expectedOk:   true,
		},
		{
			name:         "escape safe chars",
			image:        "/foo/b{:}ar",
			expectedPath: "foo/b{%3A}ar",
			safeChars:    "{}",
			expectedOk:   true,
		},
		{
			name:         "path under with base uri",
			baseDir:      "home/imagor",
			baseURI:      "/foo",
			image:        "/foo/bar",
			expectedPath: "home/imagor/bar",
			expectedOk:   true,
		},
		{
			name:         "path under no base uri",
			baseDir:      "/home/imagor",
			image:        "/foo/bar",
			expectedPath: "home/imagor/foo/bar",
			expectedOk:   true,
		},
		{
			name:       "path not under",
			baseDir:    "/home/imagor",
			baseURI:    "/foo",
			image:      "/fooo/bar",
			expectedOk: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var opts []Option
			if tt.baseURI != "" {
				opts = append(opts, WithPathPrefix(tt.baseURI))
			}
			if tt.baseDir != "" {
				opts = append(opts, WithBaseDir(tt.baseDir))
			}
			opts = append(opts, WithSafeChars(tt.safeChars))

			s := New(nil, tt.bucket, opts...)
			res, ok := s.Path(tt.image)
			if res != tt.expectedPath || ok != tt.expectedOk {
				t.Errorf("= %s,%v want %s,%v", res, ok, tt.expectedPath, tt.expectedOk)
			}
		})
	}
}

func TestCRUD(t *testing.T) {
	srv := fakestorage.NewServer([]fakestorage.Object{{
		ObjectAttrs: fakestorage.ObjectAttrs{
			BucketName: "test",
			Name:       "placeholder",
		},
		Content: []byte(""),
	}})
	ctx := context.Background()
	s := New(srv.Client(), "test", WithPathPrefix("/foo"), WithACL("publicRead"))
	var err error

	_, err = s.Get(&http.Request{}, "/bar/fooo/asdf")
	assert.Equal(t, imagor.ErrInvalid, err)

	_, err = s.Stat(context.Background(), "/bar/fooo/asdf")
	assert.Equal(t, imagor.ErrInvalid, err)

	_, err = s.Stat(context.Background(), "/foo/fooo/asdf")
	assert.Equal(t, imagor.ErrNotFound, err)

	b, err := s.Get(&http.Request{}, "/foo/fooo/asdf")
	assert.Equal(t, imagor.ErrNotFound, err)

	_, err = s.Stat(context.Background(), "/foo/fooo/asdf")
	assert.Equal(t, imagor.ErrNotFound, err)

	_, err = s.Meta(context.Background(), "/foo/fooo/asdf")
	assert.Equal(t, imagor.ErrNotFound, err)

	assert.ErrorIs(t, s.Put(ctx, "/bar/fooo/asdf", imagor.NewBlobFromBytes([]byte("bar"))), imagor.ErrInvalid)

	blob := imagor.NewBlobFromBytes([]byte("bar"))
	blob.Meta = &imagor.Meta{
		Format:      "abc",
		ContentType: "def",
		Width:       167,
		Height:      169,
	}

	require.NoError(t, s.Put(ctx, "/foo/fooo/asdf", blob))

	b, err = s.Get(&http.Request{}, "/foo/fooo/asdf")
	require.NoError(t, err)
	buf, err := b.ReadAll()
	require.NoError(t, err)
	assert.Equal(t, "bar", string(buf))

	stat, err := s.Stat(ctx, "/foo/fooo/asdf")
	require.NoError(t, err)
	assert.True(t, stat.ModifiedTime.Before(time.Now()))

	meta, err := s.Meta(context.Background(), "/foo/fooo/asdf")
	require.NoError(t, err)
	assert.Equal(t, meta, blob.Meta)

	err = s.Delete(context.Background(), "/foo/fooo/asdf")
	require.NoError(t, err)

	b, err = s.Get(&http.Request{}, "/foo/fooo/asdf")
	assert.Equal(t, imagor.ErrNotFound, err)

	assert.Equal(t, imagor.ErrInvalid, s.Delete(context.Background(), "/bar/fooo/asdf"))

	require.NoError(t, s.Put(ctx, "/foo/boo/asdf", imagor.NewBlobFromBytes([]byte("bar"))))

	_, err = s.Meta(context.Background(), "/foo/boo/asdf")
	assert.Equal(t, imagor.ErrNotFound, err)
}

func TestExpiration(t *testing.T) {
	srv := fakestorage.NewServer([]fakestorage.Object{{
		ObjectAttrs: fakestorage.ObjectAttrs{
			BucketName: "test",
			Name:       "placeholder",
		},
		Content: []byte(""),
	}})
	s := New(srv.Client(), "test", WithExpiration(time.Millisecond*10))
	ctx := context.Background()
	var err error

	_, err = s.Get(&http.Request{}, "/foo/bar/asdf")
	assert.Equal(t, imagor.ErrNotFound, err)
	blob := imagor.NewBlobFromBytes([]byte("bar"))
	blob.Meta = &imagor.Meta{
		Format:      "abc",
		ContentType: "def",
		Width:       167,
		Height:      169,
	}
	require.NoError(t, s.Put(ctx, "/foo/bar/asdf", blob))
	b, err := s.Get(&http.Request{}, "/foo/bar/asdf")
	require.NoError(t, err)
	buf, err := b.ReadAll()
	require.NoError(t, err)
	assert.Equal(t, "bar", string(buf))

	meta, err := s.Meta(context.Background(), "/foo/bar/asdf")
	require.NoError(t, err)
	assert.Equal(t, meta, blob.Meta)

	time.Sleep(time.Second)
	_, err = s.Get(&http.Request{}, "/foo/bar/asdf")
	require.ErrorIs(t, err, imagor.ErrExpired)
	_, err = s.Meta(context.Background(), "/foo/bar/asdf")
	require.ErrorIs(t, err, imagor.ErrExpired)
}
