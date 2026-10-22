//go:build !integration

package cli

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLogsOutcomeOptions(t *testing.T) {
	t.Parallel()
	cmd := NewLogsCommand()
	require.NoError(t, cmd.Flags().Set("outcomes", "true"))
	require.NoError(t, cmd.Flags().Set("outcomes-dir", "observations"))
	options, err := loadCommonLogsOptions(cmd)
	require.NoError(t, err)
	assert.True(t, options.Outcomes)
	assert.Equal(t, "observations", options.OutcomesDir)
	assert.Equal(t, []string{"usage", "outcomes"}, options.ArtifactSets)
	assert.Equal(t, []string{"usage", "safe-outputs-items", "agent-output-fallback"}, ResolveArtifactFilter(options.ArtifactSets))
	assert.False(t, shouldDownloadWorkflowRunLogs(ResolveArtifactFilter(options.ArtifactSets)))
	assert.False(t, cachedJSONLCanSatisfy(ResolveArtifactFilter(options.ArtifactSets), false, false, false))
	stdin, err := loadStdinLogsOptions(cmd)
	require.NoError(t, err)
	assert.True(t, stdin.Outcomes)
	assert.Equal(t, options.OutcomesDir, stdin.OutcomesDir)
	assert.Equal(t, options.ArtifactSets, stdin.ArtifactSets)

	cmd = NewLogsCommand()
	require.NoError(t, cmd.Flags().Set("outcomes-dir", "observations"))
	_, err = loadCommonLogsOptions(cmd)
	require.ErrorContains(t, err, "--outcomes-dir requires --outcomes")

	episode := newLogsEpisodeCommand()
	assert.NotNil(t, episode.Flags().Lookup("outcomes"))
	assert.NotNil(t, episode.Flags().Lookup("outcomes-dir"))
}

func TestLogsOutcomeArtifactSets(t *testing.T) {
	t.Parallel()
	sets := []string{"usage"}
	assert.Equal(t, []string{"usage", "outcomes"}, logsOutcomeArtifactSets(sets, true))
	assert.Equal(t, []string{"usage"}, sets, "do not mutate the caller's artifact selection")
	assert.Nil(t, logsOutcomeArtifactSets(nil, true), "unfiltered downloads already include manifests")
	assert.Equal(t, []string{"all"}, logsOutcomeArtifactSets([]string{"all"}, true))
	assert.Equal(t, []string{"usage"}, logsOutcomeArtifactSets(sets, false))
	assert.Equal(t, []string{"usage", "outcomes"}, normalizeLogsDownloadOptions(LogsDownloadOptions{Outcomes: true, ArtifactSets: sets}).ArtifactSets)
	assert.Equal(t, []string{"usage", "outcomes"}, normalizeStdinLogsOptions(StdinLogsOptions{Outcomes: true, ArtifactSets: sets}).ArtifactSets)
}

func TestLogsOutcomeRepository(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		url  string
		repo string
		want string
	}{
		{repo: "org/repo", want: "org/repo"},
		{url: "https://github.com/org/repo/actions/runs/1", repo: "org/repo", want: "org/repo"},
		{url: "https://github.example.com/org/repo/actions/runs/1", repo: "org/repo", want: "github.example.com/org/repo"},
		{url: "https://github.example.com/org/repo/actions/runs/1", repo: "github.example.com/org/repo", want: "github.example.com/org/repo"},
	} {
		assert.Equal(t, tc.want, logsOutcomeRepository(WorkflowRun{URL: tc.url, Repository: tc.repo}))
	}
	assert.Equal(t, "github.example.com/org/repo", logsOutcomeItemRepository("org/repo", "github.example.com/org/repo"))
	assert.Equal(t, "other.example.com/other/repo", logsOutcomeItemRepository("other.example.com/other/repo", "github.example.com/org/repo"))
}

