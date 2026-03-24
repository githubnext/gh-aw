package cli

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
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
	RunID        int64  `json:"run_id"`
	WorkflowName string `json:"workflow_name,omitempty"`
	Conclusion   string `json:"conclusion,omitempty"`
	CreatedAt    string `json:"created_at,omitempty"`
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
		return "No action needed; this run matches the last successful baseline closely."
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
		return "Compare prompt or task-shape changes because this run needed more turns than the last successful baseline."
	}

	return "Review the behavior change against the previous successful run before treating it as the new normal."
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

func findPreviousSuccessfulWorkflowRun(current WorkflowRun, owner, repo, hostname string, verbose bool) (*WorkflowRun, error) {
	workflowID := filepath.Base(current.WorkflowPath)
	if workflowID == "." || workflowID == "" {
		return nil, fmt.Errorf("workflow path unavailable for run %d", current.DatabaseID)
	}

	encodedWorkflowID := url.PathEscape(workflowID)
	var endpoint string
	if owner != "" && repo != "" {
		endpoint = fmt.Sprintf("repos/%s/%s/actions/workflows/%s/runs?per_page=50", owner, repo, encodedWorkflowID)
	} else {
		endpoint = fmt.Sprintf("repos/{owner}/{repo}/actions/workflows/%s/runs?per_page=50", encodedWorkflowID)
	}

	jq := fmt.Sprintf(`[.workflow_runs[] | select(.id != %d and .conclusion == "success" and .created_at < "%s") | {databaseId: .id, number: .run_number, url: .html_url, status: .status, conclusion: .conclusion, workflowName: .name, workflowPath: .path, createdAt: .created_at, startedAt: .run_started_at, updatedAt: .updated_at, event: .event, headBranch: .head_branch, headSha: .head_sha, displayTitle: .display_title}] | .[0]`, current.DatabaseID, current.CreatedAt.Format("2006-01-02T15:04:05Z07:00"))

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
	if trimmed == "null" || trimmed == "" {
		return nil, nil
	}

	var run WorkflowRun
	if err := json.Unmarshal(output, &run); err != nil {
		return nil, fmt.Errorf("failed to parse previous successful workflow run: %w", err)
	}

	if strings.HasPrefix(run.WorkflowName, ".github/") {
		if displayName := resolveWorkflowDisplayName(run.WorkflowPath, owner, repo, hostname); displayName != "" {
			run.WorkflowName = displayName
		}
	}

	return &run, nil
}

func buildAuditComparisonForRun(currentRun WorkflowRun, currentSnapshot auditComparisonSnapshot, outputDir string, owner, repo, hostname string, verbose bool) *AuditComparisonData {
	baselineRun, err := findPreviousSuccessfulWorkflowRun(currentRun, owner, repo, hostname, verbose)
	if err != nil {
		auditLog.Printf("Skipping audit comparison: failed to find baseline: %v", err)
		return &AuditComparisonData{BaselineFound: false}
	}
	if baselineRun == nil {
		return &AuditComparisonData{BaselineFound: false}
	}

	baselineOutputDir := filepath.Join(outputDir, fmt.Sprintf("baseline-%d", baselineRun.DatabaseID))
	if _, err := os.Stat(baselineOutputDir); err != nil {
		if downloadErr := downloadRunArtifacts(baselineRun.DatabaseID, baselineOutputDir, verbose, owner, repo, hostname); downloadErr != nil {
			auditLog.Printf("Skipping baseline comparison for run %d: failed to download baseline artifacts: %v", baselineRun.DatabaseID, downloadErr)
			return &AuditComparisonData{BaselineFound: false}
		}
	}

	baselineSnapshot, err := loadAuditComparisonSnapshotFromArtifacts(*baselineRun, baselineOutputDir, verbose)
	if err != nil {
		auditLog.Printf("Skipping baseline comparison for run %d: failed to load baseline snapshot: %v", baselineRun.DatabaseID, err)
		return &AuditComparisonData{BaselineFound: false}
	}

	return buildAuditComparison(currentSnapshot, baselineRun, &baselineSnapshot)
}
