// Package tools provides MCP tool implementations for GitLab operations.
package tools

import (
	"fmt"
	"net/url"
	"strconv"

	"github.com/go-mcp-gitlab/go-mcp-gitlab/pkg/gitlab"
	"github.com/go-mcp-gitlab/go-mcp-gitlab/pkg/mcp"
)

// registerCreateBranch registers the create_branch tool.
func registerCreateBranch(server *mcp.Server) {
	server.RegisterTool(
		mcp.Tool{
			Name:        "create_branch",
			Description: "Create a new branch in a GitLab project repository",
			InputSchema: mcp.JSONSchema{
				Type: "object",
				Properties: map[string]mcp.Property{
					"project_id": {
						Type:        "string",
						Description: "The project identifier - either a numeric ID (e.g., 42) or URL-encoded path (e.g., my-group/my-project)",
					},
					"branch": {
						Type:        "string",
						Description: "Name of the new branch to create",
					},
					"ref": {
						Type:        "string",
						Description: "The branch name, tag, or commit SHA to create the branch from",
					},
				},
				Required: []string{"project_id", "branch", "ref"},
			},
		},
		func(args map[string]interface{}) (*mcp.CallToolResult, error) {
			c := GetContext()
			if c == nil {
				return ErrorResult("tool context not initialized")
			}
			c.Logger.ToolCall("create_branch", args)

			projectID := GetString(args, "project_id", "")
			if projectID == "" {
				return ErrorResult("project_id is required")
			}

			branch := GetString(args, "branch", "")
			if branch == "" {
				return ErrorResult("branch is required")
			}

			ref := GetString(args, "ref", "")
			if ref == "" {
				return ErrorResult("ref is required")
			}

			endpoint := fmt.Sprintf("/projects/%s/repository/branches", url.PathEscape(projectID))

			requestBody := map[string]string{
				"branch": branch,
				"ref":    ref,
			}

			var result gitlab.Branch
			if err := c.Client.Post(endpoint, requestBody, &result); err != nil {
				return ErrorResult(fmt.Sprintf("Failed to create branch: %v", err))
			}

			return JSONResult(result)
		},
	)
}

