package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/logger"
)

var auditDiffRenderLog = logger.New("cli:audit_diff_render")

// renderAuditDiffJSON outputs the full audit diff as JSON to stdout
func renderAuditDiffJSON(diff *AuditDiff) error {
	auditDiffRenderLog.Printf("Rendering audit diff as JSON: run1=%d, run2=%d", diff.Run1ID, diff.Run2ID)
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(diff)
}

// renderAuditDiffMarkdown outputs the full audit diff as markdown to stdout
func renderAuditDiffMarkdown(diff *AuditDiff) {
	auditDiffRenderLog.Printf("Rendering audit diff as markdown: run1=%d, run2=%d", diff.Run1ID, diff.Run2ID)
	fmt.Printf("### Audit Diff: Run #%d → Run #%d\n\n", diff.Run1ID, diff.Run2ID)

	if isEmptyAuditDiff(diff) {
		fmt.Println("No behavioral changes detected between the two runs.")
		return
	}

	renderFirewallDiffMarkdownSection(diff.FirewallDiff)
	renderMCPToolsDiffMarkdownSection(diff.MCPToolsDiff)
	renderRunMetricsDiffMarkdownSection(diff.Run1ID, diff.Run2ID, diff.RunMetricsDiff)
}

// renderAuditDiffPretty outputs the full audit diff as formatted console output to stderr
func renderAuditDiffPretty(diff *AuditDiff) {
	auditDiffRenderLog.Printf("Rendering audit diff as pretty output: run1=%d, run2=%d", diff.Run1ID, diff.Run2ID)
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Audit Diff: Run #%d → Run #%d", diff.Run1ID, diff.Run2ID)))
	fmt.Fprintln(os.Stderr)

	if isEmptyAuditDiff(diff) {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("No behavioral changes detected between the two runs."))
		return
	}

	// Collect top-level summary across all sections
	var summaryParts []string
	anomalyCount := 0

	if diff.FirewallDiff != nil && !isEmptyFirewallDiff(diff.FirewallDiff) {
		fwParts := []string{}
		if len(diff.FirewallDiff.NewDomains) > 0 {
			fwParts = append(fwParts, fmt.Sprintf("%d new domains", len(diff.FirewallDiff.NewDomains)))
		}
		if len(diff.FirewallDiff.RemovedDomains) > 0 {
			fwParts = append(fwParts, fmt.Sprintf("%d removed domains", len(diff.FirewallDiff.RemovedDomains)))
		}
		if len(diff.FirewallDiff.StatusChanges) > 0 {
			fwParts = append(fwParts, fmt.Sprintf("%d status changes", len(diff.FirewallDiff.StatusChanges)))
		}
		if len(diff.FirewallDiff.VolumeChanges) > 0 {
			fwParts = append(fwParts, fmt.Sprintf("%d volume changes", len(diff.FirewallDiff.VolumeChanges)))
		}
		if len(fwParts) > 0 {
			summaryParts = append(summaryParts, "Firewall: "+strings.Join(fwParts, ", "))
		}
		anomalyCount += diff.FirewallDiff.Summary.AnomalyCount
	}

	if diff.MCPToolsDiff != nil && !isEmptyMCPToolsDiff(diff.MCPToolsDiff) {
		mcpParts := []string{}
		if diff.MCPToolsDiff.Summary.NewToolCount > 0 {
			mcpParts = append(mcpParts, fmt.Sprintf("%d new tools", diff.MCPToolsDiff.Summary.NewToolCount))
		}
		if diff.MCPToolsDiff.Summary.RemovedToolCount > 0 {
			mcpParts = append(mcpParts, fmt.Sprintf("%d removed tools", diff.MCPToolsDiff.Summary.RemovedToolCount))
		}
		if diff.MCPToolsDiff.Summary.ChangedToolCount > 0 {
			mcpParts = append(mcpParts, fmt.Sprintf("%d changed tools", diff.MCPToolsDiff.Summary.ChangedToolCount))
		}
		if len(mcpParts) > 0 {
			summaryParts = append(summaryParts, "MCP tools: "+strings.Join(mcpParts, ", "))
		}
		anomalyCount += diff.MCPToolsDiff.Summary.AnomalyCount
	}

	if len(summaryParts) > 0 {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Changes: "+strings.Join(summaryParts, " | ")))
	}
	if anomalyCount > 0 {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("⚠️  %d anomalies detected", anomalyCount)))
	}
	fmt.Fprintln(os.Stderr)

	renderFirewallDiffPrettySection(diff.FirewallDiff)
	renderMCPToolsDiffPrettySection(diff.MCPToolsDiff)
	renderRunMetricsDiffPrettySection(diff.Run1ID, diff.Run2ID, diff.RunMetricsDiff)
}

