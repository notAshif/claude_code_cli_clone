package tools

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

// ShellTool executes shell commands
type ShellTool struct {
	AllowedCommands []string
	Timeout         time.Duration
	WorkingDir      string
}

func NewShellTool(allowedCmds []string, timeout time.Duration, workDir string) *ShellTool {
	return &ShellTool{
		AllowedCommands: allowedCmds,
		Timeout:         timeout,
		WorkingDir:      workDir,
	}
}

func (t *ShellTool) Name() string {
	return "run_shell"
}

func (t *ShellTool) Description() string {
	return "Execute a shell command. Use for running tests, builds, git operations, etc."
}

func (t *ShellTool) Schema() ToolSchema {
	return ToolSchema{
		Type: "object",
		Properties: map[string]*SchemaProperty{
			"command": {Type: "string", Description: "The shell command to execute"},
		},
		Required: []string{"command"},
	}
}

func (t *ShellTool) RiskLevel() RiskLevel {
	return RiskLevelHigh
}

func (t *ShellTool) Run(ctx context.Context, input ToolInput) (ToolResult, error) {
	command, ok := input["command"].(string)
	if !ok {
		return ToolResult{}, fmt.Errorf("command must be a string")
	}

	// Validate command is allowed
	if err := t.validateCommand(command); err != nil {
		return ToolResult{}, err
	}

	// Create context with timeout
	execCtx, cancel := context.WithTimeout(ctx, t.Timeout)
	defer cancel()

	// Determine shell based on OS
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(execCtx, "cmd", "/C", command)
	} else {
		cmd = exec.CommandContext(execCtx, "sh", "-c", command)
	}

	if t.WorkingDir != "" {
		cmd.Dir = t.WorkingDir
	}

	output, err := cmd.CombinedOutput()

	result := ToolResult{
		Output: string(output),
	}

	if err != nil {
		if execCtx.Err() == context.DeadlineExceeded {
			return result, fmt.Errorf("command timed out after %v", t.Timeout)
		}
		// Command ran but exited with error - still return output
		result.Error = fmt.Errorf("command failed: %w", err)
	}

	return result, nil
}

func (t *ShellTool) validateCommand(command string) error {
	if len(t.AllowedCommands) == 0 {
		return nil // No restrictions
	}

	// Check if command starts with an allowed prefix
	cmdLower := strings.ToLower(strings.TrimSpace(command))
	for _, allowed := range t.AllowedCommands {
		if strings.HasPrefix(cmdLower, strings.ToLower(allowed)) {
			return nil
		}
	}

	return fmt.Errorf("command not in allowed list: %s", command)
}

// Blocked commands check
var blockedPatterns = []string{
	"rm -rf /",
	"rm -rf /*",
	"mkfs",
	"dd if=/dev/zero",
	":(){:|:&};:",
	"chmod -R 777 /",
	"chown -R",
}

func IsCommandBlocked(command string) bool {
	cmdLower := strings.ToLower(command)
	for _, pattern := range blockedPatterns {
		if strings.Contains(cmdLower, strings.ToLower(pattern)) {
			return true
		}
	}
	return false
}
