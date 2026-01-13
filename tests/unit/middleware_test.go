package unit

import (
	"bytes"
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

			if tt.requestIDHeader != "" && bodyID != tt.requestIDHeader {
				t.Errorf("Expected request ID %q, got %q", tt.requestIDHeader, bodyID)
			}
		})
	}
}
