// This file contains the extractMCPToolUsageData function for MCP gateway log analysis.
// It orchestrates gateway/rpc-messages log parsing to produce MCPToolUsageData.

package cli

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/github/gh-aw/pkg/errorutil"
	"github.com/github/gh-aw/pkg/timeutil"
)

// extractMCPToolUsageData creates detailed MCP tool usage data from gateway metrics
func extractMCPToolUsageData(logDir string, verbose bool) (*MCPToolUsageData, error) {
	gatewayLogsLog.Printf("Extracting MCP tool usage data from: %s", logDir)

	// Parse gateway logs (falls back to rpc-messages.jsonl automatically)
	gatewayMetrics, err := parseGatewayLogs(logDir, verbose)
	if err != nil {
		// Return nil if no log file exists (not an error for workflows without MCP)
		if errorutil.IsNotFoundError(err) {
			gatewayLogsLog.Print("No gateway log file found, skipping MCP tool usage extraction")
			return nil, nil
		}
		return nil, fmt.Errorf("failed to parse gateway logs: %w", err)
	}

	if gatewayMetrics == nil || len(gatewayMetrics.Servers) == 0 {
		gatewayLogsLog.Print("No gateway metrics or servers found")
		return nil, nil
	}
	gatewayLogsLog.Printf("Found gateway metrics: %d servers", len(gatewayMetrics.Servers))

	mcpData := &MCPToolUsageData{
		Summary:        []MCPToolSummary{},
		ToolCalls:      []MCPToolCall{},
		Servers:        []MCPServerStats{},
		FilteredEvents: gatewayMetrics.FilteredEvents,
	}

	// Build guard policy summary if there are guard policy events
	if len(gatewayMetrics.GuardPolicyEvents) > 0 {
		mcpData.GuardPolicySummary = buildGuardPolicySummary(gatewayMetrics)
	}

	gatewayLogPath, usingRPCMessages, err := extractMCPToolUsageDataLogPath(logDir)
	if err != nil {
		return nil, err
	}

	if err := extractMCPToolUsageDataToolCalls(logDir, gatewayLogPath, usingRPCMessages, mcpData); err != nil {
		return nil, err
	}

	// Build summary statistics from aggregated metrics
	buildMCPSummaryStats(gatewayMetrics, mcpData)
	gatewayLogsLog.Printf("Built MCP summary: %d tool summaries, %d server stats", len(mcpData.Summary), len(mcpData.Servers))

	return mcpData, nil
}

func extractMCPToolUsageDataLogPath(logDir string) (string, bool, error) {
	// Read the log file again to get individual tool call records.
	// Prefer gateway.jsonl; fall back to rpc-messages.jsonl when not available.
	gatewayLogPath := filepath.Join(logDir, "gateway.jsonl")
	if _, err := os.Stat(gatewayLogPath); err == nil {
		return gatewayLogPath, false, nil
	}
	mcpLogsPath := filepath.Join(logDir, "mcp-logs", "gateway.jsonl")
	if _, err := os.Stat(mcpLogsPath); err == nil {
		return mcpLogsPath, false, nil
	}
	rpcPath := findRPCMessagesPath(logDir)
	if rpcPath == "" {
		return "", false, errors.New("gateway.jsonl not found")
	}
	return rpcPath, true, nil
}

func extractMCPToolUsageDataToolCalls(logDir, gatewayLogPath string, usingRPCMessages bool, mcpData *MCPToolUsageData) error {
	if usingRPCMessages {
		gatewayLogsLog.Printf("Reading tool calls from rpc-messages.jsonl: %s", gatewayLogPath)
		toolCalls, err := buildToolCallsFromRPCMessages(gatewayLogPath)
		if err != nil {
			return fmt.Errorf("failed to read rpc-messages.jsonl: %w", err)
		}
		tokenUsageFile := findTokenUsageFile(logDir)
		mcpData.ToolCalls = correlateToolCallsWithTokenDelta(toolCalls, tokenUsageFile)
		gatewayLogsLog.Printf("Loaded %d tool calls from rpc-messages.jsonl", len(mcpData.ToolCalls))
		return nil
	}
	gatewayLogsLog.Printf("Reading tool calls from gateway.jsonl: %s", gatewayLogPath)
	if err := extractToolCallsFromGatewayLog(gatewayLogPath, mcpData); err != nil {
		return err
	}
	tokenUsageFile := findTokenUsageFile(logDir)
	mcpData.ToolCalls = correlateToolCallsWithTokenDelta(mcpData.ToolCalls, tokenUsageFile)
	gatewayLogsLog.Printf("Loaded %d tool calls from gateway.jsonl", len(mcpData.ToolCalls))
	return nil
}

