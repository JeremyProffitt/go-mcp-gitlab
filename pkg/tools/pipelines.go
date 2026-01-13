package tools

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/go-mcp-gitlab/go-mcp-gitlab/pkg/gitlab"
	"github.com/go-mcp-gitlab/go-mcp-gitlab/pkg/mcp"
)

// Common regex patterns for CI/CD log extraction
var (
	// Terraform patterns
	terraformOutputPattern   = regexp.MustCompile(`(?m)^(\w+)\s*=\s*"?([^"\n]+)"?$`)
	terraformResourcePattern = regexp.MustCompile(`(?m)^(aws_\w+|azurerm_\w+|google_\w+|kubernetes_\w+)\.(\w+):\s*(Creating|Modifying|Destroying|Creation complete|Modifications complete|Destruction complete|Still creating|Still modifying|Still destroying)`)
	terraformResourceIDPattern = regexp.MustCompile(`\[id=([^\]]+)\]`)
	terraformChangeSummary   = regexp.MustCompile(`(?m)^(?:Apply complete!|Plan:).*?(\d+)\s+(?:to\s+)?add.*?(\d+)\s+(?:to\s+)?change.*?(\d+)\s+(?:to\s+)?destroy`)

	// AWS patterns
	awsArnPattern = regexp.MustCompile(`arn:aws:[a-z0-9-]+:[a-z0-9-]*:\d*:[a-zA-Z0-9:/_-]+`)
	awsS3URIPattern = regexp.MustCompile(`s3://[a-zA-Z0-9._-]+(?:/[a-zA-Z0-9._/-]*)?`)
	awsResourceIDPattern = regexp.MustCompile(`(?:i-[0-9a-f]{8,17}|vol-[0-9a-f]{8,17}|snap-[0-9a-f]{8,17}|sg-[0-9a-f]{8,17}|subnet-[0-9a-f]{8,17}|vpc-[0-9a-f]{8,17}|igw-[0-9a-f]{8,17}|rtb-[0-9a-f]{8,17}|acl-[0-9a-f]{8,17}|eni-[0-9a-f]{8,17})`)

	// Error patterns
	errorPattern = regexp.MustCompile(`(?im)^.*(?:error|failed|failure|exception|fatal|panic|traceback|undefined|cannot|unable to|permission denied|access denied|not found|timed? ?out|refused|rejected).*$`)

	// Test result patterns
	testResultPattern = regexp.MustCompile(`(?im)^.*(?:PASS|FAIL|OK|FAILED|ERROR|SKIP|passed|failed|error|skipped|\d+\s+(?:tests?|specs?|examples?)\s+(?:passed|failed|pending)).*$`)
)

// Bridge represents a GitLab pipeline bridge (trigger job).
type Bridge struct {
	ID           int             `json:"id"`
	Name         string          `json:"name"`
	Stage        string          `json:"stage"`
	Status       string          `json:"status"`
	Ref          string          `json:"ref"`
	Tag          bool            `json:"tag"`
	CreatedAt    string          `json:"created_at,omitempty"`
	StartedAt    string          `json:"started_at,omitempty"`
	FinishedAt   string          `json:"finished_at,omitempty"`
	Duration     float64         `json:"duration,omitempty"`
	User         *gitlab.User    `json:"user,omitempty"`
	Pipeline     *gitlab.Pipeline `json:"pipeline,omitempty"`
	WebURL       string          `json:"web_url"`
	DownstreamPipeline *gitlab.Pipeline `json:"downstream_pipeline,omitempty"`
}

// TerraformResource represents a resource found in Terraform output
type TerraformResource struct {
	Type      string `json:"type"`
	Name      string `json:"name"`
	Action    string `json:"action"`
	ID        string `json:"id,omitempty"`
}

// TerraformOutput represents a Terraform output value
type TerraformOutput struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// AWSAssets represents AWS resources extracted from logs
type AWSAssets struct {
	ARNs        []string `json:"arns,omitempty"`
	S3URIs      []string `json:"s3_uris,omitempty"`
	ResourceIDs []string `json:"resource_ids,omitempty"`
}

// JobLogResult represents filtered/extracted job log output
type JobLogResult struct {
	// Raw log content (when no extraction is used)
	Log string `json:"log,omitempty"`

	// Line count info
	TotalLines    int `json:"total_lines"`
	ReturnedLines int `json:"returned_lines"`

	// Extracted data (when using extract parameter)
	TerraformOutputs   []TerraformOutput   `json:"terraform_outputs,omitempty"`
	TerraformResources []TerraformResource `json:"terraform_resources,omitempty"`
	TerraformSummary   map[string]int      `json:"terraform_summary,omitempty"`
	AWSAssets          *AWSAssets          `json:"aws_assets,omitempty"`
	Errors             []string            `json:"errors,omitempty"`
	TestResults        []string            `json:"test_results,omitempty"`
	MatchedLines       []string            `json:"matched_lines,omitempty"`
}

// filterLogLines applies search/filter parameters to log content
func filterLogLines(log string, searchPattern string, head, tail, contextLines int, invertMatch bool) ([]string, int) {
	lines := strings.Split(log, "\n")
	totalLines := len(lines)

	var result []string

	// Apply search pattern if provided
	if searchPattern != "" {
		re, err := regexp.Compile("(?i)" + searchPattern)
		if err != nil {
			// If invalid regex, fall back to substring match
			for i, line := range lines {
				matches := strings.Contains(strings.ToLower(line), strings.ToLower(searchPattern))
				if matches != invertMatch {
					// Add context lines
					start := i - contextLines
					if start < 0 {
						start = 0
					}
					end := i + contextLines + 1
					if end > len(lines) {
						end = len(lines)
					}
					for j := start; j < end; j++ {
						if len(result) == 0 || result[len(result)-1] != lines[j] {
							result = append(result, lines[j])
						}
					}
				}
			}
		} else {
			for i, line := range lines {
				matches := re.MatchString(line)
				if matches != invertMatch {
					start := i - contextLines
					if start < 0 {
						start = 0
					}
					end := i + contextLines + 1
					if end > len(lines) {
						end = len(lines)
					}
					for j := start; j < end; j++ {
						if len(result) == 0 || result[len(result)-1] != lines[j] {
							result = append(result, lines[j])
						}
					}
				}
			}
		}
	} else {
		result = lines
	}

	// Apply head/tail limits
	if head > 0 && len(result) > head {
		result = result[:head]
	}
	if tail > 0 && len(result) > tail {
		result = result[len(result)-tail:]
	}

	return result, totalLines
}

