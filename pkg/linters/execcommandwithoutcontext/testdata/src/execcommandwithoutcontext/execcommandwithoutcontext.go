package execcommandwithoutcontext

import (
	"context"
	"os/exec"
)

// Bad: exec.Command inside a function that already receives a context.
func BadRunCommand(ctx context.Context, name string) error {
	cmd := exec.Command(name) // want `use exec\.CommandContext\(ctx, \.\.\.\) instead of exec\.Command to propagate context cancellation`
	return cmd.Run()
}

// Bad: exec.Command with extra args inside a context-receiving function.
func BadRunCommandArgs(ctx context.Context, name string, args ...string) {
	cmd := exec.Command(name, args...) // want `use exec\.CommandContext\(ctx, \.\.\.\) instead of exec\.Command to propagate context cancellation`
	_ = cmd
}

// Good: exec.CommandContext is used correctly.
func GoodRunCommandContext(ctx context.Context, name string) error {
	cmd := exec.CommandContext(ctx, name)
	return cmd.Run()
}

// Good: no context parameter, exec.Command is fine.
func GoodNoContext(name string) error {
	cmd := exec.Command(name)
	return cmd.Run()
}

// Good: context parameter is blank, exec.Command is acceptable.
func GoodBlankContext(_ context.Context, name string) error {
	cmd := exec.Command(name)
	return cmd.Run()
}
