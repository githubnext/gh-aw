//go:build integration

package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/github/gh-aw/pkg/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEditCommandIntegrationMutations(t *testing.T) {
	setup := setupIntegrationTest(t)
	defer setup.cleanup()

	workflowPath := filepath.Join(setup.workflowsDir, "edit.md")
	require.NoError(t, os.MkdirAll(filepath.Join(setup.workflowsDir, "shared"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(setup.workflowsDir, "shared", "base.md"), []byte("# Base\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(setup.workflowsDir, "shared", "extra.md"), []byte("# Extra\n"), 0o644))
	require.NoError(t, os.WriteFile(workflowPath, []byte(editIntegrationWorkflow), 0o644))

	requireEditSucceeds(t, setup, "edit", "edit", "max-turns: 20")
	requireEditSucceeds(t, setup, "edit", "edit.md", "--schedule", "daily on weekdays")
	requireEditSucceeds(t, setup, "edit", workflowPath, "--add-import", "shared/extra.md")
	requireEditSucceeds(t, setup, "edit", "edit", "--remove-import", "shared/base.md")
	requireEditSucceeds(t, setup, "edit", "edit", "--add-skill", "owner/repo/skills/review@0123456789012345678901234567890123456789")
	requireEditSucceeds(t, setup, "edit", "edit", "--remove-skill", "owner/repo/skills/base@0123456789012345678901234567890123456789")
	requireEditSucceeds(t, setup, "edit", "edit", "--set", "model=small", "--unset", "description")
	requireEditSucceeds(t, setup, "edit", "edit", "--add", "labels=one", "--remove", "labels=base")

	content, err := os.ReadFile(workflowPath)
	require.NoError(t, err)
	frontmatter, err := parser.ExtractFrontmatterFromContent(string(content))
	require.NoError(t, err)
	assert.Equal(t, uint64(20), frontmatter.Frontmatter["max-turns"])
	assert.Equal(t, "small", frontmatter.Frontmatter["model"])
	assert.NotContains(t, frontmatter.Frontmatter, "description")
	assert.Equal(t, []any{"shared/extra.md"}, frontmatter.Frontmatter["imports"])
	assert.Equal(t, []any{"owner/repo/skills/review@0123456789012345678901234567890123456789"}, frontmatter.Frontmatter["skills"])
	assert.Equal(t, []any{"one"}, frontmatter.Frontmatter["labels"])
	assert.Equal(t, map[string]any{
		"schedule":          "daily on weekdays",
		"workflow_dispatch": nil,
	}, frontmatter.Frontmatter["on"])

	lockContent, err := os.ReadFile(filepath.Join(setup.workflowsDir, "edit.lock.yml"))
	require.NoError(t, err)
	assert.Contains(t, string(lockContent), "Edit Integration Workflow")
}

func TestEditCommandIntegrationDryRunAndFailureSafety(t *testing.T) {
	setup := setupIntegrationTest(t)
	defer setup.cleanup()

	workflowPath := filepath.Join(setup.workflowsDir, "edit.md")
	require.NoError(t, os.WriteFile(workflowPath, []byte(editIntegrationWorkflow), 0o644))
	original, err := os.ReadFile(workflowPath)
	require.NoError(t, err)

	output := requireEditSucceeds(t, setup, "edit", "edit", "--set", "max-turns=20", "--dry-run")
	assert.Contains(t, output, "max-turns: 20")
	content, err := os.ReadFile(workflowPath)
	require.NoError(t, err)
	assert.Equal(t, original, content)
	_, err = os.Stat(filepath.Join(setup.workflowsDir, "edit.lock.yml"))
	assert.ErrorIs(t, err, os.ErrNotExist)

	requireEditSucceeds(t, setup, "edit", "edit", "--set", "max-turns=10")
	beforeFailure, err := os.ReadFile(workflowPath)
	require.NoError(t, err)
	lockPath := filepath.Join(setup.workflowsDir, "edit.lock.yml")
	beforeFailureLock, err := os.ReadFile(lockPath)
	require.NoError(t, err)

	output = requireEditFails(t, setup, "edit", "edit", "--add-import", "shared/missing.md")
	assert.Contains(t, output, "compile edited workflow")
	content, err = os.ReadFile(workflowPath)
	require.NoError(t, err)
	assert.Equal(t, beforeFailure, content)
	lockContent, err := os.ReadFile(lockPath)
	require.NoError(t, err)
	assert.Equal(t, beforeFailureLock, lockContent)

	shorthandPath := filepath.Join(setup.workflowsDir, "shorthand.md")
	shorthand := "---\n# keep this comment\non: push\nengine: copilot\n---\n# Shorthand Workflow\n"
	require.NoError(t, os.WriteFile(shorthandPath, []byte(shorthand), 0o644))
	output = requireEditSucceeds(t, setup, "edit", "shorthand", "--schedule", "off")
	assert.Contains(t, output, "already matches")
	content, err = os.ReadFile(shorthandPath)
	require.NoError(t, err)
	assert.Equal(t, shorthand, string(content))

	require.NoError(t, os.WriteFile(workflowPath, []byte("---\nsource: owner/repo/.github/workflows/edit.md@v1\non: workflow_dispatch\n---\n# Managed\n"), 0o644))
	requireEditSucceeds(t, setup, "edit", "edit", "--set", "max-turns=20")
	content, err = os.ReadFile(workflowPath)
	require.NoError(t, err)
	frontmatter, err := parser.ExtractFrontmatterFromContent(string(content))
	require.NoError(t, err)
	assert.Equal(t, "owner/repo/.github/workflows/edit.md@v1", frontmatter.Frontmatter["source"])
	assert.Equal(t, uint64(20), frontmatter.Frontmatter["max-turns"])

	baseManagedContent := "---\non: workflow_dispatch\n---\n# Managed\n"
	currentManagedContent := string(content)
	for name, newContent := range map[string]string{
		"no upstream change": baseManagedContent,
		"upstream change":    "---\non: workflow_dispatch\n---\n# Managed\n\nUpstream change.\n",
	} {
		t.Run("source managed edit survives update merge "+name, func(t *testing.T) {
			merged, hasConflicts, mergeErr := MergeWorkflowContent(baseManagedContent, currentManagedContent, newContent, "owner/repo/.github/workflows/edit.md@v1", "v2", workflowPath, false)
			require.NoError(t, mergeErr)
			require.False(t, hasConflicts, "local source-managed edit should merge without conflicts:\n%s", merged)

			mergedFrontmatter, parseErr := parser.ExtractFrontmatterFromContent(merged)
			require.NoError(t, parseErr)
			assert.Equal(t, "owner/repo/.github/workflows/edit.md@v2", mergedFrontmatter.Frontmatter["source"])
			assert.Equal(t, uint64(20), mergedFrontmatter.Frontmatter["max-turns"])
			if name == "upstream change" {
				assert.Contains(t, merged, "Upstream change.")
			}
		})
	}
}

func requireEditSucceeds(t *testing.T, setup *integrationTestSetup, args ...string) string {
	t.Helper()
	command := exec.Command(setup.binaryPath, args...)
	command.Dir = setup.tempDir
	output, err := command.CombinedOutput()
	require.NoError(t, err, "gh aw %v failed:\n%s", args, output)
	return string(output)
}

func requireEditFails(t *testing.T, setup *integrationTestSetup, args ...string) string {
	t.Helper()
	command := exec.Command(setup.binaryPath, args...)
	command.Dir = setup.tempDir
	output, err := command.CombinedOutput()
	require.Error(t, err, "gh aw %v unexpectedly succeeded:\n%s", args, output)
	return string(output)
}

const editIntegrationWorkflow = `---
description: Edit integration workflow
labels: [base]
on:
  workflow_dispatch:
skills:
  - owner/repo/skills/base@0123456789012345678901234567890123456789
engine: claude
---
# Edit Integration Workflow

Test workflow.
`
