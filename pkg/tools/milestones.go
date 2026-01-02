// Package tools provides MCP tool implementations for GitLab milestone operations.
package tools

import (
	"fmt"
	"net/url"
	"strconv"

	"github.com/go-mcp-gitlab/go-mcp-gitlab/pkg/gitlab"
	"github.com/go-mcp-gitlab/go-mcp-gitlab/pkg/mcp"
)

// BurndownEvent represents a milestone burndown event (GitLab Premium/Ultimate).
type BurndownEvent struct {
	CreatedAt string `json:"created_at"`
	Weight    int    `json:"weight,omitempty"`
	Action    string `json:"action,omitempty"`
}

// registerListMilestones registers the list_milestones tool.
func registerListMilestones(server *mcp.Server) {
	server.RegisterTool(
		mcp.Tool{
			Name:        "list_milestones",
			Description: "List milestones in a GitLab project. Returns a paginated list of milestones with optional filtering by state and search term.",
			InputSchema: mcp.JSONSchema{
				Type: "object",
				Properties: map[string]mcp.Property{
					"project_id": {
						Type:        "string",
						Description: "The ID or URL-encoded path of the project",
					},
					"state": {
						Type:        "string",
						Description: "Filter milestones by state: active or closed",
						Enum:        []string{"active", "closed"},
					},
					"search": {
						Type:        "string",
						Description: "Search milestones by title",
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
			ctx.Logger.ToolCall("list_milestones", args)

			projectID := GetString(args, "project_id", "")
			if projectID == "" {
				return ErrorResult("project_id is required")
			}

			// Build query parameters
			params := url.Values{}

			if state := GetString(args, "state", ""); state != "" {
				params.Set("state", state)
			}

			if search := GetString(args, "search", ""); search != "" {
				params.Set("search", search)
			}

			if page := GetInt(args, "page", 0); page > 0 {
				params.Set("page", strconv.Itoa(page))
			}

			if perPage := GetInt(args, "per_page", 0); perPage > 0 {
				params.Set("per_page", strconv.Itoa(perPage))
			}

			endpoint := fmt.Sprintf("/projects/%s/milestones", url.PathEscape(projectID))
			if len(params) > 0 {
				endpoint += "?" + params.Encode()
			}

			var milestones []gitlab.Milestone
			if err := ctx.Client.Get(endpoint, &milestones); err != nil {
				return ErrorResult(fmt.Sprintf("failed to list milestones: %v", err))
			}

			return JSONResult(milestones)
		},
	)
}

// registerGetMilestone registers the get_milestone tool.
func registerGetMilestone(server *mcp.Server) {
	server.RegisterTool(
		mcp.Tool{
			Name:        "get_milestone",
			Description: "Get details of a specific milestone in a GitLab project.",
			InputSchema: mcp.JSONSchema{
				Type: "object",
				Properties: map[string]mcp.Property{
					"project_id": {
						Type:        "string",
						Description: "The ID or URL-encoded path of the project",
					},
					"milestone_id": {
						Type:        "integer",
						Description: "The ID of the milestone",
					},
				},
				Required: []string{"project_id", "milestone_id"},
			},
		},
		func(args map[string]interface{}) (*mcp.CallToolResult, error) {
			ctx := GetContext()
			if ctx == nil {
				return ErrorResult("tool context not initialized")
			}
			ctx.Logger.ToolCall("get_milestone", args)

			projectID := GetString(args, "project_id", "")
			if projectID == "" {
				return ErrorResult("project_id is required")
			}

			milestoneID := GetInt(args, "milestone_id", 0)
			if milestoneID == 0 {
				return ErrorResult("milestone_id is required")
			}

			endpoint := fmt.Sprintf("/projects/%s/milestones/%d",
				url.PathEscape(projectID),
				milestoneID,
			)

			var milestone gitlab.Milestone
			if err := ctx.Client.Get(endpoint, &milestone); err != nil {
				return ErrorResult(fmt.Sprintf("failed to get milestone: %v", err))
			}

			return JSONResult(milestone)
		},
	)
}

// registerCreateMilestone registers the create_milestone tool.
func registerCreateMilestone(server *mcp.Server) {
	server.RegisterTool(
		mcp.Tool{
			Name:        "create_milestone",
			Description: "Create a new milestone in a GitLab project.",
			InputSchema: mcp.JSONSchema{
				Type: "object",
				Properties: map[string]mcp.Property{
					"project_id": {
						Type:        "string",
						Description: "The ID or URL-encoded path of the project",
					},
					"title": {
						Type:        "string",
						Description: "The title of the milestone",
					},
					"description": {
						Type:        "string",
						Description: "The description of the milestone",
					},
					"due_date": {
						Type:        "string",
						Description: "The due date of the milestone in YYYY-MM-DD format",
					},
					"start_date": {
						Type:        "string",
						Description: "The start date of the milestone in YYYY-MM-DD format",
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
			ctx.Logger.ToolCall("create_milestone", args)

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

			if dueDate := GetString(args, "due_date", ""); dueDate != "" {
				body["due_date"] = dueDate
			}

			if startDate := GetString(args, "start_date", ""); startDate != "" {
				body["start_date"] = startDate
			}

			endpoint := fmt.Sprintf("/projects/%s/milestones", url.PathEscape(projectID))

			var milestone gitlab.Milestone
			if err := ctx.Client.Post(endpoint, body, &milestone); err != nil {
				return ErrorResult(fmt.Sprintf("failed to create milestone: %v", err))
			}

			return JSONResult(milestone)
		},
	)
}

// registerEditMilestone registers the edit_milestone tool.
func registerEditMilestone(server *mcp.Server) {
	server.RegisterTool(
		mcp.Tool{
			Name:        "edit_milestone",
			Description: "Update an existing milestone in a GitLab project.",
			InputSchema: mcp.JSONSchema{
				Type: "object",
				Properties: map[string]mcp.Property{
					"project_id": {
						Type:        "string",
						Description: "The ID or URL-encoded path of the project",
					},
					"milestone_id": {
						Type:        "integer",
						Description: "The ID of the milestone",
					},
					"title": {
						Type:        "string",
						Description: "The title of the milestone",
					},
					"description": {
						Type:        "string",
						Description: "The description of the milestone",
					},
					"due_date": {
						Type:        "string",
						Description: "The due date of the milestone in YYYY-MM-DD format",
					},
					"start_date": {
						Type:        "string",
						Description: "The start date of the milestone in YYYY-MM-DD format",
					},
					"state_event": {
						Type:        "string",
						Description: "State event to change milestone state: close or activate",
						Enum:        []string{"close", "activate"},
					},
				},
				Required: []string{"project_id", "milestone_id"},
			},
		},
		func(args map[string]interface{}) (*mcp.CallToolResult, error) {
			ctx := GetContext()
			if ctx == nil {
				return ErrorResult("tool context not initialized")
			}
			ctx.Logger.ToolCall("edit_milestone", args)

			projectID := GetString(args, "project_id", "")
			if projectID == "" {
				return ErrorResult("project_id is required")
			}

			milestoneID := GetInt(args, "milestone_id", 0)
			if milestoneID == 0 {
				return ErrorResult("milestone_id is required")
			}

			// Build request body with only provided fields
			body := make(map[string]interface{})

			if title := GetString(args, "title", ""); title != "" {
				body["title"] = title
			}

			if description, exists := args["description"]; exists {
				body["description"] = description
			}

			if dueDate, exists := args["due_date"]; exists {
				body["due_date"] = dueDate
			}

			if startDate, exists := args["start_date"]; exists {
				body["start_date"] = startDate
			}

			if stateEvent := GetString(args, "state_event", ""); stateEvent != "" {
				body["state_event"] = stateEvent
			}

			endpoint := fmt.Sprintf("/projects/%s/milestones/%d",
				url.PathEscape(projectID),
				milestoneID,
			)

			var milestone gitlab.Milestone
			if err := ctx.Client.Put(endpoint, body, &milestone); err != nil {
				return ErrorResult(fmt.Sprintf("failed to edit milestone: %v", err))
			}

			return JSONResult(milestone)
		},
	)
}

// registerDeleteMilestone registers the delete_milestone tool.
func registerDeleteMilestone(server *mcp.Server) {
	server.RegisterTool(
		mcp.Tool{
			Name:        "delete_milestone",
			Description: "Delete a milestone from a GitLab project. This action is irreversible.",
			InputSchema: mcp.JSONSchema{
				Type: "object",
				Properties: map[string]mcp.Property{
					"project_id": {
						Type:        "string",
						Description: "The ID or URL-encoded path of the project",
					},
					"milestone_id": {
						Type:        "integer",
						Description: "The ID of the milestone",
					},
				},
				Required: []string{"project_id", "milestone_id"},
			},
		},
		func(args map[string]interface{}) (*mcp.CallToolResult, error) {
			ctx := GetContext()
			if ctx == nil {
				return ErrorResult("tool context not initialized")
			}
			ctx.Logger.ToolCall("delete_milestone", args)

			projectID := GetString(args, "project_id", "")
			if projectID == "" {
				return ErrorResult("project_id is required")
			}

			milestoneID := GetInt(args, "milestone_id", 0)
			if milestoneID == 0 {
				return ErrorResult("milestone_id is required")
			}

			endpoint := fmt.Sprintf("/projects/%s/milestones/%d",
				url.PathEscape(projectID),
				milestoneID,
			)

			if err := ctx.Client.Delete(endpoint); err != nil {
				return ErrorResult(fmt.Sprintf("failed to delete milestone: %v", err))
			}

			return TextResult(fmt.Sprintf("Milestone %d deleted successfully", milestoneID))
		},
	)
}

// registerGetMilestoneIssues registers the get_milestone_issues tool.
func registerGetMilestoneIssues(server *mcp.Server) {
	server.RegisterTool(
		mcp.Tool{
			Name:        "get_milestone_issues",
			Description: "Get all issues assigned to a specific milestone.",
			InputSchema: mcp.JSONSchema{
				Type: "object",
				Properties: map[string]mcp.Property{
					"project_id": {
						Type:        "string",
						Description: "The ID or URL-encoded path of the project",
					},
					"milestone_id": {
						Type:        "integer",
						Description: "The ID of the milestone",
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
				Required: []string{"project_id", "milestone_id"},
			},
		},
		func(args map[string]interface{}) (*mcp.CallToolResult, error) {
			ctx := GetContext()
			if ctx == nil {
				return ErrorResult("tool context not initialized")
			}
			ctx.Logger.ToolCall("get_milestone_issues", args)

			projectID := GetString(args, "project_id", "")
			if projectID == "" {
				return ErrorResult("project_id is required")
			}

			milestoneID := GetInt(args, "milestone_id", 0)
			if milestoneID == 0 {
				return ErrorResult("milestone_id is required")
			}

			// Build query parameters
			params := url.Values{}

			if page := GetInt(args, "page", 0); page > 0 {
				params.Set("page", strconv.Itoa(page))
			}

			if perPage := GetInt(args, "per_page", 0); perPage > 0 {
				params.Set("per_page", strconv.Itoa(perPage))
			}

			endpoint := fmt.Sprintf("/projects/%s/milestones/%d/issues",
				url.PathEscape(projectID),
				milestoneID,
			)
			if len(params) > 0 {
				endpoint += "?" + params.Encode()
			}

			var issues []gitlab.Issue
			if err := ctx.Client.Get(endpoint, &issues); err != nil {
				return ErrorResult(fmt.Sprintf("failed to get milestone issues: %v", err))
			}

			return JSONResult(issues)
		},
	)
}

// registerGetMilestoneMergeRequests registers the get_milestone_merge_requests tool.
func registerGetMilestoneMergeRequests(server *mcp.Server) {
	server.RegisterTool(
		mcp.Tool{
			Name:        "get_milestone_merge_requests",
			Description: "Get all merge requests assigned to a specific milestone.",
			InputSchema: mcp.JSONSchema{
				Type: "object",
				Properties: map[string]mcp.Property{
					"project_id": {
						Type:        "string",
						Description: "The ID or URL-encoded path of the project",
					},
					"milestone_id": {
						Type:        "integer",
						Description: "The ID of the milestone",
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
				Required: []string{"project_id", "milestone_id"},
			},
		},
		func(args map[string]interface{}) (*mcp.CallToolResult, error) {
			ctx := GetContext()
			if ctx == nil {
				return ErrorResult("tool context not initialized")
			}
			ctx.Logger.ToolCall("get_milestone_merge_requests", args)

			projectID := GetString(args, "project_id", "")
			if projectID == "" {
				return ErrorResult("project_id is required")
			}

			milestoneID := GetInt(args, "milestone_id", 0)
			if milestoneID == 0 {
				return ErrorResult("milestone_id is required")
			}

			// Build query parameters
			params := url.Values{}

			if page := GetInt(args, "page", 0); page > 0 {
				params.Set("page", strconv.Itoa(page))
			}

			if perPage := GetInt(args, "per_page", 0); perPage > 0 {
				params.Set("per_page", strconv.Itoa(perPage))
			}

			endpoint := fmt.Sprintf("/projects/%s/milestones/%d/merge_requests",
				url.PathEscape(projectID),
				milestoneID,
			)
			if len(params) > 0 {
				endpoint += "?" + params.Encode()
			}

			var mergeRequests []gitlab.MergeRequest
			if err := ctx.Client.Get(endpoint, &mergeRequests); err != nil {
				return ErrorResult(fmt.Sprintf("failed to get milestone merge requests: %v", err))
			}

			return JSONResult(mergeRequests)
		},
	)
}

// registerPromoteMilestone registers the promote_milestone tool.
func registerPromoteMilestone(server *mcp.Server) {
	server.RegisterTool(
		mcp.Tool{
			Name:        "promote_milestone",
			Description: "Promote a project milestone to a group milestone. The milestone will be available for all projects in the group.",
			InputSchema: mcp.JSONSchema{
				Type: "object",
				Properties: map[string]mcp.Property{
					"project_id": {
						Type:        "string",
						Description: "The ID or URL-encoded path of the project",
					},
					"milestone_id": {
						Type:        "integer",
						Description: "The ID of the milestone",
					},
				},
				Required: []string{"project_id", "milestone_id"},
			},
		},
		func(args map[string]interface{}) (*mcp.CallToolResult, error) {
			ctx := GetContext()
			if ctx == nil {
				return ErrorResult("tool context not initialized")
			}
			ctx.Logger.ToolCall("promote_milestone", args)

			projectID := GetString(args, "project_id", "")
			if projectID == "" {
				return ErrorResult("project_id is required")
			}

			milestoneID := GetInt(args, "milestone_id", 0)
			if milestoneID == 0 {
				return ErrorResult("milestone_id is required")
			}

			endpoint := fmt.Sprintf("/projects/%s/milestones/%d/promote",
				url.PathEscape(projectID),
				milestoneID,
			)

			var milestone gitlab.Milestone
			if err := ctx.Client.Post(endpoint, nil, &milestone); err != nil {
				return ErrorResult(fmt.Sprintf("failed to promote milestone: %v", err))
			}

			return JSONResult(milestone)
		},
	)
}

// registerGetMilestoneBurndownEvents registers the get_milestone_burndown_events tool.
func registerGetMilestoneBurndownEvents(server *mcp.Server) {
	server.RegisterTool(
		mcp.Tool{
			Name:        "get_milestone_burndown_events",
			Description: "Get the burndown chart events for a specific milestone. This endpoint is only available in GitLab Premium/Ultimate.",
			InputSchema: mcp.JSONSchema{
				Type: "object",
				Properties: map[string]mcp.Property{
					"project_id": {
						Type:        "string",
						Description: "The ID or URL-encoded path of the project",
					},
					"milestone_id": {
						Type:        "integer",
						Description: "The ID of the milestone",
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
				Required: []string{"project_id", "milestone_id"},
			},
		},
		func(args map[string]interface{}) (*mcp.CallToolResult, error) {
			ctx := GetContext()
			if ctx == nil {
				return ErrorResult("tool context not initialized")
			}
			ctx.Logger.ToolCall("get_milestone_burndown_events", args)

			projectID := GetString(args, "project_id", "")
			if projectID == "" {
				return ErrorResult("project_id is required")
			}

			milestoneID := GetInt(args, "milestone_id", 0)
			if milestoneID == 0 {
				return ErrorResult("milestone_id is required")
			}

			// Build query parameters
			params := url.Values{}

			if page := GetInt(args, "page", 0); page > 0 {
				params.Set("page", strconv.Itoa(page))
			}

			if perPage := GetInt(args, "per_page", 0); perPage > 0 {
				params.Set("per_page", strconv.Itoa(perPage))
			}

			endpoint := fmt.Sprintf("/projects/%s/milestones/%d/burndown_events",
				url.PathEscape(projectID),
				milestoneID,
			)
			if len(params) > 0 {
				endpoint += "?" + params.Encode()
			}

			var events []BurndownEvent
			if err := ctx.Client.Get(endpoint, &events); err != nil {
				return ErrorResult(fmt.Sprintf("failed to get milestone burndown events: %v", err))
			}

			return JSONResult(events)
		},
	)
}

// initMilestoneTools registers all milestone-related tools.
func initMilestoneTools(server *mcp.Server) {
	registerListMilestones(server)
	registerGetMilestone(server)
	registerCreateMilestone(server)
	registerEditMilestone(server)
	registerDeleteMilestone(server)
	registerGetMilestoneIssues(server)
	registerGetMilestoneMergeRequests(server)
	registerPromoteMilestone(server)
	registerGetMilestoneBurndownEvents(server)
}
