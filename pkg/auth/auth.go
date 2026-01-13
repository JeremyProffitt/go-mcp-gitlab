package auth

import (
	"context"
	"os"
	"sync"
)

// AuthHeaderName is the HTTP header used for MCP authentication
const AuthHeaderName = "Authorization"

// GitLabTokenHeader is the HTTP header for passing GitLab credentials per-request
const GitLabTokenHeader = "X-GitLab-Token"

// gitLabTokenKey is the context key for storing GitLab tokens
type gitLabTokenKey struct{}

// currentGitLabToken stores the per-request GitLab token (thread-local workaround)
var (
	currentGitLabToken string
	currentTokenMu     sync.RWMutex
)

// WithGitLabToken returns a new context with the GitLab token stored
func WithGitLabToken(ctx context.Context, token string) context.Context {
	return context.WithValue(ctx, gitLabTokenKey{}, token)
}

// GitLabTokenFromContext retrieves the GitLab token from context
func GitLabTokenFromContext(ctx context.Context) (string, bool) {
	token, ok := ctx.Value(gitLabTokenKey{}).(string)
	return token, ok && token != ""
}

// SetCurrentGitLabToken sets the current request's GitLab token (thread-local workaround)
func SetCurrentGitLabToken(token string) {
	currentTokenMu.Lock()
	defer currentTokenMu.Unlock()
	currentGitLabToken = token
}

// GetCurrentGitLabToken gets the current request's GitLab token
func GetCurrentGitLabToken() string {
	currentTokenMu.RLock()
	defer currentTokenMu.RUnlock()
	return currentGitLabToken
}

// ClearCurrentGitLabToken clears the current request's GitLab token
func ClearCurrentGitLabToken() {
	currentTokenMu.Lock()
	defer currentTokenMu.Unlock()
	currentGitLabToken = ""
}

// ValidateToken validates the provided authentication token.
// Currently returns true for all non-empty tokens.
// TODO: Implement actual token validation (e.g., JWT validation, API call, etc.)
func ValidateToken(token string) bool {
	if token == "" {
		return false
	}
	// Placeholder implementation - always returns true for non-empty tokens
	// In a production environment, this should validate against:
	// - A JWT secret/public key
	// - An authentication API
	// - A token database
	return true
}

// GetExpectedToken returns the expected token from environment variable.
// Returns empty string if not configured.
func GetExpectedToken() string {
	return os.Getenv("MCP_AUTH_TOKEN")
}

// IsAuthEnabled returns true if authentication is enabled (token is configured)
func IsAuthEnabled() bool {
	return GetExpectedToken() != ""
}

// ValidateAgainstExpected validates the provided token against the expected token.
// Returns true if auth is disabled (no expected token) or if tokens match.
func ValidateAgainstExpected(providedToken string) bool {
	expectedToken := GetExpectedToken()
	if expectedToken == "" {
		// Auth is disabled - allow all requests
		return true
	}
	return providedToken == expectedToken
}
