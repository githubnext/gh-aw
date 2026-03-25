package cli

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"github.com/github/gh-aw/pkg/workflow"
)

type AuditComparisonData struct {
	BaselineFound  bool                           `json:"baseline_found"`
	Baseline       *AuditComparisonBaseline       `json:"baseline,omitempty"`
	Delta          *AuditComparisonDelta          `json:"delta,omitempty"`
	Classification *AuditComparisonClassification `json:"classification,omitempty"`
	Recommendation *AuditComparisonRecommendation `json:"recommendation,omitempty"`
}

type AuditComparisonBaseline struct {
	RunID        int64    `json:"run_id"`
	WorkflowName string   `json:"workflow_name,omitempty"`
	Conclusion   string   `json:"conclusion,omitempty"`
	CreatedAt    string   `json:"created_at,omitempty"`
	Selection    string   `json:"selection,omitempty"`
	MatchedOn    []string `json:"matched_on,omitempty"`
}

type AuditComparisonDelta struct {
	Turns           AuditComparisonIntDelta         `json:"turns"`
	Posture         AuditComparisonStringDelta      `json:"posture"`
	BlockedRequests AuditComparisonIntDelta         `json:"blocked_requests"`
	MCPFailure      *AuditComparisonMCPFailureDelta `json:"mcp_failure,omitempty"`
}

type AuditComparisonIntDelta struct {
	Before  int  `json:"before"`
	After   int  `json:"after"`
	Changed bool `json:"changed"`
}

type AuditComparisonStringDelta struct {
	Before  string `json:"before"`
	After   string `json:"after"`
	Changed bool   `json:"changed"`
}

type AuditComparisonMCPFailureDelta struct {
	Before       []string `json:"before,omitempty"`
	After        []string `json:"after,omitempty"`
	NewlyPresent bool     `json:"newly_present"`
}

type AuditComparisonClassification struct {
	Label       string   `json:"label"`
	ReasonCodes []string `json:"reason_codes,omitempty"`
}

type AuditComparisonRecommendation struct {
	Action string `json:"action"`
}

type auditComparisonSnapshot struct {
	Turns           int
	Posture         string
	BlockedRequests int
	MCPFailures     []string
}

type auditComparisonCandidate struct {
	Run                 WorkflowRun
	Snapshot            auditComparisonSnapshot
	TaskDomain          *TaskDomainInfo
	BehaviorFingerprint *BehaviorFingerprint
	Selection           string
	MatchedOn           []string
	Score               int
}

const maxAuditComparisonCandidates = 10

func buildAuditComparisonSnapshot(processedRun ProcessedRun, createdItems []CreatedItemReport) auditComparisonSnapshot {
	blockedRequests := 0
	if processedRun.FirewallAnalysis != nil {
		blockedRequests = processedRun.FirewallAnalysis.BlockedRequests
	}

	return auditComparisonSnapshot{
		Turns:           processedRun.Run.Turns,
		Posture:         deriveAuditPosture(createdItems),
		BlockedRequests: blockedRequests,
		MCPFailures:     collectMCPFailureServers(processedRun.MCPFailures),
	}
}

func loadAuditComparisonSnapshotFromArtifacts(run WorkflowRun, logsPath string, verbose bool) (auditComparisonSnapshot, error) {
	metrics, err := extractLogMetrics(logsPath, verbose, run.WorkflowPath)
	if err != nil {
		return auditComparisonSnapshot{}, fmt.Errorf("failed to extract baseline metrics: %w", err)
	}

	firewallAnalysis, err := analyzeFirewallLogs(logsPath, verbose)
	if err != nil {
		return auditComparisonSnapshot{}, fmt.Errorf("failed to analyze baseline firewall logs: %w", err)
	}

	mcpFailures, err := extractMCPFailuresFromRun(logsPath, run, verbose)
	if err != nil {
		return auditComparisonSnapshot{}, fmt.Errorf("failed to extract baseline MCP failures: %w", err)
	}

	blockedRequests := 0
	if firewallAnalysis != nil {
		blockedRequests = firewallAnalysis.BlockedRequests
	}

	return auditComparisonSnapshot{
		Turns:           metrics.Turns,
		Posture:         deriveAuditPosture(extractCreatedItemsFromManifest(logsPath)),
		BlockedRequests: blockedRequests,
		MCPFailures:     collectMCPFailureServers(mcpFailures),
	}, nil
}

