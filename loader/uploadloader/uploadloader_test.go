package uploadloader

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/cshum/imagor"
)

func TestUploadLoader_New(t *testing.T) {
	loader := New()
	if loader.MaxAllowedSize != 32<<20 {
		t.Errorf("expected default MaxAllowedSize to be 32MB, got %d", loader.MaxAllowedSize)
	}
	if loader.Accept != "image/*" {
		t.Errorf("expected default Accept to be 'image/*', got %s", loader.Accept)
	}
	if loader.FormFieldName != "image" {
		t.Errorf("expected default FormFieldName to be 'image', got %s", loader.FormFieldName)
	}
}

func TestUploadLoader_NewWithOptions(t *testing.T) {
	loader := New(
		WithMaxAllowedSize(10<<20),
		WithAccept("image/jpeg,image/png"),
		WithFormFieldName("file"),
	)
	if loader.MaxAllowedSize != 10<<20 {
		t.Errorf("expected MaxAllowedSize to be 10MB, got %d", loader.MaxAllowedSize)
	}
	if loader.Accept != "image/jpeg,image/png" {
		t.Errorf("expected Accept to be 'image/jpeg,image/png', got %s", loader.Accept)
	}
	if loader.FormFieldName != "file" {
		t.Errorf("expected FormFieldName to be 'file', got %s", loader.FormFieldName)
	}
}

func TestUploadLoader_Get_NonPOST(t *testing.T) {
	loader := New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	
	blob, err := loader.Get(req, "")
	if blob != nil {
		t.Error("expected nil blob for non-POST request")
	}
	if err != imagor.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestUploadLoader_Get_MissingContentType(t *testing.T) {
	loader := New()
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("test"))
	
	blob, err := loader.Get(req, "")
	if blob != nil {
		t.Error("expected nil blob for missing Content-Type")
	}
	if err == nil || !strings.Contains(err.Error(), "missing Content-Type header") {
		t.Errorf("expected missing Content-Type error, got %v", err)
	}
}

func TestUploadLoader_RawUpload_Success(t *testing.T) {
	loader := New()
	imageData := []byte("fake-jpeg-data")
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(imageData))
	req.Header.Set("Content-Type", "image/jpeg")
	req.ContentLength = int64(len(imageData))
	
	blob, err := loader.Get(req, "")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if blob == nil {
		t.Fatal("expected blob, got nil")
	}
	
	// Read blob data
	reader, size, err := blob.NewReader()
	if err != nil {
		t.Errorf("unexpected error reading blob: %v", err)
	}
	defer reader.Close()
	
	data, err := io.ReadAll(reader)
	if err != nil {
		t.Errorf("unexpected error reading data: %v", err)
	}
	
	if !bytes.Equal(data, imageData) {
		t.Errorf("expected data %v, got %v", imageData, data)
	}
	if size != int64(len(imageData)) {
		t.Errorf("expected size %d, got %d", len(imageData), size)
	}
	if blob.ContentType() != "image/jpeg" {
		t.Errorf("expected content type 'image/jpeg', got %s", blob.ContentType())
	}
}

func TestUploadLoader_RawUpload_UnsupportedFormat(t *testing.T) {
	loader := New(WithAccept("image/jpeg"))
	imageData := []byte("fake-png-data")
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(imageData))
	req.Header.Set("Content-Type", "image/png")
	
	blob, err := loader.Get(req, "")
	if blob != nil {
		t.Error("expected nil blob for unsupported format")
	}
	if err != imagor.ErrUnsupportedFormat {
		t.Errorf("expected ErrUnsupportedFormat, got %v", err)
	}
}

func TestUploadLoader_RawUpload_SizeExceeded(t *testing.T) {
	loader := New(WithMaxAllowedSize(10))
	imageData := []byte("this-is-longer-than-10-bytes")
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(imageData))
	req.Header.Set("Content-Type", "image/jpeg")
	req.ContentLength = int64(len(imageData))
	
	blob, err := loader.Get(req, "")
	if blob != nil {
		t.Error("expected nil blob for size exceeded")
	}
	if err != imagor.ErrMaxSizeExceeded {
		t.Errorf("expected ErrMaxSizeExceeded, got %v", err)
	}
}

func TestUploadLoader_MultipartUpload_Success(t *testing.T) {
	loader := New()
	
	// Create multipart form
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	
	// Create form file with proper content type header
	h := make(map[string][]string)
	h["Content-Type"] = []string{"image/jpeg"}
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
	
	blob, err := loader.Get(req, "")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if blob == nil {
		t.Fatal("expected blob, got nil")
	}
	
	// Read blob data
	reader, _, err := blob.NewReader()
	if err != nil {
		t.Errorf("unexpected error reading blob: %v", err)
	}
	defer reader.Close()
	
	data, err := io.ReadAll(reader)
	if err != nil {
		t.Errorf("unexpected error reading data: %v", err)
	}
	
	if !bytes.Equal(data, imageData) {
		t.Errorf("expected data %v, got %v", imageData, data)
	}
}

func TestUploadLoader_MultipartUpload_MissingField(t *testing.T) {
	loader := New()
	
	// Create multipart form with wrong field name
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	
	part, err := writer.CreateFormFile("wrong_field", "test.jpg")
	if err != nil {
		t.Fatal(err)
	}
	
	_, err = part.Write([]byte("fake-jpeg-data"))
	if err != nil {
		t.Fatal(err)
	}
	
	err = writer.Close()
	if err != nil {
		t.Fatal(err)
	}
	
	req := httptest.NewRequest(http.MethodPost, "/", &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	
	blob, err := loader.Get(req, "")
	if blob != nil {
		t.Error("expected nil blob for missing field")
	}
	if err == nil || !strings.Contains(err.Error(), "failed to get form file") {
		t.Errorf("expected form file error, got %v", err)
	}
}

func TestUploadLoader_ValidateContentType(t *testing.T) {
	tests := []struct {
		name        string
		accept      string
		contentType string
		expected    bool
	}{
		{"accept all", "*/*", "image/jpeg", true},
		{"exact match", "image/jpeg", "image/jpeg", true},
		{"wildcard match", "image/*", "image/png", true},
		{"wildcard no match", "image/*", "text/plain", false},
		{"no match", "image/jpeg", "image/png", false},
		{"empty content type", "image/*", "", false},
		{"multiple accepts match", "image/jpeg,image/png", "image/png", true},
		{"multiple accepts no match", "image/jpeg,image/png", "image/gif", false},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loader := New(WithAccept(tt.accept))
			result := loader.validateContentType(tt.contentType)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestParseContentType(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"image/jpeg", "image/jpeg"},
		{"image/jpeg; charset=utf-8", "image/jpeg"},
		{"  image/png  ", "image/png"},
		{"", ""},
		{"invalid", "invalid"},
	}
	
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := parseContentType(tt.input)
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}
