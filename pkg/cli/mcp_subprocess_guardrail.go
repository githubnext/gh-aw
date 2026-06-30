package cli

import (
	"context"
	"os/exec"

	"github.com/github/gh-aw/pkg/logger"
)

var mcpSubprocessGuardrailLog = logger.New("cli:mcp_subprocess_guardrail")

const maxActiveMCPChildProcesses = 4

type mcpSubprocessGuardrail struct {
	slots chan struct{}
}

var defaultMCPSubprocessGuardrail = newMCPSubprocessGuardrail(maxActiveMCPChildProcesses)

func newMCPSubprocessGuardrail(limit int) *mcpSubprocessGuardrail {
	return &mcpSubprocessGuardrail{
		slots: make(chan struct{}, limit),
	}
}

func (g *mcpSubprocessGuardrail) acquire(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	select {
	case g.slots <- struct{}{}:
		if err := ctx.Err(); err != nil {
			g.release()
			return err
		}
		mcpSubprocessGuardrailLog.Printf("Acquired MCP subprocess slot (%d/%d in use)", len(g.slots), cap(g.slots))
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (g *mcpSubprocessGuardrail) release() {
	<-g.slots
	mcpSubprocessGuardrailLog.Printf("Released MCP subprocess slot (%d/%d in use)", len(g.slots), cap(g.slots))
}

func (g *mcpSubprocessGuardrail) output(ctx context.Context, cmd *exec.Cmd) ([]byte, error) {
	if err := g.acquire(ctx); err != nil {
		return nil, err
	}
	defer g.release()

	return cmd.Output()
}

func (g *mcpSubprocessGuardrail) combinedOutput(ctx context.Context, cmd *exec.Cmd) ([]byte, error) {
	if err := g.acquire(ctx); err != nil {
		return nil, err
	}
	defer g.release()

	return cmd.CombinedOutput()
}

// runMCPSubprocessOutput executes cmd under the shared MCP subprocess guardrail.
// ctx governs slot acquisition and any subprocess cancellation only when cmd was
// created with the same context (for example via exec.CommandContext or ExecGHContext).
func runMCPSubprocessOutput(ctx context.Context, cmd *exec.Cmd) ([]byte, error) {
	return defaultMCPSubprocessGuardrail.output(ctx, cmd)
}

// runMCPSubprocessCombinedOutput executes cmd under the shared MCP subprocess
// guardrail. ctx governs slot acquisition and any subprocess cancellation only
// when cmd was created with the same context.
func runMCPSubprocessCombinedOutput(ctx context.Context, cmd *exec.Cmd) ([]byte, error) {
	return defaultMCPSubprocessGuardrail.combinedOutput(ctx, cmd)
}

func runMCPExecOutput(ctx context.Context, execCmd execCmdFunc, args ...string) ([]byte, error) {
	return runMCPSubprocessOutput(ctx, execCmd(ctx, args...))
}

func runMCPExecCombinedOutput(ctx context.Context, execCmd execCmdFunc, args ...string) ([]byte, error) {
	return runMCPSubprocessCombinedOutput(ctx, execCmd(ctx, args...))
}
