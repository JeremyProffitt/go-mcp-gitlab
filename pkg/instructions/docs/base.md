# GitLab MCP Server

The GitLab MCP Server provides tools to interact with the GitLab platform.

## Tool Selection Guidance

1. Use `list_*` tools for broad retrieval with basic filtering (all issues, all MRs, all branches)
2. Use `search_*` tools for targeted queries with specific criteria or keywords
3. Use `get_*` tools when you have a specific ID or path

## Context Management

1. Use pagination with reasonable page sizes (10-20 items) to avoid overwhelming responses
2. When working with a specific project, cache the `project_id` to avoid repeated lookups
3. Use `format="text"` parameter where available to reduce token usage

## Common Patterns

### Working with Projects
- Use `project_id` as either numeric ID or `"namespace/project"` path format
- The `GITLAB_DEFAULT_NAMESPACE` may be configured to scope project operations

### Working with Files
- `get_file_contents` retrieves file content with optional ref (branch/tag/commit)
- Use `get_repository_tree` to explore directory structure before fetching files

### Working with Issues and MRs
- Always check for existing issues before creating duplicates
- Use labels and milestones to organize work
- Link related issues and MRs using references (`#123`, `!456`)