// extractTerraformOutputs extracts Terraform output values from log content
func extractTerraformOutputs(log string) []TerraformOutput {
	var outputs []TerraformOutput

	// Look for the Outputs: section
	outputsIdx := strings.Index(log, "Outputs:")
	if outputsIdx == -1 {
		return outputs
	}

	// Extract from Outputs section onwards
	outputSection := log[outputsIdx:]
	matches := terraformOutputPattern.FindAllStringSubmatch(outputSection, -1)

	seen := make(map[string]bool)
	for _, match := range matches {
		if len(match) >= 3 {
			name := match[1]
			if !seen[name] {
				seen[name] = true
				outputs = append(outputs, TerraformOutput{
					Name:  name,
					Value: strings.TrimSpace(match[2]),
				})
			}
		}
	}

	return outputs
}

// extractTerraformResources extracts Terraform resource operations from log content
func extractTerraformResources(log string) []TerraformResource {
	var resources []TerraformResource

	// Find all lines that match the resource pattern
	lines := strings.Split(log, "\n")
	for _, line := range lines {
		match := terraformResourcePattern.FindStringSubmatch(line)
		if len(match) >= 4 {
			resource := TerraformResource{
				Type:   match[1],
				Name:   match[2],
				Action: match[3],
			}
			// Try to extract ID from the same line
			idMatch := terraformResourceIDPattern.FindStringSubmatch(line)
			if len(idMatch) >= 2 {
				resource.ID = idMatch[1]
			}
			resources = append(resources, resource)
		}
	}

	return resources
}

// extractTerraformSummary extracts the Terraform apply/plan summary
func extractTerraformSummary(log string) map[string]int {
	matches := terraformChangeSummary.FindStringSubmatch(log)
	if len(matches) >= 4 {
		add, _ := fmt.Sscanf(matches[1], "%d", new(int))
		change, _ := fmt.Sscanf(matches[2], "%d", new(int))
		destroy, _ := fmt.Sscanf(matches[3], "%d", new(int))
		_ = add
		_ = change
		_ = destroy

		return map[string]int{
			"add":     atoi(matches[1]),
			"change":  atoi(matches[2]),
			"destroy": atoi(matches[3]),
		}
	}
	return nil
}

// atoi converts string to int, returns 0 on error
func atoi(s string) int {
	var n int
	fmt.Sscanf(s, "%d", &n)
	return n
}

// extractAWSAssets extracts AWS ARNs, S3 URIs, and resource IDs from log content
func extractAWSAssets(log string) *AWSAssets {
	assets := &AWSAssets{}

	// Extract ARNs
	arnMatches := awsArnPattern.FindAllString(log, -1)
	seen := make(map[string]bool)
	for _, arn := range arnMatches {
		if !seen[arn] {
			seen[arn] = true
			assets.ARNs = append(assets.ARNs, arn)
		}
	}

	// Extract S3 URIs
	s3Matches := awsS3URIPattern.FindAllString(log, -1)
	seen = make(map[string]bool)
	for _, uri := range s3Matches {
		if !seen[uri] {
			seen[uri] = true
			assets.S3URIs = append(assets.S3URIs, uri)
		}
	}

	// Extract resource IDs
	idMatches := awsResourceIDPattern.FindAllString(log, -1)
	seen = make(map[string]bool)
	for _, id := range idMatches {
		if !seen[id] {
			seen[id] = true
			assets.ResourceIDs = append(assets.ResourceIDs, id)
		}
	}

	if len(assets.ARNs) == 0 && len(assets.S3URIs) == 0 && len(assets.ResourceIDs) == 0 {
		return nil
	}

	return assets
}

// extractErrors extracts error messages from log content
func extractErrors(log string) []string {
	matches := errorPattern.FindAllString(log, -1)

	// Deduplicate
	seen := make(map[string]bool)
	var errors []string
	for _, match := range matches {
		trimmed := strings.TrimSpace(match)
		if !seen[trimmed] {
			seen[trimmed] = true
			errors = append(errors, trimmed)
		}
	}

	return errors
}

// extractTestResults extracts test result lines from log content
func extractTestResults(log string) []string {
	matches := testResultPattern.FindAllString(log, -1)

	seen := make(map[string]bool)
	var results []string
	for _, match := range matches {
		trimmed := strings.TrimSpace(match)
		if !seen[trimmed] {
			seen[trimmed] = true
			results = append(results, trimmed)
		}
	}

	return results
}

