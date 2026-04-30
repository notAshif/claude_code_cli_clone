package providers

import (
	"context"
	"encoding/json"
)

// Message represents a conversation message
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// CompletionRequest defines the request structure for AI providers
type CompletionRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
	Temperature float32   `json:"temperature,omitempty"`
	Stream      bool      `json:"stream,omitempty"`
	Tools       []ToolDef `json:"tools,omitempty"`
}

// ToolDef defines a tool available to the AI
type ToolDef struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"input_schema"`
}

// CompletionResponse defines the response structure from AI providers
type CompletionResponse struct {
	ID      string   `json:"id"`
	Model   string   `json:"model"`
	Content string   `json:"content"`
	Usage   Usage    `json:"usage"`
	StopReason string `json:"stop_reason,omitempty"`
}

// Usage tracks token consumption
type Usage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// CompletionEvent represents a streaming event
type CompletionEvent struct {
	Type  string
	Text  string
	Done  bool
	Error error
}

// Provider interface defines the contract for AI providers
type Provider interface {
	Name() string
	Complete(ctx context.Context, req CompletionRequest) (CompletionResponse, error)
	Stream(ctx context.Context, req CompletionRequest) (<-chan CompletionEvent, error)
}
