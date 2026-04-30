package tools

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
)

// ApprovalPolicy defines when approval is required
type ApprovalPolicy string

const (
	ApprovalAlways  ApprovalPolicy = "always"
	ApprovalNever   ApprovalPolicy = "never"
	ApprovalOutside ApprovalPolicy = "outside" // Only for operations outside workspace
)

// ApprovalRequest represents a pending approval
type ApprovalRequest struct {
	ToolName    string
	Description string
	Details     string
	RiskLevel   RiskLevel
}

// ApprovalResponse is the result of an approval check
type ApprovalResponse struct {
	Approved bool
	Reason   string
}

// ApprovalGate handles tool call approvals
type ApprovalGate struct {
	ShellPolicy     ApprovalPolicy
	WriteFilePolicy ApprovalPolicy
	WorkspaceRoot   string
	autoApprove     map[string]bool
}

func NewApprovalGate(shellPolicy, writeFilePolicy ApprovalPolicy, workspaceRoot string) *ApprovalGate {
	return &ApprovalGate{
		ShellPolicy:     shellPolicy,
		WriteFilePolicy: writeFilePolicy,
		WorkspaceRoot:   workspaceRoot,
		autoApprove:     make(map[string]bool),
	}
}

// CheckApproval determines if a tool call should be approved
func (g *ApprovalGate) CheckApproval(ctx context.Context, req ApprovalRequest) ApprovalResponse {
	// Check if auto-approved
	if g.autoApprove[req.ToolName] {
		return ApprovalResponse{Approved: true, Reason: "auto-approved"}
	}

	// Check policy based on tool type and risk
	switch req.ToolName {
	case "run_shell":
		if g.ShellPolicy == ApprovalNever {
			return ApprovalResponse{Approved: true, Reason: "shell commands auto-approved"}
		}
		// Always require approval for shell by default
		return ApprovalResponse{Approved: false, Reason: "shell command requires approval"}

	case "write_file":
		if g.WriteFilePolicy == ApprovalNever {
			return ApprovalResponse{Approved: true, Reason: "file writes auto-approved"}
		}
		return ApprovalResponse{Approved: false, Reason: "file write requires approval"}

	default:
		// Low risk tools are auto-approved
		if req.RiskLevel <= RiskLevelLow {
			return ApprovalResponse{Approved: true, Reason: "low risk operation"}
		}
		return ApprovalResponse{Approved: false, Reason: "operation requires approval"}
	}
}

// RequestApproval prompts the user for approval
func (g *ApprovalGate) RequestApproval(req ApprovalRequest) (bool, error) {
	fmt.Printf("\n[APPROVAL REQUIRED]\n")
	fmt.Printf("Tool: %s\n", req.ToolName)
	fmt.Printf("Risk Level: %s\n", req.RiskLevel)
	fmt.Printf("Description: %s\n", req.Description)
	if req.Details != "" {
		fmt.Printf("Details: %s\n", req.Details)
	}
	fmt.Printf("\nApprove? [y/n/auto] ")

	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return false, err
	}

	response = strings.ToLower(strings.TrimSpace(response))

	switch response {
	case "y", "yes":
		return true, nil
	case "auto":
		g.autoApprove[req.ToolName] = true
		return true, nil
	default:
		return false, nil
	}
}

// CheckAndRequest combines check and prompt
func (g *ApprovalGate) CheckAndRequest(ctx context.Context, req ApprovalRequest) (bool, error) {
	resp := g.CheckApproval(ctx, req)
	if resp.Approved {
		return true, nil
	}

	return g.RequestApproval(req)
}
