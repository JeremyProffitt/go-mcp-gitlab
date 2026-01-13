# MCP Server Infrastructure Specification

> Complete deployment guide for MCP servers on AWS Lambda and ECS Fargate

---

## Table of Contents

1. [Section A: Lambda Deployment](#section-a-lambda-deployment)
2. [Section B: ECS Fargate Deployment](#section-b-ecs-fargate-deployment)
3. [Section C: Shared Infrastructure](#section-c-shared-infrastructure)

---

## Section A: Lambda Deployment

### 1. Lambda Function Configuration

#### Runtime Configuration

| Setting | Value | Notes |
|---------|-------|-------|
| Runtime | Container Image | Package type for Docker-based Lambdas |
| Memory | 256 MB | Recommended; range 128-512 MB |
| Timeout | 29 seconds | API Gateway limit is 30s; use 29s for safety |
| Architecture | arm64 | 20% cost savings; use amd64 for x86 dependencies |
| Ephemeral Storage | 512 MB | Default; increase if needed for temp files |

#### Environment Variables Per Server

**GitLab MCP Server:**
```bash
GITLAB_API_URL=https://gitlab.com/api/v4
GITLAB_TOKEN_SECRET_ARN=arn:aws:secretsmanager:us-east-1:123456789012:secret:mcp/gitlab-token
LOG_LEVEL=info
```

**Atlassian MCP Server:**
```bash
ATLASSIAN_URL=https://your-domain.atlassian.net
ATLASSIAN_EMAIL_SECRET_ARN=arn:aws:secretsmanager:us-east-1:123456789012:secret:mcp/atlassian-email
ATLASSIAN_TOKEN_SECRET_ARN=arn:aws:secretsmanager:us-east-1:123456789012:secret:mcp/atlassian-token
LOG_LEVEL=info
```

**Dynatrace MCP Server:**
```bash
DYNATRACE_URL=https://your-env.live.dynatrace.com
DYNATRACE_TOKEN_SECRET_ARN=arn:aws:secretsmanager:us-east-1:123456789012:secret:mcp/dynatrace-token
LOG_LEVEL=info
```

**PagerDuty MCP Server:**
```bash
PAGERDUTY_API_URL=https://api.pagerduty.com
PAGERDUTY_TOKEN_SECRET_ARN=arn:aws:secretsmanager:us-east-1:123456789012:secret:mcp/pagerduty-token
LOG_LEVEL=info
```

**ServiceNow MCP Server:**
```bash
SERVICENOW_INSTANCE_URL=https://your-instance.service-now.com
SERVICENOW_USERNAME_SECRET_ARN=arn:aws:secretsmanager:us-east-1:123456789012:secret:mcp/servicenow-username
SERVICENOW_PASSWORD_SECRET_ARN=arn:aws:secretsmanager:us-east-1:123456789012:secret:mcp/servicenow-password
LOG_LEVEL=info
```

#### Dockerfile for Lambda Container

```dockerfile
# Build stage
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -tags lambda.norpc -o bootstrap ./cmd/lambda

# Runtime stage
FROM public.ecr.aws/lambda/provided:al2023-arm64

# Copy the binary
COPY --from=builder /app/bootstrap ${LAMBDA_RUNTIME_DIR}/bootstrap

# Set the handler
CMD ["bootstrap"]
```

#### Lambda Handler Code (Go)

```go
package main

import (
    "context"
    "encoding/json"
    "os"

    "github.com/aws/aws-lambda-go/events"
    "github.com/aws/aws-lambda-go/lambda"
    "github.com/aws/aws-sdk-go-v2/config"
    "github.com/aws/aws-sdk-go-v2/service/secretsmanager"
)

type MCPRequest struct {
    JSONRPC string          `json:"jsonrpc"`
    ID      interface{}     `json:"id"`
    Method  string          `json:"method"`
    Params  json.RawMessage `json:"params,omitempty"`
}

type MCPResponse struct {
    JSONRPC string      `json:"jsonrpc"`
    ID      interface{} `json:"id"`
    Result  interface{} `json:"result,omitempty"`
    Error   *MCPError   `json:"error,omitempty"`
}

type MCPError struct {
    Code    int    `json:"code"`
    Message string `json:"message"`
}

var secretsClient *secretsmanager.Client

func init() {
    cfg, err := config.LoadDefaultConfig(context.Background())
    if err != nil {
        panic("unable to load AWS config: " + err.Error())
    }
    secretsClient = secretsmanager.NewFromConfig(cfg)
}

func getSecret(ctx context.Context, secretARN string) (string, error) {
    result, err := secretsClient.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{
        SecretId: &secretARN,
    })
    if err != nil {
        return "", err
    }
    return *result.SecretString, nil
}

func handler(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
    // Parse MCP request
    var mcpReq MCPRequest
    if err := json.Unmarshal([]byte(request.Body), &mcpReq); err != nil {
        return errorResponse(400, "Invalid JSON-RPC request"), nil
    }

    // Get secrets from Secrets Manager
    tokenARN := os.Getenv("API_TOKEN_SECRET_ARN")
    token, err := getSecret(ctx, tokenARN)
    if err != nil {
        return errorResponse(500, "Failed to retrieve credentials"), nil
    }

    // Process MCP method
    var result interface{}
    switch mcpReq.Method {
    case "initialize":
        result = handleInitialize()
    case "tools/list":
        result = handleToolsList()
    case "tools/call":
        result, err = handleToolsCall(ctx, mcpReq.Params, token)
    default:
        return errorResponse(400, "Unknown method"), nil
    }

    if err != nil {
        return errorResponse(500, err.Error()), nil
    }

    // Build response
    mcpResp := MCPResponse{
        JSONRPC: "2.0",
        ID:      mcpReq.ID,
        Result:  result,
    }

    respBody, _ := json.Marshal(mcpResp)
    return events.APIGatewayProxyResponse{
        StatusCode: 200,
        Headers: map[string]string{
            "Content-Type": "application/json",
        },
        Body: string(respBody),
    }, nil
}

func errorResponse(code int, message string) events.APIGatewayProxyResponse {
    resp := MCPResponse{
        JSONRPC: "2.0",
        Error: &MCPError{
            Code:    code,
            Message: message,
        },
    }
    body, _ := json.Marshal(resp)
    return events.APIGatewayProxyResponse{
        StatusCode: code,
        Headers: map[string]string{
            "Content-Type": "application/json",
        },
        Body: string(body),
    }
}

func handleInitialize() map[string]interface{} {
    return map[string]interface{}{
        "protocolVersion": "2024-11-05",
        "capabilities": map[string]interface{}{
            "tools": map[string]interface{}{},
        },
        "serverInfo": map[string]interface{}{
            "name":    os.Getenv("SERVER_NAME"),
            "version": "1.0.0",
        },
    }
}

func handleToolsList() map[string]interface{} {
    // Return available tools - implement per server
    return map[string]interface{}{
        "tools": []interface{}{},
    }
}

func handleToolsCall(ctx context.Context, params json.RawMessage, token string) (interface{}, error) {
    // Implement tool execution - server specific
    return nil, nil
}

func main() {
    lambda.Start(handler)
}
```

---

### 2. API Gateway Configuration

#### REST API Structure

```
API: mcp-servers-api
├── /gitlab
│   └── POST → gitlab-mcp-lambda (Lambda Proxy Integration)
├── /atlassian
│   └── POST → atlassian-mcp-lambda (Lambda Proxy Integration)
├── /dynatrace
│   └── POST → dynatrace-mcp-lambda (Lambda Proxy Integration)
├── /pagerduty
│   └── POST → pagerduty-mcp-lambda (Lambda Proxy Integration)
└── /servicenow
    └── POST → servicenow-mcp-lambda (Lambda Proxy Integration)
```

#### API Gateway OpenAPI Specification

```yaml
openapi: "3.0.1"
info:
  title: "MCP Servers API"
  version: "1.0.0"
  description: "API Gateway for MCP Server Lambda functions"

x-amazon-apigateway-request-validators:
  all:
    validateRequestBody: true
    validateRequestParameters: true

paths:
  /gitlab:
    post:
      summary: "GitLab MCP Server"
      x-amazon-apigateway-integration:
        type: aws_proxy
        httpMethod: POST
        uri: "arn:aws:apigateway:${AWS::Region}:lambda:path/2015-03-31/functions/arn:aws:lambda:${AWS::Region}:${AWS::AccountId}:function:gitlab-mcp-lambda/invocations"
        passthroughBehavior: when_no_match
      responses:
        "200":
          description: "Successful response"
          headers:
            Access-Control-Allow-Origin:
              schema:
                type: string
            Access-Control-Allow-Methods:
              schema:
                type: string
            Access-Control-Allow-Headers:
              schema:
                type: string
    options:
      summary: "CORS preflight"
      responses:
        "200":
          description: "CORS headers"
          headers:
            Access-Control-Allow-Origin:
              schema:
                type: string
            Access-Control-Allow-Methods:
              schema:
                type: string
            Access-Control-Allow-Headers:
              schema:
                type: string
      x-amazon-apigateway-integration:
        type: mock
        requestTemplates:
          application/json: '{"statusCode": 200}'
        responses:
          default:
            statusCode: "200"
            responseParameters:
              method.response.header.Access-Control-Allow-Headers: "'Content-Type,Authorization,X-Amz-Date,X-Api-Key'"
              method.response.header.Access-Control-Allow-Methods: "'POST,OPTIONS'"
              method.response.header.Access-Control-Allow-Origin: "'*'"

  /atlassian:
    post:
      summary: "Atlassian MCP Server"
      x-amazon-apigateway-integration:
        type: aws_proxy
        httpMethod: POST
        uri: "arn:aws:apigateway:${AWS::Region}:lambda:path/2015-03-31/functions/arn:aws:lambda:${AWS::Region}:${AWS::AccountId}:function:atlassian-mcp-lambda/invocations"
        passthroughBehavior: when_no_match
      responses:
        "200":
          description: "Successful response"
    options:
      summary: "CORS preflight"
      x-amazon-apigateway-integration:
        type: mock
        requestTemplates:
          application/json: '{"statusCode": 200}'
        responses:
          default:
            statusCode: "200"
            responseParameters:
              method.response.header.Access-Control-Allow-Headers: "'Content-Type,Authorization'"
              method.response.header.Access-Control-Allow-Methods: "'POST,OPTIONS'"
              method.response.header.Access-Control-Allow-Origin: "'*'"

  /dynatrace:
    post:
      summary: "Dynatrace MCP Server"
      x-amazon-apigateway-integration:
        type: aws_proxy
        httpMethod: POST
        uri: "arn:aws:apigateway:${AWS::Region}:lambda:path/2015-03-31/functions/arn:aws:lambda:${AWS::Region}:${AWS::AccountId}:function:dynatrace-mcp-lambda/invocations"
        passthroughBehavior: when_no_match
      responses:
        "200":
          description: "Successful response"

  /pagerduty:
    post:
      summary: "PagerDuty MCP Server"
      x-amazon-apigateway-integration:
        type: aws_proxy
        httpMethod: POST
        uri: "arn:aws:apigateway:${AWS::Region}:lambda:path/2015-03-31/functions/arn:aws:lambda:${AWS::Region}:${AWS::AccountId}:function:pagerduty-mcp-lambda/invocations"
        passthroughBehavior: when_no_match
      responses:
        "200":
          description: "Successful response"

  /servicenow:
    post:
      summary: "ServiceNow MCP Server"
      x-amazon-apigateway-integration:
        type: aws_proxy
        httpMethod: POST
        uri: "arn:aws:apigateway:${AWS::Region}:lambda:path/2015-03-31/functions/arn:aws:lambda:${AWS::Region}:${AWS::AccountId}:function:servicenow-mcp-lambda/invocations"
        passthroughBehavior: when_no_match
      responses:
        "200":
          description: "Successful response"
```

#### API Gateway Throttling Settings

```json
{
  "throttle": {
    "burstLimit": 5000,
    "rateLimit": 1000
  },
  "quota": {
    "limit": 1000000,
    "period": "MONTH"
  }
}
```

#### CORS Configuration

For Lambda proxy integration, return CORS headers from the Lambda function:

```go
func corsHeaders() map[string]string {
    return map[string]string{
        "Access-Control-Allow-Origin":  "*",
        "Access-Control-Allow-Methods": "POST,OPTIONS",
        "Access-Control-Allow-Headers": "Content-Type,Authorization,X-Amz-Date,X-Api-Key,X-Amz-Security-Token",
        "Content-Type":                 "application/json",
    }
}

func handler(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
    // Handle OPTIONS preflight
    if request.HTTPMethod == "OPTIONS" {
        return events.APIGatewayProxyResponse{
            StatusCode: 200,
            Headers:    corsHeaders(),
        }, nil
    }

    // ... rest of handler

    return events.APIGatewayProxyResponse{
        StatusCode: 200,
        Headers:    corsHeaders(),
        Body:       string(respBody),
    }, nil
}
```

---

### 3. IAM Roles and Policies

#### Lambda Execution Role

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {
        "Service": "lambda.amazonaws.com"
      },
      "Action": "sts:AssumeRole"
    }
  ]
}
```

#### CloudWatch Logs Policy

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "CloudWatchLogsAccess",
      "Effect": "Allow",
      "Action": [
        "logs:CreateLogGroup",
        "logs:CreateLogStream",
        "logs:PutLogEvents",
        "logs:DescribeLogGroups",
        "logs:DescribeLogStreams"
      ],
      "Resource": [
        "arn:aws:logs:*:*:log-group:/aws/lambda/gitlab-mcp-lambda:*",
        "arn:aws:logs:*:*:log-group:/aws/lambda/atlassian-mcp-lambda:*",
        "arn:aws:logs:*:*:log-group:/aws/lambda/dynatrace-mcp-lambda:*",
        "arn:aws:logs:*:*:log-group:/aws/lambda/pagerduty-mcp-lambda:*",
        "arn:aws:logs:*:*:log-group:/aws/lambda/servicenow-mcp-lambda:*"
      ]
    }
  ]
}
```

