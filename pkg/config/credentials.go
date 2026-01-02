// Package config provides credential resolution for GitLab tokens.
// It checks multiple sources in priority order:
// 1. Environment variables (GITLAB_PERSONAL_ACCESS_TOKEN, GITLAB_TOKEN)
// 2. GitLab CLI (glab) config file
// 3. Git credential helper
// 4. .netrc / _netrc file
package config

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// CredentialSource indicates where a credential was found
type CredentialSource string

const (
	CredentialSourceEnv           CredentialSource = "environment"
	CredentialSourceGlab          CredentialSource = "glab-cli"
	CredentialSourceGitCredential CredentialSource = "git-credential"
	CredentialSourceNetrc         CredentialSource = "netrc"
	CredentialSourceNone          CredentialSource = "none"
)

// CredentialResult holds the resolved credential and its source
type CredentialResult struct {
	Token  string
	Source CredentialSource
}

// ResolveGitLabToken attempts to find a GitLab token from multiple sources.
// It checks sources in priority order and returns the first token found.
// The gitlabHost parameter should be the GitLab host (e.g., "gitlab.com")
func ResolveGitLabToken(gitlabHost string) CredentialResult {
	// 1. Check environment variables (highest priority)
	if token := getEnvToken(); token != "" {
		return CredentialResult{Token: token, Source: CredentialSourceEnv}
	}

	// 2. Check GitLab CLI (glab) config
	if token := getGlabToken(gitlabHost); token != "" {
		return CredentialResult{Token: token, Source: CredentialSourceGlab}
	}

	// 3. Check Git credential helper
	if token := getGitCredentialToken(gitlabHost); token != "" {
		return CredentialResult{Token: token, Source: CredentialSourceGitCredential}
	}

	// 4. Check .netrc / _netrc
	if token := getNetrcToken(gitlabHost); token != "" {
		return CredentialResult{Token: token, Source: CredentialSourceNetrc}
	}

	return CredentialResult{Source: CredentialSourceNone}
}

// getEnvToken checks environment variables for GitLab token
func getEnvToken() string {
	// Check multiple common environment variable names
	envVars := []string{
		"GITLAB_PERSONAL_ACCESS_TOKEN",
		"GITLAB_TOKEN",
		"GITLAB_ACCESS_TOKEN",
		"GL_TOKEN",
	}

	for _, envVar := range envVars {
		if token := os.Getenv(envVar); token != "" {
			return token
		}
	}

	return ""
}

// glabConfig represents the structure of glab CLI config file
type glabConfig struct {
	Hosts map[string]glabHost `yaml:"hosts"`
}

type glabHost struct {
	Token       string `yaml:"token"`
	GitProtocol string `yaml:"git_protocol"`
	APIProtocol string `yaml:"api_protocol"`
}

// getGlabToken reads token from GitLab CLI (glab) config file
func getGlabToken(gitlabHost string) string {
	configPaths := getGlabConfigPaths()

	for _, configPath := range configPaths {
		if token := readGlabConfig(configPath, gitlabHost); token != "" {
			return token
		}
	}

	return ""
}

// getGlabConfigPaths returns possible paths for glab config file
func getGlabConfigPaths() []string {
	var paths []string

	// XDG config directory (Linux/macOS)
	if xdgConfig := os.Getenv("XDG_CONFIG_HOME"); xdgConfig != "" {
		paths = append(paths, filepath.Join(xdgConfig, "glab-cli", "config.yml"))
	}

	// Home directory config
	if home, err := os.UserHomeDir(); err == nil {
		paths = append(paths, filepath.Join(home, ".config", "glab-cli", "config.yml"))
	}

	// Windows AppData
	if runtime.GOOS == "windows" {
		if appData := os.Getenv("APPDATA"); appData != "" {
			paths = append(paths, filepath.Join(appData, "glab-cli", "config.yml"))
		}
	}

	return paths
}

