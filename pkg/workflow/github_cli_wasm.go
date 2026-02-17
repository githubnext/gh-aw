//go:build js || wasm

package workflow

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
)

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
