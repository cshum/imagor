package s3storage

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/cshum/imagor"
	"github.com/johannesboyne/gofakes3"
	"github.com/johannesboyne/gofakes3/backend/s3mem"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
			name:           "no-op safe chars",
			bucket:         "mybucket",
			image:          "/foo/b{:}\"ar",
			expectedBucket: "mybucket",
			expectedPath:   "/foo/b{:}\"ar",
			safeChars:      "--",
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
			cfg := aws.Config{
				Region: "us-east-1",
			}
			var opts []Option
			if tt.baseURI != "" {
				opts = append(opts, WithPathPrefix(tt.baseURI))
			}
			if tt.baseDir != "" {
				opts = append(opts, WithBaseDir(tt.baseDir))
			}
			opts = append(opts, WithSafeChars(tt.safeChars))
			s := New(cfg, tt.bucket, opts...)
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

func fakeS3Config(ts *httptest.Server, bucket string) aws.Config {
	cfg := aws.Config{
		Region:      "eu-central-1",
		Credentials: credentials.NewStaticCredentialsProvider("YOUR-ACCESSKEYID", "YOUR-SECRETACCESSKEY", ""),
		EndpointResolver: aws.EndpointResolverFunc(func(service, region string) (aws.Endpoint, error) {
			return aws.Endpoint{
				URL:               ts.URL,
				SigningRegion:     region,
				HostnameImmutable: true,
			}, nil
		}),
	}

	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.UsePathStyle = true
	})

	// Create a new bucket
	_, err := client.CreateBucket(context.Background(), &s3.CreateBucketInput{
		Bucket: aws.String(bucket),
	})
	if err != nil {
		panic(err)
	}
	return cfg
}

func TestCRUD(t *testing.T) {
	ts := fakeS3Server()
	defer ts.Close()

	var err error
	ctx := context.Background()
	r := (&http.Request{}).WithContext(ctx)
	s := New(fakeS3Config(ts, "test"), "test", WithPathPrefix("/foo"), WithACL("public-read"))

	_, err = s.Get(r, "/bar/fooo/asdf")
	assert.Equal(t, imagor.ErrInvalid, err)

	_, err = s.Stat(ctx, "/bar/fooo/asdf")
	assert.Equal(t, imagor.ErrInvalid, err)

	assert.ErrorIs(t, s.Put(ctx, "/bar/fooo/asdf", imagor.NewBlobFromBytes([]byte("bar"))), imagor.ErrInvalid)

	assert.Equal(t, imagor.ErrInvalid, s.Delete(ctx, "/bar/fooo/asdf"))

	b, err := s.Get(r, "/foo/fooo/asdf")
	_, err = b.ReadAll()
	assert.Equal(t, imagor.ErrNotFound, err)

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
	assert.NotEmpty(t, stat.ETag, b.Stat.ETag)

	err = s.Delete(ctx, "/foo/fooo/asdf")
	require.NoError(t, err)

	b, err = s.Get(r, "/foo/fooo/asdf")
	_, err = b.ReadAll()
	assert.Equal(t, imagor.ErrNotFound, err)

	_, err = s.Stat(ctx, "/foo/fooo/asdf")
	assert.Equal(t, imagor.ErrNotFound, err)

	require.NoError(t, s.Put(ctx, "/foo/boo/asdf", imagor.NewBlobFromBytes([]byte("bar"))))
}

func TestExpiration(t *testing.T) {
	ts := fakeS3Server()
	defer ts.Close()

	var err error
	ctx := context.Background()
	s := New(fakeS3Config(ts, "test"), "test", WithExpiration(time.Second))

	b, _ := s.Get(&http.Request{}, "/foo/bar/asdf")
	_, err = b.ReadAll()
	assert.Equal(t, imagor.ErrNotFound, err)
	blob := imagor.NewBlobFromBytes([]byte("bar"))
	require.NoError(t, s.Put(ctx, "/foo/bar/asdf", blob))
	b, err = s.Get(&http.Request{}, "/foo/bar/asdf")
	require.NoError(t, err)
	buf, err := b.ReadAll()
	require.NoError(t, err)
	assert.Equal(t, "bar", string(buf))

	time.Sleep(time.Second)
	b, _ = s.Get(&http.Request{}, "/foo/bar/asdf")
	_, err = b.ReadAll()
	require.ErrorIs(t, err, imagor.ErrExpired)
}
