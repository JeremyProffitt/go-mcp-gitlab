package tools

import (
	"fmt"
	"net/url"

	"github.com/go-mcp-gitlab/go-mcp-gitlab/pkg/gitlab"
	"github.com/go-mcp-gitlab/go-mcp-gitlab/pkg/mcp"
)

// registerGetProject registers the get_project tool
func registerGetProject(server *mcp.Server) {
	server.RegisterTool(
		mcp.Tool{
			Name:        "get_project",
			Description: "Get details of a specific GitLab project by ID or path",
			InputSchema: mcp.JSONSchema{
				Type: "object",
				Properties: map[string]mcp.Property{
					"project_id": {
						Type:        "string",
						Description: "The ID or URL-encoded path of the project",
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
			c.Logger.ToolCall("get_project", args)

			projectID := GetString(args, "project_id", "")
			if projectID == "" {
				return ErrorResult("project_id is required")
			}

			endpoint := fmt.Sprintf("/projects/%s", url.PathEscape(projectID))

			var project gitlab.Project
			if err := c.Client.Get(endpoint, &project); err != nil {
				return ErrorResult(fmt.Sprintf("Failed to get project: %v", err))
			}

			return JSONResult(project)
		},
	)
}

// registerListProjects registers the list_projects tool
func registerListProjects(server *mcp.Server) {
	server.RegisterTool(
		mcp.Tool{
			Name:        "list_projects",
			Description: "List all projects visible to the authenticated user. If GITLAB_DEFAULT_NAMESPACE is configured, lists projects within that namespace by default.",
			InputSchema: mcp.JSONSchema{
				Type: "object",
				Properties: map[string]mcp.Property{
					"namespace": {
						Type:        "string",
						Description: "Namespace/group ID or path to list projects from. Overrides GITLAB_DEFAULT_NAMESPACE if set.",
					},
					"page": {
						Type:        "integer",
						Description: "Page number for pagination (default: 1)",
					},
					"per_page": {
						Type:        "integer",
						Description: "Number of items per page (default: 20, max: 100)",
					},
					"search": {
						Type:        "string",
						Description: "Search term to filter projects by name",
					},
					"visibility": {
						Type:        "string",
						Description: "Filter by visibility: private, internal, or public",
						Enum:        []string{"private", "internal", "public"},
					},
					"order_by": {
						Type:        "string",
						Description: "Order by: id, name, path, created_at, updated_at, last_activity_at",
						Enum:        []string{"id", "name", "path", "created_at", "updated_at", "last_activity_at"},
					},
					"sort": {
						Type:        "string",
						Description: "Sort direction: asc or desc",
						Enum:        []string{"asc", "desc"},
					},
				},
			},
		},
		func(args map[string]interface{}) (*mcp.CallToolResult, error) {
			c := GetContext()
			if c == nil {
				return ErrorResult("tool context not initialized")
			}
			c.Logger.ToolCall("list_projects", args)

			// Determine namespace: explicit arg > config default > none
			namespace := GetString(args, "namespace", "")
			if namespace == "" && c.Config.DefaultNamespace != "" {
				namespace = c.Config.DefaultNamespace
			}

			params := url.Values{}

			if page := GetInt(args, "page", 0); page > 0 {
				params.Set("page", fmt.Sprintf("%d", page))
			}
			if perPage := GetInt(args, "per_page", 0); perPage > 0 {
				params.Set("per_page", fmt.Sprintf("%d", perPage))
			}
			if search := GetString(args, "search", ""); search != "" {
				params.Set("search", search)
			}
			if visibility := GetString(args, "visibility", ""); visibility != "" {
				params.Set("visibility", visibility)
			}
			if orderBy := GetString(args, "order_by", ""); orderBy != "" {
				params.Set("order_by", orderBy)
			}
			if sort := GetString(args, "sort", ""); sort != "" {
				params.Set("sort", sort)
			}

			// Use group endpoint if namespace is set, otherwise list all projects
			var endpoint string
			if namespace != "" {
				endpoint = fmt.Sprintf("/groups/%s/projects", url.PathEscape(namespace))
			} else {
				endpoint = "/projects"
			}
			if len(params) > 0 {
				endpoint = fmt.Sprintf("%s?%s", endpoint, params.Encode())
			}

			var projects []gitlab.Project
			if err := c.Client.Get(endpoint, &projects); err != nil {
				return ErrorResult(fmt.Sprintf("Failed to list projects: %v", err))
			}

			return JSONResult(projects)
		},
	)
}

// registerSearchRepositories registers the search_repositories tool
func registerSearchRepositories(server *mcp.Server) {
	server.RegisterTool(
		mcp.Tool{
			Name:        "search_repositories",
			Description: "Search for GitLab repositories by name or description. If GITLAB_DEFAULT_NAMESPACE is configured, searches within that namespace by default.",
			InputSchema: mcp.JSONSchema{
				Type: "object",
				Properties: map[string]mcp.Property{
					"query": {
						Type:        "string",
						Description: "Search query string",
					},
					"namespace": {
						Type:        "string",
						Description: "Namespace/group ID or path to search within. Overrides GITLAB_DEFAULT_NAMESPACE if set.",
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
				Required: []string{"query"},
			},
		},
		func(args map[string]interface{}) (*mcp.CallToolResult, error) {
			c := GetContext()
			if c == nil {
				return ErrorResult("tool context not initialized")
			}
			c.Logger.ToolCall("search_repositories", args)

			query := GetString(args, "query", "")
			if query == "" {
				return ErrorResult("query is required")
			}

			// Determine namespace: explicit arg > config default > none
			namespace := GetString(args, "namespace", "")
			if namespace == "" && c.Config.DefaultNamespace != "" {
				namespace = c.Config.DefaultNamespace
			}

			params := url.Values{}
			params.Set("search", query)

			if page := GetInt(args, "page", 0); page > 0 {
				params.Set("page", fmt.Sprintf("%d", page))
			}
			if perPage := GetInt(args, "per_page", 0); perPage > 0 {
				params.Set("per_page", fmt.Sprintf("%d", perPage))
			}

			// Use group endpoint if namespace is set, otherwise search all projects
			var endpoint string
			if namespace != "" {
				endpoint = fmt.Sprintf("/groups/%s/projects?%s", url.PathEscape(namespace), params.Encode())
			} else {
				endpoint = fmt.Sprintf("/projects?%s", params.Encode())
			}

			var projects []gitlab.Project
			if err := c.Client.Get(endpoint, &projects); err != nil {
				return ErrorResult(fmt.Sprintf("Failed to search repositories: %v", err))
			}

			return JSONResult(projects)
		},
	)
}

// registerCreateRepository registers the create_repository tool
func registerCreateRepository(server *mcp.Server) {
	server.RegisterTool(
		mcp.Tool{
			Name:        "create_repository",
			Description: "Create a new GitLab repository/project. Uses GITLAB_DEFAULT_NAMESPACE for the target namespace if not specified.",
			InputSchema: mcp.JSONSchema{
				Type: "object",
				Properties: map[string]mcp.Property{
					"name": {
						Type:        "string",
						Description: "Name of the new project",
					},
					"namespace_id": {
						Type:        "string",
						Description: "Namespace/group ID or path to create the project in. Falls back to GITLAB_DEFAULT_NAMESPACE if not set.",
					},
					"description": {
						Type:        "string",
						Description: "Description of the project",
					},
					"visibility": {
						Type:        "string",
						Description: "Visibility level: private, internal, or public",
						Enum:        []string{"private", "internal", "public"},
					},
					"initialize_with_readme": {
						Type:        "boolean",
						Description: "Initialize repository with a README file",
					},
				},
				Required: []string{"name"},
			},
		},
		func(args map[string]interface{}) (*mcp.CallToolResult, error) {
			c := GetContext()
			if c == nil {
				return ErrorResult("tool context not initialized")
			}
			c.Logger.ToolCall("create_repository", args)

			name := GetString(args, "name", "")
			if name == "" {
				return ErrorResult("name is required")
			}

			// Determine namespace: explicit arg > config default > user's personal namespace
			namespace := GetString(args, "namespace_id", "")
			if namespace == "" && c.Config.DefaultNamespace != "" {
				namespace = c.Config.DefaultNamespace
			}

			body := map[string]interface{}{
				"name": name,
			}

			if namespace != "" {
				body["namespace_id"] = namespace
			}
			if description := GetString(args, "description", ""); description != "" {
				body["description"] = description
			}
			if visibility := GetString(args, "visibility", ""); visibility != "" {
				body["visibility"] = visibility
			}
			if initWithReadme := GetBool(args, "initialize_with_readme", false); initWithReadme {
				body["initialize_with_readme"] = true
			}

			var project gitlab.Project
			if err := c.Client.Post("/projects", body, &project); err != nil {
				return ErrorResult(fmt.Sprintf("Failed to create repository: %v", err))
			}

			return JSONResult(project)
		},
	)
}

// registerForkRepository registers the fork_repository tool
func registerForkRepository(server *mcp.Server) {
	server.RegisterTool(
		mcp.Tool{
			Name:        "fork_repository",
			Description: "Fork an existing GitLab repository. Uses GITLAB_DEFAULT_NAMESPACE for the target namespace if not specified.",
			InputSchema: mcp.JSONSchema{
				Type: "object",
				Properties: map[string]mcp.Property{
					"project_id": {
						Type:        "string",
						Description: "The ID or URL-encoded path of the project to fork",
					},
					"namespace": {
						Type:        "string",
						Description: "The namespace (user or group path) to fork the project into. Falls back to GITLAB_DEFAULT_NAMESPACE if not set.",
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
			c.Logger.ToolCall("fork_repository", args)

			projectID := GetString(args, "project_id", "")
			if projectID == "" {
				return ErrorResult("project_id is required")
			}

			// Determine namespace: explicit arg > config default > user's personal namespace
			namespace := GetString(args, "namespace", "")
			if namespace == "" && c.Config.DefaultNamespace != "" {
				namespace = c.Config.DefaultNamespace
			}

			endpoint := fmt.Sprintf("/projects/%s/fork", url.PathEscape(projectID))

			var body map[string]interface{}
			if namespace != "" {
				body = map[string]interface{}{
					"namespace": namespace,
				}
			}

			var project gitlab.Project
			if err := c.Client.Post(endpoint, body, &project); err != nil {
				return ErrorResult(fmt.Sprintf("Failed to fork repository: %v", err))
			}

			return JSONResult(project)
		},
	)
}

// registerListGroupProjects registers the list_group_projects tool
func registerListGroupProjects(server *mcp.Server) {
	server.RegisterTool(
		mcp.Tool{
			Name:        "list_group_projects",
			Description: "List all projects within a GitLab group. Uses GITLAB_DEFAULT_NAMESPACE if group_id is not provided.",
			InputSchema: mcp.JSONSchema{
				Type: "object",
				Properties: map[string]mcp.Property{
					"group_id": {
						Type:        "string",
						Description: "The ID or URL-encoded path of the group. Falls back to GITLAB_DEFAULT_NAMESPACE if not set.",
					},
					"page": {
						Type:        "integer",
						Description: "Page number for pagination (default: 1)",
					},
					"per_page": {
						Type:        "integer",
						Description: "Number of items per page (default: 20, max: 100)",
					},
					"archived": {
						Type:        "boolean",
						Description: "Filter by archived status",
					},
				},
			},
		},
		func(args map[string]interface{}) (*mcp.CallToolResult, error) {
			c := GetContext()
			if c == nil {
				return ErrorResult("tool context not initialized")
			}
			c.Logger.ToolCall("list_group_projects", args)

			// Determine group: explicit arg > config default
			groupID := GetString(args, "group_id", "")
			if groupID == "" && c.Config.DefaultNamespace != "" {
				groupID = c.Config.DefaultNamespace
			}
			if groupID == "" {
				return ErrorResult("group_id is required (or set GITLAB_DEFAULT_NAMESPACE)")
			}

			params := url.Values{}

			if page := GetInt(args, "page", 0); page > 0 {
				params.Set("page", fmt.Sprintf("%d", page))
			}
			if perPage := GetInt(args, "per_page", 0); perPage > 0 {
				params.Set("per_page", fmt.Sprintf("%d", perPage))
			}
			// Handle archived parameter - only add if explicitly set
			if _, exists := args["archived"]; exists {
				archived := GetBool(args, "archived", false)
				params.Set("archived", fmt.Sprintf("%t", archived))
			}

			endpoint := fmt.Sprintf("/groups/%s/projects", url.PathEscape(groupID))
			if len(params) > 0 {
				endpoint = fmt.Sprintf("%s?%s", endpoint, params.Encode())
			}

			var projects []gitlab.Project
			if err := c.Client.Get(endpoint, &projects); err != nil {
				return ErrorResult(fmt.Sprintf("Failed to list group projects: %v", err))
			}

			return JSONResult(projects)
		},
	)
}

// registerGetRepositoryTree registers the get_repository_tree tool
func registerGetRepositoryTree(server *mcp.Server) {
	server.RegisterTool(
		mcp.Tool{
			Name:        "get_repository_tree",
			Description: "Get the repository file tree for a GitLab project",
			InputSchema: mcp.JSONSchema{
				Type: "object",
				Properties: map[string]mcp.Property{
					"project_id": {
						Type:        "string",
						Description: "The ID or URL-encoded path of the project",
					},
					"path": {
						Type:        "string",
						Description: "Path inside repository to list (default: root)",
					},
					"ref": {
						Type:        "string",
						Description: "Branch name, tag, or commit SHA (default: default branch)",
					},
					"recursive": {
						Type:        "boolean",
						Description: "Get tree recursively",
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
			c.Logger.ToolCall("get_repository_tree", args)

			projectID := GetString(args, "project_id", "")
			if projectID == "" {
				return ErrorResult("project_id is required")
			}

			params := url.Values{}

			if path := GetString(args, "path", ""); path != "" {
				params.Set("path", path)
			}
			if ref := GetString(args, "ref", ""); ref != "" {
				params.Set("ref", ref)
			}
			if recursive := GetBool(args, "recursive", false); recursive {
				params.Set("recursive", "true")
			}

			endpoint := fmt.Sprintf("/projects/%s/repository/tree", url.PathEscape(projectID))
			if len(params) > 0 {
				endpoint = fmt.Sprintf("%s?%s", endpoint, params.Encode())
			}

			var treeNodes []gitlab.TreeNode
			if err := c.Client.Get(endpoint, &treeNodes); err != nil {
				return ErrorResult(fmt.Sprintf("Failed to get repository tree: %v", err))
			}

			return JSONResult(treeNodes)
		},
	)
}

// Member represents a project or group member with access level information
type Member struct {
	ID          int    `json:"id"`
	Username    string `json:"username"`
	Name        string `json:"name"`
	State       string `json:"state"`
	AvatarURL   string `json:"avatar_url"`
	WebURL      string `json:"web_url"`
	AccessLevel int    `json:"access_level"`
	ExpiresAt   string `json:"expires_at,omitempty"`
}

// registerListProjectMembers registers the list_project_members tool
func registerListProjectMembers(server *mcp.Server) {
	server.RegisterTool(
		mcp.Tool{
			Name:        "list_project_members",
			Description: "List all members of a GitLab project",
			InputSchema: mcp.JSONSchema{
				Type: "object",
				Properties: map[string]mcp.Property{
					"project_id": {
						Type:        "string",
						Description: "The ID or URL-encoded path of the project",
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
			c.Logger.ToolCall("list_project_members", args)

			projectID := GetString(args, "project_id", "")
			if projectID == "" {
				return ErrorResult("project_id is required")
			}

			params := url.Values{}

			if page := GetInt(args, "page", 0); page > 0 {
				params.Set("page", fmt.Sprintf("%d", page))
			}
			if perPage := GetInt(args, "per_page", 0); perPage > 0 {
				params.Set("per_page", fmt.Sprintf("%d", perPage))
			}

			endpoint := fmt.Sprintf("/projects/%s/members", url.PathEscape(projectID))
			if len(params) > 0 {
				endpoint = fmt.Sprintf("%s?%s", endpoint, params.Encode())
			}

			var members []Member
			if err := c.Client.Get(endpoint, &members); err != nil {
				return ErrorResult(fmt.Sprintf("Failed to list project members: %v", err))
			}

			return JSONResult(members)
		},
	)
}