func TestLogsOutcomeManifestEvidence(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name     string
		manifest string
		wantErr  string
		found    bool
	}{
		{name: "missing"},
		{name: "empty", manifest: "\n", found: true},
		{name: "corrupt", manifest: "{", wantErr: "unexpected EOF"},
		{name: "missing type", manifest: "{}", wantErr: "no type"},
		{name: "valid", manifest: "{\"type\":\"noop\"}\n", found: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			if tc.manifest != "" {
				require.NoError(t, os.WriteFile(filepath.Join(dir, safeOutputItemsManifestFilename), []byte(tc.manifest), 0o600))
			}
			run := ProcessedRun{Run: WorkflowRun{DatabaseID: 42, Repository: "acme/repo", LogsPath: dir}}
			items, found, err := readLogsOutcomeItems(run)
			if tc.wantErr != "" {
				require.ErrorContains(t, err, tc.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.found, found)
			report, err := collectRunOutcomes(context.Background(), run, nil, false)
			require.NoError(t, err)
			assert.Empty(t, report.Items)
			if tc.found {
				assert.Empty(t, report.EvalError)
			} else {
				assert.Contains(t, report.EvalError, "unavailable")
			}
			if len(items) > 0 {
				assert.Equal(t, "noop", items[0].Type)
			}
		})
	}
}

func TestLogsOutcomeUsesRunHostForManifestRepository(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, safeOutputItemsManifestFilename),
		[]byte(`{"type":"create_issue","repo":"org/repo","number":42}`+"\n"), 0o600))
	oldGet := outcomeEvidenceGHAPIGet
	t.Cleanup(func() { outcomeEvidenceGHAPIGet = oldGet })
	var requestRepo string
	outcomeEvidenceGHAPIGet = func(_ context.Context, endpoint, repo string) (map[string]any, error) {
		require.Equal(t, "issues/42", endpoint)
		requestRepo = repo
		return map[string]any{"state": "open"}, nil
	}
	run := ProcessedRun{Run: WorkflowRun{
		DatabaseID: 42, Repository: "org/repo", URL: "https://github.example.com/org/repo/actions/runs/42", LogsPath: dir,
	}}
	report, err := collectRunOutcomes(context.Background(), run, nil, false)
	require.NoError(t, err)
	require.Len(t, report.Items, 1)
	assert.Equal(t, "github.example.com/org/repo", requestRepo)
	assert.Equal(t, requestRepo, report.Items[0].Repo)
}

func TestLogsOutcomeAggregatesDistinctEpisodeManifests(t *testing.T) {
	runDir := t.TempDir()
	collectionDir := filepath.Join(runDir, "collection")
	require.NoError(t, os.Mkdir(collectionDir, 0o700))
	first := []byte(`{"type":"create_issue","number":1}` + "\n")
	second := []byte(`{"type":"close_issue","number":2}` + "\n")
	for name, content := range map[string][]byte{"10-outcomes": first, "11-outcomes": second} {
		dir := filepath.Join(runDir, name)
		require.NoError(t, os.Mkdir(dir, 0o700))
		require.NoError(t, os.WriteFile(filepath.Join(dir, safeOutputItemsManifestFilename), content, 0o600))
		require.NoError(t, os.WriteFile(filepath.Join(dir, ".episode-complete"), []byte("v1\n"), 0o600))
	}
	require.NoError(t, os.WriteFile(filepath.Join(collectionDir, ".episode-artifacts.json"),
		[]byte(`["../10-outcomes","../11-outcomes"]`), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(collectionDir, safeOutputItemsManifestFilename), first, 0o600))
	items, found, err := readLogsOutcomeItems(ProcessedRun{Run: WorkflowRun{LogsPath: collectionDir}})
	require.NoError(t, err)
	require.True(t, found)
	require.Len(t, items, 2)
	assert.Equal(t, []string{"create_issue", "close_issue"}, []string{items[0].Type, items[1].Type})
}

