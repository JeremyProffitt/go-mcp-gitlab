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

## MCP Server LLM Usability Checklist

**IMPORTANT**: This checklist must be reviewed and all items verified on every update to this repository. Any issues found must be resolved before merging changes.

### Tool Definitions

- [ ] **Clear Purpose**: Each tool has a description that clearly explains what it does and when to use it
- [ ] **No Redundant Platform Names**: Descriptions don't include unnecessary "from [Platform]" text
- [ ] **Parameter Hints**: Tool descriptions mention key parameters or capabilities
- [ ] **Use Case Guidance**: Complex tools include when-to-use hints vs similar tools
- [ ] **Consistent Naming**: All tools use snake_case naming convention
- [ ] **Action Verbs**: Tool names start with action verbs (get_, list_, create_, update_, delete_, search_)

### Parameter Documentation

- [ ] **Examples Provided**: All string parameters include format examples in descriptions
- [ ] **Format Hints**: Date/time, ID, and structured parameters document expected formats
- [ ] **Valid Values Listed**: Parameters with fixed options list valid values (e.g., "Status: 'open', 'closed', 'all'")
- [ ] **No Redundant Defaults**: Default values are in the Default field, not repeated in description text
- [ ] **Array Format Clear**: Array parameters explain expected item format
- [ ] **Object Structure Documented**: Object parameters describe expected properties

### Schema Constraints

- [ ] **Numeric Bounds**: All limit/offset/count parameters have Minimum and Maximum constraints
- [ ] **Integer Types**: Pagination and count parameters use "integer" not "number"
- [ ] **Enum Values**: Categorical parameters have Enum arrays defined in schema
- [ ] **Array Items Typed**: All array parameters have Items property with type defined
- [ ] **Object Properties**: Complex object parameters have Properties defined where structure is known
- [ ] **Pattern Validation**: ID fields have Pattern regex where format is standardized (optional)

### Tool Annotations

- [ ] **Title Set**: All tools have a human-readable Title annotation
- [ ] **ReadOnlyHint**: All get_*, list_*, search_*, describe_* tools have ReadOnlyHint: true
- [ ] **DestructiveHint**: All delete_* tools have DestructiveHint: true
- [ ] **IdempotentHint**: Safe-to-retry operations have IdempotentHint: true
- [ ] **OpenWorldHint**: Tools interacting with external systems have OpenWorldHint: true (optional)

### Token Efficiency

- [ ] **Concise Descriptions**: Tool descriptions are under 200 characters where possible
- [ ] **No Duplicate Info**: Information isn't repeated between tool and parameter descriptions
- [ ] **Abbreviated Common Terms**: Use "Max results" instead of "Maximum number of results to return"
- [ ] **Consistent Parameter Docs**: Common parameters (limit, offset, page) use identical descriptions

### Documentation

- [ ] **README Tool Reference**: README includes descriptions of what each tool does
- [ ] **Workflow Examples**: Common multi-tool workflows are documented
- [ ] **Error Handling Guide**: Common errors and recovery strategies documented
- [ ] **Parameter Patterns**: Common parameter formats (IDs, dates, queries) documented once

### Code Quality

- [ ] **Compiles Successfully**: `go build ./...` completes without errors
- [ ] **Tests Pass**: `go test ./...` completes without failures
- [ ] **No Unused Code**: No commented-out code or unused variables
- [ ] **Consistent Formatting**: Code follows Go formatting standards (`go fmt`)

### Pre-Commit Verification

Before committing changes to this repository, run:

```bash
# Verify compilation
go build ./...

# Run all tests
go test ./...

# Check formatting
go fmt ./...

# Verify tool definitions (manual review)
# Review any new or modified tools against this checklist
```

### Issue Resolution Process

If any checklist item fails:

1. **Document the Issue**: Note which item failed and in which file
2. **Fix the Issue**: Make the necessary code changes
3. **Verify the Fix**: Re-run the relevant checks
4. **Update Tests**: Add tests for new functionality if applicable
5. **Re-verify Checklist**: Ensure fix didn't break other items
