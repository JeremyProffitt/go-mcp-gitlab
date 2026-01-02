package logging

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"
)

// LogLevel represents the severity of a log message
type LogLevel int

// ConfigSource indicates where a configuration value came from
type ConfigSource string

const (
	// SourceDefault indicates the value is the default
	SourceDefault ConfigSource = "default"
	// SourceEnvironment indicates the value came from an environment variable
	SourceEnvironment ConfigSource = "environment"
	// SourceFlag indicates the value came from a command-line flag
	SourceFlag ConfigSource = "flag"
)

const (
	// LevelOff disables all logging
	LevelOff LogLevel = iota
	// LevelError logs only errors
	LevelError
	// LevelWarn logs warnings and errors
	LevelWarn
	// LevelInfo logs general information, warnings, and errors
	LevelInfo
	// LevelAccess logs API access operations
	LevelAccess
	// LevelDebug logs detailed debugging information
	LevelDebug
)

// String returns the string representation of the log level
func (l LogLevel) String() string {
	switch l {
	case LevelOff:
		return "OFF"
	case LevelError:
		return "ERROR"
	case LevelWarn:
		return "WARN"
	case LevelInfo:
		return "INFO"
	case LevelAccess:
		return "ACCESS"
	case LevelDebug:
		return "DEBUG"
	default:
		return "UNKNOWN"
	}
}

// ParseLogLevel converts a string to LogLevel
func ParseLogLevel(s string) LogLevel {
	switch s {
	case "off", "OFF":
		return LevelOff
	case "error", "ERROR":
		return LevelError
	case "warn", "WARN", "warning", "WARNING":
		return LevelWarn
	case "info", "INFO":
		return LevelInfo
	case "access", "ACCESS":
		return LevelAccess
	case "debug", "DEBUG":
		return LevelDebug
	default:
		return LevelInfo
	}
}

// Logger is the main logging structure
type Logger struct {
	mu        sync.Mutex
	level     LogLevel
	logger    *log.Logger
	file      *os.File
	logDir    string
	appName   string
	startTime time.Time
}

// Config holds logger configuration
type Config struct {
	// LogDir is the directory for log files. If empty, uses <user_home>/<AppName>/logs
	LogDir string
	// AppName is used for default log directory and log file naming
	AppName string
	// Level is the minimum log level to output
	Level LogLevel
}

var (
	defaultLogger *Logger
	once          sync.Once
)

// DefaultLogDir returns the default log directory path
func DefaultLogDir(appName string) string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		// Fallback to current directory
		return filepath.Join(".", appName, "logs")
	}
	return filepath.Join(homeDir, appName, "logs")
}

// Init initializes the global logger with the given configuration
func Init(cfg Config) error {
	var initErr error
	once.Do(func() {
		defaultLogger, initErr = NewLogger(cfg)
	})
	return initErr
}

// NewLogger creates a new Logger instance
func NewLogger(cfg Config) (*Logger, error) {
	if cfg.AppName == "" {
		cfg.AppName = "go-mcp-gitlab"
	}

	logDir := cfg.LogDir
	if logDir == "" {
		logDir = DefaultLogDir(cfg.AppName)
	}

	// Create log directory if it doesn't exist
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory %s: %w", logDir, err)
	}

	// Create log file with timestamp
	timestamp := time.Now().Format("2006-01-02")
	logFileName := fmt.Sprintf("%s-%s.log", cfg.AppName, timestamp)
	logPath := filepath.Join(logDir, logFileName)

	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file %s: %w", logPath, err)
	}

	l := &Logger{
		level:     cfg.Level,
		logger:    log.New(file, "", 0),
		file:      file,
		logDir:    logDir,
		appName:   cfg.AppName,
		startTime: time.Now(),
	}

	return l, nil
}

// Close closes the log file
func (l *Logger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.file != nil {
		return l.file.Close()
	}
	return nil
}

