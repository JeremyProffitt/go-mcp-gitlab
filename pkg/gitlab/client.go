package gitlab

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// HTTPRequestInfo contains HTTP request details for logging
type HTTPRequestInfo struct {
	Method  string
	URL     string
	Headers map[string]string
	Body    string
}

// HTTPResponseInfo contains HTTP response details for logging
type HTTPResponseInfo struct {
	StatusCode int
	Headers    map[string]string
	Body       string
}

// Logger defines the interface for logging API calls.
type Logger interface {
	Access(method, endpoint string, statusCode int, duration time.Duration)
	Debug(msg string, args ...any)
	Error(msg string, args ...any)
	// LogHTTPRequest logs detailed HTTP request information at DEBUG level
	LogHTTPRequest(context string, req *HTTPRequestInfo, secrets ...string)
	// LogHTTPResponse logs detailed HTTP response information at DEBUG level
	LogHTTPResponse(context string, resp *HTTPResponseInfo, duration time.Duration, secrets ...string)
	// LogHTTPError logs detailed HTTP error information
	LogHTTPError(context string, req *HTTPRequestInfo, resp *HTTPResponseInfo, err error, secrets ...string)
}

// TokenProvider is a function that returns the current token to use.
// This allows for dynamic token resolution (e.g., from request headers).
type TokenProvider func() string

// Client is an HTTP client wrapper for the GitLab API.
type Client struct {
	baseURL       string
	token         string
	tokenProvider TokenProvider
	httpClient    *http.Client
	logger        Logger
}

// ClientOption is a function that configures a Client.
type ClientOption func(*Client)

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(httpClient *http.Client) ClientOption {
	return func(c *Client) {
		c.httpClient = httpClient
	}
}

// WithLogger sets a custom logger.
func WithLogger(logger Logger) ClientOption {
	return func(c *Client) {
		c.logger = logger
	}
}

// WithTokenProvider sets a dynamic token provider.
// The provider is called for each request to get the current token.
func WithTokenProvider(provider TokenProvider) ClientOption {
	return func(c *Client) {
		c.tokenProvider = provider
	}
}

// NewClient creates a new GitLab API client.
func NewClient(baseURL, token string, opts ...ClientOption) *Client {
	// Ensure baseURL doesn't have trailing slash
	baseURL = strings.TrimSuffix(baseURL, "/")

	// Ensure baseURL ends with /api/v4
	if !strings.HasSuffix(baseURL, "/api/v4") {
		baseURL = baseURL + "/api/v4"
	}

	c := &Client{
		baseURL: baseURL,
		token:   token,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: &noopLogger{},
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

// getToken returns the current token to use for requests.
// If a TokenProvider is set and returns a non-empty token, it is used.
// Otherwise, the default token is used.
func (c *Client) getToken() string {
	if c.tokenProvider != nil {
		if token := c.tokenProvider(); token != "" {
			return token
		}
	}
	return c.token
}

// Get performs an HTTP GET request to the specified endpoint.
func (c *Client) Get(endpoint string, result interface{}) error {
	return c.request(http.MethodGet, endpoint, nil, result)
}

// GetWithPagination performs an HTTP GET request and returns pagination info.
func (c *Client) GetWithPagination(endpoint string, result interface{}) (*PaginationInfo, error) {
	return c.requestWithPagination(http.MethodGet, endpoint, nil, result)
}

// Post performs an HTTP POST request to the specified endpoint.
func (c *Client) Post(endpoint string, body, result interface{}) error {
	return c.request(http.MethodPost, endpoint, body, result)
}

// Put performs an HTTP PUT request to the specified endpoint.
func (c *Client) Put(endpoint string, body, result interface{}) error {
	return c.request(http.MethodPut, endpoint, body, result)
}

// Delete performs an HTTP DELETE request to the specified endpoint.
func (c *Client) Delete(endpoint string) error {
	return c.request(http.MethodDelete, endpoint, nil, nil)
}

// GetText performs an HTTP GET request and returns the response as plain text.
// This is used for endpoints that return text/plain content (e.g., job logs).
func (c *Client) GetText(endpoint string) (string, error) {
	start := time.Now()

	// Build the full URL
	url := c.buildURL(endpoint)

	// Get the effective token for this request
	token := c.getToken()

	// Create the request
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "text/plain")

	// Log request at DEBUG level (token will be masked)
	c.logger.LogHTTPRequest("api_request_text", &HTTPRequestInfo{
		Method: http.MethodGet,
		URL:    url,
		Headers: map[string]string{
			"Authorization": "Bearer " + token,
			"Accept":        "text/plain",
		},
	}, token)

	// Execute the request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.LogHTTPError("http_request_text", &HTTPRequestInfo{
			Method: http.MethodGet,
			URL:    url,
			Headers: map[string]string{
				"Authorization": "Bearer " + token,
				"Accept":        "text/plain",
			},
		}, nil, err, token)
		c.logger.Error("request failed", "method", http.MethodGet, "endpoint", endpoint, "error", err)
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	duration := time.Since(start)

	// Read the response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	// Log response at DEBUG level (body summary for text content)
	c.logger.LogHTTPResponse("api_response_text", &HTTPResponseInfo{
		StatusCode: resp.StatusCode,
		Headers:    convertHeaders(resp.Header),
		Body:       string(respBody),
	}, duration, token)

	c.logger.Access(http.MethodGet, endpoint, resp.StatusCode, duration)

	// Check for errors
	if resp.StatusCode >= 400 {
		c.logger.LogHTTPError("api_error_text", &HTTPRequestInfo{
			Method: http.MethodGet,
			URL:    url,
		}, &HTTPResponseInfo{
			StatusCode: resp.StatusCode,
			Headers:    convertHeaders(resp.Header),
			Body:       string(respBody),
		}, nil, token)
		return "", c.handleErrorResponse(resp.StatusCode, endpoint, respBody)
	}

	return string(respBody), nil
}

