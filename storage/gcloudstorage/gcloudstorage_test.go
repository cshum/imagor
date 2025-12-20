package gcloudstorage

import (
	"bytes"
	"compress/gzip"
	"context"
	"net/http"
	"testing"
	"time"

	"cloud.google.com/go/storage"
	"github.com/cshum/imagor"
	"github.com/fsouza/fake-gcs-server/fakestorage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/api/option"
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
			name:         "no-op safe chars",
			image:        "/foo/b{:}ar",
			expectedPath: "foo/b{:}ar",
			safeChars:    "--",
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
	srv, err := fakestorage.NewServerWithOptions(fakestorage.Options{
		InitialObjects: []fakestorage.Object{{
			ObjectAttrs: fakestorage.ObjectAttrs{
				BucketName: "test",
				Name:       "placeholder",
			},
			Content: []byte(""),
		}},
		NoListener: true,
	})
	require.NoError(t, err)
	defer srv.Stop()

	// Create client manually to avoid credential conflicts
	client, err := storage.NewClient(context.Background(), option.WithHTTPClient(srv.HTTPClient()))
	require.NoError(t, err)
	defer client.Close()

	ctx := context.Background()
	r := (&http.Request{}).WithContext(ctx)
	s := New(client, "test", WithPathPrefix("/foo"), WithACL("publicRead"))

	_, err = s.Get(r, "/bar/fooo/asdf")
	assert.Equal(t, imagor.ErrInvalid, err)

	_, err = s.Stat(ctx, "/bar/fooo/asdf")
	assert.Equal(t, imagor.ErrInvalid, err)

	_, err = s.Stat(ctx, "/foo/fooo/asdf")
	assert.Equal(t, imagor.ErrNotFound, err)

	b, err := s.Get(r, "/foo/fooo/asdf")
	assert.Equal(t, imagor.ErrNotFound, err)

	_, err = s.Stat(ctx, "/foo/fooo/asdf")
	assert.Equal(t, imagor.ErrNotFound, err)

	assert.ErrorIs(t, s.Put(ctx, "/bar/fooo/asdf", imagor.NewBlobFromBytes([]byte("bar"))), imagor.ErrInvalid)

	blob := imagor.NewBlobFromBytes([]byte("bar"))

	require.NoError(t, s.Put(ctx, "/foo/fooo/asdf", blob))

	stat, err := s.Stat(ctx, "/foo/fooo/asdf")
	require.NoError(t, err)
	assert.True(t, stat.ModifiedTime.Before(time.Now()))
	assert.NotEmpty(t, stat.ETag)

	b, err = s.Get(r, "/foo/fooo/asdf")
	require.NoError(t, err)
	buf, err := b.ReadAll()
	require.NoError(t, err)
	assert.Equal(t, "bar", string(buf))
	assert.NotEmpty(t, b.Stat)
	assert.Equal(t, stat.ModifiedTime, b.Stat.ModifiedTime)
	assert.Equal(t, stat.ETag, b.Stat.ETag)

	err = s.Delete(ctx, "/foo/fooo/asdf")
	require.NoError(t, err)

	b, err = s.Get(r, "/foo/fooo/asdf")
	assert.Equal(t, imagor.ErrNotFound, err)

	assert.Equal(t, imagor.ErrInvalid, s.Delete(ctx, "/bar/fooo/asdf"))

	require.NoError(t, s.Put(ctx, "/foo/boo/asdf", imagor.NewBlobFromBytes([]byte("bar"))))
}

func TestExpiration(t *testing.T) {
	srv, err := fakestorage.NewServerWithOptions(fakestorage.Options{
		InitialObjects: []fakestorage.Object{{
			ObjectAttrs: fakestorage.ObjectAttrs{
				BucketName: "test",
				Name:       "placeholder",
			},
			Content: []byte(""),
		}},
		NoListener: true,
	})
	require.NoError(t, err)
	defer srv.Stop()

	// Create client manually to avoid credential conflicts
	client, err := storage.NewClient(context.Background(), option.WithHTTPClient(srv.HTTPClient()))
	require.NoError(t, err)
	defer client.Close()

	s := New(client, "test", WithExpiration(time.Millisecond*10))
	ctx := context.Background()

	_, err = s.Get(&http.Request{}, "/foo/bar/asdf")
	assert.Equal(t, imagor.ErrNotFound, err)
	blob := imagor.NewBlobFromBytes([]byte("bar"))
	require.NoError(t, s.Put(ctx, "/foo/bar/asdf", blob))
	b, err := s.Get(&http.Request{}, "/foo/bar/asdf")
	require.NoError(t, err)
	buf, err := b.ReadAll()
	require.NoError(t, err)
	assert.Equal(t, "bar", string(buf))

	time.Sleep(time.Second)
	_, err = s.Get(&http.Request{}, "/foo/bar/asdf")
	require.ErrorIs(t, err, imagor.ErrExpired)
}