// SetLevel sets the log level
func (l *Logger) SetLevel(level LogLevel) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.level = level
}

// log writes a log entry if the level is enabled
func (l *Logger) log(level LogLevel, format string, args ...interface{}) {
	if l == nil || level > l.level {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	timestamp := time.Now().Format("2006-01-02T15:04:05.000Z07:00")
	message := fmt.Sprintf(format, args...)
	l.logger.Printf("[%s] [%s] %s", timestamp, level.String(), message)
}

// Error logs an error message
func (l *Logger) Error(format string, args ...interface{}) {
	l.log(LevelError, format, args...)
}

// Warn logs a warning message
func (l *Logger) Warn(format string, args ...interface{}) {
	l.log(LevelWarn, format, args...)
}

// Info logs an informational message
func (l *Logger) Info(format string, args ...interface{}) {
	l.log(LevelInfo, format, args...)
}

// Access logs API access operations
func (l *Logger) Access(format string, args ...interface{}) {
	l.log(LevelAccess, format, args...)
}

// Debug logs debug information
func (l *Logger) Debug(format string, args ...interface{}) {
	l.log(LevelDebug, format, args...)
}

// ToolCall logs an MCP tool invocation
func (l *Logger) ToolCall(toolName string, args map[string]interface{}) {
	// Log tool name and argument keys only, never values that might contain sensitive data
	argKeys := make([]string, 0, len(args))
	for k := range args {
		argKeys = append(argKeys, k)
	}
	l.Info("TOOL_CALL tool=%q args=%v", toolName, argKeys)
}

// APICall logs a GitLab API call with method, endpoint, status code, and optional error
func (l *Logger) APICall(method, endpoint string, statusCode int, err error) {
	if err != nil {
		l.Access("API_CALL method=%s endpoint=%q status=%d error=%q", method, endpoint, statusCode, err.Error())
	} else {
		l.Access("API_CALL method=%s endpoint=%q status=%d", method, endpoint, statusCode)
	}
}

// APIRequest logs an outgoing GitLab API request
func (l *Logger) APIRequest(method, endpoint string) {
	l.Access("API_REQUEST method=%s endpoint=%q", method, endpoint)
}

// APIResponse logs a GitLab API response with endpoint, status code, and duration
func (l *Logger) APIResponse(endpoint string, statusCode int, duration time.Duration) {
	l.Access("API_RESPONSE endpoint=%q status=%d duration=%s", endpoint, statusCode, duration)
}

// ConfigValue holds a configuration value and its source
type ConfigValue struct {
	Value  string
	Source ConfigSource
}

// StartupInfo holds startup information with configuration sources
type StartupInfo struct {
	Version      string
	GoVersion    string
	OS           string
	Arch         string
	NumCPU       int
	LogDir       ConfigValue
	LogLevel     ConfigValue
	GitLabURL    ConfigValue
	GitLabToken  ConfigValue // Will show masked value
	PID          int
	StartTime    time.Time
}

// LogStartup logs comprehensive startup information
func (l *Logger) LogStartup(info StartupInfo) {
	l.Info("========================================")
	l.Info("SERVER STARTUP")
	l.Info("========================================")
	l.Info("Application: %s", l.appName)
	l.Info("Version: %s", info.Version)
	l.Info("Go Version: %s", info.GoVersion)
	l.Info("OS: %s", info.OS)
	l.Info("Architecture: %s", info.Arch)
	l.Info("Number of CPUs: %d", info.NumCPU)
	l.Info("Process ID: %d", info.PID)
	l.Info("Start Time: %s", info.StartTime.Format(time.RFC3339))
	l.Info("----------------------------------------")
	l.Info("CONFIGURATION (value [source])")
	l.Info("----------------------------------------")
	l.Info("Log Directory: %s [%s]", info.LogDir.Value, info.LogDir.Source)
	l.Info("Log Level: %s [%s]", info.LogLevel.Value, info.LogLevel.Source)
	l.Info("GitLab URL: %s [%s]", info.GitLabURL.Value, info.GitLabURL.Source)
	l.Info("GitLab Token: %s [%s]", info.GitLabToken.Value, info.GitLabToken.Source)
	l.Info("----------------------------------------")
	l.Info("ENVIRONMENT")
	l.Info("----------------------------------------")

	// Log working directory
	if wd, err := os.Getwd(); err == nil {
		l.Info("Working Directory: %s", wd)
	}

	// Log home directory
	if home, err := os.UserHomeDir(); err == nil {
		l.Info("Home Directory: %s", home)
	}

	l.Info("========================================")
}

// LogShutdown logs shutdown information
func (l *Logger) LogShutdown(reason string) {
	uptime := time.Since(l.startTime)
	l.Info("========================================")
	l.Info("SERVER SHUTDOWN")
	l.Info("========================================")
	l.Info("Reason: %s", reason)
	l.Info("Uptime: %s", uptime)
	l.Info("========================================")
}

// Global logger convenience functions

// GetLogger returns the default logger (may be nil if not initialized)
func GetLogger() *Logger {
	return defaultLogger
}

// SetOutput sets the output writer for the logger (useful for testing)
func (l *Logger) SetOutput(w io.Writer) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.logger.SetOutput(w)
}