func buildAuditComparisonCandidateFromSummary(summary *RunSummary, logsPath string) auditComparisonCandidate {
	createdItems := extractCreatedItemsFromManifest(logsPath)
	posture := deriveAuditPosture(createdItems)

	blockedRequests := 0
	if summary.FirewallAnalysis != nil {
		blockedRequests = summary.FirewallAnalysis.BlockedRequests
	}

	return auditComparisonCandidate{
		Run: summary.Run,
		Snapshot: auditComparisonSnapshot{
			Turns:           summary.Metrics.Turns,
			Posture:         posture,
			BlockedRequests: blockedRequests,
			MCPFailures:     collectMCPFailureServers(summary.MCPFailures),
		},
		TaskDomain:          summary.TaskDomain,
		BehaviorFingerprint: summary.BehaviorFingerprint,
	}
}

func buildAuditComparisonCandidateFromProcessedRun(processedRun ProcessedRun) auditComparisonCandidate {
	return auditComparisonCandidate{
		Run:                 processedRun.Run,
		Snapshot:            buildAuditComparisonSnapshot(processedRun, extractCreatedItemsFromManifest(processedRun.Run.LogsPath)),
		TaskDomain:          processedRun.TaskDomain,
		BehaviorFingerprint: processedRun.BehaviorFingerprint,
	}
}

func loadAuditComparisonCandidate(run WorkflowRun, logsPath string, verbose bool) (auditComparisonCandidate, error) {
	if summary, ok := loadRunSummary(logsPath, false); ok && summary != nil {
		candidate := buildAuditComparisonCandidateFromSummary(summary, logsPath)
		candidate.Run = run
		return candidate, nil
	}

	snapshot, err := loadAuditComparisonSnapshotFromArtifacts(run, logsPath, verbose)
	if err != nil {
		return auditComparisonCandidate{}, err
	}

	processedRun := ProcessedRun{Run: run}
	metrics, metricsErr := extractLogMetrics(logsPath, verbose, run.WorkflowPath)
	if metricsErr == nil {
		processedRun.Run.TokenUsage = metrics.TokenUsage
		processedRun.Run.EstimatedCost = metrics.EstimatedCost
		processedRun.Run.Turns = metrics.Turns
	}
	if firewallAnalysis, firewallErr := analyzeFirewallLogs(logsPath, verbose); firewallErr == nil {
		processedRun.FirewallAnalysis = firewallAnalysis
	}
	if mcpFailures, mcpErr := extractMCPFailuresFromRun(logsPath, run, verbose); mcpErr == nil {
		processedRun.MCPFailures = mcpFailures
	}
	awContext, _, _, taskDomain, behaviorFingerprint, _ := deriveRunAgenticAnalysis(processedRun, metrics)
	processedRun.AwContext = awContext

	return auditComparisonCandidate{
		Run:                 run,
		Snapshot:            snapshot,
		TaskDomain:          taskDomain,
		BehaviorFingerprint: behaviorFingerprint,
		Selection:           "latest_success",
		MatchedOn:           nil,
		Score:               0,
	}, nil
}

func scoreAuditComparisonCandidate(current ProcessedRun, candidate *auditComparisonCandidate) {
	if candidate == nil {
		return
	}

	score := 0
	matchedOn := make([]string, 0, 6)

	if current.Run.Event != "" && current.Run.Event == candidate.Run.Event {
		score += 5
		matchedOn = append(matchedOn, "event")
	}

	if current.TaskDomain != nil && candidate.TaskDomain != nil && current.TaskDomain.Name == candidate.TaskDomain.Name {
		score += 50
		matchedOn = append(matchedOn, "task_domain")
	}

	if current.BehaviorFingerprint != nil && candidate.BehaviorFingerprint != nil {
		if current.BehaviorFingerprint.ExecutionStyle == candidate.BehaviorFingerprint.ExecutionStyle {
			score += 20
			matchedOn = append(matchedOn, "execution_style")
		}
		if current.BehaviorFingerprint.ResourceProfile == candidate.BehaviorFingerprint.ResourceProfile {
			score += 25
			matchedOn = append(matchedOn, "resource_profile")
		}
		if current.BehaviorFingerprint.ActuationStyle == candidate.BehaviorFingerprint.ActuationStyle {
			score += 10
			matchedOn = append(matchedOn, "actuation_style")
		}
		if current.BehaviorFingerprint.DispatchMode == candidate.BehaviorFingerprint.DispatchMode {
			score += 5
			matchedOn = append(matchedOn, "dispatch_mode")
		}
		if current.BehaviorFingerprint.ToolBreadth == candidate.BehaviorFingerprint.ToolBreadth {
			score += 2
			matchedOn = append(matchedOn, "tool_breadth")
		}
	}

	candidate.Score = score
	if slices.Contains(matchedOn, "task_domain") || slices.Contains(matchedOn, "execution_style") || slices.Contains(matchedOn, "resource_profile") || slices.Contains(matchedOn, "actuation_style") {
		candidate.Selection = "cohort_match"
		candidate.MatchedOn = matchedOn
		return
	}

	candidate.Selection = "latest_success"
	candidate.MatchedOn = nil
}

