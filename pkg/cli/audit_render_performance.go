package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/github/gh-aw/pkg/console"
)

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
