// This file provides command-line interface functionality for gh-aw.
// This file (audit_report_render_summary.go) contains console rendering functions
// for overview, comparison, task domain, behavior fingerprint, jobs table,
// key findings, recommendations, engine configuration, and session analysis
// sections of the audit report.

package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/sliceutil"
	"github.com/github/gh-aw/pkg/stringutil"
)

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

// renderOverview renders the overview section using the new rendering system
func renderOverview(overview OverviewData) {
	// Format Status with optional Conclusion
	statusLine := overview.Status
	if overview.Conclusion != "" && overview.Status == "completed" {
		statusLine = fmt.Sprintf("%s (%s)", overview.Status, overview.Conclusion)
	}

	display := OverviewDisplay{
		RunID:    overview.RunID,
		Workflow: overview.WorkflowName,
		Status:   statusLine,
		Duration: overview.Duration,
		Event:    overview.Event,
		Branch:   overview.Branch,
		URL:      overview.URL,
		Files:    overview.LogsPath,
	}

	fmt.Fprint(os.Stderr, console.RenderStruct(display))
}

// renderMetrics renders the metrics section using the new rendering system
func renderMetrics(metrics MetricsData) {
	fmt.Fprint(os.Stderr, console.RenderStruct(metrics))
}

type taskDomainDisplay struct {
	Domain string `console:"header:Domain"`
	Reason string `console:"header:Reason"`
}

type behaviorFingerprintDisplay struct {
	Execution string `console:"header:Execution"`
	Tools     string `console:"header:Tools"`
	Actuation string `console:"header:Actuation"`
	Resource  string `console:"header:Resources"`
	Dispatch  string `console:"header:Dispatch"`
}

func renderTaskDomain(domain *TaskDomainInfo) {
	if domain == nil {
		return
	}
	fmt.Fprint(os.Stderr, console.RenderStruct(taskDomainDisplay{
		Domain: domain.Label,
		Reason: domain.Reason,
	}))
}

func renderBehaviorFingerprint(fingerprint *BehaviorFingerprint) {
	if fingerprint == nil {
		return
	}
	fmt.Fprint(os.Stderr, console.RenderStruct(behaviorFingerprintDisplay{
		Execution: fingerprint.ExecutionStyle,
		Tools:     fingerprint.ToolBreadth,
		Actuation: fingerprint.ActuationStyle,
		Resource:  fingerprint.ResourceProfile,
		Dispatch:  fingerprint.DispatchMode,
	}))
}

func renderAgenticAssessments(assessments []AgenticAssessment) {
	for _, assessment := range assessments {
		severity := strings.ToUpper(assessment.Severity)
		fmt.Fprintf(os.Stderr, "  [%s] %s\n", severity, assessment.Summary)
		if assessment.Evidence != "" {
			fmt.Fprintf(os.Stderr, "     Evidence: %s\n", assessment.Evidence)
		}
		if assessment.Recommendation != "" {
			fmt.Fprintf(os.Stderr, "     Recommendation: %s\n", assessment.Recommendation)
		}
		fmt.Fprintln(os.Stderr)
	}
}

// renderJobsTable renders the jobs as a table using console.RenderTable
func renderJobsTable(jobs []JobData) {
	auditReportLog.Printf("Rendering jobs table with %d jobs", len(jobs))
	config := console.TableConfig{
		Headers: []string{"Name", "Status", "Conclusion", "Duration"},
		Rows:    make([][]string, 0, len(jobs)),
	}

	for _, job := range jobs {
		conclusion := job.Conclusion
		if conclusion == "" {
			conclusion = "-"
		}
		duration := job.Duration
		if duration == "" {
			duration = "-"
		}

		row := []string{
			stringutil.Truncate(job.Name, 40),
			job.Status,
			conclusion,
			duration,
		}
		config.Rows = append(config.Rows, row)
	}

	fmt.Fprint(os.Stderr, console.RenderTable(config))
}

