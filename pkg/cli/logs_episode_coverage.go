package cli

import (
	"cmp"
	"slices"
	"strconv"
)

// EpisodeRoot preserves a recorded canonical root claim without substituting a
// selected descendant. Observed means its run is included in this episode.
type EpisodeRoot struct {
	RunID      string `json:"run_id,omitempty"`
	WorkflowID string `json:"workflow_id,omitempty"`
	Repository string `json:"repository,omitempty"`
	Observed   bool   `json:"observed"`
}

// EpisodeCoverage describes evidence available in selected runs, not a census
// of all episode members. Absence of a known gap cannot establish completeness.
type EpisodeCoverage struct {
	Scope               string   `json:"scope"`
	Status              string   `json:"status"`
	Reasons             []string `json:"reasons,omitempty"`
	MissingParentRunIDs []int64  `json:"missing_parent_run_ids,omitempty"`
	CollectionTruncated bool     `json:"collection_truncated"`
}

func addEpisodeCoverage(episodes []EpisodeData, runsByID map[int64]RunData) {
	for i := range episodes {
		episode := &episodes[i]
		episode.Coverage = EpisodeCoverage{Scope: "selected_runs", Status: "unknown"}
		roots := make(map[EpisodeRoot]struct{})
		rootRuns, rootWorkflows, rootRepos := map[string]struct{}{}, map[string]struct{}{}, map[string]struct{}{}
		for _, runID := range episode.RunIDs {
			run := runsByID[runID]
			ctx := run.AwContext
			if ctx == nil {
				continue
			}
			root := recordedEpisodeRoot(ctx, episode, runsByID)
			if root.RunID != "" || root.WorkflowID != "" || root.Repository != "" {
				if root.RunID != "" {
					rootRuns[root.RunID] = struct{}{}
				}
				if root.WorkflowID != "" {
					rootWorkflows[root.WorkflowID] = struct{}{}
				}
				if root.Repository != "" {
					rootRepos[root.Repository] = struct{}{}
				}
				roots[root] = struct{}{}
			}
			addRecordedParentCoverage(run, episode, runsByID)
		}
		if len(rootRuns) > 1 || len(rootWorkflows) > 1 || len(rootRepos) > 1 {
			episode.Coverage.addGap("conflicting_recorded_roots")
		}
		for root := range roots {
			episode.RecordedRoots = append(episode.RecordedRoots, root)
		}
		slices.SortFunc(episode.RecordedRoots, func(a, b EpisodeRoot) int {
			if a.RunID != b.RunID {
				return cmp.Compare(a.RunID, b.RunID)
			}
			if a.Repository != b.Repository {
				return cmp.Compare(a.Repository, b.Repository)
			}
			return cmp.Compare(a.WorkflowID, b.WorkflowID)
		})
		slices.Sort(episode.Coverage.MissingParentRunIDs)
		episode.Coverage.MissingParentRunIDs = slices.Compact(episode.Coverage.MissingParentRunIDs)
	}
}

func recordedEpisodeRoot(ctx *AwContext, episode *EpisodeData, runsByID map[int64]RunData) EpisodeRoot {
	root := EpisodeRoot{RunID: ctx.RootRunID, WorkflowID: ctx.RootWorkflowID, Repository: ctx.RootRepo}
	if root.RunID == "" {
		return root
	}
	rootID, valid := parseEpisodeRunID(root.RunID)
	source, present := runsByID[rootID]
	switch {
	case !valid:
		episode.Coverage.addGap("invalid_recorded_root_run_id")
	case !present:
		episode.Coverage.addGap("missing_recorded_root")
	case !slices.Contains(episode.RunIDs, rootID) || episodeRepositoryConflict(root.Repository, source.Repository):
		episode.Coverage.addGap("conflicting_root_lineage")
	default:
		root.Observed = true
	}
	return root
}

func addRecordedParentCoverage(run RunData, episode *EpisodeData, runsByID map[int64]RunData) {
	ctx := run.AwContext
	if ctx.RunID == "" {
		return
	}
	parentID, valid := parseEpisodeRunID(ctx.RunID)
	parent, present := runsByID[parentID]
	switch {
	case !valid:
		episode.Coverage.addGap("invalid_recorded_parent_run_id")
	case !present:
		episode.Coverage.addGap("missing_recorded_parent")
		episode.Coverage.MissingParentRunIDs = append(episode.Coverage.MissingParentRunIDs, parentID)
	case parentID == run.RunID:
		// Reusable workflow hops can share one GitHub run.
	case !slices.Contains(episode.RunIDs, parentID) || episodeRepositoryConflict(ctx.Repo, parent.Repository):
		episode.Coverage.addGap("conflicting_parent_lineage")
	}
}

func parseEpisodeRunID(id string) (int64, bool) {
	runID, err := strconv.ParseInt(id, 10, 64)
	return runID, err == nil && runID > 0
}

func episodeRepositoryConflict(recorded, observed string) bool {
	return recorded != "" && observed != "" && recorded != observed
}

func (coverage *EpisodeCoverage) addGap(reason string) {
	coverage.Status = "partial"
	if !slices.Contains(coverage.Reasons, reason) {
		coverage.Reasons = append(coverage.Reasons, reason)
		slices.Sort(coverage.Reasons)
	}
}

func markEpisodeCollectionTruncated(episodes []EpisodeData) {
	for i := range episodes {
		episodes[i].Coverage.CollectionTruncated = true
		episodes[i].Coverage.addGap("collection_truncated")
	}
}

func applyLogsCollectionCoverage(data *LogsData, continuations []WorkflowContinuation, countLimitReached bool) {
	data.Continuations = continuations
	if len(continuations) > 0 || countLimitReached {
		markEpisodeCollectionTruncated(data.Episodes)
	}
}
