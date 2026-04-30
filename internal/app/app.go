package app

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
	"github.com/asif/gocode-agent/internal/tui"
	"github.com/asif/gocode-agent/internal/workspace"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

// App is the main application
type App struct {
	root             *cobra.Command
	cfg              Config
	configPath       string
	providerOverride string
	modelOverride    string
}

// New creates a new App instance
func New() (*App, error) {
	cfg, err := LoadCliConfig(DefaultConfigPath)
	if err != nil {
		return nil, err
	}
	a := &App{
		cfg:        cfg,
		configPath: DefaultConfigPath,
	}
	a.root = a.buildRootCommand()
	return a, nil
}

// Run executes the application
func (a *App) Run(ctx context.Context, args []string) error {
	a.root.SetArgs(args)
	a.root.SetContext(ctx)
	return a.root.Execute()
}

// buildRootCommand creates the main cobra command
func (a *App) buildRootCommand() *cobra.Command {
	root := &cobra.Command{
		Use:   "agent",
		Short: "Terminal Assistant - AI-powered coding assistant",
		Long:  "Go Code Agent is a terminal-first AI coding assistant with TUI frontend.",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return a.loadConfigForCommand()
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return a.runTUI(cmd.Context())
		},
	}

	root.PersistentFlags().StringVar(&a.configPath, "config", a.configPath, "config file path")
	root.PersistentFlags().StringVar(&a.providerOverride, "provider", "", "provider to use for this run (ollama, openai, claude)")
	root.PersistentFlags().StringVar(&a.modelOverride, "model", "", "model to use for this run")

	root.AddCommand(
		a.newChatCommand(),
		a.newRunCommand(),
		a.newConfigCommand(),
		a.newDoctorCommand(),
	)

	return root
}

// newChatCommand creates the chat subcommand
func (a *App) newChatCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "chat",
		Short: "Start interactive chat mode",
		RunE: func(cmd *cobra.Command, args []string) error {
			return a.runTUI(cmd.Context())
		},
	}
}

// newRunCommand creates the run subcommand
func (a *App) newRunCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "run [prompt]",
		Short: "Run a one-shot instruction",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return a.runOnce(cmd.Context(), args[0])
		},
	}
}

// newConfigCommand creates the config subcommand
func (a *App) newConfigCommand() *cobra.Command {
	configCmd := &cobra.Command{
		Use:   "config",
		Short: "Manage configuration",
	}

	configCmd.AddCommand(
		&cobra.Command{
			Use:   "show",
			Short: "Show active configuration",
			RunE: func(cmd *cobra.Command, args []string) error {
				fmt.Printf("Provider: %s\n", a.cfg.Provider)
				fmt.Printf("Model: %s\n", a.cfg.Model)
				fmt.Printf("Approval (Shell): %s\n", a.cfg.Approval.Shell)
				fmt.Printf("Approval (Write): %s\n", a.cfg.Approval.WriteFile)
				fmt.Printf("Workspace Max Files: %d\n", a.cfg.Workspace.MaxFiles)
				fmt.Printf("Storage Path: %s\n", a.cfg.Storage.Path)
				return nil
			},
		},
		&cobra.Command{
			Use:   "providers",
			Short: "List available providers",
			RunE: func(cmd *cobra.Command, args []string) error {
				for _, provider := range SupportedProviders() {
					model, _ := DefaultModelForProvider(provider)
					marker := " "
					if provider == a.cfg.Provider {
						marker = "*"
					}
					fmt.Fprintf(cmd.OutOrStdout(), "%s %s (default model: %s)\n", marker, provider, model)
				}
				return nil
			},
		},
		&cobra.Command{
			Use:   "models [provider]",
			Short: "Show productive default models for providers",
			Args:  cobra.MaximumNArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				provider := a.cfg.Provider
				if len(args) == 1 {
					provider = args[0]
				}
				if provider == "all" {
					for _, p := range SupportedProviders() {
						model, _ := DefaultModelForProvider(p)
						fmt.Fprintf(cmd.OutOrStdout(), "%s: %s\n", p, model)
					}
					return nil
				}
				model, ok := DefaultModelForProvider(provider)
				if !ok {
					return fmt.Errorf("unknown provider %q", provider)
				}
				fmt.Fprintf(cmd.OutOrStdout(), "%s: %s\n", provider, model)
				return nil
			},
		},
		&cobra.Command{
			Use:   "use [provider] [model]",
			Short: "Select and save a provider/model",
			Args:  cobra.RangeArgs(1, 2),
			RunE: func(cmd *cobra.Command, args []string) error {
				model := ""
				if len(args) == 2 {
					model = args[1]
				}
				if err := a.cfg.UseProviderModel(args[0], model); err != nil {
					return err
				}
				if err := SaveCliConfig(a.configPath, a.cfg); err != nil {
					return err
				}
				fmt.Fprintf(cmd.OutOrStdout(), "Using %s / %s\n", a.cfg.Provider, a.cfg.Model)
				return nil
			},
		},
		&cobra.Command{
			Use:   "set [key] [value]",
			Short: "Set a configuration value",
			Args:  cobra.ExactArgs(2),
			RunE: func(cmd *cobra.Command, args []string) error {
				if err := a.cfg.Set(args[0], args[1]); err != nil {
					return err
				}
				if err := SaveCliConfig(a.configPath, a.cfg); err != nil {
					return err
				}
				fmt.Fprintf(cmd.OutOrStdout(), "Saved %s = %s\n", args[0], args[1])
				return nil
			},
		},
	)

	return configCmd
}