// extractToolCallsFromGatewayLog reads gateway.jsonl and appends tool call records to mcpData.
func extractToolCallsFromGatewayLog(gatewayLogPath string, mcpData *MCPToolUsageData) error {
	file, err := os.Open(gatewayLogPath)
	if err != nil {
		return fmt.Errorf("failed to open gateway.jsonl: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	buf := make([]byte, maxScannerBufferSize)
	scanner.Buffer(buf, maxScannerBufferSize)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var entry GatewayLogEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue // Skip malformed lines
		}

		if toolCall, ok := extractToolCallsFromGatewayLogEntry(entry); ok {
			mcpData.ToolCalls = append(mcpData.ToolCalls, toolCall)
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading gateway.jsonl: %w", err)
	}
	return nil
}

func extractToolCallsFromGatewayLogEntry(entry GatewayLogEntry) (MCPToolCall, bool) {
	// Only process tool call events
	if entry.Event != "tool_call" && entry.Event != "rpc_call" && entry.Event != "request" {
		return MCPToolCall{}, false
	}
	toolName := entry.ToolName
	if toolName == "" {
		toolName = entry.Method
	}
	if entry.ServerName == "" || toolName == "" {
		return MCPToolCall{}, false
	}

	toolCall := MCPToolCall{
		Timestamp:  entry.Timestamp,
		ServerName: entry.ServerName,
		ToolName:   toolName,
		Method:     entry.Method,
		InputSize:  entry.InputSize,
		OutputSize: entry.OutputSize,
		Status:     extractToolCallsFromGatewayLogStatus(entry),
		Error:      entry.Error,
	}
	if entry.Duration > 0 {
		toolCall.Duration = timeutil.FormatDuration(time.Duration(entry.Duration * float64(time.Millisecond)))
	}
	return toolCall, true
}

func extractToolCallsFromGatewayLogStatus(entry GatewayLogEntry) string {
	if entry.Status != "" {
		return entry.Status
	}
	if entry.Error != "" || entry.Level == "error" {
		return "error"
	}
	return "success"
}

// buildMCPSummaryStats populates mcpData.Summary and mcpData.Servers from aggregated gateway metrics.
func buildMCPSummaryStats(gatewayMetrics *GatewayMetrics, mcpData *MCPToolUsageData) {
	for serverName, serverMetrics := range gatewayMetrics.Servers {
		serverStats := buildMCPSummaryStatsServer(serverName, serverMetrics, mcpData)
		mcpData.Servers = append(mcpData.Servers, serverStats)
	}

	buildMCPSummaryStatsSort(mcpData)
}

func buildMCPSummaryStatsServer(serverName string, serverMetrics *GatewayServerMetrics, mcpData *MCPToolUsageData) MCPServerStats {
	serverStats := MCPServerStats{
		ServerName:    serverName,
		RequestCount:  serverMetrics.RequestCount,
		ToolCallCount: serverMetrics.ToolCallCount,
		ErrorCount:    serverMetrics.ErrorCount,
	}
	if serverMetrics.RequestCount > 0 {
		avgDur := serverMetrics.TotalDuration / float64(serverMetrics.RequestCount)
		serverStats.AvgDuration = timeutil.FormatDuration(time.Duration(avgDur * float64(time.Millisecond)))
	}
	for toolName, toolMetrics := range serverMetrics.Tools {
		summary := buildMCPSummaryStatsTool(serverName, toolName, toolMetrics, mcpData.ToolCalls)
		mcpData.Summary = append(mcpData.Summary, summary)
		serverStats.TotalInputSize += toolMetrics.TotalInputSize
		serverStats.TotalOutputSize += toolMetrics.TotalOutputSize
	}
	return serverStats
}

func buildMCPSummaryStatsTool(serverName, toolName string, toolMetrics *GatewayToolMetrics, toolCalls []MCPToolCall) MCPToolSummary {
	summary := MCPToolSummary{
		ServerName:      serverName,
		ToolName:        toolName,
		CallCount:       toolMetrics.CallCount,
		TotalInputSize:  toolMetrics.TotalInputSize,
		TotalOutputSize: toolMetrics.TotalOutputSize,
		ErrorCount:      toolMetrics.ErrorCount,
	}
	if toolMetrics.AvgDuration > 0 {
		summary.AvgDuration = timeutil.FormatDuration(time.Duration(toolMetrics.AvgDuration * float64(time.Millisecond)))
	}
	if toolMetrics.MaxDuration > 0 {
		summary.MaxDuration = timeutil.FormatDuration(time.Duration(toolMetrics.MaxDuration * float64(time.Millisecond)))
	}
	for _, tc := range toolCalls {
		if tc.ServerName == serverName && tc.ToolName == toolName {
			summary.MaxInputSize = max(summary.MaxInputSize, tc.InputSize)
			summary.MaxOutputSize = max(summary.MaxOutputSize, tc.OutputSize)
		}
	}
	return summary
}

func buildMCPSummaryStatsSort(mcpData *MCPToolUsageData) {
	slices.SortFunc(mcpData.Summary, func(a, b MCPToolSummary) int {
		if a.ServerName != b.ServerName {
			if a.ServerName < b.ServerName {
				return -1
			}
			return 1
		}
		switch {
		case a.ToolName < b.ToolName:
			return -1
		case a.ToolName > b.ToolName:
			return 1
		default:
			return 0
		}
	})

	slices.SortFunc(mcpData.Servers, func(a, b MCPServerStats) int {
		switch {
		case a.ServerName < b.ServerName:
			return -1
		case a.ServerName > b.ServerName:
			return 1
		default:
			return 0
		}
	})
}
