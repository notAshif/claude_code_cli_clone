# Go Code Agent

Go Code Agent is a terminal-first AI coding assistant written in Go. It is built as a portable CLI/TUI application with provider-flexible model adapters, workspace-aware context, approval-gated tools, and a small SDK surface for embedding agent runs in other Go programs.

The project is inspired by coding agents such as Claude Code and Codex, but keeps the implementation compact and understandable: one Go module, clear internal packages, and a single binary entrypoint.

## Features

- Interactive terminal UI powered by Bubble Tea.
- Cobra-based CLI with chat, one-shot run, config, provider, model, and doctor commands.
- Provider adapters for Ollama, OpenAI, and Anthropic Claude behind one interface.
- Workspace scanning with ignore rules for large or generated directories.
- Tool registry for file reads, file writes, search, and shell execution.
- Approval gate for risky actions such as shell commands and file writes.
- SQLite storage layer for future resumable sessions and message history.
- Public Go SDK under `pkg/sdk` for programmatic one-shot agent runs.

## Project Status

This repository is an MVP scaffold with working CLI wiring, provider clients, TUI chat flow, config management, workspace scanning, tools, and storage primitives.

Some advanced agent behaviors are intentionally still simple:

- Tool-call parsing is currently a placeholder in `internal/agent/loop.go`.
- Provider streaming falls back to a single completed response event.
- SQLite storage is implemented but not yet connected to the main CLI session lifecycle.
- The TUI supports chat input and responses, but full visual approval prompts are not yet integrated.

## Architecture

```txt
.
+-- cmd/
|   +-- agent/              # CLI entrypoint
+-- configs/
|   +-- config.yaml         # Default local configuration
+-- internal/
|   +-- agent/              # Agent loop, message state, planning/session types
|   +-- app/                # Cobra commands, config loading, app wiring
|   +-- logs/               # Logging helpers
|   +-- providers/          # Provider interface and adapters
|   |   +-- claude/
|   |   +-- ollama/
|   |   +-- openai/
|   +-- storage/            # SQLite session/message storage
|   +-- tools/              # File, search, shell, and approval tools
|   +-- tui/                # Bubble Tea terminal UI
|   +-- workspace/          # Repo scanner, git info, workspace context
+-- pkg/
|   +-- sdk/                # Public Go SDK wrapper
+-- go.mod
+-- go.sum
+-- LICENSE
+-- PLAN.md
+-- README.md
```

## How It Works

1. `cmd/agent/main.go` creates the application and passes CLI args into the app layer.
2. `internal/app` loads YAML config, applies command-line overrides, builds Cobra commands, and wires runtime dependencies.
3. The configured provider is created from `internal/providers`.
4. `internal/workspace` builds a lightweight view of the current repository, including git metadata and a file tree.
5. `internal/tools` registers read, write, search, and shell capabilities scoped to the current workspace.
6. `internal/agent` sends conversation messages to the selected provider and coordinates future tool execution.
7. `internal/tui` provides an interactive terminal chat experience.

## Requirements

- Go `1.24.2` or newer, matching `go.mod`.
- Git, for repository diagnostics and workspace metadata.
- One supported model provider:
  - Ollama running locally, or
  - OpenAI API key, or
  - Anthropic API key.

## Installation

Clone the repository and download dependencies:

```powershell
git clone <repo-url>
cd Claude_cli
go mod download
```

Build the CLI:

```powershell
go build -o agent.exe ./cmd/agent
```

Run directly without building:

```powershell
go run ./cmd/agent
```

## Quick Start

The default config uses Ollama with `llama3.1:8b`.

Start Ollama and pull the model:

```powershell
ollama pull llama3.1:8b
ollama serve
```

Launch the interactive TUI:

```powershell
go run ./cmd/agent
```

Run a one-shot prompt:

```powershell
go run ./cmd/agent run "Explain this repository structure"
```

Check environment health:

```powershell
go run ./cmd/agent doctor
```

## CLI Commands

```txt
agent
agent chat
agent run [prompt]
agent config show
agent config providers
agent config models [provider|all]
agent config use [provider] [model]
agent config set [key] [value]
agent doctor
```

Global flags:

```txt
--config string      Config file path
--provider string    Provider override: ollama, openai, or claude
--model string       Model override for this run
```

Examples:

```powershell
go run ./cmd/agent config show
go run ./cmd/agent config providers
go run ./cmd/agent config models all
go run ./cmd/agent config use openai gpt-4o-mini
go run ./cmd/agent --provider ollama --model llama3.1:8b run "Summarize the codebase"
```

## Configuration

The default configuration file is [configs/config.yaml](configs/config.yaml).

```yaml
provider: ollama
model: llama3.1:8b

approval:
  shell: always
  write_files: always

workspace:
  max_files: 5000
  max_file_bytes: 200000

storage:
  path: .agent/sessions.db
```

Supported config keys for `agent config set`:

- `provider`
- `model`
- `approval.shell`
- `approval.write_files`
- `workspace.max_files`
- `workspace.max_file_bytes`
- `storage.path`

## Providers

### Ollama

Ollama is the default provider and does not require an API key.

```powershell
go run ./cmd/agent config use ollama llama3.1:8b
go run ./cmd/agent run "What files are in this project?"
```

Default base URL:

```txt
http://localhost:11434
```

### OpenAI

Set `OPENAI_API_KEY` before running the agent:

```powershell
$env:OPENAI_API_KEY = "your-api-key"
go run ./cmd/agent config use openai gpt-4o-mini
go run ./cmd/agent run "Review the agent loop"
```

Default base URL:

```txt
https://api.openai.com/v1
```

### Anthropic Claude

Set `ANTHROPIC_API_KEY` before running the agent:

```powershell
$env:ANTHROPIC_API_KEY = "your-api-key"
go run ./cmd/agent config use claude claude-3-5-sonnet-20241022
go run ./cmd/agent run "Explain the provider abstraction"
```

Default base URL:

```txt
https://api.anthropic.com/v1
```

## Tools

The agent registers four core tools:

| Tool | Risk | Purpose |
| --- | --- | --- |
| `read_file` | Low | Read a file inside the workspace. |
| `write_file` | High | Create or overwrite a file. |
| `search_files` | Low | Search filenames or file contents. |
| `run_shell` | High | Execute shell commands in the workspace. |

Tools are scoped to the current working directory when the app starts.

## Safety Model

The safety model is centered on workspace boundaries and explicit approvals.

- File reads are limited to allowed workspace directories.
- File writes are marked high risk.
- Shell execution is marked high risk.
- `approval.shell: always` requires confirmation before shell commands.
- `approval.write_files: always` requires confirmation before writes.
- Low-risk tools can be auto-approved.

The approval prompt supports:

```txt
y       approve once
n       deny
auto    approve this tool automatically for the rest of the session
```

## Workspace Context

The workspace package builds a compact repository summary:

- Absolute root directory.
- Current working directory.
- Git branch and dirty/clean state when available.
- File tree up to a shallow depth.
- Selected environment values.

The scanner ignores common generated or heavy paths such as:

```txt
.git, node_modules, vendor, dist, build, target, .venv, bin, obj
```

## SDK Usage

The `pkg/sdk` package exposes a simple client for running prompts from Go code.

```go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/asif/gocode-agent/pkg/sdk"
)

func main() {
	client, err := sdk.NewClient(sdk.ClientConfig{
		Provider: "ollama",
		Model:    "llama3.1:8b",
	})
	if err != nil {
		log.Fatal(err)
	}

	resp, err := client.Run(context.Background(), "Summarize this workspace")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(resp.Text)
}
```

## Development

Format the code:

```powershell
go fmt ./...
```

Run tests:

```powershell
go test ./...
```

Run diagnostics:

```powershell
go run ./cmd/agent doctor
```

Build:

```powershell
go build ./cmd/agent
```

## Key Packages

| Package | Responsibility |
| --- | --- |
| `internal/app` | CLI commands, config, dependency wiring. |
| `internal/agent` | Conversation loop and tool orchestration. |
| `internal/providers` | Shared model-provider interface. |
| `internal/providers/ollama` | Ollama chat adapter. |
| `internal/providers/openai` | OpenAI chat completions adapter. |
| `internal/providers/claude` | Anthropic messages adapter. |
| `internal/tools` | Tool interfaces and built-in tools. |
| `internal/workspace` | Repo scanning, git state, context summaries. |
| `internal/tui` | Bubble Tea terminal interface. |
| `internal/storage` | SQLite-backed session persistence. |
| `pkg/sdk` | Public API for embedding the agent. |

## Roadmap

- Structured provider-native tool calling.
- Streaming responses in the TUI.
- Fully integrated approval UI inside Bubble Tea.
- Persist and resume chat sessions using SQLite.
- Add tests for config, tools, providers, and workspace scanning.
- Add CI for formatting, vetting, and tests.
- Package release binaries for Windows, macOS, and Linux.
- Add plugin support for custom providers and tools.

## Troubleshooting

### `OPENAI_API_KEY environment variable not set`

Set the variable before using the OpenAI provider:

```powershell
$env:OPENAI_API_KEY = "your-api-key"
```

### `ANTHROPIC_API_KEY environment variable not set`

Set the variable before using the Claude provider:

```powershell
$env:ANTHROPIC_API_KEY = "your-api-key"
```

### Ollama connection errors

Make sure Ollama is installed, the server is running, and the configured model exists locally:

```powershell
ollama serve
ollama pull llama3.1:8b
```

### `Not a git repository`

`agent doctor` reports this when the current directory is not inside a git repository or Git is not installed. The app can still run, but git metadata will be unavailable.

## License

See [LICENSE](LICENSE).