// formatJobLogResultAsText formats a JobLogResult as compact, LLM-friendly text
func formatJobLogResultAsText(result *JobLogResult) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Total lines: %d | Returned: %d\n", result.TotalLines, result.ReturnedLines))

	if len(result.TerraformOutputs) > 0 {
		sb.WriteString("\n=== Terraform Outputs ===\n")
		for _, o := range result.TerraformOutputs {
			sb.WriteString(fmt.Sprintf("%s: %s\n", o.Name, o.Value))
		}
	}

	if len(result.TerraformResources) > 0 {
		sb.WriteString("\n=== Terraform Resources ===\n")
		for _, r := range result.TerraformResources {
			if r.ID != "" {
				sb.WriteString(fmt.Sprintf("%s.%s: %s [id=%s]\n", r.Type, r.Name, r.Action, r.ID))
			} else {
				sb.WriteString(fmt.Sprintf("%s.%s: %s\n", r.Type, r.Name, r.Action))
			}
		}
	}

	if result.TerraformSummary != nil {
		sb.WriteString("\n=== Terraform Summary ===\n")
		sb.WriteString(fmt.Sprintf("Add: %d | Change: %d | Destroy: %d\n",
			result.TerraformSummary["add"],
			result.TerraformSummary["change"],
			result.TerraformSummary["destroy"]))
	}

	if result.AWSAssets != nil {
		sb.WriteString("\n=== AWS Assets ===\n")
		if len(result.AWSAssets.ARNs) > 0 {
			sb.WriteString("ARNs:\n")
			for _, arn := range result.AWSAssets.ARNs {
				sb.WriteString(fmt.Sprintf("  %s\n", arn))
			}
		}
		if len(result.AWSAssets.S3URIs) > 0 {
			sb.WriteString("S3 URIs:\n")
			for _, uri := range result.AWSAssets.S3URIs {
				sb.WriteString(fmt.Sprintf("  %s\n", uri))
			}
		}
		if len(result.AWSAssets.ResourceIDs) > 0 {
			sb.WriteString("Resource IDs:\n")
			for _, id := range result.AWSAssets.ResourceIDs {
				sb.WriteString(fmt.Sprintf("  %s\n", id))
			}
		}
	}

	if len(result.Errors) > 0 {
		sb.WriteString("\n=== Errors ===\n")
		for _, e := range result.Errors {
			sb.WriteString(fmt.Sprintf("- %s\n", e))
		}
	}

	if len(result.TestResults) > 0 {
		sb.WriteString("\n=== Test Results ===\n")
		for _, t := range result.TestResults {
			sb.WriteString(fmt.Sprintf("%s\n", t))
		}
	}

	if len(result.MatchedLines) > 0 {
		sb.WriteString("\n=== Matched Lines ===\n")
		for _, line := range result.MatchedLines {
			sb.WriteString(fmt.Sprintf("%s\n", line))
		}
	}

	return sb.String()
}

// registerListPipelines registers the list_pipelines tool.
func registerListPipelines(server *mcp.Server) {
	server.RegisterTool(
		mcp.Tool{
			Name:        "list_pipelines",
			Description: "List pipelines for a project. Returns a paginated array of pipeline objects with ID, status, ref, SHA, and timestamps. Filter by status to find running/failed pipelines.",
			InputSchema: mcp.JSONSchema{
				Type: "object",
				Properties: map[string]mcp.Property{
					"project_id": {
						Type:        "string",
						Description: "The project identifier - either a numeric ID (e.g., 42) or URL-encoded path (e.g., my-group/my-project)",
					},
					"scope": {
						Type:        "string",
						Description: "Filter pipelines by scope: running, pending, finished, branches, or tags",
						Enum:        []string{"running", "pending", "finished", "branches", "tags"},
					},
					"status": {
						Type:        "string",
						Description: "Filter pipelines by status: created, waiting_for_resource, preparing, pending, running, success, failed, canceled, skipped, manual, scheduled",
						Enum:        []string{"created", "waiting_for_resource", "preparing", "pending", "running", "success", "failed", "canceled", "skipped", "manual", "scheduled"},
					},
					"ref": {
						Type:        "string",
						Description: "Filter pipelines by the ref (branch or tag name)",
					},
					"sha": {
						Type:        "string",
						Description: "Filter pipelines by the SHA of the commit",
					},
					"page": {
						Type:        "integer",
						Description: "Page number for pagination",
						Default:     1,
						Minimum:     mcp.IntPtr(1),
					},
					"per_page": {
						Type:        "integer",
						Description: "Number of items per page",
						Default:     20,
						Minimum:     mcp.IntPtr(1),
						Maximum:     mcp.IntPtr(100),
					},
				},
				Required: []string{"project_id"},
			},
			Annotations: &mcp.ToolAnnotations{
				ReadOnlyHint: true,
			},
		},
		func(args map[string]interface{}) (*mcp.CallToolResult, error) {
			c := GetContext()
			if c == nil {
				return ErrorResult("tool context not initialized")
			}
			c.Logger.ToolCall("list_pipelines", args)

			projectID := GetString(args, "project_id", "")
			if projectID == "" {
				return ErrorResult("project_id is required")
			}

			params := url.Values{}
			if scope := GetString(args, "scope", ""); scope != "" {
				params.Set("scope", scope)
			}
			if status := GetString(args, "status", ""); status != "" {
				params.Set("status", status)
			}
			if ref := GetString(args, "ref", ""); ref != "" {
				params.Set("ref", ref)
			}
			if sha := GetString(args, "sha", ""); sha != "" {
				params.Set("sha", sha)
			}
			if page := GetInt(args, "page", 0); page > 0 {
				params.Set("page", fmt.Sprintf("%d", page))
			}
			if perPage := GetInt(args, "per_page", 0); perPage > 0 {
				params.Set("per_page", fmt.Sprintf("%d", perPage))
			}

			endpoint := fmt.Sprintf("/projects/%s/pipelines", url.PathEscape(projectID))
			if len(params) > 0 {
				endpoint += "?" + params.Encode()
			}

			var pipelines []gitlab.Pipeline
			pagination, err := c.Client.GetWithPagination(endpoint, &pipelines)
			if err != nil {
				return ErrorResult(fmt.Sprintf("Failed to list pipelines: %v", err))
			}

			result := map[string]interface{}{
				"pipelines":  pipelines,
				"pagination": pagination,
			}

			return JSONResult(result)
		},
	)
}

// registerGetPipeline registers the get_pipeline tool.
func registerGetPipeline(server *mcp.Server) {
	server.RegisterTool(
		mcp.Tool{
			Name:        "get_pipeline",
			Description: "Get comprehensive details of a specific pipeline by ID. Returns full pipeline info including status, ref, SHA, user who triggered it, timestamps, and duration.",
			InputSchema: mcp.JSONSchema{
				Type: "object",
				Properties: map[string]mcp.Property{
					"project_id": {
						Type:        "string",
						Description: "The project identifier - either a numeric ID (e.g., 42) or URL-encoded path (e.g., my-group/my-project)",
					},
					"pipeline_id": {
						Type:        "integer",
						Description: "The ID of the pipeline",
					},
				},
				Required: []string{"project_id", "pipeline_id"},
			},
			Annotations: &mcp.ToolAnnotations{
				ReadOnlyHint: true,
			},
		},
		func(args map[string]interface{}) (*mcp.CallToolResult, error) {
			c := GetContext()
			if c == nil {
				return ErrorResult("tool context not initialized")
			}
			c.Logger.ToolCall("get_pipeline", args)

			projectID := GetString(args, "project_id", "")
			if projectID == "" {
				return ErrorResult("project_id is required")
			}
			pipelineID := GetInt(args, "pipeline_id", 0)
			if pipelineID == 0 {
				return ErrorResult("pipeline_id is required")
			}

			endpoint := fmt.Sprintf("/projects/%s/pipelines/%d", url.PathEscape(projectID), pipelineID)

			var pipeline gitlab.Pipeline
			if err := c.Client.Get(endpoint, &pipeline); err != nil {
				return ErrorResult(fmt.Sprintf("Failed to get pipeline: %v", err))
			}

			return JSONResult(pipeline)
		},
	)
}

