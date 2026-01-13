package unit

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"triggermesh/internal/api/middleware"
	"triggermesh/internal/config"
)

func TestAuthMiddleware(t *testing.T) {
	// Create test API config
	apiConfig := config.APIConfig{
		Keys: []string{"valid-key-1", "valid-key-2"},
	}

	// Create middleware
	authMiddleware := middleware.NewAuthMiddleware(apiConfig)

	// Create a test handler
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiKey, ok := r.Context().Value("api_key").(string)
		if !ok {
			t.Error("API key not found in context")
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(apiKey))
	})

	// Test valid API key in Authorization header
	t.Run("Valid API key in Authorization header", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Authorization", "Bearer valid-key-1")
		rr := httptest.NewRecorder()

		handler := authMiddleware.Middleware(testHandler)
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rr.Code)
		}
		if rr.Body.String() != "valid-key-1" {
			t.Errorf("Expected API key in response, got %s", rr.Body.String())
		}
	})

	// Test valid API key without Bearer prefix
	t.Run("Valid API key without Bearer prefix", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Authorization", "valid-key-2")
		rr := httptest.NewRecorder()

		handler := authMiddleware.Middleware(testHandler)
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rr.Code)
		}
	})

	// Test valid API key in query parameter
	t.Run("Valid API key in query parameter", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test?api_key=valid-key-1", nil)
		rr := httptest.NewRecorder()

		handler := authMiddleware.Middleware(testHandler)
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rr.Code)
		}
	})

	// Test invalid API key
	t.Run("Invalid API key", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Authorization", "Bearer invalid-key")
		rr := httptest.NewRecorder()

		handler := authMiddleware.Middleware(testHandler)
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusUnauthorized {
			t.Errorf("Expected status 401, got %d", rr.Code)
		}
	})

	// Test missing API key
	t.Run("Missing API key", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		rr := httptest.NewRecorder()

		handler := authMiddleware.Middleware(testHandler)
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusUnauthorized {
			t.Errorf("Expected status 401, got %d", rr.Code)
		}
	})
}

func TestValidateAPIKey(t *testing.T) {
	apiConfig := config.APIConfig{
		Keys: []string{"test-key-1", "test-key-2"},
	}

	authMiddleware := middleware.NewAuthMiddleware(apiConfig)

	tests := []struct {
		name     string
		apiKey   string
		expected bool
	}{
		{"Valid key 1", "test-key-1", true},
		{"Valid key 2", "test-key-2", true},
		{"Invalid key", "invalid-key", false},
		{"Empty key", "", false},
		{"Key with Bearer prefix", "Bearer test-key-1", true},
		{"Key with spaces", "  test-key-1  ", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := authMiddleware.ValidateAPIKey(tt.apiKey)
			if result != tt.expected {
				t.Errorf("ValidateAPIKey(%q) = %v, expected %v", tt.apiKey, result, tt.expected)
			}
		})
	}
}

func TestGetAPIKey(t *testing.T) {
	tests := []struct {
		name           string
		authHeader     string
		queryParam     string
		expectedAPIKey string
	}{
		{"From Authorization header", "Bearer test-key", "", "test-key"},
		{"From Authorization header without Bearer", "test-key", "", "test-key"},
		{"From query parameter", "", "test-key", "test-key"},
		{"Authorization takes precedence", "Bearer header-key", "query-key", "header-key"},
		{"No API key", "", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			if tt.queryParam != "" {
				req.URL.RawQuery = "api_key=" + tt.queryParam
			}

			apiKey := middleware.GetAPIKey(req)
			if apiKey != tt.expectedAPIKey {
				t.Errorf("GetAPIKey() = %q, expected %q", apiKey, tt.expectedAPIKey)
			}
		})
	}
}
