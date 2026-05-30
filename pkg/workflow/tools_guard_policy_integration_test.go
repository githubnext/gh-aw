//go:build !integration

package workflow

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGuardPolicyEndToEndCompilation(t *testing.T) {
	workflowContent := `---
on:
  workflow_dispatch:
permissions:
  contents: read
engine: copilot
tools:
  github:
    allowed-repos:
      - github/*
      - github/gh-aw
    min-integrity: approved
---

# Guard Policy Integration
`

	tmpDir := t.TempDir()
	workflowPath := filepath.Join(tmpDir, "guard-policy.md")
	require.NoError(t, os.WriteFile(workflowPath, []byte(workflowContent), 0o644))

	compiler := NewCompiler()
	require.NoError(t, compiler.CompileWorkflow(workflowPath))

	lockPath := filepath.Join(tmpDir, "guard-policy.lock.yml")
	lockBytes, err := os.ReadFile(lockPath)
	require.NoError(t, err)

	lockContent := string(lockBytes)
	assert.Contains(t, lockContent, `"guard-policies"`)
	assert.Contains(t, lockContent, `"allow-only"`)
	assert.Contains(t, lockContent, `"min-integrity": "approved"`)
	assert.Contains(t, lockContent, `"repos": [`)
	assert.Contains(t, lockContent, `"github/*"`)
	assert.Contains(t, lockContent, `"github/gh-aw"`)
}
