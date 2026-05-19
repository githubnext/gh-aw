//go:build !integration

package workflow

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/github/gh-aw/pkg/testutil"
	"github.com/stretchr/testify/require"
)

func TestDeprecatedTitlePrefixAliasForCloseIssue(t *testing.T) {
	tmpDir := testutil.TempDir(t, "deprecated-title-prefix-*")
	testFile := filepath.Join(tmpDir, "workflow.md")

	content := `---
on: workflow_dispatch
safe-outputs:
  close-issue:
    title-prefix: "[bot] "
    required-labels: [automated]
---

# test
`
	require.NoError(t, os.WriteFile(testFile, []byte(content), 0644))

	compiler := NewCompiler()
	workflowData, err := compiler.ParseWorkflowFile(testFile)
	require.NoError(t, err)
	require.NotNil(t, workflowData.SafeOutputs)
	require.NotNil(t, workflowData.SafeOutputs.CloseIssues)
	require.Equal(t, "[bot] ", workflowData.SafeOutputs.CloseIssues.RequiredTitlePrefix)
}
