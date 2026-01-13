package unit

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"triggermesh/internal/api"
	"triggermesh/internal/config"
	"triggermesh/internal/engine/jenkins"
	"triggermesh/internal/storage"
)

// setupTestRouter creates a test router with a temporary database and returns cleanup function
func setupTestRouter(t *testing.T, cfg config.Config) (*api.Router, func()) {
	tmpFile, err := os.CreateTemp("", "test-router-*.db")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tmpFile.Close()

	if err := storage.Init(tmpFile.Name()); err != nil {
		t.Fatalf("Failed to init storage: %v", err)
	}

	jenkinsClient := jenkins.NewClient(cfg.Jenkins)
	jenkinsEngine := jenkins.NewTrigger(jenkinsClient)
	router := api.NewRouter(cfg, jenkinsEngine)

	cleanup := func() {
		storage.Close()
		os.Remove(tmpFile.Name())
	}

	return router, cleanup
}

// defaultTestConfig returns a default test configuration
func defaultTestConfig() config.Config {
	return config.Config{
		Server: config.ServerConfig{
			Port: 8080,
			Host: "0.0.0.0",
		},
		Jenkins: config.JenkinsConfig{
			URL:   "https://test-jenkins.example.com",
			Token: "test-token",
		},
		API: config.APIConfig{
			Keys: []string{"test-key"},
		},
	}
}

func TestHealthCheck(t *testing.T) {
	// Setup storage for health check
	tmpFile, err := os.CreateTemp("", "test-health-*.db")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	if err := storage.Init(tmpFile.Name()); err != nil {
		t.Fatalf("Failed to init storage: %v", err)
	}
	defer storage.Close()

	// Create router
	cfg := config.Config{
		Server: config.ServerConfig{
			Port: 8080,
			Host: "0.0.0.0",
		},
		Jenkins: config.JenkinsConfig{
			URL:   "https://test-jenkins.example.com",
			Token: "test-token",
		},
		API: config.APIConfig{
			Keys: []string{"test-key"},
		},
	}

	jenkinsClient := jenkins.NewClient(cfg.Jenkins)
	jenkinsEngine := jenkins.NewTrigger(jenkinsClient)
	router := api.NewRouter(cfg, jenkinsEngine)

	t.Run("Health check returns healthy", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/health", nil)
		rr := httptest.NewRecorder()

		router.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rr.Code)
		}

		var healthResp map[string]interface{}
		if err := json.Unmarshal(rr.Body.Bytes(), &healthResp); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		if healthResp["status"] != "healthy" {
			t.Errorf("Expected status 'healthy', got %v", healthResp["status"])
		}
	})
}