#### Secrets Manager Access Policy

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "SecretsManagerAccess",
      "Effect": "Allow",
      "Action": [
        "secretsmanager:GetSecretValue",
        "secretsmanager:DescribeSecret"
      ],
      "Resource": [
        "arn:aws:secretsmanager:us-east-1:123456789012:secret:mcp/*"
      ]
    },
    {
      "Sid": "KMSDecrypt",
      "Effect": "Allow",
      "Action": [
        "kms:Decrypt"
      ],
      "Resource": [
        "arn:aws:kms:us-east-1:123456789012:key/your-kms-key-id"
      ],
      "Condition": {
        "StringEquals": {
          "kms:ViaService": "secretsmanager.us-east-1.amazonaws.com"
        }
      }
    }
  ]
}
```

#### ECR Pull Permissions Policy

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "ECRPullAccess",
      "Effect": "Allow",
      "Action": [
        "ecr:GetDownloadUrlForLayer",
        "ecr:BatchGetImage",
        "ecr:BatchCheckLayerAvailability"
      ],
      "Resource": [
        "arn:aws:ecr:us-east-1:123456789012:repository/mcp-servers/*"
      ]
    },
    {
      "Sid": "ECRAuth",
      "Effect": "Allow",
      "Action": [
        "ecr:GetAuthorizationToken"
      ],
      "Resource": "*"
    }
  ]
}
```

#### Combined Lambda Execution Role Policy

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "CloudWatchLogsAccess",
      "Effect": "Allow",
      "Action": [
        "logs:CreateLogGroup",
        "logs:CreateLogStream",
        "logs:PutLogEvents"
      ],
      "Resource": "arn:aws:logs:*:*:log-group:/aws/lambda/*mcp*:*"
    },
    {
      "Sid": "SecretsManagerAccess",
      "Effect": "Allow",
      "Action": [
        "secretsmanager:GetSecretValue"
      ],
      "Resource": "arn:aws:secretsmanager:*:*:secret:mcp/*"
    },
    {
      "Sid": "ECRAccess",
      "Effect": "Allow",
      "Action": [
        "ecr:GetDownloadUrlForLayer",
        "ecr:BatchGetImage",
        "ecr:BatchCheckLayerAvailability",
        "ecr:GetAuthorizationToken"
      ],
      "Resource": "*"
    },
    {
      "Sid": "XRayTracing",
      "Effect": "Allow",
      "Action": [
        "xray:PutTraceSegments",
        "xray:PutTelemetryRecords"
      ],
      "Resource": "*"
    }
  ]
}
```

---

### 4. Secrets Management

#### Classification: Secrets Manager vs Environment Variables

| Type | Storage | Examples |
|------|---------|----------|
| **Secrets Manager** | API tokens, passwords, private keys | `GITLAB_TOKEN`, `ATLASSIAN_TOKEN`, `DB_PASSWORD` |
| **Environment Variables** | Non-sensitive configuration | `API_URL`, `LOG_LEVEL`, `TIMEOUT`, `REGION` |

**Rule of thumb:** If the value should not appear in logs, dashboards, or source control, use Secrets Manager.

#### Secret Structure

**Single Value Secret:**
```json
{
  "SecretString": "glpat-xxxxxxxxxxxxxxxxxxxx"
}
```

**Multi-Value Secret (Recommended):**
```json
{
  "token": "glpat-xxxxxxxxxxxxxxxxxxxx",
  "url": "https://gitlab.com/api/v4",
  "created": "2024-01-15",
  "rotated": "2024-06-15"
}
```

#### Creating Secrets via AWS CLI

```bash
# Single value secret
aws secretsmanager create-secret \
  --name "mcp/gitlab-token" \
  --description "GitLab API token for MCP server" \
  --secret-string "glpat-xxxxxxxxxxxxxxxxxxxx" \
  --tags Key=Application,Value=mcp-servers Key=Environment,Value=production

# Multi-value secret
aws secretsmanager create-secret \
  --name "mcp/atlassian-credentials" \
  --description "Atlassian credentials for MCP server" \
  --secret-string '{"email":"service@company.com","token":"ATATT3xFfGF0..."}' \
  --tags Key=Application,Value=mcp-servers Key=Environment,Value=production

# With KMS encryption
aws secretsmanager create-secret \
  --name "mcp/servicenow-credentials" \
  --description "ServiceNow credentials for MCP server" \
  --secret-string '{"username":"api_user","password":"secure_password"}' \
  --kms-key-id "arn:aws:kms:us-east-1:123456789012:key/your-key-id"
```

#### Retrieving Secrets in Lambda (Go)

```go
package secrets

import (
    "context"
    "encoding/json"
    "sync"
    "time"

    "github.com/aws/aws-sdk-go-v2/config"
    "github.com/aws/aws-sdk-go-v2/service/secretsmanager"
)

type SecretCache struct {
    client    *secretsmanager.Client
    cache     map[string]cachedSecret
    mu        sync.RWMutex
    ttl       time.Duration
}

type cachedSecret struct {
    value     string
    expiresAt time.Time
}

func NewSecretCache(ttl time.Duration) (*SecretCache, error) {
    cfg, err := config.LoadDefaultConfig(context.Background())
    if err != nil {
        return nil, err
    }

    return &SecretCache{
        client: secretsmanager.NewFromConfig(cfg),
        cache:  make(map[string]cachedSecret),
        ttl:    ttl,
    }, nil
}

func (sc *SecretCache) GetSecret(ctx context.Context, secretARN string) (string, error) {
    // Check cache first
    sc.mu.RLock()
    if cached, ok := sc.cache[secretARN]; ok && time.Now().Before(cached.expiresAt) {
        sc.mu.RUnlock()
        return cached.value, nil
    }
    sc.mu.RUnlock()

    // Fetch from Secrets Manager
    result, err := sc.client.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{
        SecretId: &secretARN,
    })
    if err != nil {
        return "", err
    }

    value := *result.SecretString

    // Update cache
    sc.mu.Lock()
    sc.cache[secretARN] = cachedSecret{
        value:     value,
        expiresAt: time.Now().Add(sc.ttl),
    }
    sc.mu.Unlock()

    return value, nil
}

func (sc *SecretCache) GetSecretJSON(ctx context.Context, secretARN string, target interface{}) error {
    value, err := sc.GetSecret(ctx, secretARN)
    if err != nil {
        return err
    }
    return json.Unmarshal([]byte(value), target)
}
```

#### Secret Rotation Lambda

```go
package main

import (
    "context"
    "encoding/json"
    "fmt"

    "github.com/aws/aws-lambda-go/lambda"
    "github.com/aws/aws-sdk-go-v2/config"
    "github.com/aws/aws-sdk-go-v2/service/secretsmanager"
)

type RotationEvent struct {
    SecretId           string `json:"SecretId"`
    ClientRequestToken string `json:"ClientRequestToken"`
    Step               string `json:"Step"`
}

func handler(ctx context.Context, event RotationEvent) error {
    cfg, _ := config.LoadDefaultConfig(ctx)
    client := secretsmanager.NewFromConfig(cfg)

    switch event.Step {
    case "createSecret":
        return createSecret(ctx, client, event)
    case "setSecret":
        return setSecret(ctx, client, event)
    case "testSecret":
        return testSecret(ctx, client, event)
    case "finishSecret":
        return finishSecret(ctx, client, event)
    default:
        return fmt.Errorf("unknown step: %s", event.Step)
    }
}

func createSecret(ctx context.Context, client *secretsmanager.Client, event RotationEvent) error {
    // Generate new secret value
    // Implementation depends on the service (GitLab, Atlassian, etc.)
    return nil
}

func setSecret(ctx context.Context, client *secretsmanager.Client, event RotationEvent) error {
    // Set the new secret in the target service
    return nil
}

func testSecret(ctx context.Context, client *secretsmanager.Client, event RotationEvent) error {
    // Test that the new secret works
    return nil
}

func finishSecret(ctx context.Context, client *secretsmanager.Client, event RotationEvent) error {
    // Move AWSPENDING to AWSCURRENT
    metadata, _ := client.DescribeSecret(ctx, &secretsmanager.DescribeSecretInput{
        SecretId: &event.SecretId,
    })

    currentVersion := ""
    for version, stages := range metadata.VersionIdsToStages {
        for _, stage := range stages {
            if stage == "AWSCURRENT" {
                currentVersion = version
                break
            }
        }
    }

    _, err := client.UpdateSecretVersionStage(ctx, &secretsmanager.UpdateSecretVersionStageInput{
        SecretId:            &event.SecretId,
        VersionStage:        aws.String("AWSCURRENT"),
        MoveToVersionId:     &event.ClientRequestToken,
        RemoveFromVersionId: &currentVersion,
    })

    return err
}

func main() {
    lambda.Start(handler)
}
```

---

### 5. CloudWatch Integration

#### Log Group Configuration

```bash
# Create log groups with retention
aws logs create-log-group --log-group-name /aws/lambda/gitlab-mcp-lambda
aws logs put-retention-policy --log-group-name /aws/lambda/gitlab-mcp-lambda --retention-in-days 30

aws logs create-log-group --log-group-name /aws/lambda/atlassian-mcp-lambda
aws logs put-retention-policy --log-group-name /aws/lambda/atlassian-mcp-lambda --retention-in-days 30

aws logs create-log-group --log-group-name /aws/lambda/dynatrace-mcp-lambda
aws logs put-retention-policy --log-group-name /aws/lambda/dynatrace-mcp-lambda --retention-in-days 30

aws logs create-log-group --log-group-name /aws/lambda/pagerduty-mcp-lambda
aws logs put-retention-policy --log-group-name /aws/lambda/pagerduty-mcp-lambda --retention-in-days 30

aws logs create-log-group --log-group-name /aws/lambda/servicenow-mcp-lambda
aws logs put-retention-policy --log-group-name /aws/lambda/servicenow-mcp-lambda --retention-in-days 30
```

#### CloudWatch Metric Alarms

**Error Rate Alarm:**
```json
{
  "AlarmName": "mcp-gitlab-lambda-errors",
  "AlarmDescription": "Alarm when GitLab MCP Lambda error rate exceeds threshold",
  "MetricName": "Errors",
  "Namespace": "AWS/Lambda",
  "Dimensions": [
    {
      "Name": "FunctionName",
      "Value": "gitlab-mcp-lambda"
    }
  ],
  "Statistic": "Sum",
  "Period": 60,
  "EvaluationPeriods": 3,
  "Threshold": 5,
  "ComparisonOperator": "GreaterThanThreshold",
  "TreatMissingData": "notBreaching",
  "ActionsEnabled": true,
  "AlarmActions": [
    "arn:aws:sns:us-east-1:123456789012:mcp-alerts"
  ]
}
```

**Duration Alarm:**
```json
{
  "AlarmName": "mcp-gitlab-lambda-duration",
  "AlarmDescription": "Alarm when GitLab MCP Lambda duration exceeds 25 seconds",
  "MetricName": "Duration",
  "Namespace": "AWS/Lambda",
  "Dimensions": [
    {
      "Name": "FunctionName",
      "Value": "gitlab-mcp-lambda"
    }
  ],
  "Statistic": "Average",
  "Period": 60,
  "EvaluationPeriods": 3,
  "Threshold": 25000,
  "ComparisonOperator": "GreaterThanThreshold",
  "TreatMissingData": "notBreaching",
  "AlarmActions": [
    "arn:aws:sns:us-east-1:123456789012:mcp-alerts"
  ]
}
```

**Throttle Alarm:**
```json
{
  "AlarmName": "mcp-gitlab-lambda-throttles",
  "AlarmDescription": "Alarm when GitLab MCP Lambda is throttled",
  "MetricName": "Throttles",
  "Namespace": "AWS/Lambda",
  "Dimensions": [
    {
      "Name": "FunctionName",
      "Value": "gitlab-mcp-lambda"
    }
  ],
  "Statistic": "Sum",
  "Period": 60,
  "EvaluationPeriods": 1,
  "Threshold": 1,
  "ComparisonOperator": "GreaterThanOrEqualToThreshold",
  "TreatMissingData": "notBreaching",
  "AlarmActions": [
    "arn:aws:sns:us-east-1:123456789012:mcp-alerts"
  ]
}
```

#### CloudWatch Dashboard

```json
{
  "widgets": [
    {
      "type": "metric",
      "x": 0,
      "y": 0,
      "width": 12,
      "height": 6,
      "properties": {
        "title": "Lambda Invocations",
        "metrics": [
          ["AWS/Lambda", "Invocations", "FunctionName", "gitlab-mcp-lambda"],
          ["...", "atlassian-mcp-lambda"],
          ["...", "dynatrace-mcp-lambda"],
          ["...", "pagerduty-mcp-lambda"],
          ["...", "servicenow-mcp-lambda"]
        ],
        "period": 60,
        "stat": "Sum",
        "region": "us-east-1"
      }
    },
    {
      "type": "metric",
      "x": 12,
      "y": 0,
      "width": 12,
      "height": 6,
      "properties": {
        "title": "Lambda Errors",
        "metrics": [
          ["AWS/Lambda", "Errors", "FunctionName", "gitlab-mcp-lambda", {"color": "#d62728"}],
          ["...", "atlassian-mcp-lambda"],
          ["...", "dynatrace-mcp-lambda"],
          ["...", "pagerduty-mcp-lambda"],
          ["...", "servicenow-mcp-lambda"]
        ],
        "period": 60,
        "stat": "Sum",
        "region": "us-east-1"
      }
    },
    {
      "type": "metric",
      "x": 0,
      "y": 6,
      "width": 12,
      "height": 6,
      "properties": {
        "title": "Lambda Duration (ms)",
        "metrics": [
          ["AWS/Lambda", "Duration", "FunctionName", "gitlab-mcp-lambda"],
          ["...", "atlassian-mcp-lambda"],
          ["...", "dynatrace-mcp-lambda"],
          ["...", "pagerduty-mcp-lambda"],
          ["...", "servicenow-mcp-lambda"]
        ],
        "period": 60,
        "stat": "Average",
        "region": "us-east-1"
      }
    },
    {
      "type": "metric",
      "x": 12,
      "y": 6,
      "width": 12,
      "height": 6,
      "properties": {
        "title": "Lambda Concurrent Executions",
        "metrics": [
          ["AWS/Lambda", "ConcurrentExecutions", "FunctionName", "gitlab-mcp-lambda"],
          ["...", "atlassian-mcp-lambda"],
          ["...", "dynatrace-mcp-lambda"],
          ["...", "pagerduty-mcp-lambda"],
          ["...", "servicenow-mcp-lambda"]
        ],
        "period": 60,
        "stat": "Maximum",
        "region": "us-east-1"
      }
    }
  ]
}
```

#### Create Alarms via AWS CLI

```bash
#!/bin/bash