func selectAuditComparisonBaseline(current ProcessedRun, candidates []auditComparisonCandidate) *auditComparisonCandidate {
	if len(candidates) == 0 {
		return nil
	}

	for index := range candidates {
		scoreAuditComparisonCandidate(current, &candidates[index])
	}

	sort.SliceStable(candidates, func(left, right int) bool {
		if candidates[left].Score != candidates[right].Score {
			return candidates[left].Score > candidates[right].Score
		}
		return candidates[left].Run.CreatedAt.After(candidates[right].Run.CreatedAt)
	})

	return &candidates[0]
}

func sameAuditComparisonWorkflow(left WorkflowRun, right WorkflowRun) bool {
	if left.WorkflowPath != "" && right.WorkflowPath != "" {
		return left.WorkflowPath == right.WorkflowPath
	}
	if left.WorkflowName != "" && right.WorkflowName != "" {
		return left.WorkflowName == right.WorkflowName
	}
	return false
}

func buildAuditComparisonForProcessedRuns(currentRun ProcessedRun, processedRuns []ProcessedRun) *AuditComparisonData {
	currentSnapshot := buildAuditComparisonSnapshot(currentRun, extractCreatedItemsFromManifest(currentRun.Run.LogsPath))
	candidates := make([]auditComparisonCandidate, 0, len(processedRuns))

	for _, candidateRun := range processedRuns {
		if candidateRun.Run.DatabaseID == currentRun.Run.DatabaseID {
			continue
		}
		if candidateRun.Run.Conclusion != "success" {
			continue
		}
		if !candidateRun.Run.CreatedAt.Before(currentRun.Run.CreatedAt) {
			continue
		}
		if !sameAuditComparisonWorkflow(currentRun.Run, candidateRun.Run) {
			continue
		}

		candidates = append(candidates, buildAuditComparisonCandidateFromProcessedRun(candidateRun))
	}

	selected := selectAuditComparisonBaseline(currentRun, candidates)
	if selected == nil {
		return &AuditComparisonData{BaselineFound: false}
	}

	comparison := buildAuditComparison(currentSnapshot, &selected.Run, &selected.Snapshot)
	if comparison != nil && comparison.Baseline != nil {
		comparison.Baseline.Selection = selected.Selection
		comparison.Baseline.MatchedOn = selected.MatchedOn
	}
	return comparison
}

