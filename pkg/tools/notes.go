package tools

import (
	"fmt"
	"net/url"

	"github.com/go-mcp-gitlab/go-mcp-gitlab/pkg/gitlab"
	"github.com/go-mcp-gitlab/go-mcp-gitlab/pkg/mcp"
)

// registerUpdateDraftNote registers the update_draft_note tool.
func registerUpdateDraftNote(server *mcp.Server) {
	server.RegisterTool(
		mcp.Tool{
			Name:        "update_draft_note",
			Description: "Update a draft note on a merge request.",
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
						Description: "The ID of the draft note to update",
					},
					"body": {
						Type:        "string",
						Description: "The new content of the draft note",
					},
				},
				Required: []string{"project_id", "merge_request_iid", "draft_note_id", "body"},
			},
		},
		func(args map[string]interface{}) (*mcp.CallToolResult, error) {
			ctx := GetContext()
			if ctx == nil {
				return ErrorResult("tool context not initialized")
			}
			ctx.Logger.ToolCall("update_draft_note", args)

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
			body := GetString(args, "body", "")
			if body == "" {
				return ErrorResult("body is required")
			}

			endpoint := fmt.Sprintf("/projects/%s/merge_requests/%d/draft_notes/%d",
				url.PathEscape(projectID), mrIID, draftNoteID)

			requestBody := map[string]interface{}{
				"note": body,
			}

			var draftNote DraftNote
			if err := ctx.Client.Put(endpoint, requestBody, &draftNote); err != nil {
				return ErrorResult(fmt.Sprintf("failed to update draft note: %v", err))
			}

			return JSONResult(draftNote)
		},
	)
}

// registerDeleteDraftNote registers the delete_draft_note tool.
func registerDeleteDraftNote(server *mcp.Server) {
	server.RegisterTool(
		mcp.Tool{
			Name:        "delete_draft_note",
			Description: "Delete a draft note from a merge request.",
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
						Description: "The ID of the draft note to delete",
					},
				},
				Required: []string{"project_id", "merge_request_iid", "draft_note_id"},
			},
		},
		func(args map[string]interface{}) (*mcp.CallToolResult, error) {
			ctx := GetContext()
			if ctx == nil {
				return ErrorResult("tool context not initialized")
			}
			ctx.Logger.ToolCall("delete_draft_note", args)

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

			if err := ctx.Client.Delete(endpoint); err != nil {
				return ErrorResult(fmt.Sprintf("failed to delete draft note: %v", err))
			}

			return TextResult(fmt.Sprintf("Draft note %d deleted successfully", draftNoteID))
		},
	)
}

// registerPublishDraftNote registers the publish_draft_note tool.
func registerPublishDraftNote(server *mcp.Server) {
	server.RegisterTool(
		mcp.Tool{
			Name:        "publish_draft_note",
			Description: "Publish a single draft note on a merge request.",
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
						Description: "The ID of the draft note to publish",
					},
				},
				Required: []string{"project_id", "merge_request_iid", "draft_note_id"},
			},
		},
		func(args map[string]interface{}) (*mcp.CallToolResult, error) {
			ctx := GetContext()
			if ctx == nil {
				return ErrorResult("tool context not initialized")
			}
			ctx.Logger.ToolCall("publish_draft_note", args)

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

			endpoint := fmt.Sprintf("/projects/%s/merge_requests/%d/draft_notes/%d/publish",
				url.PathEscape(projectID), mrIID, draftNoteID)

			// PUT request with empty body to publish
			var result interface{}
			if err := ctx.Client.Put(endpoint, nil, &result); err != nil {
				return ErrorResult(fmt.Sprintf("failed to publish draft note: %v", err))
			}

			return TextResult(fmt.Sprintf("Draft note %d published successfully", draftNoteID))
		},
	)
}

// registerBulkPublishDraftNotes registers the bulk_publish_draft_notes tool.
func registerBulkPublishDraftNotes(server *mcp.Server) {
	server.RegisterTool(
		mcp.Tool{
			Name:        "bulk_publish_draft_notes",
			Description: "Publish all draft notes for a merge request at once.",
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
			ctx := GetContext()
			if ctx == nil {
				return ErrorResult("tool context not initialized")
			}
			ctx.Logger.ToolCall("bulk_publish_draft_notes", args)

			projectID := GetString(args, "project_id", "")
			if projectID == "" {
				return ErrorResult("project_id is required")
			}
			mrIID := GetInt(args, "merge_request_iid", 0)
			if mrIID == 0 {
				return ErrorResult("merge_request_iid is required")
			}

			endpoint := fmt.Sprintf("/projects/%s/merge_requests/%d/draft_notes/bulk_publish",
				url.PathEscape(projectID), mrIID)

			// POST request with empty body to bulk publish
			var result interface{}
			if err := ctx.Client.Post(endpoint, nil, &result); err != nil {
				return ErrorResult(fmt.Sprintf("failed to bulk publish draft notes: %v", err))
			}

			return TextResult("All draft notes published successfully")
		},
	)
}