// renderFirewallDiffMarkdownSection renders the firewall diff sub-section as markdown
func renderFirewallDiffMarkdownSection(diff *FirewallDiff) {
	if diff == nil || isEmptyFirewallDiff(diff) {
		return
	}

	fmt.Println("#### Firewall Changes")
	fmt.Println()

	if len(diff.NewDomains) > 0 {
		fmt.Printf("**New domains (%d)**\n", len(diff.NewDomains))
		for _, entry := range diff.NewDomains {
			total := entry.Run2Allowed + entry.Run2Blocked
			statusIcon := statusEmoji(entry.Run2Status)
			anomalyTag := ""
			if entry.IsAnomaly {
				anomalyTag = " ⚠️"
			}
			fmt.Printf("- %s `%s` (%d requests, %s)%s\n", statusIcon, entry.Domain, total, entry.Run2Status, anomalyTag)
		}
		fmt.Println()
	}

	if len(diff.RemovedDomains) > 0 {
		fmt.Printf("**Removed domains (%d)**\n", len(diff.RemovedDomains))
		for _, entry := range diff.RemovedDomains {
			total := entry.Run1Allowed + entry.Run1Blocked
			fmt.Printf("- `%s` (was %s, %d requests in previous run)\n", entry.Domain, entry.Run1Status, total)
		}
		fmt.Println()
	}

	if len(diff.StatusChanges) > 0 {
		fmt.Printf("**Status changes (%d)**\n", len(diff.StatusChanges))
		for _, entry := range diff.StatusChanges {
			icon1 := statusEmoji(entry.Run1Status)
			icon2 := statusEmoji(entry.Run2Status)
			anomalyTag := ""
			if entry.IsAnomaly {
				anomalyTag = " ⚠️"
			}
			fmt.Printf("- `%s`: %s %s → %s %s%s\n", entry.Domain, icon1, entry.Run1Status, icon2, entry.Run2Status, anomalyTag)
		}
		fmt.Println()
	}

	if len(diff.VolumeChanges) > 0 {
		fmt.Printf("**Volume changes**\n")
		for _, entry := range diff.VolumeChanges {
			total1 := entry.Run1Allowed + entry.Run1Blocked
			total2 := entry.Run2Allowed + entry.Run2Blocked
			fmt.Printf("- `%s`: %d → %d requests (%s)\n", entry.Domain, total1, total2, entry.VolumeChange)
		}
		fmt.Println()
	}
}

// renderMCPToolsDiffMarkdownSection renders the MCP tools diff sub-section as markdown
func renderMCPToolsDiffMarkdownSection(diff *MCPToolsDiff) {
	if diff == nil || isEmptyMCPToolsDiff(diff) {
		return
	}

	fmt.Println("#### MCP Tool Changes")
	fmt.Println()

	if len(diff.NewTools) > 0 {
		fmt.Printf("**New tools (%d)**\n", len(diff.NewTools))
		for _, entry := range diff.NewTools {
			anomalyTag := ""
			if entry.IsAnomaly {
				anomalyTag = " ⚠️"
			}
			fmt.Printf("- `%s/%s` (%d calls)%s\n", entry.ServerName, entry.ToolName, entry.Run2CallCount, anomalyTag)
		}
		fmt.Println()
	}

	if len(diff.RemovedTools) > 0 {
		fmt.Printf("**Removed tools (%d)**\n", len(diff.RemovedTools))
		for _, entry := range diff.RemovedTools {
			fmt.Printf("- `%s/%s` (was %d calls)\n", entry.ServerName, entry.ToolName, entry.Run1CallCount)
		}
		fmt.Println()
	}

	if len(diff.ChangedTools) > 0 {
		fmt.Printf("**Changed tools (%d)**\n", len(diff.ChangedTools))
		for _, entry := range diff.ChangedTools {
			anomalyTag := ""
			if entry.IsAnomaly {
				anomalyTag = " ⚠️"
			}
			errInfo := ""
			if entry.Run1ErrorCount > 0 || entry.Run2ErrorCount > 0 {
				errInfo = fmt.Sprintf(", errors: %d → %d", entry.Run1ErrorCount, entry.Run2ErrorCount)
			}
			fmt.Printf("- `%s/%s`: %d → %d calls (%s%s)%s\n",
				entry.ServerName, entry.ToolName,
				entry.Run1CallCount, entry.Run2CallCount,
				entry.CallCountChange, errInfo, anomalyTag)
		}
		fmt.Println()
	}
}

