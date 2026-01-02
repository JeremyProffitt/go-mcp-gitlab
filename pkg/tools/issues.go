// Package tools provides MCP tool implementations for GitLab operations.
package tools

import (
	"fmt"
	"net/url"
	"strconv"

	"github.com/go-mcp-gitlab/go-mcp-gitlab/pkg/gitlab"
	"github.com/go-mcp-gitlab/go-mcp-gitlab/pkg/mcp"
)

// IssueLink represents a link between issues.
type IssueLink struct {
	ID            int           `json:"id"`
	SourceIssue   *gitlab.Issue `json:"source_issue"`
	TargetIssue   *gitlab.Issue `json:"target_issue"`
	LinkType      string        `json:"link_type"`
	LinkCreatedAt string        `json:"link_created_at"`
	LinkUpdatedAt string        `json:"link_updated_at"`
}

// Note: Discussion type is defined in merge_requests.go

// registerListIssues registers the list_issues tool.
func registerListIssues(server *mcp.Server) {
	server.RegisterTool(
		mcp.Tool{
			Name:        "list_issues",
			Description: "List issues in a GitLab project. Returns a paginated list of issues with optional filtering by state, labels, milestone, and scope.",
			InputSchema: mcp.JSONSchema{
				Type: "object",
				Properties: map[string]mcp.Property{
					"project_id": {
						Type:        "string",
						Description: "The ID or URL-encoded path of the project",
					},
					"state": {
						Type:        "string",
						Description: "Filter issues by state: opened, closed, or all",
						Enum:        []string{"opened", "closed", "all"},
					},
					"labels": {
						Type:        "string",
						Description: "Comma-separated list of label names to filter by",
					},
					"milestone": {
						Type:        "string",
						Description: "Milestone title to filter by",
					},
					"scope": {
						Type:        "string",
						Description: "Scope of issues: all, assigned_to_me, or created_by_me",
						Enum:        []string{"all", "assigned_to_me", "created_by_me"},
					},
					"page": {
						Type:        "integer",
						Description: "Page number for pagination (default: 1)",
					},
					"per_page": {
						Type:        "integer",
						Description: "Number of items per page (default: 20, max: 100)",
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
			ctx.Logger.ToolCall("list_issues", args)

			projectID := GetString(args, "project_id", "")
			if projectID == "" {
				return ErrorResult("project_id is required")
			}

			// Build query parameters
			params := url.Values{}

			if state := GetString(args, "state", ""); state != "" {
				params.Set("state", state)
			}

			if labels := GetString(args, "labels", ""); labels != "" {
				params.Set("labels", labels)
			}

			if milestone := GetString(args, "milestone", ""); milestone != "" {
				params.Set("milestone", milestone)
			}

			if scope := GetString(args, "scope", ""); scope != "" {
				params.Set("scope", scope)
			}

			if page := GetInt(args, "page", 0); page > 0 {
				params.Set("page", strconv.Itoa(page))
			}

			if perPage := GetInt(args, "per_page", 0); perPage > 0 {
				params.Set("per_page", strconv.Itoa(perPage))
			}

			endpoint := fmt.Sprintf("/projects/%s/issues", url.PathEscape(projectID))
			if len(params) > 0 {
				endpoint += "?" + params.Encode()
			}

			var issues []gitlab.Issue
			if err := ctx.Client.Get(endpoint, &issues); err != nil {
				return ErrorResult(fmt.Sprintf("failed to list issues: %v", err))
			}

			return JSONResult(issues)
		},
	)
}

// registerMyIssues registers the my_issues tool.
func registerMyIssues(server *mcp.Server) {
	server.RegisterTool(
		mcp.Tool{
			Name:        "my_issues",
			Description: "List issues assigned to the authenticated user across all projects.",
			InputSchema: mcp.JSONSchema{
				Type: "object",
				Properties: map[string]mcp.Property{
					"state": {
						Type:        "string",
						Description: "Filter issues by state: opened, closed, or all",
						Enum:        []string{"opened", "closed", "all"},
					},
					"scope": {
						Type:        "string",
						Description: "Scope of issues: all, assigned_to_me, or created_by_me",
						Enum:        []string{"all", "assigned_to_me", "created_by_me"},
					},
					"page": {
						Type:        "integer",
						Description: "Page number for pagination (default: 1)",
					},
					"per_page": {
						Type:        "integer",
						Description: "Number of items per page (default: 20, max: 100)",
					},
				},
			},
		},
		func(args map[string]interface{}) (*mcp.CallToolResult, error) {
			ctx := GetContext()
			if ctx == nil {
				return ErrorResult("tool context not initialized")
			}
			ctx.Logger.ToolCall("my_issues", args)

			// Build query parameters
			params := url.Values{}

			if state := GetString(args, "state", ""); state != "" {
				params.Set("state", state)
			}

			if scope := GetString(args, "scope", ""); scope != "" {
				params.Set("scope", scope)
			}

			if page := GetInt(args, "page", 0); page > 0 {
				params.Set("page", strconv.Itoa(page))
			}

			if perPage := GetInt(args, "per_page", 0); perPage > 0 {
				params.Set("per_page", strconv.Itoa(perPage))
			}

			endpoint := "/issues"
			if len(params) > 0 {
				endpoint += "?" + params.Encode()
			}

			var issues []gitlab.Issue
			if err := ctx.Client.Get(endpoint, &issues); err != nil {
				return ErrorResult(fmt.Sprintf("failed to list issues: %v", err))
			}

			return JSONResult(issues)
		},
	)
}

// registerGetIssue registers the get_issue tool.
func registerGetIssue(server *mcp.Server) {
	server.RegisterTool(
		mcp.Tool{
			Name:        "get_issue",
			Description: "Get details of a specific issue in a GitLab project.",
			InputSchema: mcp.JSONSchema{
				Type: "object",
				Properties: map[string]mcp.Property{
					"project_id": {
						Type:        "string",
						Description: "The ID or URL-encoded path of the project",
					},
					"issue_iid": {
						Type:        "integer",
						Description: "The internal ID of the issue within the project",
					},
				},
				Required: []string{"project_id", "issue_iid"},
			},
		},
		func(args map[string]interface{}) (*mcp.CallToolResult, error) {
			ctx := GetContext()
			if ctx == nil {
				return ErrorResult("tool context not initialized")
			}
			ctx.Logger.ToolCall("get_issue", args)

			projectID := GetString(args, "project_id", "")
			if projectID == "" {
				return ErrorResult("project_id is required")
			}

			issueIID := GetInt(args, "issue_iid", 0)
			if issueIID == 0 {
				return ErrorResult("issue_iid is required")
			}

			endpoint := fmt.Sprintf("/projects/%s/issues/%d",
				url.PathEscape(projectID),
				issueIID,
			)

			var issue gitlab.Issue
			if err := ctx.Client.Get(endpoint, &issue); err != nil {
				return ErrorResult(fmt.Sprintf("failed to get issue: %v", err))
			}

			return JSONResult(issue)
		},
	)
}

// registerCreateIssue registers the create_issue tool.
func registerCreateIssue(server *mcp.Server) {
	server.RegisterTool(
		mcp.Tool{
			Name:        "create_issue",
			Description: "Create a new issue in a GitLab project.",
			InputSchema: mcp.JSONSchema{
				Type: "object",
				Properties: map[string]mcp.Property{
					"project_id": {
						Type:        "string",
						Description: "The ID or URL-encoded path of the project",
					},
					"title": {
						Type:        "string",
						Description: "The title of the issue",
					},
					"description": {
						Type:        "string",
						Description: "The description of the issue (supports Markdown)",
					},
					"labels": {
						Type:        "string",
						Description: "Comma-separated list of label names",
					},
					"milestone_id": {
						Type:        "integer",
						Description: "The ID of a milestone to assign the issue to",
					},
					"assignee_ids": {
						Type:        "array",
						Description: "Array of user IDs to assign the issue to",
						Items:       &mcp.Property{Type: "integer"},
					},
				},
				Required: []string{"project_id", "title"},
			},
		},
		func(args map[string]interface{}) (*mcp.CallToolResult, error) {
			ctx := GetContext()
			if ctx == nil {
				return ErrorResult("tool context not initialized")
			}
			ctx.Logger.ToolCall("create_issue", args)

			projectID := GetString(args, "project_id", "")
			if projectID == "" {
				return ErrorResult("project_id is required")
			}

			title := GetString(args, "title", "")
			if title == "" {
				return ErrorResult("title is required")
			}

			// Build request body
			body := map[string]interface{}{
				"title": title,
			}

			if description := GetString(args, "description", ""); description != "" {
				body["description"] = description
			}

			if labels := GetString(args, "labels", ""); labels != "" {
				body["labels"] = labels
			}

			if milestoneID := GetInt(args, "milestone_id", 0); milestoneID > 0 {
				body["milestone_id"] = milestoneID
			}

			if assigneeIDs := getIssueIntArray(args, "assignee_ids"); len(assigneeIDs) > 0 {
				body["assignee_ids"] = assigneeIDs
			}

			endpoint := fmt.Sprintf("/projects/%s/issues", url.PathEscape(projectID))

			var issue gitlab.Issue
			if err := ctx.Client.Post(endpoint, body, &issue); err != nil {
				return ErrorResult(fmt.Sprintf("failed to create issue: %v", err))
			}

			return JSONResult(issue)
		},
	)
}

// registerUpdateIssue registers the update_issue tool.
func registerUpdateIssue(server *mcp.Server) {
	server.RegisterTool(
		mcp.Tool{
			Name:        "update_issue",
			Description: "Update an existing issue in a GitLab project.",
			InputSchema: mcp.JSONSchema{
				Type: "object",
				Properties: map[string]mcp.Property{
					"project_id": {
						Type:        "string",
						Description: "The ID or URL-encoded path of the project",
					},
					"issue_iid": {
						Type:        "integer",
						Description: "The internal ID of the issue within the project",
					},
					"title": {
						Type:        "string",
						Description: "The title of the issue",
					},
					"description": {
						Type:        "string",
						Description: "The description of the issue (supports Markdown)",
					},
					"state_event": {
						Type:        "string",
						Description: "State event to change issue state: close or reopen",
						Enum:        []string{"close", "reopen"},
					},
					"labels": {
						Type:        "string",
						Description: "Comma-separated list of label names",
					},
					"milestone_id": {
						Type:        "integer",
						Description: "The ID of a milestone to assign the issue to",
					},
					"assignee_ids": {
						Type:        "array",
						Description: "Array of user IDs to assign the issue to",
						Items:       &mcp.Property{Type: "integer"},
					},
				},
				Required: []string{"project_id", "issue_iid"},
			},
		},
		func(args map[string]interface{}) (*mcp.CallToolResult, error) {
			ctx := GetContext()
			if ctx == nil {
				return ErrorResult("tool context not initialized")
			}
			ctx.Logger.ToolCall("update_issue", args)

			projectID := GetString(args, "project_id", "")
			if projectID == "" {
				return ErrorResult("project_id is required")
			}

			issueIID := GetInt(args, "issue_iid", 0)
			if issueIID == 0 {
				return ErrorResult("issue_iid is required")
			}

			// Build request body with only provided fields
			body := make(map[string]interface{})

			if title := GetString(args, "title", ""); title != "" {
				body["title"] = title
			}

			if description, exists := args["description"]; exists {
				body["description"] = description
			}

			if stateEvent := GetString(args, "state_event", ""); stateEvent != "" {
				body["state_event"] = stateEvent
			}

			if labels, exists := args["labels"]; exists {
				body["labels"] = labels
			}

			if milestoneID := GetInt(args, "milestone_id", 0); milestoneID > 0 {
				body["milestone_id"] = milestoneID
			}

			if assigneeIDs := getIssueIntArray(args, "assignee_ids"); len(assigneeIDs) > 0 {
				body["assignee_ids"] = assigneeIDs
			}

			endpoint := fmt.Sprintf("/projects/%s/issues/%d",
				url.PathEscape(projectID),
				issueIID,
			)

			var issue gitlab.Issue
			if err := ctx.Client.Put(endpoint, body, &issue); err != nil {
				return ErrorResult(fmt.Sprintf("failed to update issue: %v", err))
			}

			return JSONResult(issue)
		},
	)
}

// registerDeleteIssue registers the delete_issue tool.
func registerDeleteIssue(server *mcp.Server) {
	server.RegisterTool(
		mcp.Tool{
			Name:        "delete_issue",
			Description: "Delete an issue from a GitLab project. This action is irreversible.",
			InputSchema: mcp.JSONSchema{
				Type: "object",
				Properties: map[string]mcp.Property{
					"project_id": {
						Type:        "string",
						Description: "The ID or URL-encoded path of the project",
					},
					"issue_iid": {
						Type:        "integer",
						Description: "The internal ID of the issue within the project",
					},
				},
				Required: []string{"project_id", "issue_iid"},
			},
		},
		func(args map[string]interface{}) (*mcp.CallToolResult, error) {
			ctx := GetContext()
			if ctx == nil {
				return ErrorResult("tool context not initialized")
			}
			ctx.Logger.ToolCall("delete_issue", args)

			projectID := GetString(args, "project_id", "")
			if projectID == "" {
				return ErrorResult("project_id is required")
			}

			issueIID := GetInt(args, "issue_iid", 0)
			if issueIID == 0 {
				return ErrorResult("issue_iid is required")
			}

			endpoint := fmt.Sprintf("/projects/%s/issues/%d",
				url.PathEscape(projectID),
				issueIID,
			)

			if err := ctx.Client.Delete(endpoint); err != nil {
				return ErrorResult(fmt.Sprintf("failed to delete issue: %v", err))
			}

			return TextResult(fmt.Sprintf("Issue #%d deleted successfully", issueIID))
		},
	)
}

// registerListIssueLinks registers the list_issue_links tool.
func registerListIssueLinks(server *mcp.Server) {
	server.RegisterTool(
		mcp.Tool{
			Name:        "list_issue_links",
			Description: "List all links for a specific issue.",
			InputSchema: mcp.JSONSchema{
				Type: "object",
				Properties: map[string]mcp.Property{
					"project_id": {
						Type:        "string",
						Description: "The ID or URL-encoded path of the project",
					},
					"issue_iid": {
						Type:        "integer",
						Description: "The internal ID of the issue within the project",
					},
				},
				Required: []string{"project_id", "issue_iid"},
			},
		},
		func(args map[string]interface{}) (*mcp.CallToolResult, error) {
			ctx := GetContext()
			if ctx == nil {
				return ErrorResult("tool context not initialized")
			}
			ctx.Logger.ToolCall("list_issue_links", args)

			projectID := GetString(args, "project_id", "")
			if projectID == "" {
				return ErrorResult("project_id is required")
			}

			issueIID := GetInt(args, "issue_iid", 0)
			if issueIID == 0 {
				return ErrorResult("issue_iid is required")
			}

			endpoint := fmt.Sprintf("/projects/%s/issues/%d/links",
				url.PathEscape(projectID),
				issueIID,
			)

			var links []IssueLink
			if err := ctx.Client.Get(endpoint, &links); err != nil {
				return ErrorResult(fmt.Sprintf("failed to list issue links: %v", err))
			}

			return JSONResult(links)
		},
	)
}

// registerGetIssueLink registers the get_issue_link tool.
func registerGetIssueLink(server *mcp.Server) {
	server.RegisterTool(
		mcp.Tool{
			Name:        "get_issue_link",
			Description: "Get details of a specific issue link.",
			InputSchema: mcp.JSONSchema{
				Type: "object",
				Properties: map[string]mcp.Property{
					"project_id": {
						Type:        "string",
						Description: "The ID or URL-encoded path of the project",
					},
					"issue_iid": {
						Type:        "integer",
						Description: "The internal ID of the issue within the project",
					},
					"link_id": {
						Type:        "integer",
						Description: "The ID of the issue link",
					},
				},
				Required: []string{"project_id", "issue_iid", "link_id"},
			},
		},
		func(args map[string]interface{}) (*mcp.CallToolResult, error) {
			ctx := GetContext()
			if ctx == nil {
				return ErrorResult("tool context not initialized")
			}
			ctx.Logger.ToolCall("get_issue_link", args)

			projectID := GetString(args, "project_id", "")
			if projectID == "" {
				return ErrorResult("project_id is required")
			}

			issueIID := GetInt(args, "issue_iid", 0)
			if issueIID == 0 {
				return ErrorResult("issue_iid is required")
			}

			linkID := GetInt(args, "link_id", 0)
			if linkID == 0 {
				return ErrorResult("link_id is required")
			}

			endpoint := fmt.Sprintf("/projects/%s/issues/%d/links/%d",
				url.PathEscape(projectID),
				issueIID,
				linkID,
			)

			var link IssueLink
			if err := ctx.Client.Get(endpoint, &link); err != nil {
				return ErrorResult(fmt.Sprintf("failed to get issue link: %v", err))
			}

			return JSONResult(link)
		},
	)
}

// registerCreateIssueLink registers the create_issue_link tool.
func registerCreateIssueLink(server *mcp.Server) {
	server.RegisterTool(
		mcp.Tool{
			Name:        "create_issue_link",
			Description: "Create a link between two issues.",
			InputSchema: mcp.JSONSchema{
				Type: "object",
				Properties: map[string]mcp.Property{
					"project_id": {
						Type:        "string",
						Description: "The ID or URL-encoded path of the project",
					},
					"issue_iid": {
						Type:        "integer",
						Description: "The internal ID of the source issue within the project",
					},
					"target_project_id": {
						Type:        "string",
						Description: "The ID or URL-encoded path of the target project",
					},
					"target_issue_iid": {
						Type:        "integer",
						Description: "The internal ID of the target issue",
					},
					"link_type": {
						Type:        "string",
						Description: "The type of the link: relates_to, blocks, or is_blocked_by",
						Enum:        []string{"relates_to", "blocks", "is_blocked_by"},
					},
				},
				Required: []string{"project_id", "issue_iid", "target_project_id", "target_issue_iid"},
			},
		},
		func(args map[string]interface{}) (*mcp.CallToolResult, error) {
			ctx := GetContext()
			if ctx == nil {
				return ErrorResult("tool context not initialized")
			}
			ctx.Logger.ToolCall("create_issue_link", args)

			projectID := GetString(args, "project_id", "")
			if projectID == "" {
				return ErrorResult("project_id is required")
			}

			issueIID := GetInt(args, "issue_iid", 0)
			if issueIID == 0 {
				return ErrorResult("issue_iid is required")
			}

			targetProjectID := GetString(args, "target_project_id", "")
			if targetProjectID == "" {
				return ErrorResult("target_project_id is required")
			}

			targetIssueIID := GetInt(args, "target_issue_iid", 0)
			if targetIssueIID == 0 {
				return ErrorResult("target_issue_iid is required")
			}

			// Build request body
			body := map[string]interface{}{
				"target_project_id": targetProjectID,
				"target_issue_iid":  targetIssueIID,
			}

			if linkType := GetString(args, "link_type", ""); linkType != "" {
				body["link_type"] = linkType
			}

			endpoint := fmt.Sprintf("/projects/%s/issues/%d/links",
				url.PathEscape(projectID),
				issueIID,
			)

			var link IssueLink
			if err := ctx.Client.Post(endpoint, body, &link); err != nil {
				return ErrorResult(fmt.Sprintf("failed to create issue link: %v", err))
			}

			return JSONResult(link)
		},
	)
}

// registerDeleteIssueLink registers the delete_issue_link tool.
func registerDeleteIssueLink(server *mcp.Server) {
	server.RegisterTool(
		mcp.Tool{
			Name:        "delete_issue_link",
			Description: "Delete an issue link.",
			InputSchema: mcp.JSONSchema{
				Type: "object",
				Properties: map[string]mcp.Property{
					"project_id": {
						Type:        "string",
						Description: "The ID or URL-encoded path of the project",
					},
					"issue_iid": {
						Type:        "integer",
						Description: "The internal ID of the issue within the project",
					},
					"link_id": {
						Type:        "integer",
						Description: "The ID of the issue link",
					},
				},
				Required: []string{"project_id", "issue_iid", "link_id"},
			},
		},
		func(args map[string]interface{}) (*mcp.CallToolResult, error) {
			ctx := GetContext()
			if ctx == nil {
				return ErrorResult("tool context not initialized")
			}
			ctx.Logger.ToolCall("delete_issue_link", args)

			projectID := GetString(args, "project_id", "")
			if projectID == "" {
				return ErrorResult("project_id is required")
			}

			issueIID := GetInt(args, "issue_iid", 0)
			if issueIID == 0 {
				return ErrorResult("issue_iid is required")
			}

			linkID := GetInt(args, "link_id", 0)
			if linkID == 0 {
				return ErrorResult("link_id is required")
			}

			endpoint := fmt.Sprintf("/projects/%s/issues/%d/links/%d",
				url.PathEscape(projectID),
				issueIID,
				linkID,
			)

			if err := ctx.Client.Delete(endpoint); err != nil {
				return ErrorResult(fmt.Sprintf("failed to delete issue link: %v", err))
			}

			return TextResult(fmt.Sprintf("Issue link %d deleted successfully", linkID))
		},
	)
}

// registerListIssueDiscussions registers the list_issue_discussions tool.
func registerListIssueDiscussions(server *mcp.Server) {
	server.RegisterTool(
		mcp.Tool{
			Name:        "list_issue_discussions",
			Description: "List all discussions (threads of notes/comments) on an issue.",
			InputSchema: mcp.JSONSchema{
				Type: "object",
				Properties: map[string]mcp.Property{
					"project_id": {
						Type:        "string",
						Description: "The ID or URL-encoded path of the project",
					},
					"issue_iid": {
						Type:        "integer",
						Description: "The internal ID of the issue within the project",
					},
					"page": {
						Type:        "integer",
						Description: "Page number for pagination (default: 1)",
					},
					"per_page": {
						Type:        "integer",
						Description: "Number of items per page (default: 20, max: 100)",
					},
				},
				Required: []string{"project_id", "issue_iid"},
			},
		},
		func(args map[string]interface{}) (*mcp.CallToolResult, error) {
			ctx := GetContext()
			if ctx == nil {
				return ErrorResult("tool context not initialized")
			}
			ctx.Logger.ToolCall("list_issue_discussions", args)

			projectID := GetString(args, "project_id", "")
			if projectID == "" {
				return ErrorResult("project_id is required")
			}

			issueIID := GetInt(args, "issue_iid", 0)
			if issueIID == 0 {
				return ErrorResult("issue_iid is required")
			}

			// Build query parameters
			params := url.Values{}

			if page := GetInt(args, "page", 0); page > 0 {
				params.Set("page", strconv.Itoa(page))
			}

			if perPage := GetInt(args, "per_page", 0); perPage > 0 {
				params.Set("per_page", strconv.Itoa(perPage))
			}

			endpoint := fmt.Sprintf("/projects/%s/issues/%d/discussions",
				url.PathEscape(projectID),
				issueIID,
			)
			if len(params) > 0 {
				endpoint += "?" + params.Encode()
			}

			var discussions []Discussion
			if err := ctx.Client.Get(endpoint, &discussions); err != nil {
				return ErrorResult(fmt.Sprintf("failed to list issue discussions: %v", err))
			}

			return JSONResult(discussions)
		},
	)
}

// RegisterIssueTools registers all issue-related tools with the MCP server.
// Includes: list_issues, my_issues, get_issue, create_issue, update_issue,
// delete_issue, list_issue_links, get_issue_link, create_issue_link,
// delete_issue_link, list_issue_discussions
func RegisterIssueTools(server *mcp.Server) {
	registerListIssues(server)
	registerMyIssues(server)
	registerGetIssue(server)
	registerCreateIssue(server)
	registerUpdateIssue(server)
	registerDeleteIssue(server)
	registerListIssueLinks(server)
	registerGetIssueLink(server)
	registerCreateIssueLink(server)
	registerDeleteIssueLink(server)
	registerListIssueDiscussions(server)
}

// getIssueIntArray extracts an integer array from arguments map.
// This is a local helper to avoid conflicts with other definitions.
func getIssueIntArray(args map[string]interface{}, key string) []int {
	if args == nil {
		return nil
	}
	val, ok := args[key]
	if !ok {
		return nil
	}

	switch v := val.(type) {
	case []int:
		return v
	case []interface{}:
		result := make([]int, 0, len(v))
		for _, item := range v {
			switch i := item.(type) {
			case int:
				result = append(result, i)
			case int64:
				result = append(result, int(i))
			case float64:
				result = append(result, int(i))
			}
		}
		return result
	default:
		return nil
	}
}