// request performs an HTTP request and decodes the response.
func (c *Client) request(method, endpoint string, body interface{}, result interface{}) error {
	_, err := c.requestWithPagination(method, endpoint, body, result)
	return err
}

// requestWithPagination performs an HTTP request and returns pagination info.
func (c *Client) requestWithPagination(method, endpoint string, body interface{}, result interface{}) (*PaginationInfo, error) {
	start := time.Now()

	// Build the full URL
	url := c.buildURL(endpoint)

	// Get the effective token for this request
	token := c.getToken()

	// Prepare the request body
	var bodyReader io.Reader
	var bodyStr string
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyStr = string(jsonBody)
		bodyReader = bytes.NewReader(jsonBody)
	}

	// Create the request
	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	// Log request at DEBUG level (token will be masked)
	c.logger.LogHTTPRequest("api_request", &HTTPRequestInfo{
		Method: method,
		URL:    url,
		Headers: map[string]string{
			"Authorization": "Bearer " + token,
			"Content-Type":  "application/json",
			"Accept":        "application/json",
		},
		Body: bodyStr,
	}, token)

	// Execute the request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.LogHTTPError("http_request", &HTTPRequestInfo{
			Method: method,
			URL:    url,
			Headers: map[string]string{
				"Authorization": "Bearer " + token,
				"Content-Type":  "application/json",
			},
			Body: bodyStr,
		}, nil, err, token)
		c.logger.Error("request failed", "method", method, "endpoint", endpoint, "error", err)
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	duration := time.Since(start)

	// Read the response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Log response at DEBUG level
	c.logger.LogHTTPResponse("api_response", &HTTPResponseInfo{
		StatusCode: resp.StatusCode,
		Headers:    convertHeaders(resp.Header),
		Body:       string(respBody),
	}, duration, token)

	c.logger.Access(method, endpoint, resp.StatusCode, duration)

	// Check for errors
	if resp.StatusCode >= 400 {
		c.logger.LogHTTPError("api_error", &HTTPRequestInfo{
			Method: method,
			URL:    url,
			Body:   bodyStr,
		}, &HTTPResponseInfo{
			StatusCode: resp.StatusCode,
			Headers:    convertHeaders(resp.Header),
			Body:       string(respBody),
		}, nil, token)
		return nil, c.handleErrorResponse(resp.StatusCode, endpoint, respBody)
	}

	// Parse pagination headers
	pagination := c.parsePaginationHeaders(resp.Header)

	// Decode the response
	if result != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, result); err != nil {
			c.logger.Debug("failed to unmarshal response", "body", string(respBody), "error", err)
			return nil, fmt.Errorf("failed to unmarshal response: %w", err)
		}
	}

	return pagination, nil
}

