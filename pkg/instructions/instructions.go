// Package instructions provides embedded documentation for the GitLab MCP server.
// These instructions are delivered to LLM clients via the MCP protocol's instructions field.
//
// IMPORTANT: When modifying tools (especially pipeline tools), update the corresponding
// markdown files in the docs/ subdirectory. See CLAUDE.md for maintenance guidelines.
package instructions

import (
	_ "embed"
	"strings"
)

//go:embed docs/base.md
var baseInstructions string

//go:embed docs/pipelines.md
var pipelineInstructions string

//go:embed docs/terraform.md
var terraformInstructions string

// EnabledFeatures represents which feature sets are enabled
type EnabledFeatures struct {
	Pipelines  bool
	Milestones bool
	Wiki       bool
}

// Generate creates the full instructions string based on enabled features.
// This is returned in the MCP initialize response to guide LLM tool usage.
func Generate(features EnabledFeatures) string {
	var parts []string

	// Always include base instructions
	parts = append(parts, strings.TrimSpace(baseInstructions))

	// Add feature-specific instructions
	if features.Pipelines {
		parts = append(parts, strings.TrimSpace(pipelineInstructions))
		parts = append(parts, strings.TrimSpace(terraformInstructions))
	}

	return strings.Join(parts, "\n\n")
}

// GenerateAll returns instructions with all features enabled.
// Useful for documentation generation or when feature flags aren't relevant.
func GenerateAll() string {
	return Generate(EnabledFeatures{
		Pipelines:  true,
		Milestones: true,
		Wiki:       true,
	})
}