// registerCreatePipeline registers the create_pipeline tool.
func registerCreatePipeline(server *mcp.Server) {
	server.RegisterTool(
		mcp.Tool{
			Name:        "create_pipeline",
			Description: "Create a new pipeline for a project. Triggers a pipeline on the specified branch or tag.",
			InputSchema: mcp.JSONSchema{
				Type: "object",
				Properties: map[string]mcp.Property{
					"project_id": {
						Type:        "string",
						Description: "The project identifier - either a numeric ID (e.g., 42) or URL-encoded path (e.g., my-group/my-project)",
					},
					"ref": {
						Type:        "string",
						Description: "The branch or tag name to run the pipeline on",
					},
					"variables": {
						Type:        "array",
						Description: "Array of variables to pass to the pipeline. Each variable should have 'key' and 'value' properties.",
						Items: &mcp.Property{
							Type: "object",
							Properties: map[string]mcp.Property{
								"key": {
									Type:        "string",
									Description: "The variable name",
								},
								"value": {
									Type:        "string",
									Description: "The variable value",
								},
							},
						},
					},
				},
				Required: []string{"project_id", "ref"},
			},
		},
		func(args map[string]interface{}) (*mcp.CallToolResult, error) {
			c := GetContext()
			if c == nil {
				return ErrorResult("tool context not initialized")
			}
			c.Logger.ToolCall("create_pipeline", args)

			projectID := GetString(args, "project_id", "")
			if projectID == "" {
				return ErrorResult("project_id is required")
			}
			ref := GetString(args, "ref", "")
			if ref == "" {
				return ErrorResult("ref is required")
			}

			body := map[string]interface{}{
				"ref": ref,
			}

			// Handle variables array
			if varsRaw, ok := args["variables"]; ok && varsRaw != nil {
				if varsArray, ok := varsRaw.([]interface{}); ok && len(varsArray) > 0 {
					body["variables"] = varsArray
				}
			}

			endpoint := fmt.Sprintf("/projects/%s/pipeline", url.PathEscape(projectID))

			var pipeline gitlab.Pipeline
			if err := c.Client.Post(endpoint, body, &pipeline); err != nil {
				return ErrorResult(fmt.Sprintf("Failed to create pipeline: %v", err))
			}

			return JSONResult(pipeline)
		},
	)
}

// registerRetryPipeline registers the retry_pipeline tool.
func registerRetryPipeline(server *mcp.Server) {
	server.RegisterTool(
		mcp.Tool{
			Name:        "retry_pipeline",
			Description: "Retry all failed jobs in a pipeline.",
			InputSchema: mcp.JSONSchema{
				Type: "object",
				Properties: map[string]mcp.Property{
					"project_id": {
						Type:        "string",
						Description: "The project identifier - either a numeric ID (e.g., 42) or URL-encoded path (e.g., my-group/my-project)",
					},
					"pipeline_id": {
						Type:        "integer",
						Description: "The ID of the pipeline",
					},
				},
				Required: []string{"project_id", "pipeline_id"},
			},
		},
		func(args map[string]interface{}) (*mcp.CallToolResult, error) {
			c := GetContext()
			if c == nil {
				return ErrorResult("tool context not initialized")
			}
			c.Logger.ToolCall("retry_pipeline", args)

			projectID := GetString(args, "project_id", "")
			if projectID == "" {
				return ErrorResult("project_id is required")
			}
			pipelineID := GetInt(args, "pipeline_id", 0)
			if pipelineID == 0 {
				return ErrorResult("pipeline_id is required")
			}

			endpoint := fmt.Sprintf("/projects/%s/pipelines/%d/retry", url.PathEscape(projectID), pipelineID)

			var pipeline gitlab.Pipeline
			if err := c.Client.Post(endpoint, nil, &pipeline); err != nil {
				return ErrorResult(fmt.Sprintf("Failed to retry pipeline: %v", err))
			}

			return JSONResult(pipeline)
		},
	)
}

// registerCancelPipeline registers the cancel_pipeline tool.
func registerCancelPipeline(server *mcp.Server) {
	server.RegisterTool(
		mcp.Tool{
			Name:        "cancel_pipeline",
			Description: "Cancel a running pipeline and all its jobs.",
			InputSchema: mcp.JSONSchema{
				Type: "object",
				Properties: map[string]mcp.Property{
					"project_id": {
						Type:        "string",
						Description: "The project identifier - either a numeric ID (e.g., 42) or URL-encoded path (e.g., my-group/my-project)",
					},
					"pipeline_id": {
						Type:        "integer",
						Description: "The ID of the pipeline",
					},
				},
				Required: []string{"project_id", "pipeline_id"},
			},
		},
		func(args map[string]interface{}) (*mcp.CallToolResult, error) {
			c := GetContext()
			if c == nil {
				return ErrorResult("tool context not initialized")
			}
			c.Logger.ToolCall("cancel_pipeline", args)

			projectID := GetString(args, "project_id", "")
			if projectID == "" {
				return ErrorResult("project_id is required")
			}
			pipelineID := GetInt(args, "pipeline_id", 0)
			if pipelineID == 0 {
				return ErrorResult("pipeline_id is required")
			}

			endpoint := fmt.Sprintf("/projects/%s/pipelines/%d/cancel", url.PathEscape(projectID), pipelineID)

			var pipeline gitlab.Pipeline
			if err := c.Client.Post(endpoint, nil, &pipeline); err != nil {
				return ErrorResult(fmt.Sprintf("Failed to cancel pipeline: %v", err))
			}

			return JSONResult(pipeline)
		},
	)
}

