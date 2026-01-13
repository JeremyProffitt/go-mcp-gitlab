# GitLab MCP Server - LLM Usage Guide

> **Note**: These instructions are automatically delivered via the MCP protocol when the server initializes.
> This file is provided for users who prefer manual CLAUDE.md-style documentation in their projects.

## Overview

The GitLab MCP Server provides tools to interact with the GitLab platform for project management, code review, CI/CD, and collaboration.

---

## Quick Reference

### Parameter Formats

| Parameter | Format | Examples |
|-----------|--------|----------|
| `project_id` | Numeric or path | `"12345"`, `"my-group/my-project"` |
| `page` | Integer (1-indexed) | `1`, `2`, `3` |
| `per_page` | Integer (1-100) | `20` (default), `50`, `100` |
| `state` | String enum | Issues: `"opened"`, `"closed"`, `"all"` |
| `labels` | Array of strings | `["bug", "priority::high"]` |
| `ref` | Branch/tag/SHA | `"main"`, `"v1.0.0"`, `"abc123"` |

### State Values by Resource Type

| Resource | Valid States |
|----------|--------------|
| Issues | `"opened"`, `"closed"`, `"all"` |
| Merge Requests | `"opened"`, `"closed"`, `"merged"`, `"all"` |
| Pipelines | `"pending"`, `"running"`, `"success"`, `"failed"`, `"canceled"`, `"skipped"` |

### project_id Formats

| Format | Example | Description |
|--------|---------|-------------|
| Numeric ID | `"12345"` | Stable identifier, survives project renames |
| Simple path | `"my-group/my-project"` | Group and project name |
| Nested path | `"my-org/team-a/services/api"` | Full namespace path for nested groups |
| URL-encoded | `"my-group%2Fmy-project"` | Required when path contains special characters |

---

## Tool Selection Guide

| Goal | Tool Type | Example |
|------|-----------|---------|
| Browse all items | `list_*` | `list_projects`, `list_issues`, `list_merge_requests` |
| Find by keywords | `search_*` | `search_repositories` with query |
| Get specific item | `get_*` | `get_project`, `get_issue`, `get_merge_request` |
| Create new item | `create_*` | `create_issue`, `create_branch` |
| Modify item | `update_*` | `update_issue`, `update_merge_request` |
| Remove item | `delete_*` | `delete_issue`, `delete_label` |

### Tool Selection by Task

| Task | Recommended Tool | Why |
|------|------------------|-----|
| Find project by name | `search_repositories` | Keyword search in name/description |
| Get project details | `get_project` | Direct lookup by ID/path |
| Browse project files | `get_repository_tree` | Lists directory structure |
| Read file content | `get_file_contents` | Returns file content with metadata |
| Find open issues | `list_issues` with `state="opened"` | Filtered retrieval |
| My assigned work | `my_issues` | Pre-filtered to current user |
| Review MR changes | `get_merge_request_diffs` | Returns code diff |
| Check build status | `get_pipeline` or `list_pipelines` | Pipeline details |

### list_* vs search_* Tools

- **list_projects**: Enumerate all accessible projects, optionally filtered by namespace/visibility. Use when browsing or when you need comprehensive listing.
- **search_repositories**: Find projects matching keywords in name/description. Use when you know partial names or are looking for specific topics.
- **list_issues** / **list_merge_requests**: Retrieve items with state/scope filters. Use for workflow queries like "all open MRs" or "my assigned issues".

---

## Common Workflows

### 1. Code Review Workflow

```
1. list_merge_requests(project_id, state="opened") - Find open MRs
2. get_merge_request(project_id, merge_request_iid) - Get MR details
3. get_merge_request_diffs(project_id, merge_request_iid) - Review code changes
4. mr_discussions(project_id, merge_request_iid) - Read existing feedback
5. create_merge_request_thread(project_id, merge_request_iid, body, position) - Add review comment
```

### 2. Issue Triage Workflow

```
1. list_issues(project_id, state="opened", labels=["needs-triage"])
2. get_issue(project_id, issue_iid) - Get full details
3. list_issue_discussions(project_id, issue_iid) - Read comments
4. update_issue(project_id, issue_iid, labels=["bug", "priority::medium"]) - Categorize
```

### 3. Repository Exploration