FUNCTIONS=("gitlab-mcp-lambda" "atlassian-mcp-lambda" "dynatrace-mcp-lambda" "pagerduty-mcp-lambda" "servicenow-mcp-lambda")
SNS_TOPIC="arn:aws:sns:us-east-1:123456789012:mcp-alerts"

for FUNC in "${FUNCTIONS[@]}"; do
  # Error alarm
  aws cloudwatch put-metric-alarm \
    --alarm-name "${FUNC}-errors" \
    --alarm-description "Error rate alarm for ${FUNC}" \
    --metric-name Errors \
    --namespace AWS/Lambda \
    --dimensions Name=FunctionName,Value=${FUNC} \
    --statistic Sum \
    --period 60 \
    --evaluation-periods 3 \
    --threshold 5 \
    --comparison-operator GreaterThanThreshold \
    --treat-missing-data notBreaching \
    --alarm-actions ${SNS_TOPIC}

  # Duration alarm
  aws cloudwatch put-metric-alarm \
    --alarm-name "${FUNC}-duration" \
    --alarm-description "Duration alarm for ${FUNC}" \
    --metric-name Duration \
    --namespace AWS/Lambda \
    --dimensions Name=FunctionName,Value=${FUNC} \
    --statistic Average \
    --period 60 \
    --evaluation-periods 3 \
    --threshold 25000 \
    --comparison-operator GreaterThanThreshold \
    --treat-missing-data notBreaching \
    --alarm-actions ${SNS_TOPIC}

  # Throttle alarm
  aws cloudwatch put-metric-alarm \
    --alarm-name "${FUNC}-throttles" \
    --alarm-description "Throttle alarm for ${FUNC}" \
    --metric-name Throttles \
    --namespace AWS/Lambda \
    --dimensions Name=FunctionName,Value=${FUNC} \
    --statistic Sum \
    --period 60 \
    --evaluation-periods 1 \
    --threshold 1 \
    --comparison-operator GreaterThanOrEqualToThreshold \
    --treat-missing-data notBreaching \
    --alarm-actions ${SNS_TOPIC}
done
```

---

### 6. CloudFormation/SAM Template

#### Complete SAM Template for One MCP Server

```yaml
AWSTemplateFormatVersion: '2010-09-09'
Transform: AWS::Serverless-2016-10-31
Description: MCP Server Lambda Deployment

Parameters:
  Environment:
    Type: String
    Default: production
    AllowedValues:
      - development
      - staging
      - production

  ServerName:
    Type: String
    Default: gitlab
    Description: Name of the MCP server (gitlab, atlassian, dynatrace, pagerduty, servicenow)

  ImageUri:
    Type: String
    Description: ECR Image URI for the Lambda function

  MemorySize:
    Type: Number
    Default: 256
    MinValue: 128
    MaxValue: 512

  SecretArn:
    Type: String
    Description: ARN of the secret containing API credentials

Globals:
  Function:
    Timeout: 29
    MemorySize: !Ref MemorySize
    Tracing: Active
    Environment:
      Variables:
        ENVIRONMENT: !Ref Environment
        LOG_LEVEL: info

Resources:
  # CloudWatch Log Group
  LambdaLogGroup:
    Type: AWS::Logs::LogGroup
    Properties:
      LogGroupName: !Sub /aws/lambda/${ServerName}-mcp-lambda
      RetentionInDays: 30
      Tags:
        - Key: Application
          Value: mcp-servers
        - Key: Environment
          Value: !Ref Environment

  # Lambda Execution Role
  LambdaExecutionRole:
    Type: AWS::IAM::Role
    Properties:
      RoleName: !Sub ${ServerName}-mcp-lambda-role
      AssumeRolePolicyDocument:
        Version: '2012-10-17'
        Statement:
          - Effect: Allow
            Principal:
              Service: lambda.amazonaws.com
            Action: sts:AssumeRole
      ManagedPolicyArns:
        - arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole
        - arn:aws:iam::aws:policy/AWSXRayDaemonWriteAccess
      Policies:
        - PolicyName: SecretsManagerAccess
          PolicyDocument:
            Version: '2012-10-17'
            Statement:
              - Effect: Allow
                Action:
                  - secretsmanager:GetSecretValue
                Resource: !Ref SecretArn
        - PolicyName: ECRAccess
          PolicyDocument:
            Version: '2012-10-17'
            Statement:
              - Effect: Allow
                Action:
                  - ecr:GetDownloadUrlForLayer
                  - ecr:BatchGetImage
                  - ecr:BatchCheckLayerAvailability
                Resource: !Sub arn:aws:ecr:${AWS::Region}:${AWS::AccountId}:repository/mcp-servers/*
              - Effect: Allow
                Action:
                  - ecr:GetAuthorizationToken
                Resource: '*'
      Tags:
        - Key: Application
          Value: mcp-servers
        - Key: Environment
          Value: !Ref Environment

  # Lambda Function
  MCPServerFunction:
    Type: AWS::Serverless::Function
    Properties:
      FunctionName: !Sub ${ServerName}-mcp-lambda
      PackageType: Image
      ImageUri: !Ref ImageUri
      Architectures:
        - arm64
      Role: !GetAtt LambdaExecutionRole.Arn
      Environment:
        Variables:
          SERVER_NAME: !Ref ServerName
          SECRET_ARN: !Ref SecretArn
      Events:
        ApiEvent:
          Type: Api
          Properties:
            RestApiId: !Ref MCPServerApi
            Path: !Sub /${ServerName}
            Method: POST
      Tags:
        Application: mcp-servers
        Environment: !Ref Environment
    DependsOn: LambdaLogGroup

  # API Gateway
  MCPServerApi:
    Type: AWS::Serverless::Api
    Properties:
      Name: !Sub mcp-${ServerName}-api
      StageName: !Ref Environment
      EndpointConfiguration:
        Type: REGIONAL
      TracingEnabled: true
      MethodSettings:
        - ResourcePath: /*
          HttpMethod: '*'
          ThrottlingBurstLimit: 5000
          ThrottlingRateLimit: 1000
          LoggingLevel: INFO
          DataTraceEnabled: true
          MetricsEnabled: true
      Cors:
        AllowMethods: "'POST,OPTIONS'"
        AllowHeaders: "'Content-Type,Authorization,X-Amz-Date,X-Api-Key,X-Amz-Security-Token'"
        AllowOrigin: "'*'"
      Tags:
        Application: mcp-servers
        Environment: !Ref Environment

  # Lambda Permission for API Gateway
  LambdaApiGatewayPermission:
    Type: AWS::Lambda::Permission
    Properties:
      FunctionName: !Ref MCPServerFunction
      Action: lambda:InvokeFunction
      Principal: apigateway.amazonaws.com
      SourceArn: !Sub arn:aws:execute-api:${AWS::Region}:${AWS::AccountId}:${MCPServerApi}/*/*/*

  # CloudWatch Alarms
  ErrorAlarm:
    Type: AWS::CloudWatch::Alarm
    Properties:
      AlarmName: !Sub ${ServerName}-mcp-lambda-errors
      AlarmDescription: !Sub Error rate alarm for ${ServerName} MCP Lambda
      MetricName: Errors
      Namespace: AWS/Lambda
      Dimensions:
        - Name: FunctionName
          Value: !Ref MCPServerFunction
      Statistic: Sum
      Period: 60
      EvaluationPeriods: 3
      Threshold: 5
      ComparisonOperator: GreaterThanThreshold
      TreatMissingData: notBreaching

  DurationAlarm:
    Type: AWS::CloudWatch::Alarm
    Properties:
      AlarmName: !Sub ${ServerName}-mcp-lambda-duration
      AlarmDescription: !Sub Duration alarm for ${ServerName} MCP Lambda
      MetricName: Duration
      Namespace: AWS/Lambda
      Dimensions:
        - Name: FunctionName
          Value: !Ref MCPServerFunction
      Statistic: Average
      Period: 60
      EvaluationPeriods: 3
      Threshold: 25000
      ComparisonOperator: GreaterThanThreshold
      TreatMissingData: notBreaching

  ThrottleAlarm:
    Type: AWS::CloudWatch::Alarm
    Properties:
      AlarmName: !Sub ${ServerName}-mcp-lambda-throttles
      AlarmDescription: !Sub Throttle alarm for ${ServerName} MCP Lambda
      MetricName: Throttles
      Namespace: AWS/Lambda
      Dimensions:
        - Name: FunctionName
          Value: !Ref MCPServerFunction
      Statistic: Sum
      Period: 60
      EvaluationPeriods: 1
      Threshold: 1
      ComparisonOperator: GreaterThanOrEqualToThreshold
      TreatMissingData: notBreaching

Outputs:
  FunctionArn:
    Description: Lambda Function ARN
    Value: !GetAtt MCPServerFunction.Arn
    Export:
      Name: !Sub ${ServerName}-mcp-lambda-arn

  ApiEndpoint:
    Description: API Gateway Endpoint
    Value: !Sub https://${MCPServerApi}.execute-api.${AWS::Region}.amazonaws.com/${Environment}/${ServerName}
    Export:
      Name: !Sub ${ServerName}-mcp-api-endpoint

  ApiId:
    Description: API Gateway ID
    Value: !Ref MCPServerApi
    Export:
      Name: !Sub ${ServerName}-mcp-api-id

  LogGroupName:
    Description: CloudWatch Log Group
    Value: !Ref LambdaLogGroup
    Export:
      Name: !Sub ${ServerName}-mcp-log-group
```

#### Deploy SAM Template

```bash
# Build and deploy
sam build

sam deploy \
  --stack-name gitlab-mcp-server \
  --parameter-overrides \
    Environment=production \
    ServerName=gitlab \
    ImageUri=123456789012.dkr.ecr.us-east-1.amazonaws.com/mcp-servers/gitlab:latest \
    MemorySize=256 \
    SecretArn=arn:aws:secretsmanager:us-east-1:123456789012:secret:mcp/gitlab-token-AbCdEf \
  --capabilities CAPABILITY_IAM CAPABILITY_NAMED_IAM \
  --region us-east-1
```

---

### 6. Multi-Server Lambda Deployment

Deploy multiple MCP servers within a single Lambda function using path-based routing.

#### Multi-Server Lambda Configuration

| Setting | Value | Notes |
|---------|-------|-------|
| Runtime | Container Image | Multi-server container with router |
| Memory | 1024 MB | Recommended for 5 servers |
| Timeout | 29 seconds | API Gateway limit |
| Architecture | arm64 | Cost optimization |
| Ephemeral Storage | 512 MB | Default |

#### Environment Variables

```bash
# All servers share these Lambda Web Adapter settings
AWS_LWA_PORT=8080
AWS_LWA_READINESS_PATH=/health
AWS_LWA_INVOKE_MODE=response_stream

# Server-specific credentials (per-request via headers, or from Secrets Manager)
GITLAB_TOKEN_SECRET_ARN=arn:aws:secretsmanager:us-east-1:123456789012:secret:mcp/gitlab
ATLASSIAN_SECRET_ARN=arn:aws:secretsmanager:us-east-1:123456789012:secret:mcp/atlassian
DYNATRACE_SECRET_ARN=arn:aws:secretsmanager:us-east-1:123456789012:secret:mcp/dynatrace
PAGERDUTY_SECRET_ARN=arn:aws:secretsmanager:us-east-1:123456789012:secret:mcp/pagerduty
SERVICENOW_SECRET_ARN=arn:aws:secretsmanager:us-east-1:123456789012:secret:mcp/servicenow
```

#### API Gateway Configuration for Multi-Server

```yaml
openapi: "3.0.1"
info:
  title: "MCP Multi-Server API"
  version: "1.0.0"

paths:
  /mcp/{server}/{proxy+}:
    x-amazon-apigateway-any-method:
      parameters:
        - name: server
          in: path
          required: true
          schema:
            type: string
            enum: [gitlab, atlassian, dynatrace, pagerduty, servicenow]
        - name: proxy
          in: path
          required: true
          schema:
            type: string
      x-amazon-apigateway-integration:
        type: aws_proxy
        httpMethod: POST
        uri: "arn:aws:apigateway:${AWS::Region}:lambda:path/2015-03-31/functions/${MultiServerLambdaArn}/invocations"
        passthroughBehavior: when_no_match

  /health:
    get:
      summary: "Aggregated health check"
      x-amazon-apigateway-integration:
        type: aws_proxy
        httpMethod: POST
        uri: "arn:aws:apigateway:${AWS::Region}:lambda:path/2015-03-31/functions/${MultiServerLambdaArn}/invocations"
```

#### Terraform Module: Multi-Server Lambda

```hcl
# modules/multi-server-lambda/main.tf

variable "environment" {
  type = string
}

variable "image_uri" {
  type = string
}

variable "memory_size" {
  type    = number
  default = 1024
}

variable "secret_arns" {
  type = map(string)
  description = "Map of server name to secret ARN"
}

# Lambda Function
resource "aws_lambda_function" "mcp_multi_server" {
  function_name = "mcp-multi-server-${var.environment}"
  package_type  = "Image"
  image_uri     = var.image_uri
  role          = aws_iam_role.lambda_execution.arn
  memory_size   = var.memory_size
  timeout       = 29
  architectures = ["arm64"]

  environment {
    variables = {
      AWS_LWA_PORT           = "8080"
      AWS_LWA_READINESS_PATH = "/health"
      AWS_LWA_INVOKE_MODE    = "response_stream"
      GITLAB_SECRET_ARN      = lookup(var.secret_arns, "gitlab", "")
      ATLASSIAN_SECRET_ARN   = lookup(var.secret_arns, "atlassian", "")
      DYNATRACE_SECRET_ARN   = lookup(var.secret_arns, "dynatrace", "")
      PAGERDUTY_SECRET_ARN   = lookup(var.secret_arns, "pagerduty", "")
      SERVICENOW_SECRET_ARN  = lookup(var.secret_arns, "servicenow", "")
    }
  }

  tags = {
    Application = "mcp-servers"
    Environment = var.environment
    Type        = "multi-server"
  }
}

# IAM Role
resource "aws_iam_role" "lambda_execution" {
  name = "mcp-multi-server-execution-${var.environment}"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Action = "sts:AssumeRole"
      Effect = "Allow"
      Principal = {
        Service = "lambda.amazonaws.com"
      }
    }]
  })
}

# IAM Policy
resource "aws_iam_role_policy" "lambda_policy" {
  name = "mcp-multi-server-policy"
  role = aws_iam_role.lambda_execution.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "logs:CreateLogGroup",
          "logs:CreateLogStream",
          "logs:PutLogEvents"
        ]
        Resource = "arn:aws:logs:*:*:log-group:/aws/lambda/mcp-multi-server-*:*"
      },
      {
        Effect = "Allow"
        Action = [
          "secretsmanager:GetSecretValue"
        ]
        Resource = values(var.secret_arns)
      },
      {
        Effect = "Allow"
        Action = [
          "ecr:GetDownloadUrlForLayer",
          "ecr:BatchGetImage",
          "ecr:BatchCheckLayerAvailability",
          "ecr:GetAuthorizationToken"
        ]
        Resource = "*"
      }
    ]
  })
}

# API Gateway
resource "aws_apigatewayv2_api" "mcp_api" {
  name          = "mcp-multi-server-api-${var.environment}"
  protocol_type = "HTTP"

  cors_configuration {
    allow_origins = ["*"]
    allow_methods = ["POST", "OPTIONS"]
    allow_headers = ["Content-Type", "Authorization", "X-*"]
  }
}

resource "aws_apigatewayv2_integration" "lambda" {
  api_id             = aws_apigatewayv2_api.mcp_api.id
  integration_type   = "AWS_PROXY"
  integration_uri    = aws_lambda_function.mcp_multi_server.invoke_arn
  integration_method = "POST"
}

# Routes for each server
resource "aws_apigatewayv2_route" "server_routes" {
  for_each = toset(["gitlab", "atlassian", "dynatrace", "pagerduty", "servicenow"])

  api_id    = aws_apigatewayv2_api.mcp_api.id
  route_key = "POST /${each.key}/{proxy+}"
  target    = "integrations/${aws_apigatewayv2_integration.lambda.id}"
}

resource "aws_apigatewayv2_route" "health" {
  api_id    = aws_apigatewayv2_api.mcp_api.id
  route_key = "GET /health"
  target    = "integrations/${aws_apigatewayv2_integration.lambda.id}"
}

resource "aws_apigatewayv2_stage" "default" {
  api_id      = aws_apigatewayv2_api.mcp_api.id
  name        = "$default"
  auto_deploy = true
}

resource "aws_lambda_permission" "api_gateway" {
  statement_id  = "AllowAPIGateway"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.mcp_multi_server.function_name
  principal     = "apigateway.amazonaws.com"
  source_arn    = "${aws_apigatewayv2_api.mcp_api.execution_arn}/*/*"
}

