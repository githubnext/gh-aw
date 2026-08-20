//go:build !integration

package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/github/gh-aw/pkg/gitutil"
	"github.com/stretchr/testify/require"
)

func TestMetricsCollectorWorkflowRequiresFullWindowAndTypedSafeOutputMetrics(t *testing.T) {
	t.Parallel()

	repoRoot, err := gitutil.FindGitRoot()
	require.NoError(t, err)

	workflowPath := filepath.Join(repoRoot, ".github", "workflows", "metrics-collector.md")
	content, err := os.ReadFile(workflowPath)
	require.NoError(t, err)

	workflowContent := string(content)
	require.Contains(t, workflowContent, `timeout-minutes: 30`)
	require.Contains(t, workflowContent, `Stop only when there is no `+"`continuation`"+` field **and** the oldest collected run is at or`)
	require.NotContains(t, workflowContent, "10 batch calls (~200 runs)")
	require.Contains(t, workflowContent, `"collection_status": "complete"`)
	require.Contains(t, workflowContent, `"collection_window": {`)
	require.Contains(t, workflowContent, `"safe_outputs_by_type": {`)
	require.Contains(t, workflowContent, `"safe_output_outcomes": {`)
	require.Contains(t, workflowContent, "Set `\"collection_status\": \"complete\"` only when workflow run counts and safe-output breakdowns")
}
