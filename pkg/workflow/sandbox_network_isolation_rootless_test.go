//go:build !integration

package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestNetworkIsolationRootless verifies that when sandbox.agent.network-isolation is
// enabled the compiled lock.yml contains no "sudo" for the AWF binary install or the
// AWF invocation (rootless mode), while legacy workflows still use "sudo -E awf".
func TestNetworkIsolationRootless(t *testing.T) {
	t.Run("network-isolation workflow omits sudo from awf invocation and install", func(t *testing.T) {
		workflowsDir := t.TempDir()

		markdown := `---
on:
  workflow_dispatch:
engine: copilot
strict: false
network:
  allowed:
    - github.com
sandbox:
  agent:
    id: awf
    network-isolation: true
---

# Test Network Isolation Rootless

This workflow verifies that sudo is omitted when network-isolation is enabled.
`

		workflowPath := filepath.Join(workflowsDir, "test-network-isolation.md")
		if err := os.WriteFile(workflowPath, []byte(markdown), 0644); err != nil {
			t.Fatalf("Failed to write workflow file: %v", err)
		}

		compiler := NewCompiler()
		if err := compiler.CompileWorkflow(workflowPath); err != nil {
			t.Fatalf("Compilation failed: %v", err)
		}

		lockPath := filepath.Join(workflowsDir, "test-network-isolation.lock.yml")
		lockContent, err := os.ReadFile(lockPath)
		if err != nil {
			t.Fatalf("Failed to read compiled workflow: %v", err)
		}
		lockStr := string(lockContent)

		// AWF invocation must not use sudo
		if strings.Contains(lockStr, "sudo -E awf") {
			t.Error("Expected no 'sudo -E awf' in lock file when network-isolation is enabled")
		}

		// AWF must still be invoked (just without sudo). Check for the main AWF invocation pattern.
		// The awf command appears in a multi-line run: | block with indentation (e.g., "          awf --config").
		// This pattern uniquely identifies the main AWF execution (not the log-parsing "awf logs summary").
		if !strings.Contains(lockStr, "\n          awf --config ") {
			t.Error("Expected rootless 'awf --config' invocation in lock file main execution step")
		}

		// Install step must pass --rootless flag
		if !strings.Contains(lockStr, "install_awf_binary.sh") {
			t.Error("Expected install_awf_binary.sh in lock file")
		}
		if !strings.Contains(lockStr, "--rootless") {
			t.Error("Expected '--rootless' flag in install step when network-isolation is enabled")
		}

		// The sudo chmod permission-fix step should be absent
		if strings.Contains(lockStr, "sudo chmod -R a+rX") {
			t.Error("Expected no 'sudo chmod -R a+rX' permission-fix step when network-isolation is enabled")
		}
	})

	t.Run("legacy workflow (no network-isolation) still uses sudo -E awf", func(t *testing.T) {
		workflowsDir := t.TempDir()

		markdown := `---
on:
  workflow_dispatch:
engine: copilot
strict: false
network:
  allowed:
    - github.com
sandbox:
  agent:
    id: awf
---

# Test Legacy Sudo

This workflow verifies that sudo is retained when network-isolation is NOT enabled.
`

		workflowPath := filepath.Join(workflowsDir, "test-legacy-sudo.md")
		if err := os.WriteFile(workflowPath, []byte(markdown), 0644); err != nil {
			t.Fatalf("Failed to write workflow file: %v", err)
		}

		compiler := NewCompiler()
		if err := compiler.CompileWorkflow(workflowPath); err != nil {
			t.Fatalf("Compilation failed: %v", err)
		}

		lockPath := filepath.Join(workflowsDir, "test-legacy-sudo.lock.yml")
		lockContent, err := os.ReadFile(lockPath)
		if err != nil {
			t.Fatalf("Failed to read compiled workflow: %v", err)
		}
		lockStr := string(lockContent)

		// Legacy topology must still use sudo -E awf
		if !strings.Contains(lockStr, "sudo -E awf") {
			t.Error("Expected 'sudo -E awf' in legacy lock file (no network-isolation)")
		}

		// Install step must NOT pass --rootless flag
		if strings.Contains(lockStr, "--rootless") {
			t.Error("Expected no '--rootless' flag in legacy install step")
		}

		// sudo chmod permission-fix step should be present
		if !strings.Contains(lockStr, "sudo chmod -R a+rX") {
			t.Error("Expected 'sudo chmod -R a+rX' permission-fix step in legacy (non-isolation) workflow")
		}
	})
}
