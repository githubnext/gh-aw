//go:build integration

package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestCompileCopilotImportsMarketplacesPlugins compiles the canonical Copilot workflow
// that uses imports.marketplaces and imports.plugins and verifies that the compiled
// lock file contains the correct `copilot plugin marketplace add` and
// `copilot plugin install` setup steps before the agent execution step.
func TestCompileCopilotImportsMarketplacesPlugins(t *testing.T) {
	setup := setupIntegrationTest(t)
	defer setup.cleanup()

	srcPath := filepath.Join(projectRoot, "pkg/cli/workflows/test-copilot-imports-marketplaces-plugins.md")
	dstPath := filepath.Join(setup.workflowsDir, "test-copilot-imports-marketplaces-plugins.md")

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

	lockFilePath := filepath.Join(setup.workflowsDir, "test-copilot-imports-marketplaces-plugins.lock.yml")
	lockContent, err := os.ReadFile(lockFilePath)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}
	lockContentStr := string(lockContent)

	// Verify marketplace registration step
	if !strings.Contains(lockContentStr, "copilot plugin marketplace add https://marketplace.example.com") {
		t.Errorf("Lock file should contain marketplace add step\nLock file content:\n%s", lockContentStr)
	}

	// Verify plugin install step
	if !strings.Contains(lockContentStr, "copilot plugin install my-plugin") {
		t.Errorf("Lock file should contain plugin install step\nLock file content:\n%s", lockContentStr)
	}

	// Verify marketplace step appears before the agent execution step (sudo -E awf / run: copilot)
	marketplaceIdx := strings.Index(lockContentStr, "copilot plugin marketplace add")
	pluginInstallIdx := strings.Index(lockContentStr, "copilot plugin install my-plugin")
	agentExecIdx := strings.Index(lockContentStr, "sudo -E awf")
	if marketplaceIdx == -1 || pluginInstallIdx == -1 || agentExecIdx == -1 {
		t.Fatalf("Could not find all expected steps: marketplace=%d, plugin=%d, agent=%d",
			marketplaceIdx, pluginInstallIdx, agentExecIdx)
	}
	if marketplaceIdx >= agentExecIdx || pluginInstallIdx >= agentExecIdx {
		t.Errorf("Marketplace/plugin steps should appear before the agent execution step")
	}

	t.Logf("Copilot marketplace/plugins workflow compiled successfully to %s", lockFilePath)
}

// TestCompileClaudeImportsMarketplacesPlugins compiles the canonical Claude workflow
// that uses imports.marketplaces and imports.plugins and verifies that the compiled
// lock file contains the correct `claude plugin marketplace add` and
// `claude plugin install` setup steps before the agent execution step.
func TestCompileClaudeImportsMarketplacesPlugins(t *testing.T) {
	setup := setupIntegrationTest(t)
	defer setup.cleanup()

	srcPath := filepath.Join(projectRoot, "pkg/cli/workflows/test-claude-imports-marketplaces-plugins.md")
	dstPath := filepath.Join(setup.workflowsDir, "test-claude-imports-marketplaces-plugins.md")

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

	lockFilePath := filepath.Join(setup.workflowsDir, "test-claude-imports-marketplaces-plugins.lock.yml")
	lockContent, err := os.ReadFile(lockFilePath)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}
	lockContentStr := string(lockContent)

	// Verify marketplace registration step
	if !strings.Contains(lockContentStr, "claude plugin marketplace add https://marketplace.example.com") {
		t.Errorf("Lock file should contain marketplace add step\nLock file content:\n%s", lockContentStr)
	}

	// Verify plugin install step
	if !strings.Contains(lockContentStr, "claude plugin install my-plugin") {
		t.Errorf("Lock file should contain plugin install step\nLock file content:\n%s", lockContentStr)
	}

	t.Logf("Claude marketplace/plugins workflow compiled successfully to %s", lockFilePath)
}

// TestCompileCopilotImportsMarketplacesPluginsShared compiles the canonical Copilot
// workflow that imports a shared workflow (via imports.aw) which defines its own
// marketplaces and plugins, and verifies that the values are merged and deduplicated
// in the generated lock file.
func TestCompileCopilotImportsMarketplacesPluginsShared(t *testing.T) {
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
	dstPath := filepath.Join(setup.workflowsDir, "test-copilot-imports-marketplaces-plugins-shared.md")
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

	lockFilePath := filepath.Join(setup.workflowsDir, "test-copilot-imports-marketplaces-plugins-shared.lock.yml")
	lockContent, err := os.ReadFile(lockFilePath)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}
	lockContentStr := string(lockContent)

	// Main workflow's own marketplace and plugin should be present
	if !strings.Contains(lockContentStr, "copilot plugin marketplace add https://main-marketplace.example.com") {
		t.Errorf("Lock file should contain main marketplace add step\nLock file content:\n%s", lockContentStr)
	}
	if !strings.Contains(lockContentStr, "copilot plugin install main-plugin") {
		t.Errorf("Lock file should contain main plugin install step\nLock file content:\n%s", lockContentStr)
	}

	// Shared workflow's marketplace and plugin should also be present (merged)
	if !strings.Contains(lockContentStr, "copilot plugin marketplace add https://shared-marketplace.example.com") {
		t.Errorf("Lock file should contain shared marketplace add step\nLock file content:\n%s", lockContentStr)
	}
	if !strings.Contains(lockContentStr, "copilot plugin install shared-plugin") {
		t.Errorf("Lock file should contain shared plugin install step\nLock file content:\n%s", lockContentStr)
	}

	// Values should appear exactly once (deduplication)
	if count := strings.Count(lockContentStr, "copilot plugin marketplace add https://main-marketplace.example.com"); count != 1 {
		t.Errorf("Main marketplace step should appear exactly once, got %d\nLock file content:\n%s", count, lockContentStr)
	}
	if count := strings.Count(lockContentStr, "copilot plugin install main-plugin"); count != 1 {
		t.Errorf("Main plugin step should appear exactly once, got %d\nLock file content:\n%s", count, lockContentStr)
	}

	t.Logf("Copilot shared marketplace/plugins workflow compiled successfully to %s", lockFilePath)
}

// TestCompileCodexImportsMarketplacesPluginsError verifies that using imports.marketplaces
// or imports.plugins with the Codex engine fails compilation with a clear error.
func TestCompileCodexImportsMarketplacesPluginsError(t *testing.T) {
setup := setupIntegrationTest(t)
defer setup.cleanup()

const workflowContent = `---
on: issues
permissions:
  contents: read
  issues: read
engine: codex
imports:
  marketplaces:
    - https://marketplace.example.com
  plugins:
    - my-plugin
---

# Test Codex Imports Marketplaces and Plugins

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
t.Fatalf("Expected compile to fail for Codex with imports.marketplaces/plugins, but it succeeded\nOutput: %s", outputStr)
}

if !strings.Contains(outputStr, "imports.marketplaces") {
t.Errorf("Error output should mention 'imports.marketplaces'\nOutput: %s", outputStr)
}

t.Logf("Correctly rejected Codex workflow with imports.marketplaces/plugins: %s", outputStr)
}