func TestCORS(t *testing.T) {
	cfg := config.Config{
		Server: config.ServerConfig{
			Port:           8080,
			Host:           "0.0.0.0",
			AllowedOrigins: []string{"https://example.com"},
		},
		Jenkins: config.JenkinsConfig{
			URL:   "https://test-jenkins.example.com",
			Token: "test-token",
		},
		API: config.APIConfig{
			Keys: []string{"test-key"},
		},
	}

	jenkinsClient := jenkins.NewClient(cfg.Jenkins)
	jenkinsEngine := jenkins.NewTrigger(jenkinsClient)
	router := api.NewRouter(cfg, jenkinsEngine)

	tests := []struct {
		name           string
		origin         string
		expectedOrigin string
	}{
		{
			name:           "Allowed origin",
			origin:         "https://example.com",
			expectedOrigin: "https://example.com",
		},
		{
			name:           "Disallowed origin",
			origin:         "https://evil.com",
			expectedOrigin: "", // Should not be set
		},
		{
			name:           "No origin header",
			origin:         "",
			expectedOrigin: "", // Should not be set
		},
		{
			name:           "Invalid origin format (no scheme)",
			origin:         "example.com",
			expectedOrigin: "", // Should not be set for invalid format
		},
		{
			name:           "Invalid origin format (file://)",
			origin:         "file:///path/to/file",
			expectedOrigin: "", // Should not be set for invalid format
		},
		{
			name:           "HTTP origin (allowed)",
			origin:         "http://example.com",
			expectedOrigin: "", // HTTP origin not in allowed list (only https://example.com)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/health", nil)
			if tt.origin != "" {
				req.Header.Set("Origin", tt.origin)
			}
			rr := httptest.NewRecorder()

			router.ServeHTTP(rr, req)

			actualOrigin := rr.Header().Get("Access-Control-Allow-Origin")
			if actualOrigin != tt.expectedOrigin {
				t.Errorf("Expected origin %q, got %q", tt.expectedOrigin, actualOrigin)
			}
		})
	}

	// Test default (allow all when no origins configured)
	t.Run("Default allow all", func(t *testing.T) {
		cfgDefault := config.Config{
			Server: config.ServerConfig{
				Port:           8080,
				Host:           "0.0.0.0",
				AllowedOrigins: []string{}, // Empty means allow all
			},
			Jenkins: config.JenkinsConfig{
				URL:   "https://test-jenkins.example.com",
				Token: "test-token",
			},
			API: config.APIConfig{
				Keys: []string{"test-key"},
			},
		}

		jenkinsClient := jenkins.NewClient(cfgDefault.Jenkins)
		jenkinsEngine := jenkins.NewTrigger(jenkinsClient)
		routerDefault := api.NewRouter(cfgDefault, jenkinsEngine)

		req := httptest.NewRequest("GET", "/health", nil)
		req.Header.Set("Origin", "https://any-origin.com")
		rr := httptest.NewRecorder()

		routerDefault.ServeHTTP(rr, req)

		actualOrigin := rr.Header().Get("Access-Control-Allow-Origin")
		if actualOrigin != "*" {
			t.Errorf("Expected origin '*', got %q", actualOrigin)
		}
	})
}

func TestCORSPreflight(t *testing.T) {
	cfg := config.Config{
		Server: config.ServerConfig{
			Port:           8080,
			Host:           "0.0.0.0",
			AllowedOrigins: []string{"https://example.com"},
		},
		Jenkins: config.JenkinsConfig{
			URL:   "https://test-jenkins.example.com",
			Token: "test-token",
		},
		API: config.APIConfig{
			Keys: []string{"test-key"},
		},
	}

	jenkinsClient := jenkins.NewClient(cfg.Jenkins)
	jenkinsEngine := jenkins.NewTrigger(jenkinsClient)
	router := api.NewRouter(cfg, jenkinsEngine)

	// Test OPTIONS request (CORS preflight)
	req := httptest.NewRequest("OPTIONS", "/api/v1/trigger/jenkins", nil)
	req.Header.Set("Origin", "https://example.com")
	req.Header.Set("Access-Control-Request-Method", "POST")
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200 for OPTIONS request, got %d", rr.Code)
	}

	// Check CORS headers
	if rr.Header().Get("Access-Control-Allow-Origin") != "https://example.com" {
		t.Errorf("Expected CORS origin header to be set")
	}
	if rr.Header().Get("Access-Control-Allow-Methods") == "" {
		t.Errorf("Expected CORS methods header to be set")
	}
	if rr.Header().Get("Access-Control-Allow-Headers") == "" {
		t.Errorf("Expected CORS headers header to be set")
	}
}

func TestRouterMiddlewareChain(t *testing.T) {
	// Setup storage
	tmpFile, err := os.CreateTemp("", "test-router-*.db")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	if err := storage.Init(tmpFile.Name()); err != nil {
		t.Fatalf("Failed to init storage: %v", err)
	}
	defer storage.Close()

	cfg := config.Config{
		Server: config.ServerConfig{
			Port:        8080,
			Host:        "0.0.0.0",
			MaxBodySize: 1024 * 1024, // 1MB
		},
		Jenkins: config.JenkinsConfig{
			URL:   "https://test-jenkins.example.com",
			Token: "test-token",
		},
		API: config.APIConfig{
			Keys: []string{"test-key"},
		},
	}

	jenkinsClient := jenkins.NewClient(cfg.Jenkins)
	jenkinsEngine := jenkins.NewTrigger(jenkinsClient)
	router := api.NewRouter(cfg, jenkinsEngine)

	// Test that all middleware are applied (RequestID, BodySizeLimit, CORS)
	req := httptest.NewRequest("GET", "/health", nil)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	// Check RequestID header is set
	if rr.Header().Get("X-Request-ID") == "" {
		t.Error("Expected X-Request-ID header to be set")
	}

	// Check CORS headers are set
	if rr.Header().Get("Access-Control-Allow-Methods") == "" {
		t.Error("Expected CORS headers to be set")
	}
}

