//go:build js || wasm

package workflow

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
)

// Note: os/exec compiles fine for GOOS=js GOARCH=wasm (it just fails at runtime).
// We must keep the *exec.Cmd return types because non-constrained callers like
// action_resolver.go reference these functions and expect *exec.Cmd. The stubs
// are never called at runtime in the wasm build since compilation skips external
// tool validation (WithSkipValidation(true)).

func setupGHCommand(ctx context.Context, args ...string) *exec.Cmd {
	return exec.Command("echo", "gh CLI not available in Wasm")
}

func ExecGH(args ...string) *exec.Cmd {
	return exec.Command("echo", "gh CLI not available in Wasm")
}

func ExecGHContext(ctx context.Context, args ...string) *exec.Cmd {
	return exec.Command("echo", "gh CLI not available in Wasm")
}

func ExecGHWithOutput(args ...string) (stdout, stderr bytes.Buffer, err error) {
	return stdout, stderr, fmt.Errorf("gh CLI not available in Wasm")
}

func RunGH(spinnerMessage string, args ...string) ([]byte, error) {
	return nil, fmt.Errorf("gh CLI not available in Wasm")
}

func RunGHCombined(spinnerMessage string, args ...string) ([]byte, error) {
	return nil, fmt.Errorf("gh CLI not available in Wasm")
}
