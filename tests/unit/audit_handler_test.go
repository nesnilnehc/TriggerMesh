package unit

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"triggermesh/internal/api/handlers"
	"triggermesh/internal/api/middleware"
	"triggermesh/internal/storage"
	"triggermesh/internal/storage/models"
)

func TestGetAuditLogsHandler(t *testing.T) {
	// Setup storage
	tmpFile, err := os.CreateTemp("", "test-audit-handler-*.db")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if err := storage.Init(tmpFile.Name()); err != nil {
		t.Fatalf("Failed to init storage: %v", err)
	}
	defer storage.Close()

	// Seed some data
	seedLogs(t, 20)

	handler := handlers.NewAuditHandler()

	tests := []struct {
		name           string
		queryParams    string
		expectedStatus int
		expectedCount  int
	}{
		{
			name:           "Default parameters",
			queryParams:    "",
			expectedStatus: http.StatusOK,
			expectedCount:  20,
		},
		{
			name:           "Limit 10",
			queryParams:    "?limit=10",
			expectedStatus: http.StatusOK,
			expectedCount:  10,
		},
		{
			name:           "Offset 10",
			queryParams:    "?offset=10",
			expectedStatus: http.StatusOK,
			expectedCount:  10,
		},
		{
			name:           "Limit 5 Offset 5",
			queryParams:    "?limit=5&offset=5",
			expectedStatus: http.StatusOK,
			expectedCount:  5,
		},
		{
			name:           "Invalid Limit (ignored)",
			queryParams:    "?limit=invalid",
			expectedStatus: http.StatusOK,
			expectedCount:  20, // defaults to 100
		},
		{
			name:           "Invalid Offset (ignored)",
			queryParams:    "?offset=invalid",
			expectedStatus: http.StatusOK,
			expectedCount:  20,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/v1/audit"+tt.queryParams, nil)
			// Add request ID to context to test error response includes it
			ctx := context.WithValue(req.Context(), middleware.RequestIDContextKey, "test-request-id-456")
			req = req.WithContext(ctx)
			rr := httptest.NewRecorder()

			handler.GetAuditLogs(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, rr.Code)
			}

			if rr.Code == http.StatusOK {
				var logs []models.AuditLog
				if err := json.NewDecoder(rr.Body).Decode(&logs); err != nil {
					t.Fatalf("Failed to decode response: %v", err)
				}

				if len(logs) != tt.expectedCount {
					t.Errorf("Expected %d logs, got %d", tt.expectedCount, len(logs))
				}
			}
		})
	}
}

func TestGetAuditLogsErrorResponse(t *testing.T) {
	// Test error response includes request ID
	handler := handlers.NewAuditHandler()

	// Initialize and then close a test database to force an error
	// This ensures we don't affect other tests by closing global storage
	tmpFile, err := os.CreateTemp("", "test-error-*.db")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	if err := storage.Init(tmpFile.Name()); err != nil {
		t.Fatalf("Failed to init storage: %v", err)
	}
	// Close the test database to force an error
	storage.Close()

	req := httptest.NewRequest("GET", "/api/v1/audit", nil)
	ctx := context.WithValue(req.Context(), middleware.RequestIDContextKey, "test-request-id-error")
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	handler.GetAuditLogs(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", rr.Code)
	}

	// Verify error response format and request ID
	var errorResp map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&errorResp); err != nil {
		t.Fatalf("Failed to decode error response: %v", err)
	}

	if errorResp["error"] == nil {
		t.Error("Expected 'error' field in error response")
	}

	if requestID, ok := errorResp["request_id"].(string); ok {
		if requestID != "test-request-id-error" {
			t.Errorf("Expected request_id 'test-request-id-error', got %q", requestID)
		}
	} else {
		t.Error("Expected 'request_id' field in error response")
	}
}

