//go:build integration

package cli

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/github/gh-aw/pkg/testutil"
)

func TestMCPServer_StdioDiagnosticsGoToStderr(t *testing.T) {
	binaryPath := "../../gh-aw"
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		t.Skip("Skipping test: gh-aw binary not found. Run 'make build' first.")
	}

	tmpDir := testutil.TempDir(t, "mcp-stdio-*")
	workflowsDir := filepath.Join(tmpDir, ".github", "workflows")
	if err := os.MkdirAll(workflowsDir, 0755); err != nil {
		t.Fatalf("Failed to create workflows directory: %v", err)
	}

	workflowPath := filepath.Join(workflowsDir, "test.md")
	workflowContent := `---
on: push
engine: copilot
---
# Test Workflow
`
	if err := os.WriteFile(workflowPath, []byte(workflowContent), 0644); err != nil {
		t.Fatalf("Failed to write workflow file: %v", err)
	}

	if err := initTestGitRepo(tmpDir); err != nil {
		t.Fatalf("Failed to initialize git repository: %v", err)
	}

	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current working directory: %v", err)
	}

	absBinaryPath := filepath.Join(originalDir, binaryPath)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, absBinaryPath, "mcp-server", "--cmd", absBinaryPath)
	cmd.Dir = tmpDir
	cmd.Stdin = strings.NewReader("")

	env := make([]string, 0, len(os.Environ()))
	for _, entry := range os.Environ() {
		if strings.HasPrefix(entry, "GITHUB_ACTOR=") {
			continue
		}
		env = append(env, entry)
	}
	cmd.Env = env

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()
	if err != nil && !errors.Is(ctx.Err(), context.DeadlineExceeded) {
		var exitErr *exec.ExitError
		if !errors.As(err, &exitErr) {
			t.Fatalf("Failed to run MCP server process: %v", err)
		}
	}

	stdoutText := strings.TrimSpace(stdout.String())
	if stdoutText != "" {
		t.Fatalf("Expected stdout to remain clean for JSON-RPC, got: %q", stdoutText)
	}

}
