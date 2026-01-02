package tools

import (
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/go-mcp-gitlab/go-mcp-gitlab/pkg/gitlab"
	"github.com/go-mcp-gitlab/go-mcp-gitlab/pkg/mcp"
)

// Event represents a GitLab event.
type Event struct {
	ID          int        `json:"id"`
	Title       string     `json:"title,omitempty"`
	ProjectID   int        `json:"project_id"`
	ActionName  string     `json:"action_name"`
	TargetID    int        `json:"target_id,omitempty"`
	TargetIID   int        `json:"target_iid,omitempty"`
	TargetType  string     `json:"target_type,omitempty"`
	TargetTitle string     `json:"target_title,omitempty"`
	Author      *gitlab.User `json:"author,omitempty"`
	AuthorID    int        `json:"author_id"`
	AuthorUsername string  `json:"author_username"`
	CreatedAt   *time.Time `json:"created_at"`
	Note        *EventNote `json:"note,omitempty"`
	PushData    *PushData  `json:"push_data,omitempty"`
	WikiPage    *EventWikiPage  `json:"wiki_page,omitempty"`
}

// EventNote represents a note attached to an event.
type EventNote struct {
	ID           int    `json:"id"`
	Body         string `json:"body"`
	NoteableID   int    `json:"noteable_id"`
	NoteableType string `json:"noteable_type"`
	NoteableIID  int    `json:"noteable_iid,omitempty"`
}

// PushData represents push data in an event.
type PushData struct {
	CommitCount int    `json:"commit_count"`
	Action      string `json:"action"`
	RefType     string `json:"ref_type"`
	CommitFrom  string `json:"commit_from,omitempty"`
	CommitTo    string `json:"commit_to,omitempty"`
	Ref         string `json:"ref"`
	CommitTitle string `json:"commit_title,omitempty"`
}

// EventWikiPage represents a wiki page reference in an event.
type EventWikiPage struct {
	Format string `json:"format"`
	Slug   string `json:"slug"`
	Title  string `json:"title"`
}

// registerGetUsers registers the get_users tool.
func registerGetUsers(server *mcp.Server) {
	server.RegisterTool(
		mcp.Tool{
			Name:        "get_users",
			Description: "Get GitLab user details by username(s). Returns user information including ID, username, name, state, and avatar URL.",
			InputSchema: mcp.JSONSchema{
				Type: "object",
				Properties: map[string]mcp.Property{
					"usernames": {
						Type:        "array",
						Description: "Array of usernames to look up",
						Items:       &mcp.Property{Type: "string"},
					},
				},
				Required: []string{"usernames"},
			},
		},
		func(args map[string]interface{}) (*mcp.CallToolResult, error) {
			ctx := GetContext()
			if ctx == nil {
				return ErrorResult("tool context not initialized")
			}
			ctx.Logger.ToolCall("get_users", args)

			usernames := GetStringArray(args, "usernames")
			if len(usernames) == 0 {
				return ErrorResult("usernames is required and must contain at least one username")
			}

			// Build query parameters with multiple username values
			params := url.Values{}
			for _, username := range usernames {
				params.Add("username", username)
			}

			endpoint := fmt.Sprintf("/users?%s", params.Encode())

			var users []gitlab.User
			if err := ctx.Client.Get(endpoint, &users); err != nil {
				return ErrorResult(fmt.Sprintf("failed to get users: %v", err))
			}

			return JSONResult(users)
		},
	)
}

