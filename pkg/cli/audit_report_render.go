package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
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

// renderConsole outputs the audit data in a compact, high-density format optimized
// for agentic readability. Each line carries maximum information with minimal decoration.
func renderConsole(data AuditData, logsPath string) {
	auditReportLog.Print("Rendering compact audit report to console")

	// Line 1: Identity + outcome
	statusIcon := "✅"
	switch data.Overview.Conclusion {
	case "failure":
		statusIcon = "❌"
	case "cancelled":
		statusIcon = "⚠️"
	}
	fmt.Fprintf(os.Stderr, "%s %s | %s | %s | %s\n",
		statusIcon, data.Overview.WorkflowName, data.Overview.Conclusion,
		data.Overview.Duration, data.Overview.URL)

	// Line 2: Context
	engineInfo := ""
	if data.EngineConfig != nil {
		parts := []string{data.EngineConfig.EngineID}
		if data.EngineConfig.Model != "" {
			parts = append(parts, data.EngineConfig.Model)
		}
		if data.EngineConfig.Version != "" {
			parts = append(parts, "v"+data.EngineConfig.Version)
		}
		engineInfo = strings.Join(parts, "/")
	}
	fmt.Fprintf(os.Stderr, "  run=%d branch=%s event=%s engine=%s\n",
		data.Overview.RunID, data.Overview.Branch, data.Overview.Event, engineInfo)

	// Line 3: Comparison (if available)
	if data.Comparison != nil && data.Comparison.BaselineFound {
		compLine := "  comparison:"
		if data.Comparison.Classification != nil {
			compLine += " " + data.Comparison.Classification.Label
		}
		if data.Comparison.Baseline != nil {
			compLine += fmt.Sprintf(" vs baseline %d", data.Comparison.Baseline.RunID)
		}
		if data.Comparison.Recommendation != nil && data.Comparison.Recommendation.Action != "" {
			compLine += " | " + data.Comparison.Recommendation.Action
		}
		fmt.Fprintln(os.Stderr, compLine)
	}

	// Line 4: Fingerprint (if available)
	if data.BehaviorFingerprint != nil {
		fmt.Fprintf(os.Stderr, "  fingerprint: %s/%s/%s/%s/%s\n",
			data.BehaviorFingerprint.ExecutionStyle,
			data.BehaviorFingerprint.ToolBreadth,
			data.BehaviorFingerprint.ActuationStyle,
			data.BehaviorFingerprint.ResourceProfile,
			data.BehaviorFingerprint.DispatchMode)
	}

	// Line 5: Metrics (always present)
	metricsLine := fmt.Sprintf("  metrics: errors=%d warnings=%d",
		data.Metrics.ErrorCount, data.Metrics.WarningCount)
	if data.Metrics.Turns > 0 {
		metricsLine += fmt.Sprintf(" turns=%d", data.Metrics.Turns)
	}
	if data.Metrics.TokenUsage > 0 {
		metricsLine += " tokens=" + console.FormatNumber(data.Metrics.TokenUsage)
	}
	if data.Metrics.AIC > 0 {
		metricsLine += fmt.Sprintf(" aic=%.2f", data.Metrics.AIC)
	}
	if data.Metrics.ActionMinutes > 0 {
		metricsLine += fmt.Sprintf(" action_min=%.0f", data.Metrics.ActionMinutes)
	}
	fmt.Fprintln(os.Stderr, metricsLine)

	// Line 6: Session performance (if present)
	if data.SessionAnalysis != nil {
		sessionLine := "  session:"
		if data.SessionAnalysis.WallTime != "" {
			sessionLine += " wall=" + data.SessionAnalysis.WallTime
		}
		if data.SessionAnalysis.TurnCount > 0 {
			sessionLine += fmt.Sprintf(" turns=%d", data.SessionAnalysis.TurnCount)
		}
		if data.SessionAnalysis.TokensPerMinute > 0 {
			sessionLine += fmt.Sprintf(" tok/min=%.0f", data.SessionAnalysis.TokensPerMinute)
		}
		if data.SessionAnalysis.TimeoutDetected {
			sessionLine += " TIMEOUT"
		}
		if data.SessionAnalysis.NoopCount > 0 {
			sessionLine += fmt.Sprintf(" noops=%d", data.SessionAnalysis.NoopCount)
		}
		fmt.Fprintln(os.Stderr, sessionLine)
	}

	// Token usage (if firewall data present)
	if data.FirewallTokenUsage != nil && data.FirewallTokenUsage.TotalRequests > 0 {
		fmt.Fprintf(os.Stderr, "  tokens: in=%s out=%s cache_read=%s reqs=%d steering=%s\n",
			console.FormatNumber(data.FirewallTokenUsage.TotalInputTokens),
			console.FormatNumber(data.FirewallTokenUsage.TotalOutputTokens),
			console.FormatNumber(data.FirewallTokenUsage.TotalCacheReadTokens),
			data.FirewallTokenUsage.TotalRequests,
			console.FormatNumber(data.FirewallTokenUsage.TotalSteeringEvents))
	}

	// GitHub API usage (one line)
	if data.GitHubRateLimitUsage != nil {
		fmt.Fprintf(os.Stderr, "  github_api: calls=%s quota=%s/%s\n",
			console.FormatNumber(data.GitHubRateLimitUsage.TotalRequestsMade),
			console.FormatNumber(data.GitHubRateLimitUsage.CoreConsumed),
			console.FormatNumber(data.GitHubRateLimitUsage.CoreLimit))
	}

	// Jobs (compact: one line if all pass, table if failures)
	if len(data.Jobs) > 0 {
		allPassed := true
		jobParts := make([]string, 0, len(data.Jobs))
		for _, job := range data.Jobs {
			if job.Conclusion != "success" && job.Conclusion != "skipped" {
				allPassed = false
			}
			jobParts = append(jobParts, fmt.Sprintf("%s:%s", job.Name, job.Duration))
		}
		if allPassed {
			fmt.Fprintf(os.Stderr, "  jobs: %d/%d passed [%s]\n", len(data.Jobs), len(data.Jobs), strings.Join(jobParts, " "))
		} else {
			fmt.Fprintf(os.Stderr, "  jobs:\n")
			for _, job := range data.Jobs {
				icon := "✓"
				switch job.Conclusion {
				case "failure":
					icon = "✗"
				case "skipped":
					icon = "○"
				case "cancelled":
					icon = "⊘"
				}
				fmt.Fprintf(os.Stderr, "    %s %s (%s) %s\n", icon, job.Name, job.Duration, job.Conclusion)
			}
		}
	}

	// Prompt info (one line)
	if data.PromptAnalysis != nil {
		promptLine := fmt.Sprintf("  prompt: %s chars", console.FormatNumber(data.PromptAnalysis.PromptSize))
		if data.PromptAnalysis.PromptFile != "" {
			promptLine += " file=" + data.PromptAnalysis.PromptFile
		}
		fmt.Fprintln(os.Stderr, promptLine)
	}

	// --- Actionable sections below (only rendered when non-trivial) ---

	// Key Findings: only show non-success findings in compact form
	actionableFindings := filterActionableFindings(data.KeyFindings)
	if len(actionableFindings) > 0 {
		fmt.Fprintln(os.Stderr, "  findings:")
		for _, f := range actionableFindings {
			fmt.Fprintf(os.Stderr, "    [%s] %s: %s\n", strings.ToUpper(f.Severity), f.Title, f.Description)
		}
	}

	// Agentic Assessments (compact)
	if len(data.AgenticAssessments) > 0 {
		fmt.Fprintln(os.Stderr, "  assessments:")
		for _, a := range data.AgenticAssessments {
			line := fmt.Sprintf("    [%s] %s", strings.ToUpper(a.Severity), a.Summary)
			if a.Evidence != "" {
				line += " | " + a.Evidence
			}
			fmt.Fprintln(os.Stderr, line)
		}
	}

	// Recommendations: only high/medium
	actionableRecs := filterActionableRecommendations(data.Recommendations)
	if len(actionableRecs) > 0 {
		fmt.Fprintln(os.Stderr, "  recommendations:")
		for _, r := range actionableRecs {
			fmt.Fprintf(os.Stderr, "    [%s] %s — %s\n", strings.ToUpper(r.Priority), r.Action, r.Reason)
		}
	}

	// Observability Insights: only non-info severity
	actionableInsights := filterActionableInsights(data.ObservabilityInsights)
	if len(actionableInsights) > 0 {
		fmt.Fprintln(os.Stderr, "  insights:")
		for _, ins := range actionableInsights {
			line := fmt.Sprintf("    [%s] %s", strings.ToUpper(ins.Severity), ins.Title)
			if ins.Evidence != "" {
				line += " | " + ins.Evidence
			}
			fmt.Fprintln(os.Stderr, line)
		}
	}

	// Errors and Warnings (always show if present)
	if len(data.Errors) > 0 {
		fmt.Fprintln(os.Stderr, "  errors:")
		for _, err := range data.Errors {
			if err.File != "" && err.Line > 0 {
				fmt.Fprintf(os.Stderr, "    %s:%d: %s\n", filepath.Base(err.File), err.Line, err.Message)
			} else {
				fmt.Fprintf(os.Stderr, "    %s\n", err.Message)
			}
		}
	}
	if len(data.Warnings) > 0 {
		fmt.Fprintln(os.Stderr, "  warnings:")
		for _, w := range data.Warnings {
			fmt.Fprintf(os.Stderr, "    %s\n", w.Message)
		}
	}

	// Missing Tools (actionable)
	if len(data.MissingTools) > 0 {
		fmt.Fprintln(os.Stderr, "  missing_tools:")
		for _, tool := range data.MissingTools {
			line := "    " + tool.Tool + ": " + tool.Reason
			if tool.Alternatives != "" {
				line += " (alt: " + tool.Alternatives + ")"
			}
			fmt.Fprintln(os.Stderr, line)
		}
	}

	// MCP Failures (actionable)
	if len(data.MCPFailures) > 0 {
		fmt.Fprintln(os.Stderr, "  mcp_failures:")
		for _, f := range data.MCPFailures {
			fmt.Fprintf(os.Stderr, "    %s: %s\n", f.ServerName, f.Status)
		}
	}

	// MCP Server Health (only if issues)
	if data.MCPServerHealth != nil {
		renderCompactMCPHealth(data.MCPServerHealth)
	}

	// Safe Output Summary (compact)
	if data.SafeOutputSummary != nil && data.SafeOutputSummary.TotalItems > 0 {
		fmt.Fprintf(os.Stderr, "  safe_outputs: %d items — %s\n",
			data.SafeOutputSummary.TotalItems, data.SafeOutputSummary.Summary)
	}

	// Created Items (compact)
	if len(data.CreatedItems) > 0 {
		fmt.Fprintln(os.Stderr, "  created:")
		for _, item := range data.CreatedItems {
			line := "    " + item.Type
			if item.URL != "" {
				line += " " + item.URL
			} else if item.Repo != "" && item.Number > 0 {
				line += fmt.Sprintf(" %s#%d", item.Repo, item.Number)
			}
			fmt.Fprintln(os.Stderr, line)
		}
	}

	// Tool Usage (compact table only when tools were used)
	if len(data.ToolUsage) > 0 {
		fmt.Fprintln(os.Stderr, "  tools:")
		for _, tool := range data.ToolUsage {
			line := fmt.Sprintf("    %s ×%d", tool.Name, tool.CallCount)
			if tool.MaxDuration != "" {
				line += " max=" + tool.MaxDuration
			}
			fmt.Fprintln(os.Stderr, line)
		}
	}

	// MCP Tool Usage (compact)
	if data.MCPToolUsage != nil && len(data.MCPToolUsage.Summary) > 0 {
		fmt.Fprintln(os.Stderr, "  mcp_tools:")
		for _, s := range data.MCPToolUsage.Summary {
			line := fmt.Sprintf("    %s/%s ×%d", s.ServerName, s.ToolName, s.CallCount)
			if s.ErrorCount > 0 {
				line += fmt.Sprintf(" errors=%d", s.ErrorCount)
			}
			if s.MaxDuration != "" {
				line += " max=" + s.MaxDuration
			}
			fmt.Fprintln(os.Stderr, line)
		}
		// Guard policy (if present)
		if data.MCPToolUsage.GuardPolicySummary != nil && data.MCPToolUsage.GuardPolicySummary.TotalBlocked > 0 {
			fmt.Fprintf(os.Stderr, "    guard_blocked: %d\n", data.MCPToolUsage.GuardPolicySummary.TotalBlocked)
		}
	}

	// Firewall Analysis (compact)
	if data.FirewallAnalysis != nil && data.FirewallAnalysis.TotalRequests > 0 {
		renderCompactFirewall(data.FirewallAnalysis)
	}

	// Policy Analysis (compact)
	if data.PolicyAnalysis != nil && len(data.PolicyAnalysis.RuleHits) > 0 {
		fmt.Fprintf(os.Stderr, "  firewall_policy: %s\n", data.PolicyAnalysis.PolicySummary)
	}

	// Experiments
	if data.Experiments != nil && len(data.Experiments.Assignments) > 0 {
		parts := make([]string, 0, len(data.Experiments.Assignments))
		for name, variant := range data.Experiments.Assignments {
			parts = append(parts, name+"="+variant)
		}
		sort.Strings(parts)
		fmt.Fprintf(os.Stderr, "  experiments: %s\n", strings.Join(parts, " "))
	}

	// Logs path (final line)
	absPath, _ := filepath.Abs(logsPath)
	fmt.Fprintf(os.Stderr, "  logs: %s\n", absPath)
}

