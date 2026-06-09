package execcommand

import (
	"context"
	"os/exec"
)

// flagged: function receives ctx context.Context but calls exec.Command()
func DoWork(ctx context.Context) {
	_ = exec.Command("git", "status") // want `use exec.CommandContext with the context.Context parameter instead of exec.Command\(\)`
}

// flagged: method with context param
type Worker struct{}

func (w *Worker) Run(ctx context.Context) {
	_ = exec.Command("ls", "-la") // want `use exec.CommandContext with the context.Context parameter instead of exec.Command\(\)`
}

// not flagged: no context parameter
func DoWorkNoCtx() {
	_ = exec.Command("git", "status")
}

// not flagged: blank identifier context parameter
func DoWorkBlank(_ context.Context) {
	_ = exec.Command("git", "status")
}

// not flagged: already uses CommandContext
func DoWorkWithCtx(ctx context.Context) {
	_ = exec.CommandContext(ctx, "git", "status")
}

// not flagged: context param but exec.CommandContext used
func RunWithCtx(ctx context.Context, name string) {
	_ = exec.CommandContext(ctx, name)
}
