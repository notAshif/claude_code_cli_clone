package workspace

import (
	"os"
	"path/filepath"
	"strings"
)

// DefaultIgnoreDirs are directories typically excluded from scanning
var DefaultIgnoreDirs = []string{
	".git",
	"node_modules",
	"vendor",
	"dist",
	"build",
	"target",
	"__pycache__",
	".venv",
	"venv",
	"bin",
	"obj",
	".idea",
	".vscode",
}

// DefaultIgnoreFiles are file patterns to exclude
var DefaultIgnoreFiles = []string{
	".DS_Store",
	"Thumbs.db",
	"*.log",
	"*.tmp",
	"*.swp",
	"*.swo",
	"*~",
}

// FileInfo represents metadata about a scanned file
type FileInfo struct {
	Path    string
	Size    int64
	ModTime int64
	IsDir   bool
}

// ScanResult holds the output of a workspace scan
type ScanResult struct {
	RootDir      string
	Files        []FileInfo
	TotalFiles   int
	TotalDirs    int
	TotalSize    int64
	IgnoredFiles int
	IgnoredDirs  int
}

// Scanner walks and catalogs a workspace
type Scanner struct {
	RootDir     string
	MaxFiles    int
	MaxFileSize int64
	IgnoreDirs  []string
	IgnoreFiles []string
}

func NewScanner(rootDir string, maxFiles int, maxFileSize int64) *Scanner {
	return &Scanner{
		RootDir:     rootDir,
		MaxFiles:    maxFiles,
		MaxFileSize: maxFileSize,
		IgnoreDirs:  DefaultIgnoreDirs,
		IgnoreFiles: DefaultIgnoreFiles,
	}
}

// Scan walks the workspace and returns file information
func (s *Scanner) Scan() (*ScanResult, error) {
	result := &ScanResult{
		RootDir: s.RootDir,
		Files:   make([]FileInfo, 0),
	}

	err := filepath.Walk(s.RootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip inaccessible paths
		}

		// Check ignore patterns
		name := info.Name()
		if info.IsDir() && s.shouldIgnoreDir(name) {
			result.IgnoredDirs++
			return filepath.SkipDir
		}

		if s.shouldIgnoreFile(name) {
			result.IgnoredFiles++
			return nil
		}

		// Track directories
		if info.IsDir() {
			result.TotalDirs++
			return nil
		}

		// Check file limits
		if len(result.Files) >= s.MaxFiles {
			return nil
		}

		// Skip files over size limit
		if info.Size() > s.MaxFileSize {
			result.IgnoredFiles++
			return nil
		}

		// Add file to results
		result.Files = append(result.Files, FileInfo{
			Path:    path,
			Size:    info.Size(),
			ModTime: info.ModTime().Unix(),
			IsDir:   false,
		})
		result.TotalSize += info.Size()
		result.TotalFiles++

		return nil
	})

	if err != nil {
		return nil, err
	}

	return result, nil
}

// GetFileTree returns a string representation of the file structure
func (s *Scanner) GetFileTree(maxDepth int) (string, error) {
	var sb strings.Builder

	err := filepath.Walk(s.RootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		// Calculate depth
		relPath, err := filepath.Rel(s.RootDir, path)
		if err != nil {
			return nil
		}

		depth := len(strings.Split(relPath, string(filepath.Separator))) - 1
		if depth > maxDepth {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Skip ignored
		if info.IsDir() && s.shouldIgnoreDir(info.Name()) {
			return filepath.SkipDir
		}
		if s.shouldIgnoreFile(info.Name()) {
			return nil
		}

		// Build indent
		indent := strings.Repeat("  ", depth)
		prefix := ""
		if info.IsDir() {
			prefix = "[DIR] "
		}

		sb.WriteString(indent)
		sb.WriteString(prefix)
		sb.WriteString(info.Name())
		sb.WriteString("\n")

		return nil
	})

	return sb.String(), err
}

func (s *Scanner) shouldIgnoreDir(name string) bool {
	for _, ignored := range s.IgnoreDirs {
		if name == ignored {
			return true
		}
	}
	return false
}

func (s *Scanner) shouldIgnoreFile(name string) bool {
	for _, pattern := range s.IgnoreFiles {
		if strings.HasSuffix(name, strings.TrimPrefix(pattern, "*")) {
			return true
		}
		if pattern == name {
			return true
		}
	}
	return false
}