func TestRouterInvalidOriginFormat(t *testing.T) {
	// Setup storage for health check
	tmpFile, err := os.CreateTemp("", "test-invalid-origin-*.db")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	if err := storage.Init(tmpFile.Name()); err != nil {
		t.Fatalf("Failed to init storage: %v", err)
	}
	defer storage.Close()

	cfg := config.Config{
		Server: config.ServerConfig{
			Port:           8080,
			Host:           "0.0.0.0",
			AllowedOrigins: []string{"https://example.com"},
		},
		Database: config.DatabaseConfig{
			Path: tmpFile.Name(),
		},
		Jenkins: config.JenkinsConfig{
			URL:   "https://test-jenkins.example.com",
			Token: "test-token",
		},
		API: config.APIConfig{
			Keys: []string{"test-key"},
		},
	}

	jenkinsClient := jenkins.NewClient(cfg.Jenkins)
	jenkinsEngine := jenkins.NewTrigger(jenkinsClient)
	router := api.NewRouter(cfg, jenkinsEngine)

	// Test with invalid origin format (should not set CORS header but continue processing)
	req := httptest.NewRequest("GET", "/health", nil)
	req.Header.Set("Origin", "invalid-origin-format")
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	// Should still process the request successfully
	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}

	// Invalid origin should not set CORS header
	if rr.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Errorf("Expected no CORS header for invalid origin, got %q", rr.Header().Get("Access-Control-Allow-Origin"))
	}
}

func TestRootEndpoint(t *testing.T) {
	cfg := config.Config{
		Server: config.ServerConfig{
			Port: 8080,
			Host: "0.0.0.0",
		},
		Jenkins: config.JenkinsConfig{
			URL:   "https://test-jenkins.example.com",
			Token: "test-token",
		},
		API: config.APIConfig{
			Keys: []string{"test-key"},
		},
	}

	jenkinsClient := jenkins.NewClient(cfg.Jenkins)
	jenkinsEngine := jenkins.NewTrigger(jenkinsClient)
	router := api.NewRouter(cfg, jenkinsEngine)

	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}

	var rootResp map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&rootResp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if rootResp["message"] != "TriggerMesh API" {
		t.Errorf("Expected message 'TriggerMesh API', got %v", rootResp["message"])
	}
}

func TestHealthCheckUnhealthy(t *testing.T) {
	// Don't initialize storage to test unhealthy state
	cfg := config.Config{
		Server: config.ServerConfig{
			Port: 8080,
			Host: "0.0.0.0",
		},
		Jenkins: config.JenkinsConfig{
			URL:   "https://test-jenkins.example.com",
			Token: "test-token",
		},
		API: config.APIConfig{
			Keys: []string{"test-key"},
		},
	}

	jenkinsClient := jenkins.NewClient(cfg.Jenkins)
	jenkinsEngine := jenkins.NewTrigger(jenkinsClient)
	router := api.NewRouter(cfg, jenkinsEngine)

	req := httptest.NewRequest("GET", "/health", nil)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected status 503, got %d", rr.Code)
	}

	var healthResp map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&healthResp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if healthResp["status"] != "unhealthy" {
		t.Errorf("Expected status 'unhealthy', got %v", healthResp["status"])
	}
}

