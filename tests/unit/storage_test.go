package unit

import (
	"os"
	"testing"
	"time"

	"triggermesh/internal/storage"
	"triggermesh/internal/storage/models"
)

func TestStorageInit(t *testing.T) {
	// Create a temporary database file
	tmpFile, err := os.CreateTemp("", "test-*.db")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	// Initialize storage
	err = storage.Init(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to initialize storage: %v", err)
	}
	defer storage.Close()

	// Verify database was created
	if _, err := os.Stat(tmpFile.Name()); os.IsNotExist(err) {
		t.Error("Database file was not created")
	}
}

func TestInsertAuditLog(t *testing.T) {
	// Create a temporary database file
	tmpFile, err := os.CreateTemp("", "test-*.db")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	// Initialize storage
	err = storage.Init(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to initialize storage: %v", err)
	}
	defer storage.Close()

	// Insert a test audit log
	auditLog := models.AuditLog{
		Timestamp: time.Now(),
		APIKey:    "test-api-key",
		Method:    "POST",
		Path:      "/api/v1/trigger/jenkins",
		Status:    200,
		JobName:   "test-job",
		Params:    `{"param1":"value1"}`,
		Result:    "success",
	}

	err = storage.InsertAuditLog(auditLog)
	if err != nil {
		t.Fatalf("Failed to insert audit log: %v", err)
	}
}

func TestGetAuditLogs(t *testing.T) {
	// Create a temporary database file
	tmpFile, err := os.CreateTemp("", "test-*.db")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	// Initialize storage
	err = storage.Init(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to initialize storage: %v", err)
	}
	defer storage.Close()

	// Insert multiple test audit logs with increasing timestamps
	// Use larger intervals to ensure different timestamps after database storage
	baseTime := time.Now()
	for i := 0; i < 5; i++ {
		auditLog := models.AuditLog{
			Timestamp: baseTime.Add(time.Duration(i) * 100 * time.Millisecond),
			APIKey:    "test-api-key",
			Method:    "POST",
			Path:      "/api/v1/trigger/jenkins",
			Status:    200,
			JobName:   "test-job",
			Params:    `{"param1":"value1"}`,
			Result:    "success",
		}
		err = storage.InsertAuditLog(auditLog)
		if err != nil {
			t.Fatalf("Failed to insert audit log: %v", err)
		}
	}

	// Retrieve audit logs
	logs, err := storage.GetAuditLogs(10, 0)
	if err != nil {
		t.Fatalf("Failed to get audit logs: %v", err)
	}

	// Verify we got the logs
	if len(logs) != 5 {
		t.Errorf("Expected 5 logs, got %d", len(logs))
	}

	// Verify logs are ordered by ID DESC (newest first, since ID is auto-increment)
	if len(logs) > 1 {
		// Check that IDs are in descending order (newest first)
		for i := 0; i < len(logs)-1; i++ {
			if logs[i].ID <= logs[i+1].ID {
				t.Errorf("Logs are not ordered by ID DESC: log[%d].ID (%d) <= log[%d].ID (%d)",
					i, logs[i].ID, i+1, logs[i+1].ID)
			}
		}
		// Verify all logs have valid timestamps
		for i, log := range logs {
			if log.Timestamp.IsZero() {
				t.Errorf("Log %d has zero timestamp", i)
			}
		}
	}

	// Test pagination
	logs, err = storage.GetAuditLogs(2, 0)
	if err != nil {
		t.Fatalf("Failed to get audit logs: %v", err)
	}
	if len(logs) != 2 {
		t.Errorf("Expected 2 logs with limit 2, got %d", len(logs))
	}

	logs, err = storage.GetAuditLogs(2, 2)
	if err != nil {
		t.Fatalf("Failed to get audit logs: %v", err)
	}
	if len(logs) != 2 {
		t.Errorf("Expected 2 logs with limit 2 offset 2, got %d", len(logs))
	}
}

