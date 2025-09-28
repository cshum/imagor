package uploadloader

import (
	"fmt"
	"io"
	"mime"
	"net/http"
	"strings"

	"github.com/cshum/imagor"
)

// UploadLoader handles POST request uploads and implements imagor.Loader interface
type UploadLoader struct {
	// MaxAllowedSize maximum bytes allowed for uploaded image
	MaxAllowedSize int

	// Accept set accepted Content-Type for uploads
	Accept string

	// FormFieldName the form field name to extract file from multipart uploads
	FormFieldName string

	accepts []string
}

// New creates UploadLoader
func New(options ...Option) *UploadLoader {
	u := &UploadLoader{
		MaxAllowedSize: 32 << 20, // 32MB default
		Accept:         "image/*",
		FormFieldName:  "image",
	}

	for _, option := range options {
		option(u)
	}

	if u.Accept != "" {
		for _, seg := range strings.Split(u.Accept, ",") {
			if typ := parseContentType(seg); typ != "" {
				u.accepts = append(u.accepts, typ)
			}
		}
	}

	return u
}

// Get implements imagor.Loader interface for POST uploads
func (u *UploadLoader) Get(r *http.Request, key string) (*imagor.Blob, error) {
	// Only handle POST requests
	if r.Method != http.MethodPost {
		return nil, imagor.ErrNotFound
	}

	// For uploads, we ignore the key parameter since there's no source URL
	// The key is typically empty or a special identifier for POST uploads

	contentType := r.Header.Get("Content-Type")
	if contentType == "" {
		return nil, imagor.NewError("missing Content-Type header", http.StatusBadRequest)
	}

	// Check if it's multipart form data
	if strings.HasPrefix(contentType, "multipart/form-data") {
		return u.handleMultipartUpload(r)
	}

	// Handle raw body upload
	return u.handleRawUpload(r)
}

func (u *UploadLoader) handleMultipartUpload(r *http.Request) (*imagor.Blob, error) {
	// Parse multipart form with size limit
	err := r.ParseMultipartForm(int64(u.MaxAllowedSize))
	if err != nil {
		return nil, imagor.NewError(
			fmt.Sprintf("failed to parse multipart form: %v", err),
			http.StatusBadRequest)
	}

	file, header, err := r.FormFile(u.FormFieldName)
	if err != nil {
		return nil, imagor.NewError(
			fmt.Sprintf("failed to get form file '%s': %v", u.FormFieldName, err),
			http.StatusBadRequest)
	}

	// Check file size
	if header.Size > int64(u.MaxAllowedSize) {
		return nil, imagor.ErrMaxSizeExceeded
	}

	// Validate content type if specified
	if !u.validateContentType(header.Header.Get("Content-Type")) {
		return nil, imagor.ErrUnsupportedFormat
	}

	return u.createBlobFromReader(file, header.Size, header.Header.Get("Content-Type"))
}

func (u *UploadLoader) handleRawUpload(r *http.Request) (*imagor.Blob, error) {
	contentType := r.Header.Get("Content-Type")

	// Validate content type
	if !u.validateContentType(contentType) {
		return nil, imagor.ErrUnsupportedFormat
	}

	// Check content length
	contentLength := r.ContentLength
	if contentLength > int64(u.MaxAllowedSize) {
		return nil, imagor.ErrMaxSizeExceeded
	}

	// Limit reader to max allowed size as additional safety
	limitedReader := io.LimitReader(r.Body, int64(u.MaxAllowedSize)+1)

	return u.createBlobFromReader(limitedReader, contentLength, contentType)
}

func (u *UploadLoader) createBlobFromReader(reader io.Reader, size int64, contentType string) (*imagor.Blob, error) {
	blob := imagor.NewBlob(func() (io.ReadCloser, int64, error) {
		// For uploads, we need to read the data once and store it
		// since the request body can only be read once
		data, err := io.ReadAll(reader)
		if err != nil {
			return nil, 0, err
		}

		// Check if we exceeded the size limit during reading
		if len(data) > u.MaxAllowedSize {
			return nil, 0, imagor.ErrMaxSizeExceeded
		}

		actualSize := int64(len(data))
		return io.NopCloser(strings.NewReader(string(data))), actualSize, nil
	})

	if contentType != "" {
		blob.SetContentType(contentType)
	}

	return blob, nil
}

func (u *UploadLoader) validateContentType(contentType string) bool {
	if len(u.accepts) == 0 {
		return true // Accept all if no restrictions
	}

	if contentType == "" {
		return false
	}

	// Parse media type to ignore parameters
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return false
	}

	for _, accept := range u.accepts {
		if accept == "*/*" || accept == mediaType {
			return true
		}
		// Handle wildcard types like "image/*"
		if strings.HasSuffix(accept, "/*") {
			prefix := strings.TrimSuffix(accept, "/*")
			if strings.HasPrefix(mediaType, prefix+"/") {
				return true
			}
		}
	}

	return false
}

func parseContentType(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	mediaType, _, err := mime.ParseMediaType(s)
	if err != nil {
		return s // Return original string if parsing fails
	}
	return mediaType
}
