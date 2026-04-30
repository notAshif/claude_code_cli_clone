package tools

import (
	"context"
	"encoding/json"
)

// RiskLevel indicates the potential danger of a tool operation
type RiskLevel int

const (
	RiskLevelNone RiskLevel = iota
	RiskLevelLow
	RiskLevelMedium
	RiskLevelHigh
)

func (r RiskLevel) String() string {
	switch r {
	case RiskLevelNone:
		return "none"
	case RiskLevelLow:
		return "low"
	case RiskLevelMedium:
		return "medium"
	case RiskLevelHigh:
		return "high"
	default:
		return "unknown"
	}
}

// ToolInput represents input parameters for a tool
type ToolInput map[string]any

// ToolResult represents the output of a tool execution
type ToolResult struct {
	Output  string
	Error   error
	Files   []string
	Changes []string
}

// ToolSchema defines the JSON schema for tool inputs
type ToolSchema struct {
	Type       string              `json:"type"`
	Properties map[string]*SchemaProperty `json:"properties,omitempty"`
	Required   []string            `json:"required,omitempty"`
}

type SchemaProperty struct {
	Type        string `json:"type"`
	Description string `json:"description"`
}

// Tool interface defines the contract for all tools
type Tool interface {
	Name() string
	Description() string
	Schema() ToolSchema
	RiskLevel() RiskLevel
	Run(ctx context.Context, input ToolInput) (ToolResult, error)
}

// ToolRegistry manages available tools
type ToolRegistry struct {
	tools map[string]Tool
}

func NewToolRegistry() *ToolRegistry {
	return &ToolRegistry{
		tools: make(map[string]Tool),
	}
}

func (r *ToolRegistry) Register(tool Tool) {
	r.tools[tool.Name()] = tool
}

func (r *ToolRegistry) Get(name string) (Tool, bool) {
	tool, ok := r.tools[name]
	return tool, ok
}

func (r *ToolRegistry) List() []Tool {
	result := make([]Tool, 0, len(r.tools))
	for _, tool := range r.tools {
		result = append(result, tool)
	}
	return result
}

func (r *ToolRegistry) SchemaJSON() ([]byte, error) {
	schemas := make(map[string]ToolSchema)
	for _, tool := range r.tools {
		schemas[tool.Name()] = tool.Schema()
	}
	return json.Marshal(schemas)
}