func TestAuditLogWithError(t *testing.T) {
	// Create a temporary database file
	tmpFile, err := os.CreateTemp("", "test-*.db")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	// Initialize storage
	err = storage.Init(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to initialize storage: %v", err)
	}
	defer storage.Close()

	// Insert an audit log with error
	auditLog := models.AuditLog{
		Timestamp: time.Now(),
		APIKey:    "test-api-key",
		Method:    "POST",
		Path:      "/api/v1/trigger/jenkins",
		Status:    500,
		JobName:   "test-job",
		Params:    `{"param1":"value1"}`,
		Result:    "failed",
		Error:     "Jenkins build failed",
	}

	err = storage.InsertAuditLog(auditLog)
	if err != nil {
		t.Fatalf("Failed to insert audit log: %v", err)
	}

	// Retrieve and verify
	logs, err := storage.GetAuditLogs(1, 0)
	if err != nil {
		t.Fatalf("Failed to get audit logs: %v", err)
	}

	if len(logs) != 1 {
		t.Fatalf("Expected 1 log, got %d", len(logs))
	}

	if logs[0].Error != "Jenkins build failed" {
		t.Errorf("Expected error message 'Jenkins build failed', got %s", logs[0].Error)
	}
	if logs[0].Status != 500 {
		t.Errorf("Expected status 500, got %d", logs[0].Status)
	}
}

func TestInsertAuditLog_Error(t *testing.T) {
	// Setup then close completely
	tmpFile, err := os.CreateTemp("", "test-*.db")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	if err = storage.Init(tmpFile.Name()); err != nil {
		t.Fatalf("Failed to init storage: %v", err)
	}
	// Force close
	storage.Close()

	auditLog := models.AuditLog{
		Timestamp: time.Now(),
		APIKey:    "test-api-key",
		Method:    "POST",
	}

	// Should fail because DB is closed (or we rely on driver behavior)
	err = storage.InsertAuditLog(auditLog)
	if err == nil {
		t.Error("Expected error inserting into closed DB, got nil")
	}
}

func TestGetAuditLogs_Error(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test-*.db")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	if err = storage.Init(tmpFile.Name()); err != nil {
		t.Fatalf("Failed to init storage: %v", err)
	}
	storage.Close()

	_, err = storage.GetAuditLogs(10, 0)
	if err == nil {
		t.Error("Expected error getting logs from closed DB, got nil")
	}
}

func TestInit_Error(t *testing.T) {
	// Test: init with a path that should fail (directory doesn't exist)
	// SQLite driver should error if parent directory doesn't exist
	err := storage.Init("/path/to/non/existent/directory/test.db")
	if err == nil {
		t.Error("Expected error initializing with non-existent directory path, got nil")
		// Clean up if somehow succeeded
		storage.Close()
		os.Remove("/path/to/non/existent/directory/test.db")
		os.RemoveAll("/path/to/non/existent/directory")
	}

	// Test: pass a directory as file path (should fail)
	tmpDir, err := os.MkdirTemp("", "test-dir")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	err = storage.Init(tmpDir) // Should fail because it's a directory, not a file
	if err == nil {
		t.Error("Expected error initializing with directory path, got nil")
		storage.Close()
	}
}

func TestPing(t *testing.T) {
	// Test Ping with initialized database
	tmpFile, err := os.CreateTemp("", "test-*.db")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	err = storage.Init(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to initialize storage: %v", err)
	}
	defer storage.Close()

	// Ping should succeed
	err = storage.Ping()
	if err != nil {
		t.Errorf("Expected Ping to succeed, got error: %v", err)
	}

	// Close and test Ping on closed database
	storage.Close()
	err = storage.Ping()
	if err == nil {
		t.Error("Expected error pinging closed database, got nil")
	}
}

