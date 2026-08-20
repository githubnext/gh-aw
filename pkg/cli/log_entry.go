package cli

import (
	"strconv"
	"strings"
	"time"
)

// LogEntrySource identifies the log stream a LogEntry was parsed from.
type LogEntrySource string

const (
	// LogSourceAccess identifies squid access log entries.
	LogSourceAccess LogEntrySource = "access"
	// LogSourceFirewall identifies firewall log entries.
	LogSourceFirewall LogEntrySource = "firewall"
	// LogSourceAudit identifies audit.jsonl entries.
	LogSourceAudit LogEntrySource = "audit"
	// LogSourceGateway identifies MCP gateway.jsonl entries.
	LogSourceGateway LogEntrySource = "gateway"
)

// Shared severity levels reported by LogEntry.EntryLevel.
const (
	LogLevelInfo  = "info"
	LogLevelError = "error"
)

// LogEntry is the shared shape implemented by every parsed log-line type
// (AccessLogEntry, FirewallLogEntry, AuditLogEntry and GatewayLogEntry).
// It lets formatting, filtering and reporting code operate on any log entry
// without special-casing each concrete type.
type LogEntry interface {
	// EntryTimestamp returns the entry timestamp, normalized to RFC3339 (UTC)
	// for sources that record epoch timestamps. Unparseable timestamps are
	// returned unchanged.
	EntryTimestamp() string
	// EntrySource returns the log stream the entry was parsed from.
	EntrySource() LogEntrySource
	// EntryLevel returns the entry severity: LogLevelInfo or LogLevelError.
	EntryLevel() string
	// EntryMessage returns a short human-readable description of the entry.
	EntryMessage() string
}

// Compile-time checks that all four log-entry types share the LogEntry shape.
var (
	_ LogEntry = AccessLogEntry{}
	_ LogEntry = FirewallLogEntry{}
	_ LogEntry = AuditLogEntry{}
	_ LogEntry = GatewayLogEntry{}
)

// formatEpochSeconds converts fractional epoch seconds to RFC3339 in UTC.
func formatEpochSeconds(seconds float64) string {
	return time.Unix(0, int64(seconds*float64(time.Second))).UTC().Format(time.RFC3339)
}

// formatEpochTimestamp converts a textual epoch-seconds timestamp to RFC3339 in UTC,
// returning the input unchanged when it is not a numeric epoch value.
func formatEpochTimestamp(timestamp string) string {
	seconds, err := strconv.ParseFloat(strings.TrimSpace(timestamp), 64)
	if err != nil {
		return timestamp
	}
	return formatEpochSeconds(seconds)
}

// levelFromAllowed maps an allow/deny outcome to a shared severity level.
func levelFromAllowed(allowed bool) string {
	if allowed {
		return LogLevelInfo
	}
	return LogLevelError
}

// joinEntryMessage builds a message from the first non-placeholder parts available.
func joinEntryMessage(parts ...string) string {
	kept := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" || part == "-" {
			continue
		}
		kept = append(kept, part)
	}
	return strings.Join(kept, " ")
}

// EntryTimestamp implements LogEntry.
func (a AccessLogEntry) EntryTimestamp() string { return formatEpochTimestamp(a.Timestamp) }

// EntrySource implements LogEntry.
func (a AccessLogEntry) EntrySource() LogEntrySource { return LogSourceAccess }

// EntryLevel implements LogEntry.
func (a AccessLogEntry) EntryLevel() string { return levelFromAllowed(isAllowedSquidStatus(a.Status)) }

// EntryMessage implements LogEntry.
func (a AccessLogEntry) EntryMessage() string { return joinEntryMessage(a.Method, a.URL, a.Status) }

// EntryTimestamp implements LogEntry.
func (f FirewallLogEntry) EntryTimestamp() string { return formatEpochTimestamp(f.Timestamp) }

// EntrySource implements LogEntry.
func (f FirewallLogEntry) EntrySource() LogEntrySource { return LogSourceFirewall }

// EntryLevel implements LogEntry.
func (f FirewallLogEntry) EntryLevel() string {
	return levelFromAllowed(isRequestAllowed(f.Decision, f.Status))
}

// EntryMessage implements LogEntry.
func (f FirewallLogEntry) EntryMessage() string {
	target := f.URL
	if strings.TrimSpace(target) == "" || target == "-" {
		target = f.Domain
	}
	return joinEntryMessage(f.Method, target, f.Decision)
}

// EntryTimestamp implements LogEntry.
func (a AuditLogEntry) EntryTimestamp() string { return formatEpochSeconds(a.Timestamp) }

// EntrySource implements LogEntry.
func (a AuditLogEntry) EntrySource() LogEntrySource { return LogSourceAudit }

// EntryLevel implements LogEntry.
func (a AuditLogEntry) EntryLevel() string { return levelFromAllowed(isEntryAllowed(a)) }

// EntryMessage implements LogEntry.
func (a AuditLogEntry) EntryMessage() string {
	target := a.URL
	if strings.TrimSpace(target) == "" || target == "-" {
		target = a.Host
	}
	return joinEntryMessage(a.Method, target, a.Decision)
}

// EntryTimestamp implements LogEntry.
func (g GatewayLogEntry) EntryTimestamp() string { return g.Timestamp }

// EntrySource implements LogEntry.
func (g GatewayLogEntry) EntrySource() LogEntrySource { return LogSourceGateway }

// EntryLevel implements LogEntry. It mirrors the error classification used
// by gateway log metrics processing (see processGatewayLogEntry), treating
// Status == "error", a non-empty Error, or Level == "error" as failure
// signals, and normalizes the result to the interface's two documented
// levels (LogLevelInfo / LogLevelError).
func (g GatewayLogEntry) EntryLevel() string {
	return levelFromAllowed(g.Status != "error" && g.Error == "" && g.Level != "error")
}

// EntryMessage implements LogEntry.
func (g GatewayLogEntry) EntryMessage() string {
	for _, candidate := range []string{g.Message, g.Error, g.Event, g.Type} {
		if strings.TrimSpace(candidate) != "" {
			return candidate
		}
	}
	return ""
}
