package s3storage

import (
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// Option S3Storage option
type Option func(h *S3Storage)

// WithBaseDir with base dir option
func WithBaseDir(baseDir string) Option {
	return func(s *S3Storage) {
		if baseDir != "" {
			baseDir = "/" + strings.Trim(baseDir, "/")
			if baseDir != "/" {
				baseDir += "/"
			}
			s.BaseDir = baseDir
		}
	}
}

// WithPathPrefix with path prefix option
func WithPathPrefix(prefix string) Option {
	return func(s *S3Storage) {
		if prefix != "" {
			prefix = "/" + strings.Trim(prefix, "/")
			if prefix != "/" {
				prefix += "/"
			}
			s.PathPrefix = prefix
		}
	}
}

var aclValuesMap = (func() map[string]bool {
	m := map[string]bool{}
	for _, acl := range types.ObjectCannedACL("").Values() {
		m[string(acl)] = true
	}
	return m
})()

// WithACL with ACL option
// https://docs.aws.amazon.com/AmazonS3/latest/userguide/acl-overview.html#canned-acl
func WithACL(acl string) Option {
	return func(h *S3Storage) {
		if aclValuesMap[acl] {
			h.ACL = acl
		}
	}
}

// WithSafeChars with safe chars option
func WithSafeChars(chars string) Option {
	return func(h *S3Storage) {
		if chars != "" {
			h.SafeChars = chars
		}
	}
}

// WithExpiration with modified time expiration option
func WithExpiration(exp time.Duration) Option {
	return func(h *S3Storage) {
		if exp > 0 {
			h.Expiration = exp
		}
	}
}

// WithStorageClass with storage class option
func WithStorageClass(storageClass string) Option {
	return func(h *S3Storage) {
		allowedStorageClasses := [6]string{"REDUCED_REDUNDANCY", "STANDARD_IA", "ONEZONE_IA",
			"INTELLIGENT_TIERING", "GLACIER", "DEEP_ARCHIVE"}
		h.StorageClass = "STANDARD"
		for _, allowedStorageClass := range allowedStorageClasses {
			if storageClass == allowedStorageClass {
				h.StorageClass = storageClass
				break
			}
		}
	}
}

// WithEndpoint with custom S3 endpoint option
func WithEndpoint(endpoint string) Option {
	return func(s *S3Storage) {
		if endpoint != "" {
			s.Endpoint = endpoint
		}
	}
}

// WithForcePathStyle with force path style option
func WithForcePathStyle(forcePathStyle bool) Option {
	return func(s *S3Storage) {
		s.ForcePathStyle = forcePathStyle
	}
}

// WithBucketRouter with bucket router option for prefix-based bucket selection
func WithBucketRouter(router BucketRouter) Option {
	return func(s *S3Storage) {
		s.BucketRouter = router
	}
}
