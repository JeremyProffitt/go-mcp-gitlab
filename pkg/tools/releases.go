// Package tools provides MCP tool implementations for GitLab operations.
package tools

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/go-mcp-gitlab/go-mcp-gitlab/pkg/gitlab"
	"github.com/go-mcp-gitlab/go-mcp-gitlab/pkg/mcp"
)

// ReleaseEvidence represents evidence collected for a release.
type ReleaseEvidence struct {
	SHA         string `json:"sha"`
	Filepath    string `json:"filepath"`
	CollectedAt string `json:"collected_at"`
}

// ReleaseLink represents a link associated with a release.
type ReleaseLink struct {
	ID             int    `json:"id"`
	Name           string `json:"name"`
	URL            string `json:"url"`
	DirectAssetURL string `json:"direct_asset_url"`
	LinkType       string `json:"link_type"`
}

// ReleaseAssets represents the assets of a release.
type ReleaseAssets struct {
	Count   int            `json:"count"`
	Sources []ReleaseSource `json:"sources"`
	Links   []ReleaseLink  `json:"links"`
}

// ReleaseSource represents a source archive for a release.
type ReleaseSource struct {
	Format string `json:"format"`
	URL    string `json:"url"`
}

// ReleaseDetailed represents a detailed GitLab release with all fields.
type ReleaseDetailed struct {
	TagName         string           `json:"tag_name"`
	Name            string           `json:"name"`
	Description     string           `json:"description"`
	DescriptionHTML string           `json:"description_html,omitempty"`
	CreatedAt       string           `json:"created_at"`
	ReleasedAt      string           `json:"released_at"`
	Author          *gitlab.User     `json:"author,omitempty"`
	Commit          *gitlab.Commit   `json:"commit,omitempty"`
	Milestones      []gitlab.Milestone `json:"milestones,omitempty"`
	CommitPath      string           `json:"commit_path,omitempty"`
	TagPath         string           `json:"tag_path,omitempty"`
	Assets          *ReleaseAssets   `json:"assets,omitempty"`
	Evidences       []ReleaseEvidence `json:"evidences,omitempty"`
}

// registerGetRelease registers the get_release tool.
func registerGetRelease(server *mcp.Server) {
	server.RegisterTool(
		mcp.Tool{
			Name:        "get_release",
			Description: "Get details of a specific release in a GitLab project by tag name.",
			InputSchema: mcp.JSONSchema{
				Type: "object",
				Properties: map[string]mcp.Property{
					"project_id": {
						Type:        "string",
						Description: "The ID or URL-encoded path of the project",
					},
					"tag_name": {
						Type:        "string",
						Description: "The tag name of the release (e.g., v1.0.0)",
					},
				},
				Required: []string{"project_id", "tag_name"},
			},
		},
		func(args map[string]interface{}) (*mcp.CallToolResult, error) {
			c := GetContext()
			if c == nil {
				return ErrorResult("tool context not initialized")
			}
			c.Logger.ToolCall("get_release", args)

			projectID := GetString(args, "project_id", "")
			if projectID == "" {
				return ErrorResult("project_id is required")
			}

			tagName := GetString(args, "tag_name", "")
			if tagName == "" {
				return ErrorResult("tag_name is required")
			}

			endpoint := fmt.Sprintf("/projects/%s/releases/%s",
				url.PathEscape(projectID),
				url.PathEscape(tagName),
			)

			var release ReleaseDetailed
			if err := c.Client.Get(endpoint, &release); err != nil {
				return ErrorResult(fmt.Sprintf("Failed to get release: %v", err))
			}

			return JSONResult(release)
		},
	)
}

