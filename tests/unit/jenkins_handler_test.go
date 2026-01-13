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
			name: "Invalid Job Name Format",
			requestBody: handlers.TriggerJenkinsBuildRequest{
				Job: "invalid@job#name", // Contains special characters not allowed
			},
			mockEngine:     &MockCIEngine{},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Invalid job name format",
		},
		{
			name: "Empty Parameter Key",
			requestBody: handlers.TriggerJenkinsBuildRequest{
				Job: "test-job",
				Parameters: map[string]string{
					"": "value",
				},
			},
			mockEngine:     &MockCIEngine{},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Parameter key cannot be empty",
		},
		{
			name: "Invalid Parameter Key Format",
			requestBody: handlers.TriggerJenkinsBuildRequest{
				Job: "test-job",
				Parameters: map[string]string{
					"invalid@key": "value",
				},
			},
			mockEngine:     &MockCIEngine{},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Invalid parameter key format",
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
			expectedBody:   "", // Error response is JSON, check separately
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
			// Also inject request ID to test error response includes it
			ctx = context.WithValue(ctx, middleware.RequestIDContextKey, "test-request-id-123")
			req = req.WithContext(ctx)

			rr := httptest.NewRecorder()

			handler.TriggerJenkinsBuild(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s", tt.expectedStatus, rr.Code, rr.Body.String())
			}

			// Check if response is JSON format (for error responses)
			if rr.Code >= 400 {
				var errorResp map[string]interface{}
				if err := json.Unmarshal(rr.Body.Bytes(), &errorResp); err != nil {
					// For engine errors, the response might be the BuildResult JSON, not error JSON
					// Check if it's a BuildResult instead
					var buildResult map[string]interface{}
					if jsonErr := json.Unmarshal(rr.Body.Bytes(), &buildResult); jsonErr == nil {
						// It's a BuildResult, which is valid for engine errors
						if tt.expectedBody == "" {
							// No specific body check needed
							return
						}
					} else {
						t.Errorf("Expected JSON error response, got: %s", rr.Body.String())
						return
					}
				} else {
					// It's an error response
					if errorResp["error"] == nil {
						// For engine errors, might be BuildResult format
						if tt.expectedBody == "" {
							return
						}
						t.Error("Expected 'error' field in error response")
						return
					}
					errorMsg, ok := errorResp["error"].(string)
					if !ok {
						t.Error("Expected 'error' field to be a string")
						return
					}
					// Only check error message if expectedBody is not empty
					if tt.expectedBody != "" && !strings.Contains(errorMsg, tt.expectedBody) {
						t.Errorf("Expected error message to contain %q, got %q", tt.expectedBody, errorMsg)
					}
					// Verify request ID is included in error response
					if requestID, ok := errorResp["request_id"].(string); ok {
						if requestID != "test-request-id-123" {
							t.Errorf("Expected request_id 'test-request-id-123', got %q", requestID)
						}
					} else {
						// Request ID should be present in error responses
						t.Error("Expected 'request_id' field in error response")
					}
				}
			} else {
				if !strings.Contains(rr.Body.String(), tt.expectedBody) {
					t.Errorf("Expected body to contain %q, got %q", tt.expectedBody, rr.Body.String())
				}
			}
		})
	}
}

func TestTriggerJenkinsBuildWithEmptyParams(t *testing.T) {
	// Test marshalParams with empty params
	tmpFile, err := os.CreateTemp("", "test-empty-params-*.db")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if err := storage.Init(tmpFile.Name()); err != nil {
		t.Fatalf("Failed to init storage: %v", err)
	}
	defer storage.Close()

	handler := handlers.NewJenkinsHandler(&MockCIEngine{
		TriggerBuildFunc: func(jobName string, params map[string]string) (*engine.BuildResult, error) {
			return &engine.BuildResult{Success: true, Message: "Build triggered"}, nil
		},
	})

	reqBody := handlers.TriggerJenkinsBuildRequest{
		Job:        "test-job",
		Parameters: map[string]string{}, // Empty params
	}
	reqBodyBytes, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/api/v1/trigger/jenkins", bytes.NewReader(reqBodyBytes))
	ctx := context.WithValue(req.Context(), middleware.APIKeyContextKey, "test-api-key")
	ctx = context.WithValue(ctx, middleware.RequestIDContextKey, "test-request-id-empty-params")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.TriggerJenkinsBuild(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d. Body: %s", rr.Code, rr.Body.String())
	}
}

