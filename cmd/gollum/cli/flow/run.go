package flow

import (
	"context"
	"fmt"

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
	return executeFlow(ctx, injector, path, cmd.Root().Writer)
}

// executeFlow loads and executes a flow
func executeFlow(ctx context.Context, injector do.Injector, path string, w writer) error {
	// Parse flow
	flow, err := parser.Parse(path)
	if err != nil {
		return fmt.Errorf("failed to parse flow: %w", err)
	}

	// Get executor service
	execSvc := do.MustInvoke[executor.FlowExecutorService](injector)

	// Create executor instance
	exec := execSvc.New(flow)

	// Set empty input (flows should define required inputs with defaults)
	exec.SetInput(make(map[string]any))

	// Validate
	if err := exec.Validate(); err != nil {
		return fmt.Errorf("flow validation failed: %w", err)
	}

	// Run
	fmt.Fprintf(w, "Executing flow: %s\n", flow.Name)
	if err := exec.Run(); err != nil {
		return fmt.Errorf("flow execution failed: %w", err)
	}

	fmt.Fprintf(w, "Flow completed successfully\n")
	return nil
}

// writer interface for output
type writer interface {
	Write([]byte) (int, error)
}
