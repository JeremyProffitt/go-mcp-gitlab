# ECS Deployment Guide for go-mcp-gitlab

This guide covers deploying go-mcp-gitlab as an HTTP service on AWS ECS (Elastic Container Service) using either Fargate or EC2 launch types.

## Architecture Overview

```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│   MCP Client    │────▶│  Load Balancer  │────▶│   ECS Service   │
│ (Claude Code/   │     │     (ALB)       │     │   (Fargate/EC2) │
│  Continue.dev)  │     └─────────────────┘     └─────────────────┘
└─────────────────┘              │                       │
                                 │                       ▼
                                 │              ┌─────────────────┐
                                 │              │   GitLab API    │
                                 │              │                 │
                                 │              └─────────────────┘
                                 ▼
                        ┌─────────────────┐
                        │ Secrets Manager │
                        └─────────────────┘
```

## Prerequisites

1. AWS CLI configured with appropriate permissions
2. Docker installed locally for building images
3. An ECR repository created for the image
4. VPC with subnets configured for ECS
5. GitLab Personal Access Token

## Quick Start

### 1. Build and Push Docker Image

```bash
# Authenticate to ECR
aws ecr get-login-password --region YOUR_REGION | docker login --username AWS --password-stdin YOUR_ACCOUNT_ID.dkr.ecr.YOUR_REGION.amazonaws.com

# Build the image
docker build -t go-mcp-gitlab .

# Tag for ECR
docker tag go-mcp-gitlab:latest YOUR_ACCOUNT_ID.dkr.ecr.YOUR_REGION.amazonaws.com/go-mcp-gitlab:latest

# Push to ECR
docker push YOUR_ACCOUNT_ID.dkr.ecr.YOUR_REGION.amazonaws.com/go-mcp-gitlab:latest
```

### 2. Create Secrets in AWS Secrets Manager

```bash
aws secretsmanager create-secret \
    --name mcp/gitlab \
    --secret-string '{
        "GITLAB_PERSONAL_ACCESS_TOKEN": "glpat-xxxxxxxxxxxx",
        "MCP_AUTH_TOKEN": "your-secure-auth-token"
    }'
```

### 3. Create ECS Resources

```bash
# Create CloudWatch Log Group
aws logs create-log-group --log-group-name /ecs/go-mcp-gitlab

# Register Task Definition
aws ecs register-task-definition --cli-input-json file://ecs-task-definition.json

# Create ECS Cluster (if not exists)
aws ecs create-cluster --cluster-name mcp-servers

# Create Service
aws ecs create-service \
    --cluster mcp-servers \
    --service-name go-mcp-gitlab \
    --task-definition go-mcp-gitlab \
    --desired-count 1 \
    --launch-type FARGATE \
    --network-configuration "awsvpcConfiguration={subnets=[subnet-xxx],securityGroups=[sg-xxx],assignPublicIp=ENABLED}"
```

## Configuration

### Environment Variables

| Variable | Required | Description |
|----------|----------|-------------|
| `GITLAB_PERSONAL_ACCESS_TOKEN` | Yes | GitLab personal access token |
| `GITLAB_API_URL` | No | GitLab API URL (default: https://gitlab.com/api/v4) |
| `GITLAB_PROJECT_ID` | No | Default project ID |
| `GITLAB_ALLOWED_PROJECT_IDS` | No | Comma-separated list of allowed project IDs |
| `GITLAB_DEFAULT_NAMESPACE` | No | Default namespace/group for project operations |
| `MCP_AUTH_TOKEN` | No | Token for HTTP authentication |
| `MCP_LOG_LEVEL` | No | Log level (default: info) |
| `USE_PIPELINE` | No | Enable pipeline tools (default: false) |
| `USE_MILESTONE` | No | Enable milestone tools (default: false) |
| `USE_GITLAB_WIKI` | No | Enable wiki tools (default: false) |
| `GITLAB_READ_ONLY_MODE` | No | Enable read-only mode (default: false) |

### GitLab Token Permissions

The GitLab Personal Access Token requires these scopes:
- `api` - Full API access
- `read_user` - Read user information
- `read_repository` - Read repository data
- `write_repository` - Write repository data (if not read-only)

### Authentication

When `MCP_AUTH_TOKEN` is set, all HTTP requests must include the `X-MCP-Auth-Token` header.

```bash
curl -X POST http://your-alb-url:3000/ \
    -H "Content-Type: application/json" \
    -H "X-MCP-Auth-Token: your-secure-auth-token" \
    -d '{"jsonrpc":"2.0","method":"tools/list","id":1}'
```

## Security Considerations

1. **Use HTTPS**: Place an ALB with HTTPS termination in front
2. **Private Subnets**: Deploy in private subnets with NAT Gateway
3. **Minimal Token Scope**: Use only required GitLab token permissions
4. **Read-Only Mode**: Enable `GITLAB_READ_ONLY_MODE` if write access isn't needed
5. **Project Restrictions**: Use `GITLAB_ALLOWED_PROJECT_IDS` to limit access
6. **Authentication**: Enable `MCP_AUTH_TOKEN` for production

## Monitoring

### CloudWatch Logs

Logs are sent to CloudWatch Logs at `/ecs/go-mcp-gitlab`.

### Health Checks

The service exposes a `/health` endpoint that returns:
```json
{"status": "healthy", "server": "go-mcp-gitlab"}
```

## Self-Hosted GitLab

For self-hosted GitLab instances:

1. Set `GITLAB_API_URL` to your GitLab instance API URL:
   ```
   GITLAB_API_URL=https://gitlab.yourcompany.com/api/v4
   ```

2. Ensure the ECS tasks can reach your GitLab instance (VPN, VPC peering, etc.)

3. If using self-signed certificates, you may need to add CA certificates to the container.

## Troubleshooting

### Common Issues

1. **Authentication failed**: Verify GitLab token is valid and has correct scopes
2. **Connection refused to GitLab**: Check outbound internet/network access
3. **Project not found**: Verify project ID and token permissions

See [INTEGRATION.md](./INTEGRATION.md) for configuring Claude Code and Continue.dev.