output "api_endpoint" {
  value = aws_apigatewayv2_stage.default.invoke_url
}

output "lambda_arn" {
  value = aws_lambda_function.mcp_multi_server.arn
}
```

#### Usage Example

```hcl
module "mcp_multi_server" {
  source = "./modules/multi-server-lambda"

  environment = "production"
  image_uri   = "123456789012.dkr.ecr.us-east-1.amazonaws.com/mcp-servers/multi:latest"
  memory_size = 1024

  secret_arns = {
    gitlab     = "arn:aws:secretsmanager:us-east-1:123456789012:secret:mcp/gitlab-AbCdEf"
    atlassian  = "arn:aws:secretsmanager:us-east-1:123456789012:secret:mcp/atlassian-GhIjKl"
    dynatrace  = "arn:aws:secretsmanager:us-east-1:123456789012:secret:mcp/dynatrace-MnOpQr"
    pagerduty  = "arn:aws:secretsmanager:us-east-1:123456789012:secret:mcp/pagerduty-StUvWx"
    servicenow = "arn:aws:secretsmanager:us-east-1:123456789012:secret:mcp/servicenow-YzAbCd"
  }
}
```

#### API Endpoints

Once deployed, the multi-server Lambda provides these endpoints:

| Endpoint | Description |
|----------|-------------|
| `POST /gitlab/mcp` | GitLab MCP server |
| `POST /atlassian/mcp` | Atlassian MCP server |
| `POST /dynatrace/mcp` | Dynatrace MCP server |
| `POST /pagerduty/mcp` | PagerDuty MCP server |
| `POST /servicenow/mcp` | ServiceNow MCP server |
| `GET /health` | Aggregated health check |

#### Cost Comparison

| Configuration | Lambda Cost (1M requests/month) | Notes |
|--------------|--------------------------------|-------|
| 5 separate Lambdas | ~$25-50 | 5x cold starts, 5x deployments |
| 1 multi-server Lambda | ~$15-25 | Single deployment, shared cold start |

---

## Section B: ECS Fargate Deployment

### 1. ECS Cluster Configuration

#### Cluster Settings

```json
{
  "clusterName": "mcp-servers-cluster",
  "capacityProviders": ["FARGATE", "FARGATE_SPOT"],
  "defaultCapacityProviderStrategy": [
    {
      "capacityProvider": "FARGATE",
      "weight": 1,
      "base": 1
    },
    {
      "capacityProvider": "FARGATE_SPOT",
      "weight": 3,
      "base": 0
    }
  ],
  "settings": [
    {
      "name": "containerInsights",
      "value": "enabled"
    }
  ],
  "configuration": {
    "executeCommandConfiguration": {
      "logging": "DEFAULT"
    }
  },
  "tags": [
    {
      "key": "Application",
      "value": "mcp-servers"
    },
    {
      "key": "Environment",
      "value": "production"
    }
  ]
}
```

#### Create Cluster via AWS CLI

```bash
aws ecs create-cluster \
  --cluster-name mcp-servers-cluster \
  --capacity-providers FARGATE FARGATE_SPOT \
  --default-capacity-provider-strategy \
    capacityProvider=FARGATE,weight=1,base=1 \
    capacityProvider=FARGATE_SPOT,weight=3,base=0 \
  --settings name=containerInsights,value=enabled \
  --tags key=Application,value=mcp-servers key=Environment,value=production
```

---

### 2. Task Definition

#### Complete Task Definition JSON

```json
{
  "family": "gitlab-mcp-server",
  "networkMode": "awsvpc",
  "requiresCompatibilities": ["FARGATE"],
  "cpu": "256",
  "memory": "512",
  "executionRoleArn": "arn:aws:iam::123456789012:role/mcp-ecs-execution-role",
  "taskRoleArn": "arn:aws:iam::123456789012:role/mcp-ecs-task-role",
  "runtimePlatform": {
    "cpuArchitecture": "ARM64",
    "operatingSystemFamily": "LINUX"
  },
  "containerDefinitions": [
    {
      "name": "gitlab-mcp-server",
      "image": "123456789012.dkr.ecr.us-east-1.amazonaws.com/mcp-servers/gitlab:latest",
      "essential": true,
      "portMappings": [
        {
          "containerPort": 8080,
          "hostPort": 8080,
          "protocol": "tcp",
          "name": "http",
          "appProtocol": "http"
        }
      ],
      "environment": [
        {
          "name": "PORT",
          "value": "8080"
        },
        {
          "name": "LOG_LEVEL",
          "value": "info"
        },
        {
          "name": "ENVIRONMENT",
          "value": "production"
        },
        {
          "name": "SERVER_NAME",
          "value": "gitlab"
        },
        {
          "name": "GITLAB_API_URL",
          "value": "https://gitlab.com/api/v4"
        }
      ],
      "secrets": [
        {
          "name": "GITLAB_TOKEN",
          "valueFrom": "arn:aws:secretsmanager:us-east-1:123456789012:secret:mcp/gitlab-token:token::"
        }
      ],
      "healthCheck": {
        "command": [
          "CMD-SHELL",
          "curl -f http://localhost:8080/health || exit 1"
        ],
        "interval": 30,
        "timeout": 5,
        "retries": 3,
        "startPeriod": 60
      },
      "logConfiguration": {
        "logDriver": "awslogs",
        "options": {
          "awslogs-group": "/ecs/mcp-servers/gitlab",
          "awslogs-region": "us-east-1",
          "awslogs-stream-prefix": "ecs",
          "awslogs-create-group": "true"
        }
      },
      "ulimits": [
        {
          "name": "nofile",
          "softLimit": 65536,
          "hardLimit": 65536
        }
      ],
      "linuxParameters": {
        "initProcessEnabled": true
      }
    }
  ],
  "tags": [
    {
      "key": "Application",
      "value": "mcp-servers"
    },
    {
      "key": "Server",
      "value": "gitlab"
    }
  ]
}
```

#### Register Task Definition via AWS CLI

```bash
aws ecs register-task-definition --cli-input-json file://task-definition.json
```

#### Dockerfile for ECS

```dockerfile
# Build stage
FROM golang:1.22-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o server ./cmd/server

# Runtime stage
FROM alpine:3.19

RUN apk --no-cache add ca-certificates curl

WORKDIR /app

COPY --from=builder /app/server .

# Create non-root user
RUN addgroup -g 1000 appgroup && \
    adduser -u 1000 -G appgroup -D appuser && \
    chown -R appuser:appgroup /app

USER appuser

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=5s --start-period=60s --retries=3 \
  CMD curl -f http://localhost:8080/health || exit 1

ENTRYPOINT ["./server"]
```

#### HTTP Server Code for ECS

```go
package main

import (
    "context"
    "encoding/json"
    "log/slog"
    "net/http"
    "os"
    "os/signal"
    "syscall"
    "time"
)

func main() {
    port := os.Getenv("PORT")
    if port == "" {
        port = "8080"
    }

    logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
    slog.SetDefault(logger)

    mux := http.NewServeMux()

    // Health check endpoint
    mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
    })

    // MCP endpoint
    mux.HandleFunc("/mcp", handleMCP)

    server := &http.Server{
        Addr:         ":" + port,
        Handler:      mux,
        ReadTimeout:  30 * time.Second,
        WriteTimeout: 30 * time.Second,
        IdleTimeout:  120 * time.Second,
    }

    // Graceful shutdown
    go func() {
        sigChan := make(chan os.Signal, 1)
        signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
        <-sigChan

        slog.Info("Shutting down server...")
        ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
        defer cancel()

        if err := server.Shutdown(ctx); err != nil {
            slog.Error("Server shutdown error", "error", err)
        }
    }()

    slog.Info("Server starting", "port", port)
    if err := server.ListenAndServe(); err != http.ErrServerClosed {
        slog.Error("Server error", "error", err)
        os.Exit(1)
    }
}

