package imagor

import (
	"bytes"
	"context"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/cshum/imagor/imagorpath"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestImagor_PostUpload_UnsafeDisabled(t *testing.T) {
	app := New(
		WithUnsafe(false), // Unsafe mode disabled
	)

	imageData := []byte("fake-jpeg-data")
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(imageData))
	req.Header.Set("Content-Type", "image/jpeg")
	req.ContentLength = int64(len(imageData))

	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status %d, got %d", http.StatusMethodNotAllowed, w.Code)
	}
}

func TestImagor_PostUpload_UnsafeEnabled_NoLoaders(t *testing.T) {
	app := New(
		WithUnsafe(true), // Unsafe mode enabled but no upload loader
	)

	imageData := []byte("fake-jpeg-data")
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(imageData))
	req.Header.Set("Content-Type", "image/jpeg")
	req.ContentLength = int64(len(imageData))

	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)

	// Should return 405 since no UploadLoader is configured
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status %d, got %d", http.StatusMethodNotAllowed, w.Code)
	}
}

func TestImagor_PostUpload_WithProcessingParams(t *testing.T) {
	app := New(
		WithUnsafe(true), // Unsafe mode enabled
	)

	imageData := []byte("fake-jpeg-data")
	// POST to a path with processing parameters
	req := httptest.NewRequest(http.MethodPost, "/200x300/filters:quality(80)/", bytes.NewReader(imageData))
	req.Header.Set("Content-Type", "image/jpeg")
	req.ContentLength = int64(len(imageData))

	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)

	// Should return 405 since no UploadLoader is configured
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status %d, got %d", http.StatusMethodNotAllowed, w.Code)
	}
}

func TestImagor_PostUpload_NonPOSTMethod(t *testing.T) {
	app := New(
		WithUnsafe(true), // Unsafe mode enabled
	)

	imageData := []byte("fake-jpeg-data")
	req := httptest.NewRequest(http.MethodPut, "/", bytes.NewReader(imageData))
	req.Header.Set("Content-Type", "image/jpeg")

	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status %d, got %d", http.StatusMethodNotAllowed, w.Code)
	}
}

// Test POST method handling in ServeHTTP
func TestImagor_ServeHTTP_PostMethodHandling(t *testing.T) {
	app := New(WithUnsafe(true))

	// Test that POST requests are rejected when no UploadLoader is configured
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader([]byte("test")))
	req.Header.Set("Content-Type", "image/jpeg")

	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)

	// POST requests should be rejected with 405 when no UploadLoader is configured
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("POST requests should return 405 when no UploadLoader is configured, got %d", w.Code)
	}
}

// uploadLoaderFunc creates a mock upload loader for testing
type uploadLoaderFunc func(r *http.Request, key string) (*Blob, error)

func (f uploadLoaderFunc) Get(r *http.Request, key string) (*Blob, error) {
	return f(r, key)
}

// createMockUploadLoader creates a mock loader that handles POST uploads
func createMockUploadLoader() Loader {
	return uploadLoaderFunc(func(r *http.Request, key string) (*Blob, error) {
		// Only handle POST requests for uploads
		if r.Method != http.MethodPost {
			return nil, ErrNotFound
		}

		// For POST uploads, the key is typically empty (no source URL)
		// Read the uploaded data from request body
		contentType := r.Header.Get("Content-Type")
		if contentType == "" {
			return nil, NewError("missing Content-Type header", http.StatusBadRequest)
		}

		// Handle multipart form data
		if contentType != "" && len(contentType) > 19 && contentType[:19] == "multipart/form-data" {
			err := r.ParseMultipartForm(32 << 20) // 32MB limit
			if err != nil {
				return nil, NewError("failed to parse multipart form", http.StatusBadRequest)
			}

			file, header, err := r.FormFile("image")
			if err != nil {
				return nil, NewError("failed to get form file", http.StatusBadRequest)
			}
			defer file.Close()

			data := make([]byte, header.Size)
			_, err = file.Read(data)
			if err != nil {
				return nil, err
			}

			blob := NewBlobFromBytes(data)
			blob.SetContentType(header.Header.Get("Content-Type"))
			return blob, nil
		}

		// Handle raw body upload
		data := make([]byte, r.ContentLength)
		_, err := r.Body.Read(data)
		if err != nil && err.Error() != "EOF" {
			return nil, err
		}

		blob := NewBlobFromBytes(data)
		blob.SetContentType(contentType)
		return blob, nil
	})
}

func TestImagor_PostUpload_Success_RawUpload(t *testing.T) {
	app := New(
		WithUnsafe(true),
		WithEnablePostRequests(true),
		WithLoaders(createMockUploadLoader()),
	)

	imageData := []byte("fake-jpeg-data")
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(imageData))
	req.Header.Set("Content-Type", "image/jpeg")
	req.ContentLength = int64(len(imageData))

	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	// Verify response headers
	if w.Header().Get("Content-Type") != "image/jpeg" {
		t.Errorf("expected Content-Type 'image/jpeg', got '%s'", w.Header().Get("Content-Type"))
	}

	// Verify response body contains the uploaded data
	responseData := w.Body.Bytes()
	if !bytes.Equal(responseData, imageData) {
		t.Errorf("expected response data %v, got %v", imageData, responseData)
	}
}

