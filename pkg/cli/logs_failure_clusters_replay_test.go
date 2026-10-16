//go:build !integration

package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Replay an existing logs JSON report without downloading artifacts or calling GitHub.
func TestFailureClustersReplay(t *testing.T) {
	input := os.Getenv("GH_AW_FAILURE_CLUSTER_REPLAY_INPUT")
	if input == "" {
		t.Skip("set GH_AW_FAILURE_CLUSTER_REPLAY_INPUT to replay a saved logs JSON report")
	}
	content, err := os.ReadFile(input)
	require.NoError(t, err)
	var data LogsData
	require.NoError(t, json.Unmarshal(content, &data))
	previous := make(map[string][]int64)
	for _, cluster := range data.FailureClusters {
		previous[cluster.ID] = cluster.RunIDs
	}
	data.FailureClusters = annotateFailureClusters(data.Runs)
	data.FailureInvestigation = buildFailureInvestigation(data.Runs, data.FailureClusters)
	current := make(map[string][]int64)
	for _, cluster := range data.FailureClusters {
		current[cluster.ID] = cluster.RunIDs
		require.LessOrEqual(t, len(cluster.RepresentativeRuns), 3)
	}
	assert.Equal(t, previous, current, "enrichment must preserve cluster IDs and memberships")
	require.NotNil(t, data.FailureInvestigation)
	require.LessOrEqual(t, len(data.FailureInvestigation.SelectedRuns), defaultFailureInvestigationBudget)
	output := os.Getenv("GH_AW_FAILURE_CLUSTER_REPLAY_OUTPUT")
	if output == "" {
		output = t.TempDir()
	}
	updated, err := json.Marshal(data)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(output, "clustering-improved.json"), updated, 0600))
	var compact bytes.Buffer
	require.NoError(t, renderFailureReportToWriter(&compact, data.FailureClusters, data.FailureInvestigation, "", false))
	require.NoError(t, os.WriteFile(filepath.Join(output, "clustering-improved.txt"), compact.Bytes(), 0600))
	t.Logf("Replayed %d runs into %d clusters; initial plan: %+v", len(data.Runs), len(data.FailureClusters), data.FailureInvestigation)
}
