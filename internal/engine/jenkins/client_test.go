package jenkins

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"triggermesh/internal/config"
)

func TestDoRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/test-path" {
			if r.Method == "POST" {
				w.WriteHeader(http.StatusCreated)
				w.Write([]byte(`{"created":true}`))
				return
			}
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"key":"value"}`))
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
	client := NewClient(cfg)
	ctx := context.Background()

	// Test GET request
	resp, err := client.DoRequest(ctx, "GET", "/test-path", nil)
	if err != nil {
		t.Fatalf("Failed to do GET request: %v", err)
	}
	if !strings.Contains(string(resp), "value") {
		t.Errorf("Expected response containing 'value', got %s", string(resp))
	}

	// Test POST request with body
	body := map[string]string{"foo": "bar"}
	resp, err = client.DoRequest(ctx, "POST", "/test-path", body)
	if err != nil {
		t.Fatalf("Failed to do POST request: %v", err)
	}
	if !strings.Contains(string(resp), "created") {
		t.Errorf("Expected response containing 'created', got %s", string(resp))
	}
}
