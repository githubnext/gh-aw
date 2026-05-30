//go:build !integration

package cli

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildSafeOutputChainMetrics(t *testing.T) {
	tmpDir := testutil.TempDir(t, "test-safe-output-chains-*")

	manifest := `{"type":"create_issue","repo":"github/gh-aw","number":101,"temporaryId":"aw_alpha","timestamp":"2026-01-01T00:00:00Z"}
{"type":"add_comment","repo":"github/gh-aw","number":101,"timestamp":"2026-01-01T00:01:00Z"}
{"type":"assign_to_agent","repo":"github/gh-aw","number":101,"timestamp":"2026-01-01T00:02:00Z"}
{"type":"create_issue","repo":"github/gh-aw","number":102,"temporaryId":"aw_beta","timestamp":"2026-01-01T00:03:00Z"}
{"type":"close_issue","repo":"github/gh-aw","number":102,"timestamp":"2026-01-01T00:04:00Z"}
`
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, safeOutputItemsManifestFilename), []byte(manifest), 0o600), "should write manifest fixture")

	temporaryIDMap := `{
  "aw_alpha": {"repo": "github/gh-aw", "number": 101},
  "aw_beta": {"repo": "github/gh-aw", "number": 102}
}
`
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, constants.TemporaryIdMapFilename), []byte(temporaryIDMap), 0o600), "should write temporary ID map fixture")

	metrics := buildSafeOutputChainMetrics(tmpDir)

	assert.Equal(t, 5, metrics.ManifestEntryCount, "manifest entries should be counted")
	assert.Equal(t, 2, metrics.TemporaryIDMappings, "temporary ID mappings should be counted")
	assert.Equal(t, 2, metrics.ChainedTargetCount, "targets with multiple actions should be counted")
	assert.Equal(t, 3, metrics.ChainedFollowupActionCount, "follow-up actions should count actions beyond the first")
	assert.Equal(t, 1, metrics.DelegatedTempTargetCount, "delegated temporary-ID targets should be counted")
	assert.Equal(t, 1, metrics.ClosedTempTargetCount, "closed temporary-ID targets should be counted")
}

func TestBuildSafeOutputChainMetricsIgnoresMalformedTemporaryIDMap(t *testing.T) {
	tmpDir := testutil.TempDir(t, "test-safe-output-chains-malformed-map-*")

	manifest := `{"type":"create_issue","repo":"github/gh-aw","number":301,"temporaryId":"aw_alpha","timestamp":"2026-01-01T00:00:00Z"}
{"type":"add_comment","repo":"github/gh-aw","number":301,"timestamp":"2026-01-01T00:01:00Z"}
`
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, safeOutputItemsManifestFilename), []byte(manifest), 0o600), "should write manifest fixture")
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, constants.TemporaryIdMapFilename), []byte(`{"aw_alpha":`), 0o600), "should write malformed temporary ID map fixture")

	metrics := buildSafeOutputChainMetrics(tmpDir)

	assert.Equal(t, 2, metrics.ManifestEntryCount, "manifest entries should still be counted when the temporary ID map is malformed")
	assert.Equal(t, 0, metrics.TemporaryIDMappings, "malformed temporary ID map should not produce mappings")
	assert.Equal(t, 0, metrics.ChainedTargetCount, "malformed temporary ID map should suppress chained target aggregation")
	assert.Equal(t, 0, metrics.ChainedFollowupActionCount, "malformed temporary ID map should suppress chained follow-up aggregation")
	assert.Equal(t, 0, metrics.DelegatedTempTargetCount, "malformed temporary ID map should suppress delegated target aggregation")
	assert.Equal(t, 0, metrics.ClosedTempTargetCount, "malformed temporary ID map should suppress closed target aggregation")
}

