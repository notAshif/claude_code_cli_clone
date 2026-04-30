package workspace

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Context holds workspace state and metadata
type Context struct {
	RootDir      string
	WorkingDir   string
	GitRepo      *GitInfo
	FileTree     string
	ActiveFiles  []string
	Environment  map[string]string
}

// BuildContext creates workspace context from the given directory
func BuildContext(dir string) (*Context, error) {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve directory: %w", err)
	}

	ctx := &Context{
		RootDir:    absDir,
		WorkingDir: absDir,
		Environment: make(map[string]string),
	}

	// Capture environment variables
	ctx.Environment["GOOS"] = os.Getenv("GOOS")
	ctx.Environment["GOARCH"] = os.Getenv("GOARCH")
	ctx.Environment["PATH"] = os.Getenv("PATH")
	ctx.Environment["HOME"] = os.Getenv("HOME")
	if ctx.Environment["HOME"] == "" {
		ctx.Environment["HOME"] = os.Getenv("USERPROFILE") // Windows
	}

	// Try to get git info
	if gitInfo, err := GetGitInfo(absDir); err == nil {
		ctx.GitRepo = gitInfo
	}

	// Generate file tree (3 levels deep)
	scanner := NewScanner(absDir, 1000, 1024*1024) // 1MB max file
	if tree, err := scanner.GetFileTree(3); err == nil {
		ctx.FileTree = tree
	}

	return ctx, nil
}

// ReadFile reads a file within the workspace
func (c *Context) ReadFile(path string) (string, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}

	// Security: ensure file is within workspace
	rel, err := filepath.Rel(c.RootDir, absPath)
	if err != nil || strings.HasPrefix(rel, "..") {
		return "", fmt.Errorf("file outside workspace: %s", path)
	}

	content, err := os.ReadFile(absPath)
	if err != nil {
		return "", err
	}

	return string(content), nil
}

// WriteFile writes a file within the workspace
func (c *Context) WriteFile(path string, content string) error {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return err
	}

	// Security: ensure file is within workspace
	rel, err := filepath.Rel(c.RootDir, absPath)
	if err != nil || strings.HasPrefix(rel, "..") {
		return fmt.Errorf("file outside workspace: %s", path)
	}

	// Create parent directory if needed
	dir := filepath.Dir(absPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	return os.WriteFile(absPath, []byte(content), 0644)
}

// GetRelativePath converts absolute path to workspace-relative
func (c *Context) GetRelativePath(absPath string) (string, error) {
	return filepath.Rel(c.RootDir, absPath)
}

// Summary returns a human-readable workspace summary
func (c *Context) Summary() string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Workspace: %s\n", c.RootDir))

	if c.GitRepo != nil {
		sb.WriteString(fmt.Sprintf("Git Branch: %s\n", c.GitRepo.CurrentBranch))
		if c.GitRepo.HasChanges {
			sb.WriteString("Status: Modified\n")
		} else {
			sb.WriteString("Status: Clean\n")
		}
	}

	sb.WriteString("\nDirectory Structure:\n")
	sb.WriteString(c.FileTree)

	return sb.String()
}
