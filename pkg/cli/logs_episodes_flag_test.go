//go:build !integration

package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func logsEpisodesFlagFixture(t *testing.T) ([]ProcessedRun, []OutcomesData, string) {
	t.Helper()
	dir := t.TempDir()
	runDir := filepath.Join(dir, "run-1")
	usageDir := filepath.Join(runDir, "usage")
	require.NoError(t, os.MkdirAll(usageDir, 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(usageDir, "agent_usage.jsonl"), []byte("{\"aic\":2.5}\n"), 0o600))
	run := ProcessedRun{
		Run:       WorkflowRun{DatabaseID: 1, Repository: "org/repo", WorkflowName: "producer", LogsPath: runDir},
		AwContext: &AwContext{EpisodeID: "ep-1"},
	}
	item := episodeObservation(1, "item-1", OutcomeStatusAccepted).OutcomeReport
	reports := []OutcomesData{{
		RunID: 1, Workflow: "producer", Items: []OutcomeReport{item},
		Summary: ComputeOutcomeSummary([]OutcomeReport{item}, nil),
	}}
	return []ProcessedRun{run}, reports, dir
}

func TestLogsEpisodesOnlyOptions(t *testing.T) {
	t.Parallel()
	cmd := NewLogsCommand()
	require.NoError(t, cmd.Flags().Set("episodes", "true"))
	options, err := loadCommonLogsOptions(cmd)
	require.NoError(t, err)
	assert.True(t, options.Episodes)
	assert.False(t, options.Outcomes, "episode membership must not imply live outcome requests")
	assert.Equal(t, []string{"usage"}, options.ArtifactSets)
	stdin, err := loadStdinLogsOptions(cmd)
	require.NoError(t, err)
	assert.True(t, stdin.Episodes)
	assert.False(t, stdin.Outcomes)
	assert.Equal(t, options.ArtifactSets, stdin.ArtifactSets)
}

func TestLogsEpisodesImplicitOutcomeJoin(t *testing.T) {
	calls := installOutcomeAPIFixture(t, "printf 'unexpected GitHub request\\n' >&2; exit 1")
	require.NoError(t, os.WriteFile(calls, nil, 0o600))
	runs, reports, dir := logsEpisodesFlagFixture(t)
	data, err := prepareLogsData(runs, renderLogsOutputOptions{
		outputDir: dir, summaryFile: "summary.json", episodes: true, outcomeReports: reports,
	})
	require.NoError(t, err)
	require.NotNil(t, data.EpisodeOutcomes)
	collectedCalls := outcomeAPICalls(t, calls)
	outcomesDir := filepath.Join(dir, "observations")
	for _, report := range reports {
		require.NoError(t, writeOutcomeJSONL(outcomesDir, report.RunID, report.Items))
	}
	var offlineJSON, diagnostics bytes.Buffer
	require.NoError(t, runOutcomesEpisodes(filepath.Join(dir, "summary.json"), outcomesDir, dir, true, &offlineJSON, &diagnostics))
	var offline EpisodeOutcomesData
	require.NoError(t, json.Unmarshal(offlineJSON.Bytes(), &offline))
	assert.Equal(t, offline, *data.EpisodeOutcomes, "combined flags must reuse the offline join semantics")
	require.NoError(t, attachLogsOutcomes(&data, reports))
	assert.Equal(t, collectedCalls, outcomeAPICalls(t, calls), "neither join must make additional GitHub requests")
}

func TestLogsEpisodesTextFormats(t *testing.T) {
	for _, format := range []string{"", "console", "tsv", "pretty", "markdown"} {
		t.Run(format, func(t *testing.T) {
			runs, reports, dir := logsEpisodesFlagFixture(t)
			stdout, stderr := captureOutput(t, func() error {
				return renderLogsOutput(runs, renderLogsOutputOptions{
					outputDir: dir, format: format, episodes: true, outcomeReports: reports,
				})
			})
			out := stdout
			if format == "pretty" || format == "tsv" {
				out = stderr
				assert.NotContains(t, stdout, "Episode ep-1")
			}
			assert.Equal(t, 1, strings.Count(out, "Episode ep-1\nCoverage:"))
			assert.Contains(t, out, "1 accepted")
		})
	}
}

func TestLogsEpisodesMarkdownFile(t *testing.T) {
	runs, reports, dir := logsEpisodesFlagFixture(t)
	reportPath := filepath.Join(dir, "report.md")
	stdout, _ := captureOutput(t, func() error {
		return renderLogsOutput(runs, renderLogsOutputOptions{
			outputDir: dir, format: "markdown", reportFile: reportPath, episodes: true, outcomeReports: reports,
		})
	})
	assert.NotContains(t, stdout, "Episode ep-1")
	content, err := os.ReadFile(reportPath)
	require.NoError(t, err)
	assert.Equal(t, 1, strings.Count(string(content), "Episode ep-1\nCoverage:"))
	assert.Contains(t, string(content), "1 accepted")
}

func TestLogsEpisodesJSONAndMissingMembership(t *testing.T) {
	runs, reports, dir := logsEpisodesFlagFixture(t)
	stdout, stderr := captureOutput(t, func() error {
		return renderLogsOutput(runs, renderLogsOutputOptions{
			outputDir: dir, jsonOutput: true, episodes: true, outcomeReports: reports,
		})
	})
	var data LogsData
	require.NoError(t, json.Unmarshal([]byte(stdout), &data))
	assert.NotContains(t, stderr, "Episode ep-1")
	require.Len(t, data.Episodes, 1)
	require.NotNil(t, data.EpisodeOutcomes)
	assert.Equal(t, 1, data.EpisodeOutcomes.Episodes[0].Summary.Accepted)

	var text bytes.Buffer
	require.NoError(t, renderLogsEpisodeOutcomes(&text, LogsData{}, true))
	assert.Contains(t, text.String(), "episode membership is unavailable")
	text.Reset()
	require.NoError(t, renderLogsEpisodeOutcomes(&text, LogsData{}, false))
	assert.Empty(t, text.String(), "default output must remain unchanged")
}

func TestLogsEpisodesCrossRunJSON(t *testing.T) {
	for _, format := range []string{"pretty", "markdown"} {
		t.Run(format, func(t *testing.T) {
			runs, reports, dir := logsEpisodesFlagFixture(t)
			for _, episodes := range []bool{false, true} {
				stdout, _ := captureOutput(t, func() error {
					return renderLogsOutput(runs, renderLogsOutputOptions{
						outputDir: dir, format: format, jsonOutput: true, episodes: episodes, outcomeReports: reports,
					})
				})
				var report CrossRunAuditReport
				require.NoError(t, json.Unmarshal([]byte(stdout), &report))
				require.NotNil(t, report.EpisodeOutcomes)
				assert.Equal(t, 1, report.EpisodeOutcomes.Episodes[0].Summary.Accepted)
				if episodes {
					require.Len(t, report.Episodes, 1)
					assert.Equal(t, "ep-1", report.Episodes[0].EpisodeID)
				} else {
					assert.Empty(t, report.Episodes, "preserve existing cross-run output without the flag")
				}
			}
		})
	}
}
