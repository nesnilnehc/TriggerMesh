package middleware

import (
	"context"
	"net/http"
	"strings"

	"triggermesh/internal/config"
	"triggermesh/internal/logger"
)

// ContextKey is a custom type for context keys to avoid collisions
type ContextKey string

// APIKeyContextKey is the context key for the API key
const APIKeyContextKey ContextKey = "api_key"

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
// Only supports Authorization header for security reasons (query parameters can be logged)
func GetAPIKey(r *http.Request) string {
	// Only get API key from Authorization header
	if authHeader := r.Header.Get("Authorization"); authHeader != "" {
		// Remove Bearer prefix if present
		return strings.TrimPrefix(authHeader, "Bearer ")
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
		ctx = context.WithValue(ctx, APIKeyContextKey, apiKey)
		r = r.WithContext(ctx)

		// Call the next handler
		next.ServeHTTP(w, r)
	})
}
