package cli

import (
	"cmp"
	"fmt"
	"slices"
	"strconv"
	"time"

	"github.com/github/gh-aw/pkg/timeutil"
)

// EpisodeEdge represents a deterministic lineage edge between two workflow runs.
type EpisodeEdge struct {
	SourceRunID int64    `json:"source_run_id"`
	TargetRunID int64    `json:"target_run_id"`
	EdgeType    string   `json:"edge_type"`
	Confidence  string   `json:"confidence"`
	Reasons     []string `json:"reasons,omitempty"`
	SourceRepo  string   `json:"source_repo,omitempty"`
	SourceRef   string   `json:"source_ref,omitempty"`
	EventType   string   `json:"event_type,omitempty"`
	EpisodeID   string   `json:"episode_id,omitempty"`
}

// EpisodeData represents a deterministic episode rollup derived from workflow runs.
type EpisodeData struct {
	EpisodeID             string   `json:"episode_id"`
	Kind                  string   `json:"kind"`
	Confidence            string   `json:"confidence"`
	Reasons               []string `json:"reasons,omitempty"`
	RootRunID             int64    `json:"root_run_id,omitempty"`
	RunIDs                []int64  `json:"run_ids"`
	WorkflowNames         []string `json:"workflow_names"`
	TotalRuns             int      `json:"total_runs"`
	TotalTokens           int      `json:"total_tokens"`
	TotalEstimatedCost    float64  `json:"total_estimated_cost"`
	TotalDuration         string   `json:"total_duration"`
	RiskyNodeCount        int      `json:"risky_node_count"`
	WriteCapableNodeCount int      `json:"write_capable_node_count"`
	MissingToolCount      int      `json:"missing_tool_count"`
	MCPFailureCount       int      `json:"mcp_failure_count"`
	BlockedRequestCount   int      `json:"blocked_request_count"`
	RiskDistribution      string   `json:"risk_distribution"`
}

type episodeAccumulator struct {
	metadata EpisodeData
	duration time.Duration
	runSet   map[int64]bool
	nameSet  map[string]bool
	rootTime time.Time
}

type episodeSeed struct {
	EpisodeID  string
	Kind       string
	Confidence string
	Reasons    []string
}