// registerListEvents registers the list_events tool.
func registerListEvents(server *mcp.Server) {
	server.RegisterTool(
		mcp.Tool{
			Name:        "list_events",
			Description: "List events for the authenticated user. Returns a list of events such as pushes, comments, issue updates, and merge request activities.",
			InputSchema: mcp.JSONSchema{
				Type: "object",
				Properties: map[string]mcp.Property{
					"action": {
						Type:        "string",
						Description: "Filter events by action type: created, updated, closed, reopened, pushed, commented, merged, joined, left, destroyed, expired",
						Enum:        []string{"created", "updated", "closed", "reopened", "pushed", "commented", "merged", "joined", "left", "destroyed", "expired"},
					},
					"target_type": {
						Type:        "string",
						Description: "Filter events by target type: issue, milestone, merge_request, note, project, snippet, user",
						Enum:        []string{"issue", "milestone", "merge_request", "note", "project", "snippet", "user"},
					},
					"before": {
						Type:        "string",
						Description: "Filter events before this date (format: YYYY-MM-DD)",
					},
					"after": {
						Type:        "string",
						Description: "Filter events after this date (format: YYYY-MM-DD)",
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
			ctx.Logger.ToolCall("list_events", args)

			// Build query parameters
			params := url.Values{}

			if action := GetString(args, "action", ""); action != "" {
				params.Set("action", action)
			}

			if targetType := GetString(args, "target_type", ""); targetType != "" {
				params.Set("target_type", targetType)
			}

			if before := GetString(args, "before", ""); before != "" {
				params.Set("before", before)
			}

			if after := GetString(args, "after", ""); after != "" {
				params.Set("after", after)
			}

			if page := GetInt(args, "page", 0); page > 0 {
				params.Set("page", strconv.Itoa(page))
			}

			if perPage := GetInt(args, "per_page", 0); perPage > 0 {
				params.Set("per_page", strconv.Itoa(perPage))
			}

			endpoint := "/events"
			if len(params) > 0 {
				endpoint += "?" + params.Encode()
			}

			var events []Event
			if err := ctx.Client.Get(endpoint, &events); err != nil {
				return ErrorResult(fmt.Sprintf("failed to list events: %v", err))
			}

			return JSONResult(events)
		},
	)
}

// registerGetProjectEvents registers the get_project_events tool.
func registerGetProjectEvents(server *mcp.Server) {
	server.RegisterTool(
		mcp.Tool{
			Name:        "get_project_events",
			Description: "List events for a specific GitLab project. Returns a list of events such as pushes, comments, issue updates, and merge request activities within the project.",
			InputSchema: mcp.JSONSchema{
				Type: "object",
				Properties: map[string]mcp.Property{
					"project_id": {
						Type:        "string",
						Description: "The ID or URL-encoded path of the project",
					},
					"action": {
						Type:        "string",
						Description: "Filter events by action type: created, updated, closed, reopened, pushed, commented, merged, joined, left, destroyed, expired",
						Enum:        []string{"created", "updated", "closed", "reopened", "pushed", "commented", "merged", "joined", "left", "destroyed", "expired"},
					},
					"target_type": {
						Type:        "string",
						Description: "Filter events by target type: issue, milestone, merge_request, note, project, snippet, user",
						Enum:        []string{"issue", "milestone", "merge_request", "note", "project", "snippet", "user"},
					},
					"before": {
						Type:        "string",
						Description: "Filter events before this date (format: YYYY-MM-DD)",
					},
					"after": {
						Type:        "string",
						Description: "Filter events after this date (format: YYYY-MM-DD)",
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
			ctx.Logger.ToolCall("get_project_events", args)

			projectID := GetString(args, "project_id", "")
			if projectID == "" {
				return ErrorResult("project_id is required")
			}

			// Build query parameters
			params := url.Values{}

			if action := GetString(args, "action", ""); action != "" {
				params.Set("action", action)
			}

			if targetType := GetString(args, "target_type", ""); targetType != "" {
				params.Set("target_type", targetType)
			}

			if before := GetString(args, "before", ""); before != "" {
				params.Set("before", before)
			}

			if after := GetString(args, "after", ""); after != "" {
				params.Set("after", after)
			}

			if page := GetInt(args, "page", 0); page > 0 {
				params.Set("page", strconv.Itoa(page))
			}

			if perPage := GetInt(args, "per_page", 0); perPage > 0 {
				params.Set("per_page", strconv.Itoa(perPage))
			}

			endpoint := fmt.Sprintf("/projects/%s/events", url.PathEscape(projectID))
			if len(params) > 0 {
				endpoint += "?" + params.Encode()
			}

			var events []Event
			if err := ctx.Client.Get(endpoint, &events); err != nil {
				return ErrorResult(fmt.Sprintf("failed to get project events: %v", err))
			}

			return JSONResult(events)
		},
	)
}

// initUserTools registers all user-related tools with the MCP server.
// Includes: get_users
func initUserTools(server *mcp.Server) {
	registerGetUsers(server)
}

// initEventTools registers all event-related tools with the MCP server.
// Includes: list_events, get_project_events
func initEventTools(server *mcp.Server) {
	registerListEvents(server)
	registerGetProjectEvents(server)
}
