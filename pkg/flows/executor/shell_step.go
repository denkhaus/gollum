package executor

import (
	"context"
	"fmt"
	"time"

	"github.com/denkhaus/gollum/pkg/flows"
	"github.com/denkhaus/gollum/pkg/shared"
	"github.com/google/uuid"
	"github.com/m-mizutani/gollem"
)

// dummyShellAgent is a placeholder agent for shell steps.
// Shell steps don't have real agent context, but we need an agent
// to create bash tools via the BashToolProvider.
type dummyShellAgent struct {
	id uuid.UUID
}

func (d *dummyShellAgent) GetID() uuid.UUID {
	return d.id
}

func (d *dummyShellAgent) GetConfig() *shared.AgentConfig {
	return &shared.AgentConfig{ID: d.id, Role: "shell-step", Description: "Shell step execution"}
}

func (d *dummyShellAgent) Session() gollem.Session {
	return nil
}

func (d *dummyShellAgent) Execute(ctx context.Context, input ...gollem.Input) (*gollem.ExecuteResponse, error) {
	return nil, fmt.Errorf("shell step agent cannot execute")
}

func (d *dummyShellAgent) GetMessageHistory(ctx context.Context) (*gollem.History, error) {
	return &gollem.History{}, nil
}

func (d *dummyShellAgent) UpdateHistory(ctx context.Context, modifier func(*gollem.History) (*gollem.History, error)) error {
	return nil
}

func (d *dummyShellAgent) UpdateSystemPrompt(ctx context.Context, newPrompt string) error {
	return nil
}

func (d *dummyShellAgent) ToSessionContext() shared.SessionContext {
	return shared.SessionContext{
		AgentID:   d.id,
		SessionID: uuid.Nil,
		ChannelID: uuid.Nil,
	}
}

// executeShellStep executes a shell step by running a command via the bash tool.
func (p *flowExecutorImpl) executeShellStep(_ context.Context, step *flows.Step, _ string) error {
	// Substitute template variables in command
	cmd := p.substituteTemplate(step.Cmd)

	// Create bash tool with a mock agent (shell steps don't have real agent context)
	dummyAgent := &dummyShellAgent{id: uuid.New()}
	if p.bashToolProvider == nil {
		return fmt.Errorf("bash tool provider not available")
	}
	bashTool := p.bashToolProvider.CreateTool(dummyAgent)

	// Execute command
	ctx := context.Background()
	args := map[string]any{
		"command": cmd,
	}

	// Parse timeout if specified
	if step.Timeout != "" {
		if timeout, err := time.ParseDuration(step.Timeout); err == nil {
			args["timeout"] = timeout.Seconds()
		}
	}

	result, err := bashTool.Run(ctx, args)
	if err != nil {
		return fmt.Errorf("bash tool execution: %w", err)
	}

	// Map outputs
	if step.Result != nil {
		// Handle simple assign
		if step.Result.AssignTo != "" {
			scope, fieldName, err := p.parseAssignTarget(step.Result.AssignTo)
			if err != nil {
				return fmt.Errorf("invalid assignTo: %w", err)
			}
			if val, ok := result["stdout"]; ok {
				if scope == flows.FlowVariableScopeContext {
					_ = p.ctx.SetContextField(fieldName, shared.AnyToString(val))
				} else {
					_ = p.ctx.SetOutputField(fieldName, shared.AnyToString(val))
				}
			}
		}
		// Handle path-based outputs
		for _, path := range step.Result.Paths {
			scope, fieldName, err := p.parseAssignTarget(path.AssignTo)
			if err != nil {
				return fmt.Errorf("invalid path assignTo: %w", err)
			}
			switch path.Path {
			case "stdout":
				if val, ok := result["stdout"]; ok {
					if scope == flows.FlowVariableScopeContext {
						_ = p.ctx.SetContextField(fieldName, shared.AnyToString(val))
					} else {
						_ = p.ctx.SetOutputField(fieldName, shared.AnyToString(val))
					}
				}
			case "stderr":
				if val, ok := result["stderr"]; ok {
					if scope == flows.FlowVariableScopeContext {
						_ = p.ctx.SetContextField(fieldName, shared.AnyToString(val))
					} else {
						_ = p.ctx.SetOutputField(fieldName, shared.AnyToString(val))
					}
				}
			}
		}
	}

	// Check exit code and trigger error transition if configured
	exitCode := 0
	if val, ok := result["exit_code"]; ok {
		if code, ok := val.(int); ok {
			exitCode = code
		} else if code, ok := val.(float64); ok {
			exitCode = int(code)
		}
	}

	if exitCode != 0 && step.OnError != nil {
		// Capture error with exit code for sys.error context
		p.captureError(step, fmt.Sprintf("command failed with exit code %d", exitCode), exitCode)

		// Transition to error state
		return p.transitionTo(step.OnError.State)
	}

	return nil
}

// substituteTemplate wraps the context's SubstituteTemplate method for shell steps.
func (p *flowExecutorImpl) substituteTemplate(cmd string) string {
	return p.ctx.SubstituteTemplate(cmd)
}
