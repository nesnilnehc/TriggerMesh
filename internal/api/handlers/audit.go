package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"triggermesh/internal/storage"
)

// AuditHandler handles audit log-related API requests
type AuditHandler struct{}

// NewAuditHandler creates a new AuditHandler instance
func NewAuditHandler() *AuditHandler {
	return &AuditHandler{}
}

// GetAuditLogs handles the GET /api/v1/audit request
func (h *AuditHandler) GetAuditLogs(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")

	// Set default values
	limit := 100
	offset := 0

	// Parse limit if provided
	if limitStr != "" {
		parsedLimit, err := strconv.Atoi(limitStr)
		if err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	// Parse offset if provided
	if offsetStr != "" {
		parsedOffset, err := strconv.Atoi(offsetStr)
		if err == nil && parsedOffset >= 0 {
			offset = parsedOffset
		}
	}

	// Get audit logs from database
	logs, err := storage.GetAuditLogs(limit, offset)
	if err != nil {
		http.Error(w, "Failed to get audit logs", http.StatusInternalServerError)
		return
	}

	// Return the logs as JSON
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	// Encode response
	if err := json.NewEncoder(w).Encode(logs); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}