func TestBuildLogsDataIncludesSafeOutputChainMetrics(t *testing.T) {
	tmpDir := testutil.TempDir(t, "test-logs-chain-metrics-*")
	runDir := filepath.Join(tmpDir, "run-91001")
	require.NoError(t, os.MkdirAll(runDir, 0o755), "should create run directory")

	manifest := `{"type":"create_issue","repo":"github/gh-aw","number":201,"temporaryId":"aw_alpha","timestamp":"2026-01-01T00:00:00Z"}
{"type":"add_comment","repo":"github/gh-aw","number":201,"timestamp":"2026-01-01T00:01:00Z"}
{"type":"assign_to_agent","repo":"github/gh-aw","number":201,"timestamp":"2026-01-01T00:02:00Z"}
`
	require.NoError(t, os.WriteFile(filepath.Join(runDir, safeOutputItemsManifestFilename), []byte(manifest), 0o600), "should write manifest fixture")
	require.NoError(t, os.WriteFile(filepath.Join(runDir, constants.TemporaryIdMapFilename), []byte(`{"aw_alpha":{"repo":"github/gh-aw","number":201}}`), 0o600), "should write temporary ID map fixture")

	processedRuns := []ProcessedRun{
		{
			Run: WorkflowRun{
				DatabaseID:   91001,
				WorkflowName: "Weekly Episode Risk Report",
				WorkflowPath: ".github/workflows/weekly-episode-risk-report.yml",
				Status:       "completed",
				Conclusion:   "success",
				Duration:     2 * time.Minute,
				TokenUsage:   400,
				CreatedAt:    time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC),
				StartedAt:    time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC),
				UpdatedAt:    time.Date(2026, 1, 1, 12, 2, 0, 0, time.UTC),
				LogsPath:     runDir,
			},
			BehaviorFingerprint: &BehaviorFingerprint{ActuationStyle: "selective_write"},
		},
	}

	logsData := buildLogsData(processedRuns, tmpDir, nil)
	require.Len(t, logsData.Runs, 1, "should produce one run")
	require.Len(t, logsData.Episodes, 1, "should produce one episode")

	run := logsData.Runs[0]
	assert.Equal(t, temporaryIDMapStatusLoaded, run.TemporaryIDMapStatus, "run should expose loaded temporary ID map status")
	assert.Equal(t, 1, run.TemporaryIDMappings, "run should expose temp-ID mapping count")
	assert.Equal(t, 1, run.ChainedTargetCount, "run should expose chained target count")
	assert.Equal(t, 2, run.ChainedFollowupActionCount, "run should expose chained follow-up action count")
	assert.Equal(t, 1, run.DelegatedTempTargetCount, "run should expose delegated temp targets")
	assert.Equal(t, 0, run.ClosedTempTargetCount, "run should expose closed temp targets")

	episode := logsData.Episodes[0]
	assert.Equal(t, 1, episode.TemporaryIDMappings, "episode should roll up temp-ID mappings")
	assert.Equal(t, 1, episode.ChainedTargetCount, "episode should roll up chained targets")
	assert.Equal(t, 2, episode.ChainedFollowupActionCount, "episode should roll up chained follow-up actions")
	assert.Equal(t, 1, episode.DelegatedTempTargetCount, "episode should roll up delegated temp targets")

	assert.Equal(t, 1, logsData.Summary.RunsWithTemporaryIDChains, "summary should count runs with temp-ID chains")
	assert.Equal(t, 1, logsData.Summary.RunsWithDelegatedTempTargets, "summary should count runs with delegated temp targets")
	assert.Equal(t, 1, logsData.Summary.TotalTemporaryIDMappings, "summary should total temp-ID mappings")
	assert.Equal(t, 1, logsData.Summary.TotalChainedTargets, "summary should total chained targets")
	assert.Equal(t, 2, logsData.Summary.TotalChainedFollowupActions, "summary should total chained follow-up actions")
	assert.Equal(t, 0, logsData.Summary.TotalClosedTempTargets, "summary should total closed temp targets")
	assert.Equal(t, 0, logsData.Summary.RunsWithMissingTemporaryIDMap, "summary should not count missing temporary ID maps when the map is present")
	assert.Equal(t, 0, logsData.Summary.RunsWithInvalidTemporaryIDMap, "summary should not count invalid temporary ID maps when the map is valid")
}

