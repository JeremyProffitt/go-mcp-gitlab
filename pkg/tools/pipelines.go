package tools

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/go-mcp-gitlab/go-mcp-gitlab/pkg/gitlab"
	"github.com/go-mcp-gitlab/go-mcp-gitlab/pkg/mcp"
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

// registerListPipelines registers the list_pipelines tool.
func registerListPipelines(server *mcp.Server) {
	server.RegisterTool(
		mcp.Tool{
			Name:        "list_pipelines",
			Description: "List pipelines for a project. Returns a paginated list of pipelines with their metadata.",
			InputSchema: mcp.JSONSchema{
				Type: "object",
				Properties: map[string]mcp.Property{
					"project_id": {
						Type:        "string",
						Description: "The ID or URL-encoded path of the project",
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
						Description: "Page number for pagination (default: 1)",
					},
					"per_page": {
						Type:        "integer",
						Description: "Number of items per page (default: 20, max: 100)",
					},
				},
				Required: []string{"project_id"},
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
			Description: "Get details of a specific pipeline by ID.",
			InputSchema: mcp.JSONSchema{
				Type: "object",
				Properties: map[string]mcp.Property{
					"project_id": {
						Type:        "string",
						Description: "The ID or URL-encoded path of the project",
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
						Description: "The ID or URL-encoded path of the project",
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
						Description: "The ID or URL-encoded path of the project",
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
						Description: "The ID or URL-encoded path of the project",
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
						Description: "The ID or URL-encoded path of the project",
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
						Description: "Page number for pagination (default: 1)",
					},
					"per_page": {
						Type:        "integer",
						Description: "Number of items per page (default: 20, max: 100)",
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
						Description: "The ID or URL-encoded path of the project",
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
						Description: "Page number for pagination (default: 1)",
					},
					"per_page": {
						Type:        "integer",
						Description: "Number of items per page (default: 20, max: 100)",
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
						Description: "The ID or URL-encoded path of the project",
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
			Name:        "get_pipeline_job_output",
			Description: "Get the log (trace) output of a specific job. Returns the job log as plain text.",
			InputSchema: mcp.JSONSchema{
				Type: "object",
				Properties: map[string]mcp.Property{
					"project_id": {
						Type:        "string",
						Description: "The ID or URL-encoded path of the project",
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
			c.Logger.ToolCall("get_pipeline_job_output", args)

			projectID := GetString(args, "project_id", "")
			if projectID == "" {
				return ErrorResult("project_id is required")
			}
			jobID := GetInt(args, "job_id", 0)
			if jobID == 0 {
				return ErrorResult("job_id is required")
			}

			endpoint := fmt.Sprintf("/projects/%s/jobs/%d/trace", url.PathEscape(projectID), jobID)

			trace, err := c.Client.GetText(endpoint)
			if err != nil {
				return ErrorResult(fmt.Sprintf("Failed to get job output: %v", err))
			}

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
						Description: "The ID or URL-encoded path of the project",
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
						Description: "The ID or URL-encoded path of the project",
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
						Description: "The ID or URL-encoded path of the project",
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
}
