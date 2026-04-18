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

// renderPerformanceMetrics renders performance metrics
func renderPerformanceMetrics(metrics *PerformanceMetrics) {
	auditReportLog.Printf("Rendering performance metrics: tokens_per_min=%.1f, cost_efficiency=%s, most_used_tool=%s",
		metrics.TokensPerMinute, metrics.CostEfficiency, metrics.MostUsedTool)
	if metrics.TokensPerMinute > 0 {
		fmt.Fprintf(os.Stderr, "  Tokens per Minute: %.1f\n", metrics.TokensPerMinute)
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
		fmt.Fprintf(os.Stderr, "  Cost Efficiency: %s\n", efficiencyDisplay)
	}

	if metrics.AvgToolDuration != "" {
		fmt.Fprintf(os.Stderr, "  Average Tool Duration: %s\n", metrics.AvgToolDuration)
	}

	if metrics.MostUsedTool != "" {
		fmt.Fprintf(os.Stderr, "  Most Used Tool: %s\n", metrics.MostUsedTool)
	}

	if metrics.NetworkRequests > 0 {
		fmt.Fprintf(os.Stderr, "  Network Requests: %d\n", metrics.NetworkRequests)
	}

	fmt.Fprintln(os.Stderr)
}

// renderEngineConfig renders engine configuration details
func renderEngineConfig(config *AuditEngineConfig) {
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
		fmt.Fprintf(os.Stderr, "  Wall Time:              %s\n", session.WallTime)
	}
	if session.TurnCount > 0 {
		fmt.Fprintf(os.Stderr, "  Turn Count:             %d\n", session.TurnCount)
	}
	if session.AvgTurnDuration != "" {
		fmt.Fprintf(os.Stderr, "  Avg Turn Duration:      %s\n", session.AvgTurnDuration)
	}
	if session.AvgTimeBetweenTurns != "" {
		fmt.Fprintf(os.Stderr, "  Avg Time Between Turns: %s\n", session.AvgTimeBetweenTurns)
	}
	if session.MaxTimeBetweenTurns != "" {
		fmt.Fprintf(os.Stderr, "  Max Time Between Turns: %s\n", session.MaxTimeBetweenTurns)
	}
	if session.CacheWarning != "" {
		fmt.Fprintf(os.Stderr, "  Cache Warning:          %s\n", console.FormatWarningMessage(session.CacheWarning))
	}
	if session.TokensPerMinute > 0 {
		fmt.Fprintf(os.Stderr, "  Tokens/Minute:          %.1f\n", session.TokensPerMinute)
	}
	if session.NoopCount > 0 {
		fmt.Fprintf(os.Stderr, "  Noop Count:             %d\n", session.NoopCount)
	}
	if session.TimeoutDetected {
		fmt.Fprintf(os.Stderr, "  Timeout Detected:       %s\n", console.FormatWarningMessage("Yes"))
	} else {
		fmt.Fprintf(os.Stderr, "  Timeout Detected:       %s\n", console.FormatSuccessMessage("No"))
	}
	fmt.Fprintln(os.Stderr)
}
