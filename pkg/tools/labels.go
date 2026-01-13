// Package tools provides MCP tool implementations for GitLab operations.
package tools

import (
	"fmt"
	"net/url"
	"strconv"

	"github.com/go-mcp-gitlab/go-mcp-gitlab/pkg/mcp"
)

// Label represents a GitLab project label.
type Label struct {
	ID                     int    `json:"id"`
	Name                   string `json:"name"`
	Color                  string `json:"color"`
	TextColor              string `json:"text_color"`
	Description            string `json:"description"`
	DescriptionHTML        string `json:"description_html"`
	OpenIssuesCount        int    `json:"open_issues_count"`
	ClosedIssuesCount      int    `json:"closed_issues_count"`
	OpenMergeRequestsCount int    `json:"open_merge_requests_count"`
	Subscribed             bool   `json:"subscribed"`
	Priority               *int   `json:"priority"`
	IsProjectLabel         bool   `json:"is_project_label"`
}

// registerListLabels registers the list_labels tool.
func registerListLabels(server *mcp.Server) {
	server.RegisterTool(
		mcp.Tool{
			Name:        "list_labels",
			Description: "List all labels for a GitLab project. Returns a paginated list of labels with optional filtering options.",
			InputSchema: mcp.JSONSchema{
				Type: "object",
				Properties: map[string]mcp.Property{
					"project_id": {
						Type:        "string",
						Description: "The project identifier - either a numeric ID (e.g., 42) or URL-encoded path (e.g., my-group/my-project)",
					},
					"page": {
						Type:        "integer",
						Description: "Page number for pagination",
						Default:     1,
						Minimum:     mcp.IntPtr(1),
					},
					"per_page": {
						Type:        "integer",
						Description: "Number of items per page",
						Default:     20,
						Minimum:     mcp.IntPtr(1),
						Maximum:     mcp.IntPtr(100),
					},
					"with_counts": {
						Type:        "boolean",
						Description: "Whether or not to include issue and merge request counts (default: false)",
					},
					"include_ancestor_groups": {
						Type:        "boolean",
						Description: "Include ancestor groups' labels (default: true)",
					},
				},
				Required: []string{"project_id"},
			},
		},
		func(args map[string]interface{}) (*mcp.CallToolResult, error) {
			ctx := GetContext()
			if ctx == nil {
				return ErrorResult("tool context not initialized")
			}
			ctx.Logger.ToolCall("list_labels", args)

			projectID := GetString(args, "project_id", "")
			if projectID == "" {
				return ErrorResult("project_id is required")
			}

			// Build query parameters
			params := url.Values{}

			if page := GetInt(args, "page", 0); page > 0 {
				params.Set("page", strconv.Itoa(page))
			}

			if perPage := GetInt(args, "per_page", 0); perPage > 0 {
				params.Set("per_page", strconv.Itoa(perPage))
			}

			if withCounts, exists := args["with_counts"]; exists {
				if boolVal, ok := withCounts.(bool); ok {
					params.Set("with_counts", strconv.FormatBool(boolVal))
				}
			}

			if includeAncestorGroups, exists := args["include_ancestor_groups"]; exists {
				if boolVal, ok := includeAncestorGroups.(bool); ok {
					params.Set("include_ancestor_groups", strconv.FormatBool(boolVal))
				}
			}

			endpoint := fmt.Sprintf("/projects/%s/labels", url.PathEscape(projectID))
			if len(params) > 0 {
				endpoint += "?" + params.Encode()
			}

			var labels []Label
			if err := ctx.Client.Get(endpoint, &labels); err != nil {
				return ErrorResult(fmt.Sprintf("failed to list labels: %v", err))
			}

			return JSONResult(labels)
		},
	)
}

