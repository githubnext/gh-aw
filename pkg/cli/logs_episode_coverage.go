package cli

import (
	"cmp"
	"net/url"
	"slices"
	"strconv"
	"strings"
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
		rootRuns, rootWorkflows := map[string]struct{}{}, map[string]struct{}{}
		var rootRepos []string
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
					if !containsRepository(rootRepos, root.Repository) {
						rootRepos = append(rootRepos, root.Repository)
					}
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
	case episodeRepositoryConflict(root.Repository, source.Repository):
		episode.Coverage.addGap("conflicting_root_lineage")
	case canonicalEpisodeID(source) == "":
		episode.Coverage.addGap("unverified_root_lineage")
	case !slices.Contains(episode.RunIDs, rootID):
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
	case episodeRepositoryConflict(ctx.Repo, parent.Repository):
		episode.Coverage.addGap("conflicting_parent_lineage")
	case canonicalEpisodeID(parent) == "":
		episode.Coverage.addGap("unverified_parent_lineage")
	case !slices.Contains(episode.RunIDs, parentID):
		episode.Coverage.addGap("conflicting_parent_lineage")
	}
}

func parseEpisodeRunID(id string) (int64, bool) {
	runID, err := strconv.ParseInt(id, 10, 64)
	return runID, err == nil && runID > 0
}

func episodeRepositoryConflict(recorded, observed string) bool {
	return recorded != "" && observed != "" && !strings.EqualFold(recorded, observed)
}

func containsRepository(repositories []string, repository string) bool {
	for _, candidate := range repositories {
		if strings.EqualFold(candidate, repository) {
			return true
		}
	}
	return false
}

func (coverage *EpisodeCoverage) addGap(reason string) {
	coverage.Status = "partial"
	if !slices.Contains(coverage.Reasons, reason) {
		coverage.Reasons = append(coverage.Reasons, reason)
		slices.Sort(coverage.Reasons)
	}
}

type logsCollectionCoverageOptions struct {
	continuations     []WorkflowContinuation
	countLimitReached bool
	countLimitTargets []logsWorkflowTarget
	// Bounded retrieval must not resolve the local repository through an
	// unbudgeted gh repo view request, or classify remote runs using local YAML.
	disableLocalClassification bool
}

func resolveLogsCollectionTargetRepositories(results []logsTargetResult, localRepository string) {
	if localRepository == "" {
		for _, result := range results {
			if result.target.repoOverride != "" {
				continue
			}
			for _, run := range result.result.processedRuns {
				runURL, _ := url.Parse(run.Run.URL)
				if repository := collectionRunRepository(run.Run.Repository, runURL); repository != "" {
					localRepository = repository
					break
				}
			}
			if localRepository != "" {
				break
			}
		}
	}
	for i := range results {
		if results[i].target.repoOverride == "" {
			results[i].target.repoOverride = localRepository
		}
	}
}

func logsCountLimitedTargets(results []logsTargetResult) []logsWorkflowTarget {
	var targets []logsWorkflowTarget
	for _, result := range results {
		if result.result.countLimitReached {
			targets = append(targets, result.target)
		}
	}
	return targets
}

func applyLogsCollectionCoverage(data *LogsData, opts logsCollectionCoverageOptions) {
	data.Continuations = opts.continuations
	if data.Continuation == nil && len(opts.continuations) == 0 && !opts.countLimitReached {
		return
	}
	targets := slices.Clone(opts.continuations)
	if data.Continuation != nil {
		targets = append(targets, WorkflowContinuation{ContinuationData: *data.Continuation})
	}
	for _, target := range opts.countLimitTargets {
		targets = append(targets, WorkflowContinuation{
			Repository: target.repoOverride, ContinuationData: ContinuationData{WorkflowName: target.workflowName},
		})
	}
	// Callers may only know that some target hit a count limit. Combined
	// collection supplies every limited target, including those without cursors.
	unscopedCountLimit := opts.countLimitReached && len(opts.countLimitTargets) == 0 && data.Continuation == nil
	runsByEpisode := make(map[string][]RunData)
	for _, run := range data.Runs {
		if episodeID := canonicalEpisodeID(run); episodeID != "" {
			runsByEpisode[episodeID] = append(runsByEpisode[episodeID], run)
		}
	}
	for i := range data.Episodes {
		episode := &data.Episodes[i]
		truncated := unscopedCountLimit
		for _, target := range targets {
			if episodeMatchesCollectionTarget(*episode, runsByEpisode[episode.EpisodeID], target) {
				truncated = true
				break
			}
		}
		if truncated {
			episode.Coverage.CollectionTruncated = true
			episode.Coverage.addGap("collection_truncated")
		}
	}
}

func episodeMatchesCollectionTarget(episode EpisodeData, runs []RunData, target WorkflowContinuation) bool {
	if len(runs) == 0 {
		if len(episode.WorkflowNames) == 0 {
			return collectionTargetMatchesRun(target, RunData{Repository: episode.Repository})
		}
		for _, name := range episode.WorkflowNames {
			if collectionTargetMatchesRun(target, RunData{Repository: episode.Repository, WorkflowName: name}) {
				return true
			}
		}
	}
	for _, run := range runs {
		if collectionTargetMatchesRun(target, run) {
			return true
		}
	}
	return false
}

func collectionTargetMatchesRun(target WorkflowContinuation, run RunData) bool {
	repository := strings.TrimSpace(target.Repository)
	runURL, err := url.Parse(run.URL)
	if host := logsRateLimitHost(repository); host != "" {
		repository = strings.TrimPrefix(repository, host+"/")
		if err == nil && runURL.Host != "" && !strings.EqualFold(host, runURL.Host) {
			return false
		}
	}
	runRepository := collectionRunRepository(run.Repository, runURL)
	if repository != "" && runRepository != "" && !strings.EqualFold(repository, runRepository) {
		return false
	}
	workflow := strings.TrimSpace(target.WorkflowName)
	if workflow == "" || (run.WorkflowName == "" && run.WorkflowPath == "") {
		return true
	}
	// Target resolution normally yields the display name; API callers can also
	// supply a workflow ID or lock-file path.
	return strings.EqualFold(workflow, run.WorkflowName) ||
		workflowIDFromPath(workflow) == workflowIDFromRun(run.WorkflowPath, run.WorkflowName)
}

func collectionRunRepository(repository string, runURL *url.URL) string {
	if repository != "" || runURL == nil {
		return repository
	}
	if repo, _, found := strings.Cut(strings.TrimPrefix(runURL.Path, "/"), "/actions/runs/"); found {
		return repo
	}
	return ""
}
