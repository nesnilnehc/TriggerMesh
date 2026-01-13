package middleware

import (
	"context"
	"net/http"
	"strings"

	"triggermesh/internal/config"
	"triggermesh/internal/logger"
)

// AuthMiddleware is an HTTP middleware that validates API keys
type AuthMiddleware struct {
	apiKeys map[string]bool
}

// NewAuthMiddleware creates a new AuthMiddleware instance
func NewAuthMiddleware(cfg config.APIConfig) *AuthMiddleware {
	// Convert API keys slice to map for O(1) lookups
	apiKeys := make(map[string]bool)
	for _, key := range cfg.Keys {
		apiKeys[key] = true
	}

	return &AuthMiddleware{
		apiKeys: apiKeys,
	}
}

// ValidateAPIKey returns true if the API key is valid
func (am *AuthMiddleware) ValidateAPIKey(apiKey string) bool {
	// Remove Bearer prefix if present
	apiKey = strings.TrimPrefix(apiKey, "Bearer ")
	apiKey = strings.TrimSpace(apiKey)

	// Check if the API key is in the map
	_, ok := am.apiKeys[apiKey]
	return ok
}

// GetAPIKey extracts the API key from the request
func GetAPIKey(r *http.Request) string {
	// Try to get API key from Authorization header
	if authHeader := r.Header.Get("Authorization"); authHeader != "" {
		// Remove Bearer prefix if present
		return strings.TrimPrefix(authHeader, "Bearer ")
	}

	// Try to get API key from query parameter
	if apiKey := r.URL.Query().Get("api_key"); apiKey != "" {
		return apiKey
	}

	return ""
}

// Middleware returns an HTTP handler that validates API keys
func (am *AuthMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get the API key from the request
		apiKey := GetAPIKey(r)

		// Validate the API key
		if !am.ValidateAPIKey(apiKey) {
			logger.Warn("Invalid API key", "ip", r.RemoteAddr, "path", r.URL.Path)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Add the API key to the request context for later use
		ctx := r.Context()
		ctx = context.WithValue(ctx, "api_key", apiKey)
		r = r.WithContext(ctx)

		// Call the next handler
		next.ServeHTTP(w, r)
	})
}
