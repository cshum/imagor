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
			expectedPath:   "foo/bar",
			expectedOk:     true,
		},
		{
			name:           "escape unsafe chars",
			bucket:         "mybucket",
			image:          "/foo/b{:}ar",
			expectedBucket: "mybucket",
			expectedPath:   "foo/b%7B%3A%7Dar",
			expectedOk:     true,
		},
		{
			name:           "escape safe chars",
			bucket:         "mybucket",
			image:          "/foo/b{:}\"ar",
			expectedBucket: "mybucket",
			expectedPath:   "foo/b{%3A}\"ar",
			safeChars:      "{}",
			expectedOk:     true,
		},
		{
			name:           "no-op safe chars",
			bucket:         "mybucket",
			image:          "/foo/b{:}\"ar",
			expectedBucket: "mybucket",
			expectedPath:   "foo/b{:}\"ar",
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
			expectedPath:   "home/imagor/bar",
			expectedOk:     true,
		},
		{
			name:           "path under no base uri",
			bucket:         "mybucket",
			baseDir:        "/home/imagor",
			image:          "/foo/bar",
			expectedBucket: "mybucket",
			expectedPath:   "home/imagor/foo/bar",
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
			expectedPath:   "home/imagor/bar",
			expectedOk:     true,
		},
		{
			name:           "leading slash stripped with root base dir",
			bucket:         "mybucket",
			baseDir:        "/",
			image:          "/foo/bar",
			expectedBucket: "mybucket",
			expectedPath:   "foo/bar",
			expectedOk:     true,
		},
		{
			name:           "leading slash stripped with nested base dir",
			bucket:         "mybucket",
			baseDir:        "/images",
			image:          "/foo/bar",
			expectedBucket: "mybucket",
			expectedPath:   "images/foo/bar",
			expectedOk:     true,
		},
		{
			name:           "leading slash stripped with base uri and root base dir",
			bucket:         "mybucket",
			baseDir:        "/",
			baseURI:        "/api",
			image:          "/api/foo/bar",
			expectedBucket: "mybucket",
			expectedPath:   "foo/bar",
			expectedOk:     true,
		},
		{
			name:           "no leading slash when base dir doesn't start with slash",
			bucket:         "mybucket",
			baseDir:        "images",
			image:          "/foo/bar",
			expectedBucket: "mybucket",
			expectedPath:   "images/foo/bar",
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

func TestWithEndpoint(t *testing.T) {
	tests := []struct {
		name             string
		endpoint         string
		expectedEndpoint string
	}{
		{
			name:             "valid endpoint",
			endpoint:         "https://s3.example.com",
			expectedEndpoint: "https://s3.example.com",
		},
		{
			name:             "valid endpoint with port",
			endpoint:         "http://localhost:9000",
			expectedEndpoint: "http://localhost:9000",
		},
		{
			name:             "empty endpoint",
			endpoint:         "",
			expectedEndpoint: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := aws.Config{
				Region: "us-east-1",
			}
			s := New(cfg, "test-bucket", WithEndpoint(tt.endpoint))
			assert.Equal(t, tt.expectedEndpoint, s.Endpoint)
		})
	}
}

func TestWithForcePathStyle(t *testing.T) {
	tests := []struct {
		name                   string
		forcePathStyle         bool
		expectedForcePathStyle bool
	}{
		{
			name:                   "force path style true",
			forcePathStyle:         true,
			expectedForcePathStyle: true,
		},
		{
			name:                   "force path style false",
			forcePathStyle:         false,
			expectedForcePathStyle: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := aws.Config{
				Region: "us-east-1",
			}
			s := New(cfg, "test-bucket", WithForcePathStyle(tt.forcePathStyle))
			assert.Equal(t, tt.expectedForcePathStyle, s.ForcePathStyle)
		})
	}
}

