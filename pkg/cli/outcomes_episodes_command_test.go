//go:build !integration

package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunOutcomesEpisodesOffline(t *testing.T) {
	t.Parallel()
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

	run := func() (EpisodeOutcomesData, string) {
		t.Helper()
		var stdout, stderr bytes.Buffer
		cmd := NewOutcomesCommand()
		cmd.SetOut(&stdout)
		cmd.SetErr(&stderr)
		cmd.SetArgs([]string{"episodes", "--logs", logsPath, "--outcomes-dir", outcomesDir, "--output", dir, "--json"})
		require.NoError(t, cmd.Execute())
		content := stdout.Bytes()
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
		return data, stderr.String()
	}
	data, warnings := run()
	assert.Empty(t, warnings)
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
	data, warnings = run()
	assert.Contains(t, warnings, "Run 2: agent_usage.jsonl line 1: malformed usage record")
	assert.Equal(t, "partial", data.Episodes[0].CostCoverage)
	assert.InDelta(t, 6.25, *data.Episodes[0].ObservedAIC, 1e-9, "only readable usage is observed")
	assert.Contains(t, data.Episodes[0].Qualifications, "Run 2: agent_usage.jsonl line 1: malformed usage record; observed cost may be incomplete")
	assert.Nil(t, data.Episodes[0].CostPerAcceptedOutcome)
	rendered.Reset()
	require.NoError(t, renderEpisodeOutcomes(&rendered, data))
	assert.Contains(t, rendered.String(), "AIC per accepted outcome: unavailable")
}

func TestRunOutcomesEpisodesWithBoundedRetrieval(t *testing.T) {
	f := newEpisodeRetrievalFixture(t)
	f.add("org/one", 2, "canonical", "org/one", 2)
	opts := episodeRetrievalTestOptions(t)
	logs, err := retrieveEpisodeLogs(t.Context(), opts, f.request)
	require.NoError(t, err)
	content, err := json.Marshal(logs)
	require.NoError(t, err)
	logsPath := filepath.Join(opts.OutputDir, "episode.json")
	require.NoError(t, os.WriteFile(logsPath, content, 0o600))
	outcomesDir := filepath.Join(opts.OutputDir, "outcomes")
	observation := episodeObservation(2, "item-1", OutcomeStatusAccepted)
	writeOutcomeJSONL(outcomesDir, 2, []OutcomeReport{observation.OutcomeReport})
	var stdout, stderr bytes.Buffer
	require.NoError(t, runOutcomesEpisodes(logsPath, outcomesDir, opts.OutputDir, true, &stdout, &stderr))
	var data EpisodeOutcomesData
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &data))
	require.Len(t, data.Episodes, 1)
	require.NotNil(t, data.Episodes[0].ObservedAIC)
	require.InDelta(t, 0.25, *data.Episodes[0].ObservedAIC, 1e-9)
	require.Equal(t, 1, data.Episodes[0].Summary.Accepted)
	require.Empty(t, stderr.String())
}

type episodeOutputErrorWriter struct {
	err error
}

func (w episodeOutputErrorWriter) Write([]byte) (int, error) {
	return 0, w.err
}