// registerCreateRelease registers the create_release tool.
func registerCreateRelease(server *mcp.Server) {
	server.RegisterTool(
		mcp.Tool{
			Name:        "create_release",
			Description: "Create a new release in a GitLab project. A release is associated with a tag. If the tag doesn't exist, you can provide a ref (branch or commit) to create the tag from.",
			InputSchema: mcp.JSONSchema{
				Type: "object",
				Properties: map[string]mcp.Property{
					"project_id": {
						Type:        "string",
						Description: "The ID or URL-encoded path of the project",
					},
					"tag_name": {
						Type:        "string",
						Description: "The tag name for the release (e.g., v1.0.0)",
					},
					"name": {
						Type:        "string",
						Description: "The release name (defaults to tag_name if not provided)",
					},
					"description": {
						Type:        "string",
						Description: "The release description (supports Markdown)",
					},
					"ref": {
						Type:        "string",
						Description: "The branch name, tag, or commit SHA to create the tag from if it doesn't exist. Required if the tag doesn't already exist.",
					},
					"milestones": {
						Type:        "array",
						Description: "Array of milestone titles to associate with the release",
						Items:       &mcp.Property{Type: "string"},
					},
					"released_at": {
						Type:        "string",
						Description: "The release date in ISO 8601 format (e.g., 2024-01-15T10:00:00Z). Defaults to the current time.",
					},
				},
				Required: []string{"project_id", "tag_name"},
			},
		},
		func(args map[string]interface{}) (*mcp.CallToolResult, error) {
			c := GetContext()
			if c == nil {
				return ErrorResult("tool context not initialized")
			}
			c.Logger.ToolCall("create_release", args)

			projectID := GetString(args, "project_id", "")
			if projectID == "" {
				return ErrorResult("project_id is required")
			}

			tagName := GetString(args, "tag_name", "")
			if tagName == "" {
				return ErrorResult("tag_name is required")
			}

			// Build request body
			body := map[string]interface{}{
				"tag_name": tagName,
			}

			if name := GetString(args, "name", ""); name != "" {
				body["name"] = name
			}

			if description := GetString(args, "description", ""); description != "" {
				body["description"] = description
			}

			if ref := GetString(args, "ref", ""); ref != "" {
				body["ref"] = ref
			}

			if milestones := GetStringArray(args, "milestones"); len(milestones) > 0 {
				body["milestones"] = milestones
			}

			if releasedAt := GetString(args, "released_at", ""); releasedAt != "" {
				body["released_at"] = releasedAt
			}

			endpoint := fmt.Sprintf("/projects/%s/releases", url.PathEscape(projectID))

			var release ReleaseDetailed
			if err := c.Client.Post(endpoint, body, &release); err != nil {
				return ErrorResult(fmt.Sprintf("Failed to create release: %v", err))
			}

			return JSONResult(release)
		},
	)
}

// registerUpdateRelease registers the update_release tool.
func registerUpdateRelease(server *mcp.Server) {
	server.RegisterTool(
		mcp.Tool{
			Name:        "update_release",
			Description: "Update an existing release in a GitLab project. Only provided fields will be updated.",
			InputSchema: mcp.JSONSchema{
				Type: "object",
				Properties: map[string]mcp.Property{
					"project_id": {
						Type:        "string",
						Description: "The ID or URL-encoded path of the project",
					},
					"tag_name": {
						Type:        "string",
						Description: "The tag name of the release to update",
					},
					"name": {
						Type:        "string",
						Description: "The new release name",
					},
					"description": {
						Type:        "string",
						Description: "The new release description (supports Markdown)",
					},
					"milestones": {
						Type:        "array",
						Description: "Array of milestone titles to associate with the release (replaces existing milestones)",
						Items:       &mcp.Property{Type: "string"},
					},
					"released_at": {
						Type:        "string",
						Description: "The new release date in ISO 8601 format (e.g., 2024-01-15T10:00:00Z)",
					},
				},
				Required: []string{"project_id", "tag_name"},
			},
		},
		func(args map[string]interface{}) (*mcp.CallToolResult, error) {
			c := GetContext()
			if c == nil {
				return ErrorResult("tool context not initialized")
			}
			c.Logger.ToolCall("update_release", args)

			projectID := GetString(args, "project_id", "")
			if projectID == "" {
				return ErrorResult("project_id is required")
			}

			tagName := GetString(args, "tag_name", "")
			if tagName == "" {
				return ErrorResult("tag_name is required")
			}

			// Build request body with only provided fields
			body := make(map[string]interface{})

			if name := GetString(args, "name", ""); name != "" {
				body["name"] = name
			}

			if description, exists := args["description"]; exists {
				body["description"] = description
			}

			if milestones := GetStringArray(args, "milestones"); milestones != nil {
				body["milestones"] = milestones
			}

			if releasedAt := GetString(args, "released_at", ""); releasedAt != "" {
				body["released_at"] = releasedAt
			}

			endpoint := fmt.Sprintf("/projects/%s/releases/%s",
				url.PathEscape(projectID),
				url.PathEscape(tagName),
			)

			var release ReleaseDetailed
			if err := c.Client.Put(endpoint, body, &release); err != nil {
				return ErrorResult(fmt.Sprintf("Failed to update release: %v", err))
			}

			return JSONResult(release)
		},
	)
}

