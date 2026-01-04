// Package config provides configuration management for go-mcp-gitlab.
// It handles CLI flags, environment variables, and tracks the source of each configuration value.
package config

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Version information (set at build time)
var (
	Version   = "dev"
	BuildTime = "unknown"
	GitCommit = "unknown"
)

// ConfigSource indicates where a configuration value originated from.
type ConfigSource string

const (
	SourceDefault     ConfigSource = "default"
	SourceEnvironment ConfigSource = "environment"
	SourceFlag        ConfigSource = "flag"
)

// Config holds all configuration settings for the GitLab MCP server.
type Config struct {
	// GitLab API
	GitLabAPIURL     string
	GitLabToken      string
	TokenSource      CredentialSource // Where the token was found

	// Project restrictions
	DefaultProjectID  string
	AllowedProjectIDs []string

	// Feature flags
	UsePipeline  bool
	UseMilestone bool
	UseWiki      bool
	ReadOnlyMode bool

	// Logging
	LogDir   string
	LogLevel string

	// Sources tracking - maps config key to its source
	Sources map[string]ConfigSource
}

// LoadConfig loads configuration from CLI flags and environment variables.
// CLI flags take precedence over environment variables, which take precedence over defaults.
// Returns the populated Config struct and any error encountered.
// If -version or -help flags are set, returns nil config with no error (caller should exit).
func LoadConfig() (*Config, error) {
	cfg := &Config{
		Sources: make(map[string]ConfigSource),
	}

	// Define CLI flags
	var (
		logDir     = flag.String("log-dir", "", "Log directory path")
		logLevel   = flag.String("log-level", "", "Log level: off, error, warn, info, access, debug")
		showVersion = flag.Bool("version", false, "Show version information")
		showHelp   = flag.Bool("help", false, "Show help message")
	)

	// Parse CLI flags
	flag.Parse()

	// Handle -version flag
	if *showVersion {
		fmt.Printf("go-mcp-gitlab version %s\n", Version)
		fmt.Printf("Build time: %s\n", BuildTime)
		fmt.Printf("Git commit: %s\n", GitCommit)
		return nil, nil
	}

	// Handle -help flag
	if *showHelp {
		printHelp()
		return nil, nil
	}

	// Load GitLab API URL
	cfg.GitLabAPIURL = cfg.loadString(
		"GitLabAPIURL",
		*new(string), // no flag for this
		"GITLAB_API_URL",
		"https://gitlab.com/api/v4",
	)

	// Load GitLab Token using credential resolver
	// This checks multiple sources: env vars, glab CLI, git credential, netrc
	gitlabHost := ExtractHostFromURL(cfg.GitLabAPIURL)
	credResult := ResolveGitLabToken(gitlabHost)
	cfg.GitLabToken = credResult.Token
	cfg.TokenSource = credResult.Source

	// Map credential source to config source for logging
	switch credResult.Source {
	case CredentialSourceEnv:
		cfg.Sources["GitLabToken"] = SourceEnvironment
	case CredentialSourceGlab, CredentialSourceGitCredential, CredentialSourceNetrc:
		cfg.Sources["GitLabToken"] = SourceDefault // Treat auto-discovered as "default" for simplicity
	default:
		cfg.Sources["GitLabToken"] = SourceDefault
	}

	// Load project restrictions
	cfg.DefaultProjectID = cfg.loadString(
		"DefaultProjectID",
		"",
		"GITLAB_PROJECT_ID",
		"",
	)

	// Load allowed project IDs (comma-separated)
	allowedProjectsStr := cfg.loadString(
		"AllowedProjectIDs",
		"",
		"GITLAB_ALLOWED_PROJECT_IDS",
		"",
	)
	if allowedProjectsStr != "" {
		cfg.AllowedProjectIDs = parseCommaSeparated(allowedProjectsStr)
	}

	// Load feature flags
	cfg.UsePipeline = cfg.loadBool(
		"UsePipeline",
		false,
		"USE_PIPELINE",
		false,
	)

	cfg.UseMilestone = cfg.loadBool(
		"UseMilestone",
		false,
		"USE_MILESTONE",
		false,
	)

	cfg.UseWiki = cfg.loadBool(
		"UseWiki",
		false,
		"USE_GITLAB_WIKI",
		false,
	)

	cfg.ReadOnlyMode = cfg.loadBool(
		"ReadOnlyMode",
		false,
		"GITLAB_READ_ONLY_MODE",
		false,
	)

	// Load logging configuration
	cfg.LogDir = ExpandPath(cfg.loadStringWithFlag(
		"LogDir",
		*logDir,
		"MCP_LOG_DIR",
		getDefaultLogDir(),
	))

	cfg.LogLevel = cfg.loadStringWithFlag(
		"LogLevel",
		*logLevel,
		"MCP_LOG_LEVEL",
		"info",
	)

	return cfg, nil
}