func (a *App) loadConfigForCommand() error {
	cfg, err := LoadCliConfig(a.configPath)
	if err != nil {
		return err
	}

	if a.providerOverride != "" {
		model := ""
		if a.modelOverride != "" {
			model = a.modelOverride
		}
		if err := cfg.UseProviderModel(a.providerOverride, model); err != nil {
			return err
		}
	} else if a.modelOverride != "" {
		if err := cfg.Set("model", a.modelOverride); err != nil {
			return err
		}
	}

	a.cfg = cfg
	return nil
}

// newDoctorCommand creates the doctor subcommand
func (a *App) newDoctorCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Check environment and configuration health",
		RunE: func(cmd *cobra.Command, args []string) error {
			return a.runDoctor(cmd.OutOrStdout())
		},
	}
}

// runTUI starts the interactive TUI
func (a *App) runTUI(ctx context.Context) error {
	// Initialize components
	agentLoop, err := a.createAgentLoop()
	if err != nil {
		return err
	}

	// Create and run TUI
	model := tui.NewModel(ctx, agentLoop)
	p := tea.NewProgram(&model, tea.WithAltScreen())
	_, err = p.Run()
	return err
}

// runOnce executes a single request
func (a *App) runOnce(ctx context.Context, prompt string) error {
	agentLoop, err := a.createAgentLoop()
	if err != nil {
		return err
	}

	resp, err := agentLoop.Run(ctx, agent.Request{Prompt: prompt})
	if err != nil {
		return err
	}

	fmt.Println(resp.Text)
	return nil
}

// runDoctor runs diagnostics
func (a *App) runDoctor(out interface{ Write([]byte) (int, error) }) error {
	var issues []string

	// Check provider configuration
	if a.cfg.Provider == "" {
		issues = append(issues, "No provider configured")
	}

	// Check API keys based on provider
	switch a.cfg.Provider {
	case "openai":
		if os.Getenv("OPENAI_API_KEY") == "" {
			issues = append(issues, "OPENAI_API_KEY not set")
		}
	case "claude":
		if os.Getenv("ANTHROPIC_API_KEY") == "" {
			issues = append(issues, "ANTHROPIC_API_KEY not set")
		}
	case "ollama":
		// Ollama doesn't require API key, check if server is reachable
	}

	// Check workspace
	wd, err := os.Getwd()
	if err != nil {
		issues = append(issues, fmt.Sprintf("Cannot get working directory: %v", err))
	}

	// Check git
	_, err = workspace.GetGitInfo(wd)
	if err != nil {
		issues = append(issues, "Not a git repository (or git not installed)")
	}

	// Output results
	output := "=== Go Code Agent Doctor ===\n\n"
	output += fmt.Sprintf("Provider: %s\n", a.cfg.Provider)
	output += fmt.Sprintf("Model: %s\n", a.cfg.Model)
	output += fmt.Sprintf("Working Dir: %s\n\n", wd)

	if len(issues) == 0 {
		output += "Status: All checks passed!\n"
	} else {
		output += "Issues found:\n"
		for _, issue := range issues {
			output += fmt.Sprintf("  - %s\n", issue)
		}
	}

	_, err = out.Write([]byte(output))
	return err
}

// createAgentLoop initializes the agent loop with all components
func (a *App) createAgentLoop() (*agent.Loop, error) {
	// Create provider
	provider, err := a.createProvider()
	if err != nil {
		return nil, err
	}

	// Create workspace context
	wd, _ := os.Getwd()
	workspaceCtx, err := workspace.BuildContext(wd)
	if err != nil {
		return nil, err
	}

	// Create tool registry
	toolRegistry := tools.NewToolRegistry()

	// Register tools
	absWd, _ := filepath.Abs(wd)
	toolRegistry.Register(tools.NewFilesystemTool(1024*1024, []string{absWd}))
	toolRegistry.Register(tools.NewWriteFileTool([]string{absWd}))
	toolRegistry.Register(tools.NewSearchTool(absWd, 1000, 1024*1024, workspace.DefaultIgnoreDirs, workspace.DefaultIgnoreFiles))
	toolRegistry.Register(tools.NewShellTool(nil, 0, absWd))

	// Create approval gate
	approvalGate := tools.NewApprovalGate(
		tools.ApprovalPolicy(a.cfg.Approval.Shell),
		tools.ApprovalPolicy(a.cfg.Approval.WriteFile),
		absWd,
	)

	// Create agent loop
	loop := agent.NewLoop(agent.Config{
		Provider:     provider,
		Tools:        toolRegistry,
		ApprovalGate: approvalGate,
		Workspace:    workspaceCtx,
	})

	return loop, nil
}

// createProvider creates the configured AI provider
func (a *App) createProvider() (providers.Provider, error) {
	switch a.cfg.Provider {
	case "openai":
		apiKey := os.Getenv("OPENAI_API_KEY")
		if apiKey == "" {
			return nil, fmt.Errorf("OPENAI_API_KEY environment variable not set")
		}
		return openai.New(openai.Config{
			APIKey: apiKey,
			Model:  a.cfg.Model,
		}), nil

	case "claude":
		apiKey := os.Getenv("ANTHROPIC_API_KEY")
		if apiKey == "" {
			return nil, fmt.Errorf("ANTHROPIC_API_KEY environment variable not set")
		}
		return claude.New(claude.Config{
			APIKey: apiKey,
			Model:  a.cfg.Model,
		}), nil

	case "ollama":
		return ollama.New(ollama.Config{
			Model: a.cfg.Model,
		}), nil

	default:
		return nil, fmt.Errorf("unknown provider: %s", a.cfg.Provider)
	}
}
