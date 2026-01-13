// Package tools provides MCP tool implementations for GitLab operations.
package tools

import (
	"encoding/base64"
	"fmt"
	"net/url"

	"github.com/go-mcp-gitlab/go-mcp-gitlab/pkg/gitlab"
	"github.com/go-mcp-gitlab/go-mcp-gitlab/pkg/mcp"
)

// FileResponse represents the GitLab API response for file operations.
type FileResponse struct {
	FileName      string `json:"file_name"`
	FilePath      string `json:"file_path"`
	Size          int    `json:"size"`
	Encoding      string `json:"encoding"`
	Content       string `json:"content"`
	ContentSHA256 string `json:"content_sha256"`
	Ref           string `json:"ref"`
	BlobID        string `json:"blob_id"`
	CommitID      string `json:"commit_id"`
	LastCommitID  string `json:"last_commit_id"`
}

// FileCreateUpdateResponse represents the response from file create/update operations.
type FileCreateUpdateResponse struct {
	FilePath string `json:"file_path"`
	Branch   string `json:"branch"`
}

// CommitAction represents an action to perform in a commit.
type CommitAction struct {
	Action   string `json:"action"`
	FilePath string `json:"file_path"`
	Content  string `json:"content,omitempty"`
	Encoding string `json:"encoding,omitempty"`
}

// CommitRequest represents a request to create a commit with multiple file changes.
type CommitRequest struct {
	Branch        string         `json:"branch"`
	CommitMessage string         `json:"commit_message"`
	Actions       []CommitAction `json:"actions"`
	AuthorEmail   string         `json:"author_email,omitempty"`
	AuthorName    string         `json:"author_name,omitempty"`
}

// CommitResponse represents the response from a commit creation.
type CommitResponse struct {
	ID             string `json:"id"`
	ShortID        string `json:"short_id"`
	Title          string `json:"title"`
	Message        string `json:"message"`
	AuthorName     string `json:"author_name"`
	AuthorEmail    string `json:"author_email"`
	CommitterName  string `json:"committer_name"`
	CommitterEmail string `json:"committer_email"`
	WebURL         string `json:"web_url"`
}

// UploadResponse represents the response from a file upload.
type UploadResponse struct {
	Alt      string `json:"alt"`
	URL      string `json:"url"`
	FullPath string `json:"full_path"`
	Markdown string `json:"markdown"`
}