func TestOutcomesEpisodesConfiguredWriters(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name       string
		json       bool
		usage      string
		usageFile  string
		warning    string
		cost       string
		failStdout bool
		failStderr bool
		wantErr    string
	}{
		{name: "human stdout", usage: `{"aic":1.25}`},
		{name: "JSON stdout", json: true, usage: `{"aic":1.25}`},
		{name: "human warnings", usage: `{"aic":1.25}` + "\ninvalid", warning: "Run 1: agent_usage.jsonl line 2: malformed usage record"},
		{name: "JSON warnings", json: true, usage: `{"aic":1.25}` + "\ninvalid", warning: "Run 1: agent_usage.jsonl line 2: malformed usage record"},
		{name: "human unavailable", cost: "unavailable"},
		{name: "human explicit zero", usage: `{"aic":0}`, cost: "0.000"},
		{name: "unpriced warnings", usage: `{"model":"unpriced","input_tokens":10}`, cost: "unavailable", warning: "Run 1: agent_usage.jsonl line 1: unpriced usage"},
		{name: "read error warning", usageFile: "agent_usage.json", usage: "{", cost: "unavailable", warning: "Run 1: Failed to read usage; cost is unavailable:"},
		{name: "human write error", usage: `{"aic":1.25}`, failStdout: true, wantErr: "failed to write episode outcomes"},
		{name: "JSON write error", json: true, usage: `{"aic":1.25}`, failStdout: true, wantErr: "failed to write episode outcomes"},
		{name: "warning write error", usage: "invalid", failStderr: true, wantErr: "failed to write episode usage warning"},
		{name: "JSON warning write error", json: true, usage: "invalid", failStderr: true, wantErr: "failed to write episode usage warning"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			logsPath := filepath.Join(dir, "logs.json")
			content, err := json.Marshal(episodeLogs(1))
			require.NoError(t, err)
			require.NoError(t, os.WriteFile(logsPath, content, 0o600))
			outcomesDir := filepath.Join(dir, "outcomes")
			writeOutcomeJSONL(outcomesDir, 1, []OutcomeReport{episodeObservation(1, "item-1", OutcomeStatusAccepted).OutcomeReport})
			usageDir := filepath.Join(dir, "run-1", "usage")
			require.NoError(t, os.MkdirAll(usageDir, 0o700))
			usageFile := tc.usageFile
			if usageFile == "" {
				usageFile = "agent_usage.jsonl"
			}
			require.NoError(t, os.WriteFile(filepath.Join(usageDir, usageFile), []byte(tc.usage+"\n"), 0o600))
			var stdout, stderr bytes.Buffer
			writeErr := errors.New("writer failed")
			cmd := NewOutcomesCommand()
			cmd.SilenceErrors = true
			cmd.SilenceUsage = true
			cmd.SetOut(&stdout)
			cmd.SetErr(&stderr)
			if tc.failStdout {
				cmd.SetOut(episodeOutputErrorWriter{writeErr})
			}
			if tc.failStderr {
				cmd.SetErr(episodeOutputErrorWriter{writeErr})
			}
			args := []string{"episodes", "--logs", logsPath, "--outcomes-dir", outcomesDir, "--output", dir}
			if tc.json {
				args = append(args, "--json")
			}
			cmd.SetArgs(args)
			err = cmd.Execute()
			if tc.wantErr != "" {
				require.ErrorContains(t, err, tc.wantErr)
				assert.ErrorIs(t, err, writeErr)
				if tc.failStderr {
					assert.Empty(t, stdout.String())
				}
				return
			}
			require.NoError(t, err)
			if tc.json {
				assert.True(t, json.Valid(stdout.Bytes()), stdout.String())
			} else {
				cost := tc.cost
				if cost == "" {
					cost = "1.250"
				}
				assert.Contains(t, stdout.String(), "Observed AIC: "+cost)
				assert.Contains(t, stdout.String(), "Observed outcomes: 1 accepted")
			}
			assert.NotContains(t, stderr.String(), "Observed outcomes:")
			if tc.warning != "" {
				assert.Contains(t, stderr.String(), tc.warning)
				assert.NotContains(t, stdout.String(), "WARNING")
			} else {
				assert.Empty(t, stderr.String())
			}
		})
	}
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

func TestOutcomesEpisodesInputErrors(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	logs := filepath.Join(dir, "logs.json")
	require.NoError(t, os.WriteFile(logs, []byte(`{"episodes":[]}`), 0o600))
	require.ErrorContains(t, runOutcomesEpisodes(logs, dir, dir, true, io.Discard, io.Discard), "regenerate it")
	require.NoError(t, os.WriteFile(logs, []byte(`{`), 0o600))
	require.ErrorContains(t, runOutcomesEpisodes(logs, dir, dir, true, io.Discard, io.Discard), "failed to parse episode logs")
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
