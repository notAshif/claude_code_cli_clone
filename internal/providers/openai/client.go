package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/asif/gocode-agent/internal/providers"
)

// Client implements the OpenAI provider
type Client struct {
	apiKey  string
	baseURL string
	model   string
	client  *http.Client
}

// Config holds OpenAI client configuration
type Config struct {
	APIKey  string
	BaseURL string
	Model   string
}

// New creates a new OpenAI client
func New(cfg Config) *Client {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
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
	return "openai"
}

// Complete sends a completion request to OpenAI
func (c *Client) Complete(ctx context.Context, req providers.CompletionRequest) (providers.CompletionResponse, error) {
	if c.apiKey == "" {
		return providers.CompletionResponse{}, fmt.Errorf("OpenAI API key not configured")
	}

	openaiReq := c.buildRequest(req)

	body, err := json.Marshal(openaiReq)
	if err != nil {
		return providers.CompletionResponse{}, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return providers.CompletionResponse{}, err
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return providers.CompletionResponse{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return providers.CompletionResponse{}, fmt.Errorf("OpenAI API error (%d): %s", resp.StatusCode, string(body))
	}

	var openaiResp OpenAIResponse
	if err := json.NewDecoder(resp.Body).Decode(&openaiResp); err != nil {
		return providers.CompletionResponse{}, err
	}

	return c.convertResponse(openaiResp), nil
}

// Stream sends a streaming completion request
func (c *Client) Stream(ctx context.Context, req providers.CompletionRequest) (<-chan providers.CompletionEvent, error) {
	// Streaming implementation would use SSE
	// For now, return a channel with single complete response
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

func (c *Client) buildRequest(req providers.CompletionRequest) OpenAIRequest {
	messages := make([]OpenAIMessage, len(req.Messages))
	for i, msg := range req.Messages {
		messages[i] = OpenAIMessage{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}

	return OpenAIRequest{
		Model:       c.getModel(req.Model),
		Messages:    messages,
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
		Stream:      req.Stream,
	}
}

func (c *Client) getModel(requestModel string) string {
	if requestModel != "" && requestModel != "default" {
		return requestModel
	}
	if c.model != "" {
		return c.model
	}
	return "gpt-4"
}

func (c *Client) convertResponse(resp OpenAIResponse) providers.CompletionResponse {
	var content string
	if len(resp.Choices) > 0 {
		content = resp.Choices[0].Message.Content
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

// OpenAIRequest is the request format for OpenAI API
type OpenAIRequest struct {
	Model       string          `json:"model"`
	Messages    []OpenAIMessage `json:"messages"`
	MaxTokens   int             `json:"max_tokens,omitempty"`
	Temperature float32         `json:"temperature,omitempty"`
	Stream      bool            `json:"stream,omitempty"`
}

// OpenAIMessage is the message format for OpenAI API
type OpenAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// OpenAIResponse is the response format from OpenAI API
type OpenAIResponse struct {
	ID      string `json:"id"`
	Model   string `json:"model"`
	Choices []struct {
		Index   int `json:"index"`
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Usage struct {
		InputTokens  int `json:"prompt_tokens"`
		OutputTokens int `json:"completion_tokens"`
	} `json:"usage"`
}
