//go:build js || wasm

package workflow

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os/exec"
)

// Note: os/exec compiles fine for GOOS=js GOARCH=wasm (it just fails at runtime).
// We must keep the *exec.Cmd return types because non-constrained callers like
// action_resolver.go reference these functions and expect *exec.Cmd. The stubs
// are never called at runtime in the wasm build since compilation skips external
// tool validation (WithSkipValidation(true)).

func setupGHCommand(ctx context.Context, args ...string) *exec.Cmd {
	return ghUnavailableCommand(ctx)
}

func ExecGH(ctx context.Context, args ...string) *exec.Cmd {
	return ExecGHContext(ctx, args...)
}

func ExecGHContext(ctx context.Context, args ...string) *exec.Cmd {
	return ghUnavailableCommand(ctx)
}

func ExecGHWithOutput(args ...string) (stdout, stderr bytes.Buffer, err error) {
	return stdout, stderr, errors.New("gh CLI not available in Wasm")
}

func RunGH(ctx context.Context, spinnerMessage string, args ...string) ([]byte, error) {
	return RunGHContext(ctx, spinnerMessage, args...)
}

func RunGHContext(ctx context.Context, spinnerMessage string, args ...string) ([]byte, error) {
	return nil, errors.New("gh CLI not available in Wasm")
}

func RunGHCombined(ctx context.Context, spinnerMessage string, args ...string) ([]byte, error) {
	return RunGHCombinedContext(ctx, spinnerMessage, args...)
}

// RunGHInputContext is a no-op stub for Wasm builds.
// The input reader is intentionally not used in this build target.
func RunGHInputContext(ctx context.Context, spinnerMessage string, input io.Reader, args ...string) ([]byte, error) {
	return nil, errors.New("gh CLI not available in Wasm")
}

func ghUnavailableCommand(ctx context.Context) *exec.Cmd {
	if ctx == nil {
		ctx = context.Background()
	}
	return exec.CommandContext(ctx, "echo", "gh CLI not available in Wasm")
}

func ForceGHHostEnv(cmd *exec.Cmd, host string) {
	// no-op in Wasm: gh CLI subprocesses are not run
}

// SetDefaultGHHost is a no-op in Wasm builds; GH CLI is unavailable.
func SetDefaultGHHost(_ string) {}

// getDefaultGHHost always returns "" in Wasm builds.
func getDefaultGHHost() string { return "" }
