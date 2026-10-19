//go:build !integration

package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func collectionCoverageProcessedRun(id int64, episode, repository, name, workflowPath string) ProcessedRun {
	return processedRunFromCachedData(RunData{
		RunID: id, Repository: repository, WorkflowName: name, WorkflowPath: workflowPath,
		AwContext: &AwContext{EpisodeID: episode},
	}, nil, ".")
}

func assertCollectionCoverage(t *testing.T, data LogsData, expected map[string]bool) {
	t.Helper()
	require.Len(t, data.Episodes, len(expected))
	for _, episode := range data.Episodes {
		truncated, present := expected[episode.EpisodeID]
		require.True(t, present, "unexpected episode %q", episode.EpisodeID)
		assert.Equal(t, truncated, episode.Coverage.CollectionTruncated, episode.EpisodeID)
		assert.Equal(t, "selected_runs", episode.Coverage.Scope)
		assert.Equal(t, "high", episode.Confidence, "coverage must not change confidence")
		if truncated {
			assert.Equal(t, "partial", episode.Coverage.Status, episode.EpisodeID)
			assert.Contains(t, episode.Coverage.Reasons, "collection_truncated")
		} else {
			assert.Equal(t, "unknown", episode.Coverage.Status, episode.EpisodeID)
			assert.NotContains(t, episode.Coverage.Reasons, "collection_truncated")
		}
	}
}

func TestLogsCollectionCoverageMergedTargetScope(t *testing.T) {
	// Remote command resolution puts the display name, not the file ID, into
	// WorkflowName. Exercise the actual continuation construction and merge.
	target := logsWorkflowTarget{
		repoOverride: "org/partial",
		workflowName: matchRemoteWorkflowName("daily-report", map[string]*GitHubWorkflow{
			"daily-report": {Name: "Daily Report", Path: ".github/workflows/daily-report.lock.yml"},
		}),
	}
	require.Equal(t, "Daily Report", target.workflowName)
	cursor := buildContinuationIfNeeded(nil, true, false, false, false, logsTargetContinuationOptions(LogsDownloadOptions{
		WorkflowName: target.workflowName, RepoOverride: target.repoOverride,
	}))
	_, continuations, _, countLimited, _, errs := mergeLogsTargetResults([]logsTargetResult{{
		target: target, result: workflowLogsResult{continuation: cursor},
	}}, nil)
	require.Empty(t, errs)
	runs := []ProcessedRun{
		collectionCoverageProcessedRun(1, "partial", "org/partial", "Daily Report", ".github/workflows/daily-report.lock.yml"),
		collectionCoverageProcessedRun(2, "other-workflow", "org/partial", "Other Report", ".github/workflows/other.lock.yml"),
		collectionCoverageProcessedRun(3, "other-repo", "org/other", "Daily Report", ".github/workflows/daily-report.lock.yml"),
		collectionCoverageProcessedRun(4, "cross-repo", "org/other", "Root", ".github/workflows/root.lock.yml"),
		collectionCoverageProcessedRun(5, "cross-repo", "org/partial", "Daily Report", ".github/workflows/daily-report.lock.yml"),
		collectionCoverageProcessedRun(6, "no-matching-pair", "org/partial", "Root", ".github/workflows/root.lock.yml"),
		collectionCoverageProcessedRun(7, "no-matching-pair", "org/other", "Daily Report", ".github/workflows/daily-report.lock.yml"),
	}
	data, err := prepareLogsData(runs, renderLogsOutputOptions{
		outputDir: ".", continuations: continuations, countLimitReached: countLimited,
	})
	require.NoError(t, err)
	assert.Equal(t, continuations, data.Continuations)
	assertCollectionCoverage(t, data, map[string]bool{
		"partial": true, "other-workflow": false, "other-repo": false,
		"cross-repo": true, "no-matching-pair": false,
	})
}

