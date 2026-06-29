//go:build !integration

package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestNetworkIsolationRootless verifies that when sandbox.agent.sudo is false
// (network isolation mode) the compiled lock.yml contains no "sudo" for the AWF binary
// install or the AWF invocation (rootless mode), while legacy workflows still use
// "sudo -E awf".
func TestNetworkIsolationRootless(t *testing.T) {
	t.Run("sudo: false workflow omits sudo from awf invocation and install", func(t *testing.T) {
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
    sudo: false
---

# Test Network Isolation Rootless

This workflow verifies that sudo is omitted when sudo is false (network isolation mode).
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
			t.Error("Expected no 'sudo -E awf' in lock file when sudo is false (network isolation mode)")
		}

		// AWF must still be invoked (just without sudo).
		if !strings.Contains(lockStr, "awf --config ") {
			t.Error("Expected rootless 'awf --config' invocation in lock file main execution step")
		}

		// Install step must pass --rootless flag
		if !strings.Contains(lockStr, "install_awf_binary.sh") {
			t.Error("Expected install_awf_binary.sh in lock file")
		}
		if !strings.Contains(lockStr, "--rootless") {
			t.Error("Expected '--rootless' flag in install step when sudo is false (network isolation mode)")
		}

		// The sudo chmod permission-fix step should be absent
		if strings.Contains(lockStr, "sudo chmod -R a+rX") {
			t.Error("Expected no 'sudo chmod -R a+rX' permission-fix step when sudo is false (network isolation mode)")
		}
	})

	t.Run("workflow with sudo omitted defaults to network isolation (no sudo -E awf)", func(t *testing.T) {
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

# Test Default Network Isolation

This workflow verifies that sudo is omitted by default when sudo is not set (network isolation is the new default).
`

		workflowPath := filepath.Join(workflowsDir, "test-default-network-isolation.md")
		if err := os.WriteFile(workflowPath, []byte(markdown), 0644); err != nil {
			t.Fatalf("Failed to write workflow file: %v", err)
		}

		compiler := NewCompiler()
		if err := compiler.CompileWorkflow(workflowPath); err != nil {
			t.Fatalf("Compilation failed: %v", err)
		}

		lockPath := filepath.Join(workflowsDir, "test-default-network-isolation.lock.yml")
		lockContent, err := os.ReadFile(lockPath)
		if err != nil {
			t.Fatalf("Failed to read compiled workflow: %v", err)
		}
		lockStr := string(lockContent)

		// Default (sudo not set) must use network isolation mode (no sudo -E awf)
		if strings.Contains(lockStr, "sudo -E awf") {
			t.Error("Expected no 'sudo -E awf' in lock file when sudo is not set (network isolation is the default)")
		}

		// AWF must still be invoked (without sudo).
		if !strings.Contains(lockStr, "awf --config ") {
			t.Error("Expected rootless 'awf --config' invocation in lock file main execution step")
		}

		// Install step must pass --rootless flag
		if !strings.Contains(lockStr, "install_awf_binary.sh") {
			t.Error("Expected install_awf_binary.sh in lock file")
		}
		if !strings.Contains(lockStr, "--rootless") {
			t.Error("Expected '--rootless' flag in install step when sudo is not set (network isolation is the default)")
		}

		// sudo chmod permission-fix step should be absent
		if strings.Contains(lockStr, "sudo chmod -R a+rX") {
			t.Error("Expected no 'sudo chmod -R a+rX' permission-fix step when sudo is not set (network isolation is the default)")
		}
	})
}
