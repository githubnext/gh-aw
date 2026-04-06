package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/timeutil"
)

var auditExpandedLog = logger.New("cli:audit_expanded")

// EngineConfig represents the engine configuration extracted from aw_info.json
type EngineConfig struct {
	EngineID        string   `json:"engine_id" console:"header:Engine ID"`
	EngineName      string   `json:"engine_name,omitempty" console:"header:Engine Name,omitempty"`
	Model           string   `json:"model,omitempty" console:"header:Model,omitempty"`
	Version         string   `json:"version,omitempty" console:"header:Version,omitempty"`
	CLIVersion      string   `json:"cli_version,omitempty" console:"header:CLI Version,omitempty"`
	FirewallVersion string   `json:"firewall_version,omitempty" console:"header:Firewall Version,omitempty"`
	MCPServers      []string `json:"mcp_servers,omitempty"`
	TriggerEvent    string   `json:"trigger_event,omitempty" console:"header:Trigger Event,omitempty"`
	Repository      string   `json:"repository,omitempty" console:"header:Repository,omitempty"`
}

// PromptAnalysis represents analysis of the input prompt
type PromptAnalysis struct {
	PromptSize int    `json:"prompt_size" console:"header:Prompt Size (chars)"`
	PromptFile string `json:"prompt_file,omitempty" console:"header:Prompt File,omitempty"`
}

// SessionAnalysis represents session and agent performance metrics
type SessionAnalysis struct {
	WallTime         string  `json:"wall_time,omitempty" console:"header:Wall Time,omitempty"`
	TurnCount        int     `json:"turn_count,omitempty" console:"header:Turn Count,omitempty"`
	AvgTurnDuration  string  `json:"avg_turn_duration,omitempty" console:"header:Avg Turn Duration,omitempty"`
	TokensPerMinute  float64 `json:"tokens_per_minute,omitempty"`
	TimeoutDetected  bool    `json:"timeout_detected"`
	NoopCount        int     `json:"noop_count,omitempty" console:"header:Noop Count,omitempty"`
	AgentActiveRatio float64 `json:"agent_active_ratio,omitempty"` // 0.0 - 1.0
}

// SafeOutputSummary provides a summary of safe output items by type
type SafeOutputSummary struct {
	TotalItems  int                    `json:"total_items" console:"header:Total Items"`
	ItemsByType map[string]int         `json:"items_by_type"`
	Summary     string                 `json:"summary" console:"header:Summary"`
	TypeDetails []SafeOutputTypeDetail `json:"type_details,omitempty"`
}

// SafeOutputTypeDetail contains counts for a specific safe output type
type SafeOutputTypeDetail struct {
	Type  string `json:"type" console:"header:Type"`
	Count int    `json:"count" console:"header:Count"`
}

// MCPServerHealth provides a health summary of MCP servers from gateway metrics
type MCPServerHealth struct {
	TotalServers  int                     `json:"total_servers"`
	HealthySvrs   int                     `json:"healthy_servers"`
	DegradedSvrs  int                     `json:"degraded_servers"`
	FailedSvrs    int                     `json:"failed_servers"`
	Summary       string                  `json:"summary" console:"header:Summary"`
	TotalRequests int                     `json:"total_requests" console:"header:Total Requests"`
	TotalErrors   int                     `json:"total_errors" console:"header:Total Errors"`
	ErrorRate     float64                 `json:"error_rate"`
	Servers       []MCPServerHealthDetail `json:"servers,omitempty"`
	SlowestCalls  []MCPSlowestToolCall    `json:"slowest_calls,omitempty"`
}

// MCPServerHealthDetail represents health details for a single MCP server
type MCPServerHealthDetail struct {
	ServerName   string  `json:"server_name" console:"header:Server"`
	RequestCount int     `json:"request_count" console:"header:Requests"`
	ToolCalls    int     `json:"tool_calls" console:"header:Tool Calls"`
	ErrorCount   int     `json:"error_count" console:"header:Errors"`
	ErrorRate    float64 `json:"error_rate"`
	ErrorRateStr string  `json:"error_rate_str" console:"header:Error Rate"`
	AvgLatency   string  `json:"avg_latency" console:"header:Avg Latency"`
	Status       string  `json:"status" console:"header:Status"`
}

// MCPSlowestToolCall represents a slow tool call for surfacing in the audit
type MCPSlowestToolCall struct {
	ServerName string `json:"server_name" console:"header:Server"`
	ToolName   string `json:"tool_name" console:"header:Tool"`
	Duration   string `json:"duration" console:"header:Duration"`
}