func handleMCP(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    var mcpReq MCPRequest
    if err := json.NewDecoder(r.Body).Decode(&mcpReq); err != nil {
        http.Error(w, "Invalid request", http.StatusBadRequest)
        return
    }

    // Process MCP request...
    response := MCPResponse{
        JSONRPC: "2.0",
        ID:      mcpReq.ID,
        Result:  map[string]interface{}{"status": "ok"},
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}
```

---

### 3. Service Configuration

#### ECS Service Definition

```json
{
  "serviceName": "gitlab-mcp-service",
  "cluster": "mcp-servers-cluster",
  "taskDefinition": "gitlab-mcp-server:1",
  "desiredCount": 2,
  "launchType": "FARGATE",
  "platformVersion": "LATEST",
  "deploymentConfiguration": {
    "deploymentCircuitBreaker": {
      "enable": true,
      "rollback": true
    },
    "maximumPercent": 200,
    "minimumHealthyPercent": 100
  },
  "networkConfiguration": {
    "awsvpcConfiguration": {
      "subnets": [
        "subnet-private-1a",
        "subnet-private-1b",
        "subnet-private-1c"
      ],
      "securityGroups": [
        "sg-ecs-tasks"
      ],
      "assignPublicIp": "DISABLED"
    }
  },
  "loadBalancers": [
    {
      "targetGroupArn": "arn:aws:elasticloadbalancing:us-east-1:123456789012:targetgroup/gitlab-mcp-tg/1234567890123456",
      "containerName": "gitlab-mcp-server",
      "containerPort": 8080
    }
  ],
  "healthCheckGracePeriodSeconds": 120,
  "schedulingStrategy": "REPLICA",
  "enableECSManagedTags": true,
  "propagateTags": "SERVICE",
  "enableExecuteCommand": true,
  "tags": [
    {
      "key": "Application",
      "value": "mcp-servers"
    }
  ]
}
```

#### Create Service via AWS CLI

```bash
aws ecs create-service \
  --cluster mcp-servers-cluster \
  --service-name gitlab-mcp-service \
  --task-definition gitlab-mcp-server:1 \
  --desired-count 2 \
  --launch-type FARGATE \
  --platform-version LATEST \
  --network-configuration "awsvpcConfiguration={subnets=[subnet-private-1a,subnet-private-1b],securityGroups=[sg-ecs-tasks],assignPublicIp=DISABLED}" \
  --load-balancers "targetGroupArn=arn:aws:elasticloadbalancing:us-east-1:123456789012:targetgroup/gitlab-mcp-tg/1234567890123456,containerName=gitlab-mcp-server,containerPort=8080" \
  --health-check-grace-period-seconds 120 \
  --deployment-configuration "deploymentCircuitBreaker={enable=true,rollback=true},maximumPercent=200,minimumHealthyPercent=100" \
  --enable-execute-command \
  --tags key=Application,value=mcp-servers
```

---

### 4. Application Load Balancer

#### ALB Configuration

```json
{
  "Name": "mcp-servers-alb",
  "Scheme": "internet-facing",
  "Type": "application",
  "IpAddressType": "ipv4",
  "SecurityGroups": ["sg-alb"],
  "Subnets": [
    "subnet-public-1a",
    "subnet-public-1b",
    "subnet-public-1c"
  ],
  "Tags": [
    {
      "Key": "Application",
      "Value": "mcp-servers"
    }
  ]
}
```

#### Target Group Configuration

```json
{
  "Name": "gitlab-mcp-tg",
  "Protocol": "HTTP",
  "Port": 8080,
  "VpcId": "vpc-123456",
  "TargetType": "ip",
  "HealthCheckProtocol": "HTTP",
  "HealthCheckPath": "/health",
  "HealthCheckPort": "traffic-port",
  "HealthyThresholdCount": 2,
  "UnhealthyThresholdCount": 3,
  "HealthCheckTimeoutSeconds": 5,
  "HealthCheckIntervalSeconds": 30,
  "Matcher": {
    "HttpCode": "200"
  },
  "Tags": [
    {
      "Key": "Application",
      "Value": "mcp-servers"
    },
    {
      "Key": "Server",
      "Value": "gitlab"
    }
  ]
}
```

#### Listener Configuration with Path-Based Routing

```json
{
  "LoadBalancerArn": "arn:aws:elasticloadbalancing:us-east-1:123456789012:loadbalancer/app/mcp-servers-alb/1234567890123456",
  "Protocol": "HTTPS",
  "Port": 443,
  "Certificates": [
    {
      "CertificateArn": "arn:aws:acm:us-east-1:123456789012:certificate/12345678-1234-1234-1234-123456789012"
    }
  ],
  "SslPolicy": "ELBSecurityPolicy-TLS13-1-2-2021-06",
  "DefaultActions": [
    {
      "Type": "fixed-response",
      "FixedResponseConfig": {
        "StatusCode": "404",
        "ContentType": "application/json",
        "MessageBody": "{\"error\":\"Not Found\"}"
      }
    }
  ]
}
```

#### Listener Rules for Each MCP Server

```bash
# GitLab rule
aws elbv2 create-rule \
  --listener-arn arn:aws:elasticloadbalancing:us-east-1:123456789012:listener/app/mcp-servers-alb/1234567890123456/1234567890123456 \
  --priority 10 \
  --conditions Field=path-pattern,Values='/gitlab*' \
  --actions Type=forward,TargetGroupArn=arn:aws:elasticloadbalancing:us-east-1:123456789012:targetgroup/gitlab-mcp-tg/1234567890123456

# Atlassian rule
aws elbv2 create-rule \
  --listener-arn arn:aws:elasticloadbalancing:us-east-1:123456789012:listener/app/mcp-servers-alb/1234567890123456/1234567890123456 \
  --priority 20 \
  --conditions Field=path-pattern,Values='/atlassian*' \
  --actions Type=forward,TargetGroupArn=arn:aws:elasticloadbalancing:us-east-1:123456789012:targetgroup/atlassian-mcp-tg/1234567890123456

# Dynatrace rule
aws elbv2 create-rule \
  --listener-arn arn:aws:elasticloadbalancing:us-east-1:123456789012:listener/app/mcp-servers-alb/1234567890123456/1234567890123456 \
  --priority 30 \
  --conditions Field=path-pattern,Values='/dynatrace*' \
  --actions Type=forward,TargetGroupArn=arn:aws:elasticloadbalancing:us-east-1:123456789012:targetgroup/dynatrace-mcp-tg/1234567890123456

# PagerDuty rule
aws elbv2 create-rule \
  --listener-arn arn:aws:elasticloadbalancing:us-east-1:123456789012:listener/app/mcp-servers-alb/1234567890123456/1234567890123456 \
  --priority 40 \
  --conditions Field=path-pattern,Values='/pagerduty*' \
  --actions Type=forward,TargetGroupArn=arn:aws:elasticloadbalancing:us-east-1:123456789012:targetgroup/pagerduty-mcp-tg/1234567890123456

# ServiceNow rule
aws elbv2 create-rule \
  --listener-arn arn:aws:elasticloadbalancing:us-east-1:123456789012:listener/app/mcp-servers-alb/1234567890123456/1234567890123456 \
  --priority 50 \
  --conditions Field=path-pattern,Values='/servicenow*' \
  --actions Type=forward,TargetGroupArn=arn:aws:elasticloadbalancing:us-east-1:123456789012:targetgroup/servicenow-mcp-tg/1234567890123456
```

#### HTTP to HTTPS Redirect

```bash
aws elbv2 create-listener \
  --load-balancer-arn arn:aws:elasticloadbalancing:us-east-1:123456789012:loadbalancer/app/mcp-servers-alb/1234567890123456 \
  --protocol HTTP \
  --port 80 \
  --default-actions Type=redirect,RedirectConfig="{Protocol=HTTPS,Port=443,StatusCode=HTTP_301}"
```

---

### 5. VPC and Networking

#### VPC Requirements

| Component | CIDR | Purpose |
|-----------|------|---------|
| VPC | 10.0.0.0/16 | Main VPC |
| Public Subnet 1a | 10.0.1.0/24 | ALB, NAT Gateway |
| Public Subnet 1b | 10.0.2.0/24 | ALB, NAT Gateway |
| Public Subnet 1c | 10.0.3.0/24 | ALB, NAT Gateway |
| Private Subnet 1a | 10.0.10.0/24 | ECS Tasks |
| Private Subnet 1b | 10.0.11.0/24 | ECS Tasks |
| Private Subnet 1c | 10.0.12.0/24 | ECS Tasks |

#### Security Group for ALB

```json
{
  "GroupName": "mcp-alb-sg",
  "Description": "Security group for MCP servers ALB",
  "VpcId": "vpc-123456",
  "SecurityGroupIngress": [
    {
      "IpProtocol": "tcp",
      "FromPort": 443,
      "ToPort": 443,
      "CidrIp": "0.0.0.0/0",
      "Description": "HTTPS from internet"
    },
    {
      "IpProtocol": "tcp",
      "FromPort": 80,
      "ToPort": 80,
      "CidrIp": "0.0.0.0/0",
      "Description": "HTTP redirect"
    }
  ],
  "SecurityGroupEgress": [
    {
      "IpProtocol": "tcp",
      "FromPort": 8080,
      "ToPort": 8080,
      "DestinationSecurityGroupId": "sg-ecs-tasks",
      "Description": "To ECS tasks"
    }
  ]
}
```

#### Security Group for ECS Tasks

```json
{
  "GroupName": "mcp-ecs-tasks-sg",
  "Description": "Security group for MCP ECS tasks",
  "VpcId": "vpc-123456",
  "SecurityGroupIngress": [
    {
      "IpProtocol": "tcp",
      "FromPort": 8080,
      "ToPort": 8080,
      "SourceSecurityGroupId": "sg-alb",
      "Description": "From ALB"
    }
  ],
  "SecurityGroupEgress": [
    {
      "IpProtocol": "tcp",
      "FromPort": 443,
      "ToPort": 443,
      "CidrIp": "0.0.0.0/0",
      "Description": "HTTPS to external APIs"
    }
  ]
}
```

#### NAT Gateway Configuration

```bash
# Create Elastic IP for NAT Gateway
aws ec2 allocate-address --domain vpc --tag-specifications 'ResourceType=elastic-ip,Tags=[{Key=Name,Value=mcp-nat-eip}]'

# Create NAT Gateway in public subnet
aws ec2 create-nat-gateway \
  --subnet-id subnet-public-1a \
  --allocation-id eipalloc-123456 \
  --tag-specifications 'ResourceType=natgateway,Tags=[{Key=Name,Value=mcp-nat-gateway}]'

# Update private route table
aws ec2 create-route \
  --route-table-id rtb-private \
  --destination-cidr-block 0.0.0.0/0 \
  --nat-gateway-id nat-123456
```

#### VPC Endpoints (Cost Optimization)

```bash
# ECR API endpoint
aws ec2 create-vpc-endpoint \
  --vpc-id vpc-123456 \
  --vpc-endpoint-type Interface \
  --service-name com.amazonaws.us-east-1.ecr.api \
  --subnet-ids subnet-private-1a subnet-private-1b \
  --security-group-ids sg-vpc-endpoints \
  --private-dns-enabled

# ECR DKR endpoint
aws ec2 create-vpc-endpoint \
  --vpc-id vpc-123456 \
  --vpc-endpoint-type Interface \
  --service-name com.amazonaws.us-east-1.ecr.dkr \
  --subnet-ids subnet-private-1a subnet-private-1b \
  --security-group-ids sg-vpc-endpoints \
  --private-dns-enabled

# S3 Gateway endpoint (for ECR layers)
aws ec2 create-vpc-endpoint \
  --vpc-id vpc-123456 \
  --vpc-endpoint-type Gateway \
  --service-name com.amazonaws.us-east-1.s3 \
  --route-table-ids rtb-private

# Secrets Manager endpoint
aws ec2 create-vpc-endpoint \
  --vpc-id vpc-123456 \
  --vpc-endpoint-type Interface \
  --service-name com.amazonaws.us-east-1.secretsmanager \
  --subnet-ids subnet-private-1a subnet-private-1b \
  --security-group-ids sg-vpc-endpoints \
  --private-dns-enabled

# CloudWatch Logs endpoint
aws ec2 create-vpc-endpoint \
  --vpc-id vpc-123456 \
  --vpc-endpoint-type Interface \
  --service-name com.amazonaws.us-east-1.logs \
  --subnet-ids subnet-private-1a subnet-private-1b \
  --security-group-ids sg-vpc-endpoints \
  --private-dns-enabled
```

---

### 6. Auto Scaling

#### Application Auto Scaling Registration

```bash
aws application-autoscaling register-scalable-target \
  --service-namespace ecs \
  --resource-id service/mcp-servers-cluster/gitlab-mcp-service \
  --scalable-dimension ecs:service:DesiredCount \
  --min-capacity 2 \
  --max-capacity 10
```

#### Target Tracking Policy - CPU

```json
{
  "ServiceNamespace": "ecs",
  "ResourceId": "service/mcp-servers-cluster/gitlab-mcp-service",
  "ScalableDimension": "ecs:service:DesiredCount",
  "PolicyName": "gitlab-cpu-scaling",
  "PolicyType": "TargetTrackingScaling",
  "TargetTrackingScalingPolicyConfiguration": {
    "TargetValue": 70.0,
    "PredefinedMetricSpecification": {
      "PredefinedMetricType": "ECSServiceAverageCPUUtilization"
    },
    "ScaleOutCooldown": 60,
    "ScaleInCooldown": 300
  }
}
```

#### Target Tracking Policy - Memory

```json
{
  "ServiceNamespace": "ecs",
  "ResourceId": "service/mcp-servers-cluster/gitlab-mcp-service",
  "ScalableDimension": "ecs:service:DesiredCount",
  "PolicyName": "gitlab-memory-scaling",
  "PolicyType": "TargetTrackingScaling",
  "TargetTrackingScalingPolicyConfiguration": {
    "TargetValue": 80.0,
    "PredefinedMetricSpecification": {
      "PredefinedMetricType": "ECSServiceAverageMemoryUtilization"
    },
    "ScaleOutCooldown": 60,
    "ScaleInCooldown": 300
  }
}
```

#### Apply Scaling Policies

```bash
# CPU scaling policy
aws application-autoscaling put-scaling-policy \
  --service-namespace ecs \
  --resource-id service/mcp-servers-cluster/gitlab-mcp-service \
  --scalable-dimension ecs:service:DesiredCount \
  --policy-name gitlab-cpu-scaling \
  --policy-type TargetTrackingScaling \
  --target-tracking-scaling-policy-configuration file://cpu-scaling-policy.json

# Memory scaling policy
aws application-autoscaling put-scaling-policy \
  --service-namespace ecs \
  --resource-id service/mcp-servers-cluster/gitlab-mcp-service \
  --scalable-dimension ecs:service:DesiredCount \
  --policy-name gitlab-memory-scaling \
  --policy-type TargetTrackingScaling \
  --target-tracking-scaling-policy-configuration file://memory-scaling-policy.json
```

---

### 7. IAM Roles

#### Task Execution Role

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {
        "Service": "ecs-tasks.amazonaws.com"
      },
      "Action": "sts:AssumeRole"
    }
  ]
}
```

#### Task Execution Role Policy

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "ECRAccess",
      "Effect": "Allow",
      "Action": [
        "ecr:GetAuthorizationToken",
        "ecr:BatchCheckLayerAvailability",
        "ecr:GetDownloadUrlForLayer",
        "ecr:BatchGetImage"
      ],
      "Resource": "*"
    },
    {
      "Sid": "CloudWatchLogs",
      "Effect": "Allow",
      "Action": [
        "logs:CreateLogGroup",
        "logs:CreateLogStream",
        "logs:PutLogEvents"
      ],
      "Resource": "arn:aws:logs:*:*:log-group:/ecs/mcp-servers/*"
    },
    {
      "Sid": "SecretsManagerAccess",
      "Effect": "Allow",
      "Action": [
        "secretsmanager:GetSecretValue"
      ],
      "Resource": "arn:aws:secretsmanager:*:*:secret:mcp/*"
    },
    {
      "Sid": "SSMParameterAccess",
      "Effect": "Allow",
      "Action": [
        "ssm:GetParameters",
        "ssm:GetParameter"
      ],
      "Resource": "arn:aws:ssm:*:*:parameter/mcp/*"
    }
  ]
}
```

#### Task Role (Application Permissions)

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "SecretsManagerReadAccess",
      "Effect": "Allow",
      "Action": [
        "secretsmanager:GetSecretValue",
        "secretsmanager:DescribeSecret"
      ],
      "Resource": "arn:aws:secretsmanager:*:*:secret:mcp/*"
    },
    {
      "Sid": "SSMExec",
      "Effect": "Allow",
      "Action": [
        "ssmmessages:CreateControlChannel",
        "ssmmessages:CreateDataChannel",
        "ssmmessages:OpenControlChannel",
        "ssmmessages:OpenDataChannel"
      ],
      "Resource": "*"
    },
    {
      "Sid": "XRayAccess",
      "Effect": "Allow",
      "Action": [
        "xray:PutTraceSegments",
        "xray:PutTelemetryRecords"
      ],
      "Resource": "*"
    }
  ]
}
```

