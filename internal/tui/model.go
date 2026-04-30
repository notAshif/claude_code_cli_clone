package tui

import (
	"context"

	"github.com/asif/gocode-agent/internal/agent"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// State represents the current TUI state
type State int

const (
	StateIdle State = iota
	StateTyping
	StateWaiting
	StateApproving
	StateError
)

// Model is the main TUI model
type Model struct {
	ctx    context.Context
	state  State
	width  int
	height int

	// Input
	input textinput.Model

	// Conversation
	messages []MessageView

	// Agent
	agentLoop *agent.Loop

	// Status
	status       string
	isProcessing bool
	errorMsg     string
	help         string

	// Approval
	approvalRequest *ApprovalView
}

// MessageView represents a displayed message
type MessageView struct {
	Role    string
	Content string
}

// ApprovalView represents a pending approval request
type ApprovalView struct {
	ToolName    string
	Description string
	Details     string
	RiskLevel   string
}

// NewModel creates a new TUI model
func NewModel(ctx context.Context, agentLoop *agent.Loop) Model {
	input := textinput.New()
	input.Placeholder = "Ask about your code..."
	input.CharLimit = 1000
	input.Width = 80
	input.Focus()
	input.Prompt = "> "

	return Model{
		ctx:       ctx,
		state:     StateIdle,
		input:     input,
		messages:  make([]MessageView, 0),
		agentLoop: agentLoop,
		help:      "Enter submit  Ctrl+C quit",
	}
}

// Init initializes the TUI
func (m Model) Init() tea.Cmd {
	return textinput.Blink
}

// Update handles messages
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		model, cmd := m.handleKeyPress(msg)
		if cmd != nil {
			return model, cmd
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.input.Width = max(20, msg.Width-8)
		return m, nil

	case AgentResponseMsg:
		return m.handleAgentResponse(msg)

	case ErrorMsg:
		m.state = StateError
		m.errorMsg = string(msg)
		return m, nil
	}

	if m.state != StateWaiting && m.state != StateApproving {
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m *Model) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyCtrlC, tea.KeyCtrlQ:
		return m, tea.Quit

	case tea.KeyEsc:
		if m.state == StateTyping && m.input.Value() == "" {
			m.state = StateIdle
			return m, nil
		}
		if m.state == StateError {
			m.state = StateIdle
			m.errorMsg = ""
			return m, nil
		}

	case tea.KeyEnter:
		if m.state == StateTyping || m.state == StateIdle || m.state == StateError {
			return m, m.submitMessage()
		}

	case tea.KeyRunes:
		if m.state == StateIdle || m.state == StateError {
			m.state = StateTyping
			m.errorMsg = ""
		}
	}

	return m, nil
}

func (m *Model) submitMessage() tea.Cmd {
	content := m.input.Value()
	if content == "" {
		return nil
	}

	// Add user message
	m.messages = append(m.messages, MessageView{
		Role:    "user",
		Content: content,
	})

	// Clear input
	m.input.SetValue("")
	m.state = StateWaiting
	m.isProcessing = true
	m.status = "Thinking..."

	return func() tea.Msg {
		resp, err := m.agentLoop.Run(m.ctx, agent.Request{Prompt: content})
		return AgentResponseMsg{
			Text:  resp.Text,
			Error: err,
		}
	}
}

func (m *Model) handleAgentResponse(msg AgentResponseMsg) (tea.Model, tea.Cmd) {
	m.isProcessing = false
	m.status = ""

	if msg.Error != nil {
		m.state = StateError
		m.errorMsg = msg.Error.Error()
		return m, nil
	}

	// Add assistant response
	m.messages = append(m.messages, MessageView{
		Role:    "assistant",
		Content: msg.Text,
	})

	m.state = StateIdle
	return m, nil
}

// AgentResponseMsg carries agent responses
type AgentResponseMsg struct {
	Text  string
	Error error
}

// ErrorMsg carries an error
type ErrorMsg string
