package imagor

import (
	"encoding/json"
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

func TestTemplateXSSPrevention(t *testing.T) {
	tests := []struct {
		name             string
		path             string
		expectedStatus   int
		shouldNotContain []string
	}{
		{
			name:           "script injection in path",
			path:           "/unsafe/<script>alert('xss')</script>/",
			expectedStatus: http.StatusOK,
			shouldNotContain: []string{
				"<script>alert('xss')</script>",
				"alert('xss')",
			},
		},
		{
			name:           "javascript protocol injection",
			path:           "/unsafe/javascript:alert('xss')/",
			expectedStatus: http.StatusOK,
			shouldNotContain: []string{
				"javascript:alert('xss')",
				"alert('xss')",
			},
		},
		{
			name:           "html entity injection",
			path:           "/unsafe/&lt;script&gt;alert('xss')&lt;/script&gt;/",
			expectedStatus: http.StatusOK,
			shouldNotContain: []string{
				"<script>alert('xss')</script>", // Should not contain unescaped script
				"javascript:alert",              // Should not contain executable javascript
			},
		},
		{
			name:           "svg injection attempt",
			path:           "/unsafe/<svg onload=alert('xss')>/",
			expectedStatus: http.StatusOK,
			shouldNotContain: []string{
				"<svg onload=alert('xss')>",
				"onload=alert('xss')",
			},
		},
		{
			name:           "img onerror injection",
			path:           "/unsafe/<img src=x onerror=alert('xss')>/",
			expectedStatus: http.StatusOK,
			shouldNotContain: []string{
				"<img src=x onerror=alert('xss')>",
				"onerror=alert('xss')",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			renderUploadForm(w, tt.path)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			body := w.Body.String()
			for _, forbidden := range tt.shouldNotContain {
				if strings.Contains(body, forbidden) {
					t.Errorf("Response body should not contain %q, but it did. Body: %s", forbidden, body)
				}
			}

			// Ensure the response is still functional
			if !strings.Contains(body, "Upload Endpoint") {
				t.Error("Response should still contain 'Upload Endpoint'")
			}
		})
	}
}

func TestTemplateInputValidation(t *testing.T) {
	tests := []struct {
		name           string
		path           string
		expectedStatus int
		description    string
	}{
		{
			name:           "null byte injection",
			path:           "/unsafe/test\x00malicious/",
			expectedStatus: http.StatusOK,
			description:    "Should handle null bytes safely",
		},
		{
			name:           "control characters",
			path:           "/unsafe/test\x01\x02\x03/",
			expectedStatus: http.StatusOK,
			description:    "Should handle control characters",
		},
		{
			name:           "unicode edge cases",
			path:           "/unsafe/test\u202e\u202d/",
			expectedStatus: http.StatusOK,
			description:    "Should handle unicode direction override",
		},
		{
			name:           "path traversal attempt",
			path:           "/unsafe/../../../etc/passwd/",
			expectedStatus: http.StatusOK,
			description:    "Should handle path traversal safely",
		},
		{
			name:           "extremely long path",
			path:           "/unsafe/" + strings.Repeat("a", 10000) + "/",
			expectedStatus: http.StatusOK,
			description:    "Should handle very long paths",
		},
		{
			name:           "json breaking characters",
			path:           "/unsafe/test\"\\'/",
			expectedStatus: http.StatusOK,
			description:    "Should handle JSON-breaking characters",
		},
		{
			name:           "newline injection",
			path:           "/unsafe/test\n\r/",
			expectedStatus: http.StatusOK,
			description:    "Should handle newline characters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()

			// Should not panic
			renderUploadForm(w, tt.path)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d for %s", tt.expectedStatus, w.Code, tt.description)
			}

			// Should still produce valid HTML
			body := w.Body.String()
			if !strings.Contains(body, "<!DOCTYPE html>") {
				t.Error("Response should contain valid HTML doctype")
			}

			// Should contain essential form elements
			if !strings.Contains(body, "<form") {
				t.Error("Response should contain form element")
			}
		})
	}
}

func TestTemplateJSONInjection(t *testing.T) {
	tests := []struct {
		name         string
		path         string
		shouldEscape []string
	}{
		{
			name: "double quote injection",
			path: "/unsafe/test\"injection/",
			shouldEscape: []string{
				"\"injection",
			},
		},
		{
			name: "backslash injection",
			path: "/unsafe/test\\injection/",
			shouldEscape: []string{
				"\\injection",
			},
		},
		{
			name: "json control characters",
			path: "/unsafe/test\b\f\n\r\t/",
			shouldEscape: []string{
				"\b", "\f", "\n", "\r", "\t",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			renderUploadForm(w, tt.path)

			body := w.Body.String()

			// Check that the JSON in the response is properly escaped
			// Look for the JSON section in the HTML
			if strings.Contains(body, "<pre>") {
				// Find JSON content between <pre> tags
				start := strings.Index(body, "<pre>")
				end := strings.Index(body[start:], "</pre>")
				if end != -1 {
					jsonContent := body[start+5 : start+end]

					// Verify it's valid JSON by attempting to parse
					var parsed interface{}
					if err := json.Unmarshal([]byte(jsonContent), &parsed); err != nil {
						t.Errorf("JSON in template should be valid, but got error: %v", err)
					}
				}
			}
		})
	}
}

func TestTemplateEdgeCases(t *testing.T) {
	t.Run("empty template data", func(t *testing.T) {
		w := httptest.NewRecorder()
		renderUploadForm(w, "")

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})

	t.Run("special characters in filenames", func(t *testing.T) {
		specialChars := []string{
			"/unsafe/test file with spaces.jpg/",
			"/unsafe/test-file_with.special@chars.jpg/",
			"/unsafe/测试文件.jpg/",
			"/unsafe/файл.jpg/",
			"/unsafe/🖼️.jpg/",
		}

		for _, path := range specialChars {
			t.Run("path_"+path, func(t *testing.T) {
				w := httptest.NewRecorder()
				renderUploadForm(w, path)

				if w.Code != http.StatusOK {
					t.Errorf("Expected status 200 for path %s, got %d", path, w.Code)
				}

				body := w.Body.String()
				if !strings.Contains(body, "Upload Endpoint") {
					t.Errorf("Response should contain 'Upload Endpoint' for path %s", path)
				}
			})
		}
	})

	t.Run("deeply nested filter parameters", func(t *testing.T) {
		// Test with nested watermark filters
		nestedPath := "/unsafe/filters:" +
			"watermark(test1.png,center,center,50):" +
			"watermark(test2.png,left,top,30):" +
			"watermark(test3.png,right,bottom,70):" +
			"fill(white):quality(90):format(jpeg)/" +
			"test.jpg"

		w := httptest.NewRecorder()
		renderUploadForm(w, nestedPath)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		body := w.Body.String()
		if !strings.Contains(body, "watermark") {
			t.Error("Response should contain watermark parameters")
		}
	})
}