#### Create Roles via AWS CLI

```bash
# Create execution role
aws iam create-role \
  --role-name mcp-ecs-execution-role \
  --assume-role-policy-document file://ecs-task-trust-policy.json

aws iam put-role-policy \
  --role-name mcp-ecs-execution-role \
  --policy-name mcp-ecs-execution-policy \
  --policy-document file://ecs-execution-policy.json

# Create task role
aws iam create-role \
  --role-name mcp-ecs-task-role \
  --assume-role-policy-document file://ecs-task-trust-policy.json

aws iam put-role-policy \
  --role-name mcp-ecs-task-role \
  --policy-name mcp-ecs-task-policy \
  --policy-document file://ecs-task-policy.json
```

---

### 8. CloudFormation Template for ECS

```yaml
AWSTemplateFormatVersion: '2010-09-09'
Description: MCP Server ECS Fargate Deployment

Parameters:
  Environment:
    Type: String
    Default: production
    AllowedValues:
      - development
      - staging
      - production

  VpcId:
    Type: AWS::EC2::VPC::Id
    Description: VPC ID

  PublicSubnets:
    Type: List<AWS::EC2::Subnet::Id>
    Description: Public subnets for ALB

  PrivateSubnets:
    Type: List<AWS::EC2::Subnet::Id>
    Description: Private subnets for ECS tasks

  CertificateArn:
    Type: String
    Description: ACM Certificate ARN for HTTPS

  GitLabImageUri:
    Type: String
    Description: ECR Image URI for GitLab MCP server

  GitLabSecretArn:
    Type: String
    Description: Secrets Manager ARN for GitLab credentials

Resources:
  # ECS Cluster
  ECSCluster:
    Type: AWS::ECS::Cluster
    Properties:
      ClusterName: !Sub mcp-servers-cluster-${Environment}
      ClusterSettings:
        - Name: containerInsights
          Value: enabled
      CapacityProviders:
        - FARGATE
        - FARGATE_SPOT
      DefaultCapacityProviderStrategy:
        - CapacityProvider: FARGATE
          Weight: 1
          Base: 1
        - CapacityProvider: FARGATE_SPOT
          Weight: 3
      Tags:
        - Key: Application
          Value: mcp-servers
        - Key: Environment
          Value: !Ref Environment

  # CloudWatch Log Group
  LogGroup:
    Type: AWS::Logs::LogGroup
    Properties:
      LogGroupName: !Sub /ecs/mcp-servers/${Environment}
      RetentionInDays: 30

  # ALB Security Group
  ALBSecurityGroup:
    Type: AWS::EC2::SecurityGroup
    Properties:
      GroupName: !Sub mcp-alb-sg-${Environment}
      GroupDescription: Security group for MCP ALB
      VpcId: !Ref VpcId
      SecurityGroupIngress:
        - IpProtocol: tcp
          FromPort: 443
          ToPort: 443
          CidrIp: 0.0.0.0/0
          Description: HTTPS
        - IpProtocol: tcp
          FromPort: 80
          ToPort: 80
          CidrIp: 0.0.0.0/0
          Description: HTTP redirect
      Tags:
        - Key: Name
          Value: !Sub mcp-alb-sg-${Environment}

  # ECS Tasks Security Group
  ECSSecurityGroup:
    Type: AWS::EC2::SecurityGroup
    Properties:
      GroupName: !Sub mcp-ecs-sg-${Environment}
      GroupDescription: Security group for MCP ECS tasks
      VpcId: !Ref VpcId
      SecurityGroupIngress:
        - IpProtocol: tcp
          FromPort: 8080
          ToPort: 8080
          SourceSecurityGroupId: !Ref ALBSecurityGroup
          Description: From ALB
      SecurityGroupEgress:
        - IpProtocol: tcp
          FromPort: 443
          ToPort: 443
          CidrIp: 0.0.0.0/0
          Description: HTTPS outbound
      Tags:
        - Key: Name
          Value: !Sub mcp-ecs-sg-${Environment}

  # Application Load Balancer
  ApplicationLoadBalancer:
    Type: AWS::ElasticLoadBalancingV2::LoadBalancer
    Properties:
      Name: !Sub mcp-alb-${Environment}
      Scheme: internet-facing
      Type: application
      SecurityGroups:
        - !Ref ALBSecurityGroup
      Subnets: !Ref PublicSubnets
      Tags:
        - Key: Application
          Value: mcp-servers

  # HTTPS Listener
  HTTPSListener:
    Type: AWS::ElasticLoadBalancingV2::Listener
    Properties:
      LoadBalancerArn: !Ref ApplicationLoadBalancer
      Protocol: HTTPS
      Port: 443
      Certificates:
        - CertificateArn: !Ref CertificateArn
      SslPolicy: ELBSecurityPolicy-TLS13-1-2-2021-06
      DefaultActions:
        - Type: fixed-response
          FixedResponseConfig:
            StatusCode: '404'
            ContentType: application/json
            MessageBody: '{"error":"Not Found"}'

  # HTTP to HTTPS Redirect
  HTTPListener:
    Type: AWS::ElasticLoadBalancingV2::Listener
    Properties:
      LoadBalancerArn: !Ref ApplicationLoadBalancer
      Protocol: HTTP
      Port: 80
      DefaultActions:
        - Type: redirect
          RedirectConfig:
            Protocol: HTTPS
            Port: '443'
            StatusCode: HTTP_301

  # Task Execution Role
  TaskExecutionRole:
    Type: AWS::IAM::Role
    Properties:
      RoleName: !Sub mcp-ecs-execution-role-${Environment}
      AssumeRolePolicyDocument:
        Version: '2012-10-17'
        Statement:
          - Effect: Allow
            Principal:
              Service: ecs-tasks.amazonaws.com
            Action: sts:AssumeRole
      ManagedPolicyArns:
        - arn:aws:iam::aws:policy/service-role/AmazonECSTaskExecutionRolePolicy
      Policies:
        - PolicyName: SecretsAccess
          PolicyDocument:
            Version: '2012-10-17'
            Statement:
              - Effect: Allow
                Action:
                  - secretsmanager:GetSecretValue
                Resource: !Sub arn:aws:secretsmanager:${AWS::Region}:${AWS::AccountId}:secret:mcp/*

  # Task Role
  TaskRole:
    Type: AWS::IAM::Role
    Properties:
      RoleName: !Sub mcp-ecs-task-role-${Environment}
      AssumeRolePolicyDocument:
        Version: '2012-10-17'
        Statement:
          - Effect: Allow
            Principal:
              Service: ecs-tasks.amazonaws.com
            Action: sts:AssumeRole
      Policies:
        - PolicyName: TaskPermissions
          PolicyDocument:
            Version: '2012-10-17'
            Statement:
              - Effect: Allow
                Action:
                  - secretsmanager:GetSecretValue
                Resource: !Sub arn:aws:secretsmanager:${AWS::Region}:${AWS::AccountId}:secret:mcp/*
              - Effect: Allow
                Action:
                  - ssmmessages:CreateControlChannel
                  - ssmmessages:CreateDataChannel
                  - ssmmessages:OpenControlChannel
                  - ssmmessages:OpenDataChannel
                Resource: '*'

  # GitLab Target Group
  GitLabTargetGroup:
    Type: AWS::ElasticLoadBalancingV2::TargetGroup
    Properties:
      Name: !Sub gitlab-mcp-tg-${Environment}
      Protocol: HTTP
      Port: 8080
      VpcId: !Ref VpcId
      TargetType: ip
      HealthCheckProtocol: HTTP
      HealthCheckPath: /health
      HealthCheckIntervalSeconds: 30
      HealthCheckTimeoutSeconds: 5
      HealthyThresholdCount: 2
      UnhealthyThresholdCount: 3
      Tags:
        - Key: Application
          Value: mcp-servers

  # GitLab Listener Rule
  GitLabListenerRule:
    Type: AWS::ElasticLoadBalancingV2::ListenerRule
    Properties:
      ListenerArn: !Ref HTTPSListener
      Priority: 10
      Conditions:
        - Field: path-pattern
          Values:
            - /gitlab*
      Actions:
        - Type: forward
          TargetGroupArn: !Ref GitLabTargetGroup

  # GitLab Task Definition
  GitLabTaskDefinition:
    Type: AWS::ECS::TaskDefinition
    Properties:
      Family: !Sub gitlab-mcp-server-${Environment}
      NetworkMode: awsvpc
      RequiresCompatibilities:
        - FARGATE
      Cpu: '256'
      Memory: '512'
      ExecutionRoleArn: !GetAtt TaskExecutionRole.Arn
      TaskRoleArn: !GetAtt TaskRole.Arn
      RuntimePlatform:
        CpuArchitecture: ARM64
        OperatingSystemFamily: LINUX
      ContainerDefinitions:
        - Name: gitlab-mcp-server
          Image: !Ref GitLabImageUri
          Essential: true
          PortMappings:
            - ContainerPort: 8080
              Protocol: tcp
          Environment:
            - Name: PORT
              Value: '8080'
            - Name: LOG_LEVEL
              Value: info
            - Name: ENVIRONMENT
              Value: !Ref Environment
            - Name: SERVER_NAME
              Value: gitlab
          Secrets:
            - Name: GITLAB_TOKEN
              ValueFrom: !Sub ${GitLabSecretArn}:token::
          HealthCheck:
            Command:
              - CMD-SHELL
              - curl -f http://localhost:8080/health || exit 1
            Interval: 30
            Timeout: 5
            Retries: 3
            StartPeriod: 60
          LogConfiguration:
            LogDriver: awslogs
            Options:
              awslogs-group: !Ref LogGroup
              awslogs-region: !Ref AWS::Region
              awslogs-stream-prefix: gitlab
      Tags:
        - Key: Application
          Value: mcp-servers

  # GitLab Service
  GitLabService:
    Type: AWS::ECS::Service
    DependsOn: GitLabListenerRule
    Properties:
      ServiceName: !Sub gitlab-mcp-service-${Environment}
      Cluster: !Ref ECSCluster
      TaskDefinition: !Ref GitLabTaskDefinition
      DesiredCount: 2
      LaunchType: FARGATE
      PlatformVersion: LATEST
      NetworkConfiguration:
        AwsvpcConfiguration:
          Subnets: !Ref PrivateSubnets
          SecurityGroups:
            - !Ref ECSSecurityGroup
          AssignPublicIp: DISABLED
      LoadBalancers:
        - TargetGroupArn: !Ref GitLabTargetGroup
          ContainerName: gitlab-mcp-server
          ContainerPort: 8080
      HealthCheckGracePeriodSeconds: 120
      DeploymentConfiguration:
        DeploymentCircuitBreaker:
          Enable: true
          Rollback: true
        MaximumPercent: 200
        MinimumHealthyPercent: 100
      EnableExecuteCommand: true
      Tags:
        - Key: Application
          Value: mcp-servers

  # Auto Scaling Target
  ScalableTarget:
    Type: AWS::ApplicationAutoScaling::ScalableTarget
    Properties:
      ServiceNamespace: ecs
      ResourceId: !Sub service/${ECSCluster}/${GitLabService.Name}
      ScalableDimension: ecs:service:DesiredCount
      MinCapacity: 2
      MaxCapacity: 10
      RoleARN: !Sub arn:aws:iam::${AWS::AccountId}:role/aws-service-role/ecs.application-autoscaling.amazonaws.com/AWSServiceRoleForApplicationAutoScaling_ECSService

  # CPU Scaling Policy
  CPUScalingPolicy:
    Type: AWS::ApplicationAutoScaling::ScalingPolicy
    Properties:
      PolicyName: !Sub gitlab-cpu-scaling-${Environment}
      PolicyType: TargetTrackingScaling
      ScalingTargetId: !Ref ScalableTarget
      TargetTrackingScalingPolicyConfiguration:
        TargetValue: 70.0
        PredefinedMetricSpecification:
          PredefinedMetricType: ECSServiceAverageCPUUtilization
        ScaleOutCooldown: 60
        ScaleInCooldown: 300

  # Memory Scaling Policy
  MemoryScalingPolicy:
    Type: AWS::ApplicationAutoScaling::ScalingPolicy
    Properties:
      PolicyName: !Sub gitlab-memory-scaling-${Environment}
      PolicyType: TargetTrackingScaling
      ScalingTargetId: !Ref ScalableTarget
      TargetTrackingScalingPolicyConfiguration:
        TargetValue: 80.0
        PredefinedMetricSpecification:
          PredefinedMetricType: ECSServiceAverageMemoryUtilization
        ScaleOutCooldown: 60
        ScaleInCooldown: 300

Outputs:
  ClusterArn:
    Description: ECS Cluster ARN
    Value: !GetAtt ECSCluster.Arn
    Export:
      Name: !Sub ${AWS::StackName}-ClusterArn

  ALBDNSName:
    Description: ALB DNS Name
    Value: !GetAtt ApplicationLoadBalancer.DNSName
    Export:
      Name: !Sub ${AWS::StackName}-ALBDNSName

  ALBHostedZoneId:
    Description: ALB Hosted Zone ID
    Value: !GetAtt ApplicationLoadBalancer.CanonicalHostedZoneID
    Export:
      Name: !Sub ${AWS::StackName}-ALBHostedZoneId

  GitLabServiceName:
    Description: GitLab Service Name
    Value: !GetAtt GitLabService.Name
    Export:
      Name: !Sub ${AWS::StackName}-GitLabServiceName
```

#### Deploy ECS CloudFormation Stack

