package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"triggermesh/internal/api/handlers"
	"triggermesh/internal/api/middleware"
	"triggermesh/internal/config"
	"triggermesh/internal/engine"
	"triggermesh/internal/logger"
	"triggermesh/internal/storage"
)

// Router represents the API router
type Router struct {
	mux            *http.ServeMux
	allowedOrigins []string
	maxBodySize    int64
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
		w.Header().Set("Content-Type", "application/json")

		// Check database connection
		if err := storage.Ping(); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			if encodeErr := json.NewEncoder(w).Encode(map[string]interface{}{
				"status": "unhealthy",
				"error":  "database connection failed",
			}); encodeErr != nil {
				logger.Error("Failed to encode health check error", "error", encodeErr)
			}
			return
		}

		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "healthy",
		}); err != nil {
			logger.Error("Failed to encode health check response", "error", err)
		}
	})

	// Protected routes
	// Jenkins routes
	mux.Handle("/api/v1/trigger/jenkins", authMiddleware.Middleware(http.HandlerFunc(jenkinsHandler.TriggerJenkinsBuild)))

	// Audit routes
	mux.Handle("/api/v1/audit", authMiddleware.Middleware(http.HandlerFunc(auditHandler.GetAuditLogs)))

	return &Router{
		mux:            mux,
		allowedOrigins: cfg.Server.AllowedOrigins,
		maxBodySize:    cfg.Server.MaxBodySize,
	}
}

// ServeHTTP implements the http.Handler interface
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	// Chain middleware: RequestID -> BodySizeLimit -> CORS -> Mux
	handler := chainMiddleware(
		http.HandlerFunc(r.mux.ServeHTTP),
		middleware.RequestIDMiddleware,
		middleware.LimitBodySize(r.maxBodySize),
		r.corsMiddleware,
	)
	handler.ServeHTTP(w, req)
}

// chainMiddleware chains multiple middleware functions together
func chainMiddleware(handler http.Handler, middlewares ...func(http.Handler) http.Handler) http.Handler {
	for i := len(middlewares) - 1; i >= 0; i-- {
		handler = middlewares[i](handler)
	}
	return handler
}

// corsMiddleware handles CORS headers and preflight requests
func (r *Router) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		origin := req.Header.Get("Origin")

		// Handle CORS based on configuration
		if len(r.allowedOrigins) == 0 {
			// Empty allowed origins means allow all (backward compatibility)
			w.Header().Set("Access-Control-Allow-Origin", "*")
		} else if origin != "" {
			// Validate origin format and check if it's allowed
			if !r.isValidOrigin(origin) {
				// Invalid origin format - reject
				logger.Warn("Invalid origin format", "origin", origin, "request_id", middleware.GetRequestID(req))
				// Don't set CORS headers, but continue processing (same-origin requests don't send Origin header)
			} else if r.isOriginAllowed(origin) {
				// Origin is valid and in the allowed list
				w.Header().Set("Access-Control-Allow-Origin", origin)
			} else {
				// Origin not in allowed list - reject CORS but allow same-origin requests
				logger.Warn("Origin not allowed", "origin", origin, "request_id", middleware.GetRequestID(req))
				// Don't set CORS headers
			}
		}
		// If origin is empty, it's a same-origin request (browsers don't send Origin header for same-origin)
		// Allow it to proceed without CORS headers

		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		// Handle OPTIONS requests for CORS preflight
		if req.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		// Call the next handler
		next.ServeHTTP(w, req)
	})
}

// isValidOrigin validates the origin format (must be http:// or https://)
func (r *Router) isValidOrigin(origin string) bool {
	return strings.HasPrefix(origin, "http://") || strings.HasPrefix(origin, "https://")
}

// isOriginAllowed checks if the given origin is in the allowed list
func (r *Router) isOriginAllowed(origin string) bool {
	for _, allowed := range r.allowedOrigins {
		if strings.EqualFold(origin, allowed) {
			return true
		}
	}
	return false
}