// registerListCommits registers the list_commits tool.
func registerListCommits(server *mcp.Server) {
	server.RegisterTool(
		mcp.Tool{
			Name:        "list_commits",
			Description: "List repository commits in a GitLab project. Returns an array of commit objects with SHA, message, author, and timestamp. Filter by ref_name for specific branch/tag commits.",
			InputSchema: mcp.JSONSchema{
				Type: "object",
				Properties: map[string]mcp.Property{
					"project_id": {
						Type:        "string",
						Description: "The project identifier - either a numeric ID (e.g., 42) or URL-encoded path (e.g., my-group/my-project)",
					},
					"ref_name": {
						Type:        "string",
						Description: "The name of a repository branch, tag, or revision range",
					},
					"since": {
						Type:        "string",
						Description: "Only commits after or on this date (ISO 8601 format)",
					},
					"until": {
						Type:        "string",
						Description: "Only commits before or on this date (ISO 8601 format)",
					},
					"path": {
						Type:        "string",
						Description: "The file path to filter commits by",
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
				},
				Required: []string{"project_id"},
			},
			Annotations: &mcp.ToolAnnotations{
				ReadOnlyHint: true,
			},
		},
		func(args map[string]interface{}) (*mcp.CallToolResult, error) {
			c := GetContext()
			if c == nil {
				return ErrorResult("tool context not initialized")
			}
			c.Logger.ToolCall("list_commits", args)

			projectID := GetString(args, "project_id", "")
			if projectID == "" {
				return ErrorResult("project_id is required")
			}

			endpoint := fmt.Sprintf("/projects/%s/repository/commits", url.PathEscape(projectID))

			// Build query parameters
			params := url.Values{}

			if refName := GetString(args, "ref_name", ""); refName != "" {
				params.Set("ref_name", refName)
			}

			if since := GetString(args, "since", ""); since != "" {
				params.Set("since", since)
			}

			if until := GetString(args, "until", ""); until != "" {
				params.Set("until", until)
			}

			if path := GetString(args, "path", ""); path != "" {
				params.Set("path", path)
			}

			if page := GetInt(args, "page", 0); page > 0 {
				params.Set("page", strconv.Itoa(page))
			}

			if perPage := GetInt(args, "per_page", 0); perPage > 0 {
				params.Set("per_page", strconv.Itoa(perPage))
			}

			if len(params) > 0 {
				endpoint = endpoint + "?" + params.Encode()
			}

			var commits []gitlab.Commit
			if err := c.Client.Get(endpoint, &commits); err != nil {
				return ErrorResult(fmt.Sprintf("Failed to list commits: %v", err))
			}

			return JSONResult(commits)
		},
	)
}

// registerGetCommit registers the get_commit tool.
func registerGetCommit(server *mcp.Server) {
	server.RegisterTool(
		mcp.Tool{
			Name:        "get_commit",
			Description: "Get comprehensive details of a specific commit by SHA or ref name. Returns full commit info including message, author, committer, parent SHAs, and stats.",
			InputSchema: mcp.JSONSchema{
				Type: "object",
				Properties: map[string]mcp.Property{
					"project_id": {
						Type:        "string",
						Description: "The project identifier - either a numeric ID (e.g., 42) or URL-encoded path (e.g., my-group/my-project)",
					},
					"sha": {
						Type:        "string",
						Description: "The commit SHA or ref name (branch/tag)",
					},
				},
				Required: []string{"project_id", "sha"},
			},
			Annotations: &mcp.ToolAnnotations{
				ReadOnlyHint: true,
			},
		},
		func(args map[string]interface{}) (*mcp.CallToolResult, error) {
			c := GetContext()
			if c == nil {
				return ErrorResult("tool context not initialized")
			}
			c.Logger.ToolCall("get_commit", args)

			projectID := GetString(args, "project_id", "")
			if projectID == "" {
				return ErrorResult("project_id is required")
			}

			sha := GetString(args, "sha", "")
			if sha == "" {
				return ErrorResult("sha is required")
			}

			endpoint := fmt.Sprintf("/projects/%s/repository/commits/%s",
				url.PathEscape(projectID),
				url.PathEscape(sha),
			)

			var commit gitlab.Commit
			if err := c.Client.Get(endpoint, &commit); err != nil {
				return ErrorResult(fmt.Sprintf("Failed to get commit: %v", err))
			}

			return JSONResult(commit)
		},
	)
}

// registerGetCommitDiff registers the get_commit_diff tool.
func registerGetCommitDiff(server *mcp.Server) {
	server.RegisterTool(
		mcp.Tool{
			Name:        "get_commit_diff",
			Description: "Get the diff (code changes) of a commit. Returns an array of diff objects showing changed files with old/new paths and line changes.",
			InputSchema: mcp.JSONSchema{
				Type: "object",
				Properties: map[string]mcp.Property{
					"project_id": {
						Type:        "string",
						Description: "The project identifier - either a numeric ID (e.g., 42) or URL-encoded path (e.g., my-group/my-project)",
					},
					"sha": {
						Type:        "string",
						Description: "The commit SHA",
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
				},
				Required: []string{"project_id", "sha"},
			},
			Annotations: &mcp.ToolAnnotations{
				ReadOnlyHint: true,
			},
		},
		func(args map[string]interface{}) (*mcp.CallToolResult, error) {
			c := GetContext()
			if c == nil {
				return ErrorResult("tool context not initialized")
			}
			c.Logger.ToolCall("get_commit_diff", args)

			projectID := GetString(args, "project_id", "")
			if projectID == "" {
				return ErrorResult("project_id is required")
			}

			sha := GetString(args, "sha", "")
			if sha == "" {
				return ErrorResult("sha is required")
			}

			endpoint := fmt.Sprintf("/projects/%s/repository/commits/%s/diff",
				url.PathEscape(projectID),
				url.PathEscape(sha),
			)

			// Build query parameters
			params := url.Values{}

			if page := GetInt(args, "page", 0); page > 0 {
				params.Set("page", strconv.Itoa(page))
			}

			if perPage := GetInt(args, "per_page", 0); perPage > 0 {
				params.Set("per_page", strconv.Itoa(perPage))
			}

			if len(params) > 0 {
				endpoint = endpoint + "?" + params.Encode()
			}

			var diffs []gitlab.Diff
			if err := c.Client.Get(endpoint, &diffs); err != nil {
				return ErrorResult(fmt.Sprintf("Failed to get commit diff: %v", err))
			}

			return JSONResult(diffs)
		},
	)
}

// registerListReleases registers the list_releases tool.
func registerListReleases(server *mcp.Server) {
	server.RegisterTool(
		mcp.Tool{
			Name:        "list_releases",
			Description: "List releases of a GitLab project. Returns an array of release objects with tag name, name, description, and release date. Ordered by released_at by default.",
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
					"order_by": {
						Type:        "string",
						Description: "Order releases by: released_at or created_at (default: released_at)",
						Enum:        []string{"released_at", "created_at"},
					},
					"sort": {
						Type:        "string",
						Description: "Sort direction: asc or desc (default: desc)",
						Enum:        []string{"asc", "desc"},
					},
				},
				Required: []string{"project_id"},
			},
			Annotations: &mcp.ToolAnnotations{
				ReadOnlyHint: true,
			},
		},
		func(args map[string]interface{}) (*mcp.CallToolResult, error) {
			c := GetContext()
			if c == nil {
				return ErrorResult("tool context not initialized")
			}
			c.Logger.ToolCall("list_releases", args)

			projectID := GetString(args, "project_id", "")
			if projectID == "" {
				return ErrorResult("project_id is required")
			}

			endpoint := fmt.Sprintf("/projects/%s/releases", url.PathEscape(projectID))

			// Build query parameters
			params := url.Values{}

			if page := GetInt(args, "page", 0); page > 0 {
				params.Set("page", strconv.Itoa(page))
			}

			if perPage := GetInt(args, "per_page", 0); perPage > 0 {
				params.Set("per_page", strconv.Itoa(perPage))
			}

			if orderBy := GetString(args, "order_by", ""); orderBy != "" {
				params.Set("order_by", orderBy)
			}

			if sort := GetString(args, "sort", ""); sort != "" {
				params.Set("sort", sort)
			}

			if len(params) > 0 {
				endpoint = endpoint + "?" + params.Encode()
			}

			var releases []gitlab.Release
			if err := c.Client.Get(endpoint, &releases); err != nil {
				return ErrorResult(fmt.Sprintf("Failed to list releases: %v", err))
			}

			return JSONResult(releases)
		},
	)
}