func buildAuditComparison(current auditComparisonSnapshot, baselineRun *WorkflowRun, baseline *auditComparisonSnapshot) *AuditComparisonData {
	if baselineRun == nil || baseline == nil {
		return &AuditComparisonData{BaselineFound: false}
	}

	reasonCodes := make([]string, 0, 4)
	delta := &AuditComparisonDelta{
		Turns: AuditComparisonIntDelta{
			Before:  baseline.Turns,
			After:   current.Turns,
			Changed: baseline.Turns != current.Turns,
		},
		Posture: AuditComparisonStringDelta{
			Before:  baseline.Posture,
			After:   current.Posture,
			Changed: baseline.Posture != current.Posture,
		},
		BlockedRequests: AuditComparisonIntDelta{
			Before:  baseline.BlockedRequests,
			After:   current.BlockedRequests,
			Changed: baseline.BlockedRequests != current.BlockedRequests,
		},
	}

	if current.Turns > baseline.Turns {
		reasonCodes = append(reasonCodes, "turns_increase")
	} else if current.Turns < baseline.Turns {
		reasonCodes = append(reasonCodes, "turns_decrease")
	}
	if baseline.Posture != current.Posture {
		reasonCodes = append(reasonCodes, "posture_changed")
	}
	if current.BlockedRequests > baseline.BlockedRequests {
		reasonCodes = append(reasonCodes, "blocked_requests_increase")
	} else if current.BlockedRequests < baseline.BlockedRequests {
		reasonCodes = append(reasonCodes, "blocked_requests_decrease")
	}

	newMCPFailure := len(baseline.MCPFailures) == 0 && len(current.MCPFailures) > 0
	mcpFailuresResolved := len(baseline.MCPFailures) > 0 && len(current.MCPFailures) == 0
	if newMCPFailure || len(baseline.MCPFailures) > 0 || len(current.MCPFailures) > 0 {
		delta.MCPFailure = &AuditComparisonMCPFailureDelta{
			Before:       baseline.MCPFailures,
			After:        current.MCPFailures,
			NewlyPresent: newMCPFailure,
		}
	}
	if newMCPFailure {
		reasonCodes = append(reasonCodes, "new_mcp_failure")
	} else if mcpFailuresResolved {
		reasonCodes = append(reasonCodes, "mcp_failures_resolved")
	}

	label := "stable"
	switch {
	case delta.Posture.Before == "read_only" && delta.Posture.After == "write_capable":
		label = "risky"
	case newMCPFailure:
		label = "risky"
	case current.BlockedRequests > baseline.BlockedRequests:
		label = "risky"
	case delta.Posture.Before != "" && delta.Posture.After != "" && delta.Posture.Before != delta.Posture.After:
		label = "changed"
	case mcpFailuresResolved:
		label = "changed"
	case current.BlockedRequests < baseline.BlockedRequests:
		label = "changed"
	case len(reasonCodes) > 0:
		label = "changed"
	}

	return &AuditComparisonData{
		BaselineFound: true,
		Baseline: &AuditComparisonBaseline{
			RunID:        baselineRun.DatabaseID,
			WorkflowName: baselineRun.WorkflowName,
			Conclusion:   baselineRun.Conclusion,
			CreatedAt:    baselineRun.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			Selection:    "latest_success",
		},
		Delta: delta,
		Classification: &AuditComparisonClassification{
			Label:       label,
			ReasonCodes: reasonCodes,
		},
		Recommendation: &AuditComparisonRecommendation{
			Action: recommendAuditComparisonAction(label, delta),
		},
	}
}

func recommendAuditComparisonAction(label string, delta *AuditComparisonDelta) string {
	if delta == nil || label == "stable" {
		return "No action needed; this run matches the selected successful baseline closely."
	}

	if delta.Posture.Before == "read_only" && delta.Posture.After == "write_capable" {
		return "Review first-time write-capable behavior and add a guardrail before enabling by default."
	}
	if delta.MCPFailure != nil && delta.MCPFailure.NewlyPresent {
		return "Inspect the new MCP failure and restore tool availability before relying on this workflow."
	}
	if delta.BlockedRequests.After > delta.BlockedRequests.Before {
		return "Review network policy changes before treating the new blocked requests as normal behavior."
	}
	if delta.Turns.After > delta.Turns.Before {
		return "Compare prompt or task-shape changes because this run needed more turns than the selected successful baseline."
	}

	return "Review the behavior change against the selected successful baseline before treating it as the new normal."
}

func deriveAuditPosture(createdItems []CreatedItemReport) string {
	if len(createdItems) > 0 {
		return "write_capable"
	}
	return "read_only"
}

func collectMCPFailureServers(failures []MCPFailureReport) []string {
	if len(failures) == 0 {
		return nil
	}

	serverSet := make(map[string]struct{}, len(failures))
	for _, failure := range failures {
		if strings.TrimSpace(failure.ServerName) == "" {
			continue
		}
		serverSet[failure.ServerName] = struct{}{}
	}

	servers := make([]string, 0, len(serverSet))
	for server := range serverSet {
		servers = append(servers, server)
	}
	sort.Strings(servers)
	return servers
}

