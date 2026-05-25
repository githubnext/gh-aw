// This file contains rendering primitives and the top-level render function for the
// unified MCP Gateway + AWF firewall timeline produced by BuildUnifiedTimeline.
//
// A dedicated rendering primitive exists for every TimelineEventKind so that each event
// type is displayed with appropriate context and formatting:
//
//   TimelineKindToolCall           – renderGatewayToolCallRow
//   TimelineKindDIFCFiltered       – renderGatewayDIFCFilteredRow
//   TimelineKindGuardPolicyBlocked – renderGatewayGuardPolicyBlockedRow
//   TimelineKindNetworkAllowed     – renderFirewallNetworkAllowedRow
//   TimelineKindNetworkBlocked     – renderFirewallNetworkBlockedRow
//
// renderTimelineEventRow dispatches to the appropriate primitive and returns a
// []string suitable for inclusion in a console.TableConfig.Rows slice.

package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/stringutil"
)

// timelineEventIcon returns a single Unicode icon for each event kind.
func timelineEventIcon(kind TimelineEventKind) string {
	switch kind {
	case TimelineKindToolCall:
		return "🔧"
	case TimelineKindDIFCFiltered:
		return "🚫"
	case TimelineKindGuardPolicyBlocked:
		return "🛡"
	case TimelineKindNetworkAllowed:
		return "✓"
	case TimelineKindNetworkBlocked:
		return "✗"
	default:
		return "·"
	}
}

// timelineEventKindLabel returns a short human-readable label for each event kind.
func timelineEventKindLabel(kind TimelineEventKind) string {
	switch kind {
	case TimelineKindToolCall:
		return "tool_call"
	case TimelineKindDIFCFiltered:
		return "difc_filtered"
	case TimelineKindGuardPolicyBlocked:
		return "guard_blocked"
	case TimelineKindNetworkAllowed:
		return "net_allowed"
	case TimelineKindNetworkBlocked:
		return "net_blocked"
	default:
		return string(kind)
	}
}

// timelineSourceLabel returns a short (2-char) label for each event source.
func timelineSourceLabel(source TimelineEventSource) string {
	switch source {
	case TimelineSourceGateway:
		return "GW"
	case TimelineSourceFirewall:
		return "FW"
	default:
		return strings.ToUpper(string(source))[:2]
	}
}

// formatTimelineTime formats a timeline event timestamp as HH:MM:SS.mmm (UTC).
// Returns "-" for the zero value.
func formatTimelineTime(evt UnifiedTimelineEvent) string {
	if evt.Time.IsZero() {
		return "-"
	}
	return evt.Time.UTC().Format("15:04:05.000")
}

// ─── Per-kind rendering primitives ───────────────────────────────────────────

// renderGatewayToolCallRow renders a TimelineKindToolCall event as a table row.
//
// Columns: Time | Src | Kind | Detail | Status
//
// Detail encodes the server and tool name (server/tool). Status shows the
// round-trip duration when available, or the status string (success/error).
// An error suffix is appended to Status when the entry carries an error message.
func renderGatewayToolCallRow(evt UnifiedTimelineEvent) []string {
	ts := formatTimelineTime(evt)
	src := timelineSourceLabel(evt.Source)
	kind := timelineEventIcon(TimelineKindToolCall) + " " + timelineEventKindLabel(TimelineKindToolCall)

	tool := evt.ToolName
	if tool == "" {
		tool = evt.Method
	}
	detail := tool
	if evt.ServerName != "" && tool != "" {
		detail = evt.ServerName + "/" + tool
	} else if evt.ServerName != "" {
		detail = evt.ServerName
	}
	detail = stringutil.Truncate(detail, 45)

	status := evt.Status
	if evt.Duration > 0 {
		status = fmt.Sprintf("%.0fms", evt.Duration)
	}
	if evt.Error != "" {
		errStr := stringutil.Truncate(evt.Error, 25)
		status = "error: " + errStr
	}
	status = stringutil.Truncate(status, 35)

	return []string{ts, src, kind, detail, status}
}