func TestLogsOutcomeAssignmentUsesFallbackArtifactEvidence(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, safeOutputItemsManifestFilename),
		[]byte(`{"type":"assign_to_agent","success":true}`+"\n"), 0o600))
	fallbackDir := filepath.Join(dir, "agent-output-fallback")
	require.NoError(t, os.Mkdir(fallbackDir, 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(fallbackDir, "safeoutputs.jsonl"),
		[]byte(`{"type":"assign_to_agent","issue_number":42}`+"\n"), 0o600))
	oldGet, oldArray := outcomeEvidenceGHAPIGet, outcomeEvidenceGHAPIGetArray
	t.Cleanup(func() {
		outcomeEvidenceGHAPIGet, outcomeEvidenceGHAPIGetArray = oldGet, oldArray
	})
	var requests []string
	outcomeEvidenceGHAPIGet = func(_ context.Context, endpoint, repo string) (map[string]any, error) {
		assert.Equal(t, "org/repo", repo)
		requests = append(requests, endpoint)
		return map[string]any{"state": "open"}, nil
	}
	outcomeEvidenceGHAPIGetArray = func(_ context.Context, endpoint, repo string) ([]map[string]any, error) {
		assert.Equal(t, "org/repo", repo)
		requests = append(requests, endpoint)
		return nil, nil
	}
	report, err := collectRunOutcomes(context.Background(), ProcessedRun{Run: WorkflowRun{
		DatabaseID: 7, Repository: "org/repo", LogsPath: dir,
	}}, nil, false)
	require.NoError(t, err)
	require.Len(t, report.Items, 1)
	assert.Equal(t, OutcomeStatusUnknown, report.Items[0].OutcomeStatus)
	assert.NotEqual(t, "missing_reference", report.Items[0].Signal)
	assert.Equal(t, []string{"issues/42", "issues/42/timeline"}, requests)
}

func TestLogsOutcomePersistenceErrorsAndCancellation(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "file")
	require.NoError(t, os.WriteFile(path, nil, 0o600))
	require.ErrorContains(t, writeOutcomeJSONL(path, 42, nil), "create outcomes directory")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := collectLogsOutcomes(ctx, []ProcessedRun{{Run: WorkflowRun{DatabaseID: 42}}}, "", false)
	require.ErrorIs(t, err, context.Canceled)
}

func TestLogsOutcomeReportAndEpisodeJoin(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	usageDir := filepath.Join(dir, "run-1", "usage")
	require.NoError(t, os.MkdirAll(usageDir, 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(usageDir, "agent_usage.jsonl"), []byte("{\"aic\":2.5}\n"), 0o600))
	run := ProcessedRun{
		Run:       WorkflowRun{DatabaseID: 1, Repository: "org/repo", WorkflowName: "producer", LogsPath: filepath.Dir(usageDir)},
		AwContext: &AwContext{EpisodeID: "ep-1"},
	}
	item := episodeObservation(1, "item-1", OutcomeStatusAccepted).OutcomeReport
	reports := []OutcomesData{{RunID: 1, Workflow: "producer", Items: []OutcomeReport{item}, Summary: ComputeOutcomeSummary([]OutcomeReport{item}, nil)}}
	data, err := prepareLogsData([]ProcessedRun{run}, renderLogsOutputOptions{outputDir: dir, summaryFile: "summary.json", outcomeReports: reports})
	require.NoError(t, err)
	require.Len(t, data.Outcomes, 1)
	require.NotNil(t, data.EpisodeOutcomes)
	require.Len(t, data.EpisodeOutcomes.Episodes, 1)
	assert.Equal(t, 1, data.EpisodeOutcomes.Episodes[0].Summary.Accepted)
	assert.InDelta(t, 2.5, *data.EpisodeOutcomes.Episodes[0].ObservedAIC, 1e-9)

	for _, verbose := range []bool{false, true} {
		var output bytes.Buffer
		require.NoError(t, renderLogsJSONToWriter(&output, data, verbose))
		var decoded LogsData
		require.NoError(t, json.Unmarshal(output.Bytes(), &decoded))
		assert.Equal(t, data.Outcomes, decoded.Outcomes)
		require.NotNil(t, decoded.EpisodeOutcomes)
		assert.Equal(t, 1, decoded.EpisodeOutcomes.Episodes[0].Summary.Accepted)
	}
	var rendered bytes.Buffer
	require.NoError(t, renderLogsOutcomes(&rendered, data))
	assert.Contains(t, rendered.String(), "1 accepted")
	saved, err := os.ReadFile(filepath.Join(dir, "summary.json"))
	require.NoError(t, err)
	assert.Contains(t, string(saved), "\"outcomes\"")
	assert.Contains(t, string(saved), "\"episode_outcomes\"")
	without, err := prepareLogsData([]ProcessedRun{run}, renderLogsOutputOptions{outputDir: dir})
	require.NoError(t, err)
	assert.Nil(t, without.Outcomes)
	assert.Nil(t, without.EpisodeOutcomes)

	reportPath := filepath.Join(dir, "report.md")
	require.NoError(t, renderLogsOutput([]ProcessedRun{run}, renderLogsOutputOptions{
		outputDir: dir, format: "markdown", reportFile: reportPath, outcomeReports: reports,
	}))
	markdown, err := os.ReadFile(reportPath)
	require.NoError(t, err)
	assert.Contains(t, string(markdown), "Outcomes for producer")
	assert.Contains(t, string(markdown), "1 accepted")
}

