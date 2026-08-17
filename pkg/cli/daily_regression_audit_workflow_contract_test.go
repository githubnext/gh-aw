//go:build !integration

package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/github/gh-aw/pkg/gitutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDailyRegressionAuditAllowsPythonJSONParsing(t *testing.T) {
	repoRoot, err := gitutil.FindGitRoot()
	require.NoError(t, err)

	workflowPath := filepath.Join(repoRoot, ".github", "workflows", "daily-regression-audit-kiro.md")
	content, err := os.ReadFile(workflowPath)
	require.NoError(t, err)
	assert.Contains(t, string(content), "- python3")

	lockPath := filepath.Join(repoRoot, ".github", "workflows", "daily-regression-audit-kiro.lock.yml")
	lockContent, err := os.ReadFile(lockPath)
	require.NoError(t, err)
	assert.Contains(t, string(lockContent), "--allow-tool shell(python3)")
}
