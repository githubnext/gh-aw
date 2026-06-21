package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const usageActivitySummarySchema = "usage-activity-summary/v1"

type usageActivitySummary struct {
	Schema   string                 `json:"schema,omitempty"`
	Firewall *usageActivityFirewall `json:"firewall,omitempty"`
	Session  *usageActivitySession  `json:"session,omitempty"`
	Gateway  *usageActivityGateway  `json:"gateway,omitempty"`
}

type usageActivityFirewall struct {
	TotalRequests   int `json:"total_requests"`
	AllowedRequests int `json:"allowed_requests"`
	BlockedRequests int `json:"blocked_requests"`
}

type usageActivitySession struct {
	TotalEvents            int `json:"total_events"`
	SessionStarts          int `json:"session_starts"`
	SessionShutdowns       int `json:"session_shutdowns"`
	Turns                  int `json:"turns"`
	AssistantMessages      int `json:"assistant_messages"`
	ReasoningEvents        int `json:"reasoning_events"`
	ToolExecutionStarts    int `json:"tool_execution_starts"`
	ToolExecutionCompletes int `json:"tool_execution_completes"`
	FailedToolExecutions   int `json:"failed_tool_executions"`
}

type usageActivityGateway struct {
	TotalCalls  int                          `json:"total_calls"`
	FailedCalls int                          `json:"failed_calls"`
	Servers     []usageActivityGatewayServer `json:"servers,omitempty"`
}

type usageActivityGatewayServer struct {
	ServerName    string `json:"server_name"`
	ToolCallCount int    `json:"tool_call_count"`
	FailedCalls   int    `json:"failed_calls"`
}

func loadUsageActivitySummary(runDir string) (*usageActivitySummary, error) {
	candidates := []string{
		filepath.Join(runDir, "usage", "activity", "summary.json"),
		filepath.Join(runDir, "activity", "summary.json"),
	}
	var lastErr error
	for _, candidate := range candidates {
		cleanPath := filepath.Clean(candidate)
		raw, err := os.ReadFile(cleanPath)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("read usage activity summary %s: %w", cleanPath, err)
		}
		var summary usageActivitySummary
		if err := json.Unmarshal(raw, &summary); err != nil {
			lastErr = fmt.Errorf("parse usage activity summary %s: %w", cleanPath, err)
			continue
		}
		if summary.Schema != usageActivitySummarySchema {
			lastErr = fmt.Errorf("unsupported usage activity summary schema %q in %s (expected %q)", summary.Schema, cleanPath, usageActivitySummarySchema)
			continue
		}
		return &summary, nil
	}
	return nil, lastErr
}

func applyUsageActivitySummaryToResult(summary *usageActivitySummary, result *DownloadResult, allowTurnBackfill bool) {
	if summary == nil || result == nil {
		return
	}

	// Preserve previously parsed turn counts (from full session artifacts/events.jsonl)
	// and only backfill when they are missing.
	if allowTurnBackfill && summary.Session != nil && result.Run.Turns == 0 && summary.Session.Turns > 0 {
		result.Run.Turns = summary.Session.Turns
	}

	if summary.Firewall != nil && result.FirewallAnalysis == nil {
		result.FirewallAnalysis = &FirewallAnalysis{
			TotalRequests:    summary.Firewall.TotalRequests,
			AllowedRequests:  summary.Firewall.AllowedRequests,
			BlockedRequests:  summary.Firewall.BlockedRequests,
			RequestsByDomain: map[string]DomainRequestStats{},
		}
	}

	if summary.Gateway != nil && result.MCPToolUsage == nil {
		servers := make([]MCPServerStats, 0, len(summary.Gateway.Servers))
		for _, server := range summary.Gateway.Servers {
			servers = append(servers, MCPServerStats{
				ServerName: server.ServerName,
				// Keep both RequestCount and ToolCallCount aligned because MCPServerStats
				// distinguishes overall request volume (RequestCount) from tool-invocation
				// volume (ToolCallCount). In usage-aggregate mode we only have per-server
				// tool-call counts, so both fields are populated from that single source.
				RequestCount:  server.ToolCallCount,
				ToolCallCount: server.ToolCallCount,
				ErrorCount:    server.FailedCalls,
			})
		}
		result.MCPToolUsage = &MCPToolUsageData{
			Summary:   []MCPToolSummary{},
			ToolCalls: []MCPToolCall{},
			Servers:   servers,
		}
	}
}