func TestEndpointAndForcePathStyleIntegration(t *testing.T) {
	ts := fakeS3Server()
	defer ts.Close()

	tests := []struct {
		name           string
		endpoint       string
		forcePathStyle bool
		bucketName     string
	}{
		{
			name:           "custom endpoint with force path style",
			endpoint:       ts.URL,
			forcePathStyle: true,
			bucketName:     "test-force-path",
		},
		{
			name:           "custom endpoint without force path style",
			endpoint:       ts.URL,
			forcePathStyle: false,
			bucketName:     "test-no-force-path",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := aws.Config{
				Region:      "eu-central-1",
				Credentials: credentials.NewStaticCredentialsProvider("test", "test", ""),
			}

			// Create S3Storage with custom endpoint and force path style options
			s := New(cfg, tt.bucketName, WithEndpoint(tt.endpoint), WithForcePathStyle(tt.forcePathStyle))

			// Verify the options are set correctly
			assert.Equal(t, tt.endpoint, s.Endpoint)
			assert.Equal(t, tt.forcePathStyle, s.ForcePathStyle)

			// Verify the S3 client is created (this tests that the options don't break client creation)
			assert.NotNil(t, s.Client)

			// Test basic functionality with the configured client
			ctx := context.Background()

			// Create bucket for this test
			_, err := s.Client.CreateBucket(ctx, &s3.CreateBucketInput{
				Bucket: aws.String(tt.bucketName),
			})
			require.NoError(t, err)

			// Test basic CRUD operations work with the custom configuration
			blob := imagor.NewBlobFromBytes([]byte("test-data"))
			err = s.Put(ctx, "/test-key", blob)
			require.NoError(t, err)

			// Verify we can retrieve the data
			r := (&http.Request{}).WithContext(ctx)
			retrievedBlob, err := s.Get(r, "/test-key")
			require.NoError(t, err)

			data, err := retrievedBlob.ReadAll()
			require.NoError(t, err)
			assert.Equal(t, "test-data", string(data))

			// Clean up
			err = s.Delete(ctx, "/test-key")
			require.NoError(t, err)
		})
	}
}

func TestWithEndpointEmptyString(t *testing.T) {
	cfg := aws.Config{
		Region: "us-east-1",
	}
	s := New(cfg, "test-bucket")
	originalEndpoint := s.Endpoint
	WithEndpoint("")(s)
	assert.Equal(t, originalEndpoint, s.Endpoint)
}

func TestWithForcePathStyleDefault(t *testing.T) {
	cfg := aws.Config{
		Region: "us-east-1",
	}
	s := New(cfg, "test-bucket")
	assert.False(t, s.ForcePathStyle) // Default should be false
	s2 := New(cfg, "test-bucket", WithForcePathStyle(true))
	assert.True(t, s2.ForcePathStyle)
}

func TestLocalstackCompatibility(t *testing.T) {
	cfg := aws.Config{
		Region: "us-east-1",
	}

	tests := []struct {
		name    string
		bucket  string
		baseDir string
		baseURI string
		image   string
	}{
		{
			name:    "root base dir with simple path",
			bucket:  "test-bucket",
			baseDir: "/",
			image:   "/my-image.jpg",
		},
		{
			name:    "nested base dir with simple path",
			bucket:  "test-bucket",
			baseDir: "/images",
			image:   "/my-image.jpg",
		},
		{
			name:    "root base dir with base URI",
			bucket:  "test-bucket",
			baseDir: "/",
			baseURI: "/api/v1",
			image:   "/api/v1/my-image.jpg",
		},
		{
			name:    "complex path with multiple segments",
			bucket:  "test-bucket",
			baseDir: "/storage/images",
			baseURI: "/uploads",
			image:   "/uploads/user/123/profile.jpg",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var opts []Option
			if tt.baseDir != "" {
				opts = append(opts, WithBaseDir(tt.baseDir))
			}
			if tt.baseURI != "" {
				opts = append(opts, WithPathPrefix(tt.baseURI))
			}

			s := New(cfg, tt.bucket, opts...)
			path, ok := s.Path(tt.image)

			// Verify the path is valid
			require.True(t, ok, "Path should be valid")

			// Critical test: ensure no leading slash for Localstack compatibility
			if len(path) > 0 {
				assert.NotEqual(t, '/', path[0], "Path should not start with leading slash for Localstack compatibility: %s", path)
			}

			// Verify path is not empty (unless it's a root case)
			assert.NotEmpty(t, path, "Path should not be empty")
		})
	}
}
