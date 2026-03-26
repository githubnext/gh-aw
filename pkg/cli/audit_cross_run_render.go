package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/stringutil"
)

// renderCrossRunReportJSON outputs the cross-run report as JSON to stdout.
func renderCrossRunReportJSON(report *CrossRunFirewallReport) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}

// renderCrossRunReportMarkdown outputs the cross-run report as Markdown to stdout.
func renderCrossRunReportMarkdown(report *CrossRunFirewallReport) {
	fmt.Println("# Audit Report — Cross-Run Firewall Analysis")
	fmt.Println()

	// Executive summary
	fmt.Println("## Executive Summary")
	fmt.Println()
	fmt.Printf("| Metric | Value |\n")
	fmt.Printf("|--------|-------|\n")
	fmt.Printf("| Runs analyzed | %d |\n", report.RunsAnalyzed)
	fmt.Printf("| Runs with firewall data | %d |\n", report.RunsWithData)
	fmt.Printf("| Runs without firewall data | %d |\n", report.RunsWithoutData)
	fmt.Printf("| Total requests | %d |\n", report.Summary.TotalRequests)
	fmt.Printf("| Allowed requests | %d |\n", report.Summary.TotalAllowed)
	fmt.Printf("| Blocked requests | %d |\n", report.Summary.TotalBlocked)
	fmt.Printf("| Overall denial rate | %.1f%% |\n", report.Summary.OverallDenyRate*100)
	fmt.Printf("| Unique domains | %d |\n", report.Summary.UniqueDomains)
	fmt.Println()

	// Domain inventory
	if len(report.DomainInventory) > 0 {
		fmt.Println("## Domain Inventory")
		fmt.Println()
		fmt.Printf("| Domain | Status | Seen In | Allowed | Blocked |\n")
		fmt.Printf("|--------|--------|---------|---------|--------|\n")
		for _, entry := range report.DomainInventory {
			icon := statusEmoji(entry.OverallStatus)
			fmt.Printf("| `%s` | %s %s | %d/%d runs | %d | %d |\n",
				entry.Domain, icon, entry.OverallStatus, entry.SeenInRuns, report.RunsAnalyzed,
				entry.TotalAllowed, entry.TotalBlocked)
		}
		fmt.Println()
	}

	// Per-run breakdown
	if len(report.PerRunBreakdown) > 0 {
		fmt.Println("## Per-Run Breakdown")
		fmt.Println()
		fmt.Printf("| Run ID | Workflow | Conclusion | Requests | Allowed | Blocked | Deny Rate | Domains |\n")
		fmt.Printf("|--------|----------|------------|----------|---------|---------|-----------|--------|\n")
		for _, run := range report.PerRunBreakdown {
			if !run.HasData {
				fmt.Printf("| %d | %s | %s | — | — | — | — | — |\n",
					run.RunID, run.WorkflowName, run.Conclusion)
				continue
			}
			fmt.Printf("| %d | %s | %s | %d | %d | %d | %.1f%% | %d |\n",
				run.RunID, run.WorkflowName, run.Conclusion,
				run.TotalRequests, run.Allowed, run.Blocked,
				run.DenyRate*100, run.UniqueDomains)
		}
		fmt.Println()
	}
}

// renderCrossRunReportPretty outputs the cross-run report as formatted console output to stderr.
func renderCrossRunReportPretty(report *CrossRunFirewallReport) {
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Audit Report — Cross-Run Firewall Analysis"))
	fmt.Fprintln(os.Stderr)

	// Executive summary
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Executive Summary"))
	fmt.Fprintf(os.Stderr, "  Runs analyzed:              %d\n", report.RunsAnalyzed)
	fmt.Fprintf(os.Stderr, "  Runs with firewall data:    %d\n", report.RunsWithData)
	fmt.Fprintf(os.Stderr, "  Runs without firewall data: %d\n", report.RunsWithoutData)
	fmt.Fprintf(os.Stderr, "  Total requests:             %d\n", report.Summary.TotalRequests)
	fmt.Fprintf(os.Stderr, "  Allowed / Blocked:          %d / %d\n", report.Summary.TotalAllowed, report.Summary.TotalBlocked)
	fmt.Fprintf(os.Stderr, "  Overall denial rate:        %.1f%%\n", report.Summary.OverallDenyRate*100)
	fmt.Fprintf(os.Stderr, "  Unique domains:             %d\n", report.Summary.UniqueDomains)
	fmt.Fprintln(os.Stderr)

	// Domain inventory
	if len(report.DomainInventory) > 0 {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Domain Inventory (%d domains)", len(report.DomainInventory))))
		for _, entry := range report.DomainInventory {
			icon := statusEmoji(entry.OverallStatus)
			fmt.Fprintf(os.Stderr, "  %s %-45s  %s  seen=%d/%d  allowed=%d  blocked=%d\n",
				icon, entry.Domain, entry.OverallStatus, entry.SeenInRuns, report.RunsAnalyzed,
				entry.TotalAllowed, entry.TotalBlocked)
		}
		fmt.Fprintln(os.Stderr)
	}

	// Per-run breakdown
	if len(report.PerRunBreakdown) > 0 {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Per-Run Breakdown"))
		for _, run := range report.PerRunBreakdown {
			if !run.HasData {
				fmt.Fprintf(os.Stderr, "  Run #%-12d  %-30s  %-10s  (no firewall data)\n",
					run.RunID, stringutil.Truncate(run.WorkflowName, 30), run.Conclusion)
				continue
			}
			fmt.Fprintf(os.Stderr, "  Run #%-12d  %-30s  %-10s  requests=%d  allowed=%d  blocked=%d  deny=%.1f%%  domains=%d\n",
				run.RunID, stringutil.Truncate(run.WorkflowName, 30), run.Conclusion,
				run.TotalRequests, run.Allowed, run.Blocked,
				run.DenyRate*100, run.UniqueDomains)
		}
		fmt.Fprintln(os.Stderr)
	}

	// Final status
	if report.RunsWithData == 0 {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage("No firewall data found in any of the analyzed runs."))
	} else {
		parts := []string{
			fmt.Sprintf("%d runs analyzed", report.RunsAnalyzed),
			fmt.Sprintf("%d unique domains", report.Summary.UniqueDomains),
			fmt.Sprintf("%.1f%% overall denial rate", report.Summary.OverallDenyRate*100),
		}
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Report complete: "+strings.Join(parts, ", ")))
	}
}