```
1. get_project(project_id) - Get project metadata and default branch
2. get_repository_tree(project_id, path="", recursive=true) - List all files
3. get_file_contents(project_id, file_path, ref="main") - Read specific file
```

### 4. Feature Branch Development

```
1. create_branch(project_id, branch="feature-xyz", ref="main")
2. create_or_update_file(project_id, file_path, content, branch="feature-xyz", commit_message="...")
3. create_merge_request(project_id, source_branch="feature-xyz", target_branch="main", title="...")
```

### 5. Pipeline Debugging (requires USE_PIPELINE=true)

```
1. list_pipelines(project_id, status="failed") - Find failed pipelines
2. list_pipeline_jobs(project_id, pipeline_id, scope=["failed"]) - Find failed jobs
3. get_pipeline_job_output(project_id, job_id, extract="errors") - Get error details
```

### 6. Release Investigation

```
1. list_releases(project_id) - Get recent releases
2. get_commit(project_id, sha=release.tag) - Get release commit details
3. list_merge_requests(project_id, state="merged", target_branch="main") - Find merged MRs
```

---

## Tool Reference by Category

### Core Operations

| Category | Read Tools | Write Tools |
|----------|------------|-------------|
| **Projects** | `get_project`, `list_projects`, `search_repositories`, `list_group_projects`, `get_repository_tree`, `list_project_members` | `create_repository`, `fork_repository` |
| **Files** | `get_file_contents` | `create_or_update_file`, `push_files`, `upload_markdown` |
| **Issues** | `list_issues`, `my_issues`, `get_issue`, `list_issue_links`, `get_issue_link`, `list_issue_discussions` | `create_issue`, `update_issue`, `delete_issue`, `create_issue_link`, `delete_issue_link` |
| **Merge Requests** | `list_merge_requests`, `get_merge_request`, `get_merge_request_diffs`, `list_merge_request_diffs`, `get_branch_diffs`, `mr_discussions`, `list_draft_notes`, `get_draft_note` | `create_merge_request`, `update_merge_request`, `merge_merge_request`, `create_note`, `create_merge_request_thread`, `update_merge_request_note`, `create_merge_request_note`, `create_draft_note` |
| **Branches/Commits** | `list_commits`, `get_commit`, `get_commit_diff`, `list_releases`, `download_attachment` | `create_branch` |
| **Labels** | `list_labels`, `get_label` | `create_label`, `update_label`, `delete_label` |
| **Namespaces** | `list_namespaces`, `get_namespace`, `verify_namespace` | - |
| **Users** | `get_users` | - |

### Feature-Flagged Operations

#### Pipeline Tools (USE_PIPELINE=true)

| Tool | Description | Key Parameters |
|------|-------------|----------------|
| `list_pipelines` | List pipelines with filters | `status`, `ref`, `scope`, `page`, `per_page` |
| `get_pipeline` | Get pipeline details by ID | `project_id`, `pipeline_id` |
| `create_pipeline` | Trigger new pipeline | `project_id`, `ref`, `variables` |
| `retry_pipeline` | Retry failed jobs in pipeline | `project_id`, `pipeline_id` |
| `cancel_pipeline` | Cancel running pipeline | `project_id`, `pipeline_id` |
| `list_pipeline_jobs` | List jobs in a pipeline | `project_id`, `pipeline_id`, `scope` |
| `get_pipeline_job` | Get job details | `project_id`, `job_id` |
| `get_pipeline_job_output` | Get job logs with filtering | `project_id`, `job_id`, `search`, `extract` |
| `play_pipeline_job` | Start manual job | `project_id`, `job_id` |
| `retry_pipeline_job` | Retry failed job | `project_id`, `job_id` |
| `cancel_pipeline_job` | Cancel running job | `project_id`, `job_id` |

#### Milestone Tools (USE_MILESTONE=true)

| Tool | Description |
|------|-------------|
| `list_milestones` | List all milestones for a project |
| `get_milestone` | Get details of a specific milestone |
| `create_milestone` | Create a new milestone |
| `edit_milestone` | Update an existing milestone |
| `delete_milestone` | Delete a milestone |
| `get_milestone_issues` | Get issues associated with a milestone |
| `get_milestone_merge_requests` | Get MRs associated with a milestone |

#### Wiki Tools (USE_GITLAB_WIKI=true)