func TestBuildLogsDataTracksTemporaryIDMapHealth(t *testing.T) {
	tmpDir := testutil.TempDir(t, "test-logs-temp-map-health-*")
	missingRunDir := filepath.Join(tmpDir, "run-92001")
	invalidRunDir := filepath.Join(tmpDir, "run-92002")
	require.NoError(t, os.MkdirAll(missingRunDir, 0o755), "should create missing-map run directory")
	require.NoError(t, os.MkdirAll(invalidRunDir, 0o755), "should create invalid-map run directory")

	missingManifest := `{"type":"create_issue","repo":"github/gh-aw","number":401,"temporaryId":"aw_missing","timestamp":"2026-01-01T00:00:00Z"}
`
	invalidManifest := `{"type":"create_issue","repo":"github/gh-aw","number":402,"temporaryId":"aw_invalid","timestamp":"2026-01-01T00:00:00Z"}
{"type":"close_issue","repo":"github/gh-aw","number":402,"timestamp":"2026-01-01T00:01:00Z"}
`
	require.NoError(t, os.WriteFile(filepath.Join(missingRunDir, safeOutputItemsManifestFilename), []byte(missingManifest), 0o600), "should write manifest for missing-map run")
	require.NoError(t, os.WriteFile(filepath.Join(invalidRunDir, safeOutputItemsManifestFilename), []byte(invalidManifest), 0o600), "should write manifest for invalid-map run")
	require.NoError(t, os.WriteFile(filepath.Join(invalidRunDir, constants.TemporaryIdMapFilename), []byte(`{"aw_invalid":`), 0o600), "should write malformed temporary ID map")

	processedRuns := []ProcessedRun{
		{
			Run: WorkflowRun{
				DatabaseID:   92001,
				WorkflowName: "Missing Temp Map",
				Status:       "completed",
				Conclusion:   "success",
				CreatedAt:    time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC),
				LogsPath:     missingRunDir,
			},
		},
		{
			Run: WorkflowRun{
				DatabaseID:   92002,
				WorkflowName: "Invalid Temp Map",
				Status:       "completed",
				Conclusion:   "success",
				CreatedAt:    time.Date(2026, 1, 1, 13, 0, 0, 0, time.UTC),
				LogsPath:     invalidRunDir,
			},
		},
	}

	logsData := buildLogsData(processedRuns, tmpDir, nil)
	require.Len(t, logsData.Runs, 2, "should produce two runs")

	assert.Equal(t, temporaryIDMapStatusMissing, logsData.Runs[0].TemporaryIDMapStatus, "run should report missing temporary ID map status")
	assert.Equal(t, temporaryIDMapStatusInvalid, logsData.Runs[1].TemporaryIDMapStatus, "run should report invalid temporary ID map status")
	assert.Equal(t, 1, logsData.Summary.RunsWithMissingTemporaryIDMap, "summary should count runs with missing temporary ID maps")
	assert.Equal(t, 1, logsData.Summary.RunsWithInvalidTemporaryIDMap, "summary should count runs with invalid temporary ID maps")
	assert.Equal(t, 0, logsData.Summary.TotalClosedTempTargets, "summary should not count closed temp targets when the temp map cannot be resolved")
	assert.Equal(t, 0, logsData.Summary.TotalTemporaryIDMappings, "summary should not count temp mappings for missing or invalid maps")
}

func TestBuildLogsDataLeavesTemporaryIDMapStatusEmptyWithoutSafeOutputArtifacts(t *testing.T) {
	tmpDir := testutil.TempDir(t, "test-logs-no-safe-output-artifacts-*")
	runDir := filepath.Join(tmpDir, "run-93001")
	require.NoError(t, os.MkdirAll(runDir, 0o755), "should create run directory")

	processedRuns := []ProcessedRun{
		{
			Run: WorkflowRun{
				DatabaseID:   93001,
				WorkflowName: "No Safe Outputs",
				Status:       "completed",
				Conclusion:   "success",
				CreatedAt:    time.Date(2026, 1, 1, 14, 0, 0, 0, time.UTC),
				LogsPath:     runDir,
			},
		},
	}

	logsData := buildLogsData(processedRuns, tmpDir, nil)
	require.Len(t, logsData.Runs, 1, "should produce one run")

	assert.Empty(t, logsData.Runs[0].TemporaryIDMapStatus, "run should leave temp map status empty when no safe-output artifacts exist")
	assert.Equal(t, 0, logsData.Summary.RunsWithMissingTemporaryIDMap, "summary should not count missing temp maps when no safe-output artifacts exist")
	assert.Equal(t, 0, logsData.Summary.RunsWithInvalidTemporaryIDMap, "summary should not count invalid temp maps when no safe-output artifacts exist")
	assert.Equal(t, 0, logsData.Summary.TotalTemporaryIDMappings, "summary should not count temp mappings when no safe-output artifacts exist")
	assert.Equal(t, 0, logsData.Summary.RunsWithTemporaryIDChains, "summary should not count temp-ID chains when no safe-output artifacts exist")
}
