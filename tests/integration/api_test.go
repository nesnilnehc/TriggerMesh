package integration

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

// setupTestServer creates a test server with a temporary database
func setupTestServer(t *testing.T) (*httptest.Server, func()) {
	// Create a temporary database file
	tmpFile, err := os.CreateTemp("", "test-integration-*.db")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tmpFile.Close()

	// Initialize storage
	err = storage.Init(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to initialize storage: %v", err)
	}

	// Create a mock Jenkins server
	mockJenkins := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Mock Crumb Issuer
		if r.URL.Path == "/crumbIssuer/api/json" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"crumb": "test-crumb", "crumbRequestField": "Jenkins-Crumb"}`))
			return
		}
		// Mock Build Job
		if r.Method == "POST" {
			w.WriteHeader(http.StatusCreated)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))

	// Create test config
	cfg := config.Config{
		Server: config.ServerConfig{
			Port: 8080,
			Host: "0.0.0.0",
		},
		Database: config.DatabaseConfig{
			Path: tmpFile.Name(),
		},
		Jenkins: config.JenkinsConfig{
			URL:     mockJenkins.URL,
			Token:   "test-token",
			Timeout: 30,
		},
		API: config.APIConfig{
			Keys: []string{"test-api-key"},
		},
	}

	// Create Jenkins client and engine
	jenkinsClient := jenkins.NewClient(cfg.Jenkins)
	jenkinsEngine := jenkins.NewTrigger(jenkinsClient)

	// Create router
	router := api.NewRouter(cfg, jenkinsEngine)

	// Create test server
	server := httptest.NewServer(router)

	// Cleanup function
	cleanup := func() {
		server.Close()
		mockJenkins.Close()
		storage.Close()
		os.Remove(tmpFile.Name())
	}

	return server, cleanup
}

func TestHealthEndpoint(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	resp, err := http.Get(server.URL + "/health")
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}

func TestRootEndpoint(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	resp, err := http.Get(server.URL + "/")
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if result["message"] != "TriggerMesh API" {
		t.Errorf("Expected message 'TriggerMesh API', got %v", result["message"])
	}
}

func TestTriggerJenkinsWithoutAuth(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	reqBody := map[string]interface{}{
		"job": "test-job",
	}
	jsonBody, _ := json.Marshal(reqBody)

	req, err := http.NewRequest("POST", server.URL+"/api/v1/trigger/jenkins", bytes.NewBuffer(jsonBody))
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", resp.StatusCode)
	}
}

func TestTriggerJenkinsWithAuth(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	reqBody := map[string]interface{}{
		"job": "test-job",
	}
	jsonBody, _ := json.Marshal(reqBody)

	req, err := http.NewRequest("POST", server.URL+"/api/v1/trigger/jenkins", bytes.NewBuffer(jsonBody))
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test-api-key")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	// The request should succeed now that we have a mock Jenkins server
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}

func TestGetAuditLogs(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	// First, make a request that will create an audit log
	reqBody := map[string]interface{}{
		"job": "test-job",
	}
	jsonBody, _ := json.Marshal(reqBody)

	req, err := http.NewRequest("POST", server.URL+"/api/v1/trigger/jenkins", bytes.NewBuffer(jsonBody))
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test-api-key")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	resp.Body.Close()

	// Now get audit logs
	req, err = http.NewRequest("GET", server.URL+"/api/v1/audit?limit=10", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer test-api-key")

	resp, err = client.Do(req)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var logs []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&logs); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Should have at least one log entry
	if len(logs) == 0 {
		t.Error("Expected at least one audit log entry")
	}
}

func TestCORSHeaders(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	req, err := http.NewRequest("OPTIONS", server.URL+"/api/v1/trigger/jenkins", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	// Check CORS headers
	if resp.Header.Get("Access-Control-Allow-Origin") != "*" {
		t.Errorf("Expected CORS header Access-Control-Allow-Origin: *, got %s",
			resp.Header.Get("Access-Control-Allow-Origin"))
	}
}