// registerDeleteRelease registers the delete_release tool.
func registerDeleteRelease(server *mcp.Server) {
	server.RegisterTool(
		mcp.Tool{
			Name:        "delete_release",
			Description: "Delete a release from a GitLab project. This only deletes the release, not the associated tag.",
			InputSchema: mcp.JSONSchema{
				Type: "object",
				Properties: map[string]mcp.Property{
					"project_id": {
						Type:        "string",
						Description: "The ID or URL-encoded path of the project",
					},
					"tag_name": {
						Type:        "string",
						Description: "The tag name of the release to delete",
					},
				},
				Required: []string{"project_id", "tag_name"},
			},
		},
		func(args map[string]interface{}) (*mcp.CallToolResult, error) {
			c := GetContext()
			if c == nil {
				return ErrorResult("tool context not initialized")
			}
			c.Logger.ToolCall("delete_release", args)

			projectID := GetString(args, "project_id", "")
			if projectID == "" {
				return ErrorResult("project_id is required")
			}

			tagName := GetString(args, "tag_name", "")
			if tagName == "" {
				return ErrorResult("tag_name is required")
			}

			endpoint := fmt.Sprintf("/projects/%s/releases/%s",
				url.PathEscape(projectID),
				url.PathEscape(tagName),
			)

			if err := c.Client.Delete(endpoint); err != nil {
				return ErrorResult(fmt.Sprintf("Failed to delete release: %v", err))
			}

			return TextResult(fmt.Sprintf("Release '%s' deleted successfully", tagName))
		},
	)
}

// registerCreateReleaseEvidence registers the create_release_evidence tool.
func registerCreateReleaseEvidence(server *mcp.Server) {
	server.RegisterTool(
		mcp.Tool{
			Name:        "create_release_evidence",
			Description: "Create evidence for a release in a GitLab project. Release evidence is a snapshot of release data collected at the time of release creation. This feature requires GitLab Premium or Ultimate.",
			InputSchema: mcp.JSONSchema{
				Type: "object",
				Properties: map[string]mcp.Property{
					"project_id": {
						Type:        "string",
						Description: "The ID or URL-encoded path of the project",
					},
					"tag_name": {
						Type:        "string",
						Description: "The tag name of the release to create evidence for",
					},
				},
				Required: []string{"project_id", "tag_name"},
			},
		},
		func(args map[string]interface{}) (*mcp.CallToolResult, error) {
			c := GetContext()
			if c == nil {
				return ErrorResult("tool context not initialized")
			}
			c.Logger.ToolCall("create_release_evidence", args)

			projectID := GetString(args, "project_id", "")
			if projectID == "" {
				return ErrorResult("project_id is required")
			}

			tagName := GetString(args, "tag_name", "")
			if tagName == "" {
				return ErrorResult("tag_name is required")
			}

			endpoint := fmt.Sprintf("/projects/%s/releases/%s/evidence",
				url.PathEscape(projectID),
				url.PathEscape(tagName),
			)

			// POST with empty body
			var result interface{}
			if err := c.Client.Post(endpoint, nil, &result); err != nil {
				return ErrorResult(fmt.Sprintf("Failed to create release evidence: %v", err))
			}

			// The API returns 201 Created with empty body on success
			if result == nil {
				return TextResult(fmt.Sprintf("Release evidence created successfully for release '%s'", tagName))
			}

			return JSONResult(result)
		},
	)
}

