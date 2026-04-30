package agent

import (
	"context"
	"fmt"
	"strings"

	"github.com/asif/gocode-agent/internal/providers"
	"github.com/asif/gocode-agent/internal/tools"
	"github.com/asif/gocode-agent/internal/workspace"
)

// Loop manages the agent conversation and tool execution cycle
type Loop struct {
	provider   providers.Provider
	tools      *tools.ToolRegistry
	approval   *tools.ApprovalGate
	workspace  *workspace.Context
	session    *Session
	messages   []Message
	maxIterations int
}

// Config holds agent loop configuration
type Config struct {
	Provider      providers.Provider
	Tools         *tools.ToolRegistry
	ApprovalGate  *tools.ApprovalGate
	Workspace     *workspace.Context
	MaxIterations int
}

// NewLoop creates a new agent loop
func NewLoop(cfg Config) *Loop {
	maxIter := cfg.MaxIterations
	if maxIter == 0 {
		maxIter = 10 // Default max iterations
	}

	return &Loop{
		provider:      cfg.Provider,
		tools:         cfg.Tools,
		approval:      cfg.ApprovalGate,
		workspace:     cfg.Workspace,
		maxIterations: maxIter,
		messages:      make([]Message, 0),
	}
}

// Request represents a user request
type Request struct {
	Prompt string
}

// Response represents the agent response
type Response struct {
	Text      string
	ToolCalls []tools.ToolResult
	Done      bool
}

// Run executes the agent loop for a given request
func (l *Loop) Run(ctx context.Context, req Request) (Response, error) {
	if err := ctx.Err(); err != nil {
		return Response{}, err
	}

	prompt := strings.TrimSpace(req.Prompt)
	if prompt == "" {
		return Response{}, fmt.Errorf("prompt is required")
	}

	// Add user message
	l.messages = append(l.messages, NewUserMessage(prompt))

	// Run agent iterations
	for i := 0; i < l.maxIterations; i++ {
		select {
		case <-ctx.Done():
			return Response{}, ctx.Err()
		default:
		}

		// Get completion from provider
		resp, err := l.getCompletion(ctx)
		if err != nil {
			return Response{}, fmt.Errorf("completion error: %w", err)
		}

		// Check if there are tool calls to execute
		toolCalls := l.parseToolCalls(resp.Content)
		if len(toolCalls) == 0 {
			// No tool calls, return the response
			return Response{
				Text: resp.Content,
				Done: true,
			}, nil
		}

		// Execute tool calls
		results, err := l.executeToolCalls(ctx, toolCalls)
		if err != nil {
			return Response{}, fmt.Errorf("tool execution error: %w", err)
		}

		// Add tool results to conversation
		l.messages = append(l.messages, l.formatToolResults(toolCalls, results))
	}

	return Response{
		Text: "Maximum iterations reached",
		Done: false,
	}, nil
}

// RunInteractive runs the agent in interactive mode
func (l *Loop) RunInteractive(ctx context.Context) error {
	// Interactive loop implementation
	return nil
}

func (l *Loop) getCompletion(ctx context.Context) (providers.CompletionResponse, error) {
	messages := ToProviderMessages(l.messages)

	req := providers.CompletionRequest{
		Model:       "default",
		Messages:    messages,
		MaxTokens:   4096,
		Temperature: 0.7,
	}

	return l.provider.Complete(ctx, req)
}

func (l *Loop) parseToolCalls(content string) []MessageToolCall {
	// Simple parsing - look for tool call markers
	// In production, this would parse structured tool calls from the provider
	var toolCalls []MessageToolCall

	// Look for patterns like [tool:name]...[/tool]
	// This is a simplified implementation
	return toolCalls
}

func (l *Loop) executeToolCalls(ctx context.Context, calls []MessageToolCall) ([]tools.ToolResult, error) {
	results := make([]tools.ToolResult, len(calls))

	for i, call := range calls {
		tool, ok := l.tools.Get(call.Name)
		if !ok {
			results[i] = tools.ToolResult{
				Error: fmt.Errorf("unknown tool: %s", call.Name),
			}
			continue
		}

		// Check approval
		approvalReq := tools.ApprovalRequest{
			ToolName:    tool.Name(),
			Description: tool.Description(),
			Details:     fmt.Sprintf("%v", call.Input),
			RiskLevel:   tool.RiskLevel(),
		}

		approved, err := l.approval.CheckAndRequest(ctx, approvalReq)
		if err != nil {
			results[i] = tools.ToolResult{Error: err}
			continue
		}

		if !approved {
			results[i] = tools.ToolResult{
				Error: fmt.Errorf("tool call denied by user"),
			}
			continue
		}

		// Execute tool
		result, err := tool.Run(ctx, call.Input)
		results[i] = result
		if err != nil {
			results[i].Error = err
		}
	}

	return results, nil
}

func (l *Loop) formatToolResults(calls []MessageToolCall, results []tools.ToolResult) Message {
	var content strings.Builder
	content.WriteString("Tool execution results:\n\n")

	for i, call := range calls {
		result := results[i]
		content.WriteString(fmt.Sprintf("## %s\n", call.Name))
		if result.Error != nil {
			content.WriteString(fmt.Sprintf("Error: %v\n", result.Error))
		} else {
			content.WriteString(fmt.Sprintf("Output: %s\n", result.Output))
		}
		content.WriteString("\n")
	}

	return NewAssistantMessage(content.String())
}

// GetMessages returns the current conversation history
func (l *Loop) GetMessages() []Message {
	return l.messages
}

// ClearMessages resets the conversation
func (l *Loop) ClearMessages() {
	l.messages = make([]Message, 0)
}
