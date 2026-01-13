package unit

import (
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
