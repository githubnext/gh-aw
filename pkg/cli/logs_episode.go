package cli

import (
	"cmp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/github/gh-aw/pkg/logger"
)

var logsEpisodeLog = logger.New("cli:logs_episode")

// firewallBlockedRequestCap is the maximum blocked-request count the Squid firewall
// proxy reports before truncating. A BlockedRequestCount equal to this value may
// represent any number of requests >= 50 and should not be treated as an exact figure.
const firewallBlockedRequestCap = 50

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
	EpisodeID                      string              `json:"episode_id"`
	Kind                           string              `json:"kind"`
	Confidence                     string              `json:"confidence"`
	Reasons                        []string            `json:"reasons,omitempty"`
	RepresentativeRunID            int64               `json:"representative_run_id,omitempty"`
	RepresentativeWorkflow         string              `json:"representative_workflow,omitempty"`
	RecordedRoots                  []EpisodeRoot       `json:"recorded_roots,omitempty"`
	Coverage                       EpisodeCoverage     `json:"coverage"`
	RunIDs                         []int64             `json:"run_ids"`
	WorkflowNames                  []string            `json:"workflow_names"`
	TotalRuns                      int                 `json:"total_runs"`
	TotalTokens                    int                 `json:"total_tokens"`
	TotalAIC                       float64             `json:"total_aic,omitempty"`
	TotalDuration                  string              `json:"total_duration"`
	RiskyNodeCount                 int                 `json:"risky_node_count"`
	ChangedNodeCount               int                 `json:"changed_node_count"`
	WriteCapableNodeCount          int                 `json:"write_capable_node_count"`
	MissingToolCount               int                 `json:"missing_tool_count"`
	MCPFailureCount                int                 `json:"mcp_failure_count"`
	BlockedRequestCount            int                 `json:"blocked_request_count"`
	LatestSuccessFallbackCount     int                 `json:"latest_success_fallback_count"`
	NewMCPFailureRunCount          int                 `json:"new_mcp_failure_run_count"`
	BlockedRequestIncreaseRunCount int                 `json:"blocked_request_increase_run_count"`
	BlockedRequestAtCap            bool                `json:"blocked_request_at_cap,omitempty"`
	ResourceHeavyNodeCount         int                 `json:"resource_heavy_node_count"`
	PoorControlNodeCount           int                 `json:"poor_control_node_count"`
	RiskDistribution               string              `json:"risk_distribution"`
	EscalationEligible             bool                `json:"escalation_eligible"`
	EscalationReason               string              `json:"escalation_reason,omitempty"`
	SuggestedRoute                 string              `json:"suggested_route,omitempty"`
	Repository                     string              `json:"repository,omitempty"`
	Organization                   string              `json:"organization,omitempty"`
	ManifestEntryCount             int                 `json:"manifest_entry_count,omitempty"`
	TemporaryIDMappings            int                 `json:"temporary_id_mappings,omitempty"`
	ChainedTargetCount             int                 `json:"chained_target_count,omitempty"`
	ChainedFollowupActionCount     int                 `json:"chained_followup_action_count,omitempty"`
	DelegatedTempTargetCount       int                 `json:"delegated_temp_target_count,omitempty"`
	ClosedTempTargetCount          int                 `json:"closed_temp_target_count,omitempty"`
	ToolCalls                      []EpisodeToolCall   `json:"tool_calls,omitempty"`
	Explanation                    *EpisodeExplanation `json:"explanation,omitempty"`
}

type episodeAccumulator struct {
	metadata           EpisodeData
	duration           time.Duration
	runSet             map[int64]bool
	nameSet            map[string]bool
	representativeTime time.Time
	toolCalls          []EpisodeToolCall
}

