package cli

import (
	"cmp"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/timeutil"
)

var logsEpisodeLog = logger.New("cli:logs_episode")

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

// EpisodeToolCall represents a single MCP tool call within an episode.
// It provides per-call observability for token consumption, latency, and error details.
type EpisodeToolCall struct {
	Tool       string `json:"tool"`
	Server     string `json:"server"`
	Tokens     int    `json:"tokens"`
	DurationMS int64  `json:"duration_ms"`
	Status     string `json:"status"`
	Error      string `json:"error,omitempty"`
}

// EpisodeData represents a deterministic episode rollup derived from workflow runs.
type EpisodeData struct {
	EpisodeID                      string            `json:"episode_id"`
	Kind                           string            `json:"kind"`
	Confidence                     string            `json:"confidence"`
	Reasons                        []string          `json:"reasons,omitempty"`
	RootRunID                      int64             `json:"root_run_id,omitempty"`
	RunIDs                         []int64           `json:"run_ids"`
	WorkflowNames                  []string          `json:"workflow_names"`
	PrimaryWorkflow                string            `json:"primary_workflow,omitempty"`
	TotalRuns                      int               `json:"total_runs"`
	TotalTokens                    int               `json:"total_tokens"`
	TotalEstimatedCost             float64           `json:"total_estimated_cost"`
	TotalDuration                  string            `json:"total_duration"`
	RiskyNodeCount                 int               `json:"risky_node_count"`
	ChangedNodeCount               int               `json:"changed_node_count"`
	WriteCapableNodeCount          int               `json:"write_capable_node_count"`
	MissingToolCount               int               `json:"missing_tool_count"`
	MCPFailureCount                int               `json:"mcp_failure_count"`
	BlockedRequestCount            int               `json:"blocked_request_count"`
	LatestSuccessFallbackCount     int               `json:"latest_success_fallback_count"`
	NewMCPFailureRunCount          int               `json:"new_mcp_failure_run_count"`
	BlockedRequestIncreaseRunCount int               `json:"blocked_request_increase_run_count"`
	ResourceHeavyNodeCount         int               `json:"resource_heavy_node_count"`
	PoorControlNodeCount           int               `json:"poor_control_node_count"`
	RiskDistribution               string            `json:"risk_distribution"`
	EscalationEligible             bool              `json:"escalation_eligible"`
	EscalationReason               string            `json:"escalation_reason,omitempty"`
	SuggestedRoute                 string            `json:"suggested_route,omitempty"`
	Repository                     string            `json:"repository,omitempty"`
	Organization                   string            `json:"organization,omitempty"`
	ToolCalls                      []EpisodeToolCall `json:"tool_calls,omitempty"`
}

type episodeAccumulator struct {
	metadata  EpisodeData
	duration  time.Duration
	runSet    map[int64]bool
	nameSet   map[string]bool
	rootTime  time.Time
	toolCalls []EpisodeToolCall
}

type episodeSeed struct {
	EpisodeID  string
	Kind       string
	Confidence string
	Reasons    []string
}

