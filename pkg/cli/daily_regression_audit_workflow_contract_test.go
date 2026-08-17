//go:build !integration

package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/gitutil"
	"github.com/stretchr/testify/require"
)

func TestDailyRegressionAuditAllowsPythonJSONParsing(t *testing.T) {
	repoRoot, err := gitutil.FindGitRoot()
	require.NoError(t, err)

	workflowPath := filepath.Join(repoRoot, ".github", "workflows", "daily-regression-audit-kiro.md")
	content, err := os.ReadFile(workflowPath)
	require.NoError(t, err)
	workflowContent := string(content)
	hasPython3 := strings.Contains(workflowContent, "- python3")
	hasJQ := strings.Contains(workflowContent, "- jq")
	require.True(t, hasPython3 || hasJQ, "workflow must allow either python3 or jq for JSON parsing")

	lockPath := filepath.Join(repoRoot, ".github", "workflows", "daily-regression-audit-kiro.lock.yml")
	lockContent, err := os.ReadFile(lockPath)
	require.NoError(t, err)
	compiled := string(lockContent)
	if hasPython3 {
		require.Contains(t, compiled, "--allow-tool shell(python3)")
	} else {
		require.NotContains(t, compiled, "--allow-tool shell(python3)")
	}
	if hasJQ {
		require.Contains(t, compiled, "--allow-tool shell(jq)")
	} else {
		require.NotContains(t, compiled, "--allow-tool shell(jq)")
	}
}