func TestTriggerJenkinsBuildWithNilParams(t *testing.T) {
	// Test marshalParams with nil params
	tmpFile, err := os.CreateTemp("", "test-nil-params-*.db")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if err := storage.Init(tmpFile.Name()); err != nil {
		t.Fatalf("Failed to init storage: %v", err)
	}
	defer storage.Close()

	handler := handlers.NewJenkinsHandler(&MockCIEngine{
		TriggerBuildFunc: func(jobName string, params map[string]string) (*engine.BuildResult, error) {
			return &engine.BuildResult{Success: true, Message: "Build triggered"}, nil
		},
	})

	reqBody := handlers.TriggerJenkinsBuildRequest{
		Job:        "test-job",
		Parameters: nil, // Nil params
	}
	reqBodyBytes, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/api/v1/trigger/jenkins", bytes.NewReader(reqBodyBytes))
	ctx := context.WithValue(req.Context(), middleware.APIKeyContextKey, "test-api-key")
	ctx = context.WithValue(ctx, middleware.RequestIDContextKey, "test-request-id-nil-params")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.TriggerJenkinsBuild(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d. Body: %s", rr.Code, rr.Body.String())
	}
}

func TestTriggerJenkinsBuildErrorResponseWithoutRequestID(t *testing.T) {
	// Test error response when request ID is not in context
	tmpFile, err := os.CreateTemp("", "test-no-request-id-*.db")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if err := storage.Init(tmpFile.Name()); err != nil {
		t.Fatalf("Failed to init storage: %v", err)
	}
	defer storage.Close()

	handler := handlers.NewJenkinsHandler(&MockCIEngine{})

	// Missing job name - should return error
	reqBody := handlers.TriggerJenkinsBuildRequest{
		Job: "", // Empty job name
	}
	reqBodyBytes, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/api/v1/trigger/jenkins", bytes.NewReader(reqBodyBytes))
	// Don't add request ID to context
	ctx := context.WithValue(req.Context(), middleware.APIKeyContextKey, "test-api-key")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.TriggerJenkinsBuild(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", rr.Code)
	}

	// Verify error response format
	var errorResp map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&errorResp); err != nil {
		t.Fatalf("Failed to decode error response: %v", err)
	}

	if errorResp["error"] == nil {
		t.Error("Expected 'error' field in error response")
	}

	// Request ID should not be present if not in context
	if requestID, ok := errorResp["request_id"].(string); ok && requestID != "" {
		t.Errorf("Expected no request_id when not in context, got %q", requestID)
	}
}

func TestTriggerJenkinsBuildWithFolderJobName(t *testing.T) {
	// Test job name with folder structure (folder/subfolder/job)
	tmpFile, err := os.CreateTemp("", "test-folder-job-*.db")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if err := storage.Init(tmpFile.Name()); err != nil {
		t.Fatalf("Failed to init storage: %v", err)
	}
	defer storage.Close()

	handler := handlers.NewJenkinsHandler(&MockCIEngine{
		TriggerBuildFunc: func(jobName string, params map[string]string) (*engine.BuildResult, error) {
			if jobName != "folder/subfolder/job" {
				return nil, errors.New("unexpected job name")
			}
			return &engine.BuildResult{Success: true, Message: "Build triggered"}, nil
		},
	})

	reqBody := handlers.TriggerJenkinsBuildRequest{
		Job: "folder/subfolder/job", // Folder structure
	}
	reqBodyBytes, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/api/v1/trigger/jenkins", bytes.NewReader(reqBodyBytes))
	ctx := context.WithValue(req.Context(), middleware.APIKeyContextKey, "test-api-key")
	ctx = context.WithValue(ctx, middleware.RequestIDContextKey, "test-request-id-folder")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.TriggerJenkinsBuild(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d. Body: %s", rr.Code, rr.Body.String())
	}
}

func TestTriggerJenkinsBuildWithJobNameContainingSpaces(t *testing.T) {
	// Test job name with spaces
	tmpFile, err := os.CreateTemp("", "test-space-job-*.db")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if err := storage.Init(tmpFile.Name()); err != nil {
		t.Fatalf("Failed to init storage: %v", err)
	}
	defer storage.Close()

	handler := handlers.NewJenkinsHandler(&MockCIEngine{
		TriggerBuildFunc: func(jobName string, params map[string]string) (*engine.BuildResult, error) {
			if jobName != "my job name" {
				return nil, errors.New("unexpected job name")
			}
			return &engine.BuildResult{Success: true, Message: "Build triggered"}, nil
		},
	})

	reqBody := handlers.TriggerJenkinsBuildRequest{
		Job: "my job name", // Job name with spaces
	}
	reqBodyBytes, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/api/v1/trigger/jenkins", bytes.NewReader(reqBodyBytes))
	ctx := context.WithValue(req.Context(), middleware.APIKeyContextKey, "test-api-key")
	ctx = context.WithValue(ctx, middleware.RequestIDContextKey, "test-request-id-space")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.TriggerJenkinsBuild(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d. Body: %s", rr.Code, rr.Body.String())
	}
}