// filterActionableFindings returns findings with severity > info/success
func filterActionableFindings(findings []Finding) []Finding {
	var result []Finding
	for _, f := range findings {
		if f.Severity == "critical" || f.Severity == "high" || f.Severity == "medium" || f.Severity == "low" {
			result = append(result, f)
		}
	}
	return result
}

// filterActionableRecommendations returns high/medium priority recommendations
func filterActionableRecommendations(recs []Recommendation) []Recommendation {
	var result []Recommendation
	for _, r := range recs {
		if r.Priority == "high" || r.Priority == "medium" {
			result = append(result, r)
		}
	}
	return result
}

// filterActionableInsights returns insights with severity > info
func filterActionableInsights(insights []ObservabilityInsight) []ObservabilityInsight {
	var result []ObservabilityInsight
	for _, ins := range insights {
		if ins.Severity == "critical" || ins.Severity == "high" || ins.Severity == "medium" || ins.Severity == "low" {
			result = append(result, ins)
		}
	}
	return result
}

// renderCompactMCPHealth renders MCP health issues in compact form (only problems)
func renderCompactMCPHealth(health *MCPServerHealth) {
	if health == nil {
		return
	}
	// Only render if there are unhealthy servers
	hasIssues := false
	for _, server := range health.Servers {
		if server.Status != "healthy" {
			hasIssues = true
			break
		}
	}
	if !hasIssues {
		return
	}
	fmt.Fprintln(os.Stderr, "  mcp_health:")
	for _, server := range health.Servers {
		if server.Status != "healthy" {
			fmt.Fprintf(os.Stderr, "    %s: %s\n", server.ServerName, server.Status)
		}
	}
}

// renderCompactFirewall renders firewall analysis in compact form
func renderCompactFirewall(fa *FirewallAnalysis) {
	if fa == nil {
		return
	}
	line := fmt.Sprintf("  firewall: %d requests", fa.TotalRequests)
	if len(fa.BlockedDomains) > 0 {
		line += fmt.Sprintf(" blocked_domains=%d", len(fa.BlockedDomains))
	}
	fmt.Fprintln(os.Stderr, line)
}
