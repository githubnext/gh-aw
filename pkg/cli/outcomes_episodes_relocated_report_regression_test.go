//go:build !integration

package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOutcomesEpisodesRelocatedReportUsesLocalRunDirectory(t *testing.T) {
	dir := t.TempDir()
	outputDir := filepath.Join(dir, "new-cache")
	runDir := filepath.Join(outputDir, "run-42")
	require.NoError(t, os.MkdirAll(filepath.Join(runDir, "usage"), 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(runDir, "usage", "agent_usage.jsonl"), []byte("{\"aic\":1.25}\n"), 0o600))
	logs := episodeLogs(42)
	logs.Runs[0].LogsPath = filepath.Join(dir, "old-cache", "run-42")
	require.NoError(t, os.MkdirAll(filepath.Join(logs.Runs[0].LogsPath, "usage"), 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(logs.Runs[0].LogsPath, "usage", "agent_usage.jsonl"), []byte("{\"aic\":99}\n"), 0o600))
	content, err := json.Marshal(logs)
	require.NoError(t, err)
	logsPath := filepath.Join(dir, "logs.json")
	require.NoError(t, os.WriteFile(logsPath, content, 0o600))
	outcomesDir := filepath.Join(dir, "outcomes")
	writeOutcomeJSONL(outcomesDir, 42, []OutcomeReport{episodeObservation(42, "item-1", OutcomeStatusAccepted).OutcomeReport})
	var stdout, stderr bytes.Buffer
	require.NoError(t, runOutcomesEpisodes(logsPath, outcomesDir, outputDir, true, &stdout, &stderr))
	var data EpisodeOutcomesData
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &data))
	require.Len(t, data.Episodes, 1)
	require.NotNil(t, data.Episodes[0].ObservedAIC)
	require.InDelta(t, 1.25, *data.Episodes[0].ObservedAIC, 1e-9)
	require.Empty(t, stderr.String())
}

func TestEpisodeOutcomeRelocatedFallbackRemainsConfined(t *testing.T) {
	for _, fallback := range []string{"missing", "file", "symlink", "directory"} {
		t.Run(fallback, func(t *testing.T) {
			dir := t.TempDir()
			outputDir := filepath.Join(dir, "output")
			require.NoError(t, os.Mkdir(outputDir, 0o700))
			runDir := filepath.Join(outputDir, "run-42")
			external := filepath.Join(dir, "external")
			require.NoError(t, os.Mkdir(external, 0o700))
			switch fallback {
			case "file":
				require.NoError(t, os.WriteFile(runDir, nil, 0o600))
			case "symlink":
				require.NoError(t, os.Symlink(external, runDir))
			case "directory":
				require.NoError(t, os.Mkdir(runDir, 0o700))
			}
			resolved, err := episodeOutcomeRunDirectory([]RunData{{RunID: 42, LogsPath: external}}, outputDir, 42)
			if fallback == "directory" {
				require.NoError(t, err)
				require.Equal(t, runDir, resolved)
			} else {
				require.ErrorContains(t, err, "outside --output")
			}
		})
	}
}