// renderGatewayDIFCFilteredRow renders a TimelineKindDIFCFiltered event as a table row.
//
// Detail shows the server and tool name. Status shows the author login (prefixed "@")
// when available, falling back to a truncated reason string.
func renderGatewayDIFCFilteredRow(evt UnifiedTimelineEvent) []string {
	ts := formatTimelineTime(evt)
	src := timelineSourceLabel(evt.Source)
	kind := timelineEventIcon(TimelineKindDIFCFiltered) + " " + timelineEventKindLabel(TimelineKindDIFCFiltered)

	detail := evt.ToolName
	if evt.ServerName != "" && evt.ToolName != "" {
		detail = evt.ServerName + "/" + evt.ToolName
	} else if evt.ServerName != "" {
		detail = evt.ServerName
	}
	detail = stringutil.Truncate(detail, 45)

	status := stringutil.Truncate(evt.Reason, 35)
	if evt.AuthorLogin != "" {
		status = "@" + evt.AuthorLogin
	}

	return []string{ts, src, kind, detail, status}
}

// renderGatewayGuardPolicyBlockedRow renders a TimelineKindGuardPolicyBlocked event as a
// table row.
//
// Detail shows the server and tool name. Status shows the reason or error message.
func renderGatewayGuardPolicyBlockedRow(evt UnifiedTimelineEvent) []string {
	ts := formatTimelineTime(evt)
	src := timelineSourceLabel(evt.Source)
	kind := timelineEventIcon(TimelineKindGuardPolicyBlocked) + " " + timelineEventKindLabel(TimelineKindGuardPolicyBlocked)

	detail := evt.ToolName
	if evt.ServerName != "" && evt.ToolName != "" {
		detail = evt.ServerName + "/" + evt.ToolName
	} else if evt.ServerName != "" {
		detail = evt.ServerName
	}
	detail = stringutil.Truncate(detail, 45)

	status := evt.Reason
	if status == "" {
		status = evt.Error
	}
	status = stringutil.Truncate(status, 35)

	return []string{ts, src, kind, detail, status}
}

// renderFirewallNetworkAllowedRow renders a TimelineKindNetworkAllowed event as a table row.
//
// Detail shows the target host. Status shows the HTTP status code.
func renderFirewallNetworkAllowedRow(evt UnifiedTimelineEvent) []string {
	ts := formatTimelineTime(evt)
	src := timelineSourceLabel(evt.Source)
	kind := timelineEventIcon(TimelineKindNetworkAllowed) + " " + timelineEventKindLabel(TimelineKindNetworkAllowed)
	detail := stringutil.Truncate(evt.Host, 45)

	status := ""
	if evt.HTTPStatus > 0 {
		status = fmt.Sprintf("HTTP %d", evt.HTTPStatus)
	}
	if evt.HTTPMethod != "" {
		status = evt.HTTPMethod + " " + status
	}
	status = strings.TrimSpace(status)
	status = stringutil.Truncate(status, 35)

	return []string{ts, src, kind, detail, status}
}

// renderFirewallNetworkBlockedRow renders a TimelineKindNetworkBlocked event as a table row.
//
// Detail shows the target host. Status shows the HTTP status code or "blocked" when no
// status is available.
func renderFirewallNetworkBlockedRow(evt UnifiedTimelineEvent) []string {
	ts := formatTimelineTime(evt)
	src := timelineSourceLabel(evt.Source)
	kind := timelineEventIcon(TimelineKindNetworkBlocked) + " " + timelineEventKindLabel(TimelineKindNetworkBlocked)
	detail := stringutil.Truncate(evt.Host, 45)

	status := "blocked"
	if evt.HTTPStatus > 0 {
		status = fmt.Sprintf("HTTP %d", evt.HTTPStatus)
	}
	if evt.HTTPMethod != "" {
		status = evt.HTTPMethod + " " + status
	}
	status = strings.TrimSpace(status)
	status = stringutil.Truncate(status, 35)

	return []string{ts, src, kind, detail, status}
}

// renderTimelineEventRow dispatches to the appropriate per-kind rendering primitive and
// returns a []string table row with columns: Time | Src | Kind | Detail | Status.
func renderTimelineEventRow(evt UnifiedTimelineEvent) []string {
	switch evt.Kind {
	case TimelineKindToolCall:
		return renderGatewayToolCallRow(evt)
	case TimelineKindDIFCFiltered:
		return renderGatewayDIFCFilteredRow(evt)
	case TimelineKindGuardPolicyBlocked:
		return renderGatewayGuardPolicyBlockedRow(evt)
	case TimelineKindNetworkAllowed:
		return renderFirewallNetworkAllowedRow(evt)
	case TimelineKindNetworkBlocked:
		return renderFirewallNetworkBlockedRow(evt)
	default:
		// Fallback for any future event kinds not yet handled.
		ts := formatTimelineTime(evt)
		return []string{ts, timelineSourceLabel(evt.Source), string(evt.Kind), "", ""}
	}
}

