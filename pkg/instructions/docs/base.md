# GitLab MCP Server

The GitLab MCP Server provides tools to interact with the GitLab platform for project management, code review, CI/CD, and collaboration.

## Tool Selection Guidance

### Discovery vs Targeted Lookup

| Goal | Tool Type | Example |
|------|-----------|---------|
| Browse all items | `list_*` | `list_projects`, `list_issues`, `list_merge_requests` |
| Find by keywords | `search_*` | `search_repositories` with query |
| Get specific item | `get_*` | `get_project`, `get_issue`, `get_merge_request` |

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

### 1. Reviewing a Merge Request

```
1. get_merge_request(project_id, mr_iid) - Get MR overview
2. get_merge_request_diff(project_id, mr_iid) - See code changes
3. list_merge_request_discussions(project_id, mr_iid) - Read review comments
4. create_merge_request_note(project_id, mr_iid, body) - Add feedback
```

### 2. Working with Issues

```
1. list_issues(project_id, state="opened") - Get open issues
2. get_issue(project_id, issue_iid) - Get full details
3. get_issue_notes(project_id, issue_iid) - Read discussion
4. update_issue(project_id, issue_iid, labels=["bug", "priority::high"]) - Categorize
```

### 3. Exploring a Repository

```
1. get_project(project_id) - Get project metadata and default branch
2. get_repository_tree(project_id, recursive=true) - List all files
3. get_file_contents(project_id, file_path, ref="main") - Read specific file
```

### 4. Creating a Branch and Merge Request

```
1. create_branch(project_id, branch="feature-x", ref="main") - Create feature branch
2. create_or_update_file(project_id, file_path, content, branch="feature-x") - Make changes
3. create_merge_request(project_id, source="feature-x", target="main", title="Add feature X") - Open MR
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

## Reducing Token Usage

- Prefer specific `get_*` calls over broad `list_*` when you know the item ID
- Apply filters (state, scope, labels) to reduce result set size
- Use smaller `per_page` values to limit response size
- Start with summary tools before diving into details
