package main

import (
	"fmt"
	"os"
	"time"

	"github.com/go-mcp-gitlab/go-mcp-gitlab/pkg/config"
	"github.com/go-mcp-gitlab/go-mcp-gitlab/pkg/gitlab"
	"github.com/go-mcp-gitlab/go-mcp-gitlab/pkg/logging"
	"github.com/go-mcp-gitlab/go-mcp-gitlab/pkg/mcp"
	"github.com/go-mcp-gitlab/go-mcp-gitlab/pkg/tools"
)

const (
	AppName = "go-mcp-gitlab"
	Version = "1.0.0"
)

// gitlabLoggerAdapter adapts logging.Logger to gitlab.Logger interface
type gitlabLoggerAdapter struct {
	logger *logging.Logger
}

func (a *gitlabLoggerAdapter) Access(method, endpoint string, statusCode int, duration time.Duration) {
	if a.logger != nil {
		a.logger.Access("API_CALL method=%s endpoint=%q status=%d duration=%s", method, endpoint, statusCode, duration)
	}
}

func (a *gitlabLoggerAdapter) Debug(msg string, args ...any) {
	if a.logger != nil {
		if len(args) > 0 {
			pairs := make([]string, 0, len(args)/2)
			for i := 0; i+1 < len(args); i += 2 {
				pairs = append(pairs, fmt.Sprintf("%v=%v", args[i], args[i+1]))
			}
			a.logger.Debug("%s %s", msg, joinStrings(pairs, " "))
		} else {
			a.logger.Debug(msg)
		}
	}
}

func (a *gitlabLoggerAdapter) Error(msg string, args ...any) {
	if a.logger != nil {
		if len(args) > 0 {
			pairs := make([]string, 0, len(args)/2)
			for i := 0; i+1 < len(args); i += 2 {
				pairs = append(pairs, fmt.Sprintf("%v=%v", args[i], args[i+1]))
			}
			a.logger.Error("%s %s", msg, joinStrings(pairs, " "))
		} else {
			a.logger.Error(msg)
		}
	}
}

func (a *gitlabLoggerAdapter) LogHTTPRequest(context string, req *gitlab.HTTPRequestInfo, secrets ...string) {
	if a.logger != nil && req != nil {
		loggingReq := &logging.HTTPRequestInfo{
			Method:  req.Method,
			URL:     req.URL,
			Headers: req.Headers,
			Body:    req.Body,
		}
		a.logger.LogHTTPRequest(context, loggingReq, secrets...)
	}
}

func (a *gitlabLoggerAdapter) LogHTTPResponse(context string, resp *gitlab.HTTPResponseInfo, duration time.Duration, secrets ...string) {
	if a.logger != nil && resp != nil {
		loggingResp := &logging.HTTPResponseInfo{
			StatusCode: resp.StatusCode,
			Headers:    resp.Headers,
			Body:       resp.Body,
		}
		a.logger.LogHTTPResponse(context, loggingResp, duration, secrets...)
	}
}

func (a *gitlabLoggerAdapter) LogHTTPError(context string, req *gitlab.HTTPRequestInfo, resp *gitlab.HTTPResponseInfo, err error, secrets ...string) {
	if a.logger != nil {
		var loggingReq *logging.HTTPRequestInfo
		var loggingResp *logging.HTTPResponseInfo

		if req != nil {
			loggingReq = &logging.HTTPRequestInfo{
				Method:  req.Method,
				URL:     req.URL,
				Headers: req.Headers,
				Body:    req.Body,
			}
		}

		if resp != nil {
			loggingResp = &logging.HTTPResponseInfo{
				StatusCode: resp.StatusCode,
				Headers:    resp.Headers,
				Body:       resp.Body,
			}
		}

		a.logger.LogHTTPError(context, loggingReq, loggingResp, err, secrets...)
	}
}

func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	result := strs[0]
	for _, s := range strs[1:] {
		result += sep + s
	}
	return result
}

func main() {
	// Load environment variables from ~/.mcp_env if it exists
	// This must happen before config loading so env vars are available
	logging.LoadEnvFile()

	// Load configuration (handles -version and -help flags internally)
	cfg, err := config.LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// If config is nil, -version or -help was shown, exit
	if cfg == nil {
		os.Exit(0)
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "Invalid configuration: %v\n", err)
		os.Exit(1)
	}

	// Initialize logger
	logger, err := logging.NewLogger(logging.Config{
		LogDir:          cfg.LogDir,
		AppName:         AppName,
		Level:           logging.ParseLogLevel(cfg.LogLevel),
		AddAppSubfolder: cfg.AddAppSubfolder,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Close()

	// Log startup information
	logger.LogStartup(logging.GetStartupInfo(
		Version,
		logging.ConfigValue{Value: cfg.LogDir, Source: convertSource(cfg.Sources["LogDir"])},
		logging.ConfigValue{Value: cfg.LogLevel, Source: convertSource(cfg.Sources["LogLevel"])},
		logging.ConfigValue{Value: cfg.GitLabAPIURL, Source: convertSource(cfg.Sources["GitLabAPIURL"])},
		logging.ConfigValue{Value: logging.MaskToken(cfg.GitLabToken), Source: convertSource(cfg.Sources["GitLabToken"])},
	))

	// Create GitLab client with logger adapter
	logAdapter := &gitlabLoggerAdapter{logger: logger}
	gitlabClient := gitlab.NewClient(cfg.GitLabAPIURL, cfg.GitLabToken, gitlab.WithLogger(logAdapter))
	logger.Info("GitLab client initialized: url=%s token_source=%s", cfg.GitLabAPIURL, cfg.TokenSource)

	// Set up the tools context
	tools.SetContext(gitlabClient, logger, cfg)

	// Create MCP server
	server := mcp.NewServer(AppName, Version)
	logger.Info("MCP server created: name=%s, version=%s", AppName, Version)

	// Register all tools
	tools.RegisterAllTools(server)
	logger.Info("Tools registered successfully")

	// Log enabled features
	features := cfg.GetEnabledFeatures()
	if len(features) > 0 {
		logger.Info("Enabled features: %v", features)
	}

	// Run the server
	logger.Info("Starting MCP server...")
	if err := server.Run(); err != nil {
		logger.Error("Server error: %v", err)
		logger.LogShutdown(fmt.Sprintf("error: %v", err))
		fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
		os.Exit(1)
	}

	logger.LogShutdown("normal exit")
}

// convertSource converts config.ConfigSource to logging.ConfigSource
func convertSource(src config.ConfigSource) logging.ConfigSource {
	switch src {
	case config.SourceFlag:
		return logging.SourceFlag
	case config.SourceEnvironment:
		return logging.SourceEnvironment
	default:
		return logging.SourceDefault
	}
}