// registerGetFileContents registers the get_file_contents tool.
func registerGetFileContents(server *mcp.Server) {
	server.RegisterTool(
		mcp.Tool{
			Name:        "get_file_contents",
			Description: "Get the contents of a file from a GitLab repository. Returns file content (decoded from base64), file metadata, blob ID, and last commit ID. Use ref to get file from specific branch/tag/commit.",
			InputSchema: mcp.JSONSchema{
				Type: "object",
				Properties: map[string]mcp.Property{
					"project_id": {
						Type:        "string",
						Description: "The project identifier - either a numeric ID (e.g., 42) or URL-encoded path (e.g., my-group/my-project)",
					},
					"file_path": {
						Type:        "string",
						Description: "The path of the file in the repository (URL-encoded automatically)",
					},
					"ref": {
						Type:        "string",
						Description: "The name of branch, tag, or commit (optional, defaults to default branch)",
					},
				},
				Required: []string{"project_id", "file_path"},
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
			ctx.Logger.ToolCall("get_file_contents", args)

			// Extract required parameters
			projectID := GetString(args, "project_id", "")
			if projectID == "" {
				return ErrorResult("project_id is required")
			}

			filePath := GetString(args, "file_path", "")
			if filePath == "" {
				return ErrorResult("file_path is required")
			}

			// Extract optional parameters
			ref := GetString(args, "ref", "")

			// Build the endpoint with URL-encoded project_id and file_path
			encodedProjectID := url.PathEscape(projectID)
			encodedFilePath := url.PathEscape(filePath)
			endpoint := fmt.Sprintf("/projects/%s/repository/files/%s", encodedProjectID, encodedFilePath)

			// Add ref query parameter if provided
			if ref != "" {
				endpoint = fmt.Sprintf("%s?ref=%s", endpoint, url.QueryEscape(ref))
			} else {
				// ref is required by the API, use HEAD as default
				endpoint = fmt.Sprintf("%s?ref=HEAD", endpoint)
			}

			// Make API request
			var fileResp FileResponse
			if err := ctx.Client.Get(endpoint, &fileResp); err != nil {
				return ErrorResult(fmt.Sprintf("Failed to get file contents: %v", err))
			}

			// Decode base64 content
			decodedContent, err := base64.StdEncoding.DecodeString(fileResp.Content)
			if err != nil {
				return ErrorResult(fmt.Sprintf("Failed to decode file content: %v", err))
			}

			// Build response
			result := map[string]interface{}{
				"file_name":      fileResp.FileName,
				"file_path":      fileResp.FilePath,
				"size":           fileResp.Size,
				"ref":            fileResp.Ref,
				"blob_id":        fileResp.BlobID,
				"commit_id":      fileResp.CommitID,
				"last_commit_id": fileResp.LastCommitID,
				"content_sha256": fileResp.ContentSHA256,
				"content":        string(decodedContent),
			}

			return JSONResult(result)
		},
	)
}

// registerCreateOrUpdateFile registers the create_or_update_file tool.
func registerCreateOrUpdateFile(server *mcp.Server) {
	server.RegisterTool(
		mcp.Tool{
			Name:        "create_or_update_file",
			Description: "Create a new file or update an existing file in a GitLab repository",
			InputSchema: mcp.JSONSchema{
				Type: "object",
				Properties: map[string]mcp.Property{
					"project_id": {
						Type:        "string",
						Description: "The project identifier - either a numeric ID (e.g., 42) or URL-encoded path (e.g., my-group/my-project)",
					},
					"file_path": {
						Type:        "string",
						Description: "The path of the file in the repository",
					},
					"content": {
						Type:        "string",
						Description: "The file content",
					},
					"branch": {
						Type:        "string",
						Description: "The name of the branch to commit to",
					},
					"commit_message": {
						Type:        "string",
						Description: "The commit message",
					},
					"author_email": {
						Type:        "string",
						Description: "The commit author's email address (optional)",
					},
					"author_name": {
						Type:        "string",
						Description: "The commit author's name (optional)",
					},
				},
				Required: []string{"project_id", "file_path", "content", "branch", "commit_message"},
			},
		},
		func(args map[string]interface{}) (*mcp.CallToolResult, error) {
			ctx := GetContext()
			if ctx == nil {
				return ErrorResult("tool context not initialized")
			}
			ctx.Logger.ToolCall("create_or_update_file", args)

			// Extract required parameters
			projectID := GetString(args, "project_id", "")
			if projectID == "" {
				return ErrorResult("project_id is required")
			}

			filePath := GetString(args, "file_path", "")
			if filePath == "" {
				return ErrorResult("file_path is required")
			}

			content := GetString(args, "content", "")
			// Note: content can be empty (creating an empty file is valid)

			branch := GetString(args, "branch", "")
			if branch == "" {
				return ErrorResult("branch is required")
			}

			commitMessage := GetString(args, "commit_message", "")
			if commitMessage == "" {
				return ErrorResult("commit_message is required")
			}

			// Extract optional parameters
			authorEmail := GetString(args, "author_email", "")
			authorName := GetString(args, "author_name", "")

			// Build the endpoint with URL-encoded project_id and file_path
			encodedProjectID := url.PathEscape(projectID)
			encodedFilePath := url.PathEscape(filePath)
			endpoint := fmt.Sprintf("/projects/%s/repository/files/%s", encodedProjectID, encodedFilePath)

			// Check if file exists to determine whether to POST (create) or PUT (update)
			checkEndpoint := fmt.Sprintf("%s?ref=%s", endpoint, url.QueryEscape(branch))
			var existingFile FileResponse
			fileExists := true
			if err := ctx.Client.Get(checkEndpoint, &existingFile); err != nil {
				if gitlab.IsNotFound(err) {
					fileExists = false
				} else {
					// For other errors, assume file doesn't exist and try to create
					fileExists = false
				}
			}

			// Encode content as base64
			encodedContent := base64.StdEncoding.EncodeToString([]byte(content))

			// Prepare request body
			requestBody := map[string]interface{}{
				"branch":         branch,
				"content":        encodedContent,
				"commit_message": commitMessage,
				"encoding":       "base64",
			}

			if authorEmail != "" {
				requestBody["author_email"] = authorEmail
			}
			if authorName != "" {
				requestBody["author_name"] = authorName
			}

			var response FileCreateUpdateResponse
			var action string

			if fileExists {
				// Update existing file with PUT
				action = "updated"
				if err := ctx.Client.Put(endpoint, requestBody, &response); err != nil {
					return ErrorResult(fmt.Sprintf("Failed to update file: %v", err))
				}
			} else {
				// Create new file with POST
				action = "created"
				if err := ctx.Client.Post(endpoint, requestBody, &response); err != nil {
					return ErrorResult(fmt.Sprintf("Failed to create file: %v", err))
				}
			}

			// Build response
			result := map[string]interface{}{
				"action":    action,
				"file_path": response.FilePath,
				"branch":    response.Branch,
				"message":   fmt.Sprintf("File %s successfully %s", filePath, action),
			}

			return JSONResult(result)
		},
	)
}

// registerPushFiles registers the push_files tool.
func registerPushFiles(server *mcp.Server) {
	server.RegisterTool(
		mcp.Tool{
			Name:        "push_files",
			Description: "Push multiple files to a GitLab repository in a single commit",
			InputSchema: mcp.JSONSchema{
				Type: "object",
				Properties: map[string]mcp.Property{
					"project_id": {
						Type:        "string",
						Description: "The project identifier - either a numeric ID (e.g., 42) or URL-encoded path (e.g., my-group/my-project)",
					},
					"branch": {
						Type:        "string",
						Description: "The name of the branch to commit to",
					},
					"commit_message": {
						Type:        "string",
						Description: "The commit message",
					},
					"actions": {
						Type:        "array",
						Description: "Array of file actions to perform",
						Items: &mcp.Property{
							Type: "object",
							Properties: map[string]mcp.Property{
								"action": {
									Type:        "string",
									Description: "The action to perform: create, update, or delete",
								},
								"file_path": {
									Type:        "string",
									Description: "The path of the file",
								},
								"content": {
									Type:        "string",
									Description: "The file content (not required for delete action)",
								},
							},
						},
					},
					"author_email": {
						Type:        "string",
						Description: "The commit author's email address (optional)",
					},
					"author_name": {
						Type:        "string",
						Description: "The commit author's name (optional)",
					},
				},
				Required: []string{"project_id", "branch", "commit_message", "actions"},
			},
		},
		func(args map[string]interface{}) (*mcp.CallToolResult, error) {
			ctx := GetContext()
			if ctx == nil {
				return ErrorResult("tool context not initialized")
			}
			ctx.Logger.ToolCall("push_files", args)

			// Extract required parameters
			projectID := GetString(args, "project_id", "")
			if projectID == "" {
				return ErrorResult("project_id is required")
			}

			branch := GetString(args, "branch", "")
			if branch == "" {
				return ErrorResult("branch is required")
			}

			commitMessage := GetString(args, "commit_message", "")
			if commitMessage == "" {
				return ErrorResult("commit_message is required")
			}

			actionsRaw, ok := args["actions"]
			if !ok {
				return ErrorResult("actions is required")
			}

			// Extract optional parameters
			authorEmail := GetString(args, "author_email", "")
			authorName := GetString(args, "author_name", "")

			// Parse actions
			actions, err := parseCommitActions(actionsRaw)
			if err != nil {
				return ErrorResult(fmt.Sprintf("Invalid actions parameter: %v", err))
			}

			// Encode content for each action that has content
			for i := range actions {
				if actions[i].Content != "" && actions[i].Action != "delete" {
					actions[i].Content = base64.StdEncoding.EncodeToString([]byte(actions[i].Content))
					actions[i].Encoding = "base64"
				}
			}

			// Build the endpoint with URL-encoded project_id
			encodedProjectID := url.PathEscape(projectID)
			endpoint := fmt.Sprintf("/projects/%s/repository/commits", encodedProjectID)

			// Prepare request body
			commitRequest := CommitRequest{
				Branch:        branch,
				CommitMessage: commitMessage,
				Actions:       actions,
				AuthorEmail:   authorEmail,
				AuthorName:    authorName,
			}

			var response CommitResponse
			if err := ctx.Client.Post(endpoint, commitRequest, &response); err != nil {
				return ErrorResult(fmt.Sprintf("Failed to push files: %v", err))
			}

			// Build response
			result := map[string]interface{}{
				"commit_id":    response.ID,
				"short_id":     response.ShortID,
				"title":        response.Title,
				"message":      response.Message,
				"author_name":  response.AuthorName,
				"author_email": response.AuthorEmail,
				"web_url":      response.WebURL,
				"files_count":  len(actions),
			}

			return JSONResult(result)
		},
	)
}

// registerUploadMarkdown registers the upload_markdown tool.
func registerUploadMarkdown(server *mcp.Server) {
	server.RegisterTool(
		mcp.Tool{
			Name:        "upload_markdown",
			Description: "Upload a file to a GitLab project and get a markdown link for use in issues/MRs",
			InputSchema: mcp.JSONSchema{
				Type: "object",
				Properties: map[string]mcp.Property{
					"project_id": {
						Type:        "string",
						Description: "The project identifier - either a numeric ID (e.g., 42) or URL-encoded path (e.g., my-group/my-project)",
					},
					"file": {
						Type:        "string",
						Description: "The file content encoded as base64",
					},
					"filename": {
						Type:        "string",
						Description: "The name of the file to upload",
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
			ctx.Logger.ToolCall("upload_markdown", args)

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

			// Decode base64 file content
			decodedContent, err := base64.StdEncoding.DecodeString(fileContent)
			if err != nil {
				return ErrorResult(fmt.Sprintf("Failed to decode file content: %v", err))
			}

			// Build the endpoint with URL-encoded project_id
			encodedProjectID := url.PathEscape(projectID)
			endpoint := fmt.Sprintf("/projects/%s/uploads", encodedProjectID)

			// Prepare request body
			// Note: GitLab's uploads API typically requires multipart/form-data
			// For simplicity, we'll use a JSON body approach that some GitLab versions support
			requestBody := map[string]interface{}{
				"file":     base64.StdEncoding.EncodeToString(decodedContent),
				"filename": filename,
			}

			var response UploadResponse
			if err := ctx.Client.Post(endpoint, requestBody, &response); err != nil {
				return ErrorResult(fmt.Sprintf("Failed to upload file: %v", err))
			}

			// Build response
			result := map[string]interface{}{
				"alt":       response.Alt,
				"url":       response.URL,
				"full_path": response.FullPath,
				"markdown":  response.Markdown,
			}

			return JSONResult(result)
		},
	)
}

// RegisterFileTools registers all file-related tools with the MCP server.
// Includes: get_file_contents, create_or_update_file, push_files, upload_markdown
func RegisterFileTools(server *mcp.Server) {
	registerGetFileContents(server)
	registerCreateOrUpdateFile(server)
	registerPushFiles(server)
	registerUploadMarkdown(server)
}

// parseCommitActions parses the actions parameter into a slice of CommitAction.
func parseCommitActions(actionsRaw interface{}) ([]CommitAction, error) {
	actionsSlice, ok := actionsRaw.([]interface{})
	if !ok {
		return nil, fmt.Errorf("actions must be an array")
	}

	actions := make([]CommitAction, 0, len(actionsSlice))
	for i, actionRaw := range actionsSlice {
		actionMap, ok := actionRaw.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("action at index %d must be an object", i)
		}

		action := CommitAction{}

		// Get action type
		actionType, ok := actionMap["action"].(string)
		if !ok {
			return nil, fmt.Errorf("action at index %d missing required 'action' field", i)
		}
		action.Action = actionType

		// Get file path
		filePath, ok := actionMap["file_path"].(string)
		if !ok {
			return nil, fmt.Errorf("action at index %d missing required 'file_path' field", i)
		}
		action.FilePath = filePath

		// Get content (optional for delete)
		if content, ok := actionMap["content"].(string); ok {
			action.Content = content
		} else if actionType != "delete" {
			return nil, fmt.Errorf("action at index %d missing required 'content' field for %s action", i, actionType)
		}

		actions = append(actions, action)
	}

	if len(actions) == 0 {
		return nil, fmt.Errorf("actions array cannot be empty")
	}

	return actions, nil
}
