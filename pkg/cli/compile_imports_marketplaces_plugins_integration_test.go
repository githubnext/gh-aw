//go:build integration

package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestCompileCopilotImportsPlugins compiles the canonical Copilot workflow
// that uses imports.plugins and verifies that the compiled lock file contains
// the correct `copilot plugin install` setup steps before the agent execution step.
func TestCompileCopilotImportsPlugins(t *testing.T) {
	setup := setupIntegrationTest(t)
	defer setup.cleanup()

	srcPath := filepath.Join(projectRoot, "pkg/cli/workflows/test-copilot-imports-marketplaces-plugins.md")
	dstPath := filepath.Join(setup.workflowsDir, "test-copilot-imports-plugins.md")

	srcContent, err := os.ReadFile(srcPath)
	if err != nil {
		t.Fatalf("Failed to read source workflow file %s: %v", srcPath, err)
	}
	if err := os.WriteFile(dstPath, srcContent, 0644); err != nil {
		t.Fatalf("Failed to write workflow to test dir: %v", err)
	}

	cmd := exec.Command(setup.binaryPath, "compile", dstPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("CLI compile command failed: %v\nOutput: %s", err, string(output))
	}

	lockFilePath := filepath.Join(setup.workflowsDir, "test-copilot-imports-plugins.lock.yml")
	lockContent, err := os.ReadFile(lockFilePath)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}
	lockContentStr := string(lockContent)

	// Verify plugin install step
	if !strings.Contains(lockContentStr, "copilot plugin install my-plugin") {
		t.Errorf("Lock file should contain plugin install step\nLock file content:\n%s", lockContentStr)
	}

	// Verify plugin step appears before the agent execution step
	pluginInstallIdx := strings.Index(lockContentStr, "copilot plugin install my-plugin")
	agentExecIdx := strings.Index(lockContentStr, "sudo -E awf")
	if pluginInstallIdx == -1 || agentExecIdx == -1 {
		t.Fatalf("Could not find all expected steps: plugin=%d, agent=%d",
			pluginInstallIdx, agentExecIdx)
	}
	if pluginInstallIdx >= agentExecIdx {
		t.Errorf("Plugin install step should appear before the agent execution step")
	}

	t.Logf("Copilot plugins workflow compiled successfully to %s", lockFilePath)
}

// TestCompileClaudeImportsPlugins compiles the canonical Claude workflow
// that uses imports.plugins and verifies that the compiled lock file contains
// the correct `claude plugin install` setup steps before the agent execution step.
func TestCompileClaudeImportsPlugins(t *testing.T) {
	setup := setupIntegrationTest(t)
	defer setup.cleanup()

	srcPath := filepath.Join(projectRoot, "pkg/cli/workflows/test-claude-imports-marketplaces-plugins.md")
	dstPath := filepath.Join(setup.workflowsDir, "test-claude-imports-plugins.md")

	srcContent, err := os.ReadFile(srcPath)
	if err != nil {
		t.Fatalf("Failed to read source workflow file %s: %v", srcPath, err)
	}
	if err := os.WriteFile(dstPath, srcContent, 0644); err != nil {
		t.Fatalf("Failed to write workflow to test dir: %v", err)
	}

	cmd := exec.Command(setup.binaryPath, "compile", dstPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("CLI compile command failed: %v\nOutput: %s", err, string(output))
	}

	lockFilePath := filepath.Join(setup.workflowsDir, "test-claude-imports-plugins.lock.yml")
	lockContent, err := os.ReadFile(lockFilePath)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}
	lockContentStr := string(lockContent)

	// Verify plugin install step
	if !strings.Contains(lockContentStr, "claude plugin install my-plugin") {
		t.Errorf("Lock file should contain plugin install step\nLock file content:\n%s", lockContentStr)
	}

	t.Logf("Claude plugins workflow compiled successfully to %s", lockFilePath)
}