// loadString loads a string configuration value from environment variable or default.
// It tracks the source of the final value.
func (c *Config) loadString(key, flagVal, envVar, defaultVal string) string {
	// Flag takes precedence (but we don't have flags for most settings)
	if flagVal != "" {
		c.Sources[key] = SourceFlag
		return flagVal
	}

	// Environment variable takes precedence over default
	if envVal := os.Getenv(envVar); envVal != "" {
		c.Sources[key] = SourceEnvironment
		return envVal
	}

	// Use default
	c.Sources[key] = SourceDefault
	return defaultVal
}

// loadStringWithFlag loads a string configuration value with flag support.
func (c *Config) loadStringWithFlag(key, flagVal, envVar, defaultVal string) string {
	// Flag takes precedence
	if flagVal != "" {
		c.Sources[key] = SourceFlag
		return flagVal
	}

	// Environment variable takes precedence over default
	if envVal := os.Getenv(envVar); envVal != "" {
		c.Sources[key] = SourceEnvironment
		return envVal
	}

	// Use default
	c.Sources[key] = SourceDefault
	return defaultVal
}

// loadBool loads a boolean configuration value from environment variable or default.
func (c *Config) loadBool(key string, flagVal bool, envVar string, defaultVal bool) bool {
	// Environment variable takes precedence over default
	if envVal := os.Getenv(envVar); envVal != "" {
		c.Sources[key] = SourceEnvironment
		return parseBool(envVal)
	}

	// Use default
	c.Sources[key] = SourceDefault
	return defaultVal
}

// Validate checks that all required configuration fields are set.
// Returns an error describing any missing required fields.
func (c *Config) Validate() error {
	var errors []string

	if c.GitLabToken == "" {
		errors = append(errors, `GitLab token not found. Checked the following sources:
    1. Environment variables: GITLAB_PERSONAL_ACCESS_TOKEN, GITLAB_TOKEN, GITLAB_ACCESS_TOKEN, GL_TOKEN
    2. GitLab CLI (glab) config: ~/.config/glab-cli/config.yml
    3. Git credential helper: git credential fill
    4. Netrc file: ~/.netrc or ~/_netrc
  Please set GITLAB_PERSONAL_ACCESS_TOKEN or configure one of the above sources.`)
	}

	if c.GitLabAPIURL == "" {
		errors = append(errors, "GitLab API URL cannot be empty")
	}

	if len(errors) > 0 {
		return fmt.Errorf("configuration validation failed:\n  - %s", strings.Join(errors, "\n  - "))
	}

	return nil
}

// GetEnabledFeatures returns a list of enabled feature flag names.
func (c *Config) GetEnabledFeatures() []string {
	var features []string
	if c.UsePipeline {
		features = append(features, "pipeline")
	}
	if c.UseMilestone {
		features = append(features, "milestone")
	}
	if c.UseWiki {
		features = append(features, "wiki")
	}
	if c.ReadOnlyMode {
		features = append(features, "read-only")
	}
	return features
}

// IsProjectAllowed checks if a project ID is allowed based on the configuration.
// Returns true if:
// - No project restrictions are configured (AllowedProjectIDs is empty)
// - The project ID matches the DefaultProjectID
// - The project ID is in the AllowedProjectIDs list
func (c *Config) IsProjectAllowed(projectID string) bool {
	// If no restrictions configured, allow all
	if len(c.AllowedProjectIDs) == 0 && c.DefaultProjectID == "" {
		return true
	}

	// Check if it matches the default project
	if c.DefaultProjectID != "" && projectID == c.DefaultProjectID {
		return true
	}

	// Check if it's in the allowed list
	for _, allowed := range c.AllowedProjectIDs {
		if projectID == allowed {
			return true
		}
	}

	return false
}

