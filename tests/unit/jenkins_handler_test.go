package unit

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"triggermesh/internal/api/handlers"
	"triggermesh/internal/api/middleware"
	"triggermesh/internal/engine"
	"triggermesh/internal/storage"
)

// MockCIEngine is a mock implementation of engine.CIEngine
type MockCIEngine struct {
	TriggerBuildFunc   func(jobName string, params map[string]string) (*engine.BuildResult, error)
	GetBuildStatusFunc func(buildID string) (*engine.BuildResult, error)
}

func (m *MockCIEngine) TriggerBuild(jobName string, params map[string]string) (*engine.BuildResult, error) {
	if m.TriggerBuildFunc != nil {
		return m.TriggerBuildFunc(jobName, params)
	}
	return &engine.BuildResult{Success: true, Message: "Mock build triggered"}, nil
}

func (m *MockCIEngine) GetBuildStatus(buildID string) (*engine.BuildResult, error) {
	if m.GetBuildStatusFunc != nil {
		return m.GetBuildStatusFunc(buildID)
	}
	return &engine.BuildResult{Success: true, Message: "Mock build status"}, nil
}

func TestTriggerJenkinsBuild(t *testing.T) {
	// Setup storage
	tmpFile, err := os.CreateTemp("", "test-jenkins-handler-*.db")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if err := storage.Init(tmpFile.Name()); err != nil {
		t.Fatalf("Failed to init storage: %v", err)
	}
	defer storage.Close()

	tests := []struct {
		name           string
		requestBody    interface{}
		mockEngine     *MockCIEngine
		expectedStatus int
		expectedBody   string
	}{
		{
			name: "Success",
			requestBody: handlers.TriggerJenkinsBuildRequest{
				Job: "test-job",
				Parameters: map[string]string{
					"param1": "value1",
				},
			},
			mockEngine: &MockCIEngine{
				TriggerBuildFunc: func(jobName string, params map[string]string) (*engine.BuildResult, error) {
					if jobName != "test-job" {
						return nil, errors.New("unexpected job name")
					}
					if params["param1"] != "value1" {
						return nil, errors.New("unexpected params")
					}
					return &engine.BuildResult{
						Success:  true,
						BuildID:  "test-job/123",
						BuildURL: "http://jenkins/job/test-job/123",
						Message:  "Build triggered successfully",
					}, nil
				},
			},
			expectedStatus: http.StatusOK,
			expectedBody:   "Build triggered successfully",
		},
		{
			name:           "Invalid JSON",
			requestBody:    "invalid-json",
			mockEngine:     &MockCIEngine{},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Invalid request body",
		},
		{
			name: "Missing Job Name",
			requestBody: handlers.TriggerJenkinsBuildRequest{
				Job: "",
			},
			mockEngine:     &MockCIEngine{},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Job name is required",
		},
		{
			name: "Job Name Too Long",
			requestBody: handlers.TriggerJenkinsBuildRequest{
				Job: strings.Repeat("a", 256),
			},
			mockEngine:     &MockCIEngine{},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Job name exceeds maximum length",
		},
		{
			name: "Too Many Parameters",
			requestBody: handlers.TriggerJenkinsBuildRequest{
				Job: "test-job",
				Parameters: func() map[string]string {
					params := make(map[string]string)
					for i := 0; i < 101; i++ {
						params[string(rune(i))] = "val"
					}
					return params
				}(),
			},
			mockEngine:     &MockCIEngine{},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Maximum 100 parameters allowed",
		},
		{
			name: "Parameter Key Too Long",
			requestBody: handlers.TriggerJenkinsBuildRequest{
				Job: "test-job",
				Parameters: map[string]string{
					strings.Repeat("a", 256): "val",
				},
			},
			mockEngine:     &MockCIEngine{},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Parameter key 'aaaaaaaa",
		},
		{
			name: "Parameter Value Too Long",
			requestBody: handlers.TriggerJenkinsBuildRequest{
				Job: "test-job",
				Parameters: map[string]string{
					"key": strings.Repeat("a", 10241),
				},
			},
			mockEngine:     &MockCIEngine{},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Parameter value for 'key' exceeds maximum length",
		},
		{
			name: "Engine Error",
			requestBody: handlers.TriggerJenkinsBuildRequest{
				Job: "test-job",
			},
			mockEngine: &MockCIEngine{
				TriggerBuildFunc: func(jobName string, params map[string]string) (*engine.BuildResult, error) {
					return &engine.BuildResult{
						Success: false,
						Message: "jenkins unreachable",
					}, errors.New("jenkins unreachable")
				},
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   "jenkins unreachable",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := handlers.NewJenkinsHandler(tt.mockEngine)

			var reqBodyBytes []byte
			if s, ok := tt.requestBody.(string); ok && s == "invalid-json" {
				reqBodyBytes = []byte("invalid-json")
			} else {
				var err error
				reqBodyBytes, err = json.Marshal(tt.requestBody)
				if err != nil {
					t.Fatalf("Failed to marshal request body: %v", err)
				}
			}

			req := httptest.NewRequest("POST", "/api/v1/trigger/jenkins", bytes.NewReader(reqBodyBytes))
			// Inject API key context as middleware would
			ctx := context.WithValue(req.Context(), middleware.APIKeyContextKey, "test-api-key")
			req = req.WithContext(ctx)

			rr := httptest.NewRecorder()

			handler.TriggerJenkinsBuild(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, rr.Code)
			}

			if !strings.Contains(rr.Body.String(), tt.expectedBody) {
				t.Errorf("Expected body to contain %q, got %q", tt.expectedBody, rr.Body.String())
			}
		})
	}
}
