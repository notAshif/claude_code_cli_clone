package ollama

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/asif/gocode-agent/internal/providers"
)

// Client implements the Ollama provider
type Client struct {
	baseURL string
	model   string
	client  *http.Client
}

// Config holds Ollama client configuration
type Config struct {
	BaseURL string
	Model   string
}

// New creates a new Ollama client
func New(cfg Config) *Client {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}

	return &Client{
		baseURL: baseURL,
		model:   cfg.Model,
		client:  &http.Client{},
	}
}

// Name returns the provider name
func (c *Client) Name() string {
	return "ollama"
}

// Complete sends a completion request to Ollama
func (c *Client) Complete(ctx context.Context, req providers.CompletionRequest) (providers.CompletionResponse, error) {
	ollamaReq := c.buildRequest(req)

	body, err := json.Marshal(ollamaReq)
	if err != nil {
		return providers.CompletionResponse{}, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/api/chat", bytes.NewReader(body))
	if err != nil {
		return providers.CompletionResponse{}, err
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return providers.CompletionResponse{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return providers.CompletionResponse{}, fmt.Errorf("Ollama API error (%d): %s", resp.StatusCode, string(body))
	}

	var ollamaResp OllamaResponse
	if err := json.NewDecoder(resp.Body).Decode(&ollamaResp); err != nil {
		return providers.CompletionResponse{}, err
	}

	return c.convertResponse(ollamaResp), nil
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

func (c *Client) buildRequest(req providers.CompletionRequest) OllamaRequest {
	messages := make([]OllamaMessage, len(req.Messages))
	for i, msg := range req.Messages {
		messages[i] = OllamaMessage{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}

	return OllamaRequest{
		Model:    c.getModel(req.Model),
		Messages: messages,
		Stream:   false,
	}
}

func (c *Client) getModel(requestModel string) string {
	if requestModel != "" && requestModel != "default" {
		return requestModel
	}
	if c.model != "" {
		return c.model
	}
	return "llama3.1:8b"
}

func (c *Client) convertResponse(resp OllamaResponse) providers.CompletionResponse {
	return providers.CompletionResponse{
		ID:      resp.Model,
		Model:   resp.Model,
		Content: resp.Message.Content,
		Usage: providers.Usage{
			InputTokens:  resp.PromptEvalCount,
			OutputTokens: resp.EvalCount,
		},
	}
}

// OllamaRequest is the request format for Ollama API
type OllamaRequest struct {
	Model    string          `json:"model"`
	Messages []OllamaMessage `json:"messages"`
	Stream   bool            `json:"stream"`
}

// OllamaMessage is the message format for Ollama API
type OllamaMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// OllamaResponse is the response format from Ollama API
type OllamaResponse struct {
	Model     string `json:"model"`
	Message   struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	} `json:"message"`
	Done            bool `json:"done"`
	PromptEvalCount int  `json:"prompt_eval_count"`
	EvalCount       int  `json:"eval_count"`
}