func TestPrettyLogsOutcomeSectionUsesStderr(t *testing.T) {
	oldStdout, oldStderr := os.Stdout, os.Stderr
	stdoutReader, stdoutWriter, err := os.Pipe()
	require.NoError(t, err)
	stderrReader, stderrWriter, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout, os.Stderr = stdoutWriter, stderrWriter
	renderErr := renderLogsOutput(nil, renderLogsOutputOptions{
		format: "pretty",
		outcomeReports: []OutcomesData{{
			RunID: 1, Workflow: "workflow", Items: []OutcomeReport{},
		}},
	})
	_ = stdoutWriter.Close()
	_ = stderrWriter.Close()
	os.Stdout, os.Stderr = oldStdout, oldStderr
	stdout, stdoutErr := io.ReadAll(stdoutReader)
	stderr, stderrErr := io.ReadAll(stderrReader)
	_ = stdoutReader.Close()
	_ = stderrReader.Close()
	require.NoError(t, renderErr)
	require.NoError(t, stdoutErr)
	require.NoError(t, stderrErr)
	assert.Empty(t, stdout)
	assert.Contains(t, string(stderr), "Outcomes for workflow (run 1)")
}

func TestEpisodeOutcomesCollectManifestsOnlyForMembers(t *testing.T) {
	fixture := newEpisodeRetrievalFixture(t)
	fixture.add("org/one", 2, "canonical", "", 0)
	fixture.add("org/one", 3, "unrelated", "", 0)
	var manifestArchive bytes.Buffer
	writer := zip.NewWriter(&manifestArchive)
	file, err := writer.Create(safeOutputItemsManifestFilename)
	require.NoError(t, err)
	_, err = fmt.Fprintln(file, `{"type":"assign_to_agent","success":true}`)
	require.NoError(t, err)
	require.NoError(t, writer.Close())
	var fallbackArchive bytes.Buffer
	writer = zip.NewWriter(&fallbackArchive)
	file, err = writer.Create("safeoutputs.jsonl")
	require.NoError(t, err)
	_, err = fmt.Fprintln(file, `{"type":"assign_to_agent","issue_number":42}`)
	require.NoError(t, err)
	require.NoError(t, writer.Close())
	var downloads []string
	request := func(ctx context.Context, host, endpoint string) ([]byte, error) {
		for _, id := range []int64{2, 3} {
			if endpoint == fmt.Sprintf("repos/org/one/actions/runs/%d/artifacts?per_page=100&page=1", id) {
				return json.Marshal(map[string]any{"artifacts": []episodeArtifact{
					{ID: id * 10, Name: "usage"},
					{ID: id*10 + 1, Name: "safe-outputs-items"},
					{ID: id*10 + 2, Name: "agent-output-fallback"},
				}})
			}
		}
		if strings.HasSuffix(endpoint, "/21/zip") || strings.HasSuffix(endpoint, "/31/zip") {
			downloads = append(downloads, endpoint)
			return manifestArchive.Bytes(), nil
		}
		if strings.HasSuffix(endpoint, "/22/zip") || strings.HasSuffix(endpoint, "/32/zip") {
			downloads = append(downloads, endpoint)
			return fallbackArchive.Bytes(), nil
		}
		return fixture.request(ctx, host, endpoint)
	}
	options := episodeRetrievalTestOptions(t)
	options.Outcomes = true
	data, err := retrieveCachedEpisodeLogs(context.Background(), options, request)
	require.NoError(t, err)
	require.Len(t, data.Runs, 1)
	assert.ElementsMatch(t, []string{"repos/org/one/actions/artifacts/21/zip", "repos/org/one/actions/artifacts/22/zip"}, downloads)
	run := ProcessedRun{Run: WorkflowRun{DatabaseID: 2, Repository: "org/one", LogsPath: data.Runs[0].LogsPath}}
	items, found, err := readLogsOutcomeItems(run)
	require.NoError(t, err)
	assert.True(t, found)
	require.Len(t, items, 1)
	assert.Equal(t, "assign_to_agent", items[0].Type)
	artifactDirs, verified, err := readEpisodeArtifactDirs(run.Run.LogsPath)
	require.NoError(t, err)
	require.True(t, verified)
	require.Len(t, artifactDirs, 3)
	enrichmentDirs := logsOutcomeAgentOutputDirs(run.Run.LogsPath)
	require.Len(t, enrichmentDirs, 1)
	assert.FileExists(t, filepath.Join(enrichmentDirs[0], "safeoutputs.jsonl"))
	items = enrichItemsFromAgentOutput(items, run.Run.LogsPath, run.Run.Repository, enrichmentDirs...)
	require.Len(t, items, 1)
	assert.Equal(t, 42, items[0].Number)
	oldGet, oldArray := outcomeEvidenceGHAPIGet, outcomeEvidenceGHAPIGetArray
	t.Cleanup(func() {
		outcomeEvidenceGHAPIGet, outcomeEvidenceGHAPIGetArray = oldGet, oldArray
	})
	outcomeEvidenceGHAPIGet = func(_ context.Context, endpoint, repo string) (map[string]any, error) {
		assert.Equal(t, "org/one", repo)
		assert.Equal(t, "issues/42", endpoint)
		return map[string]any{"state": "open"}, nil
	}
	outcomeEvidenceGHAPIGetArray = func(_ context.Context, endpoint, repo string) ([]map[string]any, error) {
		assert.Equal(t, "org/one", repo)
		assert.Equal(t, "issues/42/timeline", endpoint)
		return nil, nil
	}
	report, err := collectRunOutcomes(context.Background(), run, nil, false)
	require.NoError(t, err)
	require.Len(t, report.Items, 1)
	assert.NotEqual(t, "missing_reference", report.Items[0].Signal)
	cached, err := retrieveCachedEpisodeLogs(context.Background(), options, func(context.Context, string, string) ([]byte, error) {
		t.Fatal("cached membership must not make API requests")
		return nil, nil
	})
	require.NoError(t, err)
	assert.True(t, cached.EpisodeRetrieval.Cached)
}

func TestEpisodeOutcomesRejectUnverifiedAttemptManifest(t *testing.T) {
	options := episodeRetrievalTestOptions(t)
	fixture := newEpisodeRetrievalFixture(t)
	fixture.add("org/one", 2, "canonical", "", 0)
	collector := &episodeCollector{opts: options, request: fixture.request}
	require.NoError(t, collector.initializeScope())
	run := fixture.runs["org/one/2"].workflowRun()
	run.Attempt = 2
	fixture.archives[20] = fixture.archiveWithAttempt(2, 2, "attempt-2", 0.25, "")
	infos, dirs, err := collector.loadCompactArtifacts(context.Background(), run, []episodeArtifact{
		{ID: 20, Name: "usage"},
		{ID: 21, Name: "safe-outputs-items"},
	}, options.OutputDir)
	require.NoError(t, err)
	require.Len(t, infos, 1)
	require.Len(t, dirs, 1)
	require.Len(t, collector.report.Issues, 1)
	assert.Equal(t, "outcome_attempt_unverified", collector.report.Issues[0].Reason)
	for _, call := range fixture.calls {
		assert.NotContains(t, call, "/21/zip")
	}
}
