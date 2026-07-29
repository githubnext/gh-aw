package workflow

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/github/gh-aw/pkg/testutil"
	"github.com/stretchr/testify/require"
)

func TestComputeAllowedDomainsForSanitizationCacheReplacesByPath(t *testing.T) {
	compiler := NewCompiler()
	compiler.markdownPath = "/tmp/workflow-a.md"

	data := &WorkflowData{
		FrontmatterHash: "hash-1",
		AI:              "copilot",
	}

	first, err := compiler.computeAllowedDomainsForSanitization(data)
	require.NoError(t, err)
	require.NotEmpty(t, first)
	require.Len(t, compiler.allowedDomainsCache, 1)
	require.Equal(t, "hash-1", compiler.allowedDomainsCache["/tmp/workflow-a.md"].frontmatterHash)

	data2 := &WorkflowData{
		FrontmatterHash: "hash-2",
		AI:              "copilot",
	}
	second, err := compiler.computeAllowedDomainsForSanitization(data2)
	require.NoError(t, err)
	require.NotEmpty(t, second)
	require.Len(t, compiler.allowedDomainsCache, 1)
	require.Equal(t, "hash-2", compiler.allowedDomainsCache["/tmp/workflow-a.md"].frontmatterHash)
	require.Equal(t, second, compiler.allowedDomainsCache["/tmp/workflow-a.md"].domains)

	compiler.markdownPath = "/tmp/workflow-b.md"
	data3 := &WorkflowData{
		FrontmatterHash: "hash-3",
		AI:              "copilot",
	}
	_, err = compiler.computeAllowedDomainsForSanitization(data3)
	require.NoError(t, err)
	require.Len(t, compiler.allowedDomainsCache, 2)
}

func TestPermissionWarningsCountAcrossCompilations(t *testing.T) {
	tmpDir := testutil.TempDir(t, "permission-warning-hash")
	testFile := filepath.Join(tmpDir, "workflow.md")

	content1 := `---
on: push
permissions:
  contents: read
tools:
  github:
    toolsets: [issues]
---

# Test workflow
`

	content2 := `---
on: push
permissions:
  contents: read
tools:
  github:
    toolsets: [pull_requests]
---

# Test workflow
`

	require.NoError(t, os.WriteFile(testFile, []byte(content1), 0o644))

	compiler := NewCompiler()
	require.NoError(t, compiler.CompileWorkflow(testFile))
	require.Positive(t, compiler.GetWarningCount())

	compiler.ResetWarningCount()
	require.NoError(t, compiler.CompileWorkflow(testFile))
	require.Positive(t, compiler.GetWarningCount())

	require.NoError(t, os.WriteFile(testFile, []byte(content2), 0o644))
	compiler.ResetWarningCount()
	require.NoError(t, compiler.CompileWorkflow(testFile))
	require.Positive(t, compiler.GetWarningCount())
}
