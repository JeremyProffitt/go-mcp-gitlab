# GitLab MCP Server - LLM Usage Guide

> **Note**: These instructions are automatically delivered via the MCP protocol when the server initializes.
> This file is provided for users who prefer manual CLAUDE.md-style documentation in their projects.

## Overview

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

---

## Pipeline Tools

Pipeline tools provide access to CI/CD pipeline management and job log analysis.

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

| Parameter | Type | Description |
|-----------|------|-------------|
| `search` | string | Regex pattern to filter lines |
| `head` | int | Return first N lines |
| `tail` | int | Return last N lines |
| `context_lines` | int | Lines before/after matches |
| `invert_match` | bool | Return non-matching lines |
| `extract` | string | Predefined extractor (see below) |
| `format` | string | Output format: "json" or "text" |

### Common Workflows

**Check Latest Release:**
1. `get_latest_release_pipeline(project_id="...")` - get pipeline info
2. Check `pipeline.status` for "success" or "failed"
3. If failed, get job logs with `extract="errors"`

**Debug Failed Pipeline:**
1. `list_pipelines(project_id="...", status="failed")` - find failed pipelines
2. `list_pipeline_jobs(project_id="...", pipeline_id=<id>, scope=["failed"])` - find failed jobs
3. `get_pipeline_job_output(..., job_id=<failed_job>, extract="errors")` - get error details

---

## Terraform Log Extraction

Use the `extract` parameter with `get_pipeline_job_output` to parse common CI/CD patterns from job logs.

### Available Extractors

| Extract Value | Use Case |
|--------------|----------|
| `terraform_outputs` | Get Terraform output values (bucket_name, api_url, etc.) |
| `terraform_resources` | Get created/modified resources with IDs |
| `terraform_all` | Complete Terraform data: outputs, resources, summary, AWS assets |
| `aws_assets` | Extract ARNs, S3 URIs, and resource IDs (i-xxx, sg-xxx, etc.) |
| `errors` | Extract error/failure messages |
| `test_results` | Extract test pass/fail results |

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

Example text output:
```
Total lines: 1547 | Returned: 8

=== Terraform Outputs ===
bucket_arn: arn:aws:s3:::my-bucket-prod
api_endpoint: https://api.example.com

=== AWS Assets ===
ARNs:
  arn:aws:s3:::my-bucket-prod
  arn:aws:lambda:us-east-1:123456789:function:my-api
```

### Finding Specific Resources

Search logs for specific resource names:
```
get_pipeline_job_output(
  project_id="...",
  job_id=12345,
  search="my-lambda-function|my-bucket-name",
  context_lines=3
)
```

---

## Tool Reference

### Pipeline Tools

| Tool | Description |
|------|-------------|
| `list_pipelines` | List pipelines with filters (status, ref, scope) |
| `get_pipeline` | Get pipeline details |
| `list_pipeline_jobs` | List jobs in a pipeline |
| `get_pipeline_job` | Get job details |
| `get_pipeline_job_output` | Get job logs with search/extraction |
| `get_latest_release_pipeline` | Get pipeline for latest release tag |

### Project Tools

| Tool | Description |
|------|-------------|
| `get_project` | Get project details |
| `list_projects` | List accessible projects |
| `search_repositories` | Search for repositories |
| `get_repository_tree` | List files/directories in a repository |

### File Tools

| Tool | Description |
|------|-------------|
| `get_file_contents` | Get file content from repository |
| `create_or_update_file` | Create or update a file |
| `delete_file` | Delete a file |

### Issue Tools

| Tool | Description |
|------|-------------|
| `list_issues` | List project issues |
| `get_issue` | Get issue details |
| `create_issue` | Create a new issue |
| `update_issue` | Update an existing issue |

### Merge Request Tools

| Tool | Description |
|------|-------------|
| `list_merge_requests` | List merge requests |
| `get_merge_request` | Get MR details |
| `create_merge_request` | Create a new MR |
| `update_merge_request` | Update an existing MR |