// registerListPipelineJobs registers the list_pipeline_jobs tool.
func registerListPipelineJobs(server *mcp.Server) {
	server.RegisterTool(
		mcp.Tool{
			Name:        "list_pipeline_jobs",
			Description: "List all jobs for a specific pipeline.",
			InputSchema: mcp.JSONSchema{
				Type: "object",
				Properties: map[string]mcp.Property{
					"project_id": {
						Type:        "string",
						Description: "The project identifier - either a numeric ID (e.g., 42) or URL-encoded path (e.g., my-group/my-project)",
					},
					"pipeline_id": {
						Type:        "integer",
						Description: "The ID of the pipeline",
					},
					"scope": {
						Type:        "array",
						Description: "Filter jobs by scope. Possible values: created, pending, running, failed, success, canceled, skipped, manual",
						Items: &mcp.Property{
							Type: "string",
							Enum: []string{"created", "pending", "running", "failed", "success", "canceled", "skipped", "manual"},
						},
					},
					"page": {
						Type:        "integer",
						Description: "Page number for pagination",
						Default:     1,
						Minimum:     mcp.IntPtr(1),
					},
					"per_page": {
						Type:        "integer",
						Description: "Number of items per page",
						Default:     20,
						Minimum:     mcp.IntPtr(1),
						Maximum:     mcp.IntPtr(100),
					},
				},
				Required: []string{"project_id", "pipeline_id"},
			},
			Annotations: &mcp.ToolAnnotations{
				ReadOnlyHint: true,
			},
		},
		func(args map[string]interface{}) (*mcp.CallToolResult, error) {
			c := GetContext()
			if c == nil {
				return ErrorResult("tool context not initialized")
			}
			c.Logger.ToolCall("list_pipeline_jobs", args)

			projectID := GetString(args, "project_id", "")
			if projectID == "" {
				return ErrorResult("project_id is required")
			}
			pipelineID := GetInt(args, "pipeline_id", 0)
			if pipelineID == 0 {
				return ErrorResult("pipeline_id is required")
			}

			params := url.Values{}
			if scope := GetStringArray(args, "scope"); len(scope) > 0 {
				params.Set("scope[]", strings.Join(scope, ","))
			}
			if page := GetInt(args, "page", 0); page > 0 {
				params.Set("page", fmt.Sprintf("%d", page))
			}
			if perPage := GetInt(args, "per_page", 0); perPage > 0 {
				params.Set("per_page", fmt.Sprintf("%d", perPage))
			}

			endpoint := fmt.Sprintf("/projects/%s/pipelines/%d/jobs", url.PathEscape(projectID), pipelineID)
			if len(params) > 0 {
				endpoint += "?" + params.Encode()
			}

			var jobs []gitlab.Job
			pagination, err := c.Client.GetWithPagination(endpoint, &jobs)
			if err != nil {
				return ErrorResult(fmt.Sprintf("Failed to list pipeline jobs: %v", err))
			}

			result := map[string]interface{}{
				"jobs":       jobs,
				"pagination": pagination,
			}

			return JSONResult(result)
		},
	)
}

// registerListPipelineTriggerJobs registers the list_pipeline_trigger_jobs tool.
func registerListPipelineTriggerJobs(server *mcp.Server) {
	server.RegisterTool(
		mcp.Tool{
			Name:        "list_pipeline_trigger_jobs",
			Description: "List all trigger jobs (bridges) for a specific pipeline. Bridges are jobs that trigger downstream pipelines.",
			InputSchema: mcp.JSONSchema{
				Type: "object",
				Properties: map[string]mcp.Property{
					"project_id": {
						Type:        "string",
						Description: "The project identifier - either a numeric ID (e.g., 42) or URL-encoded path (e.g., my-group/my-project)",
					},
					"pipeline_id": {
						Type:        "integer",
						Description: "The ID of the pipeline",
					},
					"scope": {
						Type:        "array",
						Description: "Filter bridges by scope. Possible values: created, pending, running, failed, success, canceled, skipped, manual",
						Items: &mcp.Property{
							Type: "string",
							Enum: []string{"created", "pending", "running", "failed", "success", "canceled", "skipped", "manual"},
						},
					},
					"page": {
						Type:        "integer",
						Description: "Page number for pagination",
						Default:     1,
						Minimum:     mcp.IntPtr(1),
					},
					"per_page": {
						Type:        "integer",
						Description: "Number of items per page",
						Default:     20,
						Minimum:     mcp.IntPtr(1),
						Maximum:     mcp.IntPtr(100),
					},
				},
				Required: []string{"project_id", "pipeline_id"},
			},
			Annotations: &mcp.ToolAnnotations{
				ReadOnlyHint: true,
			},
		},
		func(args map[string]interface{}) (*mcp.CallToolResult, error) {
			c := GetContext()
			if c == nil {
				return ErrorResult("tool context not initialized")
			}
			c.Logger.ToolCall("list_pipeline_trigger_jobs", args)

			projectID := GetString(args, "project_id", "")
			if projectID == "" {
				return ErrorResult("project_id is required")
			}
			pipelineID := GetInt(args, "pipeline_id", 0)
			if pipelineID == 0 {
				return ErrorResult("pipeline_id is required")
			}

			params := url.Values{}
			if scope := GetStringArray(args, "scope"); len(scope) > 0 {
				params.Set("scope[]", strings.Join(scope, ","))
			}
			if page := GetInt(args, "page", 0); page > 0 {
				params.Set("page", fmt.Sprintf("%d", page))
			}
			if perPage := GetInt(args, "per_page", 0); perPage > 0 {
				params.Set("per_page", fmt.Sprintf("%d", perPage))
			}

			endpoint := fmt.Sprintf("/projects/%s/pipelines/%d/bridges", url.PathEscape(projectID), pipelineID)
			if len(params) > 0 {
				endpoint += "?" + params.Encode()
			}

			var bridges []Bridge
			pagination, err := c.Client.GetWithPagination(endpoint, &bridges)
			if err != nil {
				return ErrorResult(fmt.Sprintf("Failed to list pipeline trigger jobs: %v", err))
			}

			result := map[string]interface{}{
				"bridges":    bridges,
				"pagination": pagination,
			}

			return JSONResult(result)
		},
	)
}