// renderKeyFindings renders key findings with colored severity indicators
func renderKeyFindings(findings []Finding) {
	auditReportLog.Printf("Rendering key findings: total=%d", len(findings))
	// Group findings by severity for better presentation
	critical := sliceutil.Filter(findings, func(f Finding) bool { return f.Severity == "critical" })
	high := sliceutil.Filter(findings, func(f Finding) bool { return f.Severity == "high" })
	medium := sliceutil.Filter(findings, func(f Finding) bool { return f.Severity == "medium" })
	low := sliceutil.Filter(findings, func(f Finding) bool { return f.Severity == "low" })
	info := sliceutil.Filter(findings, func(f Finding) bool {
		return f.Severity != "critical" && f.Severity != "high" && f.Severity != "medium" && f.Severity != "low"
	})

	// Render critical findings first
	for _, finding := range critical {
		fmt.Fprintf(os.Stderr, "  🔴 %s [%s]\n", console.FormatErrorMessage(finding.Title), finding.Category)
		fmt.Fprintf(os.Stderr, "     %s\n", finding.Description)
		if finding.Impact != "" {
			fmt.Fprintf(os.Stderr, "     Impact: %s\n", finding.Impact)
		}
		fmt.Fprintln(os.Stderr)
	}

	// Then high severity
	for _, finding := range high {
		fmt.Fprintf(os.Stderr, "  🟠 %s [%s]\n", console.FormatWarningMessage(finding.Title), finding.Category)
		fmt.Fprintf(os.Stderr, "     %s\n", finding.Description)
		if finding.Impact != "" {
			fmt.Fprintf(os.Stderr, "     Impact: %s\n", finding.Impact)
		}
		fmt.Fprintln(os.Stderr)
	}

	// Medium severity
	for _, finding := range medium {
		fmt.Fprintf(os.Stderr, "  🟡 %s [%s]\n", finding.Title, finding.Category)
		fmt.Fprintf(os.Stderr, "     %s\n", finding.Description)
		if finding.Impact != "" {
			fmt.Fprintf(os.Stderr, "     Impact: %s\n", finding.Impact)
		}
		fmt.Fprintln(os.Stderr)
	}

	// Low severity
	for _, finding := range low {
		fmt.Fprintf(os.Stderr, "  ℹ️  %s [%s]\n", finding.Title, finding.Category)
		fmt.Fprintf(os.Stderr, "     %s\n", finding.Description)
		if finding.Impact != "" {
			fmt.Fprintf(os.Stderr, "     Impact: %s\n", finding.Impact)
		}
		fmt.Fprintln(os.Stderr)
	}

	// Info findings
	for _, finding := range info {
		fmt.Fprintf(os.Stderr, "  ✅ %s [%s]\n", console.FormatSuccessMessage(finding.Title), finding.Category)
		fmt.Fprintf(os.Stderr, "     %s\n", finding.Description)
		if finding.Impact != "" {
			fmt.Fprintf(os.Stderr, "     Impact: %s\n", finding.Impact)
		}
		fmt.Fprintln(os.Stderr)
	}
}

// renderRecommendations renders actionable recommendations
func renderRecommendations(recommendations []Recommendation) {
	auditReportLog.Printf("Rendering recommendations: total=%d", len(recommendations))
	// Group by priority
	high := sliceutil.Filter(recommendations, func(r Recommendation) bool { return r.Priority == "high" })
	medium := sliceutil.Filter(recommendations, func(r Recommendation) bool { return r.Priority == "medium" })
	low := sliceutil.Filter(recommendations, func(r Recommendation) bool { return r.Priority != "high" && r.Priority != "medium" })

	// Render high priority first
	for i, rec := range high {
		fmt.Fprintf(os.Stderr, "  %d. [HIGH] %s\n", i+1, console.FormatWarningMessage(rec.Action))
		fmt.Fprintf(os.Stderr, "     Reason: %s\n", rec.Reason)
		if rec.Example != "" {
			fmt.Fprintf(os.Stderr, "     Example: %s\n", rec.Example)
		}
		fmt.Fprintln(os.Stderr)
	}

	// Medium priority
	startIdx := len(high) + 1
	for i, rec := range medium {
		fmt.Fprintf(os.Stderr, "  %d. [MEDIUM] %s\n", startIdx+i, rec.Action)
		fmt.Fprintf(os.Stderr, "     Reason: %s\n", rec.Reason)
		if rec.Example != "" {
			fmt.Fprintf(os.Stderr, "     Example: %s\n", rec.Example)
		}
		fmt.Fprintln(os.Stderr)
	}

	// Low priority
	startIdx += len(medium)
	for i, rec := range low {
		fmt.Fprintf(os.Stderr, "  %d. [LOW] %s\n", startIdx+i, rec.Action)
		fmt.Fprintf(os.Stderr, "     Reason: %s\n", rec.Reason)
		if rec.Example != "" {
			fmt.Fprintf(os.Stderr, "     Example: %s\n", rec.Example)
		}
		fmt.Fprintln(os.Stderr)
	}
}

