package cli

import (
	"fmt"
	"math"
	"os"
	"sort"
	"strconv"
	"time"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/sliceutil"
	"github.com/github/gh-aw/pkg/stringutil"
)

// renderGuardPolicySummary renders the guard policy enforcement summary
func renderGuardPolicySummary(summary *GuardPolicySummary) {
	auditReportLog.Printf("Rendering guard policy summary: %d total blocked", summary.TotalBlocked)

	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, console.FormatWarningMessage(
		fmt.Sprintf("Guard Policy: %d tool call(s) blocked", summary.TotalBlocked)))
	fmt.Fprintln(os.Stderr)

	// Breakdown by reason
	fmt.Fprintln(os.Stderr, "  Block Reasons:")
	if summary.IntegrityBlocked > 0 {
		fmt.Fprintf(os.Stderr, "    Integrity below minimum : %d\n", summary.IntegrityBlocked)
	}
	if summary.RepoScopeBlocked > 0 {
		fmt.Fprintf(os.Stderr, "    Repository not allowed  : %d\n", summary.RepoScopeBlocked)
	}
	if summary.AccessDenied > 0 {
		fmt.Fprintf(os.Stderr, "    Access denied           : %d\n", summary.AccessDenied)
	}
	if summary.BlockedUserDenied > 0 {
		fmt.Fprintf(os.Stderr, "    Blocked user            : %d\n", summary.BlockedUserDenied)
	}
	if summary.PermissionDenied > 0 {
		fmt.Fprintf(os.Stderr, "    Insufficient permissions: %d\n", summary.PermissionDenied)
	}
	if summary.PrivateRepoDenied > 0 {
		fmt.Fprintf(os.Stderr, "    Private repo denied     : %d\n", summary.PrivateRepoDenied)
	}
	fmt.Fprintln(os.Stderr)

	// Most frequently blocked tools
	if len(summary.BlockedToolCounts) > 0 {
		toolNames := sliceutil.MapToSlice(summary.BlockedToolCounts)
		sort.Slice(toolNames, func(i, j int) bool {
			return summary.BlockedToolCounts[toolNames[i]] > summary.BlockedToolCounts[toolNames[j]]
		})

		toolRows := make([][]string, 0, len(toolNames))
		for _, name := range toolNames {
			toolRows = append(toolRows, []string{name, strconv.Itoa(summary.BlockedToolCounts[name])})
		}
		fmt.Fprint(os.Stderr, console.RenderTable(console.TableConfig{
			Title:   "Most Blocked Tools",
			Headers: []string{"Tool", "Blocked"},
			Rows:    toolRows,
		}))
	}

	// Guard policy event details
	if len(summary.Events) > 0 {
		fmt.Fprintln(os.Stderr)
		eventRows := make([][]string, 0, len(summary.Events))
		for _, evt := range summary.Events {
			message := evt.Message
			if len(message) > 60 {
				message = message[:57] + "..."
			}
			repo := evt.Repository
			if repo == "" {
				repo = "-"
			}
			eventRows = append(eventRows, []string{
				stringutil.Truncate(evt.ServerID, 20),
				stringutil.Truncate(evt.ToolName, 25),
				evt.Reason,
				message,
				repo,
			})
		}
		fmt.Fprint(os.Stderr, console.RenderTable(console.TableConfig{
			Title:   "Guard Policy Events",
			Headers: []string{"Server", "Tool", "Reason", "Message", "Repository"},
			Rows:    eventRows,
		}))
	}
}

// renderFirewallAnalysis renders firewall analysis with summary and domain breakdown
func renderFirewallAnalysis(analysis *FirewallAnalysis) {
	auditReportLog.Printf("Rendering firewall analysis: total=%d, allowed=%d, blocked=%d, allowed_domains=%d, blocked_domains=%d",
		analysis.TotalRequests, analysis.AllowedRequests, analysis.BlockedRequests, len(analysis.AllowedDomains), len(analysis.BlockedDomains))
	// Summary statistics
	fmt.Fprintf(os.Stderr, "  Total Requests : %d\n", analysis.TotalRequests)
	fmt.Fprintf(os.Stderr, "  Allowed        : %d\n", analysis.AllowedRequests)
	fmt.Fprintf(os.Stderr, "  Blocked        : %d\n", analysis.BlockedRequests)
	fmt.Fprintln(os.Stderr)

	// Allowed domains
	if len(analysis.AllowedDomains) > 0 {
		fmt.Fprintln(os.Stderr, "  Allowed Domains:")
		for _, domain := range analysis.AllowedDomains {
			if stats, ok := analysis.RequestsByDomain[domain]; ok {
				fmt.Fprintf(os.Stderr, "    ✓ %s (%d requests)\n", domain, stats.Allowed)
			}
		}
		fmt.Fprintln(os.Stderr)
	}

	// Blocked domains
	if len(analysis.BlockedDomains) > 0 {
		fmt.Fprintln(os.Stderr, "  Blocked Domains:")
		for _, domain := range analysis.BlockedDomains {
			if stats, ok := analysis.RequestsByDomain[domain]; ok {
				fmt.Fprintf(os.Stderr, "    ✗ %s (%d requests)\n", domain, stats.Blocked)
			}
		}
		fmt.Fprintln(os.Stderr)
	}
}

