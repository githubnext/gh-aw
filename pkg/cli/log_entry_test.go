//go:build !integration

package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLogEntryInterfaceAccessors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name              string
		entry             LogEntry
		expectedTimestamp string
		expectedSource    LogEntrySource
		expectedLevel     string
		expectedMessage   string
	}{
		{
			name: "allowed access log entry",
			entry: AccessLogEntry{
				Timestamp: "1701234567.123",
				Status:    "TCP_MISS/200",
				Method:    "GET",
				URL:       "http://example.com/api",
			},
			expectedTimestamp: "2023-11-29T05:09:27Z",
			expectedSource:    LogSourceAccess,
			expectedLevel:     LogLevelInfo,
			expectedMessage:   "GET http://example.com/api TCP_MISS/200",
		},
		{
			name: "denied access log entry",
			entry: AccessLogEntry{
				Timestamp: "1701234568.456",
				Status:    "TCP_DENIED/403",
				Method:    "CONNECT",
				URL:       "github.com:443",
			},
			expectedTimestamp: "2023-11-29T05:09:28Z",
			expectedSource:    LogSourceAccess,
			expectedLevel:     LogLevelError,
			expectedMessage:   "CONNECT github.com:443 TCP_DENIED/403",
		},
		{
			name: "allowed firewall log entry",
			entry: FirewallLogEntry{
				Timestamp: "1761332530.474",
				Domain:    "api.github.com:443",
				Method:    "CONNECT",
				Status:    "200",
				Decision:  "TCP_TUNNEL:HIER_DIRECT",
				URL:       "api.github.com:443",
			},
			expectedTimestamp: "2025-10-24T19:02:10Z",
			expectedSource:    LogSourceFirewall,
			expectedLevel:     LogLevelInfo,
			expectedMessage:   "CONNECT api.github.com:443 TCP_TUNNEL:HIER_DIRECT",
		},
		{
			name: "blocked firewall log entry falls back to domain",
			entry: FirewallLogEntry{
				Timestamp: "1761332530.500",
				Domain:    "blocked.example.com:443",
				Method:    "-",
				Status:    "403",
				Decision:  "NONE_NONE:HIER_NONE",
				URL:       "-",
			},
			expectedTimestamp: "2025-10-24T19:02:10Z",
			expectedSource:    LogSourceFirewall,
			expectedLevel:     LogLevelError,
			expectedMessage:   "blocked.example.com:443 NONE_NONE:HIER_NONE",
		},
		{
			name: "allowed audit log entry",
			entry: AuditLogEntry{
				Timestamp: 1701234567.123,
				Host:      "api.github.com:443",
				Method:    "CONNECT",
				Status:    200,
				Decision:  "TCP_TUNNEL",
			},
			expectedTimestamp: "2023-11-29T05:09:27Z",
			expectedSource:    LogSourceAudit,
			expectedLevel:     LogLevelInfo,
			expectedMessage:   "CONNECT api.github.com:443 TCP_TUNNEL",
		},
		{
			name: "denied audit log entry",
			entry: AuditLogEntry{
				Timestamp: 1701234567.123,
				Host:      "evil.com:443",
				Method:    "CONNECT",
				Status:    403,
				Decision:  "NONE_NONE",
			},
			expectedTimestamp: "2023-11-29T05:09:27Z",
			expectedSource:    LogSourceAudit,
			expectedLevel:     LogLevelError,
			expectedMessage:   "CONNECT evil.com:443 NONE_NONE",
		},
		{
			name: "gateway log entry uses its own level",
			entry: GatewayLogEntry{
				Timestamp: "2024-01-12T10:00:00Z",
				Level:     LogLevelInfo,
				Type:      "request",
				Event:     "tool_call",
				Message:   "calling search_issues",
			},
			expectedTimestamp: "2024-01-12T10:00:00Z",
			expectedSource:    LogSourceGateway,
			expectedLevel:     LogLevelInfo,
			expectedMessage:   "calling search_issues",
		},
		{
			name: "gateway log entry without level falls back to error state",
			entry: GatewayLogEntry{
				Timestamp: "2024-01-12T10:00:01Z",
				Type:      "response",
				Event:     "tool_call",
				Error:     "connection timeout",
			},
			expectedTimestamp: "2024-01-12T10:00:01Z",
			expectedSource:    LogSourceGateway,
			expectedLevel:     LogLevelError,
			expectedMessage:   "connection timeout",
		},
		{
			name: "gateway log entry reports error level when status is error even with info level",
			entry: GatewayLogEntry{
				Timestamp: "2024-01-12T10:00:02Z",
				Level:     LogLevelInfo,
				Status:    "error",
				Event:     "tool_call",
				Message:   "call failed",
			},
			expectedTimestamp: "2024-01-12T10:00:02Z",
			expectedSource:    LogSourceGateway,
			expectedLevel:     LogLevelError,
			expectedMessage:   "call failed",
		},
		{
			name: "gateway log entry with all-blank fields returns empty message",
			entry: GatewayLogEntry{
				Timestamp: "2024-01-12T10:00:03Z",
				Level:     LogLevelInfo,
			},
			expectedTimestamp: "2024-01-12T10:00:03Z",
			expectedSource:    LogSourceGateway,
			expectedLevel:     LogLevelInfo,
			expectedMessage:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expectedTimestamp, tt.entry.EntryTimestamp())
			assert.Equal(t, tt.expectedSource, tt.entry.EntrySource())
			assert.Equal(t, tt.expectedLevel, tt.entry.EntryLevel())
			assert.Equal(t, tt.expectedMessage, tt.entry.EntryMessage())
		})
	}
}

func TestFormatEpochTimestampKeepsNonEpochValues(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "2024-01-12T10:00:00Z", formatEpochTimestamp("2024-01-12T10:00:00Z"))
	assert.Empty(t, formatEpochTimestamp(""))
	assert.Equal(t, "-", formatEpochTimestamp("-"))
	// A zero epoch is a valid timestamp, not a placeholder like "-" or "", so
	// it is normalized like any other numeric epoch value.
	assert.Equal(t, "1970-01-01T00:00:00Z", formatEpochTimestamp("0"))
}