func TestContextCancel(t *testing.T) {
	srv, err := fakestorage.NewServerWithOptions(fakestorage.Options{
		InitialObjects: []fakestorage.Object{{
			ObjectAttrs: fakestorage.ObjectAttrs{
				BucketName: "test",
				Name:       "placeholder",
			},
			Content: []byte(""),
		}},
		NoListener: true,
	})
	require.NoError(t, err)
	defer srv.Stop()

	// Create client manually to avoid credential conflicts
	client, err := storage.NewClient(context.Background(), option.WithHTTPClient(srv.HTTPClient()))
	require.NoError(t, err)
	defer client.Close()

	s := New(client, "test")
	ctx, cancel := context.WithCancel(context.Background())
	r, err2 := http.NewRequestWithContext(ctx, http.MethodGet, "", nil)
	require.NoError(t, err2)
	blob := imagor.NewBlobFromBytes([]byte("bar"))
	require.NoError(t, s.Put(ctx, "/foo/bar/asdf", blob))
	b, err := s.Get(r, "/foo/bar/asdf")
	require.NoError(t, err)
	buf, err := b.ReadAll()
	require.NoError(t, err)
	assert.Equal(t, "bar", string(buf))
	cancel()
	b, err = s.Get(r, "/foo/bar/asdf")
	require.ErrorIs(t, err, context.Canceled)
}

func TestContextCancelDuringBlobInit(t *testing.T) {
	// This test validates that nil reader protection works when context is cancelled
	// after Get() succeeds but before the blob is read (during lazy initialization)
	srv, err := fakestorage.NewServerWithOptions(fakestorage.Options{
		InitialObjects: []fakestorage.Object{{
			ObjectAttrs: fakestorage.ObjectAttrs{
				BucketName: "test",
				Name:       "foo/bar/test",
			},
			Content: []byte("test content for deferred read"),
		}},
		NoListener: true,
	})
	require.NoError(t, err)
	defer srv.Stop()

	// Create client manually to avoid credential conflicts
	client, err := storage.NewClient(context.Background(), option.WithHTTPClient(srv.HTTPClient()))
	require.NoError(t, err)
	defer client.Close()

	s := New(client, "test")

	// Create a context with very short timeout to simulate cancellation during blob init
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*1)
	defer cancel()

	r, err2 := http.NewRequestWithContext(ctx, http.MethodGet, "", nil)
	require.NoError(t, err2)

	// Wait a bit to ensure context times out
	time.Sleep(time.Millisecond * 10)

	// Get should fail because context is already cancelled
	// The fix ensures we get a proper error instead of a panic
	b, err3 := s.Get(r, "/foo/bar/test")

	// Either Get fails immediately (expected), or if Get succeeds,
	// ReadAll should fail gracefully without panic
	if err3 == nil {
		_, err3 = b.ReadAll()
		require.Error(t, err3)
	}
	// Both scenarios should result in an error, never a panic
	require.Error(t, err3)
}

func TestGCloudStorage_GzipContentEncoding(t *testing.T) {
	// Test that gzip-compressed objects don't cause fanout buffer size issues

	// Create properly gzipped content
	originalContent := "this content is longer than 20 bytes when decompressed and should not be truncated"
	var gzipBuf bytes.Buffer
	gzipWriter := gzip.NewWriter(&gzipBuf)
	_, err := gzipWriter.Write([]byte(originalContent))
	require.NoError(t, err)
	require.NoError(t, gzipWriter.Close())

	gzippedContent := gzipBuf.Bytes()

	srv, err := fakestorage.NewServerWithOptions(fakestorage.Options{
		InitialObjects: []fakestorage.Object{{
			ObjectAttrs: fakestorage.ObjectAttrs{
				BucketName:      "test",
				Name:            "test-gzip",
				ContentEncoding: "gzip",
				Size:            int64(len(gzippedContent)), // Actual compressed size
			},
			Content: gzippedContent,
		}},
		NoListener: true,
	})
	require.NoError(t, err)
	defer srv.Stop()

	// Create client manually to avoid credential conflicts
	client, err := storage.NewClient(context.Background(), option.WithHTTPClient(srv.HTTPClient()))
	require.NoError(t, err)
	defer client.Close()

	ctx := context.Background()
	r := (&http.Request{}).WithContext(ctx)
	s := New(client, "test")

	// Test the fix - should not truncate despite size mismatch
	b, err := s.Get(r, "/test-gzip")
	require.NoError(t, err)

	buf, err := b.ReadAll()
	require.NoError(t, err)
	assert.Equal(t, originalContent, string(buf))

	// Verify blob stats are still set correctly
	assert.NotNil(t, b.Stat)
	// The stat size will still show compressed size, which is correct behavior
	assert.Equal(t, int64(len(gzippedContent)), b.Stat.Size)

	// The key fix is that content is fully readable despite size mismatch
	// (the fanout optimization is disabled internally for gzip content)
}