```bash
aws cloudformation deploy \
  --stack-name mcp-servers-ecs \
  --template-file ecs-infrastructure.yaml \
  --parameter-overrides \
    Environment=production \
    VpcId=vpc-123456 \
    PublicSubnets=subnet-pub-1a,subnet-pub-1b \
    PrivateSubnets=subnet-priv-1a,subnet-priv-1b \
    CertificateArn=arn:aws:acm:us-east-1:123456789012:certificate/12345678-1234-1234-1234-123456789012 \
    GitLabImageUri=123456789012.dkr.ecr.us-east-1.amazonaws.com/mcp-servers/gitlab:latest \
    GitLabSecretArn=arn:aws:secretsmanager:us-east-1:123456789012:secret:mcp/gitlab-token-AbCdEf \
  --capabilities CAPABILITY_NAMED_IAM \
  --region us-east-1
```

---

### 6. Multi-Server ECS Fargate Deployment

Deploy multiple MCP servers in a single ECS task using supervisor to manage processes.

#### Multi-Server Task Definition

```json
{
  "family": "mcp-multi-server",
  "networkMode": "awsvpc",
  "requiresCompatibilities": ["FARGATE"],
  "cpu": "1024",
  "memory": "2048",
  "executionRoleArn": "arn:aws:iam::123456789012:role/mcp-ecs-execution-role",
  "taskRoleArn": "arn:aws:iam::123456789012:role/mcp-ecs-task-role",
  "containerDefinitions": [
    {
      "name": "mcp-multi-server",
      "image": "123456789012.dkr.ecr.us-east-1.amazonaws.com/mcp-servers/multi:latest",
      "essential": true,
      "portMappings": [
        { "containerPort": 8081, "protocol": "tcp", "name": "gitlab" },
        { "containerPort": 8082, "protocol": "tcp", "name": "atlassian" },
        { "containerPort": 8083, "protocol": "tcp", "name": "dynatrace" },
        { "containerPort": 8084, "protocol": "tcp", "name": "pagerduty" },
        { "containerPort": 8085, "protocol": "tcp", "name": "servicenow" }
      ],
      "environment": [
        { "name": "LOG_LEVEL", "value": "info" }
      ],
      "secrets": [
        {
          "name": "GITLAB_TOKEN",
          "valueFrom": "arn:aws:secretsmanager:us-east-1:123456789012:secret:mcp/gitlab:token::"
        },
        {
          "name": "ATLASSIAN_TOKEN",
          "valueFrom": "arn:aws:secretsmanager:us-east-1:123456789012:secret:mcp/atlassian:token::"
        },
        {
          "name": "DYNATRACE_TOKEN",
          "valueFrom": "arn:aws:secretsmanager:us-east-1:123456789012:secret:mcp/dynatrace:token::"
        },
        {
          "name": "PAGERDUTY_TOKEN",
          "valueFrom": "arn:aws:secretsmanager:us-east-1:123456789012:secret:mcp/pagerduty:token::"
        },
        {
          "name": "SERVICENOW_PASSWORD",
          "valueFrom": "arn:aws:secretsmanager:us-east-1:123456789012:secret:mcp/servicenow:password::"
        }
      ],
      "healthCheck": {
        "command": [
          "CMD-SHELL",
          "wget -q --spider http://localhost:8081/health && wget -q --spider http://localhost:8082/health && wget -q --spider http://localhost:8083/health && wget -q --spider http://localhost:8084/health && wget -q --spider http://localhost:8085/health"
        ],
        "interval": 30,
        "timeout": 10,
        "retries": 3,
        "startPeriod": 60
      },
      "logConfiguration": {
        "logDriver": "awslogs",
        "options": {
          "awslogs-group": "/ecs/mcp-multi-server",
          "awslogs-region": "us-east-1",
          "awslogs-stream-prefix": "mcp"
        }
      }
    }
  ]
}
```

#### ALB Configuration for Multi-Server

```hcl
# Target groups for each server port
resource "aws_lb_target_group" "servers" {
  for_each = {
    gitlab     = 8081
    atlassian  = 8082
    dynatrace  = 8083
    pagerduty  = 8084
    servicenow = 8085
  }

  name        = "mcp-${each.key}-tg"
  port        = each.value
  protocol    = "HTTP"
  vpc_id      = var.vpc_id
  target_type = "ip"

  health_check {
    path                = "/health"
    port                = each.value
    healthy_threshold   = 2
    unhealthy_threshold = 3
    timeout             = 5
    interval            = 30
    matcher             = "200"
  }

  tags = {
    Server = each.key
  }
}

# Listener rules for path-based routing
resource "aws_lb_listener_rule" "servers" {
  for_each = {
    gitlab     = { priority = 100, path = "/gitlab/*" }
    atlassian  = { priority = 101, path = "/atlassian/*" }
    dynatrace  = { priority = 102, path = "/dynatrace/*" }
    pagerduty  = { priority = 103, path = "/pagerduty/*" }
    servicenow = { priority = 104, path = "/servicenow/*" }
  }

  listener_arn = aws_lb_listener.https.arn
  priority     = each.value.priority

  action {
    type             = "forward"
    target_group_arn = aws_lb_target_group.servers[each.key].arn
  }

  condition {
    path_pattern {
      values = [each.value.path]
    }
  }
}

# ECS Service with multiple target groups
resource "aws_ecs_service" "multi_server" {
  name            = "mcp-multi-server"
  cluster         = aws_ecs_cluster.main.id
  task_definition = aws_ecs_task_definition.multi_server.arn
  desired_count   = 2
  launch_type     = "FARGATE"

  network_configuration {
    subnets          = var.private_subnets
    security_groups  = [aws_security_group.ecs.id]
    assign_public_ip = false
  }

  # Register with all target groups
  dynamic "load_balancer" {
    for_each = {
      gitlab     = 8081
      atlassian  = 8082
      dynatrace  = 8083
      pagerduty  = 8084
      servicenow = 8085
    }

    content {
      target_group_arn = aws_lb_target_group.servers[load_balancer.key].arn
      container_name   = "mcp-multi-server"
      container_port   = load_balancer.value
    }
  }
}
```

#### Multi-Server ECS Endpoints

| Endpoint | Port | Path |
|----------|------|------|
| GitLab | 8081 | `/gitlab/mcp` |
| Atlassian | 8082 | `/atlassian/mcp` |
| Dynatrace | 8083 | `/dynatrace/mcp` |
| PagerDuty | 8084 | `/pagerduty/mcp` |
| ServiceNow | 8085 | `/servicenow/mcp` |

#### Cost Comparison (ECS Fargate)

| Configuration | Monthly Cost (2 tasks) | Notes |
|--------------|----------------------|-------|
| 5 separate services | ~$150-200 | 10 tasks total (2 per service) |
| 1 multi-server service | ~$40-60 | 2 tasks with 1 vCPU, 2GB each |

---

## Section C: Shared Infrastructure

### 1. ECR Repositories

#### Create Repositories

```bash
#!/bin/bash

SERVERS=("gitlab" "atlassian" "dynatrace" "pagerduty" "servicenow")
ACCOUNT_ID="123456789012"
REGION="us-east-1"

for SERVER in "${SERVERS[@]}"; do
  aws ecr create-repository \
    --repository-name "mcp-servers/${SERVER}" \
    --image-scanning-configuration scanOnPush=true \
    --encryption-configuration encryptionType=AES256 \
    --image-tag-mutability IMMUTABLE \
    --tags Key=Application,Value=mcp-servers Key=Server,Value=${SERVER}

  echo "Created repository: mcp-servers/${SERVER}"
done
```

#### Lifecycle Policy

```json
{
  "rules": [
    {
      "rulePriority": 1,
      "description": "Keep last 10 production images",
      "selection": {
        "tagStatus": "tagged",
        "tagPrefixList": ["v", "release-"],
        "countType": "imageCountMoreThan",
        "countNumber": 10
      },
      "action": {
        "type": "expire"
      }
    },
    {
      "rulePriority": 2,
      "description": "Keep last 5 staging images",
      "selection": {
        "tagStatus": "tagged",
        "tagPrefixList": ["staging-"],
        "countType": "imageCountMoreThan",
        "countNumber": 5
      },
      "action": {
        "type": "expire"
      }
    },
    {
      "rulePriority": 3,
      "description": "Delete untagged images older than 1 day",
      "selection": {
        "tagStatus": "untagged",
        "countType": "sinceImagePushed",
        "countUnit": "days",
        "countNumber": 1
      },
      "action": {
        "type": "expire"
      }
    },
    {
      "rulePriority": 4,
      "description": "Delete dev images older than 7 days",
      "selection": {
        "tagStatus": "tagged",
        "tagPrefixList": ["dev-", "pr-"],
        "countType": "sinceImagePushed",
        "countUnit": "days",
        "countNumber": 7
      },
      "action": {
        "type": "expire"
      }
    }
  ]
}
```

#### Apply Lifecycle Policy

```bash
for SERVER in gitlab atlassian dynatrace pagerduty servicenow; do
  aws ecr put-lifecycle-policy \
    --repository-name "mcp-servers/${SERVER}" \
    --lifecycle-policy-text file://lifecycle-policy.json
done
```

#### ECR Repository Policy (Cross-Account Access)

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "AllowPull",
      "Effect": "Allow",
      "Principal": {
        "AWS": [
          "arn:aws:iam::123456789012:root",
          "arn:aws:iam::987654321098:root"
        ]
      },
      "Action": [
        "ecr:GetDownloadUrlForLayer",
        "ecr:BatchGetImage",
        "ecr:BatchCheckLayerAvailability"
      ]
    }
  ]
}
```

---

### 2. Route 53 / DNS

#### Create Hosted Zone

```bash
aws route53 create-hosted-zone \
  --name mcp.example.com \
  --caller-reference "$(date +%s)" \
  --hosted-zone-config Comment="MCP Servers DNS"
```

#### A Record for ALB (Alias)

```json
{
  "Comment": "Create A record for MCP API",
  "Changes": [
    {
      "Action": "UPSERT",
      "ResourceRecordSet": {
        "Name": "api.mcp.example.com",
        "Type": "A",
        "AliasTarget": {
          "HostedZoneId": "Z35SXDOTRQ7X7K",
          "DNSName": "mcp-alb-123456789.us-east-1.elb.amazonaws.com",
          "EvaluateTargetHealth": true
        }
      }
    }
  ]
}
```

#### A Record for API Gateway

```json
{
  "Comment": "Create A record for Lambda API Gateway",
  "Changes": [
    {
      "Action": "UPSERT",
      "ResourceRecordSet": {
        "Name": "lambda.mcp.example.com",
        "Type": "A",
        "AliasTarget": {
          "HostedZoneId": "Z1UJRXOUMOOFQ8",
          "DNSName": "d-abc123def4.execute-api.us-east-1.amazonaws.com",
          "EvaluateTargetHealth": false
        }
      }
    }
  ]
}
```

#### Apply DNS Changes

```bash
aws route53 change-resource-record-sets \
  --hosted-zone-id Z1234567890ABC \
  --change-batch file://dns-records.json
```

#### Health Check for Failover

```bash
aws route53 create-health-check \
  --caller-reference "$(date +%s)" \
  --health-check-config '{
    "IPAddress": "",
    "Port": 443,
    "Type": "HTTPS",
    "ResourcePath": "/health",
    "FullyQualifiedDomainName": "api.mcp.example.com",
    "RequestInterval": 30,
    "FailureThreshold": 3,
    "EnableSNI": true
  }'
```

---

### 3. CI/CD Pipeline

#### Complete GitHub Actions Workflow

```yaml
name: MCP Server CI/CD

on:
  push:
    branches:
      - main
      - 'release/*'
    paths:
      - 'servers/**'
      - '.github/workflows/mcp-deploy.yml'
  pull_request:
    branches:
      - main
    paths:
      - 'servers/**'

env:
  AWS_REGION: us-east-1
  ECR_REGISTRY: ${{ secrets.AWS_ACCOUNT_ID }}.dkr.ecr.us-east-1.amazonaws.com
  SERVERS: gitlab atlassian dynatrace pagerduty servicenow

permissions:
  id-token: write
  contents: read