// ─── Top-level renderer ───────────────────────────────────────────────────────

// renderUnifiedTimeline renders a merged slice of UnifiedTimelineEvents as a single
// console table preceded by a summary of event counts per source and kind.
// Returns an empty string when events is empty.
func renderUnifiedTimeline(events []UnifiedTimelineEvent) string {
	if len(events) == 0 {
		return ""
	}

	// Tally event counts for the summary header.
	var gwCount, fwCount int
	var toolCalls, difcFiltered, guardBlocked, netAllowed, netBlocked int
	for _, evt := range events {
		switch evt.Source {
		case TimelineSourceGateway:
			gwCount++
		case TimelineSourceFirewall:
			fwCount++
		}
		switch evt.Kind {
		case TimelineKindToolCall:
			toolCalls++
		case TimelineKindDIFCFiltered:
			difcFiltered++
		case TimelineKindGuardPolicyBlocked:
			guardBlocked++
		case TimelineKindNetworkAllowed:
			netAllowed++
		case TimelineKindNetworkBlocked:
			netBlocked++
		}
	}

	var sb strings.Builder

	sb.WriteString("\n")
	sb.WriteString(console.FormatInfoMessage("Unified MCP + Firewall Event Timeline"))
	sb.WriteString("\n\n")

	fmt.Fprintf(&sb, "Total Events  : %d\n", len(events))
	if gwCount > 0 {
		fmt.Fprintf(&sb, "  Gateway     : %d  (tool_calls=%d, difc_filtered=%d, guard_blocked=%d)\n",
			gwCount, toolCalls, difcFiltered, guardBlocked)
	}
	if fwCount > 0 {
		fmt.Fprintf(&sb, "  Firewall    : %d  (allowed=%d, blocked=%d)\n",
			fwCount, netAllowed, netBlocked)
	}
	sb.WriteString("\n")

	// Build the table rows using per-kind primitives.
	rows := make([][]string, 0, len(events))
	for _, evt := range events {
		rows = append(rows, renderTimelineEventRow(evt))
	}

	sb.WriteString(console.RenderTable(console.TableConfig{
		Title:   "Event Timeline",
		Headers: []string{"Time", "Src", "Kind", "Detail", "Status"},
		Rows:    rows,
	}))

	return sb.String()
}

// displayUnifiedTimeline collects all JSONL events from every processed run, merges them
// into a single chronologically ordered stream, and writes the rendered timeline to
// stderr. It is a no-op when no events can be collected from any run.
func displayUnifiedTimeline(processedRuns []ProcessedRun, verbose bool) {
	gatewayLogsLog.Printf("Collecting unified timeline events from %d processed runs", len(processedRuns))

	var allEvents []UnifiedTimelineEvent
	for _, pr := range processedRuns {
		logDir := pr.Run.LogsPath
		if logDir == "" {
			continue
		}
		events, err := BuildUnifiedTimeline(logDir, verbose)
		if err != nil {
			gatewayLogsLog.Printf("BuildUnifiedTimeline error for run %d: %v", pr.Run.RunID, err)
			continue
		}
		allEvents = append(allEvents, events...)
	}

	if len(allEvents) == 0 {
		gatewayLogsLog.Print("No unified timeline events found across all runs")
		return
	}

	// Re-sort after merging events from multiple runs.
	for i := 1; i < len(allEvents); i++ {
		if allEvents[i].Time.Before(allEvents[i-1].Time) {
			// Only sort if needed (avoids allocation when already sorted).
			sortUnifiedTimelineEvents(allEvents)
			break
		}
	}

	gatewayLogsLog.Printf("Rendering unified timeline: %d total events across %d runs", len(allEvents), len(processedRuns))
	if output := renderUnifiedTimeline(allEvents); output != "" {
		fmt.Fprint(os.Stderr, output)
	}
}

// sortUnifiedTimelineEvents sorts events in-place by ascending wall-clock time.
func sortUnifiedTimelineEvents(events []UnifiedTimelineEvent) {
	for i := 1; i < len(events); i++ {
		if events[i].Time.Before(events[i-1].Time) {
			// Insertion sort is efficient for nearly-sorted slices; use stdlib for
			// general correctness.
			sortEventsStable(events)
			return
		}
	}
}

func sortEventsStable(events []UnifiedTimelineEvent) {
	// sort.SliceStable preserves insertion order for equal timestamps.
	sort.SliceStable(events, func(i, j int) bool {
		return events[i].Time.Before(events[j].Time)
	})
}
