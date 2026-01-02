# Go GitLab MCP Server - Implementation Plan

## Executive Summary

Port the TypeScript-based GitLab MCP server (https://github.com/zereight/gitlab-mcp) to Go, following the patterns and logging standards from `go-mcp-file-context-server`. This plan incorporates **heavy subagent usage** for both planning and execution phases.

---

## Source Analysis (Gathered via Subagents)

### Original TypeScript Repository Analysis
- **95+ tools** covering complete GitLab API
- **4 Authentication Methods**: PAT, OAuth2, Remote Authorization, Cookie-based
- **3 Transport Modes**: stdio, SSE, Streamable HTTP
- **Feature Flags**: USE_PIPELINE (12 tools), USE_MILESTONE (9 tools), USE_GITLAB_WIKI (6 tools)
- **Connection Pooling**: Per-URL agent caching with proxy support
- **Rate Limiting**: Per-session throttling (60 req/min default)

### Reference Go Repository Patterns
- **6 Log Levels**: OFF, ERROR, WARN, INFO, ACCESS, DEBUG
- **Configuration Tracking**: Source tracking (flag/env/default)
- **MCP Protocol**: JSON-RPC 2.0 over stdio with 10MB buffer
- **Tool Registration**: Handler function pattern with schema
- **Security**: Path validation, access logging (never content)

---

## Project Structure

```
go-mcp-gitlab/
├── main.go                          # Entry point, CLI, tool registration
├── go.mod                           # Module definition
├── go.sum                           # Dependencies
├── Makefile                         # Build targets
├── build.ps1 / build.sh             # Platform build scripts
├── .gitignore
├── .env.example                     # Example configuration
├── PLAN.md                          # This file
│
├── pkg/
│   ├── mcp/
│   │   ├── server.go                # MCP server (JSON-RPC stdio)
│   │   └── types.go                 # Protocol types
│   │
│   ├── logging/
│   │   └── logging.go               # Structured logging system
│   │
│   ├── gitlab/
│   │   ├── client.go                # GitLab API HTTP client
│   │   ├── auth.go                  # Authentication (PAT, OAuth)
│   │   ├── types.go                 # API response structures
│   │   └── errors.go                # Custom error types
│   │
│   ├── config/
│   │   └── config.go                # Configuration management
│   │
│   └── tools/
│       ├── registry.go              # Tool registration utilities
│       ├── helpers.go               # Parameter extraction helpers
│       ├── projects.go              # Project tools (8 tools)
│       ├── files.go                 # File/repository tools (4 tools)
│       ├── issues.go                # Issue tools (11 tools)
│       ├── merge_requests.go        # MR tools (16 tools)
│       ├── notes.go                 # Notes/comments tools (6 tools)
│       ├── branches.go              # Branch/commit tools (6 tools)
│       ├── labels.go                # Label tools (5 tools)
│       ├── namespaces.go            # Namespace tools (3 tools)
│       ├── users.go                 # User tools (2 tools)
│       ├── releases.go              # Release tools (7 tools)
│       ├── events.go                # Event tools (2 tools)
│       ├── pipelines.go             # Pipeline tools (12 tools) [FLAG]
│       ├── milestones.go            # Milestone tools (9 tools) [FLAG]
│       └── wikis.go                 # Wiki tools (6 tools) [FLAG]
│
└── test/
    ├── integration/                 # Integration tests
    └── mocks/                       # Mock GitLab responses
```

---

## Subagent Execution Strategy

### Subagent Types and Their Roles

| Subagent Type | Purpose | When to Use |
|---------------|---------|-------------|
| `Explore` | Codebase analysis, pattern discovery | Finding existing patterns, understanding code |
| `Plan` | Architecture design, implementation planning | Designing complex features, trade-off analysis |
| `general-purpose` | Multi-step implementation tasks | Implementing tool groups, complex features |

### Parallel Subagent Patterns

**Pattern 1: Parallel Feature Implementation**
```
Launch simultaneously:
- Subagent A: Implement project tools
- Subagent B: Implement issue tools
- Subagent C: Implement MR tools
```

**Pattern 2: Research + Implementation**
```
Phase 1: Launch Explore subagent to analyze patterns
Phase 2: Use findings to launch implementation subagents
```

**Pattern 3: Test + Implement**
```
Launch simultaneously:
- Subagent A: Implement feature
- Subagent B: Write tests for feature
```

---

## Implementation Phases with Subagent Assignments

### Phase 1: Core Infrastructure

**Objective**: Set up project foundation with MCP protocol and logging

| Task | Subagent | Description |
|------|----------|-------------|
| 1.1 | `general-purpose` | Initialize Go module, copy pkg/mcp from reference |
| 1.2 | `general-purpose` | Copy and adapt pkg/logging from reference |
| 1.3 | `general-purpose` | Create main.go with CLI parsing and config |
| 1.4 | `general-purpose` | Create Makefile and build scripts |

**Subagent Prompt Template (Phase 1.1)**:
```
Create the core MCP infrastructure for go-mcp-gitlab:
1. Initialize Go module at C:\dev\go-mcp-gitlab
2. Copy pkg/mcp/server.go and pkg/mcp/types.go from C:\dev\go-mcp-file-context-server
3. Adapt imports and package references
4. Create go.mod with module github.com/user/go-mcp-gitlab
5. Verify the code compiles

Reference the patterns in C:\dev\go-mcp-file-context-server for:
- JSON-RPC message handling
- Tool registration pattern
- Response formatting
```

---

### Phase 2: GitLab Client

**Objective**: Create GitLab API client with authentication

| Task | Subagent | Description |
|------|----------|-------------|
| 2.1 | `Explore` | Analyze GitLab API patterns from source repo |
| 2.2 | `general-purpose` | Create pkg/gitlab/client.go with HTTP handling |
| 2.3 | `general-purpose` | Create pkg/gitlab/auth.go with PAT support |
| 2.4 | `general-purpose` | Create pkg/gitlab/types.go with response structs |
| 2.5 | `general-purpose` | Create pkg/gitlab/errors.go with error types |
| 2.6 | `general-purpose` | Create pkg/config/config.go for env/flag parsing |

**Parallel Execution**: Tasks 2.2-2.5 can run in parallel after 2.1

**GitLab Client Pattern**:
```go
type Client struct {
    baseURL    string
    token      string
    httpClient *http.Client
    logger     *logging.Logger
}

func (c *Client) Get(endpoint string, result interface{}) error
func (c *Client) Post(endpoint string, body, result interface{}) error
func (c *Client) Put(endpoint string, body, result interface{}) error
func (c *Client) Delete(endpoint string) error
```

---

### Phase 3: Core Tools (Priority 1 - 50 Tools)

**Objective**: Implement essential GitLab operations

**Parallel Subagent Execution** (launch all simultaneously):

| Subagent | Tool File | Tools Count | Tools |
|----------|-----------|-------------|-------|
| A | `projects.go` | 8 | get_project, list_projects, search_repositories, create_repository, fork_repository, list_group_projects, get_repository_tree, list_project_members |
| B | `files.go` | 4 | get_file_contents, create_or_update_file, push_files, upload_markdown |
| C | `issues.go` | 11 | list_issues, my_issues, get_issue, create_issue, update_issue, delete_issue, list_issue_links, get_issue_link, create_issue_link, delete_issue_link, list_issue_discussions |
| D | `merge_requests.go` | 16 | list_merge_requests, get_merge_request, create_merge_request, update_merge_request, merge_merge_request, get_merge_request_diffs, list_merge_request_diffs, get_branch_diffs, create_note, create_merge_request_thread, mr_discussions, update_merge_request_note, create_merge_request_note, list_draft_notes, get_draft_note, create_draft_note |
| E | `branches.go` | 6 | create_branch, list_commits, get_commit, get_commit_diff, list_releases, download_attachment |

**Subagent Prompt Template (Phase 3 - Issues)**:
```
Implement GitLab issue tools for go-mcp-gitlab at C:\dev\go-mcp-gitlab\pkg\tools\issues.go

Reference patterns from:
- C:\dev\go-mcp-file-context-server\main.go (tool registration, handlers)
- C:\dev\go-mcp-file-context-server\pkg\logging\logging.go (logging patterns)

Implement these 11 tools with full GitLab API integration:
1. list_issues - GET /projects/:id/issues
2. my_issues - GET /issues (assigned to current user)
3. get_issue - GET /projects/:id/issues/:iid
4. create_issue - POST /projects/:id/issues
5. update_issue - PUT /projects/:id/issues/:iid
6. delete_issue - DELETE /projects/:id/issues/:iid
7. list_issue_links - GET /projects/:id/issues/:iid/links
8. get_issue_link - GET /projects/:id/issues/:iid/links/:link_id
9. create_issue_link - POST /projects/:id/issues/:iid/links
10. delete_issue_link - DELETE /projects/:id/issues/:iid/links/:link_id
11. list_issue_discussions - GET /projects/:id/issues/:iid/discussions

Each tool must:
- Log with logger.ToolCall() at start
- Extract parameters with helper functions
- Call GitLab API via gitlab.Client
- Return JSON-formatted results
- Handle errors with logger.Error() and errorResult()

Use the parameter schemas from the original TypeScript (documented in analysis).
```

---

### Phase 4: Secondary Tools (Priority 2 - 18 Tools)

| Subagent | Tool File | Tools Count | Tools |
|----------|-----------|-------------|-------|
| A | `notes.go` | 6 | update_draft_note, delete_draft_note, publish_draft_note, bulk_publish_draft_notes, update_issue_note, create_issue_note |
| B | `labels.go` | 5 | list_labels, get_label, create_label, update_label, delete_label |
| C | `namespaces.go` | 3 | list_namespaces, get_namespace, verify_namespace |
| D | `users.go` | 2 | get_users, (list_project_members already in projects) |
| E | `events.go` | 2 | list_events, get_project_events |

---

### Phase 5: Optional Tools (Feature Flags - 27 Tools)

**Parallel Subagent Execution**:

| Subagent | Tool File | Flag | Tools Count |
|----------|-----------|------|-------------|
| A | `pipelines.go` | USE_PIPELINE | 12 |
| B | `milestones.go` | USE_MILESTONE | 9 |
| C | `wikis.go` | USE_GITLAB_WIKI | 6 |

**Pipeline Tools (12)**:
```
list_pipelines, get_pipeline, create_pipeline, retry_pipeline, cancel_pipeline,
list_pipeline_jobs, list_pipeline_trigger_jobs, get_pipeline_job,
get_pipeline_job_output, play_pipeline_job, retry_pipeline_job, cancel_pipeline_job
```

**Milestone Tools (9)**:
```
list_milestones, get_milestone, create_milestone, edit_milestone, delete_milestone,
get_milestone_issue, get_milestone_merge_requests, promote_milestone,
get_milestone_burndown_events
```

**Wiki Tools (6)**:
```
list_wiki_pages, get_wiki_page, create_wiki_page, update_wiki_page,
delete_wiki_page, upload_wiki_attachment
```

---

### Phase 6: Release Tools (7 Tools)

| Subagent | Tool File | Tools |
|----------|-----------|-------|
| A | `releases.go` | list_releases, get_release, create_release, update_release, delete_release, create_release_evidence, download_release_asset |

---

### Phase 7: Testing & Documentation

| Subagent | Task | Description |
|----------|------|-------------|
| A | Unit Tests | Write tests for each tool category |
| B | Integration Tests | Test against real GitLab instance |
| C | Documentation | Create README.md with usage examples |
| D | Build Verification | Test cross-platform builds |

---

## Configuration Reference

### Environment Variables

```bash
# Authentication (Required - one of these)
GITLAB_PERSONAL_ACCESS_TOKEN=glpat-xxxxxxxxxxxx
# OR OAuth2
GITLAB_USE_OAUTH=true
GITLAB_OAUTH_CLIENT_ID=your_client_id
GITLAB_OAUTH_CLIENT_SECRET=your_secret
GITLAB_OAUTH_REDIRECT_URI=http://127.0.0.1:8888/callback

# GitLab API
GITLAB_API_URL=https://gitlab.com/api/v4

# Project Restriction (Optional)
GITLAB_PROJECT_ID=12345
GITLAB_ALLOWED_PROJECT_IDS=123,456,789

# Feature Flags (Optional, default: false)
USE_GITLAB_WIKI=true
USE_MILESTONE=true
USE_PIPELINE=true
GITLAB_READ_ONLY_MODE=false

# Logging
MCP_LOG_DIR=/path/to/logs
MCP_LOG_LEVEL=info  # off, error, warn, info, access, debug
```

### CLI Flags

```bash
go-mcp-gitlab [OPTIONS]

OPTIONS:
  -log-dir <path>     Log directory (default: ~/go-mcp-gitlab/logs)
  -log-level <level>  Log level: off, error, warn, info, access, debug
  -version            Show version
  -help               Show help
```

---

## Logging Standards (from Reference)

### Log Levels
| Level | Value | Description |
|-------|-------|-------------|
| OFF | 0 | Disable all logging |
| ERROR | 1 | Errors only |
| WARN | 2 | Warnings + errors |
| INFO | 3 | General information (default) |
| ACCESS | 4 | API operations (request/response metadata) |
| DEBUG | 5 | Detailed debugging |

### Logging Patterns

```go
// Tool invocation (INFO level)
logger.ToolCall("get_issue", args)  // Logs keys only, never values

// API calls (ACCESS level)
logger.Access("API_CALL method=%s endpoint=%s", method, endpoint)

// Errors (ERROR level)
logger.Error("get_issue: API error: %v", err)

// Startup (INFO level with formatted block)
logger.LogStartup(startupInfo)

// Shutdown (INFO level)
logger.LogShutdown("normal exit")
```

### Startup Log Format
```
========================================
SERVER STARTUP
========================================
Application: go-mcp-gitlab
Version: 1.0.0
Go Version: go1.21
OS: windows
Architecture: amd64
Process ID: 12345
Start Time: 2024-01-02T15:04:05Z
----------------------------------------
CONFIGURATION (value [source])
----------------------------------------
Log Directory: C:\Users\...\logs [default]
Log Level: info [environment]
GitLab API URL: https://gitlab.com/api/v4 [flag]
Auth Method: PAT [environment]
Features: pipeline, milestone [environment]
----------------------------------------
========================================
```

---

## Tool Implementation Pattern

### Standard Handler Template

```go
func handleGetIssue(args map[string]interface{}) (*mcp.CallToolResult, error) {
    // 1. Log tool call (keys only, never values)
    logger.ToolCall("get_issue", args)

    // 2. Extract required parameters
    projectID := getString(args, "project_id", "")
    issueIID := getInt(args, "issue_iid", 0)

    // 3. Validate required parameters
    if projectID == "" {
        logger.Error("get_issue: missing project_id")
        return errorResult("project_id is required")
    }
    if issueIID == 0 {
        logger.Error("get_issue: missing issue_iid")
        return errorResult("issue_iid is required")
    }

    // 4. Apply project restrictions if configured
    if err := validateProjectAccess(projectID); err != nil {
        logger.Error("get_issue: access denied: %v", err)
        return errorResult(err.Error())
    }

    // 5. Call GitLab API
    endpoint := fmt.Sprintf("/projects/%s/issues/%d", url.PathEscape(projectID), issueIID)
    var issue GitLabIssue
    if err := gitlabClient.Get(endpoint, &issue); err != nil {
        logger.Error("get_issue: API error: %v", err)
        return errorResult(fmt.Sprintf("GitLab API error: %v", err))
    }

    // 6. Log success and return result
    logger.Debug("get_issue: retrieved issue #%d from project %s", issueIID, projectID)
    result, _ := json.MarshalIndent(issue, "", "  ")
    return textResult(string(result))
}
```

### Tool Registration Template

```go
server.RegisterTool(mcp.Tool{
    Name:        "get_issue",
    Description: "Get details of a specific issue in a GitLab project",
    InputSchema: mcp.JSONSchema{
        Type: "object",
        Properties: map[string]mcp.Property{
            "project_id": {
                Type:        "string",
                Description: "Project ID or URL-encoded path",
            },
            "issue_iid": {
                Type:        "number",
                Description: "Issue IID (internal ID within project)",
            },
        },
        Required: []string{"project_id", "issue_iid"},
    },
}, handleGetIssue)
```

---

## Helper Functions

```go
// Parameter extraction (from reference)
func getString(args map[string]interface{}, key, defaultVal string) string
func getInt(args map[string]interface{}, key string, defaultVal int) int
func getInt64(args map[string]interface{}, key string, defaultVal int64) int64
func getBool(args map[string]interface{}, key string, defaultVal bool) bool
func getStringArray(args map[string]interface{}, key string) []string

// Response builders
func textResult(text string) (*mcp.CallToolResult, error)
func errorResult(message string) (*mcp.CallToolResult, error)

// Project access validation
func validateProjectAccess(projectID string) error
```

---

## Todo List with Subagent Assignments

### Phase 1: Core Infrastructure
- [ ] **[Subagent: general-purpose]** Initialize Go module and project structure
- [ ] **[Subagent: general-purpose]** Copy and adapt pkg/mcp from reference
- [ ] **[Subagent: general-purpose]** Copy and adapt pkg/logging from reference
- [ ] **[Subagent: general-purpose]** Create main.go with CLI/env configuration
- [ ] **[Subagent: general-purpose]** Create Makefile and build scripts
- [ ] **[Manual]** Verify project compiles and runs

### Phase 2: GitLab Client
- [ ] **[Subagent: Explore]** Analyze GitLab API patterns from TypeScript source
- [ ] **[Subagent: general-purpose]** Create pkg/gitlab/client.go (HTTP wrapper)
- [ ] **[Subagent: general-purpose]** Create pkg/gitlab/auth.go (PAT authentication)
- [ ] **[Subagent: general-purpose]** Create pkg/gitlab/types.go (response structs)
- [ ] **[Subagent: general-purpose]** Create pkg/gitlab/errors.go (custom errors)
- [ ] **[Subagent: general-purpose]** Create pkg/config/config.go (configuration)
- [ ] **[Subagent: general-purpose]** Create pkg/tools/helpers.go (parameter helpers)
- [ ] **[Subagent: general-purpose]** Create pkg/tools/registry.go (registration utilities)

### Phase 3: Core Tools (Parallel Execution)
- [ ] **[Subagent A: general-purpose]** Implement projects.go (8 tools)
- [ ] **[Subagent B: general-purpose]** Implement files.go (4 tools)
- [ ] **[Subagent C: general-purpose]** Implement issues.go (11 tools)
- [ ] **[Subagent D: general-purpose]** Implement merge_requests.go (16 tools)
- [ ] **[Subagent E: general-purpose]** Implement branches.go (6 tools)

### Phase 4: Secondary Tools (Parallel Execution)
- [ ] **[Subagent A: general-purpose]** Implement notes.go (6 tools)
- [ ] **[Subagent B: general-purpose]** Implement labels.go (5 tools)
- [ ] **[Subagent C: general-purpose]** Implement namespaces.go (3 tools)
- [ ] **[Subagent D: general-purpose]** Implement users.go (2 tools)
- [ ] **[Subagent E: general-purpose]** Implement events.go (2 tools)

### Phase 5: Optional Tools (Parallel Execution)
- [ ] **[Subagent A: general-purpose]** Implement pipelines.go (12 tools, USE_PIPELINE)
- [ ] **[Subagent B: general-purpose]** Implement milestones.go (9 tools, USE_MILESTONE)
- [ ] **[Subagent C: general-purpose]** Implement wikis.go (6 tools, USE_GITLAB_WIKI)

### Phase 6: Release Tools
- [ ] **[Subagent: general-purpose]** Implement releases.go (7 tools)

### Phase 7: Testing & Documentation (Parallel Execution)
- [ ] **[Subagent A: general-purpose]** Write unit tests for tool handlers
- [ ] **[Subagent B: general-purpose]** Write integration tests
- [ ] **[Subagent C: general-purpose]** Create README.md documentation
- [ ] **[Subagent D: general-purpose]** Test cross-platform builds (Linux, macOS, Windows)
- [ ] **[Manual]** Final testing with Claude Desktop/Cursor

### Phase 8: Advanced Features (Optional)
- [ ] **[Subagent: Plan]** Design OAuth2 implementation
- [ ] **[Subagent: general-purpose]** Implement OAuth2 authentication
- [ ] **[Subagent: general-purpose]** Add connection pooling
- [ ] **[Subagent: general-purpose]** Add rate limiting awareness

---

## Complete Tool Reference (95 Tools)

### Always Enabled (68 Tools)

**Project Tools (8)**
| Tool | Method | Endpoint |
|------|--------|----------|
| get_project | GET | /projects/:id |
| list_projects | GET | /projects |
| search_repositories | GET | /projects?search= |
| create_repository | POST | /projects |
| fork_repository | POST | /projects/:id/fork |
| list_group_projects | GET | /groups/:id/projects |
| get_repository_tree | GET | /projects/:id/repository/tree |
| list_project_members | GET | /projects/:id/members |

**File Tools (4)**
| Tool | Method | Endpoint |
|------|--------|----------|
| get_file_contents | GET | /projects/:id/repository/files/:path |
| create_or_update_file | POST/PUT | /projects/:id/repository/files/:path |
| push_files | POST | /projects/:id/repository/commits |
| upload_markdown | POST | /projects/:id/uploads |

**Issue Tools (11)**
| Tool | Method | Endpoint |
|------|--------|----------|
| list_issues | GET | /projects/:id/issues |
| my_issues | GET | /issues |
| get_issue | GET | /projects/:id/issues/:iid |
| create_issue | POST | /projects/:id/issues |
| update_issue | PUT | /projects/:id/issues/:iid |
| delete_issue | DELETE | /projects/:id/issues/:iid |
| list_issue_links | GET | /projects/:id/issues/:iid/links |
| get_issue_link | GET | /projects/:id/issues/:iid/links/:link_id |
| create_issue_link | POST | /projects/:id/issues/:iid/links |
| delete_issue_link | DELETE | /projects/:id/issues/:iid/links/:link_id |
| list_issue_discussions | GET | /projects/:id/issues/:iid/discussions |

**Merge Request Tools (16)**
| Tool | Method | Endpoint |
|------|--------|----------|
| list_merge_requests | GET | /projects/:id/merge_requests |
| get_merge_request | GET | /projects/:id/merge_requests/:iid |
| create_merge_request | POST | /projects/:id/merge_requests |
| update_merge_request | PUT | /projects/:id/merge_requests/:iid |
| merge_merge_request | PUT | /projects/:id/merge_requests/:iid/merge |
| get_merge_request_diffs | GET | /projects/:id/merge_requests/:iid/diffs |
| list_merge_request_diffs | GET | /projects/:id/merge_requests/:iid/diffs |
| get_branch_diffs | GET | /projects/:id/repository/compare |
| create_note | POST | /projects/:id/issues/:iid/notes |
| create_merge_request_thread | POST | /projects/:id/merge_requests/:iid/discussions |
| mr_discussions | GET | /projects/:id/merge_requests/:iid/discussions |
| update_merge_request_note | PUT | /projects/:id/merge_requests/:iid/discussions/:id/notes/:note_id |
| create_merge_request_note | POST | /projects/:id/merge_requests/:iid/discussions/:id/notes |
| list_draft_notes | GET | /projects/:id/merge_requests/:iid/draft_notes |
| get_draft_note | GET | /projects/:id/merge_requests/:iid/draft_notes/:id |
| create_draft_note | POST | /projects/:id/merge_requests/:iid/draft_notes |

**Notes Tools (6)**
| Tool | Method | Endpoint |
|------|--------|----------|
| update_draft_note | PUT | /projects/:id/merge_requests/:iid/draft_notes/:id |
| delete_draft_note | DELETE | /projects/:id/merge_requests/:iid/draft_notes/:id |
| publish_draft_note | PUT | /projects/:id/merge_requests/:iid/draft_notes/:id/publish |
| bulk_publish_draft_notes | POST | /projects/:id/merge_requests/:iid/draft_notes/bulk_publish |
| update_issue_note | PUT | /projects/:id/issues/:iid/discussions/:id/notes/:note_id |
| create_issue_note | POST | /projects/:id/issues/:iid/discussions/:id/notes |

**Branch/Commit Tools (6)**
| Tool | Method | Endpoint |
|------|--------|----------|
| create_branch | POST | /projects/:id/repository/branches |
| list_commits | GET | /projects/:id/repository/commits |
| get_commit | GET | /projects/:id/repository/commits/:sha |
| get_commit_diff | GET | /projects/:id/repository/commits/:sha/diff |
| list_releases | GET | /projects/:id/releases |
| download_attachment | GET | /projects/:id/uploads/:secret/:filename |

**Label Tools (5)**
| Tool | Method | Endpoint |
|------|--------|----------|
| list_labels | GET | /projects/:id/labels |
| get_label | GET | /projects/:id/labels/:label_id |
| create_label | POST | /projects/:id/labels |
| update_label | PUT | /projects/:id/labels/:label_id |
| delete_label | DELETE | /projects/:id/labels/:label_id |

**Namespace Tools (3)**
| Tool | Method | Endpoint |
|------|--------|----------|
| list_namespaces | GET | /namespaces |
| get_namespace | GET | /namespaces/:id |
| verify_namespace | GET | /namespaces/:path/exists |

**User Tools (1)**
| Tool | Method | Endpoint |
|------|--------|----------|
| get_users | GET | /users?username= |

**Release Tools (7)**
| Tool | Method | Endpoint |
|------|--------|----------|
| get_release | GET | /projects/:id/releases/:tag |
| create_release | POST | /projects/:id/releases |
| update_release | PUT | /projects/:id/releases/:tag |
| delete_release | DELETE | /projects/:id/releases/:tag |
| create_release_evidence | POST | /projects/:id/releases/:tag/evidence |
| download_release_asset | GET | (asset URL) |

**Event Tools (2)**
| Tool | Method | Endpoint |
|------|--------|----------|
| list_events | GET | /events |
| get_project_events | GET | /projects/:id/events |

**Iteration Tools (1)**
| Tool | Method | Endpoint |
|------|--------|----------|
| list_group_iterations | GET | /groups/:id/iterations |

### Feature-Flagged Tools (27 Tools)

**Pipeline Tools - USE_PIPELINE (12)**
| Tool | Method | Endpoint |
|------|--------|----------|
| list_pipelines | GET | /projects/:id/pipelines |
| get_pipeline | GET | /projects/:id/pipelines/:pipeline_id |
| create_pipeline | POST | /projects/:id/pipeline |
| retry_pipeline | POST | /projects/:id/pipelines/:pipeline_id/retry |
| cancel_pipeline | POST | /projects/:id/pipelines/:pipeline_id/cancel |
| list_pipeline_jobs | GET | /projects/:id/pipelines/:pipeline_id/jobs |
| list_pipeline_trigger_jobs | GET | /projects/:id/pipelines/:pipeline_id/bridges |
| get_pipeline_job | GET | /projects/:id/jobs/:job_id |
| get_pipeline_job_output | GET | /projects/:id/jobs/:job_id/trace |
| play_pipeline_job | POST | /projects/:id/jobs/:job_id/play |
| retry_pipeline_job | POST | /projects/:id/jobs/:job_id/retry |
| cancel_pipeline_job | POST | /projects/:id/jobs/:job_id/cancel |

**Milestone Tools - USE_MILESTONE (9)**
| Tool | Method | Endpoint |
|------|--------|----------|
| list_milestones | GET | /projects/:id/milestones |
| get_milestone | GET | /projects/:id/milestones/:milestone_id |
| create_milestone | POST | /projects/:id/milestones |
| edit_milestone | PUT | /projects/:id/milestones/:milestone_id |
| delete_milestone | DELETE | /projects/:id/milestones/:milestone_id |
| get_milestone_issue | GET | /projects/:id/milestones/:milestone_id/issues |
| get_milestone_merge_requests | GET | /projects/:id/milestones/:milestone_id/merge_requests |
| promote_milestone | POST | /projects/:id/milestones/:milestone_id/promote |
| get_milestone_burndown_events | GET | /projects/:id/milestones/:milestone_id/burndown_events |

**Wiki Tools - USE_GITLAB_WIKI (6)**
| Tool | Method | Endpoint |
|------|--------|----------|
| list_wiki_pages | GET | /projects/:id/wikis |
| get_wiki_page | GET | /projects/:id/wikis/:slug |
| create_wiki_page | POST | /projects/:id/wikis |
| update_wiki_page | PUT | /projects/:id/wikis/:slug |
| delete_wiki_page | DELETE | /projects/:id/wikis/:slug |
| upload_wiki_attachment | POST | /projects/:id/wikis/attachments |

---

## Success Criteria

1. All 68 core tools working correctly
2. All 27 feature-flagged tools working when enabled
3. Logging follows reference patterns exactly
4. Configuration via env vars and CLI flags
5. Clean error handling with appropriate logging
6. Compatible with Claude Desktop, Cursor, and other MCP clients
7. Cross-platform builds (Linux, macOS, Windows)
8. Unit tests for critical paths
9. README with installation and usage instructions

---

## Execution Commands

### Starting Phase 1
```
Use Task tool with subagent_type="general-purpose" to initialize project infrastructure
```

### Parallel Phase 3 Execution
```
Use Task tool 5 times in parallel with subagent_type="general-purpose":
- Subagent A: "Implement pkg/tools/projects.go with 8 project tools..."
- Subagent B: "Implement pkg/tools/files.go with 4 file tools..."
- Subagent C: "Implement pkg/tools/issues.go with 11 issue tools..."
- Subagent D: "Implement pkg/tools/merge_requests.go with 16 MR tools..."
- Subagent E: "Implement pkg/tools/branches.go with 6 branch tools..."
```

### Exploration Before Implementation
```
Use Task tool with subagent_type="Explore" to analyze patterns before complex implementations
```
