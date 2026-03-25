package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/github/gh-aw/pkg/console"
)

// renderFirewallDiffJSON outputs the diff as JSON to stdout
func renderFirewallDiffJSON(diff *FirewallDiff) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(diff)
}

// renderFirewallDiffMarkdown outputs the diff as markdown to stdout
func renderFirewallDiffMarkdown(diff *FirewallDiff) {
	fmt.Printf("### Firewall Diff: Run #%d → Run #%d\n\n", diff.Run1ID, diff.Run2ID)

	if isEmptyDiff(diff) {
		fmt.Println("No firewall behavior changes detected between the two runs.")
		return
	}

	// New domains
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

	// Removed domains
	if len(diff.RemovedDomains) > 0 {
		fmt.Printf("**Removed domains (%d)**\n", len(diff.RemovedDomains))
		for _, entry := range diff.RemovedDomains {
			total := entry.Run1Allowed + entry.Run1Blocked
			fmt.Printf("- `%s` (was %s, %d requests in previous run)\n", entry.Domain, entry.Run1Status, total)
		}
		fmt.Println()
	}

	// Status changes
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

	// Volume changes
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

// renderFirewallDiffPretty outputs the diff as formatted console output to stderr
func renderFirewallDiffPretty(diff *FirewallDiff) {
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Firewall Diff: Run #%d → Run #%d", diff.Run1ID, diff.Run2ID)))
	fmt.Fprintln(os.Stderr)

	if isEmptyDiff(diff) {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("No firewall behavior changes detected between the two runs."))
		return
	}

	// Summary line
	parts := []string{}
	if len(diff.NewDomains) > 0 {
		parts = append(parts, fmt.Sprintf("%d new", len(diff.NewDomains)))
	}
	if len(diff.RemovedDomains) > 0 {
		parts = append(parts, fmt.Sprintf("%d removed", len(diff.RemovedDomains)))
	}
	if len(diff.StatusChanges) > 0 {
		parts = append(parts, fmt.Sprintf("%d status changes", len(diff.StatusChanges)))
	}
	if len(diff.VolumeChanges) > 0 {
		parts = append(parts, fmt.Sprintf("%d volume changes", len(diff.VolumeChanges)))
	}
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Changes: "+strings.Join(parts, ", ")))
	if diff.Summary.HasAnomalies {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("⚠️  %d anomalies detected", diff.Summary.AnomalyCount)))
	}
	fmt.Fprintln(os.Stderr)

	// New domains
	if len(diff.NewDomains) > 0 {
		fmt.Fprintf(os.Stderr, "  New Domains (%d):\n", len(diff.NewDomains))
		for _, entry := range diff.NewDomains {
			total := entry.Run2Allowed + entry.Run2Blocked
			statusIcon := statusEmoji(entry.Run2Status)
			anomalyTag := ""
			if entry.IsAnomaly {
				anomalyTag = " [ANOMALY: " + entry.AnomalyNote + "]"
			}
			fmt.Fprintf(os.Stderr, "    %s %s (%d requests, %s)%s\n", statusIcon, entry.Domain, total, entry.Run2Status, anomalyTag)
		}
		fmt.Fprintln(os.Stderr)
	}

	// Removed domains
	if len(diff.RemovedDomains) > 0 {
		fmt.Fprintf(os.Stderr, "  Removed Domains (%d):\n", len(diff.RemovedDomains))
		for _, entry := range diff.RemovedDomains {
			total := entry.Run1Allowed + entry.Run1Blocked
			fmt.Fprintf(os.Stderr, "    %s (was %s, %d requests)\n", entry.Domain, entry.Run1Status, total)
		}
		fmt.Fprintln(os.Stderr)
	}

	// Status changes
	if len(diff.StatusChanges) > 0 {
		fmt.Fprintf(os.Stderr, "  Status Changes (%d):\n", len(diff.StatusChanges))
		for _, entry := range diff.StatusChanges {
			icon1 := statusEmoji(entry.Run1Status)
			icon2 := statusEmoji(entry.Run2Status)
			anomalyTag := ""
			if entry.IsAnomaly {
				anomalyTag = " [ANOMALY: " + entry.AnomalyNote + "]"
			}
			fmt.Fprintf(os.Stderr, "    %s: %s %s → %s %s%s\n", entry.Domain, icon1, entry.Run1Status, icon2, entry.Run2Status, anomalyTag)
		}
		fmt.Fprintln(os.Stderr)
	}

	// Volume changes
	if len(diff.VolumeChanges) > 0 {
		fmt.Fprintf(os.Stderr, "  Volume Changes:\n")
		for _, entry := range diff.VolumeChanges {
			total1 := entry.Run1Allowed + entry.Run1Blocked
			total2 := entry.Run2Allowed + entry.Run2Blocked
			fmt.Fprintf(os.Stderr, "    %s: %d → %d requests (%s)\n", entry.Domain, total1, total2, entry.VolumeChange)
		}
		fmt.Fprintln(os.Stderr)
	}
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

// isEmptyDiff returns true if the diff contains no changes
func isEmptyDiff(diff *FirewallDiff) bool {
	return len(diff.NewDomains) == 0 &&
		len(diff.RemovedDomains) == 0 &&
		len(diff.StatusChanges) == 0 &&
		len(diff.VolumeChanges) == 0
}