func TestChainMiddlewareMultiple(t *testing.T) {
	// Test chainMiddleware with multiple middlewares
	// This is tested indirectly through ServeHTTP, but we can add explicit tests
	tmpFile, err := os.CreateTemp("", "test-chain-*.db")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	if err := storage.Init(tmpFile.Name()); err != nil {
		t.Fatalf("Failed to init storage: %v", err)
	}
	defer storage.Close()

	cfg := config.Config{
		Server: config.ServerConfig{
			Port:        8080,
			Host:        "0.0.0.0",
			MaxBodySize: 1024, // Small size to test body limit
		},
		Jenkins: config.JenkinsConfig{
			URL:   "https://test-jenkins.example.com",
			Token: "test-token",
		},
		API: config.APIConfig{
			Keys: []string{"test-key"},
		},
	}

	jenkinsClient := jenkins.NewClient(cfg.Jenkins)
	jenkinsEngine := jenkins.NewTrigger(jenkinsClient)
	router := api.NewRouter(cfg, jenkinsEngine)

	// Test with different routes to exercise different paths
	routes := []string{"/", "/health", "/api/v1/trigger/jenkins", "/api/v1/audit", "/nonexistent"}

	for _, route := range routes {
		t.Run("Route_"+route, func(t *testing.T) {
			req := httptest.NewRequest("GET", route, nil)
			if route == "/api/v1/trigger/jenkins" || route == "/api/v1/audit" {
				req.Header.Set("Authorization", "Bearer test-key")
			}
			rr := httptest.NewRecorder()

			router.ServeHTTP(rr, req)

			// All routes should have request ID
			if rr.Header().Get("X-Request-ID") == "" {
				t.Errorf("Expected X-Request-ID header for route %s", route)
			}

			// All routes should have CORS headers
			if rr.Header().Get("Access-Control-Allow-Methods") == "" {
				t.Errorf("Expected CORS headers for route %s", route)
			}
		})
	}
}

func TestCORSWithMultipleOrigins(t *testing.T) {
	// Setup storage
	tmpFile, err := os.CreateTemp("", "test-cors-multi-*.db")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	if err := storage.Init(tmpFile.Name()); err != nil {
		t.Fatalf("Failed to init storage: %v", err)
	}
	defer storage.Close()

	cfg := config.Config{
		Server: config.ServerConfig{
			Port:           8080,
			Host:           "0.0.0.0",
			AllowedOrigins: []string{"https://example.com", "https://app.example.com", "http://localhost:3000"},
		},
		Jenkins: config.JenkinsConfig{
			URL:   "https://test-jenkins.example.com",
			Token: "test-token",
		},
		API: config.APIConfig{
			Keys: []string{"test-key"},
		},
	}

	jenkinsClient := jenkins.NewClient(cfg.Jenkins)
	jenkinsEngine := jenkins.NewTrigger(jenkinsClient)
	router := api.NewRouter(cfg, jenkinsEngine)

	tests := []struct {
		name           string
		origin         string
		expectedOrigin string
	}{
		{
			name:           "First allowed origin",
			origin:         "https://example.com",
			expectedOrigin: "https://example.com",
		},
		{
			name:           "Second allowed origin",
			origin:         "https://app.example.com",
			expectedOrigin: "https://app.example.com",
		},
		{
			name:           "Third allowed origin (HTTP)",
			origin:         "http://localhost:3000",
			expectedOrigin: "http://localhost:3000",
		},
		{
			name:           "Not in list",
			origin:         "https://evil.com",
			expectedOrigin: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/health", nil)
			req.Header.Set("Origin", tt.origin)
			rr := httptest.NewRecorder()

			router.ServeHTTP(rr, req)

			actualOrigin := rr.Header().Get("Access-Control-Allow-Origin")
			if actualOrigin != tt.expectedOrigin {
				t.Errorf("Expected origin %q, got %q", tt.expectedOrigin, actualOrigin)
			}
		})
	}
}

