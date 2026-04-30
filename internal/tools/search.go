package tools

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// SearchTool provides file and content search capabilities
type SearchTool struct {
	RootDir     string
	MaxFiles    int
	MaxFileSize int64
	IgnoreDirs  []string
	IgnoreFiles []string
}

func NewSearchTool(rootDir string, maxFiles int, maxFileSize int64, ignoreDirs, ignoreFiles []string) *SearchTool {
	return &SearchTool{
		RootDir:     rootDir,
		MaxFiles:    maxFiles,
		MaxFileSize: maxFileSize,
		IgnoreDirs:  ignoreDirs,
		IgnoreFiles: ignoreFiles,
	}
}

func (t *SearchTool) Name() string {
	return "search_files"
}

func (t *SearchTool) Description() string {
	return "Search for files by name pattern or content. Use to find relevant code in the workspace."
}

func (t *SearchTool) Schema() ToolSchema {
	return ToolSchema{
		Type: "object",
		Properties: map[string]*SchemaProperty{
			"pattern": {Type: "string", Description: "Search pattern (glob for names, regex for content)"},
			"type":    {Type: "string", Description: "Search type: 'name' or 'content'"},
			"path":    {Type: "string", Description: "Optional subdirectory to search in"},
		},
		Required: []string{"pattern", "type"},
	}
}

func (t *SearchTool) RiskLevel() RiskLevel {
	return RiskLevelLow
}

func (t *SearchTool) Run(ctx context.Context, input ToolInput) (ToolResult, error) {
	pattern, ok := input["pattern"].(string)
	if !ok {
		return ToolResult{}, fmt.Errorf("pattern must be a string")
	}

	searchType, ok := input["type"].(string)
	if !ok {
		return ToolResult{}, fmt.Errorf("type must be a string")
	}

	path, _ := input["path"].(string)
	if path == "" {
		path = t.RootDir
	}

	var results []string
	var err error

	switch searchType {
	case "name":
		results, err = t.searchByName(ctx, path, pattern)
	case "content":
		results, err = t.searchByContent(ctx, path, pattern)
	default:
		return ToolResult{}, fmt.Errorf("unknown search type: %s", searchType)
	}

	if err != nil {
		return ToolResult{}, err
	}

	return ToolResult{
		Output: strings.Join(results, "\n"),
		Files:  results,
	}, nil
}

func (t *SearchTool) searchByName(ctx context.Context, root, pattern string) ([]string, error) {
	var results []string
	filesScanned := 0

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip inaccessible paths
		}

		// Check context cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Skip ignored directories
		if info.IsDir() && t.shouldIgnoreDir(path) {
			return filepath.SkipDir
		}

		// Skip ignored files
		if t.shouldIgnoreFile(info.Name()) {
			return nil
		}

		// Limit files scanned
		filesScanned++
		if filesScanned > t.MaxFiles {
			return fmt.Errorf("max files limit reached")
		}

		// Match pattern
		matched, err := filepath.Match(pattern, info.Name())
		if err != nil {
			// Try regex if glob fails
			matched, _ = regexp.MatchString(pattern, info.Name())
		}

		if matched && !info.IsDir() {
			results = append(results, path)
		}

		return nil
	})

	if err != nil && err != ctx.Err() {
		// Non-critical errors just stop the walk
		if err.Error() != "max files limit reached" {
			return nil, err
		}
	}

	return results, nil
}

func (t *SearchTool) searchByContent(ctx context.Context, root, pattern string) ([]string, error) {
	var results []string
	filesScanned := 0

	regex, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid regex pattern: %w", err)
	}

	err = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if info.IsDir() && t.shouldIgnoreDir(path) {
			return filepath.SkipDir
		}

		if t.shouldIgnoreFile(info.Name()) || info.IsDir() {
			return nil
		}

		filesScanned++
		if filesScanned > t.MaxFiles {
			return fmt.Errorf("max files limit reached")
		}

		if info.Size() > t.MaxFileSize {
			return nil // Skip large files
		}

		// Search file content
		matches, err := t.searchFileContent(path, regex)
		if err != nil {
			return nil
		}

		if len(matches) > 0 {
			for _, match := range matches {
				results = append(results, fmt.Sprintf("%s:%s", path, match))
			}
		}

		return nil
	})

	if err != nil && err != ctx.Err() {
		if err.Error() != "max files limit reached" {
			return nil, err
		}
	}

	return results, nil
}

func (t *SearchTool) searchFileContent(path string, regex *regexp.Regexp) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var matches []string
	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		if regex.MatchString(line) {
			matches = append(matches, fmt.Sprintf("L%d: %s", lineNum, line))
		}
	}

	return matches, scanner.Err()
}

func (t *SearchTool) shouldIgnoreDir(path string) bool {
	name := filepath.Base(path)
	for _, ignored := range t.IgnoreDirs {
		if name == ignored {
			return true
		}
	}
	return false
}

func (t *SearchTool) shouldIgnoreFile(name string) bool {
	for _, ignored := range t.IgnoreFiles {
		if strings.HasSuffix(name, ignored) {
			return true
		}
	}
	return false
}
