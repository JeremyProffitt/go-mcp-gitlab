## Log Extraction

Use the `extract` parameter with `get_pipeline_job_output` to parse common CI/CD patterns from job logs.

### Available Extractors

| Extract Value | Use Case | Output Contains |
|--------------|----------|-----------------|
| `terraform_outputs` | Get Terraform output values | Variable names and values from `terraform output` |
| `terraform_resources` | Get created/modified resources | Resource types, names, and IDs |
| `terraform_all` | Complete Terraform data | Outputs + resources + summary + AWS assets |
| `aws_assets` | Extract AWS resource identifiers | ARNs, S3 URIs, resource IDs (i-xxx, sg-xxx) |
| `errors` | Extract error/failure messages | Error lines with context |
| `test_results` | Extract test pass/fail results | Test names and outcomes |

### Extractor Selection Guide

| If you want to... | Use extractor |
|-------------------|---------------|
| Find what resources were deployed | `terraform_resources` |
| Get output values (URLs, IDs, etc.) | `terraform_outputs` |
| Full deployment summary | `terraform_all` |
| Find AWS resource ARNs | `aws_assets` |
| Debug build failures | `errors` |
| Check test status | `test_results` |

### Terraform Extraction

#### terraform_outputs

Extracts Terraform output values. Useful for finding deployed resource identifiers.

```
get_pipeline_job_output(project_id, job_id, extract="terraform_outputs")
```

Example output:
```json
{
  "outputs": {
    "api_endpoint": "https://api.example.com",
    "bucket_name": "my-app-prod-bucket",
    "lambda_arn": "arn:aws:lambda:us-east-1:123456789:function:my-api"
  }
}
```

#### terraform_resources

Extracts created/modified/destroyed resources with their IDs.

```
get_pipeline_job_output(project_id, job_id, extract="terraform_resources")
```

Example output:
```json
{
  "resources": [
    {"action": "create", "type": "aws_s3_bucket", "name": "app_bucket", "id": "my-app-prod-bucket"},
    {"action": "update", "type": "aws_lambda_function", "name": "api", "id": "my-api"}
  ]
}
```

#### terraform_all

Complete Terraform extraction - combines outputs, resources, summary, and AWS assets.

```
get_pipeline_job_output(project_id, job_id, extract="terraform_all")
```

#### aws_assets

Extracts AWS resource identifiers from any log (not just Terraform).

```
get_pipeline_job_output(project_id, job_id, extract="aws_assets")
```

Extracts:
- ARNs: `arn:aws:*:*:*:*`
- S3 URIs: `s3://bucket-name/path`
- EC2 IDs: `i-xxxxxxxxx`, `sg-xxxxxxxx`, `vpc-xxxxxxxx`, etc.
- Lambda functions, API Gateway IDs, RDS instances

### Error Extraction

#### errors

Extracts error messages, stack traces, and failure indicators.

```
get_pipeline_job_output(project_id, job_id, extract="errors")
```

Detects patterns like:
- `ERROR:`, `Error:`, `FATAL:`
- Stack traces and exceptions
- Build failure messages
- Test failures

### Test Result Extraction

#### test_results

Extracts test execution results.

```
get_pipeline_job_output(project_id, job_id, extract="test_results")
```

Parses common test frameworks:
- Go test output
- pytest/unittest
- Jest/Mocha
- JUnit-style XML

### Workflow: Get Deployed AWS Resources

```
1. get_latest_release_pipeline(project_id="my-group/my-project")
   -> Returns pipeline info and job list

2. Find the deploy job ID from the jobs list (look for "deploy" or "apply" stage)

3. get_pipeline_job_output(project_id, job_id=<deploy_job_id>, extract="terraform_all")
   -> Returns structured data with all Terraform outputs and AWS resources
```

### Workflow: Debug Failed Deployment

```
1. list_pipelines(project_id, status="failed")
   -> Find the failed pipeline

2. list_pipeline_jobs(project_id, pipeline_id, scope=["failed"])
   -> Identify failed jobs

3. get_pipeline_job_output(project_id, job_id, extract="errors")
   -> Get error messages

4. If Terraform-related:
   get_pipeline_job_output(project_id, job_id, search="Error:|error:", context_lines=5)
   -> Get detailed error context
```

### Output Formats

Use `format` parameter to control output:

| Format | Description | Best For |
|--------|-------------|----------|
| `json` | Structured JSON data | Programmatic parsing |
| `text` | Compact LLM-friendly format | Reducing token usage |

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

### Combining Extract with Search

You can combine `extract` with `search` for targeted extraction:

```
# Get errors mentioning specific resource
get_pipeline_job_output(
  project_id="...",
  job_id=12345,
  extract="errors",
  search="my-lambda-function"
)

# Get terraform resources matching pattern
get_pipeline_job_output(
  project_id="...",
  job_id=12345,
  extract="terraform_resources",
  search="aws_s3"
)
```

### Token Efficiency Tips

- Use `extract` instead of fetching full logs
- Use `format="text"` for compact output
- Combine `extract` with `search` to filter results
- Use `terraform_outputs` if you only need output values (smaller than `terraform_all`)
