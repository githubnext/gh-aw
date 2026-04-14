package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/github/gh-aw/pkg/console"
)

// renderJSON outputs the audit data as JSON
func renderJSON(data AuditData) error {
	auditReportLog.Print("Rendering audit report as JSON")
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(data)
}

// renderConsole outputs the audit data as formatted console tables
func renderConsole(data AuditData, logsPath string) {
	auditReportLog.Print("Rendering audit report to console")
	fmt.Fprintln(os.Stderr, console.FormatSectionHeader("Workflow Run Audit Report"))
	fmt.Fprintln(os.Stderr)

	// Overview Section - use new rendering system
	fmt.Fprintln(os.Stderr, console.FormatSectionHeader("Overview"))
	fmt.Fprintln(os.Stderr)
	renderOverview(data.Overview)

	if data.Comparison != nil {
		fmt.Fprintln(os.Stderr, console.FormatSectionHeader("Comparison To Similar Successful Run"))
		fmt.Fprintln(os.Stderr)
		renderAuditComparison(data.Comparison)
	}

	if data.TaskDomain != nil {
		fmt.Fprintln(os.Stderr, console.FormatSectionHeader("Detected Task Domain"))
		fmt.Fprintln(os.Stderr)
		renderTaskDomain(data.TaskDomain)
	}

	if data.BehaviorFingerprint != nil {
		fmt.Fprintln(os.Stderr, console.FormatSectionHeader("Behavioral Fingerprint"))
		fmt.Fprintln(os.Stderr)
		renderBehaviorFingerprint(data.BehaviorFingerprint)
	}

	if len(data.AgenticAssessments) > 0 {
		fmt.Fprintln(os.Stderr, console.FormatSectionHeader("Agentic Assessment"))
		fmt.Fprintln(os.Stderr)
		renderAgenticAssessments(data.AgenticAssessments)
	}

	// Key Findings Section - NEW
	if len(data.KeyFindings) > 0 {
		auditReportLog.Printf("Rendering %d key findings", len(data.KeyFindings))
		fmt.Fprintln(os.Stderr, console.FormatSectionHeader("Key Findings"))
		fmt.Fprintln(os.Stderr)
		renderKeyFindings(data.KeyFindings)
	}

	// Recommendations Section - NEW
	if len(data.Recommendations) > 0 {
		auditReportLog.Printf("Rendering %d recommendations", len(data.Recommendations))
		fmt.Fprintln(os.Stderr, console.FormatSectionHeader("Recommendations"))
		fmt.Fprintln(os.Stderr)
		renderRecommendations(data.Recommendations)
	}

	if len(data.ObservabilityInsights) > 0 {
		fmt.Fprintln(os.Stderr, console.FormatSectionHeader("Observability Insights"))
		fmt.Fprintln(os.Stderr)
		renderObservabilityInsights(data.ObservabilityInsights)
	}

	// Performance Metrics Section - NEW
	if data.PerformanceMetrics != nil {
		fmt.Fprintln(os.Stderr, console.FormatSectionHeader("Performance Metrics"))
		fmt.Fprintln(os.Stderr)
		renderPerformanceMetrics(data.PerformanceMetrics)
	}

	// Token Usage Section (from firewall proxy)
	if data.FirewallTokenUsage != nil && data.FirewallTokenUsage.TotalRequests > 0 {
		fmt.Fprintln(os.Stderr, console.FormatSectionHeader("📊 Token Usage (Firewall Proxy)"))
		fmt.Fprintln(os.Stderr)
		renderTokenUsage(data.FirewallTokenUsage)
	}

	// GitHub API Rate Limit Usage Section
	if data.GitHubRateLimitUsage != nil {
		fmt.Fprintln(os.Stderr, console.FormatSectionHeader("🐙 GitHub API Usage"))
		fmt.Fprintln(os.Stderr)
		renderGitHubRateLimitUsage(data.GitHubRateLimitUsage)
	}

	// Engine Configuration Section
	if data.EngineConfig != nil {
		fmt.Fprintln(os.Stderr, console.FormatSectionHeader("Engine Configuration"))
		fmt.Fprintln(os.Stderr)
		renderEngineConfig(data.EngineConfig)
	}

	// Prompt Analysis Section
	if data.PromptAnalysis != nil {
		fmt.Fprintln(os.Stderr, console.FormatSectionHeader("Prompt Analysis"))
		fmt.Fprintln(os.Stderr)
		renderPromptAnalysis(data.PromptAnalysis)
	}

	// Session Analysis Section
	if data.SessionAnalysis != nil {
		fmt.Fprintln(os.Stderr, console.FormatSectionHeader("Session & Agent Performance"))
		fmt.Fprintln(os.Stderr)
		renderSessionAnalysis(data.SessionAnalysis)
	}

	// MCP Server Health Section
	if data.MCPServerHealth != nil {
		fmt.Fprintln(os.Stderr, console.FormatSectionHeader("MCP Server Health"))
		fmt.Fprintln(os.Stderr)
		renderMCPServerHealth(data.MCPServerHealth)
	}

	// Safe Output Summary Section
	if data.SafeOutputSummary != nil {
		fmt.Fprintln(os.Stderr, console.FormatSectionHeader("Safe Output Summary"))
		fmt.Fprintln(os.Stderr)
		renderSafeOutputSummary(data.SafeOutputSummary)
	}

	// Metrics Section - use new rendering system
	fmt.Fprintln(os.Stderr, console.FormatSectionHeader("Metrics"))
	fmt.Fprintln(os.Stderr)
	renderMetrics(data.Metrics)

	// Jobs Section - use new table rendering
	if len(data.Jobs) > 0 {
		auditReportLog.Printf("Rendering jobs table with %d jobs", len(data.Jobs))
		fmt.Fprintln(os.Stderr, console.FormatSectionHeader("Jobs"))
		fmt.Fprintln(os.Stderr)
		renderJobsTable(data.Jobs)
	}

	// Downloaded Files Section
	if len(data.DownloadedFiles) > 0 {
		fmt.Fprintln(os.Stderr, console.FormatSectionHeader("Downloaded Files"))
		fmt.Fprintln(os.Stderr)
		for _, file := range data.DownloadedFiles {
			formattedSize := console.FormatFileSize(file.Size)
			fmt.Fprintf(os.Stderr, "  • %s (%s)", file.Path, formattedSize)
			if file.Description != "" {
				fmt.Fprintf(os.Stderr, " - %s", file.Description)
			}
			fmt.Fprintln(os.Stderr)
		}
		fmt.Fprintln(os.Stderr)
	}

	// Missing Tools Section
	if len(data.MissingTools) > 0 {
		fmt.Fprintln(os.Stderr, console.FormatSectionHeader("Missing Tools"))
		fmt.Fprintln(os.Stderr)
		for _, tool := range data.MissingTools {
			fmt.Fprintf(os.Stderr, "  • %s\n", tool.Tool)
			fmt.Fprintf(os.Stderr, "    Reason: %s\n", tool.Reason)
			if tool.Alternatives != "" {
				fmt.Fprintf(os.Stderr, "    Alternatives: %s\n", tool.Alternatives)
			}
		}
		fmt.Fprintln(os.Stderr)
	}

	// Created Items Section - items created in GitHub by safe output handlers
	if len(data.CreatedItems) > 0 {
		fmt.Fprintln(os.Stderr, console.FormatSectionHeader("Created Items"))
		fmt.Fprintln(os.Stderr)
		renderCreatedItemsTable(data.CreatedItems)
	}

	// MCP Failures Section
	if len(data.MCPFailures) > 0 {
		fmt.Fprintln(os.Stderr, console.FormatSectionHeader("MCP Server Failures"))
		fmt.Fprintln(os.Stderr)
		for _, failure := range data.MCPFailures {
			fmt.Fprintf(os.Stderr, "  • %s: %s\n", failure.ServerName, failure.Status)
		}
		fmt.Fprintln(os.Stderr)
	}

	// Firewall Analysis Section
	if data.FirewallAnalysis != nil && data.FirewallAnalysis.TotalRequests > 0 {
		fmt.Fprintln(os.Stderr, console.FormatSectionHeader("Firewall Analysis"))
		fmt.Fprintln(os.Stderr)
		renderFirewallAnalysis(data.FirewallAnalysis)
	}

	// Firewall Policy Analysis Section (enriched with rule attribution)
	if data.PolicyAnalysis != nil && (len(data.PolicyAnalysis.RuleHits) > 0 || data.PolicyAnalysis.PolicySummary != "") {
		fmt.Fprintln(os.Stderr, console.FormatSectionHeader("Firewall Policy Analysis"))
		fmt.Fprintln(os.Stderr)
		renderPolicyAnalysis(data.PolicyAnalysis)
	}

	// Redacted Domains Section
	if data.RedactedDomainsAnalysis != nil && data.RedactedDomainsAnalysis.TotalDomains > 0 {
		fmt.Fprintln(os.Stderr, console.FormatSectionHeader("🔒 Redacted URL Domains"))
		fmt.Fprintln(os.Stderr)
		renderRedactedDomainsAnalysis(data.RedactedDomainsAnalysis)
	}

	// Tool Usage Section - use new table rendering
	if len(data.ToolUsage) > 0 {
		fmt.Fprintln(os.Stderr, console.FormatSectionHeader("Tool Usage"))
		fmt.Fprintln(os.Stderr)
		renderToolUsageTable(data.ToolUsage)
	}

	// MCP Tool Usage Section - detailed MCP statistics
	if data.MCPToolUsage != nil && len(data.MCPToolUsage.Summary) > 0 {
		fmt.Fprintln(os.Stderr, console.FormatSectionHeader("MCP Tool Usage"))
		fmt.Fprintln(os.Stderr)
		renderMCPToolUsageTable(data.MCPToolUsage)
	}

	// Errors and Warnings Section
	if len(data.Errors) > 0 || len(data.Warnings) > 0 {
		fmt.Fprintln(os.Stderr, console.FormatSectionHeader("Errors and Warnings"))
		fmt.Fprintln(os.Stderr)
		renderErrorsAndWarnings(data.Errors, data.Warnings)
	}

	// Location
	fmt.Fprintln(os.Stderr, console.FormatSectionHeader("Logs Location"))
	fmt.Fprintln(os.Stderr)
	absPath, _ := filepath.Abs(logsPath)
	fmt.Fprintf(os.Stderr, "  %s\n", absPath)
	fmt.Fprintln(os.Stderr)
}

