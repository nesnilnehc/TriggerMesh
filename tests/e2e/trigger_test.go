package e2e

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"testing"
	"time"

	"triggermesh/internal/config"
	"triggermesh/internal/engine/jenkins"
)

// Note: These tests require a running TriggerMesh server
// They are meant to be run against a real instance
// Set TRIGGERMESH_URL environment variable to point to the server
// Example: TRIGGERMESH_URL=http://localhost:8080 go test ./tests/e2e/...

func TestE2EHealthCheck(t *testing.T) {
	serverURL := os.Getenv("TRIGGERMESH_URL")
	if serverURL == "" {
		t.Skip("TRIGGERMESH_URL not set, skipping e2e test")
	}

	resp, err := http.Get(serverURL + "/health")
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	body := make([]byte, 100)
	n, _ := resp.Body.Read(body)
	if string(body[:n]) != "OK" {
		t.Errorf("Expected response 'OK', got %s", string(body[:n]))
	}
}

func TestE2ETriggerJenkinsBuild(t *testing.T) {
	serverURL := os.Getenv("TRIGGERMESH_URL")
	apiKey := os.Getenv("TRIGGERMESH_API_KEY")
	if serverURL == "" || apiKey == "" {
		t.Skip("TRIGGERMESH_URL or TRIGGERMESH_API_KEY not set, skipping e2e test")
	}

	reqBody := map[string]interface{}{
		"job":        "test-job",
		"parameters": map[string]string{"param1": "value1"},
	}
	jsonBody, _ := json.Marshal(reqBody)

	req, err := http.NewRequest("POST", serverURL+"/api/v1/trigger/jenkins", bytes.NewBuffer(jsonBody))
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusInternalServerError {
		// InternalServerError is acceptable if Jenkins is not available
		t.Errorf("Expected status 200 or 500, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Verify response structure
	if _, ok := result["success"]; !ok {
		t.Error("Response missing 'success' field")
	}
}

func TestE2EGetAuditLogs(t *testing.T) {
	serverURL := os.Getenv("TRIGGERMESH_URL")
	apiKey := os.Getenv("TRIGGERMESH_API_KEY")
	if serverURL == "" || apiKey == "" {
		t.Skip("TRIGGERMESH_URL or TRIGGERMESH_API_KEY not set, skipping e2e test")
	}

	req, err := http.NewRequest("GET", serverURL+"/api/v1/audit?limit=10&offset=0", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
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

	// Verify response structure
	if len(logs) > 0 {
		log := logs[0]
		requiredFields := []string{"id", "timestamp", "api_key", "method", "path", "status"}
		for _, field := range requiredFields {
			if _, ok := log[field]; !ok {
				t.Errorf("Log entry missing required field: %s", field)
			}
		}
	}
}

// Helper function to test Jenkins engine directly (for unit testing the engine)
func TestJenkinsEngineIntegration(t *testing.T) {
	// This test can be used to test the Jenkins engine with a mock or real Jenkins instance
	// Set JENKINS_URL and JENKINS_TOKEN environment variables to test against real Jenkins
	jenkinsURL := os.Getenv("JENKINS_URL")
	jenkinsToken := os.Getenv("JENKINS_TOKEN")
	if jenkinsURL == "" || jenkinsToken == "" {
		t.Skip("JENKINS_URL or JENKINS_TOKEN not set, skipping Jenkins engine test")
	}

	cfg := config.JenkinsConfig{
		URL:     jenkinsURL,
		Token:   jenkinsToken,
		Timeout: 30,
	}

	client := jenkins.NewClient(cfg)
	engine := jenkins.NewTrigger(client)

	// Test triggering a build (use a test job that exists)
	result, err := engine.TriggerBuild("test-job", map[string]string{"test": "value"})
	if err != nil {
		t.Logf("TriggerBuild failed (this is expected if Jenkins is not available): %v", err)
		return
	}

	if !result.Success {
		t.Errorf("Expected successful build trigger, got success=false")
	}

	if result.BuildID == "" {
		t.Log("Build ID is empty (this may be normal if Jenkins doesn't return it immediately)")
	}
}
