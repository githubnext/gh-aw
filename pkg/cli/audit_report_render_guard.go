package cli

import (
	"fmt"
	"os"
	"sort"
	"strconv"

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
