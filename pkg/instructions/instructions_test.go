package instructions

import (
	"strings"
	"testing"
)

func TestGenerate_BaseOnly(t *testing.T) {
	result := Generate(EnabledFeatures{})

	if !strings.Contains(result, "GitLab MCP Server") {
		t.Error("Expected base instructions to contain 'GitLab MCP Server'")
	}

	// Should NOT contain pipeline-specific content
	if strings.Contains(result, "get_pipeline_job_output") {
		t.Error("Expected base-only instructions to NOT contain pipeline tools")
	}
}

func TestGenerate_WithPipelines(t *testing.T) {
	result := Generate(EnabledFeatures{Pipelines: true})

	if !strings.Contains(result, "GitLab MCP Server") {
		t.Error("Expected instructions to contain 'GitLab MCP Server'")
	}

	// Should contain pipeline-specific content
	if !strings.Contains(result, "get_pipeline_job_output") {
		t.Error("Expected pipeline-enabled instructions to contain pipeline tools")
	}

	// Should contain terraform extraction content
	if !strings.Contains(result, "terraform_outputs") {
		t.Error("Expected pipeline-enabled instructions to contain terraform extractors")
	}
}

func TestGenerateAll(t *testing.T) {
	result := GenerateAll()

	// Should contain all content
	if !strings.Contains(result, "GitLab MCP Server") {
		t.Error("Expected all instructions to contain base content")
	}

	if !strings.Contains(result, "get_pipeline_job_output") {
		t.Error("Expected all instructions to contain pipeline content")
	}

	if !strings.Contains(result, "terraform_outputs") {
		t.Error("Expected all instructions to contain terraform content")
	}
}

func TestGenerate_NotEmpty(t *testing.T) {
	result := Generate(EnabledFeatures{})

	if len(result) == 0 {
		t.Error("Expected non-empty instructions")
	}

	// Should be reasonably sized (at least a few hundred chars)
	if len(result) < 100 {
		t.Errorf("Expected instructions to be at least 100 chars, got %d", len(result))
	}
}
