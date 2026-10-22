//go:build !integration

package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOutcomeJSONLObjectiveRoundtrip(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	outcomesDir := filepath.Join(dir, "outcomes")
	reports := []OutcomeReport{}
	for i, objective := range []struct {
		status OutcomeStatus
		value  int
		labels []string
	}{
		{OutcomeStatusAccepted, 30, []string{"security", "automation"}},
		{OutcomeStatusRejected, 70, []string{"security"}},
		{OutcomeStatusPending, 0, []string{"documentation"}},
		{OutcomeStatusAccepted, 0, nil},
	} {
		report := episodeObservation(42, "item-"+strconv.Itoa(i+1), objective.status).OutcomeReport
		report.ObjectNumber = i + 1
		report.ObjectiveValue = objective.value
		report.ObjectiveLabels = objective.labels
		reports = append(reports, report)
	}
	writeOutcomeJSONL(outcomesDir, 42, reports)
	observations, err := readEpisodeOutcomeObservations(outcomesDir)
	require.NoError(t, err)
	require.Len(t, observations, len(reports))
	for i, observation := range observations {
		assert.Equal(t, reports[i], observation.OutcomeReport)
		assert.Equal(t, int64(42), observation.RunID)
		assert.Equal(t, len(reports), observation.RunItemCount)
	}

	logsPath := filepath.Join(dir, "logs.json")
	content, err := json.Marshal(episodeLogs(42))
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(logsPath, content, 0o600))
	usageDir := filepath.Join(dir, "run-42", "usage")
	require.NoError(t, os.MkdirAll(usageDir, 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(usageDir, "agent_usage.jsonl"), []byte("{\"aic\":1.25}\n"), 0o600))

	var stdout, stderr bytes.Buffer
	cmd := NewLogsCommand()
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"outcomes-episodes", "--logs", logsPath, "--outcomes-dir", outcomesDir, "--output", dir, "--json"})
	require.NoError(t, cmd.Execute())
	assert.Empty(t, stderr.String())
	var data EpisodeOutcomesData
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &data))
	require.Len(t, data.Episodes, 1)
	episode := data.Episodes[0]
	assert.Equal(t, "selected_runs", episode.MembershipScope)
	assert.Equal(t, "complete", episode.OutcomeCoverage)
	assert.Equal(t, 4, episode.Summary.Total)
	assert.Equal(t, 2, episode.Summary.Accepted)
	assert.Equal(t, 100, episode.Summary.TotalObjectiveValue)
	assert.Equal(t, 30, episode.Summary.AcceptedObjectiveValue)
	assert.InDelta(t, 0.3, episode.Summary.ObjectiveEfficiency, 1e-9)
	assert.Equal(t, ComputeOutcomeSummary(reports, nil).DomainBreakdowns, episode.Summary.DomainBreakdowns)
	assert.Equal(t, []DomainBreakdown{
		{Label: "security", Attempted: 2, Accepted: 1, Rejected: 1, TotalObjectiveValue: 100, AcceptedObjectiveValue: 30, ObjectiveEfficiency: 0.3, AcceptanceRate: 0.5},
		{Label: "automation", Attempted: 1, Accepted: 1, TotalObjectiveValue: 30, AcceptedObjectiveValue: 30, ObjectiveEfficiency: 1, AcceptanceRate: 1},
		{Label: "documentation", Attempted: 1, Pending: 1},
		{Label: "unmapped", Attempted: 1, Accepted: 1, AcceptanceRate: 1},
	}, episode.Summary.DomainBreakdowns)
	require.NotNil(t, episode.CostPerAcceptedOutcome)
	assert.InDelta(t, 0.625, *episode.CostPerAcceptedOutcome, 1e-9)
	require.Len(t, episode.Items, len(reports))
	for i, item := range episode.Items {
		assert.Equal(t, reports[i].ObjectiveValue, item.ObjectiveValue)
		assert.Equal(t, reports[i].ObjectiveLabels, item.ObjectiveLabels)
	}

	schemaJSON, err := GenerateNamedOutputSchema("outcomes-episodes")
	require.NoError(t, err)
	var schema jsonschema.Schema
	require.NoError(t, json.Unmarshal(schemaJSON, &schema))
	resolved, err := schema.Resolve(nil)
	require.NoError(t, err)
	var value any
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &value))
	require.NoError(t, resolved.Validate(value))
}