// registerDownloadAttachment registers the download_attachment tool.
func registerDownloadAttachment(server *mcp.Server) {
	server.RegisterTool(
		mcp.Tool{
			Name:        "download_attachment",
			Description: "Download an uploaded file/attachment from a GitLab project",
			InputSchema: mcp.JSONSchema{
				Type: "object",
				Properties: map[string]mcp.Property{
					"project_id": {
						Type:        "string",
						Description: "The project identifier - either a numeric ID (e.g., 42) or URL-encoded path (e.g., my-group/my-project)",
					},
					"secret": {
						Type:        "string",
						Description: "The secret identifier of the upload",
					},
					"filename": {
						Type:        "string",
						Description: "The filename of the upload",
					},
				},
				Required: []string{"project_id", "secret", "filename"},
			},
			Annotations: &mcp.ToolAnnotations{
				ReadOnlyHint: true,
			},
		},
		func(args map[string]interface{}) (*mcp.CallToolResult, error) {
			c := GetContext()
			if c == nil {
				return ErrorResult("tool context not initialized")
			}
			c.Logger.ToolCall("download_attachment", args)

			projectID := GetString(args, "project_id", "")
			if projectID == "" {
				return ErrorResult("project_id is required")
			}

			secret := GetString(args, "secret", "")
			if secret == "" {
				return ErrorResult("secret is required")
			}

			filename := GetString(args, "filename", "")
			if filename == "" {
				return ErrorResult("filename is required")
			}

			endpoint := fmt.Sprintf("/projects/%s/uploads/%s/%s",
				url.PathEscape(projectID),
				url.PathEscape(secret),
				url.PathEscape(filename),
			)

			// For file downloads, we get raw content as a string
			var content string
			if err := c.Client.Get(endpoint, &content); err != nil {
				return ErrorResult(fmt.Sprintf("Failed to download attachment: %v", err))
			}

			// Return the file content as text
			return TextResult(content)
		},
	)
}

// RegisterBranchTools registers all branch and commit related tools with the MCP server.
func RegisterBranchTools(server *mcp.Server) {
	registerCreateBranch(server)
	registerListCommits(server)
	registerGetCommit(server)
	registerGetCommitDiff(server)
	registerListReleases(server)
	registerDownloadAttachment(server)
}
