# MCP Client Integration Guide

This guide explains how to configure MCP clients (Claude Code and Continue.dev) to connect to the go-mcp-gitlab server running in HTTP mode, including authentication configuration.

## Authentication Overview

When running in HTTP mode with authentication enabled (via `MCP_AUTH_TOKEN` environment variable), all requests must include the `X-MCP-Auth-Token` header with the configured token value.

## Claude Code Integration

### Configuration Location

Claude Code configuration is stored in:
- **macOS/Linux**: `~/.claude/claude_code_config.json`
- **Windows**: `%USERPROFILE%\.claude\claude_code_config.json`

### HTTP Mode Configuration

```json
{
  "mcpServers": {
    "gitlab": {
      "type": "http",
      "url": "http://your-alb-url:3000",
      "headers": {
        "X-MCP-Auth-Token": "your-secure-auth-token"
      }
    }
  }
}
```

### Configuration with Environment Variable

```json
{
  "mcpServers": {
    "gitlab": {
      "type": "http",
      "url": "http://your-alb-url:3000",
      "headers": {
        "X-MCP-Auth-Token": "${MCP_GITLAB_TOKEN}"
      }
    }
  }
}
```

### Local Development (stdio mode)

```json
{
  "mcpServers": {
    "gitlab": {
      "command": "/path/to/go-mcp-gitlab",
      "args": [],
      "env": {
        "GITLAB_PERSONAL_ACCESS_TOKEN": "glpat-xxxxxxxxxxxx"
      }
    }
  }
}
```

## Continue.dev Integration

### HTTP Mode Configuration

```json
{
  "experimental": {
    "modelContextProtocolServers": [
      {
        "name": "gitlab",
        "transport": {
          "type": "http",
          "url": "http://your-alb-url:3000",
          "headers": {
            "X-MCP-Auth-Token": "your-secure-auth-token"
          }
        }
      }
    ]
  }
}
```

### Local Development (stdio mode)

```json
{
  "experimental": {
    "modelContextProtocolServers": [
      {
        "name": "gitlab",
        "transport": {
          "type": "stdio",
          "command": "/path/to/go-mcp-gitlab",
          "args": []
        },
        "env": {
          "GITLAB_PERSONAL_ACCESS_TOKEN": "glpat-xxxxxxxxxxxx"
        }
      }
    ]
  }
}
```

## Testing the Connection

### Using curl

```bash
# Test health endpoint (no auth required)
curl http://your-alb-url:3000/health

# List available tools
curl -X POST http://your-alb-url:3000/ \
    -H "Content-Type: application/json" \
    -H "X-MCP-Auth-Token: your-secure-auth-token" \
    -d '{"jsonrpc":"2.0","method":"tools/list","id":1}'

# Get project information
curl -X POST http://your-alb-url:3000/ \
    -H "Content-Type: application/json" \
    -H "X-MCP-Auth-Token: your-secure-auth-token" \
    -d '{"jsonrpc":"2.0","method":"tools/call","params":{"name":"get_project","arguments":{"project_id":"123"}},"id":2}'
```

## Available Tools

The GitLab MCP server provides tools for:

| Category | Tools |
|----------|-------|
| Projects | Get project, list projects, create project |
| Issues | List issues, create issue, update issue |
| Merge Requests | List MRs, create MR, merge MR |
| Pipelines | List pipelines, get pipeline (if enabled) |
| Files | Get file, create/update file |
| Branches | List branches, create branch |
| Users | Get current user |

Enable additional tools via environment variables:
- `USE_PIPELINE=true` - Pipeline tools
- `USE_MILESTONE=true` - Milestone tools
- `USE_GITLAB_WIKI=true` - Wiki tools

## Security Best Practices

1. **Use HTTPS**: Always use HTTPS in production
2. **Rotate tokens**: Implement regular token rotation for both GitLab and MCP auth tokens
3. **Minimal GitLab scope**: Use tokens with only required GitLab permissions
4. **Project restrictions**: Configure `GITLAB_ALLOWED_PROJECT_IDS` to limit access
5. **Read-only mode**: Enable `GITLAB_READ_ONLY_MODE` if write access isn't needed

## Troubleshooting

### 401 Unauthorized
- Verify the `X-MCP-Auth-Token` header matches the server's token

### GitLab API Errors
- Check GitLab token is valid and has correct scopes
- Verify project ID exists and is accessible with the token
- Check if read-only mode is blocking write operations

### Connection Timeout
- Verify network connectivity to GitLab API
- Check VPC/firewall rules if using self-hosted GitLab
