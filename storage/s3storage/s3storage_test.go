package s3storage

import (
	"context"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/cshum/imagor"
	"github.com/johannesboyne/gofakes3"
	"github.com/johannesboyne/gofakes3/backend/s3mem"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestS3Store_Path(t *testing.T) {
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
			name:           "defaults ok",
			bucket:         "mybucket",
			image:          "/foo/bar",
			expectedBucket: "mybucket",
			expectedPath:   "/foo/bar",
			expectedOk:     true,
		},
		{
			name:           "escape unsafe chars",
			bucket:         "mybucket",
			image:          "/foo/b{:}ar",
			expectedBucket: "mybucket",
			expectedPath:   "/foo/b%7B%3A%7Dar",
			expectedOk:     true,
		},
		{
			name:           "escape safe chars",
			bucket:         "mybucket",
			image:          "/foo/b{:}\"ar",
			expectedBucket: "mybucket",
			expectedPath:   "/foo/b{%3A}\"ar",
			safeChars:      "{}",
			expectedOk:     true,
		},
		{
			name:           "path under with base uri",
			bucket:         "mybucket",
			baseDir:        "/home/imagor",
			baseURI:        "/foo",
			image:          "/foo/bar",
			expectedBucket: "mybucket",
			expectedPath:   "/home/imagor/bar",
			expectedOk:     true,
		},
		{
			name:           "path under no base uri",
			bucket:         "mybucket",
			baseDir:        "/home/imagor",
			image:          "/foo/bar",
			expectedBucket: "mybucket",
			expectedPath:   "/home/imagor/foo/bar",
			expectedOk:     true,
		},
		{
			name:           "path not under",
			bucket:         "mybucket",
			baseDir:        "/home/imagor",
			baseURI:        "/foo",
			image:          "/fooo/bar",
			expectedBucket: "mybucket",
			expectedOk:     false,
		},
		{
			name:           "extract bucket path under",
			bucket:         "mybucket/home/imagor",
			baseURI:        "/foo",
			image:          "/foo/bar",
			expectedBucket: "mybucket",
			expectedPath:   "/home/imagor/bar",
			expectedOk:     true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sess, err := session.NewSession()
			if err != nil {
				t.Error(err)
			}
			var opts []Option
			if tt.baseURI != "" {
				opts = append(opts, WithPathPrefix(tt.baseURI))
			}
			if tt.baseDir != "" {
				opts = append(opts, WithBaseDir(tt.baseDir))
			}
			opts = append(opts, WithSafeChars(tt.safeChars))
			s := New(sess, tt.bucket, opts...)
			res, ok := s.Path(tt.image)
			if res != tt.expectedPath || ok != tt.expectedOk || s.Bucket != tt.expectedBucket {
				t.Errorf("= %s,%s,%v want %s,%s,%v", tt.bucket, res, ok, tt.expectedBucket, tt.expectedPath, tt.expectedOk)
			}
		})
	}
}

func fakeS3Server() *httptest.Server {
	backend := s3mem.New()
	faker := gofakes3.New(backend)
	return httptest.NewServer(faker.Server())
}

func fakeS3Session(ts *httptest.Server, bucket string) *session.Session {
	config := &aws.Config{
		Credentials:      credentials.NewStaticCredentials("YOUR-ACCESSKEYID", "YOUR-SECRETACCESSKEY", ""),
		Endpoint:         aws.String(ts.URL),
		Region:           aws.String("eu-central-1"),
		DisableSSL:       aws.Bool(true),
		S3ForcePathStyle: aws.Bool(true),
	}
	// activate AWS Session only if credentials present
	sess, err := session.NewSession(config)
	if err != nil {
		panic(err)
	}
	s3Client := s3.New(sess)
	// Create a new bucket using the CreateBucket call.
	_, err = s3Client.CreateBucket(&s3.CreateBucketInput{
		Bucket: aws.String(bucket),
	})
	if err != nil {
		panic(err)
	}
	return sess
}

func TestCRUD(t *testing.T) {
	ts := fakeS3Server()
	defer ts.Close()

	var err error
	ctx := context.Background()
	s := New(fakeS3Session(ts, "test"), "test", WithPathPrefix("/foo"), WithACL("public-read"))

	_, err = s.Get(&http.Request{}, "/bar/fooo/asdf")
	assert.Equal(t, imagor.ErrPass, err)

	_, err = s.Stat(context.Background(), "/bar/fooo/asdf")
	assert.Equal(t, imagor.ErrPass, err)

	_, err = s.Get(&http.Request{}, "/foo/fooo/asdf")
	assert.Equal(t, imagor.ErrNotFound, err)

	assert.ErrorIs(t, s.Put(ctx, "/bar/fooo/asdf", imagor.NewBlobFromBuffer([]byte("bar"))), imagor.ErrPass)

	blob := imagor.NewBlobFromBuffer([]byte("bar"))
	blob.Meta = &imagor.Meta{
		Format:      "abc",
		ContentType: "def",
		Width:       167,
		Height:      169,
	}

	require.NoError(t, s.Put(ctx, "/foo/fooo/asdf", blob))

	b, err := s.Get(&http.Request{}, "/foo/fooo/asdf")
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

	require.NoError(t, s.Put(ctx, "/foo/boo/asdf", imagor.NewBlobFromBuffer([]byte("bar"))))

	_, err = s.Meta(context.Background(), "/foo/boo/asdf")
	assert.Equal(t, imagor.ErrNotFound, err)
}

func TestExpiration(t *testing.T) {
	ts := fakeS3Server()
	defer ts.Close()

	var err error
	ctx := context.Background()
	s := New(fakeS3Session(ts, "test"), "test", WithExpiration(time.Second))

	_, err = s.Get(&http.Request{}, "/foo/bar/asdf")
	assert.Equal(t, imagor.ErrNotFound, err)
	blob := imagor.NewBlobFromBuffer([]byte("bar"))
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
