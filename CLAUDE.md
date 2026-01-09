# GitLab MCP Server - Development Guidelines

> **For Development Only**: This file contains instructions for developing and maintaining this MCP server.
> End-user usage instructions are embedded in the server and delivered via the MCP protocol.

## Documentation Architecture

This project uses a **dual documentation approach**:

| Documentation | Location | Purpose | Audience |
|--------------|----------|---------|----------|
| **Embedded Instructions** | `pkg/instructions/docs/*.md` | Delivered via MCP protocol | LLM clients |
| **Standalone Docs** | `docs/llm-usage.md` | Optional manual setup | Users who want CLAUDE.md-style files |
| **Development Guide** | `CLAUDE.md` (this file) | Development & maintenance | Contributors |

### Embedded Instructions (MCP Protocol)

Instructions are embedded in the binary and returned during MCP initialization:

```go
// pkg/instructions/instructions.go
//go:embed docs/base.md
var baseInstructions string
```

These are automatically delivered to LLM clients - no manual setup required.

### How Instructions Flow

```
pkg/instructions/docs/*.md  -->  instructions.Generate()  -->  server.SetInstructions()
                                                                      |
                                                                      v
                                                          MCP InitializeResult.Instructions
                                                                      |
                                                                      v
                                                              LLM Client (Claude, etc.)
```

## Maintenance Requirements

### When Modifying Tools

**CRITICAL**: When adding or modifying MCP tools, you MUST update these files:

1. **`pkg/instructions/docs/base.md`** - Core tool usage patterns
2. **`pkg/instructions/docs/pipelines.md`** - Pipeline tool documentation (if pipeline-related)
3. **`pkg/instructions/docs/terraform.md`** - Terraform extraction docs (if extraction-related)
4. **`docs/llm-usage.md`** - Standalone documentation for manual setup

### Checklist for Tool Changes

- [ ] Tool description is clear and includes parameter documentation
- [ ] `pkg/instructions/docs/*.md` updated with usage examples
- [ ] `docs/llm-usage.md` updated (mirrors embedded docs)
- [ ] Build succeeds: `go build ./...`
- [ ] Instructions are included in initialize response (test manually if needed)

### Adding New Instruction Topics

1. Create a new file: `pkg/instructions/docs/newtopic.md`
2. Add embed directive in `pkg/instructions/instructions.go`:
   ```go
   //go:embed docs/newtopic.md
   var newtopicInstructions string
   ```
3. Add to `Generate()` function with appropriate feature flag check
4. Update `docs/llm-usage.md` with the same content

## Testing Instructions

Verify instructions are correctly embedded:

```bash
# Build and run, then send initialize request
echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}' | ./go-mcp-gitlab

# Check that "instructions" field is populated in response
```

## Code Organization

```
go-mcp-gitlab/
├── CLAUDE.md                          # This file (dev instructions)
├── main.go                            # Entry point, sets instructions
├── pkg/
│   ├── instructions/
│   │   ├── instructions.go            # Instruction generation logic
│   │   └── docs/
│   │       ├── base.md                # Core usage instructions
│   │       ├── pipelines.md           # Pipeline tool docs
│   │       └── terraform.md           # Terraform extraction docs
│   ├── mcp/
│   │   ├── types.go                   # InitializeResult with Instructions field
│   │   └── server.go                  # SetInstructions() method
│   └── tools/
│       └── *.go                       # Tool implementations
└── docs/
    └── llm-usage.md                   # Standalone user docs (optional)
```

## Feature Flags

Instructions are conditionally included based on enabled features:

| Feature | Env Variable | Instruction Files |
|---------|-------------|-------------------|
| Pipelines | `USE_PIPELINE=true` | `pipelines.md`, `terraform.md` |
| Milestones | `USE_MILESTONE=true` | (base only) |
| Wiki | `USE_GITLAB_WIKI=true` | (base only) |

## Quick Reference: Tool Categories

### Core Tools (Always Registered)
- Project tools: `get_project`, `list_projects`, `search_repositories`, etc.
- File tools: `get_file_contents`, `create_or_update_file`, etc.
- Issue tools: `list_issues`, `create_issue`, etc.
- MR tools: `list_merge_requests`, `create_merge_request`, etc.
- Branch tools: `list_branches`, `create_branch`, etc.

### Feature-Flagged Tools
- Pipeline tools (`USE_PIPELINE`): `list_pipelines`, `get_pipeline_job_output`, etc.
- Milestone tools (`USE_MILESTONE`): `list_milestones`, `create_milestone`, etc.
- Wiki tools (`USE_GITLAB_WIKI`): `list_wiki_pages`, `create_wiki_page`, etc.

## Pipeline Tool Deep Dive

The pipeline tools include sophisticated log analysis. Key parameters for `get_pipeline_job_output`:

| Parameter | Type | Description |
|-----------|------|-------------|
| `search` | string | Regex pattern to filter lines |
| `head` | int | Return first N lines |
| `tail` | int | Return last N lines |
| `context_lines` | int | Lines before/after matches |
| `invert_match` | bool | Return non-matching lines |
| `extract` | string | Predefined extractor |
| `format` | string | Output format: "json" or "text" |

### Extract Values

| Value | Description |
|-------|-------------|
| `terraform_outputs` | Parse Terraform output values |
| `terraform_resources` | Parse created/modified resources |
| `terraform_all` | Complete Terraform data |
| `aws_assets` | Extract ARNs, S3 URIs, resource IDs |
| `errors` | Extract error/failure messages |
| `test_results` | Extract test pass/fail results |
