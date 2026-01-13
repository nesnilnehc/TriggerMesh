package handlers

import (
	"encoding/json"
	"net/http"

	"triggermesh/internal/api/middleware"
	"triggermesh/internal/logger"
)

// writeErrorWithRequestID writes a standardized error response with optional request ID
func writeErrorWithRequestID(w http.ResponseWriter, r *http.Request, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	response := map[string]interface{}{
		"error":  message,
		"status": http.StatusText(status),
	}

	// Add request ID if available (from context, not header)
	if r != nil {
		if requestID := middleware.GetRequestID(r); requestID != "" {
			response["request_id"] = requestID
		}
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		// Log but don't try to write again (headers already sent)
		logger.Error("Failed to encode error response", "error", err, "status", status, "message", message)
	}
}
