package cli

import (
	"fmt"
	"math"
	"os"
	"strconv"
	"time"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/stringutil"
)

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
