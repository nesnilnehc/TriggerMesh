package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"triggermesh/internal/api/middleware"
	"triggermesh/internal/engine"
	"triggermesh/internal/logger"
	"triggermesh/internal/storage"
	"triggermesh/internal/storage/models"
)

// JenkinsHandler handles Jenkins-related API requests
type JenkinsHandler struct {
	jenkinsEngine engine.CIEngine
}

// NewJenkinsHandler creates a new JenkinsHandler instance
func NewJenkinsHandler(jenkinsEngine engine.CIEngine) *JenkinsHandler {
	return &JenkinsHandler{
		jenkinsEngine: jenkinsEngine,
	}
}

// TriggerJenkinsBuildRequest represents the request body for triggering a Jenkins build
type TriggerJenkinsBuildRequest struct {
	Job        string            `json:"job"`
	Parameters map[string]string `json:"parameters"`
}

var (
	// jobNameRegex validates Jenkins job names (supports folder structure: folder/subfolder/job)
	// Jenkins job names can contain: alphanumeric, underscore, hyphen, slash, and spaces
	jobNameRegex = regexp.MustCompile(`^[a-zA-Z0-9_/\- ]+$`)
	// parameterKeyRegex validates parameter keys (alphanumeric, underscore, hyphen, dot)
	// No leading/trailing dots, no consecutive dots
	parameterKeyRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]+(\.[a-zA-Z0-9_-]+)*$`)
)

// TriggerJenkinsBuild handles the POST /api/v1/trigger/jenkins request
func (h *JenkinsHandler) TriggerJenkinsBuild(w http.ResponseWriter, r *http.Request) {
	// Get API key from context
	apiKey, ok := r.Context().Value(middleware.APIKeyContextKey).(string)
	if !ok {
		apiKey = "unknown"
	}

	// Get request ID for logging
	requestID := middleware.GetRequestID(r)

	// Parse request body
	var req TriggerJenkinsBuildRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Error("Failed to parse request body", "error", err, "request_id", requestID)
		writeErrorWithRequestID(w, r, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate required fields
	if req.Job == "" {
		logger.Error("Job name is required", "request_id", requestID)
		writeErrorWithRequestID(w, r, http.StatusBadRequest, "Job name is required")
		return
	}

	// Validate job name length (Jenkins job names are typically limited)
	if len(req.Job) > 255 {
		logger.Error("Job name too long", "length", len(req.Job), "request_id", requestID)
		writeErrorWithRequestID(w, r, http.StatusBadRequest, "Job name exceeds maximum length of 255 characters")
		return
	}

	// Validate job name format (supports folder structure: folder/subfolder/job)
	if !jobNameRegex.MatchString(req.Job) {
		logger.Error("Invalid job name format", "job", req.Job, "request_id", requestID)
		writeErrorWithRequestID(w, r, http.StatusBadRequest, "Invalid job name format: only alphanumeric characters, underscores, hyphens, slashes, and spaces are allowed")
		return
	}

	// Validate parameters
	if req.Parameters != nil {
		// Limit number of parameters
		if len(req.Parameters) > 100 {
			logger.Error("Too many parameters", "count", len(req.Parameters), "request_id", requestID)
			writeErrorWithRequestID(w, r, http.StatusBadRequest, "Maximum 100 parameters allowed")
			return
		}

		// Validate parameter keys and values
		for key, value := range req.Parameters {
			// Validate parameter key is not empty
			if key == "" {
				logger.Error("Parameter key cannot be empty", "request_id", requestID)
				writeErrorWithRequestID(w, r, http.StatusBadRequest, "Parameter key cannot be empty")
				return
			}

			// Validate parameter key length
			if len(key) > 255 {
				logger.Error("Parameter key too long", "key", key, "length", len(key), "request_id", requestID)
				writeErrorWithRequestID(w, r, http.StatusBadRequest, fmt.Sprintf("Parameter key '%s' exceeds maximum length of 255 characters", key))
				return
			}

			// Validate parameter key format (no leading/trailing dots, no consecutive dots)
			if !parameterKeyRegex.MatchString(key) {
				logger.Error("Invalid parameter key format", "key", key, "request_id", requestID)
				writeErrorWithRequestID(w, r, http.StatusBadRequest, fmt.Sprintf("Invalid parameter key format '%s': only alphanumeric characters, underscores, hyphens, and dots (not leading/trailing/consecutive) are allowed", key))
				return
			}

			// Additional validation: check for leading/trailing dots and consecutive dots
			if strings.HasPrefix(key, ".") || strings.HasSuffix(key, ".") || strings.Contains(key, "..") {
				logger.Error("Invalid parameter key format", "key", key, "request_id", requestID, "reason", "leading/trailing/consecutive dots not allowed")
				writeErrorWithRequestID(w, r, http.StatusBadRequest, fmt.Sprintf("Invalid parameter key format '%s': dots cannot be leading, trailing, or consecutive", key))
				return
			}

			// Validate parameter value length (limit to 10KB per parameter)
			if len(value) > 10240 {
				logger.Error("Parameter value too long", "key", key, "length", len(value), "request_id", requestID)
				writeErrorWithRequestID(w, r, http.StatusBadRequest, fmt.Sprintf("Parameter value for '%s' exceeds maximum length of 10KB", key))
				return
			}
		}
	}

	// Trigger the build
	result, err := h.jenkinsEngine.TriggerBuild(req.Job, req.Parameters)
	if err != nil {
		logger.Error("Failed to trigger Jenkins build", "error", err, "job", req.Job, "request_id", requestID)

		// Log the failure to audit logs
		auditLog := models.AuditLog{
			Timestamp: time.Now(),
			APIKey:    apiKey,
			Method:    r.Method,
			Path:      r.URL.Path,
			Status:    http.StatusInternalServerError,
			JobName:   req.Job,
			Params:    marshalParams(req.Parameters),
			Result:    "failed",
			Error:     err.Error(),
		}
		if err := storage.InsertAuditLog(auditLog); err != nil {
			logger.Error("Failed to insert audit log", "error", err)
		}

		w.WriteHeader(http.StatusInternalServerError)
		if err := json.NewEncoder(w).Encode(result); err != nil {
			logger.Error("Failed to encode response", "error", err)
		}
		return
	}

	// Log the success to audit logs
	auditLog := models.AuditLog{
		Timestamp: time.Now(),
		APIKey:    apiKey,
		Method:    r.Method,
		Path:      r.URL.Path,
		Status:    http.StatusOK,
		JobName:   req.Job,
		Params:    marshalParams(req.Parameters),
		Result:    "success",
	}
	if err := storage.InsertAuditLog(auditLog); err != nil {
		logger.Error("Failed to insert audit log", "error", err)
	}

	// Return the result
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(result); err != nil {
		logger.Error("Failed to encode response", "error", err)
	}
}

// marshalParams marshals parameters to a JSON string
func marshalParams(params map[string]string) string {
	jsonParams, err := json.Marshal(params)
	if err != nil {
		return "{}"
	}
	return string(jsonParams)
}
