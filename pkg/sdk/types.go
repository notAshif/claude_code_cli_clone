package sdk

import (
	"time"

	"github.com/asif/gocode-agent/internal/providers"
	"github.com/asif/gocode-agent/internal/tools"
)

// ClientConfig holds configuration for the SDK client
type ClientConfig struct {
	Provider      string
	Model         string
	APIKey        string
	BaseURL       string
	Timeout       time.Duration
	MaxRetries    int
	Temperature   float32
	MaxTokens     int
	ApprovalShell string
	ApprovalWrite string
}

// Session represents an agent conversation session
type Session struct {
	ID        string
	CreatedAt time.Time
	UpdatedAt time.Time
	Messages  []providers.Message
	ToolCalls []ToolCallRecord
	Metadata  SessionMetadata
}

// SessionMetadata holds session metadata
type SessionMetadata struct {
	WorkingDir  string
	Provider    string
	Model       string
	TokenUsage  TokenUsage
}

// TokenUsage tracks token consumption
type TokenUsage struct {
	InputTokens  int
	OutputTokens int
	TotalTokens  int
}

// ToolCallRecord logs a tool invocation
type ToolCallRecord struct {
	ID        string
	Name      string
	Input     tools.ToolInput
	Output    string
	Error     string
	Duration  time.Duration
	Approved  bool
	Timestamp time.Time
}

// AgentResponse is the unified response type
type AgentResponse struct {
	Text      string
	ToolCalls []ToolCallRecord
	Done      bool
	Error     error
}

// DefaultConfig returns a default client configuration
func DefaultConfig() ClientConfig {
	return ClientConfig{
		Timeout:       60 * time.Second,
		MaxRetries:    3,
		Temperature:   0.7,
		MaxTokens:     4096,
		ApprovalShell: "always",
		ApprovalWrite: "always",
	}
}
