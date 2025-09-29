package imagor

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRenderLandingPage(t *testing.T) {
	tests := []struct {
		name           string
		expectedStatus int
		expectedType   string
		expectedBody   []string
	}{
		{
			name:           "landing page renders successfully",
			expectedStatus: http.StatusOK,
			expectedType:   "text/html",
			expectedBody: []string{
				"<title>imagor",
				"<h1>imagor</h1>",
				Version,
				"imagor.net",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()

			renderLandingPage(w)

			// Check status code
			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			// Check content type
			contentType := w.Header().Get("Content-Type")
			if !strings.Contains(contentType, tt.expectedType) {
				t.Errorf("Expected content type to contain %s, got %s", tt.expectedType, contentType)
			}

			// Check body content
			body := w.Body.String()
			for _, expected := range tt.expectedBody {
				if !strings.Contains(body, expected) {
					t.Errorf("Expected body to contain %q, but it didn't. Body: %s", expected, body)
				}
			}
		})
	}
}

func TestRenderUploadForm(t *testing.T) {
	tests := []struct {
		name           string
		path           string
		expectedStatus int
		expectedType   string
		expectedBody   []string
	}{
		{
			name:           "upload form for root path",
			path:           "/",
			expectedStatus: http.StatusOK,
			expectedType:   "text/html",
			expectedBody: []string{
				"<title>imagor",
				"Upload Endpoint",
				Version,
				"<form method=\"POST\"",
				"enctype=\"multipart/form-data\"",
				"<input type=\"file\"",
				"name=\"image\"",
			},
		},
		{
			name:           "upload form for resize path",
			path:           "/unsafe/200x300/",
			expectedStatus: http.StatusOK,
			expectedType:   "text/html",
			expectedBody: []string{
				"<title>imagor",
				"Upload Endpoint",
				"/unsafe/200x300/",
				"<form method=\"POST\"",
			},
		},
		{
			name:           "upload form for filters path",
			path:           "/unsafe/filters:quality(80):format(webp)/",
			expectedStatus: http.StatusOK,
			expectedType:   "text/html",
			expectedBody: []string{
				"Upload Endpoint",
				"/unsafe/filters:quality(80):format(webp)/",
				"filters",
				"quality",
				"webp",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()

			renderUploadForm(w, tt.path)

			// Check status code
			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			// Check content type
			contentType := w.Header().Get("Content-Type")
			if !strings.Contains(contentType, tt.expectedType) {
				t.Errorf("Expected content type to contain %s, got %s", tt.expectedType, contentType)
			}

			// Check body content
			body := w.Body.String()
			for _, expected := range tt.expectedBody {
				if !strings.Contains(body, expected) {
					t.Errorf("Expected body to contain %q, but it didn't. Body: %s", expected, body)
				}
			}
		})
	}
}

func TestTemplateDataStructure(t *testing.T) {
	// Test TemplateData structure
	data := TemplateData{
		Version: "1.0.0",
		Path:    "/test/path",
		Params:  "test params",
	}

	if data.Version != "1.0.0" {
		t.Errorf("Expected Version to be '1.0.0', got %s", data.Version)
	}

	if data.Path != "/test/path" {
		t.Errorf("Expected Path to be '/test/path', got %s", data.Path)
	}

	if data.Params != "test params" {
		t.Errorf("Expected Params to be 'test params', got %v", data.Params)
	}
}

func TestRenderUploadFormWithComplexPath(t *testing.T) {
	// Test with a complex imagor path
	path := "/unsafe/fit-in/300x200/filters:quality(90):format(jpeg):fill(white)/smart/"

	w := httptest.NewRecorder()
	renderUploadForm(w, path)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	body := w.Body.String()

	// Should contain the path
	if !strings.Contains(body, path) {
		t.Error("Expected body to contain the path")
	}

	// Should contain some JSON structure
	if !strings.Contains(body, "{") || !strings.Contains(body, "}") {
		t.Error("Expected body to contain JSON structure")
	}
}

func TestRenderUploadFormErrorHandling(t *testing.T) {
	// Test with invalid path that might cause parsing issues
	paths := []string{
		"/invalid/path/with/special/chars",
		"/unsafe/invalid-dimensions/",
		"/unsafe/filters:invalid()/",
		"",
	}

	for _, path := range paths {
		t.Run("path_"+path, func(t *testing.T) {
			w := httptest.NewRecorder()

			// Should not panic
			renderUploadForm(w, path)

			// Should return some response
			if w.Code == 0 {
				t.Error("Expected some HTTP status code")
			}

			// Should have some content
			if w.Body.Len() == 0 {
				t.Error("Expected some response body")
			}
		})
	}
}

func TestTemplateInitialization(t *testing.T) {
	// Test that templates are properly initialized
	if landingTemplate == nil {
		t.Error("Expected landingTemplate to be initialized")
	}

	if uploadTemplate == nil {
		t.Error("Expected uploadTemplate to be initialized")
	}
}

func TestRenderUploadFormJSONParams(t *testing.T) {
	// Test that imagorpath.Parse results are properly marshaled to JSON
	path := "/unsafe/200x300/filters:quality(80)/"

	w := httptest.NewRecorder()
	renderUploadForm(w, path)

	body := w.Body.String()

	// Should contain valid JSON structure
	if !strings.Contains(body, "{") || !strings.Contains(body, "}") {
		t.Error("Expected body to contain JSON structure")
	}

	// Should contain some path information
	if !strings.Contains(body, path) {
		t.Error("Expected body to contain the path")
	}
}

func TestRenderLandingPageContent(t *testing.T) {
	w := httptest.NewRecorder()
	renderLandingPage(w)

	body := w.Body.String()

	// Check for essential landing page elements
	essentialElements := []string{
		"<!DOCTYPE html>",
		"<html",
		"<head>",
		"<body>",
		"imagor",
		Version,
	}

	for _, element := range essentialElements {
		if !strings.Contains(body, element) {
			t.Errorf("Expected landing page to contain %q", element)
		}
	}
}

func TestRenderUploadFormResponseHeaders(t *testing.T) {
	w := httptest.NewRecorder()
	renderUploadForm(w, "/unsafe/200x200/")

	// Check Content-Type header
	contentType := w.Header().Get("Content-Type")
	if contentType != "text/html" {
		t.Errorf("Expected Content-Type to be 'text/html', got %s", contentType)
	}
}

func TestRenderLandingPageResponseHeaders(t *testing.T) {
	w := httptest.NewRecorder()
	renderLandingPage(w)

	// Check Content-Type header
	contentType := w.Header().Get("Content-Type")
	if contentType != "text/html" {
		t.Errorf("Expected Content-Type to be 'text/html', got %s", contentType)
	}
}