jobs:
  detect-changes:
    runs-on: ubuntu-latest
    outputs:
      matrix: ${{ steps.set-matrix.outputs.matrix }}
      has-changes: ${{ steps.set-matrix.outputs.has-changes }}
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0

      - name: Detect changed servers
        id: set-matrix
        run: |
          CHANGED_SERVERS=()

          if [[ "${{ github.event_name }}" == "push" && "${{ github.ref }}" == "refs/heads/main" ]]; then
            # On main, check what changed since last commit
            CHANGED_FILES=$(git diff --name-only HEAD~1 HEAD)
          else
            # On PR, check what changed vs main
            CHANGED_FILES=$(git diff --name-only origin/main...HEAD)
          fi

          for SERVER in gitlab atlassian dynatrace pagerduty servicenow; do
            if echo "$CHANGED_FILES" | grep -q "servers/${SERVER}/"; then
              CHANGED_SERVERS+=("$SERVER")
            fi
          done

          if [ ${#CHANGED_SERVERS[@]} -eq 0 ]; then
            echo "has-changes=false" >> $GITHUB_OUTPUT
            echo "matrix={\"server\":[]}" >> $GITHUB_OUTPUT
          else
            echo "has-changes=true" >> $GITHUB_OUTPUT
            MATRIX=$(printf '%s\n' "${CHANGED_SERVERS[@]}" | jq -R . | jq -s '{server: .}')
            echo "matrix=$MATRIX" >> $GITHUB_OUTPUT
          fi

  test:
    needs: detect-changes
    if: needs.detect-changes.outputs.has-changes == 'true'
    runs-on: ubuntu-latest
    strategy:
      matrix: ${{ fromJson(needs.detect-changes.outputs.matrix) }}
      fail-fast: false
    steps:
      - uses: actions/checkout@v4

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.22'
          cache-dependency-path: servers/${{ matrix.server }}/go.sum

      - name: Run tests
        working-directory: servers/${{ matrix.server }}
        run: |
          go mod download
          go test -v -race -coverprofile=coverage.out ./...

      - name: Upload coverage
        uses: codecov/codecov-action@v4
        with:
          files: servers/${{ matrix.server }}/coverage.out
          flags: ${{ matrix.server }}

      - name: Run linter
        uses: golangci/golangci-lint-action@v4
        with:
          working-directory: servers/${{ matrix.server }}
          version: latest

  build:
    needs: [detect-changes, test]
    if: needs.detect-changes.outputs.has-changes == 'true'
    runs-on: ubuntu-latest
    strategy:
      matrix: ${{ fromJson(needs.detect-changes.outputs.matrix) }}
      fail-fast: false
    outputs:
      image-tag: ${{ steps.meta.outputs.version }}
    steps:
      - uses: actions/checkout@v4

      - name: Configure AWS credentials
        uses: aws-actions/configure-aws-credentials@v4
        with:
          role-to-assume: arn:aws:iam::${{ secrets.AWS_ACCOUNT_ID }}:role/github-actions-role
          aws-region: ${{ env.AWS_REGION }}

      - name: Login to Amazon ECR
        id: login-ecr
        uses: aws-actions/amazon-ecr-login@v2

      - name: Set up Docker Buildx
        uses: docker/setup-buildx-action@v3

      - name: Extract metadata
        id: meta
        uses: docker/metadata-action@v5
        with:
          images: ${{ env.ECR_REGISTRY }}/mcp-servers/${{ matrix.server }}
          tags: |
            type=sha,prefix=
            type=ref,event=branch
            type=ref,event=pr
            type=semver,pattern={{version}}
            type=raw,value=latest,enable=${{ github.ref == 'refs/heads/main' }}

      - name: Build and push
        uses: docker/build-push-action@v5
        with:
          context: servers/${{ matrix.server }}
          platforms: linux/arm64
          push: ${{ github.event_name != 'pull_request' }}
          tags: ${{ steps.meta.outputs.tags }}
          labels: ${{ steps.meta.outputs.labels }}
          cache-from: type=gha
          cache-to: type=gha,mode=max
          build-args: |
            VERSION=${{ steps.meta.outputs.version }}
            BUILD_DATE=${{ github.event.head_commit.timestamp }}
            GIT_COMMIT=${{ github.sha }}

      - name: Scan image for vulnerabilities
        uses: aquasecurity/trivy-action@master
        with:
          image-ref: ${{ env.ECR_REGISTRY }}/mcp-servers/${{ matrix.server }}:${{ github.sha }}
          format: 'sarif'
          output: 'trivy-results.sarif'
          severity: 'CRITICAL,HIGH'

      - name: Upload Trivy scan results
        uses: github/codeql-action/upload-sarif@v3
        if: always()
        with:
          sarif_file: 'trivy-results.sarif'

  deploy-lambda:
    needs: [detect-changes, build]
    if: |
      needs.detect-changes.outputs.has-changes == 'true' &&
      github.ref == 'refs/heads/main' &&
      github.event_name == 'push'
    runs-on: ubuntu-latest
    environment: production
    strategy:
      matrix: ${{ fromJson(needs.detect-changes.outputs.matrix) }}
      max-parallel: 2
    steps:
      - uses: actions/checkout@v4

      - name: Configure AWS credentials
        uses: aws-actions/configure-aws-credentials@v4
        with:
          role-to-assume: arn:aws:iam::${{ secrets.AWS_ACCOUNT_ID }}:role/github-actions-deploy-role
          aws-region: ${{ env.AWS_REGION }}

      - name: Update Lambda function
        run: |
          aws lambda update-function-code \
            --function-name ${{ matrix.server }}-mcp-lambda \
            --image-uri ${{ env.ECR_REGISTRY }}/mcp-servers/${{ matrix.server }}:${{ github.sha }}

          # Wait for update to complete
          aws lambda wait function-updated \
            --function-name ${{ matrix.server }}-mcp-lambda

          # Publish new version
          VERSION=$(aws lambda publish-version \
            --function-name ${{ matrix.server }}-mcp-lambda \
            --description "Deployed from commit ${{ github.sha }}" \
            --query 'Version' --output text)

          echo "Published Lambda version: $VERSION"

          # Update alias to point to new version
          aws lambda update-alias \
            --function-name ${{ matrix.server }}-mcp-lambda \
            --name production \
            --function-version $VERSION

      - name: Verify deployment
        run: |
          # Test the Lambda function
          RESPONSE=$(aws lambda invoke \
            --function-name ${{ matrix.server }}-mcp-lambda:production \
            --payload '{"httpMethod":"POST","body":"{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"initialize\"}"}' \
            --cli-binary-format raw-in-base64-out \
            response.json)

          STATUS=$(echo $RESPONSE | jq -r '.StatusCode')
          if [ "$STATUS" != "200" ]; then
            echo "Lambda invocation failed with status: $STATUS"
            cat response.json
            exit 1
          fi

          echo "Lambda deployment verified successfully"

  deploy-ecs:
    needs: [detect-changes, build]
    if: |
      needs.detect-changes.outputs.has-changes == 'true' &&
      github.ref == 'refs/heads/main' &&
      github.event_name == 'push'
    runs-on: ubuntu-latest
    environment: production
    strategy:
      matrix: ${{ fromJson(needs.detect-changes.outputs.matrix) }}
      max-parallel: 2
    steps:
      - uses: actions/checkout@v4

      - name: Configure AWS credentials
        uses: aws-actions/configure-aws-credentials@v4
        with:
          role-to-assume: arn:aws:iam::${{ secrets.AWS_ACCOUNT_ID }}:role/github-actions-deploy-role
          aws-region: ${{ env.AWS_REGION }}

      - name: Download task definition
        run: |
          aws ecs describe-task-definition \
            --task-definition ${{ matrix.server }}-mcp-server \
            --query 'taskDefinition' > task-definition.json

      - name: Update task definition
        id: task-def
        uses: aws-actions/amazon-ecs-render-task-definition@v1
        with:
          task-definition: task-definition.json
          container-name: ${{ matrix.server }}-mcp-server
          image: ${{ env.ECR_REGISTRY }}/mcp-servers/${{ matrix.server }}:${{ github.sha }}

      - name: Deploy to ECS
        uses: aws-actions/amazon-ecs-deploy-task-definition@v1
        with:
          task-definition: ${{ steps.task-def.outputs.task-definition }}
          service: ${{ matrix.server }}-mcp-service
          cluster: mcp-servers-cluster
          wait-for-service-stability: true
          wait-for-minutes: 10

      - name: Verify deployment
        run: |
          # Get ALB DNS
          ALB_DNS=$(aws elbv2 describe-load-balancers \
            --names mcp-alb-production \
            --query 'LoadBalancers[0].DNSName' --output text)

          # Health check
          for i in {1..10}; do
            RESPONSE=$(curl -s -o /dev/null -w "%{http_code}" \
              "https://${ALB_DNS}/${{ matrix.server }}/health" || true)

            if [ "$RESPONSE" == "200" ]; then
              echo "Health check passed"
              exit 0
            fi

            echo "Attempt $i: Health check returned $RESPONSE, retrying..."
            sleep 10
          done

          echo "Health check failed after 10 attempts"
          exit 1

  notify:
    needs: [deploy-lambda, deploy-ecs]
    if: always()
    runs-on: ubuntu-latest
    steps:
      - name: Send Slack notification
        uses: slackapi/slack-github-action@v1.25.0
        with:
          payload: |
            {
              "text": "MCP Server Deployment ${{ needs.deploy-lambda.result == 'success' && needs.deploy-ecs.result == 'success' && '✅ Succeeded' || '❌ Failed' }}",
              "blocks": [
                {
                  "type": "header",
                  "text": {
                    "type": "plain_text",
                    "text": "MCP Server Deployment"
                  }
                },
                {
                  "type": "section",
                  "fields": [
                    {
                      "type": "mrkdwn",
                      "text": "*Repository:*\n${{ github.repository }}"
                    },
                    {
                      "type": "mrkdwn",
                      "text": "*Commit:*\n${{ github.sha }}"
                    },
                    {
                      "type": "mrkdwn",
                      "text": "*Lambda:*\n${{ needs.deploy-lambda.result }}"
                    },
                    {
                      "type": "mrkdwn",
                      "text": "*ECS:*\n${{ needs.deploy-ecs.result }}"
                    }
                  ]
                },
                {
                  "type": "actions",
                  "elements": [
                    {
                      "type": "button",
                      "text": {
                        "type": "plain_text",
                        "text": "View Run"
                      },
                      "url": "${{ github.server_url }}/${{ github.repository }}/actions/runs/${{ github.run_id }}"
                    }
                  ]
                }
              ]
            }
        env:
          SLACK_WEBHOOK_URL: ${{ secrets.SLACK_WEBHOOK_URL }}
          SLACK_WEBHOOK_TYPE: INCOMING_WEBHOOK
```

#### GitHub Actions IAM Role Trust Policy

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {
        "Federated": "arn:aws:iam::123456789012:oidc-provider/token.actions.githubusercontent.com"
      },
      "Action": "sts:AssumeRoleWithWebIdentity",
      "Condition": {
        "StringEquals": {
          "token.actions.githubusercontent.com:aud": "sts.amazonaws.com"
        },
        "StringLike": {
          "token.actions.githubusercontent.com:sub": "repo:your-org/mcp-servers:*"
        }
      }
    }
  ]
}
```

#### GitHub Actions Deploy Role Policy

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "ECRAccess",
      "Effect": "Allow",
      "Action": [
        "ecr:GetAuthorizationToken",
        "ecr:BatchCheckLayerAvailability",
        "ecr:GetDownloadUrlForLayer",
        "ecr:BatchGetImage",
        "ecr:PutImage",
        "ecr:InitiateLayerUpload",
        "ecr:UploadLayerPart",
        "ecr:CompleteLayerUpload"
      ],
      "Resource": "*"
    },
    {
      "Sid": "LambdaDeployment",
      "Effect": "Allow",
      "Action": [
        "lambda:UpdateFunctionCode",
        "lambda:PublishVersion",
        "lambda:UpdateAlias",
        "lambda:GetFunction",
        "lambda:InvokeFunction"
      ],
      "Resource": "arn:aws:lambda:*:*:function:*-mcp-lambda*"
    },
    {
      "Sid": "ECSDeployment",
      "Effect": "Allow",
      "Action": [
        "ecs:DescribeTaskDefinition",
        "ecs:RegisterTaskDefinition",
        "ecs:UpdateService",
        "ecs:DescribeServices",
        "ecs:ListTasks",
        "ecs:DescribeTasks"
      ],
      "Resource": "*"
    },
    {
      "Sid": "ECSPassRole",
      "Effect": "Allow",
      "Action": "iam:PassRole",
      "Resource": [
        "arn:aws:iam::*:role/mcp-ecs-*-role*"
      ]
    },
    {
      "Sid": "ELBAccess",
      "Effect": "Allow",
      "Action": [
        "elasticloadbalancing:DescribeLoadBalancers",
        "elasticloadbalancing:DescribeTargetGroups",
        "elasticloadbalancing:DescribeTargetHealth"
      ],
      "Resource": "*"
    }
  ]
}
```

---

## Quick Reference

### Deployment Commands

#### Lambda Deployment
```bash
# Build and push image
docker build -t 123456789012.dkr.ecr.us-east-1.amazonaws.com/mcp-servers/gitlab:latest .
aws ecr get-login-password --region us-east-1 | docker login --username AWS --password-stdin 123456789012.dkr.ecr.us-east-1.amazonaws.com
docker push 123456789012.dkr.ecr.us-east-1.amazonaws.com/mcp-servers/gitlab:latest

# Update Lambda
aws lambda update-function-code \
  --function-name gitlab-mcp-lambda \
  --image-uri 123456789012.dkr.ecr.us-east-1.amazonaws.com/mcp-servers/gitlab:latest
```

#### ECS Deployment
```bash
# Update service with new image
aws ecs update-service \
  --cluster mcp-servers-cluster \
  --service gitlab-mcp-service \
  --force-new-deployment

# Watch deployment
aws ecs wait services-stable \
  --cluster mcp-servers-cluster \
  --services gitlab-mcp-service
```

### Monitoring Commands

```bash
# View Lambda logs
aws logs tail /aws/lambda/gitlab-mcp-lambda --follow

# View ECS logs
aws logs tail /ecs/mcp-servers/production --follow

# Check Lambda metrics
aws cloudwatch get-metric-statistics \
  --namespace AWS/Lambda \
  --metric-name Invocations \
  --dimensions Name=FunctionName,Value=gitlab-mcp-lambda \
  --start-time $(date -u -d '1 hour ago' +%Y-%m-%dT%H:%M:%SZ) \
  --end-time $(date -u +%Y-%m-%dT%H:%M:%SZ) \
  --period 300 \
  --statistics Sum

# Check ECS service health
aws ecs describe-services \
  --cluster mcp-servers-cluster \
  --services gitlab-mcp-service \
  --query 'services[0].{running:runningCount,desired:desiredCount,pending:pendingCount}'
```

### Troubleshooting

```bash
# Lambda execution errors
aws logs filter-log-events \
  --log-group-name /aws/lambda/gitlab-mcp-lambda \
  --filter-pattern "ERROR"

# ECS task failures
aws ecs list-tasks \
  --cluster mcp-servers-cluster \
  --service-name gitlab-mcp-service \
  --desired-status STOPPED

# Get task stop reason
aws ecs describe-tasks \
  --cluster mcp-servers-cluster \
  --tasks arn:aws:ecs:us-east-1:123456789012:task/mcp-servers-cluster/abc123 \
  --query 'tasks[0].stoppedReason'

# ECS Exec into running container
aws ecs execute-command \
  --cluster mcp-servers-cluster \
  --task arn:aws:ecs:us-east-1:123456789012:task/mcp-servers-cluster/abc123 \
  --container gitlab-mcp-server \
  --command "/bin/sh" \
  --interactive
```

---

## Cost Optimization Tips

1. **Lambda**: Use arm64 architecture (20% cheaper), optimize memory settings
2. **ECS**: Use Fargate Spot for non-critical workloads (70% cheaper)
3. **VPC Endpoints**: Reduce NAT Gateway costs for ECR, Secrets Manager
4. **ECR Lifecycle**: Aggressive cleanup of old images
5. **CloudWatch Logs**: Set appropriate retention periods
6. **Reserved Capacity**: Consider Savings Plans for predictable workloads

---

*Last updated: 2024*
