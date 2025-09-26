package s3storage

import (
	"context"
	"net/http"
	"strings"
)

// BucketMiddleware is a middleware that extracts the AWS-BUCKET header
// and adds it to the request context for use by S3Storage
func BucketMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		bucket := r.Header.Get("AWS-BUCKET")
		if bucket != "" {
			bucket = strings.TrimSpace(bucket)
			ctx := context.WithValue(r.Context(), "aws-bucket", bucket)
			r = r.WithContext(ctx)
		}
		next.ServeHTTP(w, r)
	})
}
