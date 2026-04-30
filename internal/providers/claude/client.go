package claude

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/asif/gocode-agent/internal/providers"
)

// Client implements the Anthropic Claude provider
type Client struct {
	apiKey  string
	baseURL string
	model   string
	client  *http.Client
}

// Config holds Claude client configuration
type Config struct {
	APIKey  string
	BaseURL string
	Model   string
}

// New creates a new Claude client
func New(cfg Config) *Client {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = "https://api.anthropic.com/v1"
	}

	return &Client{
		apiKey:  cfg.APIKey,
		baseURL: baseURL,
		model:   cfg.Model,
		client:  &http.Client{},
	}
}

// Name returns the provider name
func (c *Client) Name() string {
	return "claude"
}

// Complete sends a completion request to Anthropic
func (c *Client) Complete(ctx context.Context, req providers.CompletionRequest) (providers.CompletionResponse, error) {
	if c.apiKey == "" {
		return providers.CompletionResponse{}, fmt.Errorf("Anthropic API key not configured")
	}

	anthropicReq := c.buildRequest(req)

	body, err := json.Marshal(anthropicReq)
	if err != nil {
		return providers.CompletionResponse{}, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/messages", bytes.NewReader(body))
	if err != nil {
		return providers.CompletionResponse{}, err
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", c.apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return providers.CompletionResponse{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return providers.CompletionResponse{}, fmt.Errorf("Anthropic API error (%d): %s", resp.StatusCode, string(body))
	}

	var anthropicResp ClaudeResponse
	if err := json.NewDecoder(resp.Body).Decode(&anthropicResp); err != nil {
		return providers.CompletionResponse{}, err
	}

	return c.convertResponse(anthropicResp), nil
}

// Stream sends a streaming completion request
func (c *Client) Stream(ctx context.Context, req providers.CompletionRequest) (<-chan providers.CompletionEvent, error) {
	ch := make(chan providers.CompletionEvent, 1)

	go func() {
		defer close(ch)
		resp, err := c.Complete(ctx, req)
		if err != nil {
			ch <- providers.CompletionEvent{
				Type:  "error",
				Error: err,
			}
			return
		}
		ch <- providers.CompletionEvent{
			Type: "content",
			Text: resp.Content,
		}
		ch <- providers.CompletionEvent{
			Type: "done",
		}
	}()

	return ch, nil
}

func (c *Client) buildRequest(req providers.CompletionRequest) ClaudeRequest {
	// Convert messages to Claude format
	messages := make([]ClaudeMessage, 0)
	var systemPrompt string

	for _, msg := range req.Messages {
		if msg.Role == "system" {
			systemPrompt = msg.Content
		} else {
			messages = append(messages, ClaudeMessage{
				Role:    msg.Role,
				Content: msg.Content,
			})
		}
	}

	return ClaudeRequest{
		Model:       c.getModel(req.Model),
		Messages:    messages,
		System:      systemPrompt,
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
	}
}

func (c *Client) getModel(requestModel string) string {
	if requestModel != "" && requestModel != "default" {
		return requestModel
	}
	if c.model != "" {
		return c.model
	}
	return "claude-3-sonnet-20240229"
}

func (c *Client) convertResponse(resp ClaudeResponse) providers.CompletionResponse {
	var content string
	if len(resp.Content) > 0 {
		content = resp.Content[0].Text
	}

	return providers.CompletionResponse{
		ID:      resp.ID,
		Model:   resp.Model,
		Content: content,
		Usage: providers.Usage{
			InputTokens:  resp.Usage.InputTokens,
			OutputTokens: resp.Usage.OutputTokens,
		},
	}
}

// ClaudeRequest is the request format for Anthropic API
type ClaudeRequest struct {
	Model       string          `json:"model"`
	Messages    []ClaudeMessage `json:"messages"`
	System      string          `json:"system,omitempty"`
	MaxTokens   int             `json:"max_tokens"`
	Temperature float32         `json:"temperature,omitempty"`
}

// ClaudeMessage is the message format for Anthropic API
type ClaudeMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ClaudeResponse is the response format from Anthropic API
type ClaudeResponse struct {
	ID      string `json:"id"`
	Model   string `json:"model"`
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	Usage struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}