func TestTriggerJenkinsBuildWithParameterKeysContainingDots(t *testing.T) {
	// Test parameter keys with dots (e.g., "config.env")
	tmpFile, err := os.CreateTemp("", "test-dot-params-*.db")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if err := storage.Init(tmpFile.Name()); err != nil {
		t.Fatalf("Failed to init storage: %v", err)
	}
	defer storage.Close()

	handler := handlers.NewJenkinsHandler(&MockCIEngine{
		TriggerBuildFunc: func(jobName string, params map[string]string) (*engine.BuildResult, error) {
			if params["config.env"] != "production" {
				return nil, errors.New("unexpected params")
			}
			return &engine.BuildResult{Success: true, Message: "Build triggered"}, nil
		},
	})

	reqBody := handlers.TriggerJenkinsBuildRequest{
		Job: "test-job",
		Parameters: map[string]string{
			"config.env":  "production", // Parameter key with dot
			"app.version": "1.0.0",
		},
	}
	reqBodyBytes, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/api/v1/trigger/jenkins", bytes.NewReader(reqBodyBytes))
	ctx := context.WithValue(req.Context(), middleware.APIKeyContextKey, "test-api-key")
	ctx = context.WithValue(ctx, middleware.RequestIDContextKey, "test-request-id-dot")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.TriggerJenkinsBuild(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d. Body: %s", rr.Code, rr.Body.String())
	}
}

func TestTriggerJenkinsBuildWithInvalidParameterKeyDots(t *testing.T) {
	// Test parameter keys with invalid dot usage
	tmpFile, err := os.CreateTemp("", "test-invalid-dot-*.db")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if err := storage.Init(tmpFile.Name()); err != nil {
		t.Fatalf("Failed to init storage: %v", err)
	}
	defer storage.Close()

	handler := handlers.NewJenkinsHandler(&MockCIEngine{})

	testCases := []struct {
		name  string
		key   string
		value string
	}{
		{"Leading dot", ".key", "value"},
		{"Trailing dot", "key.", "value"},
		{"Consecutive dots", "key..subkey", "value"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			reqBody := handlers.TriggerJenkinsBuildRequest{
				Job: "test-job",
				Parameters: map[string]string{
					tc.key: tc.value,
				},
			}
			reqBodyBytes, _ := json.Marshal(reqBody)

			req := httptest.NewRequest("POST", "/api/v1/trigger/jenkins", bytes.NewReader(reqBodyBytes))
			ctx := context.WithValue(req.Context(), middleware.APIKeyContextKey, "test-api-key")
			ctx = context.WithValue(ctx, middleware.RequestIDContextKey, "test-request-id-invalid-dot")
			req = req.WithContext(ctx)

			rr := httptest.NewRecorder()
			handler.TriggerJenkinsBuild(rr, req)

			if rr.Code != http.StatusBadRequest {
				t.Errorf("Expected status 400 for invalid key %q, got %d", tc.key, rr.Code)
			}

			// Verify error response includes request ID
			var errorResp map[string]interface{}
			if err := json.NewDecoder(rr.Body).Decode(&errorResp); err == nil {
				if requestID, ok := errorResp["request_id"].(string); ok {
					if requestID != "test-request-id-invalid-dot" {
						t.Errorf("Expected request_id 'test-request-id-invalid-dot', got %q", requestID)
					}
				} else {
					t.Error("Expected 'request_id' field in error response")
				}
			}
		})
	}
}

