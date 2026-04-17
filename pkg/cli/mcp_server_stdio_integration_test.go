//go:build integration

package cli

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

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
	cmd := exec.Command(absBinaryPath, "mcp-server", "--cmd", absBinaryPath)
	cmd.Dir = tmpDir
	cmd.Stdin = strings.NewReader("")
	cmd.Env = append(os.Environ(), "GITHUB_ACTOR=")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	_ = cmd.Run()

	stdoutText := strings.TrimSpace(stdout.String())
	if stdoutText != "" {
		t.Fatalf("Expected stdout to remain clean for JSON-RPC, got: %q", stdoutText)
	}

	stderrText := strings.TrimSpace(stderr.String())
	if stderrText == "" {
		t.Fatal("Expected MCP diagnostics on stderr, got empty output")
	}
}
