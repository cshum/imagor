package imagor

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
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