func TestLogsCollectionCoverageWorkflowAndRepositoryInputs(t *testing.T) {
	for _, tt := range []struct {
		name       string
		target     WorkflowContinuation
		repository string
		url        string
		want       bool
	}{
		{name: "display name", target: WorkflowContinuation{Repository: "org/repo", ContinuationData: ContinuationData{WorkflowName: "Daily Report"}}, repository: "org/repo", want: true},
		{name: "file ID", target: WorkflowContinuation{Repository: "org/repo", ContinuationData: ContinuationData{WorkflowName: "daily-report"}}, repository: "org/repo", want: true},
		{name: "lock filename", target: WorkflowContinuation{Repository: "org/repo", ContinuationData: ContinuationData{WorkflowName: "daily-report.lock.yml"}}, repository: "org/repo", want: true},
		{name: "workflow path", target: WorkflowContinuation{Repository: "org/repo", ContinuationData: ContinuationData{WorkflowName: ".github/workflows/daily-report.lock.yml"}}, repository: "org/repo", want: true},
		{name: "repository casing", target: WorkflowContinuation{Repository: "ORG/Repo"}, repository: "org/repo", want: true},
		{name: "repository-only target", target: WorkflowContinuation{Repository: "org/repo"}, repository: "org/repo", want: true},
		{name: "unrelated repository", target: WorkflowContinuation{Repository: "org/other"}, repository: "org/repo"},
		{name: "unrelated workflow", target: WorkflowContinuation{Repository: "org/repo", ContinuationData: ContinuationData{WorkflowName: "Other Report"}}, repository: "org/repo"},
		{name: "host-qualified target", target: WorkflowContinuation{Repository: "github.example.com/org/repo"}, repository: "org/repo", url: "https://github.example.com/org/repo/actions/runs/1", want: true},
		{name: "different host", target: WorkflowContinuation{Repository: "github.example.com/org/repo"}, repository: "org/repo", url: "https://github.com/org/repo/actions/runs/1"},
		{name: "unknown repository remains conservative", target: WorkflowContinuation{Repository: "org/repo"}, want: true},
		{name: "repository recovered from URL", target: WorkflowContinuation{Repository: "org/repo"}, url: "https://github.com/org/repo/actions/runs/1", want: true},
		{name: "URL rules out unrelated repository", target: WorkflowContinuation{Repository: "org/other"}, url: "https://github.com/org/repo/actions/runs/1"},
		{name: "omitted repository scope remains conservative", target: WorkflowContinuation{}, repository: "org/repo", want: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			run := collectionCoverageProcessedRun(1, "episode", tt.repository, "Daily Report", ".github/workflows/daily-report.lock.yml")
			run.cachedData.URL = tt.url
			data, err := prepareLogsData([]ProcessedRun{run}, renderLogsOutputOptions{outputDir: ".", continuations: []WorkflowContinuation{tt.target}})
			require.NoError(t, err)
			assertCollectionCoverage(t, data, map[string]bool{"episode": tt.want})
		})
	}
}

func TestLogsCollectionCoverageDirectBuilderAndPrepareAgree(t *testing.T) {
	run := collectionCoverageProcessedRun(1, "episode", "org/repo", "Daily Report", ".github/workflows/daily-report.lock.yml")
	for _, cursor := range []*ContinuationData{
		nil,
		{Count: 10, StartDate: "-30d"},
		{WorkflowName: "Daily Report"},
		{WorkflowName: "daily-report.lock.yml"},
	} {
		direct := buildLogsData([]ProcessedRun{run}, ".", cursor)
		prepared, err := prepareLogsData([]ProcessedRun{run}, renderLogsOutputOptions{outputDir: ".", continuation: cursor})
		require.NoError(t, err)
		assert.Equal(t, direct.Episodes, prepared.Episodes)
		assert.Equal(t, cursor, direct.Continuation)
		assertCollectionCoverage(t, direct, map[string]bool{"episode": cursor != nil})
	}
}

func TestLogsCollectionCoverageMissingWorkflowIdentityRemainsConservative(t *testing.T) {
	run := collectionCoverageProcessedRun(1, "episode", "org/repo", "", "")
	data, err := prepareLogsData([]ProcessedRun{run}, renderLogsOutputOptions{
		outputDir: ".",
		continuations: []WorkflowContinuation{{
			Repository: "org/repo", ContinuationData: ContinuationData{WorkflowName: "Daily Report"},
		}},
	})
	require.NoError(t, err)
	assertCollectionCoverage(t, data, map[string]bool{"episode": true})
}

func TestLogsCollectionCoverageScopedCountLimitsAndCursorlessTargets(t *testing.T) {
	for _, withCursor := range []bool{false, true} {
		results := []logsTargetResult{{
			target: logsWorkflowTarget{workflowName: "Daily Report", repoOverride: "org/limited"},
			result: workflowLogsResult{countLimitReached: true},
		}, {
			target: logsWorkflowTarget{workflowName: "other", repoOverride: "org/cursorless"},
			result: workflowLogsResult{countLimitReached: true},
		}}
		if withCursor {
			results[0].result.continuation = &ContinuationData{WorkflowName: "Daily Report"}
		}
		_, continuations, _, countLimited, _, errs := mergeLogsTargetResults(results, nil)
		require.Empty(t, errs)
		require.True(t, countLimited)
		runs := []ProcessedRun{
			collectionCoverageProcessedRun(1, "limited", "org/limited", "Daily Report", ""),
			collectionCoverageProcessedRun(2, "cursorless", "org/cursorless", "other", ""),
			collectionCoverageProcessedRun(3, "other-repo", "org/other", "Daily Report", ""),
			collectionCoverageProcessedRun(4, "other-workflow", "org/limited", "Other Report", ""),
		}
		data, err := prepareLogsData(runs, renderLogsOutputOptions{
			outputDir: ".", continuations: continuations, countLimitReached: countLimited,
			countLimitTargets: logsCountLimitedTargets(results),
		})
		require.NoError(t, err)
		assertCollectionCoverage(t, data, map[string]bool{
			"limited": true, "cursorless": true, "other-repo": false, "other-workflow": false,
		})
	}
}

