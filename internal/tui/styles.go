package tui

import (
	"github.com/charmbracelet/lipgloss"
)

// Styles holds all TUI styles
type Styles struct {
	Title            lipgloss.Style
	Subtitle         lipgloss.Style
	Header           lipgloss.Style
	Panel            lipgloss.Style
	UserMessage      lipgloss.Style
	AssistantMessage lipgloss.Style
	UserBubble       lipgloss.Style
	AssistantBubble  lipgloss.Style
	Status           lipgloss.Style
	Error            lipgloss.Style
	Approval         lipgloss.Style
	ToolCall         lipgloss.Style
	Help             lipgloss.Style
	InputBox         lipgloss.Style
}

var styles = Styles{
	Title: lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("15")).
		Padding(0, 1),

	Subtitle: lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Italic(true),

	Header: lipgloss.NewStyle().
		Foreground(lipgloss.Color("15")).
		Background(lipgloss.Color("24")).
		Bold(true).
		Padding(0, 1),

	Panel: lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), true).
		BorderForeground(lipgloss.Color("238")).
		Padding(1, 2),

	UserMessage: lipgloss.NewStyle().
		Foreground(lipgloss.Color("39")).
		Bold(true),

	AssistantMessage: lipgloss.NewStyle().
		Foreground(lipgloss.Color("42")).
		Bold(true),

	UserBubble: lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), false, false, false, true).
		BorderForeground(lipgloss.Color("39")).
		PaddingLeft(1),

	AssistantBubble: lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), false, false, false, true).
		BorderForeground(lipgloss.Color("42")).
		PaddingLeft(1),

	Status: lipgloss.NewStyle().
		Foreground(lipgloss.Color("15")).
		Background(lipgloss.Color("65")).
		Padding(0, 1),

	Error: lipgloss.NewStyle().
		Foreground(lipgloss.Color("196")).
		Bold(true),

	Approval: lipgloss.NewStyle().
		Foreground(lipgloss.Color("208")).
		Bold(true).
		Padding(1),

	ToolCall: lipgloss.NewStyle().
		Foreground(lipgloss.Color("27")).
		Padding(0, 1),

	Help: lipgloss.NewStyle().
		Foreground(lipgloss.Color("242")),

	InputBox: lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), true).
		BorderForeground(lipgloss.Color("240")).
		Padding(0, 1),
}

// Color scheme
const (
	ColorPrimary   = lipgloss.Color("205")
	ColorSecondary = lipgloss.Color("33")
	ColorSuccess   = lipgloss.Color("34")
	ColorWarning   = lipgloss.Color("208")
	ColorError     = lipgloss.Color("196")
	ColorMuted     = lipgloss.Color("241")
)

// Border styles
var (
	BorderBox = lipgloss.RoundedBorder()
)

// Helper styles
func UserLabel() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("33")).
		Bold(true).
		Padding(0, 1)
}

func AssistantLabel() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("34")).
		Bold(true).
		Padding(0, 1)
}

func ToolLabel() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("27")).
		Padding(0, 1)
}

func ApprovalPrompt() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("208")).
		Bold(true).
		Border(lipgloss.RoundedBorder(), true).
		Padding(1, 2)
}
