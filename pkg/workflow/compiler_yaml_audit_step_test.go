//go:build !integration

package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPreAgentAuditStepGenerated(t *testing.T) {
	tmpDir := t.TempDir()
	testContent := `---
on: push
permissions:
  contents: read
engine: copilot
strict: false
---

Test workflow to verify pre-agent audit step is generated.
`
	testFile := filepath.Join(tmpDir, "test-audit-step.md")
	require.NoError(t, os.WriteFile(testFile, []byte(testContent), 0644), "writing test workflow")

	compiler := NewCompiler()
	require.NoError(t, compiler.CompileWorkflow(testFile), "compiling workflow")

	lockFile := filepath.Join(tmpDir, "test-audit-step.lock.yml")
	content, err := os.ReadFile(lockFile)
	require.NoError(t, err, "reading lock file")
	lockContent := string(content)

	assert.Contains(t, lockContent, "name: Audit pre-agent workspace", "audit step name should be present")
	assert.Contains(t, lockContent, "id: pre-agent-audit", "audit step id should be present")
	assert.Contains(t, lockContent, "continue-on-error: true", "audit step should be resilient")
	assert.Contains(t, lockContent, constants.PreAgentAuditFilePath, "audit file path should appear in step")
}

func TestPreAgentAuditStepOrder(t *testing.T) {
	tmpDir := t.TempDir()
	testContent := `---
on: push
permissions:
  contents: read
engine: copilot
strict: false
---

Test workflow to verify audit step order.
`
	testFile := filepath.Join(tmpDir, "test-audit-order.md")
	require.NoError(t, os.WriteFile(testFile, []byte(testContent), 0644), "writing test workflow")

	compiler := NewCompiler()
	require.NoError(t, compiler.CompileWorkflow(testFile), "compiling workflow")

	lockFile := filepath.Join(tmpDir, "test-audit-order.lock.yml")
	content, err := os.ReadFile(lockFile)
	require.NoError(t, err, "reading lock file")
	lockContent := string(content)

	mountMCPIndex := indexInNonCommentLines(lockContent, "- name: Mount MCP servers as CLIs")
	auditIndex := indexInNonCommentLines(lockContent, "- name: Audit pre-agent workspace")
	agentIndex := indexInNonCommentLines(lockContent, "- name: Execute GitHub Copilot CLI")

	require.NotEqual(t, -1, mountMCPIndex, "Mount MCP servers step should be present")
	require.NotEqual(t, -1, auditIndex, "Audit pre-agent workspace step should be present")
	require.NotEqual(t, -1, agentIndex, "Agent execution step should be present")

	assert.Greater(t, auditIndex, mountMCPIndex, "audit step (%d) should appear after MCP CLI mount (%d)", auditIndex, mountMCPIndex)
	assert.Less(t, auditIndex, agentIndex, "audit step (%d) should appear before agent execution (%d)", auditIndex, agentIndex)
}

func TestPreAgentAuditStepArtifactPath(t *testing.T) {
	tmpDir := t.TempDir()
	testContent := `---
on: push
permissions:
  contents: read
engine: copilot
strict: false
---

Test workflow to verify audit file is in artifact paths.
`
	testFile := filepath.Join(tmpDir, "test-audit-artifact.md")
	require.NoError(t, os.WriteFile(testFile, []byte(testContent), 0644), "writing test workflow")

	compiler := NewCompiler()
	require.NoError(t, compiler.CompileWorkflow(testFile), "compiling workflow")

	lockFile := filepath.Join(tmpDir, "test-audit-artifact.lock.yml")
	content, err := os.ReadFile(lockFile)
	require.NoError(t, err, "reading lock file")
	lockContent := string(content)

	// The audit file should be listed in the artifact upload step paths
	assert.Contains(t, lockContent, constants.PreAgentAuditFilePath,
		"audit file path %q should appear in lock file", constants.PreAgentAuditFilePath)
}

func TestPreAgentAuditStepExcludesCommonCacheDirs(t *testing.T) {
	var sb strings.Builder
	compiler := NewCompiler()
	compiler.generatePreAgentAuditStep(&sb)
	stepYAML := sb.String()

	excludedDirs := []string{
		"node_modules",
		"__pycache__",
		".cache",
		"vendor",
		".npm",
		".yarn",
		".pnpm-store",
		"site-packages",
		".bundle",
	}
	for _, dir := range excludedDirs {
		assert.Contains(t, stepYAML, dir, "step should exclude %q from listings", dir)
	}
}

func TestPreAgentAuditStepCoversAgentDirectories(t *testing.T) {
	var sb strings.Builder
	compiler := NewCompiler()
	compiler.generatePreAgentAuditStep(&sb)
	stepYAML := sb.String()

	expectedPaths := []string{
		".github/agents",
		".github/skills",
		".github/copilot",
		".github",
		".local/share/gh/extensions",
		"RUNNER_TEMP",
	}
	for _, path := range expectedPaths {
		assert.Contains(t, stepYAML, path, "step should cover %q directory", path)
	}
}
