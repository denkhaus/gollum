package flow

import (
	"context"
	"fmt"
	"strings"

	"github.com/denkhaus/gollum/pkg/flows"
	"github.com/denkhaus/gollum/pkg/flows/executor"
	"github.com/denkhaus/gollum/pkg/flows/parser"
	"github.com/denkhaus/gollum/pkg/shared"
	"github.com/samber/do/v2"
	"github.com/urfave/cli/v3"
)

// RunCommand returns the flow run command
func RunCommand() *cli.Command {
	return &cli.Command{
		Name:      "run",
		Usage:     "Execute a flow",
		ArgsUsage: "<path>",
		Flags: []cli.Flag{
			shared.VerboseFlag,
			&cli.StringFlag{
				Name:     "input",
				Aliases:  []string{"i"},
				Usage:    "Input values as key=value pairs (comma-separated: name=value,age=25)",
				Required: false,
			},
		},
		Action: RunAction,
	}
}

// RunAction executes the flow run command
func RunAction(ctx context.Context, cmd *cli.Command) error {
	path, err := shared.GetPathArg(cmd)
	if err != nil {
		return err
	}

	injector := shared.MustGetInjectorFromRoot(cmd)
	return executeFlow(ctx, injector, path, cmd.Root().Writer, cmd)
}

// executeFlow loads and executes a flow
func executeFlow(ctx context.Context, injector do.Injector, path string, w writer, cmd *cli.Command) error {
	// Parse flow
	flow, err := parser.Parse(path)
	if err != nil {
		return fmt.Errorf("failed to parse flow: %w", err)
	}

	// Get executor service
	execSvc := do.MustInvoke[executor.FlowExecutorService](injector)

	// Create executor instance
	exec := execSvc.New(flow)

	// Parse input flag if provided
	inputs := make(map[string]string)
	if inputFlag := cmd.String("input"); inputFlag != "" {
		parsed, err := parseInputFlag(inputFlag)
		if err != nil {
			return fmt.Errorf("failed to parse inputs: %w", err)
		}
		inputs = parsed
	}

	// Set input
	if err := exec.SetInput(inputs); err != nil {
		return fmt.Errorf("input validation failed: %w", err)
	}

	// Validate
	if err := exec.Validate(); err != nil {
		return fmt.Errorf("flow validation failed: %w", err)
	}

	// Run
	if _, err := fmt.Fprintf(w, "Executing flow: %s\n", flow.Name); err != nil {
		return fmt.Errorf("failed to write output: %w", err)
	}
	if err := exec.Run(); err != nil {
		return fmt.Errorf("flow execution failed: %w", err)
	}

	// Display outputs
	execCtx := exec.GetContext()
	if execCtx != nil {
		displayOutputs(w, flow, execCtx)
	}

	// Cleanup
	exec.Close()

	if _, err := fmt.Fprintf(w, "Flow completed successfully\n"); err != nil {
		return fmt.Errorf("failed to write output: %w", err)
	}
	return nil
}

// parseInputFlag parses comma-separated key=value pairs
func parseInputFlag(input string) (map[string]string, error) {
	result := make(map[string]string)

	pairs := strings.Split(input, ",")
	for _, pair := range pairs {
		pair = strings.TrimSpace(pair)
		if pair == "" {
			continue
		}

		parts := strings.SplitN(pair, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid input format: %s (expected key=value)", pair)
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		result[key] = value
	}

	return result, nil
}

// displayOutputs shows the output field values
func displayOutputs(w writer, flow *flows.Flow, execCtx executor.ExecutionContext) {
	if flow.Output == nil {
		return
	}

	fmt.Fprintf(w, "\nOutputs:\n")

	// Display string outputs
	for _, field := range flow.Output.Strings {
		if val, err := execCtx.GetOutputField(field.Name); err == nil && val != nil {
			fmt.Fprintf(w, "  %s: %s\n", field.Name, anyToString(val))
		}
	}

	// Display int outputs
	for _, field := range flow.Output.Ints {
		if val, err := execCtx.GetOutputField(field.Name); err == nil && val != nil {
			fmt.Fprintf(w, "  %s: %s\n", field.Name, anyToString(val))
		}
	}

	// Display bool outputs
	for _, field := range flow.Output.Bools {
		if val, err := execCtx.GetOutputField(field.Name); err == nil && val != nil {
			fmt.Fprintf(w, "  %s: %s\n", field.Name, anyToString(val))
		}
	}

	// Display float outputs
	for _, field := range flow.Output.Floats {
		if val, err := execCtx.GetOutputField(field.Name); err == nil && val != nil {
			fmt.Fprintf(w, "  %s: %s\n", field.Name, anyToString(val))
		}
	}
}

// anyToString converts any value to string for display
func anyToString(v any) string {
	if v == nil {
		return ""
	}
	switch val := v.(type) {
	case string:
		return val
	case int:
		return fmt.Sprintf("%d", val)
	case int64:
		return fmt.Sprintf("%d", val)
	case float64:
		return fmt.Sprintf("%f", val)
	case bool:
		return fmt.Sprintf("%t", val)
	default:
		return fmt.Sprintf("%v", val)
	}
}

// writer interface for output
type writer interface {
	Write([]byte) (int, error)
}
