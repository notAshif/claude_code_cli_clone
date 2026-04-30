package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// View renders the TUI
func (m Model) View() string {
	var s strings.Builder

	s.WriteString(m.renderHeader())
	s.WriteString(m.renderMessages())
	s.WriteString(m.renderApproval())
	s.WriteString(m.renderStatus())
	s.WriteString(m.renderInput())

	return s.String()
}

func (m Model) renderHeader() string {
	width := contentWidth(m.width)
	title := "Go Code Agent"
	if m.agentLoop != nil {
		title += "  |  coding assistant"
	}

	return styles.Header.Width(width).Render(title) + "\n\n"
}

func (m Model) renderMessages() string {
	width := messageWidth(m.width)
	if len(m.messages) == 0 {
		welcome := "Ask a coding question, request a change, or run a one-shot task from the terminal."
		return styles.Panel.Width(contentWidth(m.width)).Render(styles.Subtitle.Render(welcome)) + "\n\n"
	}

	var s strings.Builder
	for _, msg := range m.messages {
		body := renderMessage(msg.Content, width)
		if msg.Role == "user" {
			label := styles.UserMessage.Render("You")
			s.WriteString(fmt.Sprintf("%s\n%s\n\n", label, styles.UserBubble.Width(width).Render(body)))
		} else {
			label := styles.AssistantMessage.Render("Agent")
			s.WriteString(fmt.Sprintf("%s\n%s\n\n", label, styles.AssistantBubble.Width(width).Render(body)))
		}
	}

	return s.String()
}

func (m Model) renderApproval() string {
	if m.state != StateApproving || m.approvalRequest == nil {
		return ""
	}

	req := m.approvalRequest

	approvalBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder(), true).
		Padding(1, 2).
		BorderForeground(lipgloss.Color("208"))

	content := fmt.Sprintf(
		"Tool: %s\nRisk: %s\nDescription: %s\n\n%s\n\n[y] Approve  [n] Deny  [a] Approve All",
		styles.Approval.Render(req.ToolName),
		req.RiskLevel,
		req.Description,
		req.Details,
	)

	return approvalBox.Render(content) + "\n\n"
}

func (m Model) renderStatus() string {
	if m.isProcessing {
		status := m.status
		if status == "" {
			status = "Processing..."
		}
		return styles.Status.Render(status) + "\n\n"
	}

	if m.errorMsg != "" {
		return styles.Error.Render("Error: "+m.errorMsg) + "\n\n"
	}

	return ""
}

func (m Model) renderInput() string {
	if m.state == StateWaiting {
		return styles.Subtitle.Render("Waiting for response...") + "\n" + styles.Help.Render("Ctrl+C quit") + "\n"
	}

	if m.state == StateApproving {
		return ""
	}

	width := contentWidth(m.width)
	input := styles.InputBox.Width(width).Render(m.input.View())
	return input + "\n" + styles.Help.Render(m.help) + "\n"
}

// renderMessage wraps text to fit width
func renderMessage(text string, width int) string {
	if width <= 0 {
		width = 80
	}

	words := strings.Fields(text)
	if len(words) == 0 {
		return ""
	}

	var lines []string
	currentLine := words[0]

	for _, word := range words[1:] {
		if len(currentLine)+len(word)+1 > width {
			lines = append(lines, currentLine)
			currentLine = word
		} else {
			currentLine += " " + word
		}
	}
	lines = append(lines, currentLine)

	return strings.Join(lines, "\n")
}

func contentWidth(width int) int {
	if width <= 0 {
		return 86
	}
	return max(30, width-4)
}

func messageWidth(width int) int {
	return max(24, contentWidth(width)-6)
}
