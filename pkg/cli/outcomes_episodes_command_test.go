//go:build !integration

package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/github/gh-aw/pkg/testutil"
	"github.com/google/jsonschema-go/jsonschema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunOutcomesEpisodesOffline(t *testing.T) {
	dir := t.TempDir()
	logsPath := filepath.Join(dir, "logs.json")
	logs := episodeLogs(1, 2)
	// An agent-only logs total must not be used as full observed run cost.
	logs.Episodes[0].TotalAIC = 999
	logs.Runs[0].AIC = 999
	content, err := json.Marshal(logs)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(logsPath, content, 0o600))
	outcomesDir := filepath.Join(dir, "outcomes")
	first := episodeObservation(1, "item-1", OutcomeStatusAccepted)
	second := episodeObservation(2, "item-1", OutcomeStatusPending)
	writeOutcomeJSONL(outcomesDir, 1, []OutcomeReport{first.OutcomeReport})
	writeOutcomeJSONL(outcomesDir, 2, []OutcomeReport{second.OutcomeReport})
	// Multiple observation files are allowed; no double counting.
	content, err = os.ReadFile(filepath.Join(outcomesDir, "outcomes-1.jsonl"))
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(outcomesDir, "older.jsonl"), content, 0o600))
	for _, id := range []string{"1", "2"} {
		usageDir := filepath.Join(dir, "run-"+id, "usage")
		require.NoError(t, os.MkdirAll(usageDir, 0o700))
		require.NoError(t, os.WriteFile(filepath.Join(usageDir, "agent_usage.jsonl"), []byte("{\"aic\":1.25}\n"), 0o600))
		require.NoError(t, os.WriteFile(filepath.Join(usageDir, "detection_usage.jsonl"), []byte("{\"aic\":2.5}\n"), 0o600))
	}

	run := func() EpisodeOutcomesData {
		t.Helper()
		output, err := os.CreateTemp(dir, "stdout-")
		require.NoError(t, err)
		originalStdout := os.Stdout
		os.Stdout = output
		defer func() { os.Stdout = originalStdout }()
		var runErr error
		testutil.CaptureStderr(t, func() {
			runErr = runOutcomesEpisodes(logsPath, outcomesDir, dir, true)
		})
		require.NoError(t, runErr)
		require.NoError(t, output.Close())
		content, err := os.ReadFile(output.Name())
		require.NoError(t, err)
		var data EpisodeOutcomesData
		require.NoError(t, json.Unmarshal(content, &data))

		schemaJSON, err := GenerateNamedOutputSchema("outcomes-episodes")
		require.NoError(t, err)
		var schema jsonschema.Schema
		require.NoError(t, json.Unmarshal(schemaJSON, &schema))
		resolved, err := schema.Resolve(nil)
		require.NoError(t, err)
		var value any
		require.NoError(t, json.Unmarshal(content, &value))
		require.NoError(t, resolved.Validate(value))
		return data
	}
	data := run()
	require.Len(t, data.Episodes, 1)
	row := data.Episodes[0]
	assert.Equal(t, 1, row.Summary.Accepted)
	assert.Equal(t, 1, row.Summary.Pending)
	assert.InDelta(t, 7.5, *row.ObservedAIC, 1e-9)
	assert.InDelta(t, 7.5, *row.CostPerAcceptedOutcome, 1e-9)

	var rendered bytes.Buffer
	require.NoError(t, renderEpisodeOutcomes(&rendered, data))
	assert.Contains(t, rendered.String(), "AIC per accepted outcome (selected runs): 7.500")
	assert.Contains(t, rendered.String(), "full-episode membership is not verified")

	require.NoError(t, os.WriteFile(filepath.Join(dir, "run-2", "usage", "agent_usage.jsonl"), []byte("invalid\n"), 0o600))
	data = run()
	assert.Equal(t, "partial", data.Episodes[0].CostCoverage)
	assert.InDelta(t, 6.25, *data.Episodes[0].ObservedAIC, 1e-9, "only readable usage is observed")
	assert.Contains(t, data.Episodes[0].Qualifications, "Run 2: agent_usage.jsonl line 1: malformed usage record; observed cost may be incomplete")
	assert.Nil(t, data.Episodes[0].CostPerAcceptedOutcome)
	rendered.Reset()
	require.NoError(t, renderEpisodeOutcomes(&rendered, data))
	assert.Contains(t, rendered.String(), "AIC per accepted outcome: unavailable")
}

