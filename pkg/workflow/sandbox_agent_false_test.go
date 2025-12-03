package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSandboxAgentFalseDisablesFirewall(t *testing.T) {
	t.Run("sandbox.agent: false disables firewall", func(t *testing.T) {
		// Create temp directory for test workflows
		workflowsDir := t.TempDir()

		markdown := `---
engine: copilot
network:
  allowed:
    - defaults
    - github.com
sandbox:
  agent: false
on: workflow_dispatch
---

Test workflow to verify sandbox.agent: false disables firewall.
`

		workflowPath := filepath.Join(workflowsDir, "test-agent-false.md")
		err := os.WriteFile(workflowPath, []byte(markdown), 0644)
		if err != nil {
			t.Fatalf("Failed to write workflow file: %v", err)
		}

		// Compile the workflow
		compiler := NewCompiler(false, "", "test")
		compiler.SetSkipValidation(true)

		if err := compiler.CompileWorkflow(workflowPath); err != nil {
			t.Fatalf("Compilation failed: %v", err)
		}

		// Read the compiled workflow
		lockPath := filepath.Join(workflowsDir, "test-agent-false.lock.yml")
		lockContent, err := os.ReadFile(lockPath)
		if err != nil {
			t.Fatalf("Failed to read compiled workflow: %v", err)
		}

		lockStr := string(lockContent)

		// Verify that AWF installation is NOT present
		if strings.Contains(lockStr, "gh-aw-firewall") {
			t.Error("Expected AWF firewall to be disabled, but found gh-aw-firewall in lock file")
		}

		// Verify that AWF wrapper is NOT used in the run step
		if strings.Contains(lockStr, "awf-wrapper") {
			t.Error("Expected AWF wrapper to be disabled, but found awf-wrapper in lock file")
		}
	})

	t.Run("sandbox.agent: awf enables firewall", func(t *testing.T) {
		// Create temp directory for test workflows
		workflowsDir := t.TempDir()

		markdown := `---
engine: copilot
network:
  allowed:
    - defaults
sandbox:
  agent: awf
on: workflow_dispatch
---

Test workflow to verify sandbox.agent: awf enables firewall.
`

		workflowPath := filepath.Join(workflowsDir, "test-agent-awf.md")
		err := os.WriteFile(workflowPath, []byte(markdown), 0644)
		if err != nil {
			t.Fatalf("Failed to write workflow file: %v", err)
		}

		// Compile the workflow
		compiler := NewCompiler(false, "", "test")
		compiler.SetSkipValidation(true)

		if err := compiler.CompileWorkflow(workflowPath); err != nil {
			t.Fatalf("Compilation failed: %v", err)
		}

		// Read the compiled workflow
		lockPath := filepath.Join(workflowsDir, "test-agent-awf.lock.yml")
		lockContent, err := os.ReadFile(lockPath)
		if err != nil {
			t.Fatalf("Failed to read compiled workflow: %v", err)
		}

		lockStr := string(lockContent)

		// Verify that AWF installation IS present
		if !strings.Contains(lockStr, "gh-aw-firewall") {
			t.Error("Expected AWF firewall to be enabled, but did not find gh-aw-firewall in lock file")
		}
	})

	t.Run("sandbox.agent: false prevents default firewall enablement", func(t *testing.T) {
		// Create temp directory for test workflows
		workflowsDir := t.TempDir()

		markdown := `---
engine: copilot
network:
  allowed:
    - defaults
    - github.com
sandbox:
  agent: false
on: workflow_dispatch
---

Test workflow to verify sandbox.agent: false prevents default firewall enablement.
`

		workflowPath := filepath.Join(workflowsDir, "test-no-default-firewall.md")
		err := os.WriteFile(workflowPath, []byte(markdown), 0644)
		if err != nil {
			t.Fatalf("Failed to write workflow file: %v", err)
		}

		// Compile the workflow
		compiler := NewCompiler(false, "", "test")
		compiler.SetSkipValidation(true)

		if err := compiler.CompileWorkflow(workflowPath); err != nil {
			t.Fatalf("Compilation failed: %v", err)
		}

		// Read the compiled workflow
		lockPath := filepath.Join(workflowsDir, "test-no-default-firewall.lock.yml")
		lockContent, err := os.ReadFile(lockPath)
		if err != nil {
			t.Fatalf("Failed to read compiled workflow: %v", err)
		}

		lockStr := string(lockContent)

		// With network restrictions but sandbox.agent: false, firewall should NOT be enabled by default
		if strings.Contains(lockStr, "gh-aw-firewall") {
			t.Error("Expected firewall to be disabled with sandbox.agent: false, but found gh-aw-firewall in lock file")
		}
	})
}

func TestNetworkFirewallDeprecationWarning(t *testing.T) {
	t.Run("network.firewall compiles successfully (deprecated)", func(t *testing.T) {
		// Create temp directory for test workflows
		workflowsDir := t.TempDir()

		markdown := `---
engine: copilot
network:
  allowed:
    - defaults
  firewall: false
strict: false
on: workflow_dispatch
---

Test workflow to verify network.firewall still works (deprecated).
`

		workflowPath := filepath.Join(workflowsDir, "test-firewall-deprecated.md")
		err := os.WriteFile(workflowPath, []byte(markdown), 0644)
		if err != nil {
			t.Fatalf("Failed to write workflow file: %v", err)
		}

		// Compile the workflow
		compiler := NewCompiler(false, "", "test")
		compiler.SetSkipValidation(true)

		// The compilation should succeed (deprecated fields should still work)
		if err := compiler.CompileWorkflow(workflowPath); err != nil {
			t.Fatalf("Compilation failed: %v", err)
		}
	})
}

func TestSandboxAgentFalseExtraction(t *testing.T) {
	t.Run("extractAgentSandboxConfig handles false", func(t *testing.T) {
		compiler := NewCompiler(false, "", "test")
		
		// Test with false value
		agentConfig := compiler.extractAgentSandboxConfig(false)
		if agentConfig == nil {
			t.Fatal("Expected agentConfig to be non-nil for false value")
		}
		if !agentConfig.Disabled {
			t.Error("Expected agentConfig.Disabled to be true")
		}
	})

	t.Run("extractAgentSandboxConfig handles true (invalid)", func(t *testing.T) {
		compiler := NewCompiler(false, "", "test")
		
		// Test with true value (should be invalid)
		agentConfig := compiler.extractAgentSandboxConfig(true)
		if agentConfig != nil {
			t.Error("Expected agentConfig to be nil for true value (invalid)")
		}
	})

	t.Run("extractAgentSandboxConfig handles string", func(t *testing.T) {
		compiler := NewCompiler(false, "", "test")
		
		// Test with "awf" string
		agentConfig := compiler.extractAgentSandboxConfig("awf")
		if agentConfig == nil {
			t.Fatal("Expected agentConfig to be non-nil for 'awf' value")
		}
		if agentConfig.Disabled {
			t.Error("Expected agentConfig.Disabled to be false for 'awf' value")
		}
		if agentConfig.Type != SandboxTypeAWF {
			t.Errorf("Expected agentConfig.Type to be 'awf', got %s", agentConfig.Type)
		}
	})
}