// registerDownloadReleaseAsset registers the download_release_asset tool.
func registerDownloadReleaseAsset(server *mcp.Server) {
	server.RegisterTool(
		mcp.Tool{
			Name:        "download_release_asset",
			Description: "Download a release asset from a GitLab project. Use the direct_asset_url from the release's assets.links array.",
			InputSchema: mcp.JSONSchema{
				Type: "object",
				Properties: map[string]mcp.Property{
					"project_id": {
						Type:        "string",
						Description: "The ID or URL-encoded path of the project",
					},
					"tag_name": {
						Type:        "string",
						Description: "The tag name of the release",
					},
					"asset_link_url": {
						Type:        "string",
						Description: "The direct_asset_url from the release's assets.links array. This is the full URL to the asset.",
					},
				},
				Required: []string{"project_id", "tag_name", "asset_link_url"},
			},
		},
		func(args map[string]interface{}) (*mcp.CallToolResult, error) {
			c := GetContext()
			if c == nil {
				return ErrorResult("tool context not initialized")
			}
			c.Logger.ToolCall("download_release_asset", args)

			projectID := GetString(args, "project_id", "")
			if projectID == "" {
				return ErrorResult("project_id is required")
			}

			tagName := GetString(args, "tag_name", "")
			if tagName == "" {
				return ErrorResult("tag_name is required")
			}

			assetLinkURL := GetString(args, "asset_link_url", "")
			if assetLinkURL == "" {
				return ErrorResult("asset_link_url is required")
			}

			// Parse the asset URL to extract the path after /api/v4
			// The direct_asset_url typically looks like:
			// https://gitlab.example.com/api/v4/projects/1/packages/generic/mypackage/0.0.1/file.txt
			// or for project uploads:
			// https://gitlab.example.com/group/project/-/releases/v1.0/downloads/file.txt

			// Get the base URL from the client
			baseURL := c.Client.BaseURL()

			// Try to extract the relative path from the asset URL
			var endpoint string

			// Check if it's an API URL
			if strings.Contains(assetLinkURL, "/api/v4/") {
				// Extract the path after /api/v4
				parts := strings.SplitN(assetLinkURL, "/api/v4", 2)
				if len(parts) == 2 {
					endpoint = parts[1]
				} else {
					return ErrorResult("invalid asset_link_url format: could not extract API path")
				}
			} else if strings.Contains(assetLinkURL, "/-/releases/") {
				// This is a project release download URL, need to redirect through the release downloads endpoint
				// Format: /projects/:id/releases/:tag_name/downloads/:filepath
				// We need to parse out the filepath from the URL
				parts := strings.SplitN(assetLinkURL, "/downloads/", 2)
				if len(parts) == 2 {
					filepath := parts[1]
					endpoint = fmt.Sprintf("/projects/%s/releases/%s/downloads/%s",
						url.PathEscape(projectID),
						url.PathEscape(tagName),
						filepath,
					)
				} else {
					return ErrorResult("invalid asset_link_url format: could not extract download path")
				}
			} else {
				// If it's a full URL not matching known patterns, try to use it as-is
				// but extract just the path component
				parsedURL, err := url.Parse(assetLinkURL)
				if err != nil {
					return ErrorResult(fmt.Sprintf("invalid asset_link_url: %v", err))
				}
				endpoint = parsedURL.Path

				// Remove /api/v4 prefix if present
				endpoint = strings.TrimPrefix(endpoint, "/api/v4")
			}

			// Validate we have an endpoint
			if endpoint == "" {
				return ErrorResult("could not determine API endpoint from asset_link_url")
			}

			// Log the computed endpoint for debugging
			c.Logger.Debug("downloading release asset: baseURL=%s endpoint=%s", baseURL, endpoint)

			// Download the asset content
			var content string
			if err := c.Client.Get(endpoint, &content); err != nil {
				return ErrorResult(fmt.Sprintf("Failed to download release asset: %v", err))
			}

			return TextResult(content)
		},
	)
}

// initReleaseTools registers all release-related tools with the MCP server.
// Note: list_releases is already registered in branches.go
func initReleaseTools(server *mcp.Server) {
	registerGetRelease(server)
	registerCreateRelease(server)
	registerUpdateRelease(server)
	registerDeleteRelease(server)
	registerCreateReleaseEvidence(server)
	registerDownloadReleaseAsset(server)
}
