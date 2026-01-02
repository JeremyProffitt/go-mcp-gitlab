package tools

import (
	"fmt"
	"net/url"

	"github.com/go-mcp-gitlab/go-mcp-gitlab/pkg/gitlab"
	"github.com/go-mcp-gitlab/go-mcp-gitlab/pkg/mcp"
)

// DraftNote represents a GitLab draft note.
type DraftNote struct {
	ID                    int    `json:"id"`
	AuthorID              int    `json:"author_id"`
	MergeRequestID        int    `json:"merge_request_id"`
	ResolveDiscussion     bool   `json:"resolve_discussion"`
	DiscussionID          string `json:"discussion_id,omitempty"`
	Note                  string `json:"note"`
	Position              any    `json:"position,omitempty"`
	CommitID              string `json:"commit_id,omitempty"`
	LineCode              string `json:"line_code,omitempty"`
	InReplyToDiscussionID string `json:"in_reply_to_discussion_id,omitempty"`
}

// CompareResult represents a comparison between branches/commits.
type CompareResult struct {
	Commit         *gitlab.Commit  `json:"commit"`
	Commits        []gitlab.Commit `json:"commits"`
	Diffs          []gitlab.Diff   `json:"diffs"`
	CompareTimeout bool            `json:"compare_timeout"`
	CompareSameRef bool            `json:"compare_same_ref"`
}

// Discussion represents a GitLab discussion thread.
type Discussion struct {
	ID             string        `json:"id"`
	IndividualNote bool          `json:"individual_note"`
	Notes          []gitlab.Note `json:"notes"`
}

