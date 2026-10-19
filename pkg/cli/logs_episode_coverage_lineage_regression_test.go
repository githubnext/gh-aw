//go:build !integration

package cli

import (
	"bytes"
	"encoding/json"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEpisodeCoverageUnlinkedSelectedRoot(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		context    *AwContext
		repository string
		reason     string
	}{
		{name: "root without inbound context", repository: "org/repo", reason: "unverified_root_lineage"},
		{name: "root without canonical identity", context: &AwContext{}, repository: "org/repo", reason: "unverified_root_lineage"},
		{name: "root records another canonical episode", context: &AwContext{EpisodeID: "other"}, repository: "org/repo", reason: "conflicting_root_lineage"},
		{name: "unlinked root repository mismatch", repository: "org/other", reason: "conflicting_root_lineage"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			allRuns := episodeCoverageTestRuns()
			allRuns[0].AwContext = tt.context
			allRuns[0].Repository = tt.repository
			episodes, edges := buildEpisodeData([]RunData{allRuns[0], allRuns[2]}, nil)
			assert.Empty(t, edges, "an unlinked selected root cannot supply the missing caller edge")
			var episode *EpisodeData
			for i := range episodes {
				if episodes[i].EpisodeID == "canonical" {
					episode = &episodes[i]
				}
			}
			require.NotNil(t, episode)
			assert.Equal(t, []int64{300}, episode.RunIDs, "root claims must not infer episode membership")
			assert.Equal(t, int64(300), episode.RepresentativeRunID)
			assert.Equal(t, "high", episode.Confidence)
			assert.Equal(t, "partial", episode.Coverage.Status)
			assert.ElementsMatch(t, []string{"missing_recorded_parent", tt.reason}, episode.Coverage.Reasons)
			assert.Equal(t, []int64{200}, episode.Coverage.MissingParentRunIDs)
			require.Len(t, episode.RecordedRoots, 1)
			assert.Equal(t, EpisodeRoot{
				RunID: "100", WorkflowID: allRuns[2].AwContext.RootWorkflowID, Repository: "org/repo",
			}, episode.RecordedRoots[0])
			if tt.context == nil || tt.context.EpisodeID == "" {
				require.Len(t, episodes, 1, "a run without canonical context must not form an episode")
			}
		})
	}
}

func TestEpisodeCoverageSameRunParentRepository(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name       string
		repository string
		status     string
		reasons    []string
	}{
		{name: "same repository", repository: "org/repo", status: "unknown"},
		{name: "case-insensitive identity", repository: "Org/Repo", status: "unknown"},
		{name: "different repository", repository: "org/other", status: "partial", reasons: []string{"conflicting_parent_lineage"}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			run := episodeCoverageTestRuns()[0]
			run.AwContext = &AwContext{
				EpisodeID: "canonical", RunID: "100", Repo: tt.repository,
				HopID: "100-1:reusable", ParentHopID: "100-1:caller",
			}
			episodes, edges := buildEpisodeData([]RunData{run}, nil)
			require.Len(t, episodes, 1)
			assert.Empty(t, edges, "same-run reusable hops must not create a self-edge")
			assert.Empty(t, episodes[0].RecordedRoots)
			assert.Equal(t, "high", episodes[0].Confidence)
			assert.Equal(t, tt.status, episodes[0].Coverage.Status)
			assert.Equal(t, tt.reasons, episodes[0].Coverage.Reasons)
			var out bytes.Buffer
			require.NoError(t, renderLogsJSONToWriter(&out, LogsData{Episodes: episodes, Edges: edges}, false))
			var rendered LogsData
			require.NoError(t, json.Unmarshal(out.Bytes(), &rendered))
			assert.Equal(t, episodes, rendered.Episodes, "compact JSON must preserve parent evidence without root claims")
		})
	}
}

func TestEpisodeCoverageRepositoryCase(t *testing.T) {
	t.Parallel()
	runs := episodeCoverageTestRuns()
	runs[1].AwContext.Repo, runs[1].AwContext.RootRepo = "Org/Repo", "Org/Repo"
	runs[2].AwContext.Repo, runs[2].AwContext.RootRepo = "ORG/REPO", "ORG/REPO"
	episodes, edges := buildEpisodeData(runs, nil)
	require.Len(t, episodes, 1)
	require.Len(t, edges, 2, "recorded parent edges must use the same case-insensitive repository check")
	assert.Equal(t, "unknown", episodes[0].Coverage.Status)
	assert.Empty(t, episodes[0].Coverage.Reasons)
	require.Len(t, episodes[0].RecordedRoots, 3, "retain the exact spelling of each recorded claim")
	var repositories []string
	for _, root := range episodes[0].RecordedRoots {
		assert.True(t, root.Observed)
		repositories = append(repositories, root.Repository)
	}
	assert.ElementsMatch(t, []string{"org/repo", "Org/Repo", "ORG/REPO"}, repositories)
	slices.Reverse(runs)
	reversedEpisodes, reversedEdges := buildEpisodeData(runs, nil)
	assert.Equal(t, episodes, reversedEpisodes)
	assert.Equal(t, edges, reversedEdges)
}

func TestEpisodeCoveragePartialRootClaims(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name   string
		roots  []EpisodeRoot
		status string
	}{
		{name: "workflow only", roots: []EpisodeRoot{{WorkflowID: "root"}}, status: "unknown"},
		{name: "repository only", roots: []EpisodeRoot{{Repository: "org/repo"}}, status: "unknown"},
		{name: "workflow and repository", roots: []EpisodeRoot{{WorkflowID: "root", Repository: "org/repo"}}, status: "unknown"},
		{name: "complementary partial claims", roots: []EpisodeRoot{{WorkflowID: "root"}, {Repository: "org/repo"}}, status: "unknown"},
		{name: "conflicting workflow claims", roots: []EpisodeRoot{{WorkflowID: "root"}, {WorkflowID: "other"}}, status: "partial"},
		{name: "conflicting repository claims", roots: []EpisodeRoot{{Repository: "org/repo"}, {Repository: "org/other"}}, status: "partial"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			runs := episodeCoverageTestRuns()[:len(tt.roots)]
			for i, root := range tt.roots {
				runs[i].AwContext.RootRunID = root.RunID
				runs[i].AwContext.RootWorkflowID = root.WorkflowID
				runs[i].AwContext.RootRepo = root.Repository
			}
			episodes, _ := buildEpisodeData(runs, nil)
			require.Len(t, episodes, 1)
			assert.ElementsMatch(t, tt.roots, episodes[0].RecordedRoots, "partial root identity must be preserved verbatim")
			assert.Equal(t, tt.status, episodes[0].Coverage.Status)
			if tt.status == "partial" {
				assert.Equal(t, []string{"conflicting_recorded_roots"}, episodes[0].Coverage.Reasons)
			} else {
				assert.Empty(t, episodes[0].Coverage.Reasons, "a missing root run ID is not an invalid run ID")
			}
			var out bytes.Buffer
			require.NoError(t, renderLogsJSONToWriter(&out, LogsData{Episodes: episodes}, false))
			var rendered LogsData
			require.NoError(t, json.Unmarshal(out.Bytes(), &rendered))
			require.Len(t, rendered.Episodes, 1)
			assert.ElementsMatch(t, tt.roots, rendered.Episodes[0].RecordedRoots)
		})
	}
}
