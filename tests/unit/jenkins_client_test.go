package unit

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"triggermesh/internal/config"
	"triggermesh/internal/engine/jenkins"
)

const crumbIssuerPath = "/crumbIssuer/api/json"

func TestNewClient(t *testing.T) {
	cfg := config.JenkinsConfig{
		URL:      "http://jenkins.example.com/",
		Username: "user",
		Token:    "token",
		Timeout:  10,
	}

	client := jenkins.NewClient(cfg)
	if client == nil {
		t.Fatal("Expected client to be created")
	}
}

func TestTriggerBuild_Success(t *testing.T) {
	// Mock Jenkins Server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify Auth
		auth := r.Header.Get("Authorization")
		expectedAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte("user:token"))
		if auth != expectedAuth {
			t.Errorf("Expected Auth header %q, got %q", expectedAuth, auth)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		if r.URL.Path == crumbIssuerPath {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"crumb":"test-crumb","crumbRequestField":"Jenkins-Crumb"}`))
			return
		}

		if r.URL.Path == "/job/test-job/build" {
			if r.Method != "POST" {
				t.Errorf("Expected POST request, got %s", r.Method)
			}

			w.Header().Set("Location", "http://jenkins.example.com/job/test-job/100/")
			w.WriteHeader(http.StatusCreated)
			return
		}

		t.Errorf("Unexpected request: %s %s", r.Method, r.URL.Path)
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	cfg := config.JenkinsConfig{
		URL:      server.URL,
		Username: "user",
		Token:    "token",
		Timeout:  5,
	}
	client := jenkins.NewClient(cfg)
	trigger := jenkins.NewTrigger(client)

	// Test TriggerBuild without params
	result, err := trigger.TriggerBuild("test-job", nil)
	if err != nil {
		t.Fatalf("Failed to trigger build: %v", err)
	}

	if !result.Success {
		t.Error("Expected success result")
	}
	if result.BuildURL != server.URL+"/job/test-job/100/" {
		t.Errorf("Expected build URL %q, got %q", server.URL+"/job/test-job/100/", result.BuildURL)
	}
}

func TestTriggerBuild_WithParams(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == crumbIssuerPath {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"crumb":"test-crumb","crumbRequestField":"Jenkins-Crumb"}`))
			return
		}

		if r.URL.Path == "/job/test-job/buildWithParameters" {
			if err := r.ParseForm(); err != nil {
				t.Errorf("Failed to parse form: %v", err)
			}
			if r.FormValue("param1") != "value1" {
				t.Errorf("Expected param1=value1, got %s", r.FormValue("param1"))
			}

			w.Header().Set("Location", "http://jenkins.example.com/job/test-job/101/")
			w.WriteHeader(http.StatusCreated)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	cfg := config.JenkinsConfig{
		URL:      server.URL,
		Username: "user",
		Token:    "token",
		Timeout:  5,
	}
	client := jenkins.NewClient(cfg)
	trigger := jenkins.NewTrigger(client)

	result, err := trigger.TriggerBuild("test-job", map[string]string{"param1": "value1"})
	if err != nil {
		t.Fatalf("Failed to trigger build: %v", err)
	}
	if result.BuildID != "test-job/101" {
		t.Errorf("Expected build ID test-job/101, got %s", result.BuildID)
	}
}

func TestTriggerBuild_Failure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == crumbIssuerPath {
			// Mock crumb failure, client should proceed
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if strings.Contains(r.URL.Path, "build") {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Jenkins Error"))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	cfg := config.JenkinsConfig{
		URL:      server.URL,
		Username: "user",
		Token:    "token",
		Timeout:  5,
	}
	client := jenkins.NewClient(cfg)
	trigger := jenkins.NewTrigger(client)

	_, err := trigger.TriggerBuild("test-job", nil)
	if err == nil {
		t.Fatal("Expected error, got nil")
	}
	if !strings.Contains(err.Error(), "500") {
		t.Errorf("Expected 500 error, got %v", err)
	}
}

func TestGetBuildStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/job/test-job/123/api/json") {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"number":123,"url":"http://jenkins.example.com/job/test-job/123/"}`))
			return
		}
		if strings.HasPrefix(r.URL.Path, "/job/test-job/404/api/json") {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if strings.HasPrefix(r.URL.Path, "/job/test-job/500/api/json") {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if strings.HasPrefix(r.URL.Path, "/job/test-job/bad-json/api/json") {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`invalid-json`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	cfg := config.JenkinsConfig{
		URL:      server.URL,
		Username: "user",
		Token:    "token",
		Timeout:  5,
	}
	client := jenkins.NewClient(cfg)
	trigger := jenkins.NewTrigger(client)

	tests := []struct {
		name          string
		buildID       string
		expectSuccess bool
		expectMessage string
		expectError   bool
	}{
		{"Success", "test-job/123", true, "Retrieved build status", false},
		{"Not Found", "test-job/404", false, "Failed to get Jenkins build status", true},
		{"Server Error", "test-job/500", false, "Failed to get Jenkins build status", true},
		{"Empty ID", "", false, "Build ID cannot be empty", true},
		{"Invalid ID Format", "invalid-id", false, "Invalid build ID format", true},
		{"Bad JSON", "test-job/bad-json", true, "Retrieved build status", false}, // Returns success with basic info
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := trigger.GetBuildStatus(tt.buildID)
			if tt.expectError {
				if err == nil {
					t.Error("Expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got %v", err)
				}
			}

			if result.Success != tt.expectSuccess {
				t.Errorf("Expected success %v, got %v", tt.expectSuccess, result.Success)
			}
			if !strings.Contains(result.Message, tt.expectMessage) {
				t.Errorf("Expected message containing %q, got %q", tt.expectMessage, result.Message)
			}
		})
	}
}