func TestLogsCollectionCoverageMixedLocalRemoteSameWorkflow(t *testing.T) {
	for _, tt := range []struct {
		name              string
		localRepository   string
		countLimited      bool
		cursor            bool
		repositoryFromURL bool
	}{
		{name: "timeout cursor", localRepository: "org/local", cursor: true},
		{name: "count cursor", localRepository: "org/local", countLimited: true, cursor: true},
		{name: "cursorless count bound", localRepository: "org/local", countLimited: true},
		{name: "local repository from collected member", countLimited: true},
		{name: "local repository from member URL", countLimited: true, repositoryFromURL: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			local := collectionCoverageProcessedRun(1, "local", "org/local", "Daily Report", "")
			if tt.repositoryFromURL {
				local.Run.Repository = ""
				local.cachedData.Repository = ""
				local.Run.URL = "https://github.com/org/local/actions/runs/1"
				local.cachedData.URL = local.Run.URL
			}
			results := []logsTargetResult{{
				target: logsWorkflowTarget{workflowName: "Daily Report"},
				result: workflowLogsResult{processedRuns: []ProcessedRun{local}, countLimitReached: tt.countLimited},
			}, {
				target: logsWorkflowTarget{workflowName: "Daily Report", repoOverride: "org/remote"},
				result: workflowLogsResult{processedRuns: []ProcessedRun{
					collectionCoverageProcessedRun(2, "remote", "org/remote", "Daily Report", ""),
				}},
			}}
			if tt.cursor {
				results[0].result.continuation = &ContinuationData{WorkflowName: "Daily Report"}
			}
			resolveLogsCollectionTargetRepositories(results, tt.localRepository)
			require.Equal(t, "org/local", results[0].target.repoOverride)
			runs, continuations, _, countLimited, _, errs := mergeLogsTargetResults(results, nil)
			require.Empty(t, errs)
			for _, continuation := range continuations {
				assert.Equal(t, "org/local", continuation.Repository)
			}
			data, err := prepareLogsData(runs, renderLogsOutputOptions{
				outputDir: ".", continuations: continuations, countLimitReached: countLimited,
				countLimitTargets: logsCountLimitedTargets(results),
			})
			require.NoError(t, err)
			assertCollectionCoverage(t, data, map[string]bool{"local": true, "remote": false})
		})
	}
}

func TestLogsCollectionCoverageLocalTargetWithoutProgress(t *testing.T) {
	for _, localRepository := range []string{"org/local", ""} {
		results := []logsTargetResult{{
			target: logsWorkflowTarget{workflowName: "Daily Report"},
			result: workflowLogsResult{countLimitReached: true},
		}, {
			target: logsWorkflowTarget{workflowName: "Daily Report", repoOverride: "org/remote"},
			result: workflowLogsResult{processedRuns: []ProcessedRun{
				collectionCoverageProcessedRun(2, "remote", "org/remote", "Daily Report", ""),
			}},
		}}
		resolveLogsCollectionTargetRepositories(results, localRepository)
		assert.Equal(t, localRepository, results[0].target.repoOverride, "never infer local scope from remote results")
		runs, continuations, _, countLimited, _, errs := mergeLogsTargetResults(results, nil)
		require.Empty(t, errs)
		data, err := prepareLogsData(runs, renderLogsOutputOptions{
			outputDir: ".", continuations: continuations, countLimitReached: countLimited,
			countLimitTargets: logsCountLimitedTargets(results),
		})
		require.NoError(t, err)
		assertCollectionCoverage(t, data, map[string]bool{"remote": localRepository == ""})
	}
}

func TestLogsCollectionCoverageUnscopedCountLimitRemainsConservative(t *testing.T) {
	runs := []ProcessedRun{
		collectionCoverageProcessedRun(1, "first", "org/first", "first", ""),
		collectionCoverageProcessedRun(2, "second", "org/second", "second", ""),
	}
	for _, continuations := range [][]WorkflowContinuation{
		nil,
		{{Repository: "org/first", ContinuationData: ContinuationData{WorkflowName: "first"}}},
	} {
		data, err := prepareLogsData(runs, renderLogsOutputOptions{
			outputDir: ".", countLimitReached: true, continuations: continuations,
		})
		require.NoError(t, err)
		assertCollectionCoverage(t, data, map[string]bool{"first": true, "second": true})
	}
}

func TestLogsCollectionCoveragePreservesLineageGaps(t *testing.T) {
	run := collectionCoverageProcessedRun(1, "episode", "org/repo", "Daily Report", "")
	run.cachedData.AwContext.RunID = "99"
	data := buildLogsData([]ProcessedRun{run}, ".", &ContinuationData{})
	require.Len(t, data.Episodes, 1)
	assert.Equal(t, []string{"collection_truncated", "missing_recorded_parent"}, data.Episodes[0].Coverage.Reasons)
	assert.Equal(t, []int64{99}, data.Episodes[0].Coverage.MissingParentRunIDs)
	assertCollectionCoverage(t, data, map[string]bool{"episode": true})
}
