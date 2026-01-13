# Container Image Specification for Lambda-Compatible MCP Servers

This document provides comprehensive specifications for building Lambda-compatible container images for MCP (Model Context Protocol) servers. It enables engineers and LLMs to build production-ready container images without additional guidance.

---

## Table of Contents

1. [Base Image Selection](#1-base-image-selection)
2. [Lambda Web Adapter Integration](#2-lambda-web-adapter-integration)
3. [Dockerfile Specification](#3-dockerfile-specification)
4. [Per-Server Configuration](#4-per-server-configuration)
5. [Build Process](#5-build-process)
6. [ECR Repository Structure](#6-ecr-repository-structure)
7. [Local Testing](#7-local-testing)
8. [Security Considerations](#8-security-considerations)
9. [Multi-Server Deployment](#9-multi-server-deployment)

---

## 1. Base Image Selection

### Primary Base Image

Use the AWS Lambda base image for custom runtimes:

```
public.ecr.aws/lambda/provided:al2023
```

This image provides:
- Amazon Linux 2023 environment
- Lambda Runtime Interface Client (RIC) pre-installed
- Optimized for Lambda execution environment
- Regular security updates from AWS

### Architecture Selection

#### AMD64 (x86_64)

```dockerfile
FROM --platform=linux/amd64 public.ecr.aws/lambda/provided:al2023
```

**Use when:**
- Maximum compatibility is required
- Using dependencies without ARM builds
- CI/CD pipeline runs on x86 infrastructure

**Build flags:**
```bash
GOOS=linux GOARCH=amd64
```

#### ARM64 (Graviton2/3)

```dockerfile
FROM --platform=linux/arm64 public.ecr.aws/lambda/provided:al2023
```

**Use when:**
- Cost optimization is priority (up to 34% cheaper)
- Better price-performance ratio needed
- All dependencies support ARM

**Build flags:**
```bash
GOOS=linux GOARCH=arm64
```

### Image Size Considerations

| Factor | Recommendation |
|--------|----------------|
| Base image | `provided:al2023` is ~50MB compressed |
| Go binary | Use `-ldflags="-s -w"` to strip debug info (30-50% reduction) |
| Multi-stage builds | Always use to exclude build tools |
| Static linking | Use `CGO_ENABLED=0` for standalone binaries |
| UPX compression | Optional, reduces binary by 50-70% but increases startup time |

**Target image sizes:**
- Base + Lambda Adapter: ~60MB
- With Go MCP server binary: ~70-90MB
- Maximum recommended: <150MB for fast cold starts

---

## 2. Lambda Web Adapter Integration

### Overview

The Lambda Web Adapter (LWA) enables running HTTP-based applications in Lambda without code changes. It acts as a reverse proxy between Lambda's invoke API and your HTTP server.

### Source Image

```
public.ecr.aws/awsguru/aws-lambda-adapter:0.8.4
```

### Installation

Copy the adapter binary to the Lambda extensions directory:

```dockerfile
COPY --from=public.ecr.aws/awsguru/aws-lambda-adapter:0.8.4 /lambda-adapter /opt/extensions/lambda-adapter
```

The adapter is automatically discovered and started by Lambda's extension system.

### Environment Variables

#### AWS_LWA_PORT (Required)

The port your application listens on.

```dockerfile
ENV AWS_LWA_PORT=8080
```

**Notes:**
- Must match the port in your application's startup command
- Common values: 8080, 3000, 9000
- Do not use ports below 1024 (require root)

#### AWS_LWA_READINESS_PATH (Required)

The health check endpoint path.

```dockerfile
ENV AWS_LWA_READINESS_PATH=/health
```

**Notes:**
- Endpoint must return HTTP 200 when ready
- Should be fast (<100ms response time)
- Do not include authentication on this endpoint

#### AWS_LWA_INVOKE_MODE (Optional)

Controls response streaming behavior.

```dockerfile
ENV AWS_LWA_INVOKE_MODE=response_stream
```

**Values:**
- `buffered` (default): Collects entire response before returning
- `response_stream`: Streams response chunks as received

**Use `response_stream` when:**
- Server-Sent Events (SSE) are used
- Long-running operations with progress updates
- Large response payloads

#### AWS_LWA_ASYNC_INIT (Optional)

Enable async initialization for faster cold starts.

```dockerfile
ENV AWS_LWA_ASYNC_INIT=true
```

#### AWS_LWA_REMOVE_BASE_PATH (Optional)

Remove a path prefix before forwarding to the application.

```dockerfile
ENV AWS_LWA_REMOVE_BASE_PATH=/v1
```

### Readiness Check Behavior

The Lambda Web Adapter performs readiness checks as follows:

1. **Poll interval**: Every 10ms
2. **Timeout**: 10 seconds total (configurable via `AWS_LWA_READINESS_TIMEOUT`)
3. **Success criteria**: HTTP 200 response from readiness path
4. **Failure behavior**: Lambda initialization fails if readiness not achieved

**Readiness check sequence:**
```
Lambda Start
    │
    ▼
Start Extension (LWA)
    │
    ▼
Start Application ──────────────────┐
    │                               │
    ▼                               │
Poll /health every 10ms             │
    │                               │
    ├─── 200 OK ───► Ready          │
    │                               │
    └─── Not Ready ─► Retry ────────┘
            │
            └─── Timeout (10s) ───► Init Failed
```

**Example health endpoint implementation (Go):**

```go
http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
    w.WriteHeader(http.StatusOK)
    w.Write([]byte("OK"))
})
```

---

## 3. Dockerfile Specification

### Multi-Stage Build Template

```dockerfile
# ==============================================================================
# Stage 1: Builder
# ==============================================================================
FROM --platform=linux/amd64 golang:1.21-alpine AS builder

# Install build dependencies (if needed)
RUN apk add --no-cache git ca-certificates

# Set working directory
WORKDIR /build

# Download dependencies first (layer caching)
COPY go.mod go.sum ./
RUN go mod download && go mod verify

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build \
    -ldflags="-s -w -X main.version=${VERSION:-dev}" \
    -o server \
    .

# ==============================================================================
# Stage 2: Runtime
# ==============================================================================
FROM --platform=linux/amd64 public.ecr.aws/lambda/provided:al2023

# Copy Lambda Web Adapter
COPY --from=public.ecr.aws/awsguru/aws-lambda-adapter:0.8.4 \
    /lambda-adapter /opt/extensions/lambda-adapter

# Copy the compiled binary
COPY --from=builder /build/server /app/server

# Copy CA certificates (for HTTPS calls)
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Lambda Web Adapter configuration
ENV AWS_LWA_PORT=8080
ENV AWS_LWA_READINESS_PATH=/health

# Entry point
CMD ["/app/server", "--http", "--host", "127.0.0.1", "--port", "8080"]
```

### Go Build Flags Explained

```bash
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o server .
```

| Flag | Purpose |
|------|---------|
| `CGO_ENABLED=0` | Disable CGO for static binary (no glibc dependency) |
| `GOOS=linux` | Target Linux OS |
| `GOARCH=amd64` | Target x86_64 architecture (use `arm64` for Graviton) |
| `-ldflags="-s -w"` | Strip debug symbols and DWARF info |
| `-o server` | Output binary name |

### Complete Dockerfile: go-mcp-gitlab

```dockerfile
# ==============================================================================
# go-mcp-gitlab Lambda Container Image
# ==============================================================================

# Build stage
FROM --platform=linux/amd64 golang:1.21-alpine AS builder

RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /build

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download && go mod verify

# Copy source
COPY . .

# Build with version injection
ARG VERSION=dev
ARG GIT_COMMIT=unknown
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build \
    -ldflags="-s -w \
        -X main.version=${VERSION} \
        -X main.commit=${GIT_COMMIT} \
        -X main.buildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
    -o go-mcp-gitlab \
    ./cmd/go-mcp-gitlab

# Runtime stage
FROM --platform=linux/amd64 public.ecr.aws/lambda/provided:al2023

# Install Lambda Web Adapter
COPY --from=public.ecr.aws/awsguru/aws-lambda-adapter:0.8.4 \
    /lambda-adapter /opt/extensions/lambda-adapter

# Copy binary and certificates
COPY --from=builder /build/go-mcp-gitlab /app/server
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# Lambda Web Adapter configuration
ENV AWS_LWA_PORT=8080
ENV AWS_LWA_READINESS_PATH=/health
ENV AWS_LWA_INVOKE_MODE=response_stream

# Application defaults (override at runtime)
ENV TZ=UTC

# Health check for local testing
HEALTHCHECK --interval=5s --timeout=3s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8080/health || exit 1

# Entry point
CMD ["/app/server", "--http", "--host", "127.0.0.1", "--port", "8080"]
```

### Complete Dockerfile: go-mcp-atlassian

```dockerfile
# ==============================================================================
# go-mcp-atlassian Lambda Container Image
# ==============================================================================

FROM --platform=linux/amd64 golang:1.21-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY . .

ARG VERSION=dev
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build \
    -ldflags="-s -w -X main.version=${VERSION}" \
    -o go-mcp-atlassian \
    ./cmd/go-mcp-atlassian

FROM --platform=linux/amd64 public.ecr.aws/lambda/provided:al2023

COPY --from=public.ecr.aws/awsguru/aws-lambda-adapter:0.8.4 \
    /lambda-adapter /opt/extensions/lambda-adapter

COPY --from=builder /build/go-mcp-atlassian /app/server
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

ENV AWS_LWA_PORT=8080
ENV AWS_LWA_READINESS_PATH=/health
ENV AWS_LWA_INVOKE_MODE=response_stream

CMD ["/app/server", "--http", "--host", "127.0.0.1", "--port", "8080"]
```

### Complete Dockerfile: go-mcp-dynatrace

```dockerfile
# ==============================================================================
# go-mcp-dynatrace Lambda Container Image
# ==============================================================================

FROM --platform=linux/amd64 golang:1.21-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY . .

ARG VERSION=dev
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build \
    -ldflags="-s -w -X main.version=${VERSION}" \
    -o go-mcp-dynatrace \
    ./cmd/go-mcp-dynatrace

FROM --platform=linux/amd64 public.ecr.aws/lambda/provided:al2023

COPY --from=public.ecr.aws/awsguru/aws-lambda-adapter:0.8.4 \
    /lambda-adapter /opt/extensions/lambda-adapter

COPY --from=builder /build/go-mcp-dynatrace /app/server
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

ENV AWS_LWA_PORT=8080
ENV AWS_LWA_READINESS_PATH=/health
ENV AWS_LWA_INVOKE_MODE=response_stream

CMD ["/app/server", "--http", "--host", "127.0.0.1", "--port", "8080"]
```

### Complete Dockerfile: go-mcp-pagerduty

```dockerfile
# ==============================================================================
# go-mcp-pagerduty Lambda Container Image
# ==============================================================================

FROM --platform=linux/amd64 golang:1.21-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY . .

ARG VERSION=dev
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build \
    -ldflags="-s -w -X main.version=${VERSION}" \
    -o pagerduty-mcp \
    ./cmd/pagerduty-mcp

FROM --platform=linux/amd64 public.ecr.aws/lambda/provided:al2023

COPY --from=public.ecr.aws/awsguru/aws-lambda-adapter:0.8.4 \
    /lambda-adapter /opt/extensions/lambda-adapter

COPY --from=builder /build/pagerduty-mcp /app/server
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

ENV AWS_LWA_PORT=8080
ENV AWS_LWA_READINESS_PATH=/health
ENV AWS_LWA_INVOKE_MODE=response_stream

CMD ["/app/server", "--http", "--host", "127.0.0.1", "--port", "8080"]
```

### Complete Dockerfile: go-mcp-servicenow

```dockerfile
# ==============================================================================
# go-mcp-servicenow Lambda Container Image
# ==============================================================================

FROM --platform=linux/amd64 golang:1.21-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY . .

ARG VERSION=dev
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build \
    -ldflags="-s -w -X main.version=${VERSION}" \
    -o go-mcp-servicenow \
    ./cmd/go-mcp-servicenow

FROM --platform=linux/amd64 public.ecr.aws/lambda/provided:al2023

COPY --from=public.ecr.aws/awsguru/aws-lambda-adapter:0.8.4 \
    /lambda-adapter /opt/extensions/lambda-adapter

COPY --from=builder /build/go-mcp-servicenow /app/server
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

ENV AWS_LWA_PORT=8080
ENV AWS_LWA_READINESS_PATH=/health
ENV AWS_LWA_INVOKE_MODE=response_stream

CMD ["/app/server", "--http", "--host", "127.0.0.1", "--port", "8080"]
```

### ARM64 Variant Template

For ARM64/Graviton deployments, modify the Dockerfile:

```dockerfile
# Build stage - ARM64
FROM --platform=linux/arm64 golang:1.21-alpine AS builder

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY . .

# Note: GOARCH=arm64
RUN CGO_ENABLED=0 GOOS=linux GOARCH=arm64 \
    go build \
    -ldflags="-s -w" \
    -o server \
    .

# Runtime stage - ARM64
FROM --platform=linux/arm64 public.ecr.aws/lambda/provided:al2023

COPY --from=public.ecr.aws/awsguru/aws-lambda-adapter:0.8.4 \
    /lambda-adapter /opt/extensions/lambda-adapter

COPY --from=builder /build/server /app/server

ENV AWS_LWA_PORT=8080
ENV AWS_LWA_READINESS_PATH=/health

CMD ["/app/server", "--http", "--host", "127.0.0.1", "--port", "8080"]
```

---

## 4. Per-Server Configuration

### Server Configuration Table

| Server | Binary Name | Default Port | Source Path | Required Environment Variables |
|--------|-------------|--------------|-------------|-------------------------------|
| go-mcp-gitlab | go-mcp-gitlab | 8080 | ./cmd/go-mcp-gitlab | `GITLAB_API_URL`, `GITLAB_TOKEN` |
| go-mcp-atlassian | go-mcp-atlassian | 8080 | ./cmd/go-mcp-atlassian | `JIRA_URL`, `JIRA_TOKEN`, `CONFLUENCE_URL`, `CONFLUENCE_TOKEN` |
| go-mcp-dynatrace | go-mcp-dynatrace | 8080 | ./cmd/go-mcp-dynatrace | `DT_ENVIRONMENT`, `DT_API_TOKEN` |
| go-mcp-pagerduty | pagerduty-mcp | 8080 | ./cmd/pagerduty-mcp | `PAGERDUTY_API_KEY`, `PAGERDUTY_API_HOST` |
| go-mcp-servicenow | go-mcp-servicenow | 8080 | ./cmd/go-mcp-servicenow | `SERVICENOW_INSTANCE_URL`, `SERVICENOW_USERNAME`, `SERVICENOW_PASSWORD` |

### Detailed Environment Variable Reference

#### go-mcp-gitlab

| Variable | Required | Description | Example |
|----------|----------|-------------|---------|
| `GITLAB_API_URL` | Yes | GitLab API base URL | `https://gitlab.com/api/v4` |
| `GITLAB_TOKEN` | Yes | Personal access token or OAuth token | `glpat-xxxxxxxxxxxx` |
| `GITLAB_INSECURE` | No | Skip TLS verification (not recommended) | `false` |

#### go-mcp-atlassian

| Variable | Required | Description | Example |
|----------|----------|-------------|---------|
| `JIRA_URL` | Yes | Jira instance URL | `https://mycompany.atlassian.net` |
| `JIRA_TOKEN` | Yes | Jira API token | `ATATT3xFfGF0...` |
| `JIRA_EMAIL` | Yes | Email for authentication | `user@company.com` |
| `CONFLUENCE_URL` | Yes | Confluence instance URL | `https://mycompany.atlassian.net/wiki` |
| `CONFLUENCE_TOKEN` | Yes | Confluence API token | `ATATT3xFfGF0...` |
| `CONFLUENCE_EMAIL` | Yes | Email for authentication | `user@company.com` |

#### go-mcp-dynatrace

| Variable | Required | Description | Example |
|----------|----------|-------------|---------|
| `DT_ENVIRONMENT` | Yes | Dynatrace environment ID | `abc12345` |
| `DT_API_TOKEN` | Yes | Dynatrace API token | `dt0c01.XXXXXX...` |
| `DT_CLUSTER_URL` | No | Custom cluster URL | `https://abc12345.live.dynatrace.com` |

#### go-mcp-pagerduty

| Variable | Required | Description | Example |
|----------|----------|-------------|---------|
| `PAGERDUTY_API_KEY` | Yes | PagerDuty API key | `u+abcdefghijklmnop` |
| `PAGERDUTY_API_HOST` | No | API host (default: api.pagerduty.com) | `api.pagerduty.com` |

#### go-mcp-servicenow

| Variable | Required | Description | Example |
|----------|----------|-------------|---------|
| `SERVICENOW_INSTANCE_URL` | Yes | ServiceNow instance URL | `https://mycompany.service-now.com` |
| `SERVICENOW_USERNAME` | Yes | ServiceNow username | `api_user` |
| `SERVICENOW_PASSWORD` | Yes | ServiceNow password | `********` |
| `SERVICENOW_CLIENT_ID` | No | OAuth client ID | `abc123...` |
| `SERVICENOW_CLIENT_SECRET` | No | OAuth client secret | `xyz789...` |

---

## 5. Build Process

### Standard Build Commands

#### Single Server Build

```bash
# Set variables
export SERVER_NAME=go-mcp-gitlab
export VERSION=1.0.0
export GIT_COMMIT=$(git rev-parse --short HEAD)

# Build the image
docker build \
    --platform linux/amd64 \
    --build-arg VERSION=${VERSION} \
    --build-arg GIT_COMMIT=${GIT_COMMIT} \
    -t mcp-servers/${SERVER_NAME}:${VERSION} \
    -t mcp-servers/${SERVER_NAME}:latest \
    -f Dockerfile \
    .
```

#### Build All Servers Script

```bash
#!/bin/bash
# build-all.sh - Build all MCP server images

set -euo pipefail

VERSION=${1:-"dev"}
GIT_COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
REGISTRY=${REGISTRY:-""}
PLATFORM=${PLATFORM:-"linux/amd64"}

SERVERS=(
    "go-mcp-gitlab"
    "go-mcp-atlassian"
    "go-mcp-dynatrace"
    "go-mcp-pagerduty"
    "go-mcp-servicenow"
)

for server in "${SERVERS[@]}"; do
    echo "=========================================="
    echo "Building ${server}..."
    echo "=========================================="

    IMAGE_NAME="${REGISTRY}mcp-servers/${server}"

    docker build \
        --platform "${PLATFORM}" \
        --build-arg VERSION="${VERSION}" \
        --build-arg GIT_COMMIT="${GIT_COMMIT}" \
        -t "${IMAGE_NAME}:${VERSION}" \
        -t "${IMAGE_NAME}:latest" \
        -t "${IMAGE_NAME}:${GIT_COMMIT}" \
        -f "${server}/Dockerfile" \
        "${server}/"

    echo "Built: ${IMAGE_NAME}:${VERSION}"
done

echo "=========================================="
echo "All builds complete!"
echo "=========================================="
```

### Version Injection via ldflags

```bash
# Full ldflags example
go build -ldflags="\
    -s -w \
    -X main.version=${VERSION} \
    -X main.commit=${GIT_COMMIT} \
    -X main.buildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ) \
    -X main.goVersion=$(go version | cut -d' ' -f3)" \
    -o server .
```

**In your Go code:**

```go
package main

var (
    version   = "dev"
    commit    = "unknown"
    buildTime = "unknown"
    goVersion = "unknown"
)

func main() {
    // Use version info
    log.Printf("Starting server version=%s commit=%s built=%s go=%s",
        version, commit, buildTime, goVersion)
}
```

### Binary Size Optimization

#### Standard Stripping (-s -w flags)

```bash
# Before: ~15MB
go build -o server .

# After: ~10MB (30-40% reduction)
go build -ldflags="-s -w" -o server .
```

#### Additional UPX Compression (Optional)

```dockerfile
# In builder stage
RUN apk add --no-cache upx

# After go build
RUN upx --best --lzma server
# Result: ~4MB (60-70% reduction from stripped binary)
```

**Trade-offs:**
- Slower startup (decompression overhead)
- May trigger security scanners
- Not recommended for Lambda (startup time is critical)

### Cross-Platform Builds

```bash
# Build for both architectures
docker buildx build \
    --platform linux/amd64,linux/arm64 \
    --build-arg VERSION=${VERSION} \
    -t mcp-servers/go-mcp-gitlab:${VERSION} \
    --push \
    .
```

---

## 6. ECR Repository Structure

### Repository Naming Convention

```
{account-id}.dkr.ecr.{region}.amazonaws.com/mcp-servers/{server-name}
```

**Examples:**
```
123456789012.dkr.ecr.us-east-1.amazonaws.com/mcp-servers/go-mcp-gitlab
123456789012.dkr.ecr.us-east-1.amazonaws.com/mcp-servers/go-mcp-atlassian
123456789012.dkr.ecr.us-east-1.amazonaws.com/mcp-servers/go-mcp-dynatrace
123456789012.dkr.ecr.us-east-1.amazonaws.com/mcp-servers/go-mcp-pagerduty
123456789012.dkr.ecr.us-east-1.amazonaws.com/mcp-servers/go-mcp-servicenow
```

### Tag Strategy

| Tag Type | Format | Example | Use Case |
|----------|--------|---------|----------|
| Latest | `latest` | `latest` | Development, quick testing |
| Semantic Version | `v{major}.{minor}.{patch}` | `v1.2.3` | Production releases |
| Git SHA | `{short-sha}` | `a1b2c3d` | CI/CD, traceability |
| Branch | `{branch-name}` | `main`, `develop` | Feature testing |
| Date | `{YYYYMMDD}` | `20240115` | Daily builds |

### Create ECR Repositories

```bash
#!/bin/bash
# create-ecr-repos.sh

REGION=${AWS_REGION:-us-east-1}

SERVERS=(
    "go-mcp-gitlab"
    "go-mcp-atlassian"
    "go-mcp-dynatrace"
    "go-mcp-pagerduty"
    "go-mcp-servicenow"
)

for server in "${SERVERS[@]}"; do
    echo "Creating repository: mcp-servers/${server}"

    aws ecr create-repository \
        --repository-name "mcp-servers/${server}" \
        --region "${REGION}" \
        --image-scanning-configuration scanOnPush=true \
        --encryption-configuration encryptionType=AES256 \
        --tags Key=Project,Value=MCP Key=Server,Value="${server}"
done
```

### Lifecycle Policy

Apply to each repository to manage image retention:

```json
{
    "rules": [
        {
            "rulePriority": 1,
            "description": "Keep last 10 tagged images",
            "selection": {
                "tagStatus": "tagged",
                "tagPrefixList": ["v"],
                "countType": "imageCountMoreThan",
                "countNumber": 10
            },
            "action": {
                "type": "expire"
            }
        },
        {
            "rulePriority": 2,
            "description": "Keep last 5 latest tags",
            "selection": {
                "tagStatus": "tagged",
                "tagPrefixList": ["latest"],
                "countType": "imageCountMoreThan",
                "countNumber": 5
            },
            "action": {
                "type": "expire"
            }
        },
        {
            "rulePriority": 3,
            "description": "Expire untagged images older than 7 days",
            "selection": {
                "tagStatus": "untagged",
                "countType": "sinceImagePushed",
                "countUnit": "days",
                "countNumber": 7
            },
            "action": {
                "type": "expire"
            }
        },
        {
            "rulePriority": 4,
            "description": "Keep last 20 SHA-tagged images",
            "selection": {
                "tagStatus": "any",
                "countType": "imageCountMoreThan",
                "countNumber": 20
            },
            "action": {
                "type": "expire"
            }
        }
    ]
}
```

**Apply lifecycle policy:**

```bash
aws ecr put-lifecycle-policy \
    --repository-name "mcp-servers/go-mcp-gitlab" \
    --lifecycle-policy-text file://lifecycle-policy.json
```

### Push Images to ECR

```bash
#!/bin/bash
# push-to-ecr.sh

ACCOUNT_ID=$(aws sts get-caller-identity --query Account --output text)
REGION=${AWS_REGION:-us-east-1}
REGISTRY="${ACCOUNT_ID}.dkr.ecr.${REGION}.amazonaws.com"
SERVER_NAME=$1
VERSION=$2

# Authenticate with ECR
aws ecr get-login-password --region "${REGION}" | \
    docker login --username AWS --password-stdin "${REGISTRY}"

# Tag images
docker tag "mcp-servers/${SERVER_NAME}:${VERSION}" \
    "${REGISTRY}/mcp-servers/${SERVER_NAME}:${VERSION}"

docker tag "mcp-servers/${SERVER_NAME}:${VERSION}" \
    "${REGISTRY}/mcp-servers/${SERVER_NAME}:latest"

# Push images
docker push "${REGISTRY}/mcp-servers/${SERVER_NAME}:${VERSION}"
docker push "${REGISTRY}/mcp-servers/${SERVER_NAME}:latest"

echo "Pushed: ${REGISTRY}/mcp-servers/${SERVER_NAME}:${VERSION}"
```

---

## 7. Local Testing

### Lambda Runtime Interface Emulator (RIE)

The RIE allows testing Lambda container images locally.

#### Install RIE

```bash
# Download RIE
mkdir -p ~/.aws-lambda-rie
curl -Lo ~/.aws-lambda-rie/aws-lambda-rie \
    https://github.com/aws/aws-lambda-runtime-interface-emulator/releases/latest/download/aws-lambda-rie

chmod +x ~/.aws-lambda-rie/aws-lambda-rie
```

#### Run with RIE

```bash
# Run container with RIE
docker run -d \
    --name mcp-gitlab-test \
    -p 9000:8080 \
    -v ~/.aws-lambda-rie:/aws-lambda \
    -e GITLAB_API_URL="https://gitlab.com/api/v4" \
    -e GITLAB_TOKEN="your-token" \
    --entrypoint /aws-lambda/aws-lambda-rie \
    mcp-servers/go-mcp-gitlab:latest \
    /app/server --http --host 127.0.0.1 --port 8080
```

#### Invoke Function

```bash
# Invoke the Lambda function
curl -XPOST "http://localhost:9000/2015-03-31/functions/function/invocations" \
    -d '{
        "httpMethod": "POST",
        "path": "/mcp",
        "headers": {
            "Content-Type": "application/json"
        },
        "body": "{\"jsonrpc\":\"2.0\",\"method\":\"tools/list\",\"id\":1}"
    }'
```

### Docker Compose for Local Development

```yaml
# docker-compose.yml
version: '3.8'

services:
  go-mcp-gitlab:
    build:
      context: ./go-mcp-gitlab
      dockerfile: Dockerfile
    ports:
      - "8081:8080"
    environment:
      - GITLAB_API_URL=https://gitlab.com/api/v4
      - GITLAB_TOKEN=${GITLAB_TOKEN}
      - AWS_LWA_PORT=8080
      - AWS_LWA_READINESS_PATH=/health
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080/health"]
      interval: 10s
      timeout: 5s
      retries: 3

  go-mcp-atlassian:
    build:
      context: ./go-mcp-atlassian
      dockerfile: Dockerfile
    ports:
      - "8082:8080"
    environment:
      - JIRA_URL=${JIRA_URL}
      - JIRA_TOKEN=${JIRA_TOKEN}
      - JIRA_EMAIL=${JIRA_EMAIL}
      - CONFLUENCE_URL=${CONFLUENCE_URL}
      - CONFLUENCE_TOKEN=${CONFLUENCE_TOKEN}
      - CONFLUENCE_EMAIL=${CONFLUENCE_EMAIL}
      - AWS_LWA_PORT=8080
      - AWS_LWA_READINESS_PATH=/health
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080/health"]
      interval: 10s
      timeout: 5s
      retries: 3

  go-mcp-dynatrace:
    build:
      context: ./go-mcp-dynatrace
      dockerfile: Dockerfile
    ports:
      - "8083:8080"
    environment:
      - DT_ENVIRONMENT=${DT_ENVIRONMENT}
      - DT_API_TOKEN=${DT_API_TOKEN}
      - AWS_LWA_PORT=8080
      - AWS_LWA_READINESS_PATH=/health
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080/health"]
      interval: 10s
      timeout: 5s
      retries: 3

  go-mcp-pagerduty:
    build:
      context: ./go-mcp-pagerduty
      dockerfile: Dockerfile
    ports:
      - "8084:8080"
    environment:
      - PAGERDUTY_API_KEY=${PAGERDUTY_API_KEY}
      - PAGERDUTY_API_HOST=api.pagerduty.com
      - AWS_LWA_PORT=8080
      - AWS_LWA_READINESS_PATH=/health
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080/health"]
      interval: 10s
      timeout: 5s
      retries: 3

  go-mcp-servicenow:
    build:
      context: ./go-mcp-servicenow
      dockerfile: Dockerfile
    ports:
      - "8085:8080"
    environment:
      - SERVICENOW_INSTANCE_URL=${SERVICENOW_INSTANCE_URL}
      - SERVICENOW_USERNAME=${SERVICENOW_USERNAME}
      - SERVICENOW_PASSWORD=${SERVICENOW_PASSWORD}
      - AWS_LWA_PORT=8080
      - AWS_LWA_READINESS_PATH=/health
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080/health"]
      interval: 10s
      timeout: 5s
      retries: 3
```

**Start all services:**

```bash
docker-compose up -d
```

### Sample curl Commands

#### Health Check

```bash
# Check health endpoint
curl -s http://localhost:8081/health
# Expected: OK or {"status": "healthy"}
```

#### MCP Tools List

```bash
# List available tools (JSON-RPC)
curl -s -X POST http://localhost:8081/mcp \
    -H "Content-Type: application/json" \
    -d '{
        "jsonrpc": "2.0",
        "method": "tools/list",
        "id": 1
    }' | jq .
```

#### MCP Tool Call

```bash
# Call a specific tool
curl -s -X POST http://localhost:8081/mcp \
    -H "Content-Type: application/json" \
    -d '{
        "jsonrpc": "2.0",
        "method": "tools/call",
        "params": {
            "name": "list_projects",
            "arguments": {
                "per_page": 10
            }
        },
        "id": 2
    }' | jq .
```

#### SSE Streaming Test

```bash
# Test Server-Sent Events (SSE)
curl -N -X POST http://localhost:8081/mcp/sse \
    -H "Content-Type: application/json" \
    -H "Accept: text/event-stream" \
    -d '{
        "jsonrpc": "2.0",
        "method": "tools/call",
        "params": {
            "name": "search_issues",
            "arguments": {
                "query": "bug"
            }
        },
        "id": 3
    }'
```

#### Test Script

```bash
#!/bin/bash
# test-mcp-server.sh

SERVER_URL=${1:-"http://localhost:8081"}

echo "Testing MCP server at ${SERVER_URL}"
echo "========================================"

# Health check
echo "1. Health check..."
HEALTH=$(curl -s -o /dev/null -w "%{http_code}" "${SERVER_URL}/health")
if [ "$HEALTH" == "200" ]; then
    echo "   [PASS] Health check returned 200"
else
    echo "   [FAIL] Health check returned ${HEALTH}"
    exit 1
fi

# List tools
echo "2. List tools..."
TOOLS=$(curl -s -X POST "${SERVER_URL}/mcp" \
    -H "Content-Type: application/json" \
    -d '{"jsonrpc":"2.0","method":"tools/list","id":1}')

TOOL_COUNT=$(echo "$TOOLS" | jq -r '.result.tools | length')
if [ "$TOOL_COUNT" -gt 0 ]; then
    echo "   [PASS] Found ${TOOL_COUNT} tools"
else
    echo "   [FAIL] No tools found"
    exit 1
fi

# List first 5 tools
echo "3. Available tools:"
echo "$TOOLS" | jq -r '.result.tools[:5][].name' | while read tool; do
    echo "   - ${tool}"
done

echo "========================================"
echo "All tests passed!"
```

---

## 8. Security Considerations

### Non-Root User Execution

While the Lambda base image runs as root by default, you can create a non-root user for additional security:

```dockerfile
FROM public.ecr.aws/lambda/provided:al2023

# Create non-root user
RUN dnf install -y shadow-utils && \
    useradd -r -s /bin/false appuser && \
    dnf clean all

# Copy files with proper ownership
COPY --from=builder --chown=appuser:appuser /build/server /app/server

# Note: Lambda requires specific directories to be writable
# This approach works for the application code but Lambda runtime needs root

# Alternative: Use read-only filesystem instead
```

**Note:** Lambda's execution environment has specific requirements. The recommended approach is using read-only filesystem settings in Lambda configuration rather than changing users.

### Read-Only Filesystem

Configure in Lambda function settings (not Dockerfile):

```json
{
    "EphemeralStorage": {
        "Size": 512
    },
    "FileSystemConfigs": []
}
```

In Terraform/CloudFormation:

```hcl
resource "aws_lambda_function" "mcp_server" {
  # ... other config ...

  file_system_config {
    # Only mount specific directories if needed
  }

  ephemeral_storage {
    size = 512  # MB, for /tmp only
  }
}
```

### Never Bake Secrets Into Images

**DO NOT do this:**

```dockerfile
# WRONG - Never do this!
ENV GITLAB_TOKEN=glpat-xxxxxxxxxxxx
ENV API_KEY=secret123
```

**Correct approach - use runtime environment variables:**

```bash
# Pass secrets at runtime via Lambda configuration
aws lambda update-function-configuration \
    --function-name go-mcp-gitlab \
    --environment "Variables={GITLAB_TOKEN=glpat-xxxxxxxxxxxx}"
```

### Secrets Management Best Practices

#### AWS Secrets Manager Integration

```go
package main

import (
    "context"
    "encoding/json"

    "github.com/aws/aws-sdk-go-v2/config"
    "github.com/aws/aws-sdk-go-v2/service/secretsmanager"
)

type Secrets struct {
    GitLabToken string `json:"gitlab_token"`
    APIKey      string `json:"api_key"`
}

func getSecrets(ctx context.Context, secretName string) (*Secrets, error) {
    cfg, err := config.LoadDefaultConfig(ctx)
    if err != nil {
        return nil, err
    }

    client := secretsmanager.NewFromConfig(cfg)

    result, err := client.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{
        SecretId: &secretName,
    })
    if err != nil {
        return nil, err
    }

    var secrets Secrets
    if err := json.Unmarshal([]byte(*result.SecretString), &secrets); err != nil {
        return nil, err
    }

    return &secrets, nil
}
```

#### Environment Variables at Runtime

Lambda configuration (Terraform):

```hcl
resource "aws_lambda_function" "mcp_gitlab" {
  function_name = "go-mcp-gitlab"
  package_type  = "Image"
  image_uri     = "${aws_ecr_repository.mcp_gitlab.repository_url}:latest"

  environment {
    variables = {
      GITLAB_API_URL = "https://gitlab.com/api/v4"
      # Reference to Secrets Manager
      SECRETS_ARN    = aws_secretsmanager_secret.mcp_secrets.arn
    }
  }

  # IAM role with Secrets Manager access
  role = aws_iam_role.lambda_execution.arn
}
```

### Image Scanning

Enable ECR image scanning:

```bash
# Enable scan on push
aws ecr put-image-scanning-configuration \
    --repository-name mcp-servers/go-mcp-gitlab \
    --image-scanning-configuration scanOnPush=true
```

Check scan results:

```bash
aws ecr describe-image-scan-findings \
    --repository-name mcp-servers/go-mcp-gitlab \
    --image-id imageTag=latest
```

### Minimal Attack Surface

```dockerfile
# Use minimal base image
FROM public.ecr.aws/lambda/provided:al2023

# Don't install unnecessary packages
# Don't copy source code to runtime image
# Only copy the compiled binary and certificates

COPY --from=builder /build/server /app/server
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
```

### Security Checklist

| Check | Status | Notes |
|-------|--------|-------|
| No secrets in Dockerfile | Required | Use runtime env vars |
| No secrets in image layers | Required | Verify with `docker history` |
| Image scanning enabled | Required | ECR scan on push |
| Multi-stage build | Required | Don't include build tools |
| Static binary | Required | `CGO_ENABLED=0` |
| CA certificates only | Required | No extra certs |
| Latest base image | Recommended | Regular updates |
| Minimal packages | Recommended | No unnecessary tools |
| Read-only filesystem | Recommended | Configure in Lambda |
| VPC configuration | Optional | If accessing private resources |

### Verify No Secrets in Image

```bash
# Check image history for potential secrets
docker history mcp-servers/go-mcp-gitlab:latest --no-trunc

# Inspect image layers
docker inspect mcp-servers/go-mcp-gitlab:latest

# Extract and inspect filesystem
docker save mcp-servers/go-mcp-gitlab:latest | tar -xf - -C /tmp/image-inspect
# Examine layer contents

# Use Trivy for security scanning
trivy image mcp-servers/go-mcp-gitlab:latest
```

---

## 9. Multi-Server Deployment

Running multiple MCP servers in a single container reduces operational overhead and infrastructure costs. This section covers two approaches: multi-port containers for ECS/local deployment and router-based Lambda deployments.

### Multi-Server Container (ECS/Local)

Run multiple MCP servers on different ports within a single container.

#### Multi-Server Dockerfile

```dockerfile
# ==============================================================================
# Multi-Server MCP Container Image
# ==============================================================================

# Build stage - build all servers
FROM --platform=linux/amd64 golang:1.21-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /build

# Build go-mcp-gitlab
COPY go-mcp-gitlab/ ./go-mcp-gitlab/
RUN cd go-mcp-gitlab && \
    go mod download && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w" -o /bin/go-mcp-gitlab ./cmd/go-mcp-gitlab

# Build go-mcp-atlassian
COPY go-mcp-atlassian/ ./go-mcp-atlassian/
RUN cd go-mcp-atlassian && \
    go mod download && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w" -o /bin/go-mcp-atlassian ./cmd/go-mcp-atlassian

# Build go-mcp-dynatrace
COPY go-mcp-dynatrace/ ./go-mcp-dynatrace/
RUN cd go-mcp-dynatrace && \
    go mod download && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w" -o /bin/go-mcp-dynatrace ./cmd/go-mcp-dynatrace

# Build go-mcp-pagerduty
COPY go-mcp-pagerduty/ ./go-mcp-pagerduty/
RUN cd go-mcp-pagerduty && \
    go mod download && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w" -o /bin/go-mcp-pagerduty ./cmd/pagerduty-mcp

# Build go-mcp-servicenow
COPY go-mcp-servicenow/ ./go-mcp-servicenow/
RUN cd go-mcp-servicenow && \
    go mod download && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w" -o /bin/go-mcp-servicenow ./cmd/go-mcp-servicenow

# Runtime stage
FROM --platform=linux/amd64 alpine:3.19

RUN apk add --no-cache ca-certificates supervisor

# Copy all server binaries
COPY --from=builder /bin/go-mcp-gitlab /app/go-mcp-gitlab
COPY --from=builder /bin/go-mcp-atlassian /app/go-mcp-atlassian
COPY --from=builder /bin/go-mcp-dynatrace /app/go-mcp-dynatrace
COPY --from=builder /bin/go-mcp-pagerduty /app/go-mcp-pagerduty
COPY --from=builder /bin/go-mcp-servicenow /app/go-mcp-servicenow

# Copy supervisor configuration
COPY supervisord.conf /etc/supervisord.conf

# Expose all server ports
EXPOSE 8081 8082 8083 8084 8085

# Health check endpoint (hits all servers)
HEALTHCHECK --interval=10s --timeout=5s --start-period=10s --retries=3 \
    CMD wget -q --spider http://localhost:8081/health && \
        wget -q --spider http://localhost:8082/health && \
        wget -q --spider http://localhost:8083/health && \
        wget -q --spider http://localhost:8084/health && \
        wget -q --spider http://localhost:8085/health

CMD ["/usr/bin/supervisord", "-c", "/etc/supervisord.conf"]
```

#### Supervisor Configuration

Create `supervisord.conf` to manage multiple server processes:

```ini
[supervisord]
nodaemon=true
user=root
logfile=/var/log/supervisord.log
pidfile=/var/run/supervisord.pid

[program:gitlab]
command=/app/go-mcp-gitlab --http --host 0.0.0.0 --port 8081
autostart=true
autorestart=true
stdout_logfile=/dev/stdout
stdout_logfile_maxbytes=0
stderr_logfile=/dev/stderr
stderr_logfile_maxbytes=0

[program:atlassian]
command=/app/go-mcp-atlassian --http --host 0.0.0.0 --port 8082
autostart=true
autorestart=true
stdout_logfile=/dev/stdout
stdout_logfile_maxbytes=0
stderr_logfile=/dev/stderr
stderr_logfile_maxbytes=0

[program:dynatrace]
command=/app/go-mcp-dynatrace --http --host 0.0.0.0 --port 8083
autostart=true
autorestart=true
stdout_logfile=/dev/stdout
stdout_logfile_maxbytes=0
stderr_logfile=/dev/stderr
stderr_logfile_maxbytes=0

[program:pagerduty]
command=/app/go-mcp-pagerduty --http --host 0.0.0.0 --port 8084
autostart=true
autorestart=true
stdout_logfile=/dev/stdout
stdout_logfile_maxbytes=0
stderr_logfile=/dev/stderr
stderr_logfile_maxbytes=0

[program:servicenow]
command=/app/go-mcp-servicenow --http --host 0.0.0.0 --port 8085
autostart=true
autorestart=true
stdout_logfile=/dev/stdout
stdout_logfile_maxbytes=0
stderr_logfile=/dev/stderr
stderr_logfile_maxbytes=0
```

### Port Allocation

| Server | Port | Health Endpoint |
|--------|------|-----------------|
| go-mcp-gitlab | 8081 | `http://localhost:8081/health` |
| go-mcp-atlassian | 8082 | `http://localhost:8082/health` |
| go-mcp-dynatrace | 8083 | `http://localhost:8083/health` |
| go-mcp-pagerduty | 8084 | `http://localhost:8084/health` |
| go-mcp-servicenow | 8085 | `http://localhost:8085/health` |

### Multi-Server Lambda with Router

For Lambda deployments, use a path-based router that forwards requests to the appropriate server.

#### Router Dockerfile (Lambda)

```dockerfile
# ==============================================================================
# Multi-Server MCP Lambda Container with Router
# ==============================================================================

FROM --platform=linux/amd64 golang:1.21-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /build

# Build all servers (same as multi-server container above)
COPY go-mcp-gitlab/ ./go-mcp-gitlab/
RUN cd go-mcp-gitlab && go mod download && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" \
    -o /bin/go-mcp-gitlab ./cmd/go-mcp-gitlab

COPY go-mcp-atlassian/ ./go-mcp-atlassian/
RUN cd go-mcp-atlassian && go mod download && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" \
    -o /bin/go-mcp-atlassian ./cmd/go-mcp-atlassian

COPY go-mcp-dynatrace/ ./go-mcp-dynatrace/
RUN cd go-mcp-dynatrace && go mod download && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" \
    -o /bin/go-mcp-dynatrace ./cmd/go-mcp-dynatrace

COPY go-mcp-pagerduty/ ./go-mcp-pagerduty/
RUN cd go-mcp-pagerduty && go mod download && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" \
    -o /bin/go-mcp-pagerduty ./cmd/pagerduty-mcp

COPY go-mcp-servicenow/ ./go-mcp-servicenow/
RUN cd go-mcp-servicenow && go mod download && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" \
    -o /bin/go-mcp-servicenow ./cmd/go-mcp-servicenow

# Build router
COPY router/ ./router/
RUN cd router && go mod download && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" \
    -o /bin/mcp-router .

# Runtime stage
FROM --platform=linux/amd64 public.ecr.aws/lambda/provided:al2023

# Copy Lambda Web Adapter
COPY --from=public.ecr.aws/awsguru/aws-lambda-adapter:0.8.4 \
    /lambda-adapter /opt/extensions/lambda-adapter

# Copy all binaries
COPY --from=builder /bin/go-mcp-gitlab /app/go-mcp-gitlab
COPY --from=builder /bin/go-mcp-atlassian /app/go-mcp-atlassian
COPY --from=builder /bin/go-mcp-dynatrace /app/go-mcp-dynatrace
COPY --from=builder /bin/go-mcp-pagerduty /app/go-mcp-pagerduty
COPY --from=builder /bin/go-mcp-servicenow /app/go-mcp-servicenow
COPY --from=builder /bin/mcp-router /app/mcp-router
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Lambda Web Adapter configuration
ENV AWS_LWA_PORT=8080
ENV AWS_LWA_READINESS_PATH=/health
ENV AWS_LWA_INVOKE_MODE=response_stream

# Start the router (which spawns and manages all servers)
CMD ["/app/mcp-router"]
```

#### Router Implementation (Go)

Create `router/main.go`:

```go
package main

import (
    "fmt"
    "io"
    "log"
    "net/http"
    "net/http/httputil"
    "net/url"
    "os"
    "os/exec"
    "sync"
    "time"
)

type Server struct {
    Name    string
    Binary  string
    Port    int
    Process *exec.Cmd
}

var servers = []Server{
    {Name: "gitlab", Binary: "/app/go-mcp-gitlab", Port: 8081},
    {Name: "atlassian", Binary: "/app/go-mcp-atlassian", Port: 8082},
    {Name: "dynatrace", Binary: "/app/go-mcp-dynatrace", Port: 8083},
    {Name: "pagerduty", Binary: "/app/go-mcp-pagerduty", Port: 8084},
    {Name: "servicenow", Binary: "/app/go-mcp-servicenow", Port: 8085},
}

func main() {
    // Start all MCP servers
    var wg sync.WaitGroup
    for i := range servers {
        wg.Add(1)
        go func(s *Server) {
            defer wg.Done()
            startServer(s)
        }(&servers[i])
    }

    // Wait for servers to start
    time.Sleep(2 * time.Second)

    // Create router
    mux := http.NewServeMux()

    // Health endpoint - checks all servers
    mux.HandleFunc("/health", healthHandler)

    // Route to specific servers by path
    for _, s := range servers {
        serverURL, _ := url.Parse(fmt.Sprintf("http://127.0.0.1:%d", s.Port))
        proxy := httputil.NewSingleHostReverseProxy(serverURL)
        path := fmt.Sprintf("/%s/", s.Name)
        mux.Handle(path, http.StripPrefix(path[:len(path)-1], proxy))
    }

    // Start router on port 8080
    log.Println("Router listening on :8080")
    if err := http.ListenAndServe(":8080", mux); err != nil {
        log.Fatal(err)
    }
}

func startServer(s *Server) {
    s.Process = exec.Command(s.Binary, "--http", "--host", "127.0.0.1", "--port", fmt.Sprintf("%d", s.Port))
    s.Process.Stdout = os.Stdout
    s.Process.Stderr = os.Stderr
    if err := s.Process.Start(); err != nil {
        log.Printf("Failed to start %s: %v", s.Name, err)
        return
    }
    log.Printf("Started %s on port %d", s.Name, s.Port)
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
    allHealthy := true
    results := make(map[string]string)

    for _, s := range servers {
        resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/health", s.Port))
        if err != nil || resp.StatusCode != 200 {
            allHealthy = false
            results[s.Name] = "unhealthy"
        } else {
            results[s.Name] = "healthy"
        }
        if resp != nil {
            io.Copy(io.Discard, resp.Body)
            resp.Body.Close()
        }
    }

    if allHealthy {
        w.WriteHeader(http.StatusOK)
        fmt.Fprintf(w, `{"status":"ok","servers":%v}`, results)
    } else {
        w.WriteHeader(http.StatusServiceUnavailable)
        fmt.Fprintf(w, `{"status":"degraded","servers":%v}`, results)
    }
}
```

### Multi-Server Routing Patterns

#### Path-Based Routing (Recommended)

Requests are routed based on URL path:

```
POST /gitlab/mcp      → go-mcp-gitlab:8081/mcp
POST /atlassian/mcp   → go-mcp-atlassian:8082/mcp
POST /dynatrace/mcp   → go-mcp-dynatrace:8083/mcp
POST /pagerduty/mcp   → go-mcp-pagerduty:8084/mcp
POST /servicenow/mcp  → go-mcp-servicenow:8085/mcp
```

#### Header-Based Routing (Alternative)

Route based on custom header:

```go
func routeByHeader(w http.ResponseWriter, r *http.Request) {
    serverName := r.Header.Get("X-MCP-Server")
    for _, s := range servers {
        if s.Name == serverName {
            proxy := httputil.NewSingleHostReverseProxy(
                &url.URL{Scheme: "http", Host: fmt.Sprintf("127.0.0.1:%d", s.Port)},
            )
            proxy.ServeHTTP(w, r)
            return
        }
    }
    http.Error(w, "Unknown server", http.StatusBadRequest)
}
```

### Selective Server Deployment

Build images with only specific servers enabled:

```dockerfile
# Build stage with selective servers
ARG ENABLE_GITLAB=true
ARG ENABLE_ATLASSIAN=true
ARG ENABLE_DYNATRACE=false
ARG ENABLE_PAGERDUTY=false
ARG ENABLE_SERVICENOW=false

# Conditionally build servers
RUN if [ "$ENABLE_GITLAB" = "true" ]; then \
        cd go-mcp-gitlab && go build -o /bin/go-mcp-gitlab ./cmd/go-mcp-gitlab; \
    fi

# In supervisord.conf or router, check if binary exists before starting
```

### Resource Considerations

| Configuration | Memory | vCPU | Use Case |
|--------------|--------|------|----------|
| Single server | 256 MB | 0.25 | Development, low traffic |
| 2-3 servers | 512 MB | 0.5 | Small teams |
| All 5 servers | 1024 MB | 1.0 | Production multi-tenant |

### Multi-Server Docker Compose (Local Development)

```yaml
version: '3.8'

services:
  mcp-multi:
    build:
      context: .
      dockerfile: Dockerfile.multi
    ports:
      - "8081:8081"  # GitLab
      - "8082:8082"  # Atlassian
      - "8083:8083"  # Dynatrace
      - "8084:8084"  # PagerDuty
      - "8085:8085"  # ServiceNow
    environment:
      # GitLab
      - GITLAB_API_URL=${GITLAB_API_URL}
      # Atlassian
      - JIRA_URL=${JIRA_URL}
      - CONFLUENCE_URL=${CONFLUENCE_URL}
      # Dynatrace
      - DT_ENVIRONMENT=${DT_ENVIRONMENT}
      # PagerDuty
      - PAGERDUTY_API_HOST=api.pagerduty.com
      # ServiceNow
      - SERVICENOW_INSTANCE_URL=${SERVICENOW_INSTANCE_URL}
    healthcheck:
      test: ["CMD", "wget", "-q", "--spider", "http://localhost:8081/health"]
      interval: 10s
      timeout: 5s
      retries: 3
```

### Multi-Server Deployment Concerns

Before deploying multiple MCP servers in a single container or Lambda, consider these trade-offs:

#### Fault Isolation

| Concern | Impact | Mitigation |
|---------|--------|------------|
| **Process crash propagation** | One server crash in supervisor mode terminates all servers | Use `autorestart=true` in supervisor; implement watchdog |
| **Memory leaks** | Memory leak in one server affects all servers | Monitor per-process memory; set memory limits per process |
| **Runaway CPU** | CPU-intensive operation blocks other servers | Use process-level CPU limits with cgroups |
| **Deadlocks** | Deadlock in router blocks all traffic | Implement health-based routing with failover |

#### Scaling Limitations

| Concern | Impact | Mitigation |
|---------|--------|------------|
| **No independent scaling** | Cannot scale GitLab server without scaling all servers | Use separate containers if traffic patterns vary significantly |
| **Resource waste** | Low-traffic servers consume resources | Use selective deployment with only needed servers |
| **Cold start penalty** | Larger image = slower Lambda cold starts | Consider separate Lambdas for latency-critical servers |
| **Uneven load distribution** | One busy server affects others | Implement request queuing per server |

#### Operational Complexity

| Concern | Impact | Mitigation |
|---------|--------|------------|
| **Deployment coupling** | One server update requires full redeployment | Use semantic versioning; implement blue-green deployment |
| **Debugging difficulty** | Log interleaving makes debugging harder | Use structured logging with server tags; separate log streams |
| **Health check complexity** | Aggregated health masks individual failures | Implement per-server health endpoints; detailed health response |
| **Version mismatch** | All servers must be compatible versions | Maintain compatibility matrix; test combinations |

#### Security Considerations

| Concern | Impact | Mitigation |
|---------|--------|------------|
| **Shared credential exposure** | Compromise of one server exposes all credentials | Use per-request credentials via headers instead of env vars |
| **Attack surface** | More servers = larger attack surface | Minimize enabled servers; use selective deployment |
| **Privilege escalation** | Shared process space allows lateral movement | Use non-root users; read-only filesystem |
| **Audit complexity** | Harder to audit which server accessed what | Implement request tracing with correlation IDs |

#### Resource Constraints

| Resource | Lambda Limit | ECS Limit | Recommendation |
|----------|-------------|-----------|----------------|
| Memory | 10,240 MB max | Task-level limit | 1024 MB minimum for 5 servers |
| CPU | Proportional to memory | 4 vCPU max | 1 vCPU minimum for 5 servers |
| Timeout | 15 minutes | No limit | Keep requests under 30s |
| Connections | 1000 concurrent | Network-dependent | Implement connection pooling |
| Disk | 10 GB /tmp | 200 GB ephemeral | Minimal disk usage |

#### When to Use Multi-Server vs Separate Deployments

**Use Multi-Server When:**
- Development/staging environments with low traffic
- Cost optimization is primary concern
- All servers have similar traffic patterns
- Simplified operations is desired
- Teams are small with limited DevOps resources

**Use Separate Deployments When:**
- Production environments with variable traffic
- Individual server SLAs required
- Independent scaling needed
- Security isolation required
- Different update cadences per server
- Debugging and monitoring are critical

#### Monitoring Recommendations

```yaml
# CloudWatch Alarms for Multi-Server Container
Alarms:
  - Name: "MCP-MultiServer-HighMemory"
    Metric: MemoryUtilization
    Threshold: 80%
    Action: Scale up or alert

  - Name: "MCP-MultiServer-ServerUnhealthy"
    Metric: HealthyHostCount
    Threshold: < expected
    Action: Alert and investigate

  - Name: "MCP-MultiServer-HighLatency"
    Metric: TargetResponseTime
    Threshold: 5000ms
    Action: Investigate specific server

# Per-Server Metrics (Custom)
CustomMetrics:
  - Name: "MCP-{Server}-RequestCount"
  - Name: "MCP-{Server}-ErrorRate"
  - Name: "MCP-{Server}-ResponseTime"
```

#### Graceful Degradation Pattern

```go
// Router with per-server circuit breaker
type ServerCircuit struct {
    failures    int
    lastFailure time.Time
    open        bool
}

func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
    server := extractServer(req.URL.Path)

    if r.circuits[server].open {
        // Return 503 for this server only, others still work
        http.Error(w, fmt.Sprintf("%s temporarily unavailable", server), 503)
        return
    }

    // Forward request...
}
```

---

## Appendix A: Quick Reference

### Build Commands Cheat Sheet

```bash
# Build single image
docker build -t mcp-servers/go-mcp-gitlab:v1.0.0 .

# Build with version
docker build --build-arg VERSION=1.0.0 -t mcp-servers/go-mcp-gitlab:v1.0.0 .

# Build for ARM64
docker build --platform linux/arm64 -t mcp-servers/go-mcp-gitlab:v1.0.0-arm64 .

# Multi-arch build
docker buildx build --platform linux/amd64,linux/arm64 -t mcp-servers/go-mcp-gitlab:v1.0.0 --push .

# Push to ECR
aws ecr get-login-password | docker login --username AWS --password-stdin ${ECR_REGISTRY}
docker push ${ECR_REGISTRY}/mcp-servers/go-mcp-gitlab:v1.0.0
```

### Environment Variable Template

```bash
# .env.example
# GitLab
GITLAB_API_URL=https://gitlab.com/api/v4
GITLAB_TOKEN=

# Atlassian
JIRA_URL=https://your-domain.atlassian.net
JIRA_TOKEN=
JIRA_EMAIL=
CONFLUENCE_URL=https://your-domain.atlassian.net/wiki
CONFLUENCE_TOKEN=
CONFLUENCE_EMAIL=

# Dynatrace
DT_ENVIRONMENT=
DT_API_TOKEN=

# PagerDuty
PAGERDUTY_API_KEY=
PAGERDUTY_API_HOST=api.pagerduty.com

# ServiceNow
SERVICENOW_INSTANCE_URL=https://your-instance.service-now.com
SERVICENOW_USERNAME=
SERVICENOW_PASSWORD=
```

---

## Appendix B: Troubleshooting

### Common Issues

| Issue | Cause | Solution |
|-------|-------|----------|
| Container fails to start | Missing Lambda adapter | Verify COPY from adapter image |
| Readiness timeout | Health endpoint not responding | Check AWS_LWA_PORT matches app port |
| Binary not found | Wrong build path | Verify go build output path |
| Permission denied | Non-executable binary | Add `RUN chmod +x /app/server` |
| TLS errors | Missing CA certs | Copy ca-certificates.crt |
| Slow cold starts | Large image size | Use multi-stage, strip binary |

### Debug Commands

```bash
# Check container logs
docker logs mcp-gitlab-test

# Execute shell in running container
docker exec -it mcp-gitlab-test /bin/sh

# Check binary exists and is executable
docker run --rm mcp-servers/go-mcp-gitlab:latest ls -la /app/

# Verify environment variables
docker run --rm mcp-servers/go-mcp-gitlab:latest env

# Test health endpoint directly
docker run --rm -p 8080:8080 mcp-servers/go-mcp-gitlab:latest &
sleep 5
curl http://localhost:8080/health
```

---

*Document Version: 1.0.0*
*Last Updated: 2024-01-15*
