package tui

import tea "github.com/charmbracelet/bubbletea"

func (m *Model) handleApprovalRequest(msg ApprovalRequestMsg) (tea.Model, tea.Cmd) {
	m.state = StateApproving
	m.approvalRequest = &ApprovalView{
		ToolName:    msg.ToolName,
		Description: msg.Description,
		Details:     msg.Details,
		RiskLevel:   msg.RiskLevel,
	}
	return m, nil
}

// ApprovalRequestMsg carries an approval request
type ApprovalRequestMsg struct {
	ToolName    string
	Description string
	Details     string
	RiskLevel   string
}

// Approval commands
type ApproveCmd struct{}
type DenyCmd struct{}
type ApproveAllCmd struct{}
