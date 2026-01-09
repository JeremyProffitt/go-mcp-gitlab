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
| `extract` | string | Predefined extractor (see Terraform section) |
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
