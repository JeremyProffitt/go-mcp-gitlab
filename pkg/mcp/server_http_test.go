package mcp

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-mcp-gitlab/go-mcp-gitlab/pkg/auth"
)

// createTestHandler creates an HTTP handler for the MCP server for testing purposes.
// This mirrors the internal setup in RunHTTPWithAuthorizer but allows for test server usage.
func createTestHandler(s *Server, authorizer auth.Authorizer) http.Handler {
	mux := http.NewServeMux()

	// Health check endpoint (no auth required)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"status":  "ok",
			"version": s.version,
		})
	})

	// MCP endpoint handler
	mcpHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}

		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      nil,
				"error":   map[string]interface{}{"code": -32700, "message": "Parse error"},
			})
			return
		}

		response := s.handleMessageWithContext(r, body)
		if response != nil {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
		}
	})

	// Apply auth middleware
	mux.Handle("/", auth.AuthMiddleware(authorizer, mcpHandler))

	return mux
}

func TestHTTPHealthEndpoint(t *testing.T) {
	server := NewServer("test-server", "1.0.0")

	ts := httptest.NewServer(createTestHandler(server, nil))
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/health")
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// Check content type
	contentType := resp.Header.Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", contentType)
	}

	// Parse response body
	var result map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Check response fields
	if result["status"] != "ok" {
		t.Errorf("Expected status 'ok', got %q", result["status"])
	}
	if result["version"] != "1.0.0" {
		t.Errorf("Expected version '1.0.0', got %q", result["version"])
	}
}

func TestHTTPAuthMiddleware_MissingHeader(t *testing.T) {
	server := NewServer("test-server", "1.0.0")

	// Use MockAuthorizer to enable authentication requirement
	ts := httptest.NewServer(createTestHandler(server, &auth.MockAuthorizer{}))
	defer ts.Close()

	// Make POST request without Authorization header
	resp, err := http.Post(ts.URL+"/", "application/json", bytes.NewReader([]byte(`{}`)))
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	// Should return 401 Unauthorized
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", resp.StatusCode)
	}

	// Check that error message is returned
	body, _ := io.ReadAll(resp.Body)
	if !bytes.Contains(body, []byte("Unauthorized")) {
		t.Errorf("Expected Unauthorized message in body, got: %s", string(body))
	}
}

func TestHTTPAuthMiddleware_WithHeader(t *testing.T) {
	server := NewServer("test-server", "1.0.0")

	// Use MockAuthorizer which always authorizes
	ts := httptest.NewServer(createTestHandler(server, &auth.MockAuthorizer{}))
	defer ts.Close()

	// Create request with Authorization header
	reqBody := []byte(`{"jsonrpc":"2.0","id":1,"method":"ping"}`)
	req, err := http.NewRequest(http.MethodPost, ts.URL+"/", bytes.NewReader(reqBody))
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test-token")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	// Should return 200 OK (request proceeds with MockAuthorizer)
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Errorf("Expected status 200, got %d. Body: %s", resp.StatusCode, string(body))
	}
}