// registerGetPipelineJob registers the get_pipeline_job tool.
func registerGetPipelineJob(server *mcp.Server) {
	server.RegisterTool(
		mcp.Tool{
			Name:        "get_pipeline_job",
			Description: "Get details of a specific job by ID.",
			InputSchema: mcp.JSONSchema{
				Type: "object",
				Properties: map[string]mcp.Property{
					"project_id": {
						Type:        "string",
						Description: "The project identifier - either a numeric ID (e.g., 42) or URL-encoded path (e.g., my-group/my-project)",
					},
					"job_id": {
						Type:        "integer",
						Description: "The ID of the job",
					},
				},
				Required: []string{"project_id", "job_id"},
			},
			Annotations: &mcp.ToolAnnotations{
				ReadOnlyHint: true,
			},
		},
		func(args map[string]interface{}) (*mcp.CallToolResult, error) {
			c := GetContext()
			if c == nil {
				return ErrorResult("tool context not initialized")
			}
			c.Logger.ToolCall("get_pipeline_job", args)

			projectID := GetString(args, "project_id", "")
			if projectID == "" {
				return ErrorResult("project_id is required")
			}
			jobID := GetInt(args, "job_id", 0)
			if jobID == 0 {
				return ErrorResult("job_id is required")
			}

			endpoint := fmt.Sprintf("/projects/%s/jobs/%d", url.PathEscape(projectID), jobID)

			var job gitlab.Job
			if err := c.Client.Get(endpoint, &job); err != nil {
				return ErrorResult(fmt.Sprintf("Failed to get job: %v", err))
			}

			return JSONResult(job)
		},
	)
}

// registerGetPipelineJobOutput registers the get_pipeline_job_output tool.
func registerGetPipelineJobOutput(server *mcp.Server) {
	server.RegisterTool(
		mcp.Tool{
			Name: "get_pipeline_job_output",
			Description: `Get the log (trace) output of a specific job with optional filtering and extraction.

BASIC USAGE: Returns the full job log as plain text when no filters are specified.

SEARCH & FILTER OPTIONS:
- search: Regex pattern to filter lines (case-insensitive). Use for custom searches like "bucket|lambda" or "deployment.*failed"
- head: Return only the first N lines (useful for seeing job startup)
- tail: Return only the last N lines (useful for seeing final results/errors)
- context_lines: Include N lines before/after each match (like grep -C)
- invert_match: Return lines that DON'T match the search pattern

PREDEFINED EXTRACTORS (use 'extract' parameter):
- "terraform_outputs": Extract Terraform output values (bucket_name, api_url, etc.)
- "terraform_resources": Extract resource operations with IDs (aws_s3_bucket.main: Creation complete [id=my-bucket])
- "terraform_all": Extract both outputs and resources with apply/plan summary
- "aws_assets": Extract all AWS ARNs, S3 URIs, and resource IDs (i-xxx, vol-xxx, sg-xxx, etc.)
- "errors": Extract error/failure messages from the log
- "test_results": Extract test pass/fail/skip result lines

COMMON USE CASES:
1. Find why a job failed: use extract="errors" or search="error|failed|exception"
2. Get Terraform-created resources: use extract="terraform_all" or extract="aws_assets"
3. Check test results: use extract="test_results"
4. See deployment outputs: use extract="terraform_outputs"
5. Get last 100 lines of long job: use tail=100
6. Find specific resource: use search="aws_lambda|my-function-name"`,
			InputSchema: mcp.JSONSchema{
				Type: "object",
				Properties: map[string]mcp.Property{
					"project_id": {
						Type:        "string",
						Description: "The project identifier - either a numeric ID (e.g., 42) or URL-encoded path (e.g., my-group/my-project)",
					},
					"job_id": {
						Type:        "integer",
						Description: "The ID of the job",
					},
					"search": {
						Type:        "string",
						Description: "Regex pattern to filter log lines (case-insensitive). Examples: 'error|failed', 'aws_s3_bucket', 'terraform.*complete'",
					},
					"head": {
						Type:        "integer",
						Description: "Return only the first N lines of the (filtered) output",
					},
					"tail": {
						Type:        "integer",
						Description: "Return only the last N lines of the (filtered) output",
					},
					"context_lines": {
						Type:        "integer",
						Description: "Number of lines to include before and after each search match (like grep -C). Default: 0",
					},
					"invert_match": {
						Type:        "boolean",
						Description: "If true, return lines that DON'T match the search pattern (like grep -v)",
					},
					"extract": {
						Type:        "string",
						Description: "Use a predefined extractor to parse structured data from logs",
						Enum: []string{
							"terraform_outputs",
							"terraform_resources",
							"terraform_all",
							"aws_assets",
							"errors",
							"test_results",
						},
					},
					"format": {
						Type:        "string",
						Description: "Output format: 'json' for structured data (default), 'text' for compact LLM-friendly format with less tokens",
						Enum:        []string{"json", "text"},
					},
				},
				Required: []string{"project_id", "job_id"},
			},
			Annotations: &mcp.ToolAnnotations{
				ReadOnlyHint: true,
			},
		},
		func(args map[string]interface{}) (*mcp.CallToolResult, error) {
			c := GetContext()
			if c == nil {
				return ErrorResult("tool context not initialized")
			}
			c.Logger.ToolCall("get_pipeline_job_output", args)

			projectID := GetString(args, "project_id", "")
			if projectID == "" {
				return ErrorResult("project_id is required")
			}
			jobID := GetInt(args, "job_id", 0)
			if jobID == 0 {
				return ErrorResult("job_id is required")
			}

			// Get optional filter parameters
			searchPattern := GetString(args, "search", "")
			head := GetInt(args, "head", 0)
			tail := GetInt(args, "tail", 0)
			contextLines := GetInt(args, "context_lines", 0)
			invertMatch := GetBool(args, "invert_match", false)
			extract := GetString(args, "extract", "")
			format := GetString(args, "format", "json")

			endpoint := fmt.Sprintf("/projects/%s/jobs/%d/trace", url.PathEscape(projectID), jobID)

			trace, err := c.Client.GetText(endpoint)
			if err != nil {
				return ErrorResult(fmt.Sprintf("Failed to get job output: %v", err))
			}

			// If using an extractor, return structured data
			if extract != "" {
				result := JobLogResult{
					TotalLines: len(strings.Split(trace, "\n")),
				}

				switch extract {
				case "terraform_outputs":
					result.TerraformOutputs = extractTerraformOutputs(trace)
					result.ReturnedLines = len(result.TerraformOutputs)

				case "terraform_resources":
					result.TerraformResources = extractTerraformResources(trace)
					result.ReturnedLines = len(result.TerraformResources)

				case "terraform_all":
					result.TerraformOutputs = extractTerraformOutputs(trace)
					result.TerraformResources = extractTerraformResources(trace)
					result.TerraformSummary = extractTerraformSummary(trace)
					result.AWSAssets = extractAWSAssets(trace)
					result.ReturnedLines = len(result.TerraformOutputs) + len(result.TerraformResources)

				case "aws_assets":
					result.AWSAssets = extractAWSAssets(trace)
					if result.AWSAssets != nil {
						result.ReturnedLines = len(result.AWSAssets.ARNs) + len(result.AWSAssets.S3URIs) + len(result.AWSAssets.ResourceIDs)
					}

				case "errors":
					result.Errors = extractErrors(trace)
					result.ReturnedLines = len(result.Errors)

				case "test_results":
					result.TestResults = extractTestResults(trace)
					result.ReturnedLines = len(result.TestResults)

				default:
					return ErrorResult(fmt.Sprintf("Unknown extract type: %s. Valid options: terraform_outputs, terraform_resources, terraform_all, aws_assets, errors, test_results", extract))
				}

				// Return in requested format
				if format == "text" {
					return TextResult(formatJobLogResultAsText(&result))
				}
				return JSONResult(result)
			}

			// If using search/filter parameters, apply them
			if searchPattern != "" || head > 0 || tail > 0 {
				lines, totalLines := filterLogLines(trace, searchPattern, head, tail, contextLines, invertMatch)
				result := JobLogResult{
					TotalLines:    totalLines,
					ReturnedLines: len(lines),
					MatchedLines:  lines,
				}
				// Return in requested format
				if format == "text" {
					return TextResult(formatJobLogResultAsText(&result))
				}
				return JSONResult(result)
			}

			// Default: return full log as text
			return TextResult(trace)
		},
	)
}

