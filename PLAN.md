# Go CLI Coding Assistant Project Plan

## Summary

Build a portable Go project similar in spirit to Claude Code/Codex: a terminal-first coding assistant with a TUI frontend, Go backend orchestration layer, provider-flexible AI adapters, repo-aware context, shell/file tools, approval gates, and a clean structure suitable for GitHub and LinkedIn presentation.

The v1 will be a solid MVP: impressive enough to demo, small enough to finish, and designed so future features like plugins, multi-agent workers, or a web UI can be added cleanly.

## Architecture

Use a single Go module with clear frontend/backend separation:

```txt
go-code-agent/
├─ cmd/
│  └─ agent/
│     └─ main.go
├─ internal/
│  ├─ app/
│  │  ├─ app.go
│  │  └─ config.go
│  ├─ tui/
│  │  ├─ model.go
│  │  ├─ update.go
│  │  ├─ view.go
│  │  └─ styles.go
│  ├─ agent/
│  │  ├─ loop.go
│  │  ├─ planner.go
│  │  ├─ messages.go
│  │  └─ session.go
│  ├─ providers/
│  │  ├─ provider.go
│  │  ├─ openai/
│  │  │  └─ client.go
│  │  ├─ anthropic/
│  │  │  └─ client.go
│  │  └─ ollama/
│  │     └─ client.go
│  ├─ tools/
│  │  ├─ tool.go
│  │  ├─ filesystem.go
│  │  ├─ shell.go
│  │  ├─ search.go
│  │  └─ approval.go
│  ├─ workspace/
│  │  ├─ scanner.go
│  │  ├─ context.go
│  │  └─ git.go
│  ├─ storage/
│  │  ├─ store.go
│  │  └─ sqlite.go
│  └─ log/
│     └─ logger.go
├─ pkg/
│  └─ sdk/
│     ├─ client.go
│     └─ types.go
├─ configs/
│  └─ example.yaml
├─ docs/
│  ├─ ARCHITECTURE.md
│  ├─ DEMO.md
│  └─ ROADMAP.md
├─ examples/
│  └─ sample-session.md
├─ scripts/
│  ├─ dev.ps1
│  └─ test.ps1
├─ .github/
│  └─ workflows/
│     └─ ci.yml
├─ go.mod
├─ go.sum
├─ README.md
├─ LICENSE
└─ .gitignore
```

## Key Implementation Changes

- CLI entrypoint:
  - Use `cobra` for commands: `agent`, `agent chat`, `agent run`, `agent config`, `agent doctor`.
  - Default command opens the interactive TUI.
- TUI frontend:
  - Use `bubbletea`, `bubbles`, and `lipgloss`.
  - Main panels: conversation, current plan/tool status, approval prompts, workspace summary.
  - Support keyboard-driven chat input, command history, cancel, and approve/deny tool calls.
- Backend agent core:
  - `internal/agent` owns the reasoning loop, message state, provider calls, tool selection, and session lifecycle.
  - Keep provider calls streaming-friendly from the start.
  - Store sessions locally so demos can show resumable history.
- Provider adapter:
  - Define a generic interface in `internal/providers/provider.go`.
  - Implement adapters for OpenAI, Anthropic, and Ollama behind the same interface.
  - Select provider/model from config or env vars.
- Tool system:
  - Define a `Tool` interface with name, schema, validation, execution, and risk level.
  - Include filesystem read/write, repo search, shell command, git status/diff, and approval flow.
  - Require explicit approval for shell commands and file writes in v1.
- Workspace intelligence:
  - Scan the current repo with ignore rules.
  - Build lightweight context from file tree, selected files, git state, and search results.
  - Avoid indexing huge directories like `.git`, `node_modules`, `vendor`, `dist`, and build outputs.
- Storage/config:
  - Use YAML config for user settings.
  - Use SQLite for local sessions, tool logs, and provider metadata.
  - Keep secrets in environment variables, not config files.

## Public Interfaces

Core provider interface:

```go
type Provider interface {
    Name() string
    Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error)
    Stream(ctx context.Context, req CompletionRequest) (<-chan CompletionEvent, error)
}
```

Core tool interface:

```go
type Tool interface {
    Name() string
    Description() string
    RiskLevel() RiskLevel
    Run(ctx context.Context, input ToolInput) (ToolResult, error)
}
```

Primary CLI commands:

```txt
agent
agent chat
agent run "fix the failing tests"
agent config show
agent config set provider openai
agent doctor
```

Example config:

```yaml
provider: openai
model: gpt-5.4
workspace:
  max_files: 5000
  max_file_bytes: 200000
approval:
  shell: always
  write_files: always
storage:
  path: ~/.go-code-agent/sessions.db
```

## Test Plan

- Unit tests:
  - Provider interface request mapping.
  - Tool validation and approval behavior.
  - Workspace ignore/scanning behavior.
  - Config loading and env override behavior.
- Integration tests:
  - Run a fake provider through the agent loop.
  - Execute safe read/search tools against a fixture repo.
  - Verify denied approvals do not mutate files or run commands.
- CLI/TUI smoke tests:
  - `agent doctor` exits successfully with valid config.
  - `agent chat` can start with a fake provider.
  - CI runs `go test ./...`, `go vet ./...`, and formatting checks.

## GitHub/LinkedIn Presentation

- README should include:
  - Project pitch.
  - Architecture diagram.
  - Demo GIF or screenshots.
  - Installation steps.
  - Example commands.
  - Safety model.
  - Roadmap.
- Docs should highlight:
  - Provider-agnostic architecture.
  - Approval-based tool execution.
  - Go portability and single-binary distribution.
  - Why this is scalable: modular providers, tools, sessions, and future plugin support.

## Assumptions

- First frontend is a terminal TUI, not a web dashboard.
- Backend is Go-native and runs inside the same binary for v1.
- Provider architecture supports OpenAI, Anthropic, and Ollama, but implementation can start with one real provider plus fake provider tests.
- V1 prioritizes a reliable MVP over multi-agent orchestration, remote execution, or a separate hosted backend.
- Files should be created only after leaving Plan Mode; this plan defines the exact scaffold and behavior to implement.