// renderRunMetricsDiffMarkdownSection renders the run metrics diff sub-section as markdown
func renderRunMetricsDiffMarkdownSection(run1ID, run2ID int64, diff *RunMetricsDiff) {
	if diff == nil {
		return
	}

	fmt.Println("#### Run Metrics")
	fmt.Println()
	fmt.Printf("| Metric | Run #%d | Run #%d | Change |\n", run1ID, run2ID)
	fmt.Println("|--------|---------|---------|--------|")

	if diff.Run1TokenUsage > 0 || diff.Run2TokenUsage > 0 {
		fmt.Printf("| Token usage | %d | %d | %s |\n", diff.Run1TokenUsage, diff.Run2TokenUsage, diff.TokenUsageChange)
	}
	if diff.Run1Duration != "" || diff.Run2Duration != "" {
		fmt.Printf("| Duration | %s | %s | %s |\n", diff.Run1Duration, diff.Run2Duration, diff.DurationChange)
	}
	if diff.Run1Turns > 0 || diff.Run2Turns > 0 {
		turnsChange := fmt.Sprintf("%+d", diff.TurnsChange)
		fmt.Printf("| Turns | %d | %d | %s |\n", diff.Run1Turns, diff.Run2Turns, turnsChange)
	}
	fmt.Println()
}

// renderFirewallDiffPrettySection renders the firewall diff as a pretty console sub-section
func renderFirewallDiffPrettySection(diff *FirewallDiff) {
	if diff == nil || isEmptyFirewallDiff(diff) {
		return
	}

	fmt.Fprintf(os.Stderr, "  Firewall Changes:\n")

	if len(diff.NewDomains) > 0 {
		fmt.Fprintf(os.Stderr, "    New Domains (%d):\n", len(diff.NewDomains))
		for _, entry := range diff.NewDomains {
			total := entry.Run2Allowed + entry.Run2Blocked
			statusIcon := statusEmoji(entry.Run2Status)
			anomalyTag := ""
			if entry.IsAnomaly {
				anomalyTag = " [ANOMALY: " + entry.AnomalyNote + "]"
			}
			fmt.Fprintf(os.Stderr, "      %s %s (%d requests, %s)%s\n", statusIcon, entry.Domain, total, entry.Run2Status, anomalyTag)
		}
	}

	if len(diff.RemovedDomains) > 0 {
		fmt.Fprintf(os.Stderr, "    Removed Domains (%d):\n", len(diff.RemovedDomains))
		for _, entry := range diff.RemovedDomains {
			total := entry.Run1Allowed + entry.Run1Blocked
			fmt.Fprintf(os.Stderr, "      %s (was %s, %d requests)\n", entry.Domain, entry.Run1Status, total)
		}
	}

	if len(diff.StatusChanges) > 0 {
		fmt.Fprintf(os.Stderr, "    Status Changes (%d):\n", len(diff.StatusChanges))
		for _, entry := range diff.StatusChanges {
			icon1 := statusEmoji(entry.Run1Status)
			icon2 := statusEmoji(entry.Run2Status)
			anomalyTag := ""
			if entry.IsAnomaly {
				anomalyTag = " [ANOMALY: " + entry.AnomalyNote + "]"
			}
			fmt.Fprintf(os.Stderr, "      %s: %s %s → %s %s%s\n", entry.Domain, icon1, entry.Run1Status, icon2, entry.Run2Status, anomalyTag)
		}
	}

	if len(diff.VolumeChanges) > 0 {
		fmt.Fprintf(os.Stderr, "    Volume Changes:\n")
		for _, entry := range diff.VolumeChanges {
			total1 := entry.Run1Allowed + entry.Run1Blocked
			total2 := entry.Run2Allowed + entry.Run2Blocked
			fmt.Fprintf(os.Stderr, "      %s: %d → %d requests (%s)\n", entry.Domain, total1, total2, entry.VolumeChange)
		}
	}

	fmt.Fprintln(os.Stderr)
}