// readGlabConfig reads and parses a glab config file
func readGlabConfig(configPath, gitlabHost string) string {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return ""
	}

	var config glabConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return ""
	}

	// Try exact host match
	if host, ok := config.Hosts[gitlabHost]; ok && host.Token != "" {
		return host.Token
	}

	// Try without port
	hostWithoutPort := strings.Split(gitlabHost, ":")[0]
	if host, ok := config.Hosts[hostWithoutPort]; ok && host.Token != "" {
		return host.Token
	}

	return ""
}

// getGitCredentialToken uses git credential helper to get token
func getGitCredentialToken(gitlabHost string) string {
	// First check if a credential helper is configured
	helperCmd := exec.Command("git", "config", "--get", "credential.helper")
	helperOutput, err := helperCmd.Output()
	if err != nil || strings.TrimSpace(string(helperOutput)) == "" {
		// No credential helper configured, skip this method
		return ""
	}

	helper := strings.TrimSpace(string(helperOutput))

	// Skip helpers that may require interactive input or GUI
	// These helpers are known to potentially block or show dialogs
	interactiveHelpers := []string{"manager", "manager-core", "osxkeychain", "wincred"}
	for _, interactive := range interactiveHelpers {
		if strings.Contains(strings.ToLower(helper), interactive) {
			// These helpers may require user interaction
			// Skip them to avoid blocking - credentials should be
			// accessed via environment variables or config files instead
			return ""
		}
	}

	// Only proceed with non-interactive helpers like "store" or "cache"
	// Prepare the credential request
	input := fmt.Sprintf("protocol=https\nhost=%s\n\n", gitlabHost)

	// Use a context with timeout to prevent blocking
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Run git credential fill with timeout
	cmd := exec.CommandContext(ctx, "git", "credential", "fill")
	cmd.Stdin = strings.NewReader(input)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		// Command failed or timed out
		return ""
	}

	// Parse the output to find password (token)
	scanner := bufio.NewScanner(&stdout)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "password=") {
			return strings.TrimPrefix(line, "password=")
		}
	}

	return ""
}

// getNetrcToken reads token from .netrc or _netrc file
func getNetrcToken(gitlabHost string) string {
	netrcPaths := getNetrcPaths()

	for _, netrcPath := range netrcPaths {
		if token := readNetrc(netrcPath, gitlabHost); token != "" {
			return token
		}
	}

	return ""
}

// getNetrcPaths returns possible paths for netrc file
func getNetrcPaths() []string {
	var paths []string

	if home, err := os.UserHomeDir(); err == nil {
		paths = append(paths, filepath.Join(home, ".netrc"))
		if runtime.GOOS == "windows" {
			paths = append(paths, filepath.Join(home, "_netrc"))
		}
	}

	// Also check NETRC environment variable
	if netrcPath := os.Getenv("NETRC"); netrcPath != "" {
		paths = append([]string{netrcPath}, paths...)
	}

	return paths
}

// readNetrc parses a netrc file and extracts password for the given host
func readNetrc(netrcPath, gitlabHost string) string {
	file, err := os.Open(netrcPath)
	if err != nil {
		return ""
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var currentMachine string
	var foundMachine bool

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip comments and empty lines
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Tokenize the line
		tokens := strings.Fields(line)
		for i := 0; i < len(tokens); i++ {
			token := tokens[i]

			switch token {
			case "machine":
				if i+1 < len(tokens) {
					currentMachine = tokens[i+1]
					foundMachine = (currentMachine == gitlabHost ||
						currentMachine == strings.Split(gitlabHost, ":")[0])
					i++
				}
			case "default":
				// Default entry matches any host
				if !foundMachine {
					foundMachine = true
					currentMachine = "default"
				}
			case "password":
				if foundMachine && i+1 < len(tokens) {
					return tokens[i+1]
				}
				i++
			case "login", "account":
				// Skip login and account tokens
				if i+1 < len(tokens) {
					i++
				}
			}
		}
	}

	return ""
}

// ExtractHostFromURL extracts the host from a GitLab API URL
func ExtractHostFromURL(apiURL string) string {
	// Remove protocol
	url := apiURL
	url = strings.TrimPrefix(url, "https://")
	url = strings.TrimPrefix(url, "http://")

	// Remove path
	if idx := strings.Index(url, "/"); idx != -1 {
		url = url[:idx]
	}

	return url
}
