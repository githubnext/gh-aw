//go:build !integration

package cli

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func registerCustomGraderFixture(t *testing.T, runDir, prefix string) {
	t.Helper()
	writeGraderFiles(t, filepath.Join(runDir, prefix+"usage", "graders"), map[string]any{
		"version": 2, "results": []map[string]any{{"id": "loops", "source": "builtin", "value": 0, "status": "pass"}},
	}, map[string]any{"version": 2, "graders": []any{}, "customGraders": []string{"custom-count"}})
}

func customGraderArtifactFixture(t *testing.T) map[string]string {
	t.Helper()
	results, err := json.Marshal(map[string]any{
		"version": 2, "results": []map[string]any{
			{"id": "loops", "source": "builtin", "value": 0, "status": "pass"},
			{"id": "custom-count", "source": "inline", "value": 3, "status": "pass"},
		},
	})
	require.NoError(t, err)
	return map[string]string{
		"grader_results.json":  string(results),
		"grader_manifest.json": `{"version":2,"graders":[{"id":"custom-count","direction":"lower_is_better"}],"customGraders":["custom-count"]}`,
	}
}

func TestAuditCustomGradersNotRegisteredAvoidsRequests(t *testing.T) {
	calls := installSessionTestGH(t, nil)
	dir := t.TempDir()
	writeGraderFiles(t, filepath.Join(dir, "usage", "graders"), sampleGraderResults(), sampleGraderManifest())
	downloaded, err := downloadRegisteredCustomGraders(context.Background(), auditRunConfig{runID: 123, outputDir: dir})
	require.NoError(t, err)
	assert.False(t, downloaded)
	_, err = os.Stat(calls)
	assert.True(t, os.IsNotExist(err), "built-in-only runs must not make extra requests")
}

func TestAuditDownloadsRegisteredCustomGraders(t *testing.T) {
	for _, prefix := range []string{"", "a1b2c3-"} {
		t.Run(prefix, func(t *testing.T) {
			calls := installSessionTestGH(t, map[string]map[string]string{prefix + "graders": customGraderArtifactFixture(t)})
			dir := t.TempDir()
			registerCustomGraderFixture(t, dir, prefix)
			cfg := auditRunConfig{runID: 123, outputDir: dir, owner: "owner", repo: "repo"}
			downloaded, err := downloadRegisteredCustomGraders(context.Background(), cfg)
			require.NoError(t, err)
			assert.True(t, downloaded)
			results := extractGradersData(dir)
			require.NotNil(t, results)
			require.Len(t, results.Results, 2)
			assert.Equal(t, "custom-count", results.Results[0].ID)
			assert.Equal(t, "loops", results.Results[1].ID)
			assert.InDelta(t, 3, *results.Results[0].Value, 1e-9)
			before, err := os.ReadFile(calls)
			require.NoError(t, err)
			downloaded, err = downloadRegisteredCustomGraders(context.Background(), cfg)
			require.NoError(t, err)
			assert.False(t, downloaded)
			after, err := os.ReadFile(calls)
			require.NoError(t, err)
			assert.Equal(t, string(before), string(after), "already-downloaded custom results must not cause more requests")
		})
	}
}

func TestAuditMissingCustomArtifactKeepsBuiltinResults(t *testing.T) {
	installSessionTestGH(t, map[string]map[string]string{"usage": {}})
	dir := t.TempDir()
	registerCustomGraderFixture(t, dir, "")
	downloaded, err := downloadRegisteredCustomGraders(context.Background(), auditRunConfig{runID: 123, outputDir: dir})
	require.NoError(t, err)
	assert.False(t, downloaded)
	require.Len(t, extractGradersData(dir).Results, 1)
}

func TestAuditCustomArtifactErrorsAreSurfaced(t *testing.T) {
	for _, env := range []string{"SESSION_TEST_LIST_ERROR", "SESSION_TEST_DOWNLOAD_ERROR"} {
		t.Run(env, func(t *testing.T) {
			installSessionTestGH(t, map[string]map[string]string{"graders": customGraderArtifactFixture(t)})
			t.Setenv(env, "graders")
			dir := t.TempDir()
			registerCustomGraderFixture(t, dir, "")
			cfg := auditRunConfig{runID: 123, outputDir: dir}
			downloaded, err := downloadRegisteredCustomGraders(context.Background(), cfg)
			require.ErrorContains(t, err, "custom grader artifact")
			assert.False(t, downloaded)
			assert.NoDirExists(t, filepath.Join(dir, "graders"))
			require.Len(t, extractGradersData(dir).Results, 1)
			t.Setenv(env, "")
			downloaded, err = downloadRegisteredCustomGraders(context.Background(), cfg)
			require.NoError(t, err)
			assert.True(t, downloaded)
			require.Len(t, extractGradersData(dir).Results, 2)
		})
	}
}

func TestCachedAuditRefreshesNewCustomResults(t *testing.T) {
	installSessionTestGH(t, map[string]map[string]string{"graders": customGraderArtifactFixture(t)})
	dir := t.TempDir()
	registerCustomGraderFixture(t, dir, "")
	require.NoError(t, saveRunSummary(dir, &RunSummary{
		CLIVersion: GetVersion(), RunID: 123, ProcessedAt: time.Now(),
		RunAnalysis: RunAnalysis{Run: WorkflowRun{DatabaseID: 123}},
	}, false))
	done, _, err := renderCachedAuditIfAvailable(context.Background(), auditRunConfig{
		runID: 123, outputDir: dir, artifactFilter: []string{constants.UsageArtifactName.String()},
	})
	require.NoError(t, err)
	assert.False(t, done, "a cached built-in-only report must be rebuilt when custom results become available")
	require.Len(t, extractGradersData(dir).Results, 2)
}
