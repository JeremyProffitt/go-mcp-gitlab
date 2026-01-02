package tools

import (
	"fmt"
	"net/url"

	"github.com/go-mcp-gitlab/go-mcp-gitlab/pkg/gitlab"
	"github.com/go-mcp-gitlab/go-mcp-gitlab/pkg/mcp"
)

// NamespaceExistsResponse represents the response from the namespace exists API.
type NamespaceExistsResponse struct {
	Exists   bool   `json:"exists"`
	Suggests []string `json:"suggests,omitempty"`
}

// registerListNamespaces registers the list_namespaces tool
func registerListNamespaces(server *mcp.Server) {
	server.RegisterTool(
		mcp.Tool{
			Name:        "list_namespaces",
			Description: "List all namespaces (groups and user namespaces) accessible to the authenticated user",
			InputSchema: mcp.JSONSchema{
				Type: "object",
				Properties: map[string]mcp.Property{
					"page": {
						Type:        "integer",
						Description: "Page number for pagination (default: 1)",
					},
					"per_page": {
						Type:        "integer",
						Description: "Number of items per page (default: 20, max: 100)",
					},
					"search": {
						Type:        "string",
						Description: "Search term to filter namespaces by name or path",
					},
				},
			},
		},
		func(args map[string]interface{}) (*mcp.CallToolResult, error) {
			c := GetContext()
			if c == nil {
				return ErrorResult("tool context not initialized")
			}
			c.Logger.ToolCall("list_namespaces", args)

			params := url.Values{}

			if page := GetInt(args, "page", 0); page > 0 {
				params.Set("page", fmt.Sprintf("%d", page))
			}
			if perPage := GetInt(args, "per_page", 0); perPage > 0 {
				params.Set("per_page", fmt.Sprintf("%d", perPage))
			}
			if search := GetString(args, "search", ""); search != "" {
				params.Set("search", search)
			}

			endpoint := "/namespaces"
			if len(params) > 0 {
				endpoint = fmt.Sprintf("/namespaces?%s", params.Encode())
			}

			var namespaces []gitlab.Namespace
			if err := c.Client.Get(endpoint, &namespaces); err != nil {
				return ErrorResult(fmt.Sprintf("Failed to list namespaces: %v", err))
			}

			return JSONResult(namespaces)
		},
	)
}

// registerGetNamespace registers the get_namespace tool
func registerGetNamespace(server *mcp.Server) {
	server.RegisterTool(
		mcp.Tool{
			Name:        "get_namespace",
			Description: "Get details of a specific namespace by ID or path",
			InputSchema: mcp.JSONSchema{
				Type: "object",
				Properties: map[string]mcp.Property{
					"namespace_id": {
						Type:        "string",
						Description: "The ID or URL-encoded path of the namespace",
					},
				},
				Required: []string{"namespace_id"},
			},
		},
		func(args map[string]interface{}) (*mcp.CallToolResult, error) {
			c := GetContext()
			if c == nil {
				return ErrorResult("tool context not initialized")
			}
			c.Logger.ToolCall("get_namespace", args)

			namespaceID := GetString(args, "namespace_id", "")
			if namespaceID == "" {
				return ErrorResult("namespace_id is required")
			}

			endpoint := fmt.Sprintf("/namespaces/%s", url.PathEscape(namespaceID))

			var namespace gitlab.Namespace
			if err := c.Client.Get(endpoint, &namespace); err != nil {
				return ErrorResult(fmt.Sprintf("Failed to get namespace: %v", err))
			}

			return JSONResult(namespace)
		},
	)
}

// registerVerifyNamespace registers the verify_namespace tool
func registerVerifyNamespace(server *mcp.Server) {
	server.RegisterTool(
		mcp.Tool{
			Name:        "verify_namespace",
			Description: "Check if a namespace path exists. Returns exists: true/false and suggestions if the path is taken",
			InputSchema: mcp.JSONSchema{
				Type: "object",
				Properties: map[string]mcp.Property{
					"namespace_path": {
						Type:        "string",
						Description: "The namespace path to verify",
					},
				},
				Required: []string{"namespace_path"},
			},
		},
		func(args map[string]interface{}) (*mcp.CallToolResult, error) {
			c := GetContext()
			if c == nil {
				return ErrorResult("tool context not initialized")
			}
			c.Logger.ToolCall("verify_namespace", args)

			namespacePath := GetString(args, "namespace_path", "")
			if namespacePath == "" {
				return ErrorResult("namespace_path is required")
			}

			endpoint := fmt.Sprintf("/namespaces/%s/exists", url.PathEscape(namespacePath))

			var response NamespaceExistsResponse
			if err := c.Client.Get(endpoint, &response); err != nil {
				return ErrorResult(fmt.Sprintf("Failed to verify namespace: %v", err))
			}

			return JSONResult(response)
		},
	)
}

// RegisterNamespaceTools registers all namespace-related tools with the MCP server.
// Includes: list_namespaces, get_namespace, verify_namespace
func initNamespaceTools(server *mcp.Server) {
	registerListNamespaces(server)
	registerGetNamespace(server)
	registerVerifyNamespace(server)
}
