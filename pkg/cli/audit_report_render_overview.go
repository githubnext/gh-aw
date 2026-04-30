package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/github/gh-aw/pkg/console"
)

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
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("  [%s] %s", severity, assessment.Summary)))
		if assessment.Evidence != "" {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("     Evidence: %s", assessment.Evidence)))
		}
		if assessment.Recommendation != "" {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("     Recommendation: %s", assessment.Recommendation)))
		}
		fmt.Fprintln(os.Stderr)
	}
}

// renderPerformanceMetrics renders performance metrics
func renderPerformanceMetrics(metrics *PerformanceMetrics) {
	auditReportLog.Printf("Rendering performance metrics: tokens_per_min=%.1f, cost_efficiency=%s, most_used_tool=%s",
		metrics.TokensPerMinute, metrics.CostEfficiency, metrics.MostUsedTool)
	if metrics.TokensPerMinute > 0 {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("  Tokens per Minute: %.1f", metrics.TokensPerMinute)))
	}

	if metrics.CostEfficiency != "" {
		efficiencyDisplay := metrics.CostEfficiency
		switch metrics.CostEfficiency {
		case "excellent", "good":
			efficiencyDisplay = console.FormatSuccessMessage(metrics.CostEfficiency)
		case "moderate":
			efficiencyDisplay = console.FormatWarningMessage(metrics.CostEfficiency)
		case "poor":
			efficiencyDisplay = console.FormatErrorMessage(metrics.CostEfficiency)
		}
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("  Cost Efficiency: %s", efficiencyDisplay)))
	}

	if metrics.AvgToolDuration != "" {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("  Average Tool Duration: %s", metrics.AvgToolDuration)))
	}

	if metrics.MostUsedTool != "" {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("  Most Used Tool: %s", metrics.MostUsedTool)))
	}

	if metrics.NetworkRequests > 0 {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("  Network Requests: %d", metrics.NetworkRequests)))
	}

	fmt.Fprintln(os.Stderr)
}

// renderEngineConfig renders engine configuration details
func renderEngineConfig(config *AuditEngineConfig) {
	if config == nil {
		return
	}
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("  Engine ID:         %s", config.EngineID)))
	if config.EngineName != "" {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("  Engine Name:       %s", config.EngineName)))
	}
	if config.Model != "" {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("  Model:             %s", config.Model)))
	}
	if config.Version != "" {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("  Version:           %s", config.Version)))
	}
	if config.CLIVersion != "" {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("  CLI Version:       %s", config.CLIVersion)))
	}
	if config.FirewallVersion != "" {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("  Firewall Version:  %s", config.FirewallVersion)))
	}
	if config.TriggerEvent != "" {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("  Trigger Event:     %s", config.TriggerEvent)))
	}
	if config.Repository != "" {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("  Repository:        %s", config.Repository)))
	}
	if len(config.MCPServers) > 0 {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("  MCP Servers:       %s", strings.Join(config.MCPServers, ", "))))
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
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("  Prompt File:       %s", analysis.PromptFile)))
	}
	fmt.Fprintln(os.Stderr)
}

// renderSessionAnalysis renders session and agent performance metrics
func renderSessionAnalysis(session *SessionAnalysis) {
	if session == nil {
		return
	}
	if session.WallTime != "" {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("  Wall Time:              %s", session.WallTime)))
	}
	if session.TurnCount > 0 {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("  Turn Count:             %d", session.TurnCount)))
	}
	if session.AvgTurnDuration != "" {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("  Avg Turn Duration:      %s", session.AvgTurnDuration)))
	}
	if session.AvgTimeBetweenTurns != "" {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("  Avg Time Between Turns: %s", session.AvgTimeBetweenTurns)))
	}
	if session.MaxTimeBetweenTurns != "" {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("  Max Time Between Turns: %s", session.MaxTimeBetweenTurns)))
	}
	if session.CacheWarning != "" {
		fmt.Fprintf(os.Stderr, "  Cache Warning:          %s\n", console.FormatWarningMessage(session.CacheWarning))
	}
	if session.TokensPerMinute > 0 {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("  Tokens/Minute:          %.1f", session.TokensPerMinute)))
	}
	if session.NoopCount > 0 {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("  Noop Count:             %d", session.NoopCount)))
	}
	if session.TimeoutDetected {
		fmt.Fprintf(os.Stderr, "  Timeout Detected:       %s\n", console.FormatWarningMessage("Yes"))
	} else {
		fmt.Fprintf(os.Stderr, "  Timeout Detected:       %s\n", console.FormatSuccessMessage("No"))
	}
	fmt.Fprintln(os.Stderr)
}
