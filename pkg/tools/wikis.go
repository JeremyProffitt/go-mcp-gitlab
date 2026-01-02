// Package tools provides MCP tool implementations for GitLab wiki operations.
package tools

import (
	"encoding/base64"
	"fmt"
	"net/url"

	"github.com/go-mcp-gitlab/go-mcp-gitlab/pkg/mcp"
)

// WikiPage represents a GitLab wiki page.
type WikiPage struct {
	Content  string `json:"content,omitempty"`
	Format   string `json:"format"`
	Slug     string `json:"slug"`
	Title    string `json:"title"`
	Encoding string `json:"encoding,omitempty"`
}

// WikiAttachmentResponse represents the response from wiki attachment upload.
type WikiAttachmentResponse struct {
	FileName string `json:"file_name"`
	FilePath string `json:"file_path"`
	Branch   string `json:"branch"`
	Link     struct {
		URL      string `json:"url"`
		Markdown string `json:"markdown"`
	} `json:"link"`
}

// registerListWikiPages registers the list_wiki_pages tool.
func registerListWikiPages(server *mcp.Server) {
	server.RegisterTool(
		mcp.Tool{
			Name:        "list_wiki_pages",
			Description: "List all wiki pages for a GitLab project",
			InputSchema: mcp.JSONSchema{
				Type: "object",
				Properties: map[string]mcp.Property{
					"project_id": {
						Type:        "string",
						Description: "The ID or URL-encoded path of the project",
					},
					"with_content": {
						Type:        "boolean",
						Description: "Include page content in the response (optional, default: false)",
					},
					"page": {
						Type:        "integer",
						Description: "Page number for pagination (optional, default: 1)",
					},
					"per_page": {
						Type:        "integer",
						Description: "Number of results per page (optional, default: 20, max: 100)",
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
			ctx.Logger.ToolCall("list_wiki_pages", args)

			// Extract required parameters
			projectID := GetString(args, "project_id", "")
			if projectID == "" {
				return ErrorResult("project_id is required")
			}

			// Extract optional parameters
			withContent := GetBool(args, "with_content", false)
			page := GetInt(args, "page", 0)
			perPage := GetInt(args, "per_page", 0)

			// Build the endpoint with URL-encoded project_id
			encodedProjectID := url.PathEscape(projectID)
			endpoint := fmt.Sprintf("/projects/%s/wikis", encodedProjectID)

			// Build query parameters
			params := url.Values{}
			if withContent {
				params.Set("with_content", "true")
			}
			if page > 0 {
				params.Set("page", fmt.Sprintf("%d", page))
			}
			if perPage > 0 {
				params.Set("per_page", fmt.Sprintf("%d", perPage))
			}

			if len(params) > 0 {
				endpoint = fmt.Sprintf("%s?%s", endpoint, params.Encode())
			}

			// Make API request
			var wikiPages []WikiPage
			if err := ctx.Client.Get(endpoint, &wikiPages); err != nil {
				return ErrorResult(fmt.Sprintf("Failed to list wiki pages: %v", err))
			}

			return JSONResult(wikiPages)
		},
	)
}

// registerGetWikiPage registers the get_wiki_page tool.
func registerGetWikiPage(server *mcp.Server) {
	server.RegisterTool(
		mcp.Tool{
			Name:        "get_wiki_page",
			Description: "Get a specific wiki page from a GitLab project",
			InputSchema: mcp.JSONSchema{
				Type: "object",
				Properties: map[string]mcp.Property{
					"project_id": {
						Type:        "string",
						Description: "The ID or URL-encoded path of the project",
					},
					"slug": {
						Type:        "string",
						Description: "The URL-encoded slug of the wiki page (e.g., 'home' or 'getting-started')",
					},
				},
				Required: []string{"project_id", "slug"},
			},
		},
		func(args map[string]interface{}) (*mcp.CallToolResult, error) {
			ctx := GetContext()
			if ctx == nil {
				return ErrorResult("tool context not initialized")
			}
			ctx.Logger.ToolCall("get_wiki_page", args)

			// Extract required parameters
			projectID := GetString(args, "project_id", "")
			if projectID == "" {
				return ErrorResult("project_id is required")
			}

			slug := GetString(args, "slug", "")
			if slug == "" {
				return ErrorResult("slug is required")
			}

			// Build the endpoint with URL-encoded project_id and slug
			encodedProjectID := url.PathEscape(projectID)
			encodedSlug := url.PathEscape(slug)
			endpoint := fmt.Sprintf("/projects/%s/wikis/%s", encodedProjectID, encodedSlug)

			// Make API request
			var wikiPage WikiPage
			if err := ctx.Client.Get(endpoint, &wikiPage); err != nil {
				return ErrorResult(fmt.Sprintf("Failed to get wiki page: %v", err))
			}

			return JSONResult(wikiPage)
		},
	)
}

// registerCreateWikiPage registers the create_wiki_page tool.
func registerCreateWikiPage(server *mcp.Server) {
	server.RegisterTool(
		mcp.Tool{
			Name:        "create_wiki_page",
			Description: "Create a new wiki page in a GitLab project",
			InputSchema: mcp.JSONSchema{
				Type: "object",
				Properties: map[string]mcp.Property{
					"project_id": {
						Type:        "string",
						Description: "The ID or URL-encoded path of the project",
					},
					"title": {
						Type:        "string",
						Description: "The title of the wiki page",
					},
					"content": {
						Type:        "string",
						Description: "The content of the wiki page",
					},
					"format": {
						Type:        "string",
						Description: "The format of the wiki page: markdown, rdoc, asciidoc, or org (optional, default: markdown)",
					},
				},
				Required: []string{"project_id", "title", "content"},
			},
		},
		func(args map[string]interface{}) (*mcp.CallToolResult, error) {
			ctx := GetContext()
			if ctx == nil {
				return ErrorResult("tool context not initialized")
			}
			ctx.Logger.ToolCall("create_wiki_page", args)

			// Check read-only mode
			if ctx.Config != nil && ctx.Config.ReadOnlyMode {
				return ErrorResult("cannot create wiki page: server is in read-only mode")
			}

			// Extract required parameters
			projectID := GetString(args, "project_id", "")
			if projectID == "" {
				return ErrorResult("project_id is required")
			}

			title := GetString(args, "title", "")
			if title == "" {
				return ErrorResult("title is required")
			}

			content := GetString(args, "content", "")
			if content == "" {
				return ErrorResult("content is required")
			}

			// Extract optional parameters
			format := GetString(args, "format", "")

			// Build the endpoint with URL-encoded project_id
			encodedProjectID := url.PathEscape(projectID)
			endpoint := fmt.Sprintf("/projects/%s/wikis", encodedProjectID)

			// Prepare request body
			requestBody := map[string]interface{}{
				"title":   title,
				"content": content,
			}

			if format != "" {
				// Validate format
				validFormats := map[string]bool{"markdown": true, "rdoc": true, "asciidoc": true, "org": true}
				if !validFormats[format] {
					return ErrorResult("format must be one of: markdown, rdoc, asciidoc, org")
				}
				requestBody["format"] = format
			}

			// Make API request
			var wikiPage WikiPage
			if err := ctx.Client.Post(endpoint, requestBody, &wikiPage); err != nil {
				return ErrorResult(fmt.Sprintf("Failed to create wiki page: %v", err))
			}

			return JSONResult(wikiPage)
		},
	)
}

// registerUpdateWikiPage registers the update_wiki_page tool.
func registerUpdateWikiPage(server *mcp.Server) {
	server.RegisterTool(
		mcp.Tool{
			Name:        "update_wiki_page",
			Description: "Update an existing wiki page in a GitLab project",
			InputSchema: mcp.JSONSchema{
				Type: "object",
				Properties: map[string]mcp.Property{
					"project_id": {
						Type:        "string",
						Description: "The ID or URL-encoded path of the project",
					},
					"slug": {
						Type:        "string",
						Description: "The URL-encoded slug of the wiki page to update",
					},
					"title": {
						Type:        "string",
						Description: "The new title of the wiki page (optional)",
					},
					"content": {
						Type:        "string",
						Description: "The new content of the wiki page (optional)",
					},
					"format": {
						Type:        "string",
						Description: "The format of the wiki page: markdown, rdoc, asciidoc, or org (optional)",
					},
				},
				Required: []string{"project_id", "slug"},
			},
		},
		func(args map[string]interface{}) (*mcp.CallToolResult, error) {
			ctx := GetContext()
			if ctx == nil {
				return ErrorResult("tool context not initialized")
			}
			ctx.Logger.ToolCall("update_wiki_page", args)

			// Check read-only mode
			if ctx.Config != nil && ctx.Config.ReadOnlyMode {
				return ErrorResult("cannot update wiki page: server is in read-only mode")
			}

			// Extract required parameters
			projectID := GetString(args, "project_id", "")
			if projectID == "" {
				return ErrorResult("project_id is required")
			}

			slug := GetString(args, "slug", "")
			if slug == "" {
				return ErrorResult("slug is required")
			}

			// Extract optional parameters
			title := GetString(args, "title", "")
			content := GetString(args, "content", "")
			format := GetString(args, "format", "")

			// At least one update field must be provided
			if title == "" && content == "" && format == "" {
				return ErrorResult("at least one of title, content, or format must be provided")
			}

			// Build the endpoint with URL-encoded project_id and slug
			encodedProjectID := url.PathEscape(projectID)
			encodedSlug := url.PathEscape(slug)
			endpoint := fmt.Sprintf("/projects/%s/wikis/%s", encodedProjectID, encodedSlug)

			// Prepare request body with only provided fields
			requestBody := make(map[string]interface{})

			if title != "" {
				requestBody["title"] = title
			}
			if content != "" {
				requestBody["content"] = content
			}
			if format != "" {
				// Validate format
				validFormats := map[string]bool{"markdown": true, "rdoc": true, "asciidoc": true, "org": true}
				if !validFormats[format] {
					return ErrorResult("format must be one of: markdown, rdoc, asciidoc, org")
				}
				requestBody["format"] = format
			}

			// Make API request
			var wikiPage WikiPage
			if err := ctx.Client.Put(endpoint, requestBody, &wikiPage); err != nil {
				return ErrorResult(fmt.Sprintf("Failed to update wiki page: %v", err))
			}

			return JSONResult(wikiPage)
		},
	)
}

// registerDeleteWikiPage registers the delete_wiki_page tool.
func registerDeleteWikiPage(server *mcp.Server) {
	server.RegisterTool(
		mcp.Tool{
			Name:        "delete_wiki_page",
			Description: "Delete a wiki page from a GitLab project",
			InputSchema: mcp.JSONSchema{
				Type: "object",
				Properties: map[string]mcp.Property{
					"project_id": {
						Type:        "string",
						Description: "The ID or URL-encoded path of the project",
					},
					"slug": {
						Type:        "string",
						Description: "The URL-encoded slug of the wiki page to delete",
					},
				},
				Required: []string{"project_id", "slug"},
			},
		},
		func(args map[string]interface{}) (*mcp.CallToolResult, error) {
			ctx := GetContext()
			if ctx == nil {
				return ErrorResult("tool context not initialized")
			}
			ctx.Logger.ToolCall("delete_wiki_page", args)

			// Check read-only mode
			if ctx.Config != nil && ctx.Config.ReadOnlyMode {
				return ErrorResult("cannot delete wiki page: server is in read-only mode")
			}

			// Extract required parameters
			projectID := GetString(args, "project_id", "")
			if projectID == "" {
				return ErrorResult("project_id is required")
			}

			slug := GetString(args, "slug", "")
			if slug == "" {
				return ErrorResult("slug is required")
			}

			// Build the endpoint with URL-encoded project_id and slug
			encodedProjectID := url.PathEscape(projectID)
			encodedSlug := url.PathEscape(slug)
			endpoint := fmt.Sprintf("/projects/%s/wikis/%s", encodedProjectID, encodedSlug)

			// Make API request (DELETE returns no content on success)
			if err := ctx.Client.Delete(endpoint); err != nil {
				return ErrorResult(fmt.Sprintf("Failed to delete wiki page: %v", err))
			}

			result := map[string]interface{}{
				"message": fmt.Sprintf("Wiki page '%s' successfully deleted", slug),
				"slug":    slug,
			}

			return JSONResult(result)
		},
	)
}

// registerUploadWikiAttachment registers the upload_wiki_attachment tool.
func registerUploadWikiAttachment(server *mcp.Server) {
	server.RegisterTool(
		mcp.Tool{
			Name:        "upload_wiki_attachment",
			Description: "Upload an attachment to a GitLab project wiki and get a markdown link",
			InputSchema: mcp.JSONSchema{
				Type: "object",
				Properties: map[string]mcp.Property{
					"project_id": {
						Type:        "string",
						Description: "The ID or URL-encoded path of the project",
					},
					"file": {
						Type:        "string",
						Description: "The file content encoded as base64",
					},
					"filename": {
						Type:        "string",
						Description: "The name of the file to upload",
					},
					"branch": {
						Type:        "string",
						Description: "The branch to upload to (optional, defaults to wiki default branch)",
					},
				},
				Required: []string{"project_id", "file", "filename"},
			},
		},
		func(args map[string]interface{}) (*mcp.CallToolResult, error) {
			ctx := GetContext()
			if ctx == nil {
				return ErrorResult("tool context not initialized")
			}
			ctx.Logger.ToolCall("upload_wiki_attachment", args)

			// Check read-only mode
			if ctx.Config != nil && ctx.Config.ReadOnlyMode {
				return ErrorResult("cannot upload wiki attachment: server is in read-only mode")
			}

			// Extract required parameters
			projectID := GetString(args, "project_id", "")
			if projectID == "" {
				return ErrorResult("project_id is required")
			}

			fileContent := GetString(args, "file", "")
			if fileContent == "" {
				return ErrorResult("file is required")
			}

			filename := GetString(args, "filename", "")
			if filename == "" {
				return ErrorResult("filename is required")
			}

			// Extract optional parameters
			branch := GetString(args, "branch", "")

			// Validate base64 content
			_, err := base64.StdEncoding.DecodeString(fileContent)
			if err != nil {
				return ErrorResult(fmt.Sprintf("Invalid base64 file content: %v", err))
			}

			// Build the endpoint with URL-encoded project_id
			encodedProjectID := url.PathEscape(projectID)
			endpoint := fmt.Sprintf("/projects/%s/wikis/attachments", encodedProjectID)

			// Prepare request body
			requestBody := map[string]interface{}{
				"file": map[string]interface{}{
					"content":  fileContent,
					"filename": filename,
				},
			}

			if branch != "" {
				requestBody["branch"] = branch
			}

			// Make API request
			var response WikiAttachmentResponse
			if err := ctx.Client.Post(endpoint, requestBody, &response); err != nil {
				return ErrorResult(fmt.Sprintf("Failed to upload wiki attachment: %v", err))
			}

			// Build response
			result := map[string]interface{}{
				"file_name": response.FileName,
				"file_path": response.FilePath,
				"branch":    response.Branch,
				"url":       response.Link.URL,
				"markdown":  response.Link.Markdown,
			}

			return JSONResult(result)
		},
	)
}

// initWikiTools registers all wiki-related tools with the MCP server.
func initWikiTools(server *mcp.Server) {
	registerListWikiPages(server)
	registerGetWikiPage(server)
	registerCreateWikiPage(server)
	registerUpdateWikiPage(server)
	registerDeleteWikiPage(server)
	registerUploadWikiAttachment(server)
}
