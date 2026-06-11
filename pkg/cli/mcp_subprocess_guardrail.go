package cli

import (
	"context"
	"os/exec"
)

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
	select {
	case g.slots <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (g *mcpSubprocessGuardrail) release() {
	<-g.slots
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

func runMCPSubprocessOutput(ctx context.Context, cmd *exec.Cmd) ([]byte, error) {
	return defaultMCPSubprocessGuardrail.output(ctx, cmd)
}

func runMCPSubprocessCombinedOutput(ctx context.Context, cmd *exec.Cmd) ([]byte, error) {
	return defaultMCPSubprocessGuardrail.combinedOutput(ctx, cmd)
}

func runMCPExecOutput(ctx context.Context, execCmd execCmdFunc, args ...string) ([]byte, error) {
	return runMCPSubprocessOutput(ctx, execCmd(ctx, args...))
}

func runMCPExecCombinedOutput(ctx context.Context, execCmd execCmdFunc, args ...string) ([]byte, error) {
	return runMCPSubprocessCombinedOutput(ctx, execCmd(ctx, args...))
}