// registerGetLabel registers the get_label tool.
func registerGetLabel(server *mcp.Server) {
	server.RegisterTool(
		mcp.Tool{
			Name:        "get_label",
			Description: "Get details of a specific label in a GitLab project.",
			InputSchema: mcp.JSONSchema{
				Type: "object",
				Properties: map[string]mcp.Property{
					"project_id": {
						Type:        "string",
						Description: "The project identifier - either a numeric ID (e.g., 42) or URL-encoded path (e.g., my-group/my-project)",
					},
					"label_id": {
						Type:        "string",
						Description: "The ID or name of the label",
					},
				},
				Required: []string{"project_id", "label_id"},
			},
			Annotations: &mcp.ToolAnnotations{
				ReadOnlyHint: true,
			},
		},
		func(args map[string]interface{}) (*mcp.CallToolResult, error) {
			ctx := GetContext()
			if ctx == nil {
				return ErrorResult("tool context not initialized")
			}
			ctx.Logger.ToolCall("get_label", args)

			projectID := GetString(args, "project_id", "")
			if projectID == "" {
				return ErrorResult("project_id is required")
			}

			labelID := GetString(args, "label_id", "")
			if labelID == "" {
				return ErrorResult("label_id is required")
			}

			endpoint := fmt.Sprintf("/projects/%s/labels/%s",
				url.PathEscape(projectID),
				url.PathEscape(labelID),
			)

			var label Label
			if err := ctx.Client.Get(endpoint, &label); err != nil {
				return ErrorResult(fmt.Sprintf("failed to get label: %v", err))
			}

			return JSONResult(label)
		},
	)
}

// registerCreateLabel registers the create_label tool.
func registerCreateLabel(server *mcp.Server) {
	server.RegisterTool(
		mcp.Tool{
			Name:        "create_label",
			Description: "Create a new label in a GitLab project.",
			InputSchema: mcp.JSONSchema{
				Type: "object",
				Properties: map[string]mcp.Property{
					"project_id": {
						Type:        "string",
						Description: "The project identifier - either a numeric ID (e.g., 42) or URL-encoded path (e.g., my-group/my-project)",
					},
					"name": {
						Type:        "string",
						Description: "The name of the label",
					},
					"color": {
						Type:        "string",
						Description: "The color of the label in hex format (e.g., #FF0000)",
					},
					"description": {
						Type:        "string",
						Description: "The description of the label",
					},
					"priority": {
						Type:        "integer",
						Description: "The priority of the label. Must be greater than or equal to 0. Null to remove priority.",
					},
				},
				Required: []string{"project_id", "name", "color"},
			},
		},
		func(args map[string]interface{}) (*mcp.CallToolResult, error) {
			ctx := GetContext()
			if ctx == nil {
				return ErrorResult("tool context not initialized")
			}
			ctx.Logger.ToolCall("create_label", args)

			projectID := GetString(args, "project_id", "")
			if projectID == "" {
				return ErrorResult("project_id is required")
			}

			name := GetString(args, "name", "")
			if name == "" {
				return ErrorResult("name is required")
			}

			color := GetString(args, "color", "")
			if color == "" {
				return ErrorResult("color is required")
			}

			// Build request body
			body := map[string]interface{}{
				"name":  name,
				"color": color,
			}

			if description := GetString(args, "description", ""); description != "" {
				body["description"] = description
			}

			if priority, exists := args["priority"]; exists {
				if priorityVal := GetInt(args, "priority", -1); priorityVal >= 0 {
					body["priority"] = priority
				}
			}

			endpoint := fmt.Sprintf("/projects/%s/labels", url.PathEscape(projectID))

			var label Label
			if err := ctx.Client.Post(endpoint, body, &label); err != nil {
				return ErrorResult(fmt.Sprintf("failed to create label: %v", err))
			}

			return JSONResult(label)
		},
	)
}

