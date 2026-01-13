package api

import (
	"encoding/json"
	"net/http"

	"triggermesh/internal/api/handlers"
	"triggermesh/internal/api/middleware"
	"triggermesh/internal/config"
	"triggermesh/internal/engine"
	"triggermesh/internal/logger"
)

// Router represents the API router
type Router struct {
	mux *http.ServeMux
}

// NewRouter creates a new Router instance
func NewRouter(
	cfg config.Config,
	jenkinsEngine engine.CIEngine,
) *Router {
	// Create a new ServeMux
	mux := http.NewServeMux()

	// Create handlers
	jenkinsHandler := handlers.NewJenkinsHandler(jenkinsEngine)
	auditHandler := handlers.NewAuditHandler()

	// Create middleware
	authMiddleware := middleware.NewAuthMiddleware(cfg.API)

	// Public routes
	// Root path handler
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]interface{}{
			"message": "TriggerMesh API",
			"version": "1.0.0",
			"endpoints": []string{
				"/health - Health check",
				"/api/v1/trigger/jenkins - Trigger Jenkins build",
				"/api/v1/audit - Get audit logs",
			},
		}); err != nil {
			logger.Error("Failed to encode response", "error", err)
		}
	})

	// Health check
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte("OK")); err != nil {
			logger.Error("Failed to write response", "error", err)
		}
	})

	// Protected routes
	// Jenkins routes
	mux.Handle("/api/v1/trigger/jenkins", authMiddleware.Middleware(http.HandlerFunc(jenkinsHandler.TriggerJenkinsBuild)))

	// Audit routes
	mux.Handle("/api/v1/audit", authMiddleware.Middleware(http.HandlerFunc(auditHandler.GetAuditLogs)))

	return &Router{
		mux: mux,
	}
}

// ServeHTTP implements the http.Handler interface
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	// Set CORS headers
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	// Handle OPTIONS requests for CORS
	if req.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Call the mux to handle the request
	r.mux.ServeHTTP(w, req)
}
