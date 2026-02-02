// Package optimizer provides prompt optimization strategies for improving agent prompts
// through trajectory analysis and feedback processing.
package optimizer

import (
	"fmt"
	"strings"
	"time"

	"github.com/denkhaus/gollum/pkg/shared"
	"github.com/m-mizutani/gollem"
)

const (
	// defaultUpdateInstructions is the default text used when no specific update instructions are provided.
	defaultUpdateInstructions = "No specific instructions provided"
)

// Trajectory represents a single conversation with optional feedback.
// Uses gollem.Message to avoid type duplication with the core library.
type Trajectory struct {
	Messages []gollem.Message `json:"messages"` // Conversation history

	// Feedback contains optional evaluation data
	// Can be: nil, string, *Feedback, or *EditFeedback
	Feedback interface{} `json:"feedback,omitempty"`
}

// Validate ensures trajectory data is well-formed.
func (t *Trajectory) Validate() error {
	if len(t.Messages) == 0 {
		return fmt.Errorf("trajectory must have at least one message")
	}

	for i, msg := range t.Messages {
		if msg.Role == "" {
			return fmt.Errorf("message %d: role cannot be empty", i)
		}
		if len(msg.Contents) == 0 {
			return fmt.Errorf("message %d: must have at least one content", i)
		}
	}

	return t.ValidateFeedback()
}

// ValidateFeedback ensures feedback is valid type.
func (t *Trajectory) ValidateFeedback() error {
	if t.Feedback == nil {
		return nil // No feedback is valid
	}

	switch fb := t.Feedback.(type) {
	case string:
		if fb == "" {
			return fmt.Errorf("string feedback cannot be empty")
		}
	case *Feedback:
		if fb.Score < 0 || fb.Score > 1 {
			return fmt.Errorf("score must be between 0 and 1")
		}
	case *EditFeedback:
		if fb.Revised == "" {
			return fmt.Errorf("edit feedback must have revised text")
		}
	default:
		return fmt.Errorf("unsupported feedback type: %T", fb)
	}

	return nil
}