func buildEpisodeData(runs []RunData, processedRuns []ProcessedRun) ([]EpisodeData, []EpisodeEdge) {
	runsByID := make(map[int64]RunData, len(runs))
	processedByID := make(map[int64]ProcessedRun, len(processedRuns))
	seedsByRunID := make(map[int64]episodeSeed, len(runs))
	parents := make(map[int64]int64, len(runs))
	for _, run := range runs {
		runsByID[run.DatabaseID] = run
		episodeID, kind, confidence, reasons := classifyEpisode(run)
		seedsByRunID[run.DatabaseID] = episodeSeed{EpisodeID: episodeID, Kind: kind, Confidence: confidence, Reasons: append([]string(nil), reasons...)}
		parents[run.DatabaseID] = run.DatabaseID
	}
	for _, processedRun := range processedRuns {
		processedByID[processedRun.Run.DatabaseID] = processedRun
	}

	edges := make([]EpisodeEdge, 0)
	for _, run := range runs {
		if edge, ok := buildEpisodeEdge(run, seedsByRunID[run.DatabaseID].EpisodeID, runsByID); ok {
			edges = append(edges, edge)
			unionEpisodes(parents, edge.SourceRunID, edge.TargetRunID)
		}
	}

	episodeMap := make(map[string]*episodeAccumulator)
	rootMetadata := make(map[int64]episodeSeed)
	for _, run := range runs {
		root := findEpisodeParent(parents, run.DatabaseID)
		seed := seedsByRunID[run.DatabaseID]
		best, exists := rootMetadata[root]
		if !exists || compareEpisodeSeeds(seed, best) > 0 {
			rootMetadata[root] = seed
		}
	}

	for _, run := range runs {
		root := findEpisodeParent(parents, run.DatabaseID)
		selectedSeed := rootMetadata[root]
		episodeID, kind, confidence, reasons := selectedSeed.EpisodeID, selectedSeed.Kind, selectedSeed.Confidence, selectedSeed.Reasons
		acc, exists := episodeMap[episodeID]
		if !exists {
			acc = &episodeAccumulator{
				metadata: EpisodeData{
					EpisodeID:        episodeID,
					Kind:             kind,
					Confidence:       confidence,
					Reasons:          append([]string(nil), reasons...),
					RunIDs:           []int64{},
					WorkflowNames:    []string{},
					RiskDistribution: "none",
				},
				runSet:   make(map[int64]bool),
				nameSet:  make(map[string]bool),
				rootTime: run.CreatedAt,
			}
			episodeMap[episodeID] = acc
		}

		if !acc.runSet[run.DatabaseID] {
			acc.runSet[run.DatabaseID] = true
			acc.metadata.RunIDs = append(acc.metadata.RunIDs, run.DatabaseID)
		}
		if run.WorkflowName != "" && !acc.nameSet[run.WorkflowName] {
			acc.nameSet[run.WorkflowName] = true
			acc.metadata.WorkflowNames = append(acc.metadata.WorkflowNames, run.WorkflowName)
		}

		acc.metadata.TotalRuns++
		acc.metadata.TotalTokens += run.TokenUsage
		acc.metadata.TotalEstimatedCost += run.EstimatedCost
		if run.Comparison != nil && run.Comparison.Classification != nil && run.Comparison.Classification.Label == "risky" {
			acc.metadata.RiskyNodeCount++
		}
		if run.BehaviorFingerprint != nil && run.BehaviorFingerprint.ActuationStyle != "read_only" {
			acc.metadata.WriteCapableNodeCount++
		}
		acc.metadata.MissingToolCount += run.MissingToolCount
		if pr, ok := processedByID[run.DatabaseID]; ok {
			acc.metadata.MCPFailureCount += len(pr.MCPFailures)
			if pr.FirewallAnalysis != nil {
				acc.metadata.BlockedRequestCount += pr.FirewallAnalysis.BlockedRequests
			}
		}
		if !run.CreatedAt.IsZero() && (acc.metadata.RootRunID == 0 || run.CreatedAt.Before(acc.rootTime)) {
			acc.rootTime = run.CreatedAt
			acc.metadata.RootRunID = run.DatabaseID
		}
		if run.StartedAt.IsZero() && run.UpdatedAt.IsZero() {
			acc.duration += run.CreatedAt.Sub(run.CreatedAt)
		} else if !run.StartedAt.IsZero() && !run.UpdatedAt.IsZero() && run.UpdatedAt.After(run.StartedAt) {
			acc.duration += run.UpdatedAt.Sub(run.StartedAt)
		} else if pr, ok := processedByID[run.DatabaseID]; ok && pr.Run.Duration > 0 {
			acc.duration += pr.Run.Duration
		}
	}

	for index := range edges {
		root := findEpisodeParent(parents, edges[index].TargetRunID)
		if selectedSeed, ok := rootMetadata[root]; ok {
			edges[index].EpisodeID = selectedSeed.EpisodeID
		}
	}

	episodes := make([]EpisodeData, 0, len(episodeMap))
	for _, acc := range episodeMap {
		slices.Sort(acc.metadata.RunIDs)
		slices.Sort(acc.metadata.WorkflowNames)
		if acc.duration > 0 {
			acc.metadata.TotalDuration = timeutil.FormatDuration(acc.duration)
		}
		switch acc.metadata.RiskyNodeCount {
		case 0:
			acc.metadata.RiskDistribution = "none"
		case 1:
			acc.metadata.RiskDistribution = "concentrated"
		default:
			acc.metadata.RiskDistribution = "distributed"
		}
		episodes = append(episodes, acc.metadata)
	}

	slices.SortFunc(episodes, func(a, b EpisodeData) int {
		if a.RootRunID != b.RootRunID {
			return cmp.Compare(a.RootRunID, b.RootRunID)
		}
		return cmp.Compare(a.EpisodeID, b.EpisodeID)
	})
	slices.SortFunc(edges, func(a, b EpisodeEdge) int {
		if a.SourceRunID != b.SourceRunID {
			return cmp.Compare(a.SourceRunID, b.SourceRunID)
		}
		return cmp.Compare(a.TargetRunID, b.TargetRunID)
	})

	return episodes, edges
}