| Tool | Description |
|------|-------------|
| `list_wiki_pages` | List all wiki pages for a project |
| `get_wiki_page` | Get a specific wiki page |
| `create_wiki_page` | Create a new wiki page |
| `update_wiki_page` | Update an existing wiki page |
| `delete_wiki_page` | Delete a wiki page |
| `upload_wiki_attachment` | Upload an attachment to the wiki |

---

## Pipeline Tools (Detailed)

### Getting Job Logs

Use `get_pipeline_job_output` to retrieve and filter job logs:

```
# Basic usage - get full log
get_pipeline_job_output(project_id="my-group/my-project", job_id=12345)

# Search for specific patterns (regex supported)
get_pipeline_job_output(project_id="...", job_id=12345, search="error|failed")

# Get last N lines (useful for final output/summary)
get_pipeline_job_output(project_id="...", job_id=12345, tail=50)

# Search with context lines (like grep -C)
get_pipeline_job_output(project_id="...", job_id=12345, search="terraform apply", context_lines=5)
```

### Key Parameters for get_pipeline_job_output

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `project_id` | string | required | Project ID or path |
| `job_id` | integer | required | Job ID to get logs from |
| `search` | string | - | Regex pattern to filter lines |
| `head` | integer | - | Return first N lines only |
| `tail` | integer | - | Return last N lines only |
| `context_lines` | integer | 0 | Lines before/after matches |
| `invert_match` | boolean | false | Return non-matching lines |
| `extract` | string | - | Predefined extractor (see below) |
| `format` | string | "json" | Output format: "json" or "text" |

---

## Log Extraction

Use the `extract` parameter with `get_pipeline_job_output` to parse common CI/CD patterns from job logs.

### Available Extractors

| Extract Value | Use Case | Output Contains |
|--------------|----------|-----------------|
| `terraform_outputs` | Get Terraform output values | Variable names and values |
| `terraform_resources` | Get created/modified resources | Resource types, names, and IDs |
| `terraform_all` | Complete Terraform data | Outputs + resources + summary + AWS assets |
| `aws_assets` | Extract AWS resource identifiers | ARNs, S3 URIs, resource IDs |
| `errors` | Extract error/failure messages | Error lines with context |
| `test_results` | Extract test pass/fail results | Test names and outcomes |

### Example: Get Deployed AWS Resources

```
1. get_latest_release_pipeline(project_id="my-group/my-project")
   -> Returns pipeline info and job list

2. Find the deploy job ID from the jobs list

3. get_pipeline_job_output(project_id="...", job_id=<deploy_job_id>, extract="terraform_all")
   -> Returns structured data with all Terraform outputs and AWS resources
```

### Output Formats

Use `format` parameter to control output:
- `json` (default): Structured JSON data
- `text`: Compact LLM-friendly format with fewer tokens

---

## Token Efficiency Tips

1. **Use targeted tools**: Prefer `get_issue(id)` over `list_issues()` when you know the ID
2. **Apply filters**: Use `state`, `labels`, `scope` parameters to reduce results
3. **Limit page size**: Use `per_page=10` for initial exploration
4. **Use text format**: Set `format="text"` where available for compact output
5. **Cache project_id**: Store the project ID after first lookup to avoid repeated resolution

---

## Error Handling

| Error Type | Likely Cause | Solution |
|------------|--------------|----------|
| 404 Not Found | Invalid project_id or item ID | Verify ID exists and is accessible |
| 403 Forbidden | Insufficient permissions | Check token scopes |
| 401 Unauthorized | Invalid or expired token | Regenerate GitLab token |
| 400 Bad Request | Invalid parameter format | Check parameter types and values |

---

## Quick Tool Finder

| If you want to... | Use this tool |
|-------------------|---------------|
| Find a project by name | `search_repositories` |
| Get project details | `get_project` |
| List files in a directory | `get_repository_tree` |
| Read a file | `get_file_contents` |
| Create/update a file | `create_or_update_file` |
| List open issues | `list_issues` with `state="opened"` |
| Find my assigned issues | `my_issues` |
| Create an issue | `create_issue` |
| List open MRs | `list_merge_requests` with `state="opened"` |
| See MR code changes | `get_merge_request_diffs` |
| Create a branch | `create_branch` |
| Create a merge request | `create_merge_request` |
| Check pipeline status | `get_pipeline` |
| Get job logs | `get_pipeline_job_output` |
| Find errors in job logs | `get_pipeline_job_output` with `extract="errors"` |