func TestHTTPMCPInitialize(t *testing.T) {
	server := NewServer("test-gitlab-server", "2.0.0")
	server.SetInstructions("Test instructions for the server")

	// No authorizer - auth disabled
	ts := httptest.NewServer(createTestHandler(server, nil))
	defer ts.Close()

	// Send initialize request
	initRequest := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "initialize",
		Params: map[string]interface{}{
			"protocolVersion": "2024-11-05",
			"capabilities":    map[string]interface{}{},
			"clientInfo": map[string]interface{}{
				"name":    "test-client",
				"version": "1.0.0",
			},
		},
	}

	reqBody, err := json.Marshal(initRequest)
	if err != nil {
		t.Fatalf("Failed to marshal request: %v", err)
	}

	resp, err := http.Post(ts.URL+"/", "application/json", bytes.NewReader(reqBody))
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected status 200, got %d. Body: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var rpcResponse JSONRPCResponse
	if err := json.NewDecoder(resp.Body).Decode(&rpcResponse); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Check response structure
	if rpcResponse.JSONRPC != "2.0" {
		t.Errorf("Expected jsonrpc '2.0', got %q", rpcResponse.JSONRPC)
	}

	if rpcResponse.Error != nil {
		t.Fatalf("Unexpected error in response: %+v", rpcResponse.Error)
	}

	// Parse the result as InitializeResult
	resultMap, ok := rpcResponse.Result.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected result to be a map, got %T", rpcResponse.Result)
	}

	// Check protocol version
	if pv, ok := resultMap["protocolVersion"].(string); !ok || pv != "2024-11-05" {
		t.Errorf("Expected protocolVersion '2024-11-05', got %v", resultMap["protocolVersion"])
	}

	// Check server info
	serverInfo, ok := resultMap["serverInfo"].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected serverInfo to be a map, got %T", resultMap["serverInfo"])
	}
	if serverInfo["name"] != "test-gitlab-server" {
		t.Errorf("Expected server name 'test-gitlab-server', got %v", serverInfo["name"])
	}
	if serverInfo["version"] != "2.0.0" {
		t.Errorf("Expected server version '2.0.0', got %v", serverInfo["version"])
	}

	// Check instructions
	if instructions, ok := resultMap["instructions"].(string); !ok || instructions != "Test instructions for the server" {
		t.Errorf("Expected instructions 'Test instructions for the server', got %v", resultMap["instructions"])
	}

	// Check capabilities
	capabilities, ok := resultMap["capabilities"].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected capabilities to be a map, got %T", resultMap["capabilities"])
	}
	if capabilities["tools"] == nil {
		t.Error("Expected tools capability to be present")
	}
}

func TestHTTPMCPToolsList(t *testing.T) {
	server := NewServer("test-server", "1.0.0")

	// Register some test tools
	tool1 := Tool{
		Name:        "test_tool_1",
		Description: "First test tool",
		InputSchema: JSONSchema{
			Type: "object",
			Properties: map[string]Property{
				"param1": {Type: "string", Description: "A test parameter"},
			},
			Required: []string{"param1"},
		},
	}
	tool2 := Tool{
		Name:        "test_tool_2",
		Description: "Second test tool",
		InputSchema: JSONSchema{
			Type:       "object",
			Properties: map[string]Property{},
		},
	}

	server.RegisterTool(tool1, func(args map[string]interface{}) (*CallToolResult, error) {
		return &CallToolResult{Content: []ContentItem{{Type: "text", Text: "tool1 result"}}}, nil
	})
	server.RegisterTool(tool2, func(args map[string]interface{}) (*CallToolResult, error) {
		return &CallToolResult{Content: []ContentItem{{Type: "text", Text: "tool2 result"}}}, nil
	})

	// No authorizer - auth disabled
	ts := httptest.NewServer(createTestHandler(server, nil))
	defer ts.Close()

	// Send tools/list request
	listRequest := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      2,
		Method:  "tools/list",
	}

	reqBody, err := json.Marshal(listRequest)
	if err != nil {
		t.Fatalf("Failed to marshal request: %v", err)
	}

	resp, err := http.Post(ts.URL+"/", "application/json", bytes.NewReader(reqBody))
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected status 200, got %d. Body: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var rpcResponse JSONRPCResponse
	if err := json.NewDecoder(resp.Body).Decode(&rpcResponse); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if rpcResponse.Error != nil {
		t.Fatalf("Unexpected error in response: %+v", rpcResponse.Error)
	}

	// Parse the result
	resultMap, ok := rpcResponse.Result.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected result to be a map, got %T", rpcResponse.Result)
	}

	tools, ok := resultMap["tools"].([]interface{})
	if !ok {
		t.Fatalf("Expected tools to be an array, got %T", resultMap["tools"])
	}

	// Check that we have 2 tools
	if len(tools) != 2 {
		t.Errorf("Expected 2 tools, got %d", len(tools))
	}

	// Verify tool names
	toolNames := make(map[string]bool)
	for _, tool := range tools {
		toolMap, ok := tool.(map[string]interface{})
		if !ok {
			continue
		}
		if name, ok := toolMap["name"].(string); ok {
			toolNames[name] = true
		}
	}

	if !toolNames["test_tool_1"] {
		t.Error("Expected test_tool_1 to be in tools list")
	}
	if !toolNames["test_tool_2"] {
		t.Error("Expected test_tool_2 to be in tools list")
	}
}

