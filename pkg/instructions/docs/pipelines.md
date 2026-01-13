## Pipeline Tools

Pipeline tools provide access to CI/CD pipeline management and job log analysis. Requires `USE_PIPELINE=true`.

### Pipeline Tool Reference

| Tool | Description | Key Parameters |
|------|-------------|----------------|
| `list_pipelines` | List pipelines with filters | `status`, `ref`, `scope`, `page`, `per_page` |
| `get_pipeline` | Get pipeline details by ID | `project_id`, `pipeline_id` |
| `create_pipeline` | Trigger new pipeline | `project_id`, `ref`, `variables` |
| `retry_pipeline` | Retry failed jobs in pipeline | `project_id`, `pipeline_id` |
| `cancel_pipeline` | Cancel running pipeline | `project_id`, `pipeline_id` |
| `list_pipeline_jobs` | List jobs in a pipeline | `project_id`, `pipeline_id`, `scope` |
| `list_pipeline_trigger_jobs` | List trigger/bridge jobs | `project_id`, `pipeline_id` |
| `get_pipeline_job` | Get job details | `project_id`, `job_id` |
| `get_pipeline_job_output` | Get job logs with filtering | `project_id`, `job_id`, `search`, `extract` |
| `play_pipeline_job` | Start manual job | `project_id`, `job_id` |
| `retry_pipeline_job` | Retry failed job | `project_id`, `job_id` |
| `cancel_pipeline_job` | Cancel running job | `project_id`, `job_id` |

### Pipeline States

| State | Description |
|-------|-------------|
| `pending` | Waiting to be picked up by runner |
| `running` | Currently executing |
| `success` | Completed successfully |
| `failed` | Completed with errors |
| `canceled` | Manually canceled |
| `skipped` | Skipped due to rules/conditions |
| `manual` | Waiting for manual trigger |

### Job Scopes for Filtering

Use with `list_pipeline_jobs`:
- `pending`, `running`, `success`, `failed`, `canceled`, `skipped`, `manual`

### Getting Job Logs

Use `get_pipeline_job_output` to retrieve and filter job logs:

```
# Basic usage - get full log
get_pipeline_job_output(project_id="my-group/my-project", job_id=12345)

# Search for specific patterns (regex supported)
get_pipeline_job_output(project_id="...", job_id=12345, search="error|failed")

# Get last N lines (useful for final output/summary)
get_pipeline_job_output(project_id="...", job_id=12345, tail=50)

# Get first N lines (useful for setup/initialization)
get_pipeline_job_output(project_id="...", job_id=12345, head=100)

# Search with context lines (like grep -C)
get_pipeline_job_output(project_id="...", job_id=12345, search="terraform apply", context_lines=5)

# Invert match - find lines NOT matching pattern
get_pipeline_job_output(project_id="...", job_id=12345, search="DEBUG", invert_match=true)
```

### Key Parameters for get_pipeline_job_output

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `project_id` | string | required | Project ID or path |
| `job_id` | integer | required | Job ID to get logs from |
| `search` | string | - | Regex pattern to filter lines |
| `head` | integer | - | Return first N lines only |
| `tail` | integer | - | Return last N lines only |
| `context_lines` | integer | 0 | Lines before/after matches (like grep -C) |
| `invert_match` | boolean | false | Return non-matching lines |
| `extract` | string | - | Predefined extractor (see Terraform section) |
| `format` | string | "json" | Output format: "json" or "text" |

### Common Workflows

#### Debug Failed Pipeline

```
1. list_pipelines(project_id, status="failed") - Find failed pipelines
2. list_pipeline_jobs(project_id, pipeline_id, scope=["failed"]) - Find failed jobs
3. get_pipeline_job_output(project_id, job_id, extract="errors") - Get error details
```

#### Check Latest Release Status

```
1. get_latest_release_pipeline(project_id) - Get pipeline info for latest release
2. Check pipeline.status for "success" or "failed"
3. If failed: list_pipeline_jobs(project_id, pipeline_id, scope=["failed"])
4. get_pipeline_job_output(project_id, job_id, tail=100) - See final output
```

#### Monitor Running Pipeline

```
1. get_pipeline(project_id, pipeline_id) - Get current status
2. list_pipeline_jobs(project_id, pipeline_id, scope=["running"]) - See active jobs
3. get_pipeline_job_output(project_id, job_id, tail=50) - See recent output
```

#### Retry Failed Build

```
1. list_pipeline_jobs(project_id, pipeline_id, scope=["failed"]) - Identify failed jobs
2. retry_pipeline_job(project_id, job_id) - Retry specific job
   OR
   retry_pipeline(project_id, pipeline_id) - Retry all failed jobs
```

#### Trigger Manual Deployment

```
1. list_pipeline_jobs(project_id, pipeline_id, scope=["manual"]) - Find manual jobs
2. play_pipeline_job(project_id, job_id) - Trigger the deployment job
3. get_pipeline_job_output(project_id, job_id, tail=50) - Monitor progress
```

### Token Efficiency for Pipeline Tools

- Use `scope` parameter to filter jobs instead of fetching all
- Use `tail=50` to get recent output instead of full log
- Use `extract="errors"` to get only error messages
- Use `format="text"` for compact output