func findPreviousSuccessfulWorkflowRuns(current WorkflowRun, owner, repo, hostname string, verbose bool) ([]WorkflowRun, error) {
	_ = verbose
	workflowID := filepath.Base(current.WorkflowPath)
	if workflowID == "." || workflowID == "" {
		return nil, fmt.Errorf("workflow path unavailable for run %d", current.DatabaseID)
	}

	encodedWorkflowID := url.PathEscape(workflowID)
	var endpoint string
	if owner != "" && repo != "" {
		endpoint = fmt.Sprintf("repos/%s/%s/actions/workflows/%s/runs?per_page=%d", owner, repo, encodedWorkflowID, maxAuditComparisonCandidates)
	} else {
		endpoint = fmt.Sprintf("repos/{owner}/{repo}/actions/workflows/%s/runs?per_page=%d", encodedWorkflowID, maxAuditComparisonCandidates)
	}

	jq := fmt.Sprintf(`[.workflow_runs[] | select(.id != %d and .conclusion == "success" and .created_at < "%s") | {databaseId: .id, number: .run_number, url: .html_url, status: .status, conclusion: .conclusion, workflowName: .name, workflowPath: .path, createdAt: .created_at, startedAt: .run_started_at, updatedAt: .updated_at, event: .event, headBranch: .head_branch, headSha: .head_sha, displayTitle: .display_title}]`, current.DatabaseID, current.CreatedAt.Format("2006-01-02T15:04:05Z07:00"))

	args := []string{"api"}
	if hostname != "" && hostname != "github.com" {
		args = append(args, "--hostname", hostname)
	}
	args = append(args, endpoint, "--jq", jq)

	output, err := workflow.RunGHCombined("Fetching previous successful workflow run...", args...)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch previous successful workflow run: %w", err)
	}

	trimmed := strings.TrimSpace(string(output))
	if trimmed == "null" || trimmed == "" || trimmed == "[]" {
		return nil, nil
	}

	var runs []WorkflowRun
	if err := json.Unmarshal(output, &runs); err != nil {
		return nil, fmt.Errorf("failed to parse previous successful workflow runs: %w", err)
	}

	for index := range runs {
		if strings.HasPrefix(runs[index].WorkflowName, ".github/") {
			if displayName := resolveWorkflowDisplayName(runs[index].WorkflowPath, owner, repo, hostname); displayName != "" {
				runs[index].WorkflowName = displayName
			}
		}
	}

	return runs, nil
}

func buildAuditComparisonForRun(currentRun ProcessedRun, currentSnapshot auditComparisonSnapshot, outputDir string, owner, repo, hostname string, verbose bool) *AuditComparisonData {
	baselineRuns, err := findPreviousSuccessfulWorkflowRuns(currentRun.Run, owner, repo, hostname, verbose)
	if err != nil {
		auditLog.Printf("Skipping audit comparison: failed to find baseline: %v", err)
		return &AuditComparisonData{BaselineFound: false}
	}
	if len(baselineRuns) == 0 {
		return &AuditComparisonData{BaselineFound: false}
	}

	candidates := make([]auditComparisonCandidate, 0, len(baselineRuns))
	for _, baselineRun := range baselineRuns {
		baselineOutputDir := filepath.Join(outputDir, fmt.Sprintf("baseline-%d", baselineRun.DatabaseID))
		if _, err := os.Stat(baselineOutputDir); err != nil {
			if downloadErr := downloadRunArtifacts(baselineRun.DatabaseID, baselineOutputDir, verbose, owner, repo, hostname); downloadErr != nil {
				auditLog.Printf("Skipping candidate baseline for run %d: failed to download baseline artifacts: %v", baselineRun.DatabaseID, downloadErr)
				continue
			}
		}

		candidate, candidateErr := loadAuditComparisonCandidate(baselineRun, baselineOutputDir, verbose)
		if candidateErr != nil {
			auditLog.Printf("Skipping candidate baseline for run %d: failed to load baseline snapshot: %v", baselineRun.DatabaseID, candidateErr)
			continue
		}
		candidates = append(candidates, candidate)
	}

	selected := selectAuditComparisonBaseline(currentRun, candidates)
	if selected == nil {
		return &AuditComparisonData{BaselineFound: false}
	}

	comparison := buildAuditComparison(currentSnapshot, &selected.Run, &selected.Snapshot)
	if comparison != nil && comparison.Baseline != nil {
		comparison.Baseline.Selection = selected.Selection
		comparison.Baseline.MatchedOn = selected.MatchedOn
	}
	return comparison
}
