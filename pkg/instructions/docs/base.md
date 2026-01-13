# GitLab MCP Server

The GitLab MCP Server provides tools to interact with the GitLab platform for project management, code review, CI/CD, and collaboration.

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

## Tool Selection Guidance

### Discovery vs Targeted Lookup

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

## Project Identification

The `project_id` parameter accepts two formats:
- **Numeric ID**: `"42"` - stable, survives renames
- **Path format**: `"my-group/my-project"` - human-readable, must be URL-encoded for subgroups

Example paths:
- Simple: `my-org/backend-api`
- Nested: `my-org/team-a/microservices/auth-service`

If `GITLAB_DEFAULT_NAMESPACE` is configured, many tools will automatically scope to that namespace.

## Common Workflow Examples

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

## Pagination Best Practices

- Default `per_page` is 20 items (max 100)
- Start with `page=1` and increment for more results
- Use smaller page sizes (10-20) to avoid overwhelming context
- Check if more pages exist before fetching additional data

## Parameter Interdependencies

Some parameters work together or are mutually exclusive:

- **list_merge_requests**: Combine `state` with `scope` (e.g., state="opened", scope="assigned_to_me")
- **list_issues**: Use `labels` parameter for multi-label filtering (AND logic)
- **get_merge_request**: Use EITHER `merge_request_iid` OR `branch_name` to identify the MR

## Token Efficiency Tips

- Prefer specific `get_*` calls over broad `list_*` when you know the item ID
- Apply filters (state, scope, labels) to reduce result set size
- Use smaller `per_page` values to limit response size
- Use `format="text"` where available for compact output
- Cache project_id after first lookup to avoid repeated resolution

## Error Handling

| Error Type | Likely Cause | Solution |
|------------|--------------|----------|
| 404 Not Found | Invalid project_id or item ID | Verify ID exists and is accessible |
| 403 Forbidden | Insufficient permissions | Check token scopes |
| 401 Unauthorized | Invalid or expired token | Regenerate GitLab token |
| 400 Bad Request | Invalid parameter format | Check parameter types and values |

## Feature Flags

Some tools require feature flags to be enabled:

| Flag | Tools Enabled |
|------|---------------|
| `USE_PIPELINE=true` | Pipeline and job management tools |
| `USE_MILESTONE=true` | Milestone management tools |
| `USE_GITLAB_WIKI=true` | Wiki page management tools |