func TestImagor_PostUpload_Success_MultipartUpload(t *testing.T) {
	app := New(
		WithUnsafe(true),
		WithEnablePostRequests(true),
		WithLoaders(createMockUploadLoader()),
	)

	// Create multipart form
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// Create form file with proper content type header
	part, err := writer.CreatePart(map[string][]string{
		"Content-Disposition": {`form-data; name="image"; filename="test.jpg"`},
		"Content-Type":        {"image/jpeg"},
	})
	if err != nil {
		t.Fatal(err)
	}

	imageData := []byte("fake-jpeg-data")
	_, err = part.Write(imageData)
	if err != nil {
		t.Fatal(err)
	}

	err = writer.Close()
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, "/", &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	// Verify response headers
	if w.Header().Get("Content-Type") != "image/jpeg" {
		t.Errorf("expected Content-Type 'image/jpeg', got '%s'", w.Header().Get("Content-Type"))
	}

	// Verify response body contains the uploaded data
	responseData := w.Body.Bytes()
	if !bytes.Equal(responseData, imageData) {
		t.Errorf("expected response data %v, got %v", imageData, responseData)
	}
}

func TestImagor_PostUpload_Success_WithProcessingParams(t *testing.T) {
	app := New(
		WithUnsafe(true),
		WithEnablePostRequests(true),
		WithLoaders(createMockUploadLoader()),
	)

	imageData := []byte("fake-jpeg-data")
	// POST to a path with processing parameters
	req := httptest.NewRequest(http.MethodPost, "/200x300/filters:quality(80)/", bytes.NewReader(imageData))
	req.Header.Set("Content-Type", "image/jpeg")
	req.ContentLength = int64(len(imageData))

	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	// Verify response headers
	if w.Header().Get("Content-Type") != "image/jpeg" {
		t.Errorf("expected Content-Type 'image/jpeg', got '%s'", w.Header().Get("Content-Type"))
	}

	// For processing params, the response data might be different due to processing
	// but we should still get a valid response
	responseData := w.Body.Bytes()
	if len(responseData) == 0 {
		t.Error("expected non-empty response data")
	}
}

func TestImagor_PostUpload_Success_DifferentContentTypes(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		data        []byte
	}{
		{"JPEG", "image/jpeg", []byte("fake-jpeg-data")},
		{"PNG", "image/png", []byte("fake-png-data")},
		{"GIF", "image/gif", []byte("fake-gif-data")},
		{"WebP", "image/webp", []byte("fake-webp-data")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := New(
				WithUnsafe(true),
				WithEnablePostRequests(true),
				WithLoaders(createMockUploadLoader()),
			)

			req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(tt.data))
			req.Header.Set("Content-Type", tt.contentType)
			req.ContentLength = int64(len(tt.data))

			w := httptest.NewRecorder()
			app.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
			}

			// Verify response headers
			if w.Header().Get("Content-Type") != tt.contentType {
				t.Errorf("expected Content-Type '%s', got '%s'", tt.contentType, w.Header().Get("Content-Type"))
			}

			// Verify response body contains the uploaded data
			responseData := w.Body.Bytes()
			if !bytes.Equal(responseData, tt.data) {
				t.Errorf("expected response data %v, got %v", tt.data, responseData)
			}
		})
	}
}

func TestImagor_PostUpload_Success_PostRequestsDisabled(t *testing.T) {
	app := New(
		WithUnsafe(true),
		WithEnablePostRequests(false),
		WithLoaders(createMockUploadLoader()),
	)

	imageData := []byte("fake-jpeg-data")
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(imageData))
	req.Header.Set("Content-Type", "image/jpeg")
	req.ContentLength = int64(len(imageData))

	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)

	// Should return 405 even with UploadLoader when EnablePostRequests is false
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status %d, got %d", http.StatusMethodNotAllowed, w.Code)
	}
}

func TestImagor_PostUpload_Success_CustomUploadLoader(t *testing.T) {
	// Test with custom upload loader configuration (simplified for testing)
	app := New(
		WithUnsafe(true),
		WithEnablePostRequests(true),
		WithLoaders(createMockUploadLoader()),
	)

	imageData := []byte("fake-jpeg-data")
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(imageData))
	req.Header.Set("Content-Type", "image/jpeg")
	req.ContentLength = int64(len(imageData))

	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	// Verify response headers
	if w.Header().Get("Content-Type") != "image/jpeg" {
		t.Errorf("expected Content-Type 'image/jpeg', got '%s'", w.Header().Get("Content-Type"))
	}

	// Verify response body contains the uploaded data
	responseData := w.Body.Bytes()
	if !bytes.Equal(responseData, imageData) {
		t.Errorf("expected response data %v, got %v", imageData, responseData)
	}
}

func TestImagor_PostUpload_DoesNotEnableFanoutWithStorages(t *testing.T) {
	data := []byte("fake-jpeg-data")
	var calls int
	storage := &countingReaderStorage{}
	app := New(
		WithUnsafe(true),
		WithEnablePostRequests(true),
		WithLoaders(uploadLoaderFunc(func(r *http.Request, key string) (*Blob, error) {
			if r.Method != http.MethodPost {
				return nil, ErrNotFound
			}
			return NewBlob(func() (io.ReadCloser, int64, error) {
				calls++
				return io.NopCloser(bytes.NewReader(data)), int64(len(data)), nil
			}), nil
		})),
		WithStorages(storage),
		WithProcessors(processorFunc(func(ctx context.Context, blob *Blob, p imagorpath.Params, load LoadFunc) (*Blob, error) {
			assert.False(t, blob.fanout)
			buf, err := blob.ReadAll()
			require.NoError(t, err)
			assert.Equal(t, data, buf)
			return NewBlobFromBytes(buf), nil
		})),
	)

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(data))
	req.Header.Set("Content-Type", "image/jpeg")
	req.ContentLength = int64(len(data))

	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, string(data), w.Body.String())
	assert.Equal(t, 1, calls, "POST upload should reuse the sniffed source without fanout")
	assert.Equal(t, 0, storage.Reads(), "POST upload should not trigger source storage save")
}