func buildEpisodeData(runs []RunData, processedRuns []ProcessedRun) ([]EpisodeData, []EpisodeEdge) {
	logsEpisodeLog.Printf("Building episode data: runs=%d processed_runs=%d", len(runs), len(processedRuns))
	runs = slices.Clone(runs)
	slices.SortFunc(runs, func(a, b RunData) int { return cmp.Compare(a.RunID, b.RunID) })
	runsByID := make(map[int64]RunData, len(runs))
	processedByID := make(map[int64]ProcessedRun, len(processedRuns))
	for _, run := range runs {
		runsByID[run.RunID] = run
	}
	for _, processedRun := range processedRuns {
		processedByID[processedRun.Run.DatabaseID] = processedRun
	}

	edges := make([]EpisodeEdge, 0)
	for _, run := range runs {
		if edge, ok := buildEpisodeEdge(run, runsByID); ok {
			edges = append(edges, edge)
		}
	}
	episodes := rollupEpisodes(runs, processedByID)
	addEpisodeCoverage(episodes, runsByID)
	for i := range episodes {
		episodes[i].Explanation = explainEpisode(episodes[i], runs, edges)
	}
	slices.SortFunc(episodes, func(a, b EpisodeData) int {
		if a.RepresentativeRunID != b.RepresentativeRunID {
			return cmp.Compare(a.RepresentativeRunID, b.RepresentativeRunID)
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

func rollupEpisodes(runs []RunData, processedByID map[int64]ProcessedRun) []EpisodeData {
	episodeMap := make(map[string]*episodeAccumulator)
	for _, run := range runs {
		episodeID := canonicalEpisodeID(run)
		if episodeID == "" {
			logsEpisodeLog.Printf("Skipping episode rollup for run %d: context.episode_id unavailable", run.RunID)
			continue
		}
		kind := canonicalEpisodeKind(run)
		acc, exists := episodeMap[episodeID]
		if !exists {
			acc = &episodeAccumulator{
				metadata: EpisodeData{
					EpisodeID:        episodeID,
					Kind:             kind,
					Confidence:       "high",
					Reasons:          []string{"context.episode_id"},
					RunIDs:           []int64{},
					WorkflowNames:    []string{},
					RiskDistribution: "none",
				},
				runSet:  make(map[int64]bool),
				nameSet: make(map[string]bool),
			}
			episodeMap[episodeID] = acc
		} else if canonicalEpisodeKindRank(kind) > canonicalEpisodeKindRank(acc.metadata.Kind) {
			acc.metadata.Kind = kind
		}

		acc.addRun(run)
		acc.addSignals(run)
		acc.addProcessedRun(processedByID[run.RunID])
		acc.addIdentity(run)
		acc.addDuration(run, processedByID[run.RunID])
	}

	episodes := make([]EpisodeData, 0, len(episodeMap))
	for _, acc := range episodeMap {
		episodes = append(episodes, acc.finalize())
	}
	return episodes
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

func canonicalEpisodeKindRank(kind string) int {
	switch kind {
	case "workflow_call":
		return 4
	case "dispatch_workflow":
		return 3
	default:
		return 1
	}
}

func canonicalEpisodeKind(run RunData) string {
	if run.Event == "workflow_call" {
		return "workflow_call"
	}
	if run.AwContext.RunID == strconv.FormatInt(run.RunID, 10) && run.AwContext.ParentHopID == "" {
		return "standalone"
	}
	return "dispatch_workflow"
}

func canonicalEpisodeID(run RunData) string {
	if run.AwContext == nil {
		return ""
	}
	return strings.TrimSpace(run.AwContext.EpisodeID)
}

func buildEpisodeEdge(run RunData, runsByID map[int64]RunData) (EpisodeEdge, bool) {
	episodeID := canonicalEpisodeID(run)
	if episodeID == "" || run.AwContext.RunID == "" {
		return EpisodeEdge{}, false
	}
	sourceRunID, valid := parseEpisodeRunID(run.AwContext.RunID)
	if !valid {
		logsEpisodeLog.Printf("Ignoring invalid recorded parent run ID for run %d", run.RunID)
		return EpisodeEdge{}, false
	}
	source, ok := runsByID[sourceRunID]
	if !ok {
		logsEpisodeLog.Printf("Dispatch source run %d not found in run set for run %d", sourceRunID, run.RunID)
		return EpisodeEdge{}, false
	}
	if sourceRunID == run.RunID {
		return EpisodeEdge{}, false
	}
	if canonicalEpisodeID(source) != episodeID {
		logsEpisodeLog.Printf("Ignoring episode edge without matching canonical identity: source_run=%d target_run=%d", sourceRunID, run.RunID)
		return EpisodeEdge{}, false
	}
	if episodeRepositoryConflict(run.AwContext.Repo, source.Repository) {
		logsEpisodeLog.Printf("Ignoring dispatch source repository mismatch: source_run=%d target_run=%d", sourceRunID, run.RunID)
		return EpisodeEdge{}, false
	}
	logsEpisodeLog.Printf("Building dispatch episode edge: target_run=%d source_run=%d", run.RunID, sourceRunID)
	reasons := []string{"context.run_id", "context.episode_id"}
	if run.AwContext.WorkflowID != "" {
		reasons = append(reasons, "context.workflow_id")
	}
	edgeType := "dispatch_workflow"
	if run.Event == "workflow_call" {
		edgeType = "workflow_call"
	}
	return EpisodeEdge{
		SourceRunID: sourceRunID,
		TargetRunID: run.RunID,
		EdgeType:    edgeType,
		Confidence:  "high",
		Reasons:     reasons,
		SourceRepo:  run.AwContext.Repo,
		SourceRef:   run.AwContext.WorkflowID,
		EventType:   run.AwContext.EventType,
		EpisodeID:   episodeID,
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
	logsEpisodeLog.Printf("Building suggested route for episode: id=%s kind=%s representative_workflow=%s", episode.EpisodeID, episode.Kind, episode.RepresentativeWorkflow)
	if episode.RepresentativeWorkflow != "" {
		return "workflow:" + episode.RepresentativeWorkflow
	}
	if len(episode.WorkflowNames) > 0 {
		return "workflow:" + episode.WorkflowNames[0]
	}
	return "repo:owners"
}
