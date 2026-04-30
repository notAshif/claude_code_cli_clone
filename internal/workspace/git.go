package workspace

import (
	"os/exec"
	"strings"
)

// GitInfo holds git repository state
type GitInfo struct {
	RootDir       string
	CurrentBranch string
	RemoteURL     string
	HasChanges    bool
	StagedFiles   []string
	ModifiedFiles []string
	UntrackedFiles []string
}

// GetGitInfo retrieves git information for the given directory
func GetGitInfo(dir string) (*GitInfo, error) {
	info := &GitInfo{
		RootDir: dir,
	}

	// Check if it's a git repo
	cmd := exec.Command("git", "rev-parse", "--is-inside-work-tree")
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, err
	}

	if strings.TrimSpace(string(output)) != "true" {
		return nil, nil // Not a git repo
	}

	// Get current branch
	cmd = exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = dir
	output, err = cmd.CombinedOutput()
	if err == nil {
		info.CurrentBranch = strings.TrimSpace(string(output))
	}

	// Get remote URL (origin)
	cmd = exec.Command("git", "config", "--get", "remote.origin.url")
	cmd.Dir = dir
	output, err = cmd.CombinedOutput()
	if err == nil {
		info.RemoteURL = strings.TrimSpace(string(output))
	}

	// Get staged files
	cmd = exec.Command("git", "diff", "--cached", "--name-only")
	cmd.Dir = dir
	output, err = cmd.CombinedOutput()
	if err == nil && len(output) > 0 {
		info.StagedFiles = strings.Split(strings.TrimSpace(string(output)), "\n")
	}

	// Get modified files
	cmd = exec.Command("git", "diff", "--name-only")
	cmd.Dir = dir
	output, err = cmd.CombinedOutput()
	if err == nil && len(output) > 0 {
		info.ModifiedFiles = strings.Split(strings.TrimSpace(string(output)), "\n")
	}

	// Get untracked files
	cmd = exec.Command("git", "ls-files", "--others", "--exclude-standard")
	cmd.Dir = dir
	output, err = cmd.CombinedOutput()
	if err == nil && len(output) > 0 {
		info.UntrackedFiles = strings.Split(strings.TrimSpace(string(output)), "\n")
	}

	// Determine if there are changes
	info.HasChanges = len(info.StagedFiles) > 0 ||
		len(info.ModifiedFiles) > 0 ||
		len(info.UntrackedFiles) > 0

	return info, nil
}

// GetDiff returns the current diff
func GetDiff(dir string) (string, error) {
	cmd := exec.Command("git", "diff")
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	return string(output), err
}

// GetStagedDiff returns the staged diff
func GetStagedDiff(dir string) (string, error) {
	cmd := exec.Command("git", "diff", "--cached")
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	return string(output), err
}

// GetCommitHistory returns recent commits
func GetCommitHistory(dir string, count int) ([]CommitInfo, error) {
	cmd := exec.Command("git", "log", "-n", string(rune(count)), "--format=%H|%an|%ae|%s|%ai")
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, err
	}

	var commits []CommitInfo
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		parts := strings.Split(line, "|")
		if len(parts) >= 5 {
			commits = append(commits, CommitInfo{
				Hash:    parts[0],
				Author:  parts[1],
				Email:   parts[2],
				Message: parts[3],
				Date:    parts[4],
			})
		}
	}

	return commits, nil
}

// CommitInfo represents a git commit
type CommitInfo struct {
	Hash    string
	Author  string
	Email   string
	Message string
	Date    string
}
