package unit

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"triggermesh/internal/api/middleware"
)

func TestLimitBodySize(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Try to read the entire body to trigger size limit
		// MaxBytesReader will return an error when reading beyond the limit
		body, err := io.ReadAll(r.Body)
		if err != nil {
			// MaxBytesReader returns http.MaxBytesError when limit is exceeded
			if err.Error() == "http: request body too large" {
				http.Error(w, "Request too large", http.StatusRequestEntityTooLarge)
				return
			}
		}
		_ = body // Use the body to avoid unused variable
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	tests := []struct {
		name           string
		bodySize       int
		expectedStatus int
	}{
		{
			name:           "Small body (1KB)",
			bodySize:       1024,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Large body (2MB) - should fail",
			bodySize:       2 * 1024 * 1024,
			expectedStatus: http.StatusRequestEntityTooLarge,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := bytes.Repeat([]byte("a"), tt.bodySize)
			req := httptest.NewRequest("POST", "/test", bytes.NewReader(body))
			rr := httptest.NewRecorder()

			// LimitBodySize now takes maxSize as parameter and returns a middleware function
			maxSize := int64(1024 * 1024) // 1MB limit for tests
			middleware.LimitBodySize(maxSize)(handler).ServeHTTP(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s", tt.expectedStatus, rr.Code, rr.Body.String())
			}
		})
	}
}

func TestRequestIDMiddleware(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := middleware.GetRequestID(r)
		if requestID == "" {
			t.Error("Request ID should be set")
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(requestID))
	})

	tests := []struct {
		name              string
		requestIDHeader   string
		shouldGenerateNew bool
	}{
		{
			name:              "No request ID header - should generate",
			requestIDHeader:   "",
			shouldGenerateNew: true,
		},
		{
			name:              "Request ID in header - should use it",
			requestIDHeader:   "custom-request-id",
			shouldGenerateNew: false,
		},
		{
			name:              "Empty request ID header - should generate",
			requestIDHeader:   "   ", // Whitespace only
			shouldGenerateNew: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			if tt.requestIDHeader != "" {
				req.Header.Set("X-Request-ID", tt.requestIDHeader)
			}
			rr := httptest.NewRecorder()

			middleware.RequestIDMiddleware(handler).ServeHTTP(rr, req)

			if rr.Code != http.StatusOK {
				t.Errorf("Expected status 200, got %d", rr.Code)
			}

			// Check response header
			responseID := rr.Header().Get("X-Request-ID")
			if responseID == "" {
				t.Error("Response should include X-Request-ID header")
			}

			// Check body contains request ID
			bodyID := rr.Body.String()
			if bodyID == "" {
				t.Error("Response body should contain request ID")
			}

			if tt.requestIDHeader != "" && tt.requestIDHeader != "   " && bodyID != tt.requestIDHeader {
				t.Errorf("Expected request ID %q, got %q", tt.requestIDHeader, bodyID)
			}
		})
	}
}

func TestGetRequestID(t *testing.T) {
	tests := []struct {
		name      string
		setupFunc func(*http.Request)
		expected  string
	}{
		{
			name: "Request ID in context",
			setupFunc: func(r *http.Request) {
				ctx := context.WithValue(r.Context(), middleware.RequestIDContextKey, "test-id-123")
				*r = *r.WithContext(ctx)
			},
			expected: "test-id-123",
		},
		{
			name: "No request ID in context",
			setupFunc: func(r *http.Request) {
				// Don't set request ID
			},
			expected: "",
		},
		{
			name: "Wrong type in context",
			setupFunc: func(r *http.Request) {
				ctx := context.WithValue(r.Context(), middleware.RequestIDContextKey, 123)
				*r = *r.WithContext(ctx)
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			tt.setupFunc(req)

			result := middleware.GetRequestID(req)
			if result != tt.expected {
				t.Errorf("Expected request ID %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestLimitBodySizeEdgeCases(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Request too large", http.StatusRequestEntityTooLarge)
			return
		}
		_ = body
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	tests := []struct {
		name           string
		bodySize       int
		maxSize        int64
		expectedStatus int
	}{
		{
			name:           "Body exactly at limit",
			bodySize:       1024,
			maxSize:        1024,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Body one byte over limit",
			bodySize:       1025,
			maxSize:        1024,
			expectedStatus: http.StatusRequestEntityTooLarge,
		},
		{
			name:           "Zero body size limit",
			bodySize:       1,
			maxSize:        0,
			expectedStatus: http.StatusRequestEntityTooLarge,
		},
		{
			name:           "Very small body with small limit",
			bodySize:       10,
			maxSize:        100,
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := bytes.Repeat([]byte("a"), tt.bodySize)
			req := httptest.NewRequest("POST", "/test", bytes.NewReader(body))
			rr := httptest.NewRecorder()

			middleware.LimitBodySize(tt.maxSize)(handler).ServeHTTP(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, rr.Code)
			}
		})
	}
}

func TestRequestIDMiddlewareContextPropagation(t *testing.T) {
	// Test that request ID is properly propagated through context
	var capturedRequestID string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedRequestID = middleware.GetRequestID(r)
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	middleware.RequestIDMiddleware(handler).ServeHTTP(rr, req)

	if capturedRequestID == "" {
		t.Error("Request ID should be captured from context")
	}

	if rr.Header().Get("X-Request-ID") != capturedRequestID {
		t.Error("Request ID in header should match context")
	}
}