// buildURL constructs the full URL for an API endpoint.
func (c *Client) buildURL(endpoint string) string {
	// Ensure endpoint starts with /
	if !strings.HasPrefix(endpoint, "/") {
		endpoint = "/" + endpoint
	}
	return c.baseURL + endpoint
}

// handleErrorResponse creates an APIError from an error response.
func (c *Client) handleErrorResponse(statusCode int, endpoint string, body []byte) *APIError {
	apiErr := &APIError{
		StatusCode: statusCode,
		Endpoint:   endpoint,
	}

	// Try to parse the error message from the response
	var errResp struct {
		Message string   `json:"message"`
		Error   string   `json:"error"`
		Errors  []string `json:"errors"`
	}

	if err := json.Unmarshal(body, &errResp); err == nil {
		if errResp.Message != "" {
			apiErr.Message = errResp.Message
		} else if errResp.Error != "" {
			apiErr.Message = errResp.Error
		} else if len(errResp.Errors) > 0 {
			apiErr.Message = strings.Join(errResp.Errors, "; ")
		}
	}

	// Set default message if none was found
	if apiErr.Message == "" {
		apiErr.Message = http.StatusText(statusCode)
	}

	return apiErr
}

// convertHeaders converts http.Header to map[string]string for logging.
// All headers are included for debugging purposes.
func convertHeaders(headers http.Header) map[string]string {
	headersMap := make(map[string]string)
	for key, values := range headers {
		if len(values) > 0 {
			headersMap[key] = values[0]
		}
	}
	return headersMap
}

// parsePaginationHeaders extracts pagination information from response headers.
func (c *Client) parsePaginationHeaders(headers http.Header) *PaginationInfo {
	pagination := &PaginationInfo{}

	if page := headers.Get("X-Page"); page != "" {
		pagination.Page, _ = strconv.Atoi(page)
	}
	if perPage := headers.Get("X-Per-Page"); perPage != "" {
		pagination.PerPage, _ = strconv.Atoi(perPage)
	}
	if total := headers.Get("X-Total"); total != "" {
		pagination.Total, _ = strconv.Atoi(total)
	}
	if totalPages := headers.Get("X-Total-Pages"); totalPages != "" {
		pagination.TotalPages, _ = strconv.Atoi(totalPages)
	}
	if nextPage := headers.Get("X-Next-Page"); nextPage != "" {
		pagination.NextPage, _ = strconv.Atoi(nextPage)
	}
	if prevPage := headers.Get("X-Prev-Page"); prevPage != "" {
		pagination.PrevPage, _ = strconv.Atoi(prevPage)
	}

	return pagination
}

// BaseURL returns the base URL of the client.
func (c *Client) BaseURL() string {
	return c.baseURL
}

// noopLogger is a no-op implementation of the Logger interface.
type noopLogger struct{}

func (l *noopLogger) Access(method, endpoint string, statusCode int, duration time.Duration)    {}
func (l *noopLogger) Debug(msg string, args ...any)                                             {}
func (l *noopLogger) Error(msg string, args ...any)                                             {}
func (l *noopLogger) LogHTTPRequest(context string, req *HTTPRequestInfo, secrets ...string)    {}
func (l *noopLogger) LogHTTPResponse(context string, resp *HTTPResponseInfo, duration time.Duration, secrets ...string) {
}
func (l *noopLogger) LogHTTPError(context string, req *HTTPRequestInfo, resp *HTTPResponseInfo, err error, secrets ...string) {
}
