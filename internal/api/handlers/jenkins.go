package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

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

// TriggerJenkinsBuild handles the POST /api/v1/trigger/jenkins request
func (h *JenkinsHandler) TriggerJenkinsBuild(w http.ResponseWriter, r *http.Request) {
	// Get API key from context
	apiKey, ok := r.Context().Value("api_key").(string)
	if !ok {
		apiKey = "unknown"
	}

	// Parse request body
	var req TriggerJenkinsBuildRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Error("Failed to parse request body", "error", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.Job == "" {
		logger.Error("Job name is required")
		http.Error(w, "Job name is required", http.StatusBadRequest)
		return
	}

	// Validate job name length (Jenkins job names are typically limited)
	if len(req.Job) > 255 {
		logger.Error("Job name too long", "length", len(req.Job))
		http.Error(w, "Job name exceeds maximum length of 255 characters", http.StatusBadRequest)
		return
	}

	// Validate parameters
	if req.Parameters != nil {
		// Limit number of parameters
		if len(req.Parameters) > 100 {
			logger.Error("Too many parameters", "count", len(req.Parameters))
			http.Error(w, "Maximum 100 parameters allowed", http.StatusBadRequest)
			return
		}

		// Validate parameter keys and values
		for key, value := range req.Parameters {
			// Validate parameter key length
			if len(key) > 255 {
				logger.Error("Parameter key too long", "key", key, "length", len(key))
				http.Error(w, fmt.Sprintf("Parameter key '%s' exceeds maximum length of 255 characters", key), http.StatusBadRequest)
				return
			}

			// Validate parameter value length (limit to 10KB per parameter)
			if len(value) > 10240 {
				logger.Error("Parameter value too long", "key", key, "length", len(value))
				http.Error(w, fmt.Sprintf("Parameter value for '%s' exceeds maximum length of 10KB", key), http.StatusBadRequest)
				return
			}
		}
	}

	// Trigger the build
	result, err := h.jenkinsEngine.TriggerBuild(req.Job, req.Parameters)
	if err != nil {
		logger.Error("Failed to trigger Jenkins build", "error", err, "job", req.Job)

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
		storage.InsertAuditLog(auditLog)

		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(result)
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
	storage.InsertAuditLog(auditLog)

	// Return the result
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(result)
}

// marshalParams marshals parameters to a JSON string
func marshalParams(params map[string]string) string {
	jsonParams, err := json.Marshal(params)
	if err != nil {
		return "{}"
	}
	return string(jsonParams)
}
