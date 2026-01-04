package main

import (
	"fmt"
	"os"

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

	// Create GitLab client
	gitlabClient := gitlab.NewClient(cfg.GitLabAPIURL, cfg.GitLabToken)
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