// registerUpdateIssueNote registers the update_issue_note tool.
func registerUpdateIssueNote(server *mcp.Server) {
	server.RegisterTool(
		mcp.Tool{
			Name:        "update_issue_note",
			Description: "Update an existing note in an issue discussion.",
			InputSchema: mcp.JSONSchema{
				Type: "object",
				Properties: map[string]mcp.Property{
					"project_id": {
						Type:        "string",
						Description: "The ID or URL-encoded path of the project",
					},
					"issue_iid": {
						Type:        "integer",
						Description: "The internal ID of the issue",
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
				Required: []string{"project_id", "issue_iid", "discussion_id", "note_id", "body"},
			},
		},
		func(args map[string]interface{}) (*mcp.CallToolResult, error) {
			ctx := GetContext()
			if ctx == nil {
				return ErrorResult("tool context not initialized")
			}
			ctx.Logger.ToolCall("update_issue_note", args)

			projectID := GetString(args, "project_id", "")
			if projectID == "" {
				return ErrorResult("project_id is required")
			}
			issueIID := GetInt(args, "issue_iid", 0)
			if issueIID == 0 {
				return ErrorResult("issue_iid is required")
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

			endpoint := fmt.Sprintf("/projects/%s/issues/%d/discussions/%s/notes/%d",
				url.PathEscape(projectID), issueIID, url.PathEscape(discussionID), noteID)

			requestBody := map[string]interface{}{
				"body": body,
			}

			var note gitlab.Note
			if err := ctx.Client.Put(endpoint, requestBody, &note); err != nil {
				return ErrorResult(fmt.Sprintf("failed to update issue note: %v", err))
			}

			return JSONResult(note)
		},
	)
}

// registerCreateIssueNote registers the create_issue_note tool.
func registerCreateIssueNote(server *mcp.Server) {
	server.RegisterTool(
		mcp.Tool{
			Name:        "create_issue_note",
			Description: "Add a new note to an existing issue discussion thread.",
			InputSchema: mcp.JSONSchema{
				Type: "object",
				Properties: map[string]mcp.Property{
					"project_id": {
						Type:        "string",
						Description: "The ID or URL-encoded path of the project",
					},
					"issue_iid": {
						Type:        "integer",
						Description: "The internal ID of the issue",
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
				Required: []string{"project_id", "issue_iid", "discussion_id", "body"},
			},
		},
		func(args map[string]interface{}) (*mcp.CallToolResult, error) {
			ctx := GetContext()
			if ctx == nil {
				return ErrorResult("tool context not initialized")
			}
			ctx.Logger.ToolCall("create_issue_note", args)

			projectID := GetString(args, "project_id", "")
			if projectID == "" {
				return ErrorResult("project_id is required")
			}
			issueIID := GetInt(args, "issue_iid", 0)
			if issueIID == 0 {
				return ErrorResult("issue_iid is required")
			}
			discussionID := GetString(args, "discussion_id", "")
			if discussionID == "" {
				return ErrorResult("discussion_id is required")
			}
			body := GetString(args, "body", "")
			if body == "" {
				return ErrorResult("body is required")
			}

			endpoint := fmt.Sprintf("/projects/%s/issues/%d/discussions/%s/notes",
				url.PathEscape(projectID), issueIID, url.PathEscape(discussionID))

			requestBody := map[string]interface{}{
				"body": body,
			}

			var note gitlab.Note
			if err := ctx.Client.Post(endpoint, requestBody, &note); err != nil {
				return ErrorResult(fmt.Sprintf("failed to create issue note: %v", err))
			}

			return JSONResult(note)
		},
	)
}

// RegisterNoteTools registers all note-related tools with the MCP server.
// Includes: update_draft_note, delete_draft_note, publish_draft_note,
// bulk_publish_draft_notes, update_issue_note, create_issue_note
func RegisterNoteTools(server *mcp.Server) {
	registerUpdateDraftNote(server)
	registerDeleteDraftNote(server)
	registerPublishDraftNote(server)
	registerBulkPublishDraftNotes(server)
	registerUpdateIssueNote(server)
	registerCreateIssueNote(server)
}
