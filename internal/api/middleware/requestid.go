package middleware

import (
	"context"
	crypto_rand "crypto/rand"
	"encoding/hex"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"time"

	"triggermesh/internal/logger"
)

// RequestIDContextKey is the context key for the request ID
const RequestIDContextKey ContextKey = "request_id"

// GetRequestID extracts the request ID from the request context
func GetRequestID(r *http.Request) string {
	if requestID, ok := r.Context().Value(RequestIDContextKey).(string); ok {
		return requestID
	}
	return ""
}

// generateRequestID generates a unique request ID
func generateRequestID() string {
	bytes := make([]byte, 16)
	if _, err := crypto_rand.Read(bytes); err != nil {
		// Log error when crypto/rand fails - this indicates a system-level issue
		logger.Error("crypto/rand failed, using fallback request ID",
			"error", err,
			"fallback_reason", "system_crypto_unavailable")

		// Better fallback: combine timestamp with process ID and random component for uniqueness
		// Use nanosecond timestamp + process ID + random int for uniqueness
		//nolint:gosec // G404: math/rand is acceptable here as fallback when crypto/rand fails
		// We combine it with timestamp and process ID for sufficient uniqueness
		fallbackRand := rand.New(rand.NewSource(time.Now().UnixNano()))
		return fmt.Sprintf("req-%d-%d-%d",
			time.Now().UnixNano(),
			os.Getpid(),
			fallbackRand.Int63())
	}
	return hex.EncodeToString(bytes)
}

// RequestIDMiddleware adds a unique request ID to each request
func RequestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Generate or get request ID from header
		requestID := r.Header.Get("X-Request-ID")
		if requestID == "" {
			requestID = generateRequestID()
		}

		// Add request ID to context
		ctx := r.Context()
		ctx = context.WithValue(ctx, RequestIDContextKey, requestID)
		r = r.WithContext(ctx)

		// Add request ID to response header
		w.Header().Set("X-Request-ID", requestID)

		// Log request with request ID
		logger.Info("Request received", "request_id", requestID, "method", r.Method, "path", r.URL.Path, "ip", r.RemoteAddr)

		// Call the next handler
		next.ServeHTTP(w, r)
	})
}