// renderRedactedDomainsAnalysis renders redacted domains analysis
func renderRedactedDomainsAnalysis(analysis *RedactedDomainsAnalysis) {
	auditReportLog.Printf("Rendering redacted domains analysis: total_domains=%d", analysis.TotalDomains)
	// Summary statistics
	fmt.Fprintf(os.Stderr, "  Total Domains Redacted: %d\n", analysis.TotalDomains)
	fmt.Fprintln(os.Stderr)

	// List domains
	if len(analysis.Domains) > 0 {
		fmt.Fprintln(os.Stderr, "  Redacted Domains:")
		for _, domain := range analysis.Domains {
			fmt.Fprintf(os.Stderr, "    🔒 %s\n", domain)
		}
		fmt.Fprintln(os.Stderr)
	}
}

// renderPolicyAnalysis renders the enriched firewall policy analysis with rule attribution
func renderPolicyAnalysis(analysis *PolicyAnalysis) {
	auditReportLog.Printf("Rendering policy analysis: rules=%d, denied=%d", len(analysis.RuleHits), analysis.DeniedCount)

	// Policy summary using RenderStruct
	display := PolicySummaryDisplay{
		Policy:        analysis.PolicySummary,
		TotalRequests: analysis.TotalRequests,
		Allowed:       analysis.AllowedCount,
		Denied:        analysis.DeniedCount,
		UniqueDomains: analysis.UniqueDomains,
	}
	fmt.Fprint(os.Stderr, console.RenderStruct(display))
	fmt.Fprintln(os.Stderr)

	// Rule hit table
	if len(analysis.RuleHits) > 0 {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Policy Rules:"))
		fmt.Fprintln(os.Stderr)

		ruleConfig := console.TableConfig{
			Headers: []string{"Rule", "Action", "Description", "Hits"},
			Rows:    make([][]string, 0, len(analysis.RuleHits)),
		}

		for _, rh := range analysis.RuleHits {
			row := []string{
				stringutil.Truncate(rh.Rule.ID, 30),
				rh.Rule.Action,
				stringutil.Truncate(rh.Rule.Description, 50),
				strconv.Itoa(rh.Hits),
			}
			ruleConfig.Rows = append(ruleConfig.Rows, row)
		}

		fmt.Fprint(os.Stderr, console.RenderTable(ruleConfig))
		fmt.Fprintln(os.Stderr)
	}

	// Denied requests detail
	if len(analysis.DeniedRequests) > 0 {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Denied Requests (%d):", len(analysis.DeniedRequests))))
		fmt.Fprintln(os.Stderr)

		deniedConfig := console.TableConfig{
			Headers: []string{"Time", "Domain", "Rule", "Reason"},
			Rows:    make([][]string, 0, len(analysis.DeniedRequests)),
		}

		for _, req := range analysis.DeniedRequests {
			timeStr := formatUnixTimestamp(req.Timestamp)
			row := []string{
				timeStr,
				stringutil.Truncate(req.Host, 40),
				stringutil.Truncate(req.RuleID, 25),
				stringutil.Truncate(req.Reason, 40),
			}
			deniedConfig.Rows = append(deniedConfig.Rows, row)
		}

		fmt.Fprint(os.Stderr, console.RenderTable(deniedConfig))
		fmt.Fprintln(os.Stderr)
	}
}

// formatUnixTimestamp converts a Unix timestamp (float64) to a human-readable time string (HH:MM:SS).
func formatUnixTimestamp(ts float64) string {
	if ts <= 0 {
		return "-"
	}
	sec := int64(math.Floor(ts))
	nsec := int64((ts - float64(sec)) * 1e9)
	t := time.Unix(sec, nsec).UTC()
	return t.Format("15:04:05")
}