// TestCompileCopilotImportsPluginsShared compiles the canonical Copilot
// workflow that imports a shared workflow (via imports.aw) which defines its own
// plugins, and verifies that the values are merged and deduplicated in the generated
// lock file.
func TestCompileCopilotImportsPluginsShared(t *testing.T) {
	setup := setupIntegrationTest(t)
	defer setup.cleanup()

	// Copy both the shared fixture and the main workflow into the test dir
	sharedDir := filepath.Join(setup.workflowsDir, "shared")
	if err := os.MkdirAll(sharedDir, 0755); err != nil {
		t.Fatalf("Failed to create shared dir: %v", err)
	}

	sharedSrc := filepath.Join(projectRoot, "pkg/cli/workflows/shared/marketplace-plugins.md")
	sharedDst := filepath.Join(sharedDir, "marketplace-plugins.md")
	sharedContent, err := os.ReadFile(sharedSrc)
	if err != nil {
		t.Fatalf("Failed to read shared workflow file %s: %v", sharedSrc, err)
	}
	if err := os.WriteFile(sharedDst, sharedContent, 0644); err != nil {
		t.Fatalf("Failed to write shared workflow: %v", err)
	}

	srcPath := filepath.Join(projectRoot, "pkg/cli/workflows/test-copilot-imports-marketplaces-plugins-shared.md")
	dstPath := filepath.Join(setup.workflowsDir, "test-copilot-imports-plugins-shared.md")
	srcContent, err := os.ReadFile(srcPath)
	if err != nil {
		t.Fatalf("Failed to read source workflow file %s: %v", srcPath, err)
	}
	if err := os.WriteFile(dstPath, srcContent, 0644); err != nil {
		t.Fatalf("Failed to write workflow to test dir: %v", err)
	}

	cmd := exec.Command(setup.binaryPath, "compile", dstPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("CLI compile command failed: %v\nOutput: %s", err, string(output))
	}

	lockFilePath := filepath.Join(setup.workflowsDir, "test-copilot-imports-plugins-shared.lock.yml")
	lockContent, err := os.ReadFile(lockFilePath)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}
	lockContentStr := string(lockContent)

	// Main workflow's own plugin should be present
	if !strings.Contains(lockContentStr, "copilot plugin install main-plugin") {
		t.Errorf("Lock file should contain main plugin install step\nLock file content:\n%s", lockContentStr)
	}

	// Shared workflow's plugin should also be present (merged)
	if !strings.Contains(lockContentStr, "copilot plugin install shared-plugin") {
		t.Errorf("Lock file should contain shared plugin install step\nLock file content:\n%s", lockContentStr)
	}

	// Values should appear exactly once (deduplication)
	if count := strings.Count(lockContentStr, "copilot plugin install main-plugin"); count != 1 {
		t.Errorf("Main plugin step should appear exactly once, got %d\nLock file content:\n%s", count, lockContentStr)
	}

	t.Logf("Copilot shared plugins workflow compiled successfully to %s", lockFilePath)
}

// TestCompileCodexImportsPluginsError verifies that using imports.plugins with the
// Codex engine fails compilation with a clear error.
func TestCompileCodexImportsPluginsError(t *testing.T) {
	setup := setupIntegrationTest(t)
	defer setup.cleanup()

	const workflowContent = `---
on: issues
permissions:
  contents: read
  issues: read
engine: codex
imports:
  plugins:
    - my-plugin
---

# Test Codex Imports Plugins

Process the issue.
`
	dstPath := filepath.Join(setup.workflowsDir, "test-codex-unsupported-imports.md")
	if err := os.WriteFile(dstPath, []byte(workflowContent), 0644); err != nil {
		t.Fatalf("Failed to write workflow: %v", err)
	}

	cmd := exec.Command(setup.binaryPath, "compile", dstPath)
	output, err := cmd.CombinedOutput()
	outputStr := string(output)

	if err == nil {
		t.Fatalf("Expected compile to fail for Codex with imports.plugins, but it succeeded\nOutput: %s", outputStr)
	}

	if !strings.Contains(outputStr, "imports.plugins") {
		t.Errorf("Error output should mention 'imports.plugins'\nOutput: %s", outputStr)
	}

	t.Logf("Correctly rejected Codex workflow with imports.plugins: %s", outputStr)
}
