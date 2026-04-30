package sdk

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/asif/gocode-agent/internal/agent"
	"github.com/asif/gocode-agent/internal/providers"
	"github.com/asif/gocode-agent/internal/providers/claude"
	"github.com/asif/gocode-agent/internal/providers/ollama"
	"github.com/asif/gocode-agent/internal/providers/openai"
	"github.com/asif/gocode-agent/internal/tools"
	"github.com/asif/gocode-agent/internal/workspace"
)

// Client is the SDK entry point for running agent requests.
type Client struct {
	cfg  ClientConfig
	loop *agent.Loop
}

// NewClient creates and wires a ready-to-use SDK client.
func NewClient(cfg ClientConfig) (*Client, error) {
	cfg = withDefaults(cfg)

	provider, err := buildProvider(cfg)
	if err != nil {
		return nil, err
	}

	wd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	absWd, err := filepath.Abs(wd)
	if err != nil {
		return nil, err
	}

	workspaceCtx, err := workspace.BuildContext(absWd)
	if err != nil {
		return nil, err
	}

	registry := tools.NewToolRegistry()
	registry.Register(tools.NewFilesystemTool(1024*1024, []string{absWd}))
	registry.Register(tools.NewWriteFileTool([]string{absWd}))
	registry.Register(tools.NewSearchTool(absWd, 1000, 1024*1024, workspace.DefaultIgnoreDirs, workspace.DefaultIgnoreFiles))
	registry.Register(tools.NewShellTool(nil, 0, absWd))

	approval := tools.NewApprovalGate(
		tools.ApprovalPolicy(cfg.ApprovalShell),
		tools.ApprovalPolicy(cfg.ApprovalWrite),
		absWd,
	)

	loop := agent.NewLoop(agent.Config{
		Provider:     provider,
		Tools:        registry,
		ApprovalGate: approval,
		Workspace:    workspaceCtx,
		MaxIterations: cfg.MaxRetries + 1,
	})

	return &Client{
		cfg:  cfg,
		loop: loop,
	}, nil
}

// Run executes a single prompt and returns a normalized SDK response.
func (c *Client) Run(ctx context.Context, prompt string) (AgentResponse, error) {
	resp, err := c.loop.Run(ctx, agent.Request{Prompt: prompt})
	if err != nil {
		return AgentResponse{}, err
	}

	return AgentResponse{
		Text: resp.Text,
		Done: resp.Done,
	}, nil
}

// ClearHistory resets in-memory conversation state.
func (c *Client) ClearHistory() {
	c.loop.ClearMessages()
}

func withDefaults(cfg ClientConfig) ClientConfig {
	def := DefaultConfig()
	if cfg.Timeout == 0 {
		cfg.Timeout = def.Timeout
	}
	if cfg.MaxRetries == 0 {
		cfg.MaxRetries = def.MaxRetries
	}
	if cfg.Temperature == 0 {
		cfg.Temperature = def.Temperature
	}
	if cfg.MaxTokens == 0 {
		cfg.MaxTokens = def.MaxTokens
	}
	if cfg.ApprovalShell == "" {
		cfg.ApprovalShell = def.ApprovalShell
	}
	if cfg.ApprovalWrite == "" {
		cfg.ApprovalWrite = def.ApprovalWrite
	}
	if cfg.Provider == "" {
		cfg.Provider = "ollama"
	}
	return cfg
}

func buildProvider(cfg ClientConfig) (providers.Provider, error) {
	switch cfg.Provider {
	case "openai":
		apiKey := cfg.APIKey
		if apiKey == "" {
			apiKey = os.Getenv("OPENAI_API_KEY")
		}
		if apiKey == "" {
			return nil, fmt.Errorf("OPENAI_API_KEY not configured")
		}
		return openai.New(openai.Config{
			APIKey:  apiKey,
			BaseURL: cfg.BaseURL,
			Model:   cfg.Model,
		}), nil
	case "claude":
		apiKey := cfg.APIKey
		if apiKey == "" {
			apiKey = os.Getenv("ANTHROPIC_API_KEY")
		}
		if apiKey == "" {
			return nil, fmt.Errorf("ANTHROPIC_API_KEY not configured")
		}
		return claude.New(claude.Config{
			APIKey:  apiKey,
			BaseURL: cfg.BaseURL,
			Model:   cfg.Model,
		}), nil
	case "ollama":
		return ollama.New(ollama.Config{
			BaseURL: cfg.BaseURL,
			Model:   cfg.Model,
		}), nil
	default:
		return nil, fmt.Errorf("unknown provider: %s", cfg.Provider)
	}
}