// findAwInfoPath returns the first existing aw_info.json path from known locations.
// The activation artifact may or may not have been flattened to the root directory.
func findAwInfoPath(logsPath string) string {
	candidates := []string{
		filepath.Join(logsPath, "aw_info.json"),
		filepath.Join(logsPath, "activation", "aw_info.json"),
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

// extractEngineConfig parses aw_info.json and returns an EngineConfig
func extractEngineConfig(logsPath string) *EngineConfig {
	if logsPath == "" {
		return nil
	}

	awInfoPath := findAwInfoPath(logsPath)
	if awInfoPath == "" {
		auditExpandedLog.Printf("aw_info.json not found in %s", logsPath)
		return nil
	}
	awInfo, err := parseAwInfo(awInfoPath, false)
	if err != nil || awInfo == nil {
		auditExpandedLog.Printf("Failed to parse aw_info.json for engine config: %v", err)
		return nil
	}

	config := &EngineConfig{
		EngineID:        awInfo.EngineID,
		EngineName:      awInfo.EngineName,
		Model:           awInfo.Model,
		Version:         awInfo.Version,
		CLIVersion:      awInfo.CLIVersion,
		FirewallVersion: awInfo.GetFirewallVersion(),
		TriggerEvent:    awInfo.EventName,
		Repository:      awInfo.Repository,
	}

	// Extract MCP server names from aw_info.json steps metadata
	if mcpNames, ok := extractMCPServerNamesFromAwInfo(logsPath); ok {
		config.MCPServers = mcpNames
	}

	auditExpandedLog.Printf("Extracted engine config: engine=%s, model=%s, mcp_servers=%d",
		config.EngineID, config.Model, len(config.MCPServers))
	return config
}

// extractPromptAnalysis reads prompt.txt and returns analysis metrics
func extractPromptAnalysis(logsPath string) *PromptAnalysis {
	if logsPath == "" {
		return nil
	}

	// Try multiple possible locations for prompt.txt.
	// The activation artifact may or may not have been flattened to the root.
	promptPaths := []string{
		filepath.Join(logsPath, "prompt.txt"),
		filepath.Join(logsPath, "aw-prompts", "prompt.txt"),
		filepath.Join(logsPath, "activation", "aw-prompts", "prompt.txt"),
		filepath.Join(logsPath, "agent", "aw-prompts", "prompt.txt"),
	}

	for _, promptPath := range promptPaths {
		data, err := os.ReadFile(promptPath)
		if err != nil {
			continue
		}

		// Store a stable relative path instead of machine-specific absolute path
		relPromptPath, relErr := filepath.Rel(logsPath, promptPath)
		if relErr != nil {
			relPromptPath = filepath.Base(promptPath)
		}

		analysis := &PromptAnalysis{
			PromptSize: len(data),
			PromptFile: relPromptPath,
		}

		auditExpandedLog.Printf("Extracted prompt analysis: size=%d chars from %s", analysis.PromptSize, relPromptPath)
		return analysis
	}

	auditExpandedLog.Printf("No prompt.txt found in %s", logsPath)
	return nil
}

// buildSessionAnalysis creates session performance metrics from available data
func buildSessionAnalysis(processedRun ProcessedRun, metrics LogMetrics) *SessionAnalysis {
	run := processedRun.Run

	session := &SessionAnalysis{
		TurnCount: metrics.Turns,
		NoopCount: run.NoopCount,
	}

	// Wall time from run duration
	if run.Duration > 0 {
		session.WallTime = timeutil.FormatDuration(run.Duration)
	}

	// Average turn duration
	if metrics.Turns > 0 && run.Duration > 0 {
		avgTurnDuration := run.Duration / (time.Duration(metrics.Turns))
		session.AvgTurnDuration = timeutil.FormatDuration(avgTurnDuration)
	}

	// Tokens per minute
	if metrics.TokenUsage > 0 && run.Duration > 0 {
		minutes := run.Duration.Minutes()
		if minutes > 0 {
			session.TokensPerMinute = float64(metrics.TokenUsage) / minutes
		}
	}

	// Timeout detection: check if the run was cancelled (typically indicates timeout)
	if run.Conclusion == "cancelled" || run.Conclusion == "timed_out" {
		session.TimeoutDetected = true
	}

	// Check for timeout patterns in job conclusions
	for _, job := range processedRun.JobDetails {
		if job.Conclusion == "cancelled" || job.Conclusion == "timed_out" {
			session.TimeoutDetected = true
			break
		}
	}

	auditExpandedLog.Printf("Built session analysis: turns=%d, wall_time=%s, timeout=%v",
		session.TurnCount, session.WallTime, session.TimeoutDetected)
	return session
}

// buildSafeOutputSummary creates a summary of safe output items by type
func buildSafeOutputSummary(items []CreatedItemReport) *SafeOutputSummary {
	if len(items) == 0 {
		return nil
	}

	summary := &SafeOutputSummary{
		TotalItems:  len(items),
		ItemsByType: make(map[string]int),
	}

	// Count items by type
	for _, item := range items {
		itemType := item.Type
		if itemType == "" {
			itemType = "unknown"
		}
		summary.ItemsByType[itemType]++
	}

	// Build type details sorted by count (desc), then type name (asc) for determinism
	for itemType, count := range summary.ItemsByType {
		summary.TypeDetails = append(summary.TypeDetails, SafeOutputTypeDetail{
			Type:  itemType,
			Count: count,
		})
	}
	sort.Slice(summary.TypeDetails, func(i, j int) bool {
		if summary.TypeDetails[i].Count == summary.TypeDetails[j].Count {
			return summary.TypeDetails[i].Type < summary.TypeDetails[j].Type
		}
		return summary.TypeDetails[i].Count > summary.TypeDetails[j].Count
	})

	// Build human-readable summary string
	summary.Summary = buildSafeOutputSummaryString(summary.TypeDetails)

	auditExpandedLog.Printf("Built safe output summary: %d items across %d types",
		summary.TotalItems, len(summary.ItemsByType))
	return summary
}

// buildSafeOutputSummaryString creates a human-readable summary like "2 PRs, 1 comment, 1 review"
func buildSafeOutputSummaryString(details []SafeOutputTypeDetail) string {
	if len(details) == 0 {
		return "No items"
	}

	parts := make([]string, 0, len(details))
	for _, detail := range details {
		displayType := prettifySafeOutputType(detail.Type)
		parts = append(parts, fmt.Sprintf("%d %s", detail.Count, displayType))
	}
	return strings.Join(parts, ", ")
}

// prettifySafeOutputType converts safe output types to human-readable names
func prettifySafeOutputType(itemType string) string {
	typeMap := map[string]string{
		"create_pull_request":   "PR(s)",
		"create_issue":          "issue(s)",
		"add_comment":           "comment(s)",
		"add_issue_comment":     "issue comment(s)",
		"create_review":         "review(s)",
		"add_labels":            "label operation(s)",
		"close_issue":           "issue close(s)",
		"create_discussion":     "discussion(s)",
		"create_release":        "release(s)",
		"update_pull_request":   "PR update(s)",
		"merge_pull_request":    "PR merge(s)",
		"create_or_update_file": "file operation(s)",
	}
	if display, ok := typeMap[itemType]; ok {
		return display
	}
	return itemType
}

// buildMCPServerHealth creates MCP server health summary from gateway metrics and MCP failures
func buildMCPServerHealth(mcpToolUsage *MCPToolUsageData, mcpFailures []MCPFailureReport) *MCPServerHealth {
	if mcpToolUsage == nil && len(mcpFailures) == 0 {
		return nil
	}

	health := &MCPServerHealth{}

	// Track failed servers from MCPFailures
	failedServers := make(map[string]bool)
	for _, failure := range mcpFailures {
		failedServers[failure.ServerName] = true
	}
	health.FailedSvrs = len(failedServers)

	// Process server statistics from mcpToolUsage
	if mcpToolUsage != nil {
		for _, server := range mcpToolUsage.Servers {
			health.TotalRequests += server.RequestCount
			health.TotalErrors += server.ErrorCount

			errorRate := 0.0
			if server.RequestCount > 0 {
				errorRate = float64(server.ErrorCount) / float64(server.RequestCount) * 100
			}

			status := "✅ healthy"
			if _, isFailed := failedServers[server.ServerName]; isFailed {
				status = "❌ failed"
			} else if errorRate > 10 {
				status = "⚠️ degraded"
			}

			health.Servers = append(health.Servers, MCPServerHealthDetail{
				ServerName:   server.ServerName,
				RequestCount: server.RequestCount,
				ToolCalls:    server.ToolCallCount,
				ErrorCount:   server.ErrorCount,
				ErrorRate:    errorRate,
				ErrorRateStr: fmt.Sprintf("%.1f%%", errorRate),
				AvgLatency:   server.AvgDuration,
				Status:       status,
			})
		}

		// Build slowest tool calls from individual call records (top 5)
		health.SlowestCalls = buildSlowestToolCalls(mcpToolUsage.ToolCalls, 5)
	}

	// Add failed servers that don't appear in stats
	for serverName := range failedServers {
		found := false
		for _, s := range health.Servers {
			if s.ServerName == serverName {
				found = true
				break
			}
		}
		if !found {
			health.Servers = append(health.Servers, MCPServerHealthDetail{
				ServerName: serverName,
				Status:     "❌ failed",
			})
		}
	}

	health.TotalServers = len(health.Servers)

	// Count servers by status for accurate summary
	degradedCount := 0
	for _, s := range health.Servers {
		if strings.Contains(s.Status, "degraded") {
			degradedCount++
		}
	}
	health.DegradedSvrs = degradedCount
	health.HealthySvrs = health.TotalServers - health.FailedSvrs - health.DegradedSvrs

	// Calculate overall error rate
	if health.TotalRequests > 0 {
		health.ErrorRate = float64(health.TotalErrors) / float64(health.TotalRequests) * 100
	}

	// Sort servers by request count (highest first)
	sort.Slice(health.Servers, func(i, j int) bool {
		return health.Servers[i].RequestCount > health.Servers[j].RequestCount
	})

	// Build summary string
	health.Summary = fmt.Sprintf("%d server(s), %d healthy, %d degraded, %d failed",
		health.TotalServers, health.HealthySvrs, health.DegradedSvrs, health.FailedSvrs)

	auditExpandedLog.Printf("Built MCP server health: %s, total_requests=%d, error_rate=%.1f%%",
		health.Summary, health.TotalRequests, health.ErrorRate)
	return health
}

// buildSlowestToolCalls extracts the N slowest tool calls from the call records
func buildSlowestToolCalls(calls []MCPToolCall, topN int) []MCPSlowestToolCall {
	if len(calls) == 0 {
		return nil
	}

	// Filter calls that have duration information
	type callWithDuration struct {
		call     MCPToolCall
		duration time.Duration
	}

	var withDuration []callWithDuration
	for _, call := range calls {
		if call.Duration == "" {
			continue
		}
		d, err := time.ParseDuration(call.Duration)
		if err != nil {
			// Try parsing as bare number (milliseconds) only if no unit suffix present
			if !strings.ContainsAny(call.Duration, "smhμnuµ") {
				d, err = time.ParseDuration(call.Duration + "ms")
			}
			if err != nil {
				continue
			}
		}
		withDuration = append(withDuration, callWithDuration{call: call, duration: d})
	}

	// Sort by duration descending
	sort.Slice(withDuration, func(i, j int) bool {
		return withDuration[i].duration > withDuration[j].duration
	})

	// Take top N
	if len(withDuration) > topN {
		withDuration = withDuration[:topN]
	}

	result := make([]MCPSlowestToolCall, 0, len(withDuration))
	for _, wd := range withDuration {
		result = append(result, MCPSlowestToolCall{
			ServerName: wd.call.ServerName,
			ToolName:   wd.call.ToolName,
			Duration:   timeutil.FormatDuration(wd.duration),
		})
	}

	return result
}

// extractMCPServerNamesFromAwInfo extracts MCP server names from aw_info.json steps metadata
// and returns them along with a boolean indicating whether any servers were found.
// We need to inspect the raw JSON since AwInfoSteps.MCPServers may not be
// deserialized as a map for all formats.
func extractMCPServerNamesFromAwInfo(logsPath string) ([]string, bool) {
	awInfoPath := findAwInfoPath(logsPath)
	if awInfoPath == "" {
		return nil, false
	}
	data, err := os.ReadFile(awInfoPath)
	if err != nil {
		return nil, false
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, false
	}

	stepsRaw, ok := raw["steps"]
	if !ok {
		return nil, false
	}

	var steps map[string]json.RawMessage
	if err := json.Unmarshal(stepsRaw, &steps); err != nil {
		return nil, false
	}

	mcpRaw, ok := steps["mcp_servers"]
	if !ok {
		return nil, false
	}

	var servers map[string]json.RawMessage
	if err := json.Unmarshal(mcpRaw, &servers); err != nil {
		return nil, false
	}

	names := make([]string, 0, len(servers))
	for name := range servers {
		names = append(names, name)
	}
	sort.Strings(names)
	return names, len(names) > 0
}