func buildEpisodeData(runs []RunData, processedRuns []ProcessedRun) ([]EpisodeData, []EpisodeEdge) {
	logsEpisodeLog.Printf("Building episode data: runs=%d processed_runs=%d", len(runs), len(processedRuns))
	runsByID := make(map[int64]RunData, len(runs))
	processedByID := make(map[int64]ProcessedRun, len(processedRuns))
	seedsByRunID := make(map[int64]episodeSeed, len(runs))
	parents := make(map[int64]int64, len(runs))
	for _, run := range runs {
		runsByID[run.RunID] = run
		episodeID, kind, confidence, reasons := classifyEpisode(run)
		seedsByRunID[run.RunID] = episodeSeed{EpisodeID: episodeID, Kind: kind, Confidence: confidence, Reasons: append([]string(nil), reasons...)}
		parents[run.RunID] = run.RunID
	}
	for _, processedRun := range processedRuns {
		processedByID[processedRun.Run.DatabaseID] = processedRun
	}

	edges := make([]EpisodeEdge, 0)
	for _, run := range runs {
		if edge, ok := buildEpisodeEdge(run, runs, runsByID); ok {
			edges = append(edges, edge)
			unionEpisodes(parents, edge.SourceRunID, edge.TargetRunID)
		}
	}

	episodeMap := make(map[string]*episodeAccumulator)
	rootMetadata := make(map[int64]episodeSeed)
	for _, run := range runs {
		root := findEpisodeParent(parents, run.RunID)
		seed := seedsByRunID[run.RunID]
		best, exists := rootMetadata[root]
		if !exists || compareEpisodeSeeds(seed, best) > 0 {
			rootMetadata[root] = seed
		}
	}

	for _, run := range runs {
		root := findEpisodeParent(parents, run.RunID)
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

		if !acc.runSet[run.RunID] {
			acc.runSet[run.RunID] = true
			acc.metadata.RunIDs = append(acc.metadata.RunIDs, run.RunID)
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
		if run.Comparison != nil && run.Comparison.Classification != nil && run.Comparison.Classification.Label == "changed" {
			acc.metadata.ChangedNodeCount++
		}
		if run.BehaviorFingerprint != nil && run.BehaviorFingerprint.ActuationStyle != "read_only" {
			acc.metadata.WriteCapableNodeCount++
		}
		if run.Comparison != nil && run.Comparison.Baseline != nil && run.Comparison.Baseline.Selection == "latest_success" {
			acc.metadata.LatestSuccessFallbackCount++
		}
		if hasComparisonReasonCode(run.Comparison, "new_mcp_failure") {
			acc.metadata.NewMCPFailureRunCount++
		}
		if hasComparisonReasonCode(run.Comparison, "blocked_requests_increase") {
			acc.metadata.BlockedRequestIncreaseRunCount++
		}
		if hasAssessmentKindAtLeast(run.AgenticAssessments, "resource_heavy_for_domain", "medium") {
			acc.metadata.ResourceHeavyNodeCount++
		}
		if hasAssessmentKindAtLeast(run.AgenticAssessments, "poor_agentic_control", "medium") {
			acc.metadata.PoorControlNodeCount++
		}
		acc.metadata.MissingToolCount += run.MissingToolCount
		if pr, ok := processedByID[run.RunID]; ok {
			acc.metadata.MCPFailureCount += len(pr.MCPFailures)
			if pr.FirewallAnalysis != nil {
				acc.metadata.BlockedRequestCount += pr.FirewallAnalysis.BlockedRequests
			}
			// Collect per-tool-call metrics for this run.
			if pr.MCPToolUsage != nil {
				for _, tc := range pr.MCPToolUsage.ToolCalls {
					acc.toolCalls = append(acc.toolCalls, mcpToolCallToEpisodeToolCall(tc))
				}
			}
		}
		if !run.CreatedAt.IsZero() && (acc.metadata.RootRunID == 0 || run.CreatedAt.Before(acc.rootTime)) {
			acc.rootTime = run.CreatedAt
			acc.metadata.RootRunID = run.RunID
			acc.metadata.PrimaryWorkflow = run.WorkflowName
		}
		if acc.metadata.PrimaryWorkflow == "" && run.WorkflowName != "" {
			acc.metadata.PrimaryWorkflow = run.WorkflowName
		}
		if acc.metadata.Repository == "" && run.Repository != "" {
			acc.metadata.Repository = run.Repository
			if parts := strings.SplitN(run.Repository, "/", 2); len(parts) == 2 {
				acc.metadata.Organization = parts[0]
			}
		}
		if run.StartedAt.IsZero() && run.UpdatedAt.IsZero() {
			acc.duration += run.CreatedAt.Sub(run.CreatedAt)
		} else if !run.StartedAt.IsZero() && !run.UpdatedAt.IsZero() && run.UpdatedAt.After(run.StartedAt) {
			acc.duration += run.UpdatedAt.Sub(run.StartedAt)
		} else if pr, ok := processedByID[run.RunID]; ok && pr.Run.Duration > 0 {
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
		if acc.metadata.PrimaryWorkflow == "" && len(acc.metadata.WorkflowNames) > 0 {
			acc.metadata.PrimaryWorkflow = acc.metadata.WorkflowNames[0]
		}
		switch acc.metadata.RiskyNodeCount {
		case 0:
			acc.metadata.RiskDistribution = "none"
		case 1:
			acc.metadata.RiskDistribution = "concentrated"
		default:
			acc.metadata.RiskDistribution = "distributed"
		}
		acc.metadata.EscalationEligible, acc.metadata.EscalationReason = classifyEpisodeEscalation(acc.metadata)
		acc.metadata.SuggestedRoute = buildSuggestedRoute(acc.metadata)
		if len(acc.toolCalls) > 0 {
			// Sort tool calls for deterministic output (server, then tool name, then status).
			slices.SortFunc(acc.toolCalls, func(a, b EpisodeToolCall) int {
				if a.Server != b.Server {
					return cmp.Compare(a.Server, b.Server)
				}
				if a.Tool != b.Tool {
					return cmp.Compare(a.Tool, b.Tool)
				}
				return cmp.Compare(a.Status, b.Status)
			})
			acc.metadata.ToolCalls = acc.toolCalls
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

	logsEpisodeLog.Printf("Built %d episodes and %d edges from %d runs", len(episodes), len(edges), len(runs))
	return episodes, edges
}

// mcpToolCallToEpisodeToolCall converts an MCPToolCall record to the lightweight
// EpisodeToolCall format used in episode rollups.
// Token count is estimated from input/output byte sizes using CharsPerToken.
// Duration is converted from a formatted string to milliseconds.
func mcpToolCallToEpisodeToolCall(tc MCPToolCall) EpisodeToolCall {
	tokens := (tc.InputSize + tc.OutputSize) / CharsPerToken
	var durationMS int64
	if tc.Duration != "" {
		durationMS = parseDurationString(tc.Duration).Milliseconds()
	}
	return EpisodeToolCall{
		Tool:       tc.ToolName,
		Server:     tc.ServerName,
		Tokens:     tokens,
		DurationMS: durationMS,
		Status:     tc.Status,
		Error:      tc.Error,
	}
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
	logsEpisodeLog.Printf("Classifying episode for run: id=%d event=%s", run.RunID, run.Event)
	if run.AwContext != nil {
		if run.AwContext.WorkflowCallID != "" {
			return "dispatch:" + run.AwContext.WorkflowCallID, "dispatch_workflow", "high", []string{"context.workflow_call_id"}
		}
		if run.AwContext.RunID != "" && run.AwContext.WorkflowID != "" {
			return fmt.Sprintf("dispatch:%s:%s:%s", run.AwContext.Repo, run.AwContext.RunID, run.AwContext.WorkflowID), "dispatch_workflow", "medium", []string{"context.run_id", "context.workflow_id"}
		}
	}
	if episodeID, kind, confidence, reasons, ok := classifyWorkflowCallEpisode(run); ok {
		return episodeID, kind, confidence, reasons
	}
	if run.Event == "workflow_run" {
		return fmt.Sprintf("workflow_run:%d", run.RunID), "workflow_run", "low", []string{"event=workflow_run", "upstream run metadata unavailable in logs summary"}
	}
	return fmt.Sprintf("standalone:%d", run.RunID), "standalone", "high", []string{"no_shared_lineage_markers"}
}

func buildEpisodeEdge(run RunData, runs []RunData, runsByID map[int64]RunData) (EpisodeEdge, bool) {
	if edge, ok := buildDispatchEpisodeEdge(run, runsByID); ok {
		return edge, true
	}
	if edge, ok := buildWorkflowCallEpisodeEdge(run, runs); ok {
		return edge, true
	}
	if edge, ok := buildWorkflowRunEpisodeEdge(run, runs); ok {
		return edge, true
	}
	return EpisodeEdge{}, false
}

func buildDispatchEpisodeEdge(run RunData, runsByID map[int64]RunData) (EpisodeEdge, bool) {
	if run.AwContext == nil || run.AwContext.RunID == "" {
		return EpisodeEdge{}, false
	}
	sourceRunID, err := strconv.ParseInt(run.AwContext.RunID, 10, 64)
	if err != nil {
		logsEpisodeLog.Printf("Failed to parse dispatch source run ID for run %d: %v", run.RunID, err)
		return EpisodeEdge{}, false
	}
	if _, ok := runsByID[sourceRunID]; !ok {
		logsEpisodeLog.Printf("Dispatch source run %d not found in run set for run %d", sourceRunID, run.RunID)
		return EpisodeEdge{}, false
	}
	logsEpisodeLog.Printf("Building dispatch episode edge: target_run=%d source_run=%d", run.RunID, sourceRunID)
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
		TargetRunID: run.RunID,
		EdgeType:    "dispatch_workflow",
		Confidence:  confidence,
		Reasons:     reasons,
		SourceRepo:  run.AwContext.Repo,
		SourceRef:   run.AwContext.WorkflowID,
		EventType:   run.AwContext.EventType,
	}, true
}

func classifyWorkflowCallEpisode(run RunData) (string, string, string, []string, bool) {
	if run.Event != "workflow_call" {
		return "", "", "", nil, false
	}
	logsEpisodeLog.Printf("Classifying workflow_call episode: run_id=%d repo=%s ref=%s", run.RunID, run.Repository, run.Ref)
	reasons := []string{"event=workflow_call"}
	if run.Repository == "" || run.Ref == "" || run.SHA == "" || run.RunAttempt == "" || run.Actor == "" {
		return fmt.Sprintf("workflow_call:%d", run.RunID), "workflow_call", "low", append(reasons, "insufficient_aw_info_metadata"), true
	}
	parts := make([]string, 0, 6)
	parts = append(parts, run.Repository)
	reasons = append(reasons, "repository")
	parts = append(parts, run.Ref)
	reasons = append(reasons, "ref")
	parts = append(parts, run.SHA)
	reasons = append(reasons, "sha")
	parts = append(parts, run.RunAttempt)
	reasons = append(reasons, "run_attempt")
	parts = append(parts, run.Actor)
	reasons = append(reasons, "actor")
	if run.TargetRepo != "" {
		parts = append(parts, "target="+run.TargetRepo)
		reasons = append(reasons, "target_repo")
	}
	confidence := "medium"
	if run.TargetRepo != "" {
		confidence = "high"
	}
	return "workflow_call:" + strings.Join(parts, ":"), "workflow_call", confidence, reasons, true
}

func buildWorkflowCallEpisodeEdge(run RunData, runs []RunData) (EpisodeEdge, bool) {
	if run.Event != "workflow_call" || run.Repository == "" || run.SHA == "" || run.Ref == "" || run.Actor == "" || run.RunAttempt == "" {
		return EpisodeEdge{}, false
	}
	candidates := filterLineageCandidates(runs, run, func(candidate RunData) bool {
		return candidate.Repository == run.Repository &&
			candidate.SHA == run.SHA &&
			candidate.Ref == run.Ref &&
			candidate.Actor == run.Actor &&
			candidate.RunAttempt == run.RunAttempt
	})
	if edge, ok := buildUniqueCandidateEdge(run, candidates, "workflow_call", "medium", []string{"event=workflow_call", "repository_match", "sha_match", "ref_match", "actor_match", "run_attempt_match", "unique_upstream_candidate"}); ok {
		if run.TargetRepo != "" {
			edge.Confidence = "high"
			edge.Reasons = append(edge.Reasons, "target_repo")
			edge.SourceRef = run.TargetRepo
		}
		return edge, true
	}
	return EpisodeEdge{}, false
}

func buildWorkflowRunEpisodeEdge(run RunData, runs []RunData) (EpisodeEdge, bool) {
	if run.Event != "workflow_run" || run.HeadSHA == "" || run.Branch == "" {
		return EpisodeEdge{}, false
	}
	candidates := filterLineageCandidates(runs, run, func(candidate RunData) bool {
		if candidate.HeadSHA != run.HeadSHA || candidate.Branch != run.Branch {
			return false
		}
		if run.Repository != "" && candidate.Repository != "" && candidate.Repository != run.Repository {
			return false
		}
		return true
	})
	confidence := "low"
	reasons := []string{"event=workflow_run", "head_sha_match", "head_branch_match", "unique_upstream_candidate"}
	if run.Repository != "" {
		confidence = "medium"
		reasons = append(reasons, "repository")
	}
	return buildUniqueCandidateEdge(run, candidates, "workflow_run", confidence, reasons)
}

func filterLineageCandidates(runs []RunData, child RunData, matches func(RunData) bool) []RunData {
	candidates := make([]RunData, 0)
	for _, candidate := range runs {
		if candidate.RunID == child.RunID {
			continue
		}
		if child.CreatedAt.IsZero() || candidate.CreatedAt.IsZero() || candidate.CreatedAt.After(child.CreatedAt) {
			continue
		}
		if !matches(candidate) {
			continue
		}
		candidates = append(candidates, candidate)
	}
	nonNested := make([]RunData, 0, len(candidates))
	for _, candidate := range candidates {
		if candidate.Event != child.Event {
			nonNested = append(nonNested, candidate)
		}
	}
	if len(nonNested) == 1 {
		return nonNested
	}
	if len(nonNested) > 1 {
		candidates = nonNested
	}
	slices.SortFunc(candidates, func(left, right RunData) int {
		if !left.CreatedAt.Equal(right.CreatedAt) {
			if left.CreatedAt.Before(right.CreatedAt) {
				return 1
			}
			return -1
		}
		return cmp.Compare(left.RunID, right.RunID)
	})
	return candidates
}

func buildUniqueCandidateEdge(run RunData, candidates []RunData, edgeType, confidence string, reasons []string) (EpisodeEdge, bool) {
	if len(candidates) != 1 {
		return EpisodeEdge{}, false
	}
	parent := candidates[0]
	return EpisodeEdge{
		SourceRunID: parent.RunID,
		TargetRunID: run.RunID,
		EdgeType:    edgeType,
		Confidence:  confidence,
		Reasons:     append([]string(nil), reasons...),
		SourceRepo:  parent.Repository,
		SourceRef:   parent.WorkflowPath,
		EventType:   parent.Event,
	}, true
}

func hasComparisonReasonCode(comparison *AuditComparisonData, code string) bool {
	if comparison == nil || comparison.Classification == nil {
		return false
	}
	return slices.Contains(comparison.Classification.ReasonCodes, code)
}

func hasAssessmentKindAtLeast(assessments []AgenticAssessment, kind, minimumSeverity string) bool {
	for _, assessment := range assessments {
		if assessment.Kind != kind {
			continue
		}
		if severityRank(assessment.Severity) >= severityRank(minimumSeverity) {
			return true
		}
	}
	return false
}

func severityRank(severity string) int {
	switch severity {
	case "high":
		return 3
	case "medium":
		return 2
	default:
		return 1
	}
}

func classifyEpisodeEscalation(episode EpisodeData) (bool, string) {
	logsEpisodeLog.Printf("Classifying episode escalation: episode_id=%s risky_nodes=%d mcp_failures=%d resource_heavy=%d poor_control=%d", episode.EpisodeID, episode.RiskyNodeCount, episode.NewMCPFailureRunCount, episode.ResourceHeavyNodeCount, episode.PoorControlNodeCount)
	switch {
	case episode.RiskyNodeCount >= 2:
		return true, "repeated_risky_runs"
	case episode.NewMCPFailureRunCount >= 2:
		return true, "repeated_new_mcp_failures"
	case episode.BlockedRequestIncreaseRunCount >= 2:
		return true, "repeated_blocked_request_increase"
	case episode.ResourceHeavyNodeCount >= 2:
		return true, "repeated_resource_heavy_for_domain"
	case episode.PoorControlNodeCount >= 2:
		return true, "repeated_poor_agentic_control"
	default:
		return false, ""
	}
}

func buildSuggestedRoute(episode EpisodeData) string {
	logsEpisodeLog.Printf("Building suggested route for episode: id=%s kind=%s primary_workflow=%s", episode.EpisodeID, episode.Kind, episode.PrimaryWorkflow)
	if episode.PrimaryWorkflow != "" {
		return "workflow:" + episode.PrimaryWorkflow
	}
	if len(episode.WorkflowNames) > 0 {
		return "workflow:" + episode.WorkflowNames[0]
	}
	return "repo:owners"
}