func TestGetAuditLogsTimestampParsing(t *testing.T) {
	// Test timestamp parsing with different formats
	tmpFile, err := os.CreateTemp("", "test-timestamp-*.db")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	if err = storage.Init(tmpFile.Name()); err != nil {
		t.Fatalf("Failed to init storage: %v", err)
	}
	defer storage.Close()

	// Insert logs with different timestamp formats
	// The storage layer formats timestamps, but we can test retrieval
	logs := []models.AuditLog{
		{
			Timestamp: time.Now(),
			APIKey:    "key1",
			Method:    "GET",
			Path:      "/test1",
			Status:    200,
			Result:    "success",
		},
		{
			Timestamp: time.Now().Add(1 * time.Second),
			APIKey:    "key2",
			Method:    "POST",
			Path:      "/test2",
			Status:    201,
			Result:    "success",
		},
	}

	for _, log := range logs {
		if err = storage.InsertAuditLog(log); err != nil {
			t.Fatalf("Failed to insert audit log: %v", err)
		}
	}

	// Retrieve logs and verify timestamps are parsed correctly
	retrievedLogs, err := storage.GetAuditLogs(10, 0)
	if err != nil {
		t.Fatalf("Failed to get audit logs: %v", err)
	}

	if len(retrievedLogs) < 2 {
		t.Fatalf("Expected at least 2 logs, got %d", len(retrievedLogs))
	}

	// Verify timestamps are not zero
	for i, log := range retrievedLogs {
		if log.Timestamp.IsZero() {
			t.Errorf("Log %d has zero timestamp", i)
		}
	}
}

func TestGetAuditLogsEmptyResult(t *testing.T) {
	// Test GetAuditLogs with empty database
	tmpFile, err := os.CreateTemp("", "test-empty-*.db")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	if err = storage.Init(tmpFile.Name()); err != nil {
		t.Fatalf("Failed to init storage: %v", err)
	}
	defer storage.Close()

	// Get logs from empty database
	logs, err := storage.GetAuditLogs(10, 0)
	if err != nil {
		t.Fatalf("Failed to get audit logs: %v", err)
	}

	if len(logs) != 0 {
		t.Errorf("Expected 0 logs from empty database, got %d", len(logs))
	}
}

func TestGetAuditLogsWithRowsErr(t *testing.T) {
	// This is hard to test directly, but we can test edge cases
	tmpFile, err := os.CreateTemp("", "test-rows-err-*.db")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	if err = storage.Init(tmpFile.Name()); err != nil {
		t.Fatalf("Failed to init storage: %v", err)
	}
	defer storage.Close()

	// Insert a log
	log := models.AuditLog{
		Timestamp: time.Now(),
		APIKey:    "test-key",
		Method:    "GET",
		Path:      "/test",
		Status:    200,
		Result:    "success",
	}
	if err = storage.InsertAuditLog(log); err != nil {
		t.Fatalf("Failed to insert audit log: %v", err)
	}

	// Close database to force error on query
	storage.Close()

	// Should fail when database is closed
	_, err = storage.GetAuditLogs(10, 0)
	if err == nil {
		t.Error("Expected error getting logs from closed database, got nil")
	}
}

func TestGetAuditLogsTimestampFallback(t *testing.T) {
	// Test timestamp parsing fallback when format doesn't match
	tmpFile, err := os.CreateTemp("", "test-timestamp-fallback-*.db")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	if err = storage.Init(tmpFile.Name()); err != nil {
		t.Fatalf("Failed to init storage: %v", err)
	}
	defer storage.Close()

	// Insert a log directly with a timestamp that will need fallback parsing
	// We can't directly control the timestamp format, but we can test the retrieval
	log := models.AuditLog{
		Timestamp: time.Now(),
		APIKey:    "test-key",
		Method:    "GET",
		Path:      "/test",
		Status:    200,
		Result:    "success",
	}
	if err = storage.InsertAuditLog(log); err != nil {
		t.Fatalf("Failed to insert audit log: %v", err)
	}

	// Retrieve logs - should handle timestamp parsing correctly
	logs, err := storage.GetAuditLogs(10, 0)
	if err != nil {
		t.Fatalf("Failed to get audit logs: %v", err)
	}

	if len(logs) == 0 {
		t.Fatal("Expected at least one log")
	}

	// Verify timestamp is not zero (fallback should work)
	if logs[0].Timestamp.IsZero() {
		t.Error("Expected non-zero timestamp, got zero")
	}
}