func renderAuditComparison(comparison *AuditComparisonData) {
	if comparison == nil {
		return
	}

	if !comparison.BaselineFound || comparison.Baseline == nil || comparison.Delta == nil || comparison.Classification == nil {
		fmt.Fprintln(os.Stderr, "  No suitable successful run was available for baseline comparison.")
		fmt.Fprintln(os.Stderr)
		return
	}

	fmt.Fprintf(os.Stderr, "  Baseline: run %d", comparison.Baseline.RunID)
	if comparison.Baseline.Conclusion != "" {
		fmt.Fprintf(os.Stderr, " (%s)", comparison.Baseline.Conclusion)
	}
	fmt.Fprintln(os.Stderr)
	if comparison.Baseline.Selection != "" {
		fmt.Fprintf(os.Stderr, "  Selection: %s\n", strings.ReplaceAll(comparison.Baseline.Selection, "_", " "))
	}
	if len(comparison.Baseline.MatchedOn) > 0 {
		fmt.Fprintf(os.Stderr, "  Matched on: %s\n", strings.Join(comparison.Baseline.MatchedOn, ", "))
	}
	fmt.Fprintf(os.Stderr, "  Classification: %s\n", comparison.Classification.Label)
	fmt.Fprintln(os.Stderr, "  Changes:")

	if comparison.Delta.Turns.Changed {
		fmt.Fprintf(os.Stderr, "    - Turns: %d -> %d\n", comparison.Delta.Turns.Before, comparison.Delta.Turns.After)
	}
	if comparison.Delta.Posture.Changed {
		fmt.Fprintf(os.Stderr, "    - Posture: %s -> %s\n", comparison.Delta.Posture.Before, comparison.Delta.Posture.After)
	}
	if comparison.Delta.BlockedRequests.Changed {
		fmt.Fprintf(os.Stderr, "    - Blocked requests: %d -> %d\n", comparison.Delta.BlockedRequests.Before, comparison.Delta.BlockedRequests.After)
	}
	if comparison.Delta.MCPFailure != nil && comparison.Delta.MCPFailure.NewlyPresent {
		fmt.Fprintf(os.Stderr, "    - New MCP failure: %s\n", strings.Join(comparison.Delta.MCPFailure.After, ", "))
	}
	if len(comparison.Classification.ReasonCodes) == 0 {
		fmt.Fprintln(os.Stderr, "    - No meaningful behavior change from the selected successful baseline")
	}
	if comparison.Recommendation != nil && comparison.Recommendation.Action != "" {
		fmt.Fprintf(os.Stderr, "  Recommended action: %s\n", comparison.Recommendation.Action)
	}
	fmt.Fprintln(os.Stderr)
}
