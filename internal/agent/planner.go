package agent

import (
	"context"
	"strconv"
	"strings"
)

// Plan represents a multi-step execution plan
type Plan struct {
	ID          string
	Objective   string
	Steps       []PlanStep
	CurrentStep int
	Status      PlanStatus
}

// PlanStep is a single step in a plan
type PlanStep struct {
	ID          string
	Description string
	ToolName    string
	ToolInput   map[string]any
	Status      StepStatus
	Output      string
	Error       string
}

// PlanStatus indicates the overall plan state
type PlanStatus string

const (
	PlanStatusPending   PlanStatus = "pending"
	PlanStatusRunning   PlanStatus = "running"
	PlanStatusCompleted PlanStatus = "completed"
	PlanStatusFailed    PlanStatus = "failed"
	PlanStatusCancelled PlanStatus = "cancelled"
)

// StepStatus indicates a step's state
type StepStatus string

const (
	StepStatusPending   StepStatus = "pending"
	StepStatusRunning   StepStatus = "running"
	StepStatusCompleted StepStatus = "completed"
	StepStatusFailed    StepStatus = "failed"
	StepStatusSkipped   StepStatus = "skipped"
)

// Planner generates execution plans from objectives
type Planner struct {
	provider Provider
}

// Provider interface for planner (subset of full provider)
type Provider interface {
	Complete(ctx context.Context, prompt string) (string, error)
}

// NewPlanner creates a new planner
func NewPlanner(provider Provider) *Planner {
	return &Planner{provider: provider}
}

// CreatePlan generates a plan from an objective
func (p *Planner) CreatePlan(ctx context.Context, objective string) (*Plan, error) {
	// Generate plan steps using the provider
	prompt := p.buildPlanPrompt(objective)

	response, err := p.provider.Complete(ctx, prompt)
	if err != nil {
		return nil, err
	}

	plan := p.parsePlanResponse(objective, response)
	return plan, nil
}

// NextStep returns the next step to execute
func (p *Planner) NextStep(plan *Plan) *PlanStep {
	if plan.CurrentStep >= len(plan.Steps) {
		return nil
	}
	return &plan.Steps[plan.CurrentStep]
}

// MarkStepCompleted marks a step as completed and advances
func (p *Planner) MarkStepCompleted(plan *Plan, stepID string, output string) {
	for i, step := range plan.Steps {
		if step.ID == stepID {
			plan.Steps[i].Status = StepStatusCompleted
			plan.Steps[i].Output = output
			plan.CurrentStep = i + 1
			break
		}
	}

	// Check if plan is complete
	if plan.CurrentStep >= len(plan.Steps) {
		plan.Status = PlanStatusCompleted
	}
}

// MarkStepFailed marks a step as failed
func (p *Planner) MarkStepFailed(plan *Plan, stepID string, err error) {
	for i, step := range plan.Steps {
		if step.ID == stepID {
			plan.Steps[i].Status = StepStatusFailed
			plan.Steps[i].Error = err.Error()
			plan.Status = PlanStatusFailed
			break
		}
	}
}

// GetProgress returns completion percentage
func (p *Planner) GetProgress(plan *Plan) int {
	if len(plan.Steps) == 0 {
		return 0
	}
	completed := 0
	for _, step := range plan.Steps {
		if step.Status == StepStatusCompleted {
			completed++
		}
	}
	return (completed * 100) / len(plan.Steps)
}

// Summary returns a human-readable plan summary
func (p *Planner) Summary(plan *Plan) string {
	var sb strings.Builder

	sb.WriteString("Plan: ")
	sb.WriteString(plan.Objective)
	sb.WriteString("\nStatus: ")
	sb.WriteString(string(plan.Status))
	sb.WriteString("\nProgress: ")
	sb.WriteString(strconv.Itoa(p.GetProgress(plan)))
	sb.WriteString("%\n\nSteps:\n")

	for _, step := range plan.Steps {
		statusIcon := "[ ]"
		switch step.Status {
		case StepStatusCompleted:
			statusIcon = "[x]"
		case StepStatusRunning:
			statusIcon = "[>]"
		case StepStatusFailed:
			statusIcon = "[!]"
		}

		sb.WriteString("  ")
		sb.WriteString(statusIcon)
		sb.WriteString(" ")
		sb.WriteString(step.Description)
		sb.WriteString("\n")

		if step.Error != "" {
			sb.WriteString("      Error: ")
			sb.WriteString(step.Error)
			sb.WriteString("\n")
		}
	}

	return sb.String()
}

func (p *Planner) buildPlanPrompt(objective string) string {
	return "Create a step-by-step plan to accomplish: " + objective
}

func (p *Planner) parsePlanResponse(objective, response string) *Plan {
	plan := &Plan{
		ID:        generateID(),
		Objective: objective,
		Status:    PlanStatusPending,
		Steps:     make([]PlanStep, 0),
	}

	// Simple parsing - split by newlines and create steps
	// In production, this would parse structured output
	lines := strings.Split(response, "\n")
	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		plan.Steps = append(plan.Steps, PlanStep{
			ID:          generateID(),
			Description: line,
			Status:      StepStatusPending,
		})
		_ = i
	}

	return plan
}