func TestHTTPMCPToolsCall(t *testing.T) {
	server := NewServer("test-server", "1.0.0")

	// Register a test tool that echoes the input
	echoTool := Tool{
		Name:        "echo",
		Description: "Echoes the input message",
		InputSchema: JSONSchema{
			Type: "object",
			Properties: map[string]Property{
				"message": {Type: "string", Description: "Message to echo"},
			},
			Required: []string{"message"},
		},
	}

	server.RegisterTool(echoTool, func(args map[string]interface{}) (*CallToolResult, error) {
		msg, _ := args["message"].(string)
		return &CallToolResult{
			Content: []ContentItem{{Type: "text", Text: "Echo: " + msg}},
		}, nil
	})

	ts := httptest.NewServer(createTestHandler(server, nil))
	defer ts.Close()

	// Send tools/call request
	callRequest := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      3,
		Method:  "tools/call",
		Params: map[string]interface{}{
			"name": "echo",
			"arguments": map[string]interface{}{
				"message": "Hello, World!",
			},
		},
	}

	reqBody, err := json.Marshal(callRequest)
	if err != nil {
		t.Fatalf("Failed to marshal request: %v", err)
	}

	resp, err := http.Post(ts.URL+"/", "application/json", bytes.NewReader(reqBody))
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected status 200, got %d. Body: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var rpcResponse JSONRPCResponse
	if err := json.NewDecoder(resp.Body).Decode(&rpcResponse); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if rpcResponse.Error != nil {
		t.Fatalf("Unexpected error in response: %+v", rpcResponse.Error)
	}

	// Parse the result
	resultMap, ok := rpcResponse.Result.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected result to be a map, got %T", rpcResponse.Result)
	}

	content, ok := resultMap["content"].([]interface{})
	if !ok || len(content) == 0 {
		t.Fatalf("Expected content array with at least one item")
	}

	contentItem, ok := content[0].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected content item to be a map")
	}

	text, ok := contentItem["text"].(string)
	if !ok || text != "Echo: Hello, World!" {
		t.Errorf("Expected 'Echo: Hello, World!', got %v", contentItem["text"])
	}
}

func TestHTTPMethodNotAllowed(t *testing.T) {
	server := NewServer("test-server", "1.0.0")

	ts := httptest.NewServer(createTestHandler(server, nil))
	defer ts.Close()

	// Try GET on root endpoint (should fail)
	resp, err := http.Get(ts.URL + "/")
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", resp.StatusCode)
	}
}

func TestHTTPInvalidJSON(t *testing.T) {
	server := NewServer("test-server", "1.0.0")

	ts := httptest.NewServer(createTestHandler(server, nil))
	defer ts.Close()

	// Send invalid JSON
	resp, err := http.Post(ts.URL+"/", "application/json", bytes.NewReader([]byte(`{invalid json`)))
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	// Should return 200 with JSON-RPC error (parse error)
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var rpcResponse JSONRPCResponse
	if err := json.NewDecoder(resp.Body).Decode(&rpcResponse); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if rpcResponse.Error == nil {
		t.Fatal("Expected error in response")
	}

	if rpcResponse.Error.Code != ParseError {
		t.Errorf("Expected parse error code %d, got %d", ParseError, rpcResponse.Error.Code)
	}
}

func TestHTTPUnknownMethod(t *testing.T) {
	server := NewServer("test-server", "1.0.0")

	ts := httptest.NewServer(createTestHandler(server, nil))
	defer ts.Close()

	// Send request with unknown method
	unknownRequest := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      4,
		Method:  "unknown/method",
	}

	reqBody, _ := json.Marshal(unknownRequest)
	resp, err := http.Post(ts.URL+"/", "application/json", bytes.NewReader(reqBody))
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	var rpcResponse JSONRPCResponse
	if err := json.NewDecoder(resp.Body).Decode(&rpcResponse); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if rpcResponse.Error == nil {
		t.Fatal("Expected error in response")
	}

	if rpcResponse.Error.Code != MethodNotFound {
		t.Errorf("Expected method not found error code %d, got %d", MethodNotFound, rpcResponse.Error.Code)
	}
}
