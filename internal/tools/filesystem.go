package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// FilesystemTool provides file operations
type FilesystemTool struct {
	MaxFileSize int64
	AllowedDirs []string
}

func NewFilesystemTool(maxSize int64, allowedDirs []string) *FilesystemTool {
	return &FilesystemTool{
		MaxFileSize: maxSize,
		AllowedDirs: allowedDirs,
	}
}

func (t *FilesystemTool) Name() string {
	return "read_file"
}

func (t *FilesystemTool) Description() string {
	return "Read contents of a file. Use this to examine existing files in the workspace."
}

func (t *FilesystemTool) Schema() ToolSchema {
	return ToolSchema{
		Type: "object",
		Properties: map[string]*SchemaProperty{
			"path": {Type: "string", Description: "Absolute or relative path to the file"},
		},
		Required: []string{"path"},
	}
}

func (t *FilesystemTool) RiskLevel() RiskLevel {
	return RiskLevelLow
}

func (t *FilesystemTool) Run(ctx context.Context, input ToolInput) (ToolResult, error) {
	path, ok := input["path"].(string)
	if !ok {
		return ToolResult{}, fmt.Errorf("path must be a string")
	}

	// Validate path is within allowed directories
	if err := t.validatePath(path); err != nil {
		return ToolResult{}, err
	}

	info, err := os.Stat(path)
	if err != nil {
		return ToolResult{}, fmt.Errorf("cannot access file: %w", err)
	}

	if info.IsDir() {
		return ToolResult{}, fmt.Errorf("path is a directory, not a file")
	}

	if info.Size() > t.MaxFileSize {
		return ToolResult{}, fmt.Errorf("file too large: %d bytes (max: %d)", info.Size(), t.MaxFileSize)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return ToolResult{}, fmt.Errorf("cannot read file: %w", err)
	}

	return ToolResult{
		Output: string(content),
		Files:  []string{path},
	}, nil
}

func (t *FilesystemTool) validatePath(path string) error {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}

	// Check against allowed directories
	for _, allowed := range t.AllowedDirs {
		if strings.HasPrefix(absPath, allowed) {
			return nil
		}
	}

	return fmt.Errorf("path outside allowed directories")
}

// WriteFileTool for writing files
type WriteFileTool struct {
	AllowedDirs []string
}

func NewWriteFileTool(allowedDirs []string) *WriteFileTool {
	return &WriteFileTool{AllowedDirs: allowedDirs}
}

func (t *WriteFileTool) Name() string {
	return "write_file"
}

func (t *WriteFileTool) Description() string {
	return "Write content to a file. Creates the file if it doesn't exist, overwrites if it does."
}

func (t *WriteFileTool) Schema() ToolSchema {
	return ToolSchema{
		Type: "object",
		Properties: map[string]*SchemaProperty{
			"path":    {Type: "string", Description: "Path to the file"},
			"content": {Type: "string", Description: "Content to write"},
		},
		Required: []string{"path", "content"},
	}
}

func (t *WriteFileTool) RiskLevel() RiskLevel {
	return RiskLevelHigh
}

func (t *WriteFileTool) Run(ctx context.Context, input ToolInput) (ToolResult, error) {
	path, ok := input["path"].(string)
	if !ok {
		return ToolResult{}, fmt.Errorf("path must be a string")
	}

	content, ok := input["content"].(string)
	if !ok {
		return ToolResult{}, fmt.Errorf("content must be a string")
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return ToolResult{}, fmt.Errorf("invalid path: %w", err)
	}

	// Create parent directory if needed
	dir := filepath.Dir(absPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return ToolResult{}, fmt.Errorf("cannot create directory: %w", err)
	}

	if err := os.WriteFile(absPath, []byte(content), 0644); err != nil {
		return ToolResult{}, fmt.Errorf("cannot write file: %w", err)
	}

	return ToolResult{
		Output:  fmt.Sprintf("Successfully wrote to %s", absPath),
		Files:   []string{absPath},
		Changes: []string{"created"},
	}, nil
}
