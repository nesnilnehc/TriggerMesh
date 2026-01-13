package unit

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"triggermesh/internal/api/handlers"
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