func TestEpisodeOutcomeCostAvailability(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name      string
		content   string
		available bool
	}{
		{"unpriced tokens", `{"model":"unpriced","input_tokens":10}`, false},
		{"explicit zero", `{"ai_credits":0}`, true},
		{"positive cost", `{"ai_credits":2.5}`, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			require.NoError(t, os.WriteFile(filepath.Join(dir, "agent_usage.json"), []byte(tc.content), 0o600))
			usage, err := analyzeTokenUsageAICOnly(dir, false)
			require.NoError(t, err)
			require.NotNil(t, usage)
			assert.Equal(t, tc.available, usage.AICFound)
		})
	}
}

func TestEpisodeUsageWarnings(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name     string
		file     string
		content  string
		warnings bool
	}{
		{"legacy zero", "agent_usage.jsonl", `{"aic":0}`, false},
		{"legacy positive", "agent_usage.jsonl", `{"aic":2.5}`, false},
		{"unpriced legacy", "detection_usage.jsonl", `{"model":"unpriced","input_tokens":10}`, true},
		{"unpriced AWF", "token_usage.jsonl", `{"event":"token_usage","provider":"unknown","model":"unpriced","input_tokens":10}`, true},
		{"malformed", "agent_usage.jsonl", `{`, true},
		{"missing stream", "agent_usage.jsonl", "", false},
		{"non-usage metadata", "aw-info.jsonl", `{"workflow":"test"}`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			usageDir := filepath.Join(dir, "usage")
			require.NoError(t, os.MkdirAll(usageDir, 0o700))
			require.NoError(t, os.WriteFile(filepath.Join(usageDir, tc.file), []byte(tc.content+"\n"), 0o600))
			warnings := episodeUsageWarnings(dir)
			assert.Equal(t, tc.warnings, len(warnings) > 0)
			if tc.name == "legacy zero" {
				usage, err := analyzeTokenUsageAICOnly(dir, false)
				require.NoError(t, err)
				require.NotNil(t, usage)
				assert.True(t, usage.AICFound)
				assert.Zero(t, usage.TotalAIC)
			}
		})
	}
}

func TestOutcomesEpisodesInputErrors(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	logs := filepath.Join(dir, "logs.json")
	require.NoError(t, os.WriteFile(logs, []byte(`{"episodes":[]}`), 0o600))
	require.ErrorContains(t, runOutcomesEpisodes(logs, dir, dir, true), "regenerate it")
	require.NoError(t, os.WriteFile(logs, []byte(`{`), 0o600))
	require.ErrorContains(t, runOutcomesEpisodes(logs, dir, dir, true), "failed to parse episode logs")
	_, err := readEpisodeOutcomeObservations(dir)
	require.ErrorContains(t, err, "no outcome observations")
	require.NoError(t, os.WriteFile(filepath.Join(dir, "outcomes.jsonl"), []byte("{\n"), 0o600))
	_, err = readEpisodeOutcomeObservations(dir)
	require.ErrorContains(t, err, "failed to parse outcome observation 1")
}

func TestEpisodeOutcomesOptionalCoverageInput(t *testing.T) {
	t.Parallel()
	var logs episodeOutcomesLogs
	require.NoError(t, json.Unmarshal([]byte(`{
		"runs":[{"run_id":1}],
		"episodes":[{
			"episode_id":"ep-1","run_ids":[1],"total_runs":1,
			"recorded_roots":[{"run_id":"99","observed":false}],
			"coverage":{
				"scope":"selected_runs","status":"partial",
				"reasons":["missing_parent"],"missing_parent_run_ids":[99],"collection_truncated":true
			}
		}]
	}`), &logs))
	data, err := buildEpisodeOutcomes(logs, []EpisodeOutcomeObservation{episodeObservation(1, "item-1", OutcomeStatusAccepted)},
		map[int64]episodeOutcomeCost{1: episodeCost(2)})
	require.NoError(t, err)
	row := data.Episodes[0]
	require.NotNil(t, row.MembershipCoverage)
	assert.Equal(t, "partial", row.MembershipCoverage.Status)
	assert.True(t, row.MembershipCoverage.CollectionTruncated)
	assert.Contains(t, row.Qualifications, "Membership coverage: missing_parent")
	assert.Nil(t, row.CostPerAcceptedOutcome)
}

func TestNewOutcomesEpisodesSubcommand(t *testing.T) {
	t.Parallel()
	cmd := NewOutcomesCommand()
	found, _, err := cmd.Find([]string{"episodes"})
	require.NoError(t, err)
	require.Equal(t, "episodes", found.Name())
	assert.NotNil(t, found.Flags().Lookup("logs"))
	assert.NotNil(t, found.Flags().Lookup("outcomes-dir"))
	assert.Equal(t, defaultLogsOutputDir, found.Flags().Lookup("output").DefValue)
}
