// Package tools provides helper utilities for MCP tool handlers.
package tools

import (
	"encoding/json"
	"fmt"

	"github.com/go-mcp-gitlab/go-mcp-gitlab/pkg/mcp"
)

// GetString extracts a string value from arguments map.
// Returns defaultVal if the key doesn't exist or is not a string.
func GetString(args map[string]interface{}, key, defaultVal string) string {
	if args == nil {
		return defaultVal
	}
	val, ok := args[key]
	if !ok {
		return defaultVal
	}
	strVal, ok := val.(string)
	if !ok {
		return defaultVal
	}
	return strVal
}

// GetInt extracts an integer value from arguments map.
// Handles both int and float64 (JSON numbers are parsed as float64).
// Returns defaultVal if the key doesn't exist or is not a valid number.
func GetInt(args map[string]interface{}, key string, defaultVal int) int {
	if args == nil {
		return defaultVal
	}
	val, ok := args[key]
	if !ok {
		return defaultVal
	}

	switch v := val.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	case float32:
		return int(v)
	default:
		return defaultVal
	}
}

// GetInt64 extracts an int64 value from arguments map.
// Handles both int64 and float64 (JSON numbers are parsed as float64).
// Returns defaultVal if the key doesn't exist or is not a valid number.
func GetInt64(args map[string]interface{}, key string, defaultVal int64) int64 {
	if args == nil {
		return defaultVal
	}
	val, ok := args[key]
	if !ok {
		return defaultVal
	}

	switch v := val.(type) {
	case int:
		return int64(v)
	case int64:
		return v
	case float64:
		return int64(v)
	case float32:
		return int64(v)
	default:
		return defaultVal
	}
}

// GetBool extracts a boolean value from arguments map.
// Returns defaultVal if the key doesn't exist or is not a boolean.
func GetBool(args map[string]interface{}, key string, defaultVal bool) bool {
	if args == nil {
		return defaultVal
	}
	val, ok := args[key]
	if !ok {
		return defaultVal
	}
	boolVal, ok := val.(bool)
	if !ok {
		return defaultVal
	}
	return boolVal
}

// GetStringArray extracts a string array from arguments map.
// Handles both []string and []interface{} (JSON arrays are parsed as []interface{}).
// Returns nil if the key doesn't exist or is not a valid array.
func GetStringArray(args map[string]interface{}, key string) []string {
	if args == nil {
		return nil
	}
	val, ok := args[key]
	if !ok {
		return nil
	}

	switch v := val.(type) {
	case []string:
		return v
	case []interface{}:
		result := make([]string, 0, len(v))
		for _, item := range v {
			if strItem, ok := item.(string); ok {
				result = append(result, strItem)
			}
		}
		return result
	default:
		return nil
	}
}

// TextResult creates a successful CallToolResult with a text content item.
func TextResult(text string) (*mcp.CallToolResult, error) {
	return &mcp.CallToolResult{
		Content: []mcp.ContentItem{
			{
				Type: "text",
				Text: text,
			},
		},
		IsError: false,
	}, nil
}

// ErrorResult creates an error CallToolResult with the given message.
func ErrorResult(message string) (*mcp.CallToolResult, error) {
	return &mcp.CallToolResult{
		Content: []mcp.ContentItem{
			{
				Type: "text",
				Text: message,
			},
		},
		IsError: true,
	}, nil
}

// JSONResult creates a successful CallToolResult with JSON-encoded data.
// The data is marshaled with indentation for readability.
func JSONResult(data interface{}) (*mcp.CallToolResult, error) {
	jsonBytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return ErrorResult(fmt.Sprintf("failed to marshal JSON response: %v", err))
	}

	return &mcp.CallToolResult{
		Content: []mcp.ContentItem{
			{
				Type: "text",
				Text: string(jsonBytes),
			},
		},
		IsError: false,
	}, nil
}