// ExpandPath expands ~ to the user's home directory in file paths.
// This is necessary because ~ is a shell feature and is not automatically
// expanded when paths are passed via environment variables or config files.
func ExpandPath(path string) string {
	if path == "" {
		return path
	}
	if path == "~" {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return home
	}
	if len(path) > 1 && path[0] == '~' && (path[1] == '/' || path[1] == '\\') {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return filepath.Join(home, path[2:])
	}
	return path
}

// getDefaultLogDir returns the default log directory path.
func getDefaultLogDir() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return fmt.Sprintf("%s%cgo-mcp-gitlab%clogs", homeDir, os.PathSeparator, os.PathSeparator)
}

// parseCommaSeparated splits a comma-separated string into a slice of trimmed strings.
func parseCommaSeparated(s string) []string {
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

// parseBool converts a string to a boolean value.
// Accepts: "true", "1", "yes", "on" (case-insensitive) as true, everything else as false.
func parseBool(s string) bool {
	lower := strings.ToLower(strings.TrimSpace(s))
	switch lower {
	case "true", "1", "yes", "on":
		return true
	default:
		return false
	}
}

// printHelp prints the help message.
func printHelp() {
	fmt.Println("go-mcp-gitlab - GitLab MCP Server")
	fmt.Println()
	fmt.Println("Usage: go-mcp-gitlab [OPTIONS]")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  -log-dir <path>     Log directory (default: ~/go-mcp-gitlab/logs)")
	fmt.Println("  -log-level <level>  Log level: off, error, warn, info, access, debug (default: info)")
	fmt.Println("  -version            Show version information")
	fmt.Println("  -help               Show this help message")
	fmt.Println()
	fmt.Println("GitLab Token (checked in order):")
	fmt.Println("  1. Environment variables:")
	fmt.Println("     - GITLAB_PERSONAL_ACCESS_TOKEN")
	fmt.Println("     - GITLAB_TOKEN")
	fmt.Println("     - GITLAB_ACCESS_TOKEN")
	fmt.Println("     - GL_TOKEN")
	fmt.Println("  2. GitLab CLI (glab) config: ~/.config/glab-cli/config.yml")
	fmt.Println("  3. Git credential helper: git credential fill")
	fmt.Println("  4. Netrc file: ~/.netrc or ~/_netrc")
	fmt.Println()
	fmt.Println("Environment Variables:")
	fmt.Println("  GITLAB_API_URL                GitLab API URL (default: https://gitlab.com/api/v4)")
	fmt.Println("  GITLAB_PROJECT_ID             Default project ID")
	fmt.Println("  GITLAB_ALLOWED_PROJECT_IDS    Comma-separated list of allowed project IDs")
	fmt.Println("  USE_PIPELINE                  Enable pipeline tools (default: false)")
	fmt.Println("  USE_MILESTONE                 Enable milestone tools (default: false)")
	fmt.Println("  USE_GITLAB_WIKI               Enable wiki tools (default: false)")
	fmt.Println("  GITLAB_READ_ONLY_MODE         Enable read-only mode (default: false)")
	fmt.Println("  MCP_LOG_DIR                   Log directory path")
	fmt.Println("  MCP_LOG_LEVEL                 Log level")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  # Using environment variable")
	fmt.Println("  export GITLAB_PERSONAL_ACCESS_TOKEN=glpat-xxxxxxxxxxxx")
	fmt.Println("  go-mcp-gitlab")
	fmt.Println()
	fmt.Println("  # Using glab CLI (if already configured)")
	fmt.Println("  glab auth login  # Configure token once")
	fmt.Println("  go-mcp-gitlab    # Token auto-detected")
	fmt.Println()
	fmt.Println("  # Using git credential helper")
	fmt.Println("  git config --global credential.helper store")
	fmt.Println("  git clone https://gitlab.com/user/repo.git  # Saves credentials")
	fmt.Println("  go-mcp-gitlab  # Token auto-detected")
}