// GetStartupInfo returns a populated StartupInfo struct
func GetStartupInfo(version string, logDir, logLevel, gitlabURL, gitlabToken ConfigValue) StartupInfo {
	return StartupInfo{
		Version:     version,
		GoVersion:   runtime.Version(),
		OS:          runtime.GOOS,
		Arch:        runtime.GOARCH,
		NumCPU:      runtime.NumCPU(),
		LogDir:      logDir,
		LogLevel:    logLevel,
		GitLabURL:   gitlabURL,
		GitLabToken: gitlabToken,
		PID:         os.Getpid(),
		StartTime:   time.Now(),
	}
}

// MaskToken masks a token for safe logging (shows first 4 and last 4 characters)
func MaskToken(token string) string {
	if len(token) <= 8 {
		return "****"
	}
	return token[:4] + "..." + token[len(token)-4:]
}

// Global convenience functions that use the default logger

// Error logs an error using the default logger
func Error(format string, args ...interface{}) {
	if defaultLogger != nil {
		defaultLogger.Error(format, args...)
	}
}

// Warn logs a warning using the default logger
func Warn(format string, args ...interface{}) {
	if defaultLogger != nil {
		defaultLogger.Warn(format, args...)
	}
}

// Info logs info using the default logger
func Info(format string, args ...interface{}) {
	if defaultLogger != nil {
		defaultLogger.Info(format, args...)
	}
}

// Access logs access using the default logger
func Access(format string, args ...interface{}) {
	if defaultLogger != nil {
		defaultLogger.Access(format, args...)
	}
}

// Debug logs debug using the default logger
func Debug(format string, args ...interface{}) {
	if defaultLogger != nil {
		defaultLogger.Debug(format, args...)
	}
}

// ToolCall logs tool call using the default logger
func ToolCall(toolName string, args map[string]interface{}) {
	if defaultLogger != nil {
		defaultLogger.ToolCall(toolName, args)
	}
}

// APICall logs API call using the default logger
func APICall(method, endpoint string, statusCode int, err error) {
	if defaultLogger != nil {
		defaultLogger.APICall(method, endpoint, statusCode, err)
	}
}

// APIRequest logs API request using the default logger
func APIRequest(method, endpoint string) {
	if defaultLogger != nil {
		defaultLogger.APIRequest(method, endpoint)
	}
}

// APIResponse logs API response using the default logger
func APIResponse(endpoint string, statusCode int, duration time.Duration) {
	if defaultLogger != nil {
		defaultLogger.APIResponse(endpoint, statusCode, duration)
	}
}