// renderMCPToolsDiffPrettySection renders the MCP tools diff as a pretty console sub-section
func renderMCPToolsDiffPrettySection(diff *MCPToolsDiff) {
	if diff == nil || isEmptyMCPToolsDiff(diff) {
		return
	}

	fmt.Fprintf(os.Stderr, "  MCP Tool Changes:\n")

	if len(diff.NewTools) > 0 {
		fmt.Fprintf(os.Stderr, "    New Tools (%d):\n", len(diff.NewTools))
		for _, entry := range diff.NewTools {
			anomalyTag := ""
			if entry.IsAnomaly {
				anomalyTag = " [ANOMALY: " + entry.AnomalyNote + "]"
			}
			fmt.Fprintf(os.Stderr, "      + %s/%s (%d calls)%s\n", entry.ServerName, entry.ToolName, entry.Run2CallCount, anomalyTag)
		}
	}

	if len(diff.RemovedTools) > 0 {
		fmt.Fprintf(os.Stderr, "    Removed Tools (%d):\n", len(diff.RemovedTools))
		for _, entry := range diff.RemovedTools {
			fmt.Fprintf(os.Stderr, "      - %s/%s (was %d calls)\n", entry.ServerName, entry.ToolName, entry.Run1CallCount)
		}
	}

	if len(diff.ChangedTools) > 0 {
		fmt.Fprintf(os.Stderr, "    Changed Tools (%d):\n", len(diff.ChangedTools))
		for _, entry := range diff.ChangedTools {
			anomalyTag := ""
			if entry.IsAnomaly {
				anomalyTag = " [ANOMALY: " + entry.AnomalyNote + "]"
			}
			errInfo := ""
			if entry.Run1ErrorCount > 0 || entry.Run2ErrorCount > 0 {
				errInfo = fmt.Sprintf(", errors: %d → %d", entry.Run1ErrorCount, entry.Run2ErrorCount)
			}
			fmt.Fprintf(os.Stderr, "      ~ %s/%s: %d → %d calls (%s%s)%s\n",
				entry.ServerName, entry.ToolName,
				entry.Run1CallCount, entry.Run2CallCount,
				entry.CallCountChange, errInfo, anomalyTag)
		}
	}

	fmt.Fprintln(os.Stderr)
}

// renderRunMetricsDiffPrettySection renders the run metrics diff as a pretty console sub-section
func renderRunMetricsDiffPrettySection(run1ID, run2ID int64, diff *RunMetricsDiff) {
	if diff == nil {
		return
	}

	fmt.Fprintf(os.Stderr, "  Run Metrics (Run #%d → Run #%d):\n", run1ID, run2ID)

	if diff.Run1TokenUsage > 0 || diff.Run2TokenUsage > 0 {
		fmt.Fprintf(os.Stderr, "    Token usage:  %d → %d (%s)\n", diff.Run1TokenUsage, diff.Run2TokenUsage, diff.TokenUsageChange)
	}
	if diff.Run1Duration != "" || diff.Run2Duration != "" {
		changeStr := ""
		if diff.DurationChange != "" {
			changeStr = " (" + diff.DurationChange + ")"
		}
		fmt.Fprintf(os.Stderr, "    Duration:     %s → %s%s\n", diff.Run1Duration, diff.Run2Duration, changeStr)
	}
	if diff.Run1Turns > 0 || diff.Run2Turns > 0 {
		fmt.Fprintf(os.Stderr, "    Turns:        %d → %d (%+d)\n", diff.Run1Turns, diff.Run2Turns, diff.TurnsChange)
	}

	fmt.Fprintln(os.Stderr)
}

// statusEmoji returns the status emoji for a domain status
func statusEmoji(status string) string {
	switch status {
	case "allowed":
		return "✅"
	case "denied":
		return "❌"
	case "mixed":
		return "⚠️"
	default:
		return "❓"
	}
}

// isEmptyFirewallDiff returns true if the firewall diff contains no changes
func isEmptyFirewallDiff(diff *FirewallDiff) bool {
	return len(diff.NewDomains) == 0 &&
		len(diff.RemovedDomains) == 0 &&
		len(diff.StatusChanges) == 0 &&
		len(diff.VolumeChanges) == 0
}

// isEmptyMCPToolsDiff returns true if the MCP tools diff contains no changes
func isEmptyMCPToolsDiff(diff *MCPToolsDiff) bool {
	return len(diff.NewTools) == 0 &&
		len(diff.RemovedTools) == 0 &&
		len(diff.ChangedTools) == 0
}

// isEmptyAuditDiff returns true if the audit diff contains no changes across all sections
func isEmptyAuditDiff(diff *AuditDiff) bool {
	fwEmpty := diff.FirewallDiff == nil || isEmptyFirewallDiff(diff.FirewallDiff)
	mcpEmpty := diff.MCPToolsDiff == nil || isEmptyMCPToolsDiff(diff.MCPToolsDiff)
	metricsEmpty := diff.RunMetricsDiff == nil
	return fwEmpty && mcpEmpty && metricsEmpty
}
