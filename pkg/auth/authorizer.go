package auth

import (
	"context"
	"net/http"
)

// Authorizer is the interface for authentication providers.
type Authorizer interface {
	// Authorize validates the provided token and returns true if authorized.
	Authorize(ctx context.Context, token string) (bool, error)
}

// MockAuthorizer is a mock implementation that always authorizes.
type MockAuthorizer struct{}

// Authorize always returns true for MockAuthorizer.
func (m *MockAuthorizer) Authorize(ctx context.Context, token string) (bool, error) {
	return true, nil
}

// AuthMiddleware creates an HTTP middleware that checks for Authorization header.
// It skips authentication for the /health endpoint.
// If authorizer is nil and no expected token is configured, all requests pass.
func AuthMiddleware(authorizer Authorizer, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip authentication for /health endpoint
		if r.URL.Path == "/health" {
			next.ServeHTTP(w, r)
			return
		}

		// Check if authentication is enabled (either via authorizer or env token)
		if authorizer == nil && !IsAuthEnabled() {
			// No authentication configured, pass through
			next.ServeHTTP(w, r)
			return
		}

		// Get the Authorization header
		token := r.Header.Get("Authorization")
		if token == "" {
			http.Error(w, `{"jsonrpc":"2.0","id":null,"error":{"code":-32001,"message":"Unauthorized: missing Authorization header"}}`, http.StatusUnauthorized)
			return
		}

		// If we have an authorizer, use it
		if authorizer != nil {
			authorized, err := authorizer.Authorize(r.Context(), token)
			if err != nil {
				http.Error(w, `{"jsonrpc":"2.0","id":null,"error":{"code":-32001,"message":"Unauthorized: authorization error"}}`, http.StatusUnauthorized)
				return
			}
			if !authorized {
				http.Error(w, `{"jsonrpc":"2.0","id":null,"error":{"code":-32001,"message":"Unauthorized: invalid token"}}`, http.StatusUnauthorized)
				return
			}
		} else {
			// Fall back to expected token validation from environment
			if !ValidateAgainstExpected(token) {
				http.Error(w, `{"jsonrpc":"2.0","id":null,"error":{"code":-32001,"message":"Unauthorized: invalid token"}}`, http.StatusUnauthorized)
				return
			}
		}

		next.ServeHTTP(w, r)
	})
}