// FormatForLLM formats the trajectory for LLM consumption.
func (t *Trajectory) FormatForLLM() string {
	var sb strings.Builder

	for _, msg := range t.Messages {
		// Capitalize first letter of role (user -> User, assistant -> Assistant)
		role := string(msg.Role)
		if len(role) > 0 {
			role = strings.ToUpper(role[:1]) + role[1:]
		}

		for _, content := range msg.Contents {
			switch content.Type {
			case gollem.MessageContentTypeText:
				if text, err := content.GetTextContent(); err == nil {
					sb.WriteString(fmt.Sprintf("%s: %s\n\n", role, text.Text))
				}

			case gollem.MessageContentTypeToolCall:
				if tc, err := content.GetToolCallContent(); err == nil {
					sb.WriteString(fmt.Sprintf("%s: Called tool '%s' with args %v\n\n",
						role, tc.Name, tc.Arguments))
				}

			case gollem.MessageContentTypeToolResponse:
				if tr, err := content.GetToolResponseContent(); err == nil {
					sb.WriteString(fmt.Sprintf("%s: Tool response: %v\n\n",
						role, tr.Response))
				}
			}
		}
	}

	if t.Feedback != nil {
		sb.WriteString("### Feedback\n\n")
		switch fb := t.Feedback.(type) {
		case string:
			sb.WriteString(fb)
		case *Feedback:
			sb.WriteString(fmt.Sprintf("Score: %.2f\n", fb.Score))
			if fb.Comment != "" {
				sb.WriteString(fmt.Sprintf("Comment: %s\n", fb.Comment))
			}
			if len(fb.FailureModes) > 0 {
				sb.WriteString(fmt.Sprintf("Issues: %s\n",
					strings.Join(fb.FailureModes, ", ")))
			}
			if fb.Outcome != "" {
				sb.WriteString(fmt.Sprintf("Outcome: %s\n", fb.Outcome))
			}
		case *EditFeedback:
			sb.WriteString(fmt.Sprintf("Revised: %s\n", fb.Revised))
			if len(fb.Edits) > 0 {
				sb.WriteString("\nEdits:\n")
				for _, edit := range fb.Edits {
					sb.WriteString(fmt.Sprintf("  - '%s' -> '%s'", edit.OldText, edit.NewText))
					if edit.Reason != "" {
						sb.WriteString(fmt.Sprintf(" (Reason: %s)", edit.Reason))
					}
					sb.WriteString("\n")
				}
			}
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

// Feedback provides structured evaluation data.
type Feedback struct {
	Score        float64  `json:"score,omitempty"`         // Numeric rating (0.0 - 1.0)
	Comment      string   `json:"comment,omitempty"`       // Free-form feedback text
	FailureModes []string `json:"failure_modes,omitempty"` // Identified issues
	Outcome      string   `json:"outcome,omitempty"`       // "success", "failure", etc.
}

// EditFeedback provides revised response with specific edit annotations.
type EditFeedback struct {
	Revised string     `json:"revised"` // The corrected response
	Edits   []TextEdit `json:"edits,omitempty"`
}

// TextEdit represents a specific change annotation.
type TextEdit struct {
	OldText string `json:"old_text"`
	NewText string `json:"new_text"`
	Reason  string `json:"reason,omitempty"`
}

// OptimizerInput contains the data needed for prompt optimization.
//
//nolint:revive // stuttering is acceptable for clarity
type OptimizerInput struct {
	Prompt             string        `json:"prompt"`                        // The current prompt to optimize
	Trajectories       []*Trajectory `json:"trajectories"`                  // Conversation data to analyze
	UpdateInstructions string        `json:"update_instructions,omitempty"` // Developer constraints
	Feedback           string        `json:"feedback,omitempty"`            // Optional user feedback
}

// OptimizerResult contains the result of prompt optimization.
//
//nolint:revive // stuttering is acceptable for clarity
type OptimizerResult struct {
	NewPrompt          string `json:"new_prompt"`                   // The optimized prompt
	ChangeDescription  string `json:"change_description,omitempty"` // Description of changes made
	WarrantsAdjustment bool   `json:"warrants_adjustment"`          // Whether changes were needed
}

// GradientReflectionResponse is the structured output from gradient reflection phase.
type GradientReflectionResponse struct {
	WarrantsAdjustment bool   `json:"warrants_adjustment"`       // Whether adjustment is needed
	Hypotheses         string `json:"hypotheses,omitempty"`      // Identified problems
	Recommendations    string `json:"recommendations,omitempty"` // Suggested fixes
	Reasoning          string `json:"reasoning,omitempty"`       // Analysis explanation
}

// MetaPromptResponse is the structured output from meta-prompt strategy.
type MetaPromptResponse struct {
	WarrantsAdjustment bool   `json:"warrants_adjustment"`      // Whether adjustment is needed
	UpdatedPrompt      string `json:"updated_prompt,omitempty"` // The optimized prompt
	Reasoning          string `json:"reasoning,omitempty"`      // Analysis explanation
}

// PromptMemoryResponse is the structured output from prompt memory strategy.
type PromptMemoryResponse struct {
	WarrantsAdjustment bool   `json:"warrants_adjustment"`      // Whether adjustment is needed
	UpdatedPrompt      string `json:"updated_prompt,omitempty"` // The optimized prompt
	Reasoning          string `json:"reasoning,omitempty"`      // Analysis explanation
}

// OptimizerConfig contains configuration for the optimizer.
//
//nolint:revive // stuttering is acceptable for clarity
type OptimizerConfig struct {
	Kind               shared.OptimizerStrategy `json:"kind"`                           // Optimization strategy
	MaxReflectionSteps int                      `json:"max_reflection_steps,omitempty"` // Default: 5
	MinReflectionSteps int                      `json:"min_reflection_steps,omitempty"` // Default: 1
	Provider           shared.LLMProvider       `json:"provider,omitempty"`             // LLM provider name
	GradientPrompt     string                   `json:"gradient_prompt,omitempty"`      // Custom reflection prompt
	MetaPrompt         string                   `json:"metaprompt,omitempty"`           // Custom update prompt
}

// FormatSessions converts multiple trajectories to LLM-readable format.
func FormatSessions(trajectories []*Trajectory) string {
	var sb strings.Builder

	for i, traj := range trajectories {
		sb.WriteString(fmt.Sprintf("## Session %d\n\n", i+1))
		sb.WriteString(traj.FormatForLLM())
	}

	return sb.String()
}

// _time is a placeholder for potential time tracking in future metrics.
type _time struct {
	Timestamp time.Time
}

// _nolint is used to suppress unused warnings for types reserved for future use.
var _ = _time{} //nolint:unused