// registerListMergeRequests registers the list_merge_requests tool.
func registerListMergeRequests(server *mcp.Server) {
	server.RegisterTool(
		mcp.Tool{
			Name:        "list_merge_requests",
			Description: "List merge requests for a project. Returns a paginated list of merge requests with their metadata.",
			InputSchema: mcp.JSONSchema{
				Type: "object",
				Properties: map[string]mcp.Property{
					"project_id": {
						Type:        "string",
						Description: "The ID or URL-encoded path of the project",
					},
					"state": {
						Type:        "string",
						Description: "Filter by state: opened, closed, merged, or all",
						Enum:        []string{"opened", "closed", "merged", "all"},
					},
					"scope": {
						Type:        "string",
						Description: "Filter by scope: created_by_me, assigned_to_me, or all",
						Enum:        []string{"created_by_me", "assigned_to_me", "all"},
					},
					"order_by": {
						Type:        "string",
						Description: "Order by: created_at or updated_at",
						Enum:        []string{"created_at", "updated_at"},
					},
					"sort": {
						Type:        "string",
						Description: "Sort order: asc or desc",
						Enum:        []string{"asc", "desc"},
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
			c := GetContext()
			if c == nil {
				return ErrorResult("tool context not initialized")
			}
			c.Logger.ToolCall("list_merge_requests", args)

			projectID := GetString(args, "project_id", "")
			if projectID == "" {
				return ErrorResult("project_id is required")
			}

			params := url.Values{}
			if state := GetString(args, "state", ""); state != "" {
				params.Set("state", state)
			}
			if scope := GetString(args, "scope", ""); scope != "" {
				params.Set("scope", scope)
			}
			if orderBy := GetString(args, "order_by", ""); orderBy != "" {
				params.Set("order_by", orderBy)
			}
			if sort := GetString(args, "sort", ""); sort != "" {
				params.Set("sort", sort)
			}
			if page := GetInt(args, "page", 0); page > 0 {
				params.Set("page", fmt.Sprintf("%d", page))
			}
			if perPage := GetInt(args, "per_page", 0); perPage > 0 {
				params.Set("per_page", fmt.Sprintf("%d", perPage))
			}

			endpoint := fmt.Sprintf("/projects/%s/merge_requests", url.PathEscape(projectID))
			if len(params) > 0 {
				endpoint += "?" + params.Encode()
			}

			var mergeRequests []gitlab.MergeRequest
			pagination, err := c.Client.GetWithPagination(endpoint, &mergeRequests)
			if err != nil {
				return ErrorResult(fmt.Sprintf("Failed to list merge requests: %v", err))
			}

			result := map[string]interface{}{
				"merge_requests": mergeRequests,
				"pagination":     pagination,
			}

			return JSONResult(result)
		},
	)
}

// registerGetMergeRequest registers the get_merge_request tool.
func registerGetMergeRequest(server *mcp.Server) {
	server.RegisterTool(
		mcp.Tool{
			Name:        "get_merge_request",
			Description: "Get details of a specific merge request by IID or find by branch name.",
			InputSchema: mcp.JSONSchema{
				Type: "object",
				Properties: map[string]mcp.Property{
					"project_id": {
						Type:        "string",
						Description: "The ID or URL-encoded path of the project",
					},
					"merge_request_iid": {
						Type:        "integer",
						Description: "The internal ID of the merge request",
					},
					"branch_name": {
						Type:        "string",
						Description: "The source branch name to find merge requests for",
					},
				},
				Required: []string{"project_id"},
			},
		},
		func(args map[string]interface{}) (*mcp.CallToolResult, error) {
			c := GetContext()
			if c == nil {
				return ErrorResult("tool context not initialized")
			}
			c.Logger.ToolCall("get_merge_request", args)

			projectID := GetString(args, "project_id", "")
			if projectID == "" {
				return ErrorResult("project_id is required")
			}

			mrIID := GetInt(args, "merge_request_iid", 0)
			branchName := GetString(args, "branch_name", "")

			if mrIID == 0 && branchName == "" {
				return ErrorResult("Either merge_request_iid or branch_name is required")
			}

			var mr gitlab.MergeRequest

			if mrIID > 0 {
				endpoint := fmt.Sprintf("/projects/%s/merge_requests/%d", url.PathEscape(projectID), mrIID)
				if err := c.Client.Get(endpoint, &mr); err != nil {
					return ErrorResult(fmt.Sprintf("Failed to get merge request: %v", err))
				}
			} else {
				// Search by branch name
				params := url.Values{}
				params.Set("source_branch", branchName)
				params.Set("per_page", "1")
				endpoint := fmt.Sprintf("/projects/%s/merge_requests?%s", url.PathEscape(projectID), params.Encode())

				var mergeRequests []gitlab.MergeRequest
				if err := c.Client.Get(endpoint, &mergeRequests); err != nil {
					return ErrorResult(fmt.Sprintf("Failed to search merge requests: %v", err))
				}

				if len(mergeRequests) == 0 {
					return ErrorResult(fmt.Sprintf("No merge request found for branch: %s", branchName))
				}
				mr = mergeRequests[0]
			}

			return JSONResult(mr)
		},
	)
}

// registerCreateMergeRequest registers the create_merge_request tool.
func registerCreateMergeRequest(server *mcp.Server) {
	server.RegisterTool(
		mcp.Tool{
			Name:        "create_merge_request",
			Description: "Create a new merge request in a project.",
			InputSchema: mcp.JSONSchema{
				Type: "object",
				Properties: map[string]mcp.Property{
					"project_id": {
						Type:        "string",
						Description: "The ID or URL-encoded path of the project",
					},
					"source_branch": {
						Type:        "string",
						Description: "The source branch for the merge request",
					},
					"target_branch": {
						Type:        "string",
						Description: "The target branch for the merge request",
					},
					"title": {
						Type:        "string",
						Description: "The title of the merge request",
					},
					"description": {
						Type:        "string",
						Description: "The description of the merge request",
					},
					"assignee_id": {
						Type:        "integer",
						Description: "The ID of the user to assign the merge request to",
					},
					"remove_source_branch": {
						Type:        "boolean",
						Description: "Whether to remove the source branch after merge",
					},
				},
				Required: []string{"project_id", "source_branch", "target_branch", "title"},
			},
		},
		func(args map[string]interface{}) (*mcp.CallToolResult, error) {
			c := GetContext()
			if c == nil {
				return ErrorResult("tool context not initialized")
			}
			c.Logger.ToolCall("create_merge_request", args)

			projectID := GetString(args, "project_id", "")
			if projectID == "" {
				return ErrorResult("project_id is required")
			}
			sourceBranch := GetString(args, "source_branch", "")
			if sourceBranch == "" {
				return ErrorResult("source_branch is required")
			}
			targetBranch := GetString(args, "target_branch", "")
			if targetBranch == "" {
				return ErrorResult("target_branch is required")
			}
			title := GetString(args, "title", "")
			if title == "" {
				return ErrorResult("title is required")
			}

			body := map[string]interface{}{
				"source_branch": sourceBranch,
				"target_branch": targetBranch,
				"title":         title,
			}

			if description := GetString(args, "description", ""); description != "" {
				body["description"] = description
			}
			if assigneeID := GetInt(args, "assignee_id", 0); assigneeID > 0 {
				body["assignee_id"] = assigneeID
			}
			if removeSource := GetBool(args, "remove_source_branch", false); removeSource {
				body["remove_source_branch"] = true
			}

			endpoint := fmt.Sprintf("/projects/%s/merge_requests", url.PathEscape(projectID))

			var mr gitlab.MergeRequest
			if err := c.Client.Post(endpoint, body, &mr); err != nil {
				return ErrorResult(fmt.Sprintf("Failed to create merge request: %v", err))
			}

			return JSONResult(mr)
		},
	)
}

// registerUpdateMergeRequest registers the update_merge_request tool.
func registerUpdateMergeRequest(server *mcp.Server) {
	server.RegisterTool(
		mcp.Tool{
			Name:        "update_merge_request",
			Description: "Update an existing merge request.",
			InputSchema: mcp.JSONSchema{
				Type: "object",
				Properties: map[string]mcp.Property{
					"project_id": {
						Type:        "string",
						Description: "The ID or URL-encoded path of the project",
					},
					"merge_request_iid": {
						Type:        "integer",
						Description: "The internal ID of the merge request",
					},
					"title": {
						Type:        "string",
						Description: "The new title for the merge request",
					},
					"description": {
						Type:        "string",
						Description: "The new description for the merge request",
					},
					"target_branch": {
						Type:        "string",
						Description: "The new target branch for the merge request",
					},
					"assignee_id": {
						Type:        "integer",
						Description: "The ID of the user to assign the merge request to",
					},
					"state_event": {
						Type:        "string",
						Description: "State event: close or reopen",
						Enum:        []string{"close", "reopen"},
					},
				},
				Required: []string{"project_id", "merge_request_iid"},
			},
		},
		func(args map[string]interface{}) (*mcp.CallToolResult, error) {
			c := GetContext()
			if c == nil {
				return ErrorResult("tool context not initialized")
			}
			c.Logger.ToolCall("update_merge_request", args)

			projectID := GetString(args, "project_id", "")
			if projectID == "" {
				return ErrorResult("project_id is required")
			}
			mrIID := GetInt(args, "merge_request_iid", 0)
			if mrIID == 0 {
				return ErrorResult("merge_request_iid is required")
			}

			body := make(map[string]interface{})
			if title := GetString(args, "title", ""); title != "" {
				body["title"] = title
			}
			if _, exists := args["description"]; exists {
				body["description"] = args["description"]
			}
			if targetBranch := GetString(args, "target_branch", ""); targetBranch != "" {
				body["target_branch"] = targetBranch
			}
			if assigneeID := GetInt(args, "assignee_id", 0); assigneeID > 0 {
				body["assignee_id"] = assigneeID
			}
			if stateEvent := GetString(args, "state_event", ""); stateEvent != "" {
				body["state_event"] = stateEvent
			}

			endpoint := fmt.Sprintf("/projects/%s/merge_requests/%d", url.PathEscape(projectID), mrIID)

			var mr gitlab.MergeRequest
			if err := c.Client.Put(endpoint, body, &mr); err != nil {
				return ErrorResult(fmt.Sprintf("Failed to update merge request: %v", err))
			}

			return JSONResult(mr)
		},
	)
}

// registerMergeMergeRequest registers the merge_merge_request tool.
func registerMergeMergeRequest(server *mcp.Server) {
	server.RegisterTool(
		mcp.Tool{
			Name:        "merge_merge_request",
			Description: "Merge a merge request.",
			InputSchema: mcp.JSONSchema{
				Type: "object",
				Properties: map[string]mcp.Property{
					"project_id": {
						Type:        "string",
						Description: "The ID or URL-encoded path of the project",
					},
					"merge_request_iid": {
						Type:        "integer",
						Description: "The internal ID of the merge request",
					},
					"merge_commit_message": {
						Type:        "string",
						Description: "Custom merge commit message",
					},
					"squash": {
						Type:        "boolean",
						Description: "Whether to squash commits before merging",
					},
					"should_remove_source_branch": {
						Type:        "boolean",
						Description: "Whether to remove the source branch after merge",
					},
				},
				Required: []string{"project_id", "merge_request_iid"},
			},
		},
		func(args map[string]interface{}) (*mcp.CallToolResult, error) {
			c := GetContext()
			if c == nil {
				return ErrorResult("tool context not initialized")
			}
			c.Logger.ToolCall("merge_merge_request", args)

			projectID := GetString(args, "project_id", "")
			if projectID == "" {
				return ErrorResult("project_id is required")
			}
			mrIID := GetInt(args, "merge_request_iid", 0)
			if mrIID == 0 {
				return ErrorResult("merge_request_iid is required")
			}

			body := make(map[string]interface{})
			if commitMsg := GetString(args, "merge_commit_message", ""); commitMsg != "" {
				body["merge_commit_message"] = commitMsg
			}
			if _, exists := args["squash"]; exists {
				body["squash"] = GetBool(args, "squash", false)
			}
			if _, exists := args["should_remove_source_branch"]; exists {
				body["should_remove_source_branch"] = GetBool(args, "should_remove_source_branch", false)
			}

			endpoint := fmt.Sprintf("/projects/%s/merge_requests/%d/merge", url.PathEscape(projectID), mrIID)

			var mr gitlab.MergeRequest
			if err := c.Client.Put(endpoint, body, &mr); err != nil {
				return ErrorResult(fmt.Sprintf("Failed to merge merge request: %v", err))
			}

			return JSONResult(mr)
		},
	)
}

// registerGetMergeRequestDiffs registers the get_merge_request_diffs tool.
func registerGetMergeRequestDiffs(server *mcp.Server) {
	server.RegisterTool(
		mcp.Tool{
			Name:        "get_merge_request_diffs",
			Description: "Get the diffs for a merge request.",
			InputSchema: mcp.JSONSchema{
				Type: "object",
				Properties: map[string]mcp.Property{
					"project_id": {
						Type:        "string",
						Description: "The ID or URL-encoded path of the project",
					},
					"merge_request_iid": {
						Type:        "integer",
						Description: "The internal ID of the merge request",
					},
				},
				Required: []string{"project_id", "merge_request_iid"},
			},
		},
		func(args map[string]interface{}) (*mcp.CallToolResult, error) {
			c := GetContext()
			if c == nil {
				return ErrorResult("tool context not initialized")
			}
			c.Logger.ToolCall("get_merge_request_diffs", args)

			projectID := GetString(args, "project_id", "")
			if projectID == "" {
				return ErrorResult("project_id is required")
			}
			mrIID := GetInt(args, "merge_request_iid", 0)
			if mrIID == 0 {
				return ErrorResult("merge_request_iid is required")
			}

			endpoint := fmt.Sprintf("/projects/%s/merge_requests/%d/diffs", url.PathEscape(projectID), mrIID)

			var diffs []gitlab.Diff
			if err := c.Client.Get(endpoint, &diffs); err != nil {
				return ErrorResult(fmt.Sprintf("Failed to get merge request diffs: %v", err))
			}

			return JSONResult(diffs)
		},
	)
}

// registerListMergeRequestDiffs registers the list_merge_request_diffs tool.
func registerListMergeRequestDiffs(server *mcp.Server) {
	server.RegisterTool(
		mcp.Tool{
			Name:        "list_merge_request_diffs",
			Description: "List diffs for a merge request with pagination support.",
			InputSchema: mcp.JSONSchema{
				Type: "object",
				Properties: map[string]mcp.Property{
					"project_id": {
						Type:        "string",
						Description: "The ID or URL-encoded path of the project",
					},
					"merge_request_iid": {
						Type:        "integer",
						Description: "The internal ID of the merge request",
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
				Required: []string{"project_id", "merge_request_iid"},
			},
		},
		func(args map[string]interface{}) (*mcp.CallToolResult, error) {
			c := GetContext()
			if c == nil {
				return ErrorResult("tool context not initialized")
			}
			c.Logger.ToolCall("list_merge_request_diffs", args)

			projectID := GetString(args, "project_id", "")
			if projectID == "" {
				return ErrorResult("project_id is required")
			}
			mrIID := GetInt(args, "merge_request_iid", 0)
			if mrIID == 0 {
				return ErrorResult("merge_request_iid is required")
			}

			params := url.Values{}
			if page := GetInt(args, "page", 0); page > 0 {
				params.Set("page", fmt.Sprintf("%d", page))
			}
			if perPage := GetInt(args, "per_page", 0); perPage > 0 {
				params.Set("per_page", fmt.Sprintf("%d", perPage))
			}

			endpoint := fmt.Sprintf("/projects/%s/merge_requests/%d/diffs", url.PathEscape(projectID), mrIID)
			if len(params) > 0 {
				endpoint += "?" + params.Encode()
			}

			var diffs []gitlab.Diff
			pagination, err := c.Client.GetWithPagination(endpoint, &diffs)
			if err != nil {
				return ErrorResult(fmt.Sprintf("Failed to list merge request diffs: %v", err))
			}

			result := map[string]interface{}{
				"diffs":      diffs,
				"pagination": pagination,
			}

			return JSONResult(result)
		},
	)
}

// registerGetBranchDiffs registers the get_branch_diffs tool.
func registerGetBranchDiffs(server *mcp.Server) {
	server.RegisterTool(
		mcp.Tool{
			Name:        "get_branch_diffs",
			Description: "Compare two branches, tags, or commits and get the diff.",
			InputSchema: mcp.JSONSchema{
				Type: "object",
				Properties: map[string]mcp.Property{
					"project_id": {
						Type:        "string",
						Description: "The ID or URL-encoded path of the project",
					},
					"from": {
						Type:        "string",
						Description: "The source branch, tag, or commit SHA",
					},
					"to": {
						Type:        "string",
						Description: "The target branch, tag, or commit SHA",
					},
					"straight": {
						Type:        "boolean",
						Description: "If true, compare from and to without merge base (default: false)",
					},
				},
				Required: []string{"project_id", "from", "to"},
			},
		},
		func(args map[string]interface{}) (*mcp.CallToolResult, error) {
			c := GetContext()
			if c == nil {
				return ErrorResult("tool context not initialized")
			}
			c.Logger.ToolCall("get_branch_diffs", args)

			projectID := GetString(args, "project_id", "")
			if projectID == "" {
				return ErrorResult("project_id is required")
			}
			from := GetString(args, "from", "")
			if from == "" {
				return ErrorResult("from is required")
			}
			to := GetString(args, "to", "")
			if to == "" {
				return ErrorResult("to is required")
			}

			params := url.Values{}
			params.Set("from", from)
			params.Set("to", to)
			if straight := GetBool(args, "straight", false); straight {
				params.Set("straight", "true")
			}

			endpoint := fmt.Sprintf("/projects/%s/repository/compare?%s", url.PathEscape(projectID), params.Encode())

			var result CompareResult
			if err := c.Client.Get(endpoint, &result); err != nil {
				return ErrorResult(fmt.Sprintf("Failed to compare branches: %v", err))
			}

			return JSONResult(result)
		},
	)
}

// registerCreateNote registers the create_note tool.
func registerCreateNote(server *mcp.Server) {
	server.RegisterTool(
		mcp.Tool{
			Name:        "create_note",
			Description: "Create a note (comment) on an issue or merge request.",
			InputSchema: mcp.JSONSchema{
				Type: "object",
				Properties: map[string]mcp.Property{
					"project_id": {
						Type:        "string",
						Description: "The ID or URL-encoded path of the project",
					},
					"noteable_type": {
						Type:        "string",
						Description: "The type of noteable: issue or merge_request",
						Enum:        []string{"issue", "merge_request"},
					},
					"noteable_iid": {
						Type:        "integer",
						Description: "The internal ID of the issue or merge request",
					},
					"body": {
						Type:        "string",
						Description: "The content of the note",
					},
				},
				Required: []string{"project_id", "noteable_type", "noteable_iid", "body"},
			},
		},
		func(args map[string]interface{}) (*mcp.CallToolResult, error) {
			c := GetContext()
			if c == nil {
				return ErrorResult("tool context not initialized")
			}
			c.Logger.ToolCall("create_note", args)

			projectID := GetString(args, "project_id", "")
			if projectID == "" {
				return ErrorResult("project_id is required")
			}
			noteableType := GetString(args, "noteable_type", "")
			if noteableType == "" {
				return ErrorResult("noteable_type is required")
			}
			noteableIID := GetInt(args, "noteable_iid", 0)
			if noteableIID == 0 {
				return ErrorResult("noteable_iid is required")
			}
			body := GetString(args, "body", "")
			if body == "" {
				return ErrorResult("body is required")
			}

			var endpoint string
			switch noteableType {
			case "issue":
				endpoint = fmt.Sprintf("/projects/%s/issues/%d/notes", url.PathEscape(projectID), noteableIID)
			case "merge_request":
				endpoint = fmt.Sprintf("/projects/%s/merge_requests/%d/notes", url.PathEscape(projectID), noteableIID)
			default:
				return ErrorResult("noteable_type must be 'issue' or 'merge_request'")
			}

			requestBody := map[string]interface{}{
				"body": body,
			}

			var note gitlab.Note
			if err := c.Client.Post(endpoint, requestBody, &note); err != nil {
				return ErrorResult(fmt.Sprintf("Failed to create note: %v", err))
			}

			return JSONResult(note)
		},
	)
}

// registerCreateMergeRequestThread registers the create_merge_request_thread tool.
func registerCreateMergeRequestThread(server *mcp.Server) {
	server.RegisterTool(
		mcp.Tool{
			Name:        "create_merge_request_thread",
			Description: "Create a new discussion thread on a merge request, optionally on a specific line of code.",
			InputSchema: mcp.JSONSchema{
				Type: "object",
				Properties: map[string]mcp.Property{
					"project_id": {
						Type:        "string",
						Description: "The ID or URL-encoded path of the project",
					},
					"merge_request_iid": {
						Type:        "integer",
						Description: "The internal ID of the merge request",
					},
					"body": {
						Type:        "string",
						Description: "The content of the discussion",
					},
					"position": {
						Type:        "object",
						Description: "Position information for code discussions",
						Properties: map[string]mcp.Property{
							"base_sha": {
								Type:        "string",
								Description: "Base commit SHA in the source branch",
							},
							"start_sha": {
								Type:        "string",
								Description: "SHA referencing commit in target branch",
							},
							"head_sha": {
								Type:        "string",
								Description: "SHA referencing HEAD of source branch",
							},
							"position_type": {
								Type:        "string",
								Description: "Type of position: text or image",
								Enum:        []string{"text", "image"},
							},
							"new_path": {
								Type:        "string",
								Description: "File path after change",
							},
							"old_path": {
								Type:        "string",
								Description: "File path before change",
							},
							"new_line": {
								Type:        "integer",
								Description: "Line number after change",
							},
							"old_line": {
								Type:        "integer",
								Description: "Line number before change",
							},
						},
					},
				},
				Required: []string{"project_id", "merge_request_iid", "body"},
			},
		},
		func(args map[string]interface{}) (*mcp.CallToolResult, error) {
			c := GetContext()
			if c == nil {
				return ErrorResult("tool context not initialized")
			}
			c.Logger.ToolCall("create_merge_request_thread", args)

			projectID := GetString(args, "project_id", "")
			if projectID == "" {
				return ErrorResult("project_id is required")
			}
			mrIID := GetInt(args, "merge_request_iid", 0)
			if mrIID == 0 {
				return ErrorResult("merge_request_iid is required")
			}
			body := GetString(args, "body", "")
			if body == "" {
				return ErrorResult("body is required")
			}

			requestBody := map[string]interface{}{
				"body": body,
			}

			if position, ok := args["position"].(map[string]interface{}); ok {
				requestBody["position"] = position
			}

			endpoint := fmt.Sprintf("/projects/%s/merge_requests/%d/discussions", url.PathEscape(projectID), mrIID)

			var discussion Discussion
			if err := c.Client.Post(endpoint, requestBody, &discussion); err != nil {
				return ErrorResult(fmt.Sprintf("Failed to create discussion thread: %v", err))
			}

			return JSONResult(discussion)
		},
	)
}

// registerMRDiscussions registers the mr_discussions tool.
func registerMRDiscussions(server *mcp.Server) {
	server.RegisterTool(
		mcp.Tool{
			Name:        "mr_discussions",
			Description: "List all discussions (threads) on a merge request.",
			InputSchema: mcp.JSONSchema{
				Type: "object",
				Properties: map[string]mcp.Property{
					"project_id": {
						Type:        "string",
						Description: "The ID or URL-encoded path of the project",
					},
					"merge_request_iid": {
						Type:        "integer",
						Description: "The internal ID of the merge request",
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
				Required: []string{"project_id", "merge_request_iid"},
			},
		},
		func(args map[string]interface{}) (*mcp.CallToolResult, error) {
			c := GetContext()
			if c == nil {
				return ErrorResult("tool context not initialized")
			}
			c.Logger.ToolCall("mr_discussions", args)

			projectID := GetString(args, "project_id", "")
			if projectID == "" {
				return ErrorResult("project_id is required")
			}
			mrIID := GetInt(args, "merge_request_iid", 0)
			if mrIID == 0 {
				return ErrorResult("merge_request_iid is required")
			}

			params := url.Values{}
			if page := GetInt(args, "page", 0); page > 0 {
				params.Set("page", fmt.Sprintf("%d", page))
			}
			if perPage := GetInt(args, "per_page", 0); perPage > 0 {
				params.Set("per_page", fmt.Sprintf("%d", perPage))
			}

			endpoint := fmt.Sprintf("/projects/%s/merge_requests/%d/discussions", url.PathEscape(projectID), mrIID)
			if len(params) > 0 {
				endpoint += "?" + params.Encode()
			}

			var discussions []Discussion
			pagination, err := c.Client.GetWithPagination(endpoint, &discussions)
			if err != nil {
				return ErrorResult(fmt.Sprintf("Failed to list discussions: %v", err))
			}

			result := map[string]interface{}{
				"discussions": discussions,
				"pagination":  pagination,
			}

			return JSONResult(result)
		},
	)
}

// registerUpdateMergeRequestNote registers the update_merge_request_note tool.
func registerUpdateMergeRequestNote(server *mcp.Server) {
	server.RegisterTool(
		mcp.Tool{
			Name:        "update_merge_request_note",
			Description: "Update an existing note in a merge request discussion.",
			InputSchema: mcp.JSONSchema{
				Type: "object",
				Properties: map[string]mcp.Property{
					"project_id": {
						Type:        "string",
						Description: "The ID or URL-encoded path of the project",
					},
					"merge_request_iid": {
						Type:        "integer",
						Description: "The internal ID of the merge request",
					},
					"discussion_id": {
						Type:        "string",
						Description: "The ID of the discussion",
					},
					"note_id": {
						Type:        "integer",
						Description: "The ID of the note to update",
					},
					"body": {
						Type:        "string",
						Description: "The new content of the note",
					},
				},
				Required: []string{"project_id", "merge_request_iid", "discussion_id", "note_id", "body"},
			},
		},
		func(args map[string]interface{}) (*mcp.CallToolResult, error) {
			c := GetContext()
			if c == nil {
				return ErrorResult("tool context not initialized")
			}
			c.Logger.ToolCall("update_merge_request_note", args)

			projectID := GetString(args, "project_id", "")
			if projectID == "" {
				return ErrorResult("project_id is required")
			}
			mrIID := GetInt(args, "merge_request_iid", 0)
			if mrIID == 0 {
				return ErrorResult("merge_request_iid is required")
			}
			discussionID := GetString(args, "discussion_id", "")
			if discussionID == "" {
				return ErrorResult("discussion_id is required")
			}
			noteID := GetInt(args, "note_id", 0)
			if noteID == 0 {
				return ErrorResult("note_id is required")
			}
			body := GetString(args, "body", "")
			if body == "" {
				return ErrorResult("body is required")
			}

			endpoint := fmt.Sprintf("/projects/%s/merge_requests/%d/discussions/%s/notes/%d",
				url.PathEscape(projectID), mrIID, url.PathEscape(discussionID), noteID)

			requestBody := map[string]interface{}{
				"body": body,
			}

			var note gitlab.Note
			if err := c.Client.Put(endpoint, requestBody, &note); err != nil {
				return ErrorResult(fmt.Sprintf("Failed to update note: %v", err))
			}

			return JSONResult(note)
		},
	)
}

// registerCreateMergeRequestNote registers the create_merge_request_note tool.
func registerCreateMergeRequestNote(server *mcp.Server) {
	server.RegisterTool(
		mcp.Tool{
			Name:        "create_merge_request_note",
			Description: "Add a new note to an existing merge request discussion thread.",
			InputSchema: mcp.JSONSchema{
				Type: "object",
				Properties: map[string]mcp.Property{
					"project_id": {
						Type:        "string",
						Description: "The ID or URL-encoded path of the project",
					},
					"merge_request_iid": {
						Type:        "integer",
						Description: "The internal ID of the merge request",
					},
					"discussion_id": {
						Type:        "string",
						Description: "The ID of the discussion thread",
					},
					"body": {
						Type:        "string",
						Description: "The content of the note",
					},
				},
				Required: []string{"project_id", "merge_request_iid", "discussion_id", "body"},
			},
		},
		func(args map[string]interface{}) (*mcp.CallToolResult, error) {
			c := GetContext()
			if c == nil {
				return ErrorResult("tool context not initialized")
			}
			c.Logger.ToolCall("create_merge_request_note", args)

			projectID := GetString(args, "project_id", "")
			if projectID == "" {
				return ErrorResult("project_id is required")
			}
			mrIID := GetInt(args, "merge_request_iid", 0)
			if mrIID == 0 {
				return ErrorResult("merge_request_iid is required")
			}
			discussionID := GetString(args, "discussion_id", "")
			if discussionID == "" {
				return ErrorResult("discussion_id is required")
			}
			body := GetString(args, "body", "")
			if body == "" {
				return ErrorResult("body is required")
			}

			endpoint := fmt.Sprintf("/projects/%s/merge_requests/%d/discussions/%s/notes",
				url.PathEscape(projectID), mrIID, url.PathEscape(discussionID))

			requestBody := map[string]interface{}{
				"body": body,
			}

			var note gitlab.Note
			if err := c.Client.Post(endpoint, requestBody, &note); err != nil {
				return ErrorResult(fmt.Sprintf("Failed to create note: %v", err))
			}

			return JSONResult(note)
		},
	)
}

// registerListDraftNotes registers the list_draft_notes tool.
func registerListDraftNotes(server *mcp.Server) {
	server.RegisterTool(
		mcp.Tool{
			Name:        "list_draft_notes",
			Description: "List all draft notes for a merge request authored by the current user.",
			InputSchema: mcp.JSONSchema{
				Type: "object",
				Properties: map[string]mcp.Property{
					"project_id": {
						Type:        "string",
						Description: "The ID or URL-encoded path of the project",
					},
					"merge_request_iid": {
						Type:        "integer",
						Description: "The internal ID of the merge request",
					},
				},
				Required: []string{"project_id", "merge_request_iid"},
			},
		},
		func(args map[string]interface{}) (*mcp.CallToolResult, error) {
			c := GetContext()
			if c == nil {
				return ErrorResult("tool context not initialized")
			}
			c.Logger.ToolCall("list_draft_notes", args)

			projectID := GetString(args, "project_id", "")
			if projectID == "" {
				return ErrorResult("project_id is required")
			}
			mrIID := GetInt(args, "merge_request_iid", 0)
			if mrIID == 0 {
				return ErrorResult("merge_request_iid is required")
			}

			endpoint := fmt.Sprintf("/projects/%s/merge_requests/%d/draft_notes", url.PathEscape(projectID), mrIID)

			var draftNotes []DraftNote
			if err := c.Client.Get(endpoint, &draftNotes); err != nil {
				return ErrorResult(fmt.Sprintf("Failed to list draft notes: %v", err))
			}

			return JSONResult(draftNotes)
		},
	)
}

// registerGetDraftNote registers the get_draft_note tool.
func registerGetDraftNote(server *mcp.Server) {
	server.RegisterTool(
		mcp.Tool{
			Name:        "get_draft_note",
			Description: "Get a specific draft note for a merge request.",
			InputSchema: mcp.JSONSchema{
				Type: "object",
				Properties: map[string]mcp.Property{
					"project_id": {
						Type:        "string",
						Description: "The ID or URL-encoded path of the project",
					},
					"merge_request_iid": {
						Type:        "integer",
						Description: "The internal ID of the merge request",
					},
					"draft_note_id": {
						Type:        "integer",
						Description: "The ID of the draft note",
					},
				},
				Required: []string{"project_id", "merge_request_iid", "draft_note_id"},
			},
		},
		func(args map[string]interface{}) (*mcp.CallToolResult, error) {
			c := GetContext()
			if c == nil {
				return ErrorResult("tool context not initialized")
			}
			c.Logger.ToolCall("get_draft_note", args)

			projectID := GetString(args, "project_id", "")
			if projectID == "" {
				return ErrorResult("project_id is required")
			}
			mrIID := GetInt(args, "merge_request_iid", 0)
			if mrIID == 0 {
				return ErrorResult("merge_request_iid is required")
			}
			draftNoteID := GetInt(args, "draft_note_id", 0)
			if draftNoteID == 0 {
				return ErrorResult("draft_note_id is required")
			}

			endpoint := fmt.Sprintf("/projects/%s/merge_requests/%d/draft_notes/%d",
				url.PathEscape(projectID), mrIID, draftNoteID)

			var draftNote DraftNote
			if err := c.Client.Get(endpoint, &draftNote); err != nil {
				return ErrorResult(fmt.Sprintf("Failed to get draft note: %v", err))
			}

			return JSONResult(draftNote)
		},
	)
}

// registerCreateDraftNote registers the create_draft_note tool.
func registerCreateDraftNote(server *mcp.Server) {
	server.RegisterTool(
		mcp.Tool{
			Name:        "create_draft_note",
			Description: "Create a draft note on a merge request. Draft notes are visible only to the author until published.",
			InputSchema: mcp.JSONSchema{
				Type: "object",
				Properties: map[string]mcp.Property{
					"project_id": {
						Type:        "string",
						Description: "The ID or URL-encoded path of the project",
					},
					"merge_request_iid": {
						Type:        "integer",
						Description: "The internal ID of the merge request",
					},
					"body": {
						Type:        "string",
						Description: "The content of the draft note",
					},
					"in_reply_to_discussion_id": {
						Type:        "string",
						Description: "The ID of a discussion to reply to (optional)",
					},
				},
				Required: []string{"project_id", "merge_request_iid", "body"},
			},
		},
		func(args map[string]interface{}) (*mcp.CallToolResult, error) {
			c := GetContext()
			if c == nil {
				return ErrorResult("tool context not initialized")
			}
			c.Logger.ToolCall("create_draft_note", args)

			projectID := GetString(args, "project_id", "")
			if projectID == "" {
				return ErrorResult("project_id is required")
			}
			mrIID := GetInt(args, "merge_request_iid", 0)
			if mrIID == 0 {
				return ErrorResult("merge_request_iid is required")
			}
			body := GetString(args, "body", "")
			if body == "" {
				return ErrorResult("body is required")
			}

			requestBody := map[string]interface{}{
				"note": body,
			}

			if replyTo := GetString(args, "in_reply_to_discussion_id", ""); replyTo != "" {
				requestBody["in_reply_to_discussion_id"] = replyTo
			}

			endpoint := fmt.Sprintf("/projects/%s/merge_requests/%d/draft_notes", url.PathEscape(projectID), mrIID)

			var draftNote DraftNote
			if err := c.Client.Post(endpoint, requestBody, &draftNote); err != nil {
				return ErrorResult(fmt.Sprintf("Failed to create draft note: %v", err))
			}

			return JSONResult(draftNote)
		},
	)
}

// initMergeRequestTools registers all merge request related tools with the MCP server.
// This function is called by RegisterMergeRequestTools in registry.go.
func initMergeRequestTools(server *mcp.Server) {
	registerListMergeRequests(server)
	registerGetMergeRequest(server)
	registerCreateMergeRequest(server)
	registerUpdateMergeRequest(server)
	registerMergeMergeRequest(server)
	registerGetMergeRequestDiffs(server)
	registerListMergeRequestDiffs(server)
	registerGetBranchDiffs(server)
	registerCreateNote(server)
	registerCreateMergeRequestThread(server)
	registerMRDiscussions(server)
	registerUpdateMergeRequestNote(server)
	registerCreateMergeRequestNote(server)
	registerListDraftNotes(server)
	registerGetDraftNote(server)
	registerCreateDraftNote(server)
}
