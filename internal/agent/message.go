package agent

import (
	"time"

	"github.com/asif/gocode-agent/internal/providers"
	"github.com/asif/gocode-agent/internal/tools"
)

// Message represents a conversation message
type Message struct {
	ID        string             `json:"id"`
	Role      string             `json:"role"` // "user" or "assistant"
	Content   string             `json:"content"`
	Timestamp time.Time          `json:"timestamp"`
	ToolCalls []MessageToolCall  `json:"tool_calls,omitempty"`
}

// MessageToolCall represents a tool call within a message
type MessageToolCall struct {
	ID       string         `json:"id"`
	Name     string         `json:"name"`
	Input    tools.ToolInput `json:"input"`
	Output   string         `json:"output,omitempty"`
	Error    string         `json:"error,omitempty"`
	Approved bool           `json:"approved"`
}

// ToProviderMessage converts to provider message format
func (m *Message) ToProviderMessage() providers.Message {
	return providers.Message{
		Role:    m.Role,
		Content: m.Content,
	}
}

// ToProviderMessages converts a slice of messages
func ToProviderMessages(messages []Message) []providers.Message {
	result := make([]providers.Message, len(messages))
	for i, msg := range messages {
		result[i] = msg.ToProviderMessage()
	}
	return result
}

// NewUserMessage creates a new user message
func NewUserMessage(content string) Message {
	return Message{
		ID:        generateID(),
		Role:      "user",
		Content:   content,
		Timestamp: time.Now(),
	}
}

// NewAssistantMessage creates a new assistant message
func NewAssistantMessage(content string) Message {
	return Message{
		ID:        generateID(),
		Role:      "assistant",
		Content:   content,
		Timestamp: time.Now(),
	}
}

// NewToolCallMessage creates a message with tool calls
func NewToolCallMessage(content string, toolCalls []MessageToolCall) Message {
	return Message{
		ID:        generateID(),
		Role:      "assistant",
		Content:   content,
		Timestamp: time.Now(),
		ToolCalls: toolCalls,
	}
}

// generateID creates a unique ID
func generateID() string {
	return time.Now().Format("20060102150405") + "-" + randomString(8)
}

func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[time.Now().UnixNano()%int64(len(letters))]
	}
	return string(b)
}