func findEpisodeParent(parents map[int64]int64, runID int64) int64 {
	parent, exists := parents[runID]
	if !exists || parent == runID {
		return runID
	}
	root := findEpisodeParent(parents, parent)
	parents[runID] = root
	return root
}

func unionEpisodes(parents map[int64]int64, leftRunID, rightRunID int64) {
	leftRoot := findEpisodeParent(parents, leftRunID)
	rightRoot := findEpisodeParent(parents, rightRunID)
	if leftRoot == rightRoot {
		return
	}
	parents[leftRoot] = rightRoot
}

func compareEpisodeSeeds(left, right episodeSeed) int {
	if left.Kind != right.Kind {
		return cmp.Compare(seedKindRank(left.Kind), seedKindRank(right.Kind))
	}
	if left.Confidence != right.Confidence {
		return cmp.Compare(seedConfidenceRank(left.Confidence), seedConfidenceRank(right.Confidence))
	}
	return cmp.Compare(left.EpisodeID, right.EpisodeID)
}

func seedKindRank(kind string) int {
	switch kind {
	case "workflow_call":
		return 4
	case "dispatch_workflow":
		return 3
	case "workflow_run":
		return 2
	default:
		return 1
	}
}

func seedConfidenceRank(confidence string) int {
	switch confidence {
	case "high":
		return 3
	case "medium":
		return 2
	default:
		return 1
	}
}

func classifyEpisode(run RunData) (string, string, string, []string) {
	if run.AwContext != nil {
		if run.AwContext.WorkflowCallID != "" {
			return "dispatch:" + run.AwContext.WorkflowCallID, "dispatch_workflow", "high", []string{"context.workflow_call_id"}
		}
		if run.AwContext.RunID != "" && run.AwContext.WorkflowID != "" {
			return fmt.Sprintf("dispatch:%s:%s:%s", run.AwContext.Repo, run.AwContext.RunID, run.AwContext.WorkflowID), "dispatch_workflow", "medium", []string{"context.run_id", "context.workflow_id"}
		}
	}
	if run.Event == "workflow_run" {
		return fmt.Sprintf("workflow_run:%d", run.DatabaseID), "workflow_run", "low", []string{"event=workflow_run", "upstream run metadata unavailable in logs summary"}
	}
	return fmt.Sprintf("standalone:%d", run.DatabaseID), "standalone", "high", []string{"no_shared_lineage_markers"}
}

func buildEpisodeEdge(run RunData, episodeID string, runsByID map[int64]RunData) (EpisodeEdge, bool) {
	if run.AwContext == nil || run.AwContext.RunID == "" {
		return EpisodeEdge{}, false
	}
	sourceRunID, err := strconv.ParseInt(run.AwContext.RunID, 10, 64)
	if err != nil {
		return EpisodeEdge{}, false
	}
	if _, ok := runsByID[sourceRunID]; !ok {
		return EpisodeEdge{}, false
	}
	confidence := "medium"
	reasons := []string{"context.run_id"}
	if run.AwContext.WorkflowCallID != "" {
		confidence = "high"
		reasons = append(reasons, "context.workflow_call_id")
	}
	if run.AwContext.WorkflowID != "" {
		reasons = append(reasons, "context.workflow_id")
	}
	return EpisodeEdge{
		SourceRunID: sourceRunID,
		TargetRunID: run.DatabaseID,
		EdgeType:    "dispatch_workflow",
		Confidence:  confidence,
		Reasons:     reasons,
		SourceRepo:  run.AwContext.Repo,
		SourceRef:   run.AwContext.WorkflowID,
		EventType:   run.AwContext.EventType,
		EpisodeID:   episodeID,
	}, true
}