func TestRouterBodySizeLimit(t *testing.T) {
	// Setup storage
	tmpFile, err := os.CreateTemp("", "test-body-size-*.db")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	if err := storage.Init(tmpFile.Name()); err != nil {
		t.Fatalf("Failed to init storage: %v", err)
	}
	defer storage.Close()

	// Set a small body size limit
	cfg := config.Config{
		Server: config.ServerConfig{
			Port:        8080,
			Host:        "0.0.0.0",
			MaxBodySize: 1024, // 1KB limit
		},
		Jenkins: config.JenkinsConfig{
			URL:   "https://test-jenkins.example.com",
			Token: "test-token",
		},
		API: config.APIConfig{
			Keys: []string{"test-key"},
		},
	}

	jenkinsClient := jenkins.NewClient(cfg.Jenkins)
	jenkinsEngine := jenkins.NewTrigger(jenkinsClient)
	router := api.NewRouter(cfg, jenkinsEngine)

	// Test with body that exceeds limit
	largeBody := make([]byte, 2048) // 2KB, exceeds 1KB limit
	for i := range largeBody {
		largeBody[i] = 'a'
	}

	req := httptest.NewRequest("POST", "/api/v1/trigger/jenkins", bytes.NewReader(largeBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test-key")
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	// Should return 413 or 400 (request body too large)
	if rr.Code != http.StatusRequestEntityTooLarge && rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status 413 or 400 for large body, got %d", rr.Code)
	}
}

func TestRouterDifferentHTTPMethods(t *testing.T) {
	// Setup storage
	tmpFile, err := os.CreateTemp("", "test-methods-*.db")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	if err := storage.Init(tmpFile.Name()); err != nil {
		t.Fatalf("Failed to init storage: %v", err)
	}
	defer storage.Close()

	cfg := config.Config{
		Server: config.ServerConfig{
			Port: 8080,
			Host: "0.0.0.0",
		},
		Jenkins: config.JenkinsConfig{
			URL:   "https://test-jenkins.example.com",
			Token: "test-token",
		},
		API: config.APIConfig{
			Keys: []string{"test-key"},
		},
	}

	jenkinsClient := jenkins.NewClient(cfg.Jenkins)
	jenkinsEngine := jenkins.NewTrigger(jenkinsClient)
	router := api.NewRouter(cfg, jenkinsEngine)

	methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH", "HEAD"}

	for _, method := range methods {
		t.Run("Method_"+method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/health", nil)
			rr := httptest.NewRecorder()

			router.ServeHTTP(rr, req)

			// All methods should have request ID and CORS headers
			if rr.Header().Get("X-Request-ID") == "" {
				t.Errorf("Expected X-Request-ID header for %s method", method)
			}

			if rr.Header().Get("Access-Control-Allow-Methods") == "" {
				t.Errorf("Expected CORS headers for %s method", method)
			}
		})
	}
}

func TestRouterWithCustomMaxBodySize(t *testing.T) {
	// Test router with custom max body size configuration
	tmpFile, err := os.CreateTemp("", "test-custom-body-*.db")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	if err := storage.Init(tmpFile.Name()); err != nil {
		t.Fatalf("Failed to init storage: %v", err)
	}
	defer storage.Close()

	cfg := config.Config{
		Server: config.ServerConfig{
			Port:        8080,
			Host:        "0.0.0.0",
			MaxBodySize: 512, // 512 bytes
		},
		Jenkins: config.JenkinsConfig{
			URL:   "https://test-jenkins.example.com",
			Token: "test-token",
		},
		API: config.APIConfig{
			Keys: []string{"test-key"},
		},
	}

	jenkinsClient := jenkins.NewClient(cfg.Jenkins)
	jenkinsEngine := jenkins.NewTrigger(jenkinsClient)
	router := api.NewRouter(cfg, jenkinsEngine)

	// Test that the custom body size is applied
	req := httptest.NewRequest("GET", "/health", nil)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	// Should work fine for GET requests
	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}
}

