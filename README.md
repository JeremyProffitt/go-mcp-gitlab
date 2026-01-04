# go-mcp-gitlab

A comprehensive Model Context Protocol (MCP) server for GitLab integration. Written in Go, it provides full GitLab API capabilities with multi-source credential resolution, project restrictions, feature flags, and read-only mode support.

## Features

- **Cross-Platform**: Supports Windows, macOS, and Linux
- **Multi-Source Credential Resolution**: Automatically discovers GitLab tokens from environment variables, glab CLI, git credentials, and netrc files
- **Project Restrictions**: Limit access to specific projects via allowlist
- **Feature Flags**: Enable/disable optional tool sets (pipelines, milestones, wikis)
- **Read-Only Mode**: Restrict all operations to read-only for safety
- **Comprehensive Logging**: Detailed logging with configurable levels
- **MCP Protocol Compliant**: Full JSON-RPC 2.0 and MCP protocol support
- **70+ GitLab Operations**: Projects, issues, merge requests, files, branches, pipelines, and more

## Installation

### From Binary Releases

Download the latest binary for your platform from the [Releases](https://github.com/go-mcp-gitlab/go-mcp-gitlab/releases) page:

| Platform | Architecture | File |
|----------|--------------|------|
| macOS | Universal (Intel + Apple Silicon) | go-mcp-gitlab-darwin-universal |
| Linux | x64 | go-mcp-gitlab-linux-amd64 |
| Linux | ARM64 | go-mcp-gitlab-linux-arm64 |
| Windows | x64 | go-mcp-gitlab-windows-amd64.exe |

### From Source

```bash
git clone https://github.com/go-mcp-gitlab/go-mcp-gitlab.git
cd go-mcp-gitlab
go build -o go-mcp-gitlab .
```

## Usage

### Command Line Options

```bash
go-mcp-gitlab [options]
```

| Option | Environment Variable | Default | Description |
|--------|---------------------|---------|-------------|
| `-log-dir` | `MCP_LOG_DIR` | `~/go-mcp-gitlab/logs` | Directory for log files |
| `-log-level` | `MCP_LOG_LEVEL` | `info` | Log level: off\|error\|warn\|info\|access\|debug |
| `-version` | - | - | Show version information |
| `-help` | - | - | Show help message |

### Environment Variables

| Variable | Description |
|----------|-------------|
| `GITLAB_API_URL` | GitLab API URL (default: `https://gitlab.com/api/v4`) |
| `GITLAB_PERSONAL_ACCESS_TOKEN` | GitLab personal access token (highest priority) |
| `GITLAB_TOKEN` | Alternative token variable |
| `GITLAB_ACCESS_TOKEN` | Alternative token variable |
| `GL_TOKEN` | Alternative token variable |
| `GITLAB_PROJECT_ID` | Default project ID for operations |
| `GITLAB_ALLOWED_PROJECT_IDS` | Comma-separated list of allowed project IDs |
| `USE_PIPELINE` | Enable pipeline tools (default: false) |
| `USE_MILESTONE` | Enable milestone tools (default: false) |
| `USE_GITLAB_WIKI` | Enable wiki tools (default: false) |
| `GITLAB_READ_ONLY_MODE` | Enable read-only mode (default: false) |

### GitLab Token Resolution

Tokens are resolved in the following priority order:

1. **Environment Variables** (highest priority)
   - `GITLAB_PERSONAL_ACCESS_TOKEN`
   - `GITLAB_TOKEN`
   - `GITLAB_ACCESS_TOKEN`
   - `GL_TOKEN`

2. **GitLab CLI (glab) Config**
   - `~/.config/glab-cli/config.yml`
   - `$XDG_CONFIG_HOME/glab-cli/config.yml`
   - `%APPDATA%/glab-cli/config.yml` (Windows)

3. **Git Credential Helper**
   - Non-interactive helpers only (store, cache)
   - 2-second timeout to prevent blocking

4. **Netrc File**
   - `~/.netrc` or `~/_netrc` (Windows)
   - `$NETRC` environment variable

### Configuration Priority

Configuration values are resolved in the following priority order:
1. Command-line flags (highest priority)
2. Environment variables
3. Default values (lowest priority)

## MCP Tools

### Project Tools

| Tool | Description |
|------|-------------|
| `get_project` | Get details of a specific GitLab project by ID or path |
| `list_projects` | List all projects visible to the authenticated user |
| `search_repositories` | Search for GitLab repositories by name or description |
| `create_repository` | Create a new GitLab repository/project |
| `fork_repository` | Fork an existing GitLab repository |
| `list_group_projects` | List all projects within a GitLab group |
| `get_repository_tree` | Get the repository file tree for a GitLab project |
| `list_project_members` | List all members of a GitLab project |

### File Tools

| Tool | Description |
|------|-------------|
| `get_file_contents` | Get the contents of a file from a GitLab repository |
| `create_or_update_file` | Create a new file or update an existing file in a repository |
| `push_files` | Push multiple files to a repository in a single commit |
| `upload_markdown` | Upload a file and get a markdown link for use in issues/MRs |

### Issue Tools

| Tool | Description |
|------|-------------|
| `list_issues` | List issues in a GitLab project with optional filtering |
| `my_issues` | List issues assigned to the authenticated user across all projects |
| `get_issue` | Get details of a specific issue |
| `create_issue` | Create a new issue in a GitLab project |
| `update_issue` | Update an existing issue |
| `delete_issue` | Delete an issue from a GitLab project |
| `list_issue_links` | List all links for a specific issue |
| `get_issue_link` | Get details of a specific issue link |
| `create_issue_link` | Create a link between two issues |
| `delete_issue_link` | Delete an issue link |
| `list_issue_discussions` | List all discussions on an issue |

### Merge Request Tools

| Tool | Description |
|------|-------------|
| `list_merge_requests` | List merge requests for a project |
| `get_merge_request` | Get details of a specific merge request |
| `create_merge_request` | Create a new merge request |
| `update_merge_request` | Update an existing merge request |
| `merge_merge_request` | Merge a merge request |
| `get_merge_request_diffs` | Get the diffs for a merge request |
| `list_merge_request_diffs` | List diffs with pagination support |
| `get_branch_diffs` | Compare two branches, tags, or commits |
| `create_note` | Create a note (comment) on an issue or merge request |
| `create_merge_request_thread` | Create a new discussion thread on a merge request |
| `mr_discussions` | List all discussions on a merge request |
| `update_merge_request_note` | Update an existing note in a merge request discussion |
| `create_merge_request_note` | Add a new note to an existing discussion thread |
| `list_draft_notes` | List all draft notes for a merge request |
| `get_draft_note` | Get a specific draft note |
| `create_draft_note` | Create a draft note on a merge request |

### Branch & Commit Tools

| Tool | Description |
|------|-------------|
| `create_branch` | Create a new branch in a GitLab project repository |
| `list_commits` | List repository commits in a GitLab project |
| `get_commit` | Get a specific commit from a repository |
| `get_commit_diff` | Get the diff of a commit |
| `list_releases` | List releases of a GitLab project |
| `download_attachment` | Download an uploaded file/attachment from a project |

### Label Tools

| Tool | Description |
|------|-------------|
| `list_labels` | List all labels for a project |
| `get_label` | Get details of a specific label |
| `create_label` | Create a new label |
| `update_label` | Update an existing label |
| `delete_label` | Delete a label |

### Namespace Tools

| Tool | Description |
|------|-------------|
| `list_namespaces` | List all namespaces |
| `get_namespace` | Get details of a specific namespace |
| `verify_namespace` | Verify if a namespace path exists |

### User Tools

| Tool | Description |
|------|-------------|
| `get_users` | Get user information |

### Pipeline Tools (Feature-Flagged)

*Enabled when `USE_PIPELINE=true`*

| Tool | Description |
|------|-------------|
| `list_pipelines` | List pipelines for a project |
| `get_pipeline` | Get details of a specific pipeline |
| `create_pipeline` | Create a new pipeline |
| `retry_pipeline` | Retry all failed jobs in a pipeline |
| `cancel_pipeline` | Cancel a running pipeline |
| `list_pipeline_jobs` | List all jobs for a specific pipeline |
| `list_pipeline_trigger_jobs` | List all trigger jobs (bridges) for a pipeline |
| `get_pipeline_job` | Get details of a specific job |
| `get_pipeline_job_output` | Get the log output of a specific job |
| `play_pipeline_job` | Trigger a manual job to start |
| `retry_pipeline_job` | Retry a failed or canceled job |
| `cancel_pipeline_job` | Cancel a running job |

### Milestone Tools (Feature-Flagged)

*Enabled when `USE_MILESTONE=true`*

| Tool | Description |
|------|-------------|
| `list_milestones` | List all milestones for a project |
| `get_milestone` | Get details of a specific milestone |
| `create_milestone` | Create a new milestone |
| `edit_milestone` | Update an existing milestone |
| `delete_milestone` | Delete a milestone |
| `get_milestone_issues` | Get issues associated with a milestone |
| `get_milestone_merge_requests` | Get merge requests associated with a milestone |
| `promote_milestone` | Promote a project milestone to a group milestone |
| `get_milestone_burndown_events` | Get burndown events for a milestone |

### Wiki Tools (Feature-Flagged)

*Enabled when `USE_GITLAB_WIKI=true`*

| Tool | Description |
|------|-------------|
| `list_wiki_pages` | List all wiki pages for a project |
| `get_wiki_page` | Get a specific wiki page |
| `create_wiki_page` | Create a new wiki page |
| `update_wiki_page` | Update an existing wiki page |
| `delete_wiki_page` | Delete a wiki page |
| `upload_wiki_attachment` | Upload an attachment to the wiki |

## Integration

### Claude Desktop

Add to your Claude Desktop configuration file:

**macOS**: `~/Library/Application Support/Claude/claude_desktop_config.json`
**Windows**: `%APPDATA%/Claude/claude_desktop_config.json`

```json
{
  "mcpServers": {
    "gitlab": {
      "command": "/path/to/go-mcp-gitlab",
      "args": ["-log-level", "info"],
      "env": {
        "GITLAB_PERSONAL_ACCESS_TOKEN": "glpat-xxxxxxxxxxxx"
      }
    }
  }
}
```

### Claude Code

Create a `.mcp.json` file in your project root:

```json
{
  "mcpServers": {
    "gitlab": {
      "command": "/path/to/go-mcp-gitlab",
      "args": ["-log-level", "info"],
      "env": {
        "GITLAB_PERSONAL_ACCESS_TOKEN": "glpat-xxxxxxxxxxxx"
      }
    }
  }
}
```

Or create `.claude/mcp.json` for workspace-specific configuration:

```json
{
  "mcpServers": {
    "gitlab": {
      "command": "${workspaceFolder}/go-mcp-gitlab",
      "args": [
        "-log-dir", "${workspaceFolder}/logs",
        "-log-level", "info"
      ],
      "env": {
        "USE_PIPELINE": "true",
        "USE_MILESTONE": "true"
      }
    }
  }
}
```

### Continue.dev

Create a `.continue/config.json` file:

```json
{
  "experimental": {
    "modelContextProtocolServers": [
      {
        "name": "go-mcp-gitlab",
        "transport": {
          "type": "stdio",
          "command": "/path/to/go-mcp-gitlab",
          "args": ["-log-level", "info"]
        }
      }
    ]
  }
}
```

Or use `.continue/config.yaml`:

```yaml
experimental:
  modelContextProtocolServers:
    - name: go-mcp-gitlab
      transport:
        type: stdio
        command: /path/to/go-mcp-gitlab
        args:
          - -log-level
          - info
```

With environment variables and feature flags:

**JSON** (`.continue/config.json`):
```json
{
  "experimental": {
    "modelContextProtocolServers": [
      {
        "name": "go-mcp-gitlab",
        "transport": {
          "type": "stdio",
          "command": "/path/to/go-mcp-gitlab",
          "args": ["-log-level", "debug"]
        },
        "env": {
          "GITLAB_PERSONAL_ACCESS_TOKEN": "glpat-xxxxxxxxxxxx",
          "GITLAB_API_URL": "https://gitlab.mycompany.com/api/v4",
          "USE_PIPELINE": "true",
          "USE_MILESTONE": "true",
          "USE_GITLAB_WIKI": "true"
        }
      }
    ]
  }
}
```

**YAML** (`.continue/config.yaml`):
```yaml
experimental:
  modelContextProtocolServers:
    - name: go-mcp-gitlab
      transport:
        type: stdio
        command: /path/to/go-mcp-gitlab
        args:
          - -log-level
          - debug
      env:
        GITLAB_PERSONAL_ACCESS_TOKEN: glpat-xxxxxxxxxxxx
        GITLAB_API_URL: https://gitlab.mycompany.com/api/v4
        USE_PIPELINE: "true"
        USE_MILESTONE: "true"
        USE_GITLAB_WIKI: "true"
```

## Security

### Read-Only Mode

Enable read-only mode to prevent any write operations:

```bash
export GITLAB_READ_ONLY_MODE=true
go-mcp-gitlab
```

In read-only mode, all create, update, and delete operations will be rejected.

### Project Restrictions

Limit the server to only access specific projects:

```bash
# Set a default project
export GITLAB_PROJECT_ID="my-group/my-project"

# Or allow multiple specific projects
export GITLAB_ALLOWED_PROJECT_IDS="project-1,group/project-2,12345"
```

### Token Security

- Tokens are never logged in full; only masked versions appear in logs
- Git credential helper calls have a 2-second timeout to prevent blocking
- Interactive credential helpers (manager, osxkeychain, wincred) are skipped

### Self-Hosted GitLab

For self-hosted GitLab instances:

```bash
export GITLAB_API_URL="https://gitlab.mycompany.com/api/v4"
export GITLAB_PERSONAL_ACCESS_TOKEN="glpat-xxxxxxxxxxxx"
go-mcp-gitlab
```

## Global Environment File

All go-mcp servers support loading environment variables from `~/.mcp_env`. This provides a central location to configure credentials and settings, especially useful on macOS where GUI applications don't inherit shell environment variables from `.zshrc` or `.bashrc`.

### File Format

Create `~/.mcp_env` with KEY=VALUE pairs:

```bash
# ~/.mcp_env - MCP Server Environment Variables

# GitLab Configuration
GITLAB_PERSONAL_ACCESS_TOKEN=glpat-xxxxxxxxxxxx
GITLAB_API_URL=https://gitlab.com/api/v4
USE_PIPELINE=true
USE_MILESTONE=true

# Logging
MCP_LOG_DIR=~/mcp-logs
MCP_LOG_LEVEL=info
```

### Features

- Lines starting with `#` are treated as comments
- Empty lines are ignored
- Values can be quoted with single or double quotes
- **Existing environment variables are NOT overwritten** (env vars take precedence)
- Paths with `~` are automatically expanded to your home directory

### Path Expansion

All path-related settings support `~` expansion:

```bash
MCP_LOG_DIR=~/logs/gitlab
```

This works in the `~/.mcp_env` file, environment variables, and command-line flags.

## Logging

Logs are written to date-stamped files in the log directory:

```
~/go-mcp-gitlab/logs/go-mcp-gitlab-2025-01-15.log
```

When `MCP_LOG_DIR` is set or `-log-dir` flag is used, logs are automatically placed in a subfolder named after the binary. This allows multiple MCP servers to share the same log directory:

```
MCP_LOG_DIR=/var/log/mcp
  └── go-mcp-gitlab/
      └── go-mcp-gitlab-2025-01-15.log
```

### Log Levels

| Level | Description |
|-------|-------------|
| `off` | No logging |
| `error` | Errors only |
| `warn` | Warnings and errors |
| `info` | General information (default) |
| `access` | API call details |
| `debug` | Detailed debugging information |

### Log Format

```
[2025-01-15T10:30:45.123Z] [INFO] TOOL_CALL tool="list_projects" args=[page, per_page]
[2025-01-15T10:30:45.150Z] [ACCESS] API_CALL method="GET" endpoint="/projects" status=200 duration=27ms
```

**Security Note**: Sensitive data (tokens, file contents) is never logged.

## Development

### Prerequisites

- Go 1.21 or later

### Building

```bash
# Build for current platform
go build -o go-mcp-gitlab .

# Build for all platforms
make build
```

### Testing

```bash
# Run unit tests
go test -v ./pkg/...

# Run all tests with coverage
go test -v -race -coverprofile=coverage.out ./...
```

### Project Structure

```
go-mcp-gitlab/
├── main.go                    # Entry point, server initialization
├── go.mod                     # Go module definition
├── go.sum                     # Dependency checksums
├── pkg/
│   ├── config/
│   │   ├── config.go          # Configuration management
│   │   └── credentials.go     # Multi-source credential resolution
│   ├── gitlab/
│   │   ├── client.go          # GitLab API client
│   │   ├── types.go           # GitLab data types
│   │   └── errors.go          # Error handling
│   ├── logging/
│   │   └── logging.go         # Logging implementation
│   ├── mcp/
│   │   ├── server.go          # MCP server implementation
│   │   └── types.go           # MCP protocol types
│   └── tools/
│       ├── registry.go        # Tool registration
│       ├── helpers.go         # Utility functions
│       ├── projects.go        # Project tools
│       ├── files.go           # File tools
│       ├── issues.go          # Issue tools
│       ├── merge_requests.go  # Merge request tools
│       ├── branches.go        # Branch/commit tools
│       ├── labels.go          # Label tools
│       ├── namespaces.go      # Namespace tools
│       ├── users.go           # User tools
│       ├── notes.go           # Note/comment tools
│       ├── pipelines.go       # Pipeline tools (feature-flagged)
│       ├── milestones.go      # Milestone tools (feature-flagged)
│       ├── wikis.go           # Wiki tools (feature-flagged)
│       └── releases.go        # Release tools
├── .github/workflows/
│   ├── ci.yml                 # CI workflow
│   └── release.yml            # Release workflow
├── .mcp.json                  # MCP configuration example
└── README.md                  # This file
```

## Examples

### Using Environment Variable

```bash
export GITLAB_PERSONAL_ACCESS_TOKEN=glpat-xxxxxxxxxxxx
go-mcp-gitlab
```

### Using glab CLI (if already configured)

```bash
glab auth login  # Configure token once
go-mcp-gitlab    # Token auto-detected
```

### Using Git Credential Helper

```bash
git config --global credential.helper store
git clone https://gitlab.com/user/repo.git  # Saves credentials
go-mcp-gitlab  # Token auto-detected
```

### Enabling All Features

```bash
export GITLAB_PERSONAL_ACCESS_TOKEN=glpat-xxxxxxxxxxxx
export USE_PIPELINE=true
export USE_MILESTONE=true
export USE_GITLAB_WIKI=true
go-mcp-gitlab -log-level debug
```

### Read-Only Mode with Project Restrictions

```bash
export GITLAB_PERSONAL_ACCESS_TOKEN=glpat-xxxxxxxxxxxx
export GITLAB_READ_ONLY_MODE=true
export GITLAB_ALLOWED_PROJECT_IDS="my-group/project-a,my-group/project-b"
go-mcp-gitlab
```

## License

MIT License - see LICENSE file for details.

## Acknowledgments

- Inspired by [gitlab-mcp-server](https://github.com/modelcontextprotocol/servers/tree/main/src/gitlab)
- Built following patterns from [go-mcp-commander](https://github.com/user/go-mcp-commander)
- Implements the [Model Context Protocol](https://modelcontextprotocol.io/)