// renderEngineConfig renders engine configuration details
func renderEngineConfig(config *EngineConfig) {
	if config == nil {
		return
	}
	fmt.Fprintf(os.Stderr, "  Engine ID:         %s\n", config.EngineID)
	if config.EngineName != "" {
		fmt.Fprintf(os.Stderr, "  Engine Name:       %s\n", config.EngineName)
	}
	if config.Model != "" {
		fmt.Fprintf(os.Stderr, "  Model:             %s\n", config.Model)
	}
	if config.Version != "" {
		fmt.Fprintf(os.Stderr, "  Version:           %s\n", config.Version)
	}
	if config.CLIVersion != "" {
		fmt.Fprintf(os.Stderr, "  CLI Version:       %s\n", config.CLIVersion)
	}
	if config.FirewallVersion != "" {
		fmt.Fprintf(os.Stderr, "  Firewall Version:  %s\n", config.FirewallVersion)
	}
	if config.TriggerEvent != "" {
		fmt.Fprintf(os.Stderr, "  Trigger Event:     %s\n", config.TriggerEvent)
	}
	if config.Repository != "" {
		fmt.Fprintf(os.Stderr, "  Repository:        %s\n", config.Repository)
	}
	if len(config.MCPServers) > 0 {
		fmt.Fprintf(os.Stderr, "  MCP Servers:       %s\n", strings.Join(config.MCPServers, ", "))
	}
	fmt.Fprintln(os.Stderr)
}

// renderPromptAnalysis renders prompt analysis metrics
func renderPromptAnalysis(analysis *PromptAnalysis) {
	if analysis == nil {
		return
	}
	fmt.Fprintf(os.Stderr, "  Prompt Size:       %s chars\n", console.FormatNumber(analysis.PromptSize))
	if analysis.PromptFile != "" {
		fmt.Fprintf(os.Stderr, "  Prompt File:       %s\n", analysis.PromptFile)
	}
	fmt.Fprintln(os.Stderr)
}

// renderSessionAnalysis renders session and agent performance metrics
func renderSessionAnalysis(session *SessionAnalysis) {
	if session == nil {
		return
	}
	if session.WallTime != "" {
		fmt.Fprintf(os.Stderr, "  Wall Time:         %s\n", session.WallTime)
	}
	if session.TurnCount > 0 {
		fmt.Fprintf(os.Stderr, "  Turn Count:        %d\n", session.TurnCount)
	}
	if session.AvgTurnDuration != "" {
		fmt.Fprintf(os.Stderr, "  Avg Turn Duration: %s\n", session.AvgTurnDuration)
	}
	if session.TokensPerMinute > 0 {
		fmt.Fprintf(os.Stderr, "  Tokens/Minute:     %.1f\n", session.TokensPerMinute)
	}
	if session.NoopCount > 0 {
		fmt.Fprintf(os.Stderr, "  Noop Count:        %d\n", session.NoopCount)
	}
	if session.TimeoutDetected {
		fmt.Fprintf(os.Stderr, "  Timeout Detected:  %s\n", console.FormatWarningMessage("Yes"))
	} else {
		fmt.Fprintf(os.Stderr, "  Timeout Detected:  %s\n", console.FormatSuccessMessage("No"))
	}
	fmt.Fprintln(os.Stderr)
}
