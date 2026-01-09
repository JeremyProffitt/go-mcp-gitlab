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