// registerUpdateLabel registers the update_label tool.
func registerUpdateLabel(server *mcp.Server) {
	server.RegisterTool(
		mcp.Tool{
			Name:        "update_label",
			Description: "Update an existing label in a GitLab project.",
			InputSchema: mcp.JSONSchema{
				Type: "object",
				Properties: map[string]mcp.Property{
					"project_id": {
						Type:        "string",
						Description: "The project identifier - either a numeric ID (e.g., 42) or URL-encoded path (e.g., my-group/my-project)",
					},
					"label_id": {
						Type:        "string",
						Description: "The ID or name of the label to update",
					},
					"new_name": {
						Type:        "string",
						Description: "The new name of the label",
					},
					"color": {
						Type:        "string",
						Description: "The new color of the label in hex format (e.g., #FF0000)",
					},
					"description": {
						Type:        "string",
						Description: "The new description of the label",
					},
					"priority": {
						Type:        "integer",
						Description: "The new priority of the label. Must be greater than or equal to 0. Null to remove priority.",
					},
				},
				Required: []string{"project_id", "label_id"},
			},
		},
		func(args map[string]interface{}) (*mcp.CallToolResult, error) {
			ctx := GetContext()
			if ctx == nil {
				return ErrorResult("tool context not initialized")
			}
			ctx.Logger.ToolCall("update_label", args)

			projectID := GetString(args, "project_id", "")
			if projectID == "" {
				return ErrorResult("project_id is required")
			}

			labelID := GetString(args, "label_id", "")
			if labelID == "" {
				return ErrorResult("label_id is required")
			}

			// Build request body with only provided fields
			body := make(map[string]interface{})

			if newName := GetString(args, "new_name", ""); newName != "" {
				body["new_name"] = newName
			}

			if color := GetString(args, "color", ""); color != "" {
				body["color"] = color
			}

			if description, exists := args["description"]; exists {
				body["description"] = description
			}

			if _, exists := args["priority"]; exists {
				priorityVal := GetInt(args, "priority", -1)
				if priorityVal >= 0 {
					body["priority"] = priorityVal
				} else {
					// Allow setting null priority by explicitly setting nil
					body["priority"] = nil
				}
			}

			endpoint := fmt.Sprintf("/projects/%s/labels/%s",
				url.PathEscape(projectID),
				url.PathEscape(labelID),
			)

			var label Label
			if err := ctx.Client.Put(endpoint, body, &label); err != nil {
				return ErrorResult(fmt.Sprintf("failed to update label: %v", err))
			}

			return JSONResult(label)
		},
	)
}

// registerDeleteLabel registers the delete_label tool.
func registerDeleteLabel(server *mcp.Server) {
	server.RegisterTool(
		mcp.Tool{
			Name:        "delete_label",
			Description: "Delete a label from a GitLab project. This action is irreversible.",
			InputSchema: mcp.JSONSchema{
				Type: "object",
				Properties: map[string]mcp.Property{
					"project_id": {
						Type:        "string",
						Description: "The project identifier - either a numeric ID (e.g., 42) or URL-encoded path (e.g., my-group/my-project)",
					},
					"label_id": {
						Type:        "string",
						Description: "The ID or name of the label to delete",
					},
				},
				Required: []string{"project_id", "label_id"},
			},
		},
		func(args map[string]interface{}) (*mcp.CallToolResult, error) {
			ctx := GetContext()
			if ctx == nil {
				return ErrorResult("tool context not initialized")
			}
			ctx.Logger.ToolCall("delete_label", args)

			projectID := GetString(args, "project_id", "")
			if projectID == "" {
				return ErrorResult("project_id is required")
			}

			labelID := GetString(args, "label_id", "")
			if labelID == "" {
				return ErrorResult("label_id is required")
			}

			endpoint := fmt.Sprintf("/projects/%s/labels/%s",
				url.PathEscape(projectID),
				url.PathEscape(labelID),
			)

			if err := ctx.Client.Delete(endpoint); err != nil {
				return ErrorResult(fmt.Sprintf("failed to delete label: %v", err))
			}

			return TextResult(fmt.Sprintf("Label '%s' deleted successfully", labelID))
		},
	)
}

// RegisterLabelToolsImpl registers all label-related tools with the MCP server.
// Includes: list_labels, get_label, create_label, update_label, delete_label
func RegisterLabelToolsImpl(server *mcp.Server) {
	registerListLabels(server)
	registerGetLabel(server)
	registerCreateLabel(server)
	registerUpdateLabel(server)
	registerDeleteLabel(server)
}