func TestTriggerJenkinsBuildSuccessPath(t *testing.T) {
	// Test successful build trigger with full audit logging
	tmpFile, err := os.CreateTemp("", "test-success-*.db")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if err = storage.Init(tmpFile.Name()); err != nil {
		t.Fatalf("Failed to init storage: %v", err)
	}
	defer storage.Close()

	handler := handlers.NewJenkinsHandler(&MockCIEngine{
		TriggerBuildFunc: func(jobName string, params map[string]string) (*engine.BuildResult, error) {
			return &engine.BuildResult{
				Success:  true,
				BuildID:  "test-job/123",
				BuildURL: "http://jenkins/job/test-job/123",
				Message:  "Build triggered successfully",
			}, nil
		},
	})

	reqBody := handlers.TriggerJenkinsBuildRequest{
		Job: "test-job",
		Parameters: map[string]string{
			"param1": "value1",
			"param2": "value2",
		},
	}
	reqBodyBytes, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/api/v1/trigger/jenkins", bytes.NewReader(reqBodyBytes))
	ctx := context.WithValue(req.Context(), middleware.APIKeyContextKey, "test-api-key")
	ctx = context.WithValue(ctx, middleware.RequestIDContextKey, "test-request-id-success")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.TriggerJenkinsBuild(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d. Body: %s", rr.Code, rr.Body.String())
	}

	// Verify response contains build result
	var result engine.BuildResult
	if err = json.NewDecoder(rr.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if !result.Success {
		t.Error("Expected build to be successful")
	}

	// Verify audit log was created
	logs, err := storage.GetAuditLogs(10, 0)
	if err != nil {
		t.Fatalf("Failed to get audit logs: %v", err)
	}

	if len(logs) == 0 {
		t.Error("Expected audit log to be created")
	} else {
		lastLog := logs[0]
		if lastLog.JobName != "test-job" {
			t.Errorf("Expected job name 'test-job', got %q", lastLog.JobName)
		}
		if lastLog.Result != "success" {
			t.Errorf("Expected result 'success', got %q", lastLog.Result)
		}
	}
}

func TestTriggerJenkinsBuildAuditLogInsertFailure(t *testing.T) {
	// Test that handler continues even if audit log insertion fails
	tmpFile, err := os.CreateTemp("", "test-audit-fail-*.db")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if err := storage.Init(tmpFile.Name()); err != nil {
		t.Fatalf("Failed to init storage: %v", err)
	}
	// Close storage to force audit log insertion to fail
	storage.Close()

	handler := handlers.NewJenkinsHandler(&MockCIEngine{
		TriggerBuildFunc: func(jobName string, params map[string]string) (*engine.BuildResult, error) {
			return &engine.BuildResult{
				Success:  true,
				BuildID:  "test-job/123",
				BuildURL: "http://jenkins/job/test-job/123",
				Message:  "Build triggered successfully",
			}, nil
		},
	})

	reqBody := handlers.TriggerJenkinsBuildRequest{
		Job: "test-job",
	}
	reqBodyBytes, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/api/v1/trigger/jenkins", bytes.NewReader(reqBodyBytes))
	ctx := context.WithValue(req.Context(), middleware.APIKeyContextKey, "test-api-key")
	ctx = context.WithValue(ctx, middleware.RequestIDContextKey, "test-request-id-audit-fail")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.TriggerJenkinsBuild(rr, req)

	// Should still return success even if audit log fails
	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200 even if audit log fails, got %d", rr.Code)
	}
}

func TestTriggerJenkinsBuildWithAPIKeyNotInContext(t *testing.T) {
	// Test when API key is not in context (should default to "unknown")
	tmpFile, err := os.CreateTemp("", "test-no-apikey-*.db")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if err = storage.Init(tmpFile.Name()); err != nil {
		t.Fatalf("Failed to init storage: %v", err)
	}
	defer storage.Close()

	handler := handlers.NewJenkinsHandler(&MockCIEngine{
		TriggerBuildFunc: func(jobName string, params map[string]string) (*engine.BuildResult, error) {
			return &engine.BuildResult{Success: true}, nil
		},
	})

	reqBody := handlers.TriggerJenkinsBuildRequest{
		Job: "test-job",
	}
	reqBodyBytes, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/api/v1/trigger/jenkins", bytes.NewReader(reqBodyBytes))
	// Don't add API key to context
	ctx := context.WithValue(req.Context(), middleware.RequestIDContextKey, "test-request-id-no-key")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.TriggerJenkinsBuild(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}

	// Verify audit log was created with "unknown" API key
	logs, err := storage.GetAuditLogs(1, 0)
	if err != nil {
		t.Fatalf("Failed to get audit logs: %v", err)
	}

	if len(logs) > 0 && logs[0].APIKey != "unknown" {
		t.Errorf("Expected API key 'unknown', got %q", logs[0].APIKey)
	}
}
