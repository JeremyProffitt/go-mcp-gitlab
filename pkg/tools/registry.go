package tools

import (
	"sync"

	"github.com/go-mcp-gitlab/go-mcp-gitlab/pkg/config"
	"github.com/go-mcp-gitlab/go-mcp-gitlab/pkg/gitlab"
	"github.com/go-mcp-gitlab/go-mcp-gitlab/pkg/logging"
	"github.com/go-mcp-gitlab/go-mcp-gitlab/pkg/mcp"
)

// Context holds shared dependencies for tool handlers.
// It provides access to the GitLab client, logger, and configuration
// that all tool handlers need.
type Context struct {
	Client *gitlab.Client
	Logger *logging.Logger
	Config *config.Config
}

var (
	// ctx is the global context for tool handlers
	ctx  *Context
	ctxMu sync.RWMutex
)

// SetContext initializes the global tool context with the provided dependencies.
// This should be called once during server initialization before any tools are invoked.
func SetContext(client *gitlab.Client, logger *logging.Logger, cfg *config.Config) {
	ctxMu.Lock()
	defer ctxMu.Unlock()
	ctx = &Context{
		Client: client,
		Logger: logger,
		Config: cfg,
	}
}

// GetContext returns the global tool context.
// Returns nil if SetContext has not been called.
func GetContext() *Context {
	ctxMu.RLock()
	defer ctxMu.RUnlock()
	return ctx
}

// RegisterProjectTools registers all project-related tools with the MCP server.
// Includes: get_project, list_projects, search_repositories, create_repository,
// fork_repository, list_group_projects, get_repository_tree, list_project_members
func RegisterProjectTools(server *mcp.Server) {
	registerGetProject(server)
	registerListProjects(server)
	registerSearchRepositories(server)
	registerCreateRepository(server)
	registerForkRepository(server)
	registerListGroupProjects(server)
	registerGetRepositoryTree(server)
	registerListProjectMembers(server)
}

// Note: RegisterFileTools is implemented in files.go with signature:
// RegisterFileTools(server *mcp.Server, ctx *ToolContext)

// Note: RegisterIssueTools is implemented in issues.go with signature:
// RegisterIssueTools(server *mcp.Server, ctx *ToolContext)

// RegisterMergeRequestTools registers all merge request-related tools with the MCP server.
// Includes: list_merge_requests, get_merge_request, create_merge_request, update_merge_request, etc.
func RegisterMergeRequestTools(server *mcp.Server) {
	initMergeRequestTools(server)
}

// Note: RegisterBranchTools is implemented in branches.go with signature:
// RegisterBranchTools(server *mcp.Server, ctx *ToolContext)

// RegisterLabelTools registers all label-related tools with the MCP server.
// Includes: list_labels, get_label, create_label, update_label, delete_label
func RegisterLabelTools(server *mcp.Server) {
	RegisterLabelToolsImpl(server)
}

// RegisterNamespaceTools registers all namespace-related tools with the MCP server.
// Includes: list_namespaces, get_namespace, verify_namespace
func RegisterNamespaceTools(server *mcp.Server) {
	initNamespaceTools(server)
}

// RegisterUserTools registers all user-related tools with the MCP server.
// Includes: get_users
func RegisterUserTools(server *mcp.Server) {
	initUserTools(server)
}

// RegisterEventTools registers all event-related tools with the MCP server.
// Includes: list_events, get_project_events
func RegisterEventTools(server *mcp.Server) {
	initEventTools(server)
}

// RegisterReleaseTools registers all release-related tools with the MCP server.
// Includes: get_release, create_release, update_release, delete_release,
// create_release_evidence, download_release_asset
// Note: list_releases is registered via RegisterBranchTools
func RegisterReleaseTools(server *mcp.Server) {
	initReleaseTools(server)
}

// RegisterPipelineTools registers all pipeline-related tools with the MCP server.
// This is a feature-flagged tool set, only registered when USE_PIPELINE is enabled.
// Includes: list_pipelines, get_pipeline, create_pipeline, retry_pipeline, cancel_pipeline,
// list_pipeline_jobs, list_pipeline_trigger_jobs, get_pipeline_job, get_pipeline_job_output,
// play_pipeline_job, retry_pipeline_job, cancel_pipeline_job
func RegisterPipelineTools(server *mcp.Server) {
	// Check if pipeline feature is enabled
	c := GetContext()
	if c == nil || c.Config == nil || !c.Config.UsePipeline {
		return
	}
	initPipelineTools(server)
}

// RegisterMilestoneTools registers all milestone-related tools with the MCP server.
// This is a feature-flagged tool set, only registered when USE_MILESTONE is enabled.
// Includes: list_milestones, get_milestone, create_milestone, edit_milestone, delete_milestone,
// get_milestone_issues, get_milestone_merge_requests, promote_milestone, get_milestone_burndown_events
func RegisterMilestoneTools(server *mcp.Server) {
	// Check if milestone feature is enabled
	c := GetContext()
	if c == nil || c.Config == nil || !c.Config.UseMilestone {
		return
	}
	initMilestoneTools(server)
}

// RegisterWikiTools registers all wiki-related tools with the MCP server.
// This is a feature-flagged tool set, only registered when USE_GITLAB_WIKI is enabled.
// Includes: list_wiki_pages, get_wiki_page, create_wiki_page, update_wiki_page, delete_wiki_page,
// upload_wiki_attachment
func RegisterWikiTools(server *mcp.Server) {
	// Check if wiki feature is enabled
	c := GetContext()
	if c == nil || c.Config == nil || !c.Config.UseWiki {
		return
	}
	initWikiTools(server)
}

// RegisterAllTools is a convenience function that registers all available tools.
// It respects feature flags for optional tool sets.
func RegisterAllTools(server *mcp.Server) {
	// Core tools (always registered)
	RegisterProjectTools(server)
	RegisterFileTools(server)
	RegisterIssueTools(server)
	RegisterMergeRequestTools(server)
	RegisterBranchTools(server)
	RegisterLabelTools(server)
	RegisterNamespaceTools(server)
	RegisterUserTools(server)
	RegisterEventTools(server)
	RegisterReleaseTools(server)

	// Feature-flagged tools (conditionally registered)
	RegisterPipelineTools(server)
	RegisterMilestoneTools(server)
	RegisterWikiTools(server)
}