func TestRouterAllRoutes(t *testing.T) {
	cfg := defaultTestConfig()
	router, cleanup := setupTestRouter(t, cfg)
	defer cleanup()

	// Test all registered routes
	routes := []struct {
		method string
		path   string
		auth   bool
	}{
		{"GET", "/", false},
		{"GET", "/health", false},
		{"POST", "/api/v1/trigger/jenkins", true},
		{"GET", "/api/v1/audit", true},
		{"GET", "/nonexistent", false},
	}

	for _, route := range routes {
		t.Run(route.method+"_"+route.path, func(t *testing.T) {
			req := httptest.NewRequest(route.method, route.path, nil)
			if route.auth {
				req.Header.Set("Authorization", "Bearer test-key")
			}
			rr := httptest.NewRecorder()

			router.ServeHTTP(rr, req)

			// All routes should have request ID
			if rr.Header().Get("X-Request-ID") == "" {
				t.Errorf("Expected X-Request-ID header for %s %s", route.method, route.path)
			}

			// All routes should have CORS headers
			if rr.Header().Get("Access-Control-Allow-Methods") == "" {
				t.Errorf("Expected CORS headers for %s %s", route.method, route.path)
			}
		})
	}
}

func TestRouterChainMiddlewareWithDifferentCounts(t *testing.T) {
	// Test chainMiddleware with different numbers of middlewares
	cfg := defaultTestConfig()
	router, cleanup := setupTestRouter(t, cfg)
	defer cleanup()

	// Test that chainMiddleware works with different route types
	testRoutes := []string{
		"/",
		"/health",
		"/api/v1/trigger/jenkins",
		"/api/v1/audit",
	}

	for _, route := range testRoutes {
		t.Run("Route_"+route, func(t *testing.T) {
			req := httptest.NewRequest("GET", route, nil)
			if route == "/api/v1/trigger/jenkins" || route == "/api/v1/audit" {
				req.Header.Set("Authorization", "Bearer test-key")
			}
			rr := httptest.NewRecorder()

			router.ServeHTTP(rr, req)

			// Verify middleware chain is applied
			if rr.Header().Get("X-Request-ID") == "" {
				t.Errorf("Request ID middleware not applied for %s", route)
			}
		})
	}
}

func TestNewRouterWithAllConfigOptions(t *testing.T) {
	// Test NewRouter with all configuration options set
	tmpFile, err := os.CreateTemp("", "test-router-config-*.db")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	if err := storage.Init(tmpFile.Name()); err != nil {
		t.Fatalf("Failed to init storage: %v", err)
	}
	defer storage.Close()

	cfg := config.Config{
		Server: config.ServerConfig{
			Port:           9090,
			Host:           "127.0.0.1",
			AllowedOrigins: []string{"https://example.com", "https://app.example.com"},
			MaxBodySize:    5 * 1024 * 1024, // 5MB
		},
		Database: config.DatabaseConfig{
			Path: tmpFile.Name(),
		},
		Jenkins: config.JenkinsConfig{
			URL:      "https://test-jenkins.example.com",
			Username: "test-user",
			Token:    "test-token",
			Timeout:  60,
		},
		API: config.APIConfig{
			Keys: []string{"key1", "key2", "key3"},
		},
	}

	jenkinsClient := jenkins.NewClient(cfg.Jenkins)
	jenkinsEngine := jenkins.NewTrigger(jenkinsClient)
	router := api.NewRouter(cfg, jenkinsEngine)

	// Verify router was created successfully
	if router == nil {
		t.Fatal("Router should not be nil")
	}

	// Test that router works with all routes
	req := httptest.NewRequest("GET", "/health", nil)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}
}

func TestRouterServeHTTPWithAllMiddleware(t *testing.T) {
	// Test that ServeHTTP properly chains all middleware
	cfg := defaultTestConfig()
	cfg.Server.MaxBodySize = 1024 * 1024
	router, cleanup := setupTestRouter(t, cfg)
	defer cleanup()

	// Test multiple requests to ensure middleware chain works consistently
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest("GET", "/health", nil)
		rr := httptest.NewRecorder()

		router.ServeHTTP(rr, req)

		// Verify all middleware are applied
		if rr.Header().Get("X-Request-ID") == "" {
			t.Errorf("Request %d: Expected X-Request-ID header", i)
		}
		if rr.Header().Get("Access-Control-Allow-Methods") == "" {
			t.Errorf("Request %d: Expected CORS headers", i)
		}
	}
}