func TestGetAuditLogsErrorResponseWithoutRequestID(t *testing.T) {
	// Test error response when request ID is not in context
	handler := handlers.NewAuditHandler()

	// Initialize and then close a test database to force an error
	// This ensures we don't affect other tests by closing global storage
	tmpFile, err := os.CreateTemp("", "test-error-no-id-*.db")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	if err := storage.Init(tmpFile.Name()); err != nil {
		t.Fatalf("Failed to init storage: %v", err)
	}
	// Close the test database to force an error
	storage.Close()

	req := httptest.NewRequest("GET", "/api/v1/audit", nil)
	// Don't add request ID to context
	rr := httptest.NewRecorder()

	handler.GetAuditLogs(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", rr.Code)
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

func TestGetAuditLogsWithLargeLimit(t *testing.T) {
	// Setup storage
	tmpFile, err := os.CreateTemp("", "test-audit-large-*.db")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if err := storage.Init(tmpFile.Name()); err != nil {
		t.Fatalf("Failed to init storage: %v", err)
	}
	defer storage.Close()

	// Seed some data
	seedLogs(t, 5)

	handler := handlers.NewAuditHandler()

	req := httptest.NewRequest("GET", "/api/v1/audit?limit=1000", nil)
	ctx := context.WithValue(req.Context(), middleware.RequestIDContextKey, "test-request-id-large")
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	handler.GetAuditLogs(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}

	var logs []models.AuditLog
	if err := json.NewDecoder(rr.Body).Decode(&logs); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Should return all available logs (5), not 1000
	if len(logs) != 5 {
		t.Errorf("Expected 5 logs, got %d", len(logs))
	}
}

func TestGetAuditLogsWithOffsetBeyondData(t *testing.T) {
	// Setup storage
	tmpFile, err := os.CreateTemp("", "test-audit-offset-*.db")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if err := storage.Init(tmpFile.Name()); err != nil {
		t.Fatalf("Failed to init storage: %v", err)
	}
	defer storage.Close()

	// Seed some data
	seedLogs(t, 5)

	handler := handlers.NewAuditHandler()

	// Request with offset beyond available data
	req := httptest.NewRequest("GET", "/api/v1/audit?limit=10&offset=100", nil)
	ctx := context.WithValue(req.Context(), middleware.RequestIDContextKey, "test-request-id-offset")
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	handler.GetAuditLogs(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}

	var logs []models.AuditLog
	if err := json.NewDecoder(rr.Body).Decode(&logs); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Should return empty array when offset is beyond data
	if len(logs) != 0 {
		t.Errorf("Expected 0 logs with offset beyond data, got %d", len(logs))
	}
}

func TestGetAuditLogsWithZeroLimit(t *testing.T) {
	// Setup storage
	tmpFile, err := os.CreateTemp("", "test-audit-zero-*.db")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if err := storage.Init(tmpFile.Name()); err != nil {
		t.Fatalf("Failed to init storage: %v", err)
	}
	defer storage.Close()

	// Seed some data
	seedLogs(t, 10)

	handler := handlers.NewAuditHandler()

	// Request with zero limit (should use default)
	req := httptest.NewRequest("GET", "/api/v1/audit?limit=0", nil)
	ctx := context.WithValue(req.Context(), middleware.RequestIDContextKey, "test-request-id-zero")
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	handler.GetAuditLogs(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}

	var logs []models.AuditLog
	if err := json.NewDecoder(rr.Body).Decode(&logs); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Should use default limit (100)
	if len(logs) != 10 {
		t.Errorf("Expected 10 logs, got %d", len(logs))
	}
}

func TestGetAuditLogsWithNegativeValues(t *testing.T) {
	// Setup storage
	tmpFile, err := os.CreateTemp("", "test-audit-negative-*.db")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if err := storage.Init(tmpFile.Name()); err != nil {
		t.Fatalf("Failed to init storage: %v", err)
	}
	defer storage.Close()

	// Seed some data
	seedLogs(t, 10)

	handler := handlers.NewAuditHandler()

	// Request with negative limit and offset (should be ignored, use defaults)
	req := httptest.NewRequest("GET", "/api/v1/audit?limit=-10&offset=-5", nil)
	ctx := context.WithValue(req.Context(), middleware.RequestIDContextKey, "test-request-id-negative")
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	handler.GetAuditLogs(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}

	var logs []models.AuditLog
	if err := json.NewDecoder(rr.Body).Decode(&logs); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Should use default limit (100) and offset (0)
	if len(logs) != 10 {
		t.Errorf("Expected 10 logs, got %d", len(logs))
	}
}

func seedLogs(t *testing.T, count int) {
	for i := 0; i < count; i++ {
		log := models.AuditLog{
			Timestamp: time.Now(),
			APIKey:    "key",
			Method:    "GET",
			Path:      "/test",
			Status:    200,
			JobName:   "job",
			Params:    fmt.Sprintf(`{"i":%d}`, i),
			Result:    "success",
		}
		if err := storage.InsertAuditLog(log); err != nil {
			t.Fatalf("Failed to seed log: %v", err)
		}
	}
}