// registerPlayPipelineJob registers the play_pipeline_job tool.
func registerPlayPipelineJob(server *mcp.Server) {
	server.RegisterTool(
		mcp.Tool{
			Name:        "play_pipeline_job",
			Description: "Trigger a manual job to start. Only works for jobs that are in 'manual' status.",
			InputSchema: mcp.JSONSchema{
				Type: "object",
				Properties: map[string]mcp.Property{
					"project_id": {
						Type:        "string",
						Description: "The project identifier - either a numeric ID (e.g., 42) or URL-encoded path (e.g., my-group/my-project)",
					},
					"job_id": {
						Type:        "integer",
						Description: "The ID of the job",
					},
					"job_variables": {
						Type:        "array",
						Description: "Array of variables to pass to the job. Each variable should have 'key' and 'value' properties.",
						Items: &mcp.Property{
							Type: "object",
							Properties: map[string]mcp.Property{
								"key": {
									Type:        "string",
									Description: "The variable name",
								},
								"value": {
									Type:        "string",
									Description: "The variable value",
								},
							},
						},
					},
				},
				Required: []string{"project_id", "job_id"},
			},
		},
		func(args map[string]interface{}) (*mcp.CallToolResult, error) {
			c := GetContext()
			if c == nil {
				return ErrorResult("tool context not initialized")
			}
			c.Logger.ToolCall("play_pipeline_job", args)

			projectID := GetString(args, "project_id", "")
			if projectID == "" {
				return ErrorResult("project_id is required")
			}
			jobID := GetInt(args, "job_id", 0)
			if jobID == 0 {
				return ErrorResult("job_id is required")
			}

			var body map[string]interface{}
			// Handle job_variables array
			if varsRaw, ok := args["job_variables"]; ok && varsRaw != nil {
				if varsArray, ok := varsRaw.([]interface{}); ok && len(varsArray) > 0 {
					body = map[string]interface{}{
						"job_variables_attributes": varsArray,
					}
				}
			}

			endpoint := fmt.Sprintf("/projects/%s/jobs/%d/play", url.PathEscape(projectID), jobID)

			var job gitlab.Job
			if err := c.Client.Post(endpoint, body, &job); err != nil {
				return ErrorResult(fmt.Sprintf("Failed to play job: %v", err))
			}

			return JSONResult(job)
		},
	)
}

// registerRetryPipelineJob registers the retry_pipeline_job tool.
func registerRetryPipelineJob(server *mcp.Server) {
	server.RegisterTool(
		mcp.Tool{
			Name:        "retry_pipeline_job",
			Description: "Retry a failed or canceled job.",
			InputSchema: mcp.JSONSchema{
				Type: "object",
				Properties: map[string]mcp.Property{
					"project_id": {
						Type:        "string",
						Description: "The project identifier - either a numeric ID (e.g., 42) or URL-encoded path (e.g., my-group/my-project)",
					},
					"job_id": {
						Type:        "integer",
						Description: "The ID of the job",
					},
				},
				Required: []string{"project_id", "job_id"},
			},
		},
		func(args map[string]interface{}) (*mcp.CallToolResult, error) {
			c := GetContext()
			if c == nil {
				return ErrorResult("tool context not initialized")
			}
			c.Logger.ToolCall("retry_pipeline_job", args)

			projectID := GetString(args, "project_id", "")
			if projectID == "" {
				return ErrorResult("project_id is required")
			}
			jobID := GetInt(args, "job_id", 0)
			if jobID == 0 {
				return ErrorResult("job_id is required")
			}

			endpoint := fmt.Sprintf("/projects/%s/jobs/%d/retry", url.PathEscape(projectID), jobID)

			var job gitlab.Job
			if err := c.Client.Post(endpoint, nil, &job); err != nil {
				return ErrorResult(fmt.Sprintf("Failed to retry job: %v", err))
			}

			return JSONResult(job)
		},
	)
}

// registerCancelPipelineJob registers the cancel_pipeline_job tool.
func registerCancelPipelineJob(server *mcp.Server) {
	server.RegisterTool(
		mcp.Tool{
			Name:        "cancel_pipeline_job",
			Description: "Cancel a running job.",
			InputSchema: mcp.JSONSchema{
				Type: "object",
				Properties: map[string]mcp.Property{
					"project_id": {
						Type:        "string",
						Description: "The project identifier - either a numeric ID (e.g., 42) or URL-encoded path (e.g., my-group/my-project)",
					},
					"job_id": {
						Type:        "integer",
						Description: "The ID of the job",
					},
				},
				Required: []string{"project_id", "job_id"},
			},
		},
		func(args map[string]interface{}) (*mcp.CallToolResult, error) {
			c := GetContext()
			if c == nil {
				return ErrorResult("tool context not initialized")
			}
			c.Logger.ToolCall("cancel_pipeline_job", args)

			projectID := GetString(args, "project_id", "")
			if projectID == "" {
				return ErrorResult("project_id is required")
			}
			jobID := GetInt(args, "job_id", 0)
			if jobID == 0 {
				return ErrorResult("job_id is required")
			}

			endpoint := fmt.Sprintf("/projects/%s/jobs/%d/cancel", url.PathEscape(projectID), jobID)

			var job gitlab.Job
			if err := c.Client.Post(endpoint, nil, &job); err != nil {
				return ErrorResult(fmt.Sprintf("Failed to cancel job: %v", err))
			}

			return JSONResult(job)
		},
	)
}

// registerGetLatestReleasePipeline registers the get_latest_release_pipeline tool.
func registerGetLatestReleasePipeline(server *mcp.Server) {
	server.RegisterTool(
		mcp.Tool{
			Name: "get_latest_release_pipeline",
			Description: `Get pipeline information for the latest release (tag) in a project.

This is useful for:
- Checking the deployment status of the most recent release
- Getting job logs from the production deployment
- Extracting Terraform outputs or AWS assets from the release pipeline

The tool fetches the latest release, finds the pipeline that ran for that tag, and returns pipeline details along with its jobs.

Combine with get_pipeline_job_output to extract specific data:
1. Use this tool to find the pipeline and job IDs
2. Use get_pipeline_job_output with extract="terraform_all" or extract="aws_assets" to get deployed resources`,
			InputSchema: mcp.JSONSchema{
				Type: "object",
				Properties: map[string]mcp.Property{
					"project_id": {
						Type:        "string",
						Description: "The project identifier - either a numeric ID (e.g., 42) or URL-encoded path (e.g., my-group/my-project)",
					},
					"include_jobs": {
						Type:        "boolean",
						Description: "If true, also fetch and include the list of jobs for the pipeline (default: true)",
					},
				},
				Required: []string{"project_id"},
			},
			Annotations: &mcp.ToolAnnotations{
				ReadOnlyHint: true,
			},
		},
		func(args map[string]interface{}) (*mcp.CallToolResult, error) {
			c := GetContext()
			if c == nil {
				return ErrorResult("tool context not initialized")
			}
			c.Logger.ToolCall("get_latest_release_pipeline", args)

			projectID := GetString(args, "project_id", "")
			if projectID == "" {
				return ErrorResult("project_id is required")
			}

			includeJobs := GetBool(args, "include_jobs", true)

			// Step 1: Get the latest release
			releasesEndpoint := fmt.Sprintf("/projects/%s/releases?per_page=1", url.PathEscape(projectID))
			var releases []struct {
				TagName   string `json:"tag_name"`
				Name      string `json:"name"`
				CreatedAt string `json:"created_at"`
			}
			if err := c.Client.Get(releasesEndpoint, &releases); err != nil {
				return ErrorResult(fmt.Sprintf("Failed to get releases: %v", err))
			}

			if len(releases) == 0 {
				return ErrorResult("No releases found for this project")
			}

			latestRelease := releases[0]

			// Step 2: Get pipelines for the tag
			pipelinesEndpoint := fmt.Sprintf("/projects/%s/pipelines?ref=%s&per_page=1",
				url.PathEscape(projectID),
				url.PathEscape(latestRelease.TagName))

			var pipelines []gitlab.Pipeline
			if err := c.Client.Get(pipelinesEndpoint, &pipelines); err != nil {
				return ErrorResult(fmt.Sprintf("Failed to get pipelines for tag %s: %v", latestRelease.TagName, err))
			}

			if len(pipelines) == 0 {
				return ErrorResult(fmt.Sprintf("No pipeline found for tag %s", latestRelease.TagName))
			}

			pipeline := pipelines[0]

			// Build result
			result := map[string]interface{}{
				"release": map[string]interface{}{
					"tag_name":   latestRelease.TagName,
					"name":       latestRelease.Name,
					"created_at": latestRelease.CreatedAt,
				},
				"pipeline": pipeline,
			}

			// Step 3: Optionally get jobs
			if includeJobs {
				jobsEndpoint := fmt.Sprintf("/projects/%s/pipelines/%d/jobs",
					url.PathEscape(projectID),
					pipeline.ID)

				var jobs []gitlab.Job
				if err := c.Client.Get(jobsEndpoint, &jobs); err != nil {
					c.Logger.Warn("Failed to get jobs for pipeline %d: %v", pipeline.ID, err)
				} else {
					result["jobs"] = jobs
				}
			}

			return JSONResult(result)
		},
	)
}

// initPipelineTools registers all pipeline-related tools with the MCP server.
// This function is called by RegisterPipelineTools in registry.go when the
// USE_PIPELINE feature flag is enabled.
func initPipelineTools(server *mcp.Server) {
	registerListPipelines(server)
	registerGetPipeline(server)
	registerCreatePipeline(server)
	registerRetryPipeline(server)
	registerCancelPipeline(server)
	registerListPipelineJobs(server)
	registerListPipelineTriggerJobs(server)
	registerGetPipelineJob(server)
	registerGetPipelineJobOutput(server)
	registerPlayPipelineJob(server)
	registerRetryPipelineJob(server)
	registerCancelPipelineJob(server)
	registerGetLatestReleasePipeline(server)
}
