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
	"sort"
	"strings"
	"time"

	"github.com/github/gh-aw/pkg/timeutil"
)

// extractMCPToolUsageData creates detailed MCP tool usage data from gateway metrics
func extractMCPToolUsageData(logDir string, verbose bool) (*MCPToolUsageData, error) {
	// Parse gateway logs (falls back to rpc-messages.jsonl automatically)
	gatewayMetrics, err := parseGatewayLogs(logDir, verbose)
	if err != nil {
		// Return nil if no log file exists (not an error for workflows without MCP)
		if strings.Contains(err.Error(), "not found") {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to parse gateway logs: %w", err)
	}

	if gatewayMetrics == nil || len(gatewayMetrics.Servers) == 0 {
		return nil, nil
	}

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

	// Read the log file again to get individual tool call records.
	// Prefer gateway.jsonl; fall back to rpc-messages.jsonl when not available.
	gatewayLogPath := filepath.Join(logDir, "gateway.jsonl")
	usingRPCMessages := false

	if _, err := os.Stat(gatewayLogPath); os.IsNotExist(err) {
		mcpLogsPath := filepath.Join(logDir, "mcp-logs", "gateway.jsonl")
		if _, err := os.Stat(mcpLogsPath); os.IsNotExist(err) {
			// Fall back to rpc-messages.jsonl
			rpcPath := findRPCMessagesPath(logDir)
			if rpcPath == "" {
				return nil, errors.New("gateway.jsonl not found")
			}
			gatewayLogPath = rpcPath
			usingRPCMessages = true
		} else {
			gatewayLogPath = mcpLogsPath
		}
	}

	if usingRPCMessages {
		// Build tool call records from rpc-messages.jsonl
		toolCalls, err := buildToolCallsFromRPCMessages(gatewayLogPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read rpc-messages.jsonl: %w", err)
		}
		mcpData.ToolCalls = toolCalls
	} else {
		if err := extractToolCallsFromGatewayLog(gatewayLogPath, mcpData); err != nil {
			return nil, err
		}
	}

	// Build summary statistics from aggregated metrics
	buildMCPSummaryStats(gatewayMetrics, mcpData)

	return mcpData, nil
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

		// Only process tool call events
		if entry.Event == "tool_call" || entry.Event == "rpc_call" || entry.Event == "request" {
			toolName := entry.ToolName
			if toolName == "" {
				toolName = entry.Method
			}

			// Skip entries without tool information
			if entry.ServerName == "" || toolName == "" {
				continue
			}

			// Create individual tool call record
			toolCall := MCPToolCall{
				Timestamp:  entry.Timestamp,
				ServerName: entry.ServerName,
				ToolName:   toolName,
				Method:     entry.Method,
				InputSize:  entry.InputSize,
				OutputSize: entry.OutputSize,
				Status:     entry.Status,
				Error:      entry.Error,
			}

			if entry.Duration > 0 {
				toolCall.Duration = timeutil.FormatDuration(time.Duration(entry.Duration * float64(time.Millisecond)))
			}

			mcpData.ToolCalls = append(mcpData.ToolCalls, toolCall)
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading gateway.jsonl: %w", err)
	}
	return nil
}

// buildMCPSummaryStats populates mcpData.Summary and mcpData.Servers from aggregated gateway metrics.
func buildMCPSummaryStats(gatewayMetrics *GatewayMetrics, mcpData *MCPToolUsageData) {
	for serverName, serverMetrics := range gatewayMetrics.Servers {
		// Server-level stats
		serverStats := MCPServerStats{
			ServerName:      serverName,
			RequestCount:    serverMetrics.RequestCount,
			ToolCallCount:   serverMetrics.ToolCallCount,
			TotalInputSize:  0,
			TotalOutputSize: 0,
			ErrorCount:      serverMetrics.ErrorCount,
		}

		if serverMetrics.RequestCount > 0 {
			avgDur := serverMetrics.TotalDuration / float64(serverMetrics.RequestCount)
			serverStats.AvgDuration = timeutil.FormatDuration(time.Duration(avgDur * float64(time.Millisecond)))
		}

		// Tool-level stats
		for toolName, toolMetrics := range serverMetrics.Tools {
			summary := MCPToolSummary{
				ServerName:      serverName,
				ToolName:        toolName,
				CallCount:       toolMetrics.CallCount,
				TotalInputSize:  toolMetrics.TotalInputSize,
				TotalOutputSize: toolMetrics.TotalOutputSize,
				MaxInputSize:    0, // Will be calculated below
				MaxOutputSize:   0, // Will be calculated below
				ErrorCount:      toolMetrics.ErrorCount,
			}

			if toolMetrics.AvgDuration > 0 {
				summary.AvgDuration = timeutil.FormatDuration(time.Duration(toolMetrics.AvgDuration * float64(time.Millisecond)))
			}
			if toolMetrics.MaxDuration > 0 {
				summary.MaxDuration = timeutil.FormatDuration(time.Duration(toolMetrics.MaxDuration * float64(time.Millisecond)))
			}

			// Calculate max input/output sizes from individual tool calls
			for _, tc := range mcpData.ToolCalls {
				if tc.ServerName == serverName && tc.ToolName == toolName {
					if tc.InputSize > summary.MaxInputSize {
						summary.MaxInputSize = tc.InputSize
					}
					if tc.OutputSize > summary.MaxOutputSize {
						summary.MaxOutputSize = tc.OutputSize
					}
				}
			}

			mcpData.Summary = append(mcpData.Summary, summary)

			// Update server totals
			serverStats.TotalInputSize += toolMetrics.TotalInputSize
			serverStats.TotalOutputSize += toolMetrics.TotalOutputSize
		}

		mcpData.Servers = append(mcpData.Servers, serverStats)
	}

	// Sort summaries by server name, then tool name
	sort.Slice(mcpData.Summary, func(i, j int) bool {
		if mcpData.Summary[i].ServerName != mcpData.Summary[j].ServerName {
			return mcpData.Summary[i].ServerName < mcpData.Summary[j].ServerName
		}
		return mcpData.Summary[i].ToolName < mcpData.Summary[j].ToolName
	})

	// Sort servers by name
	sort.Slice(mcpData.Servers, func(i, j int) bool {
		return mcpData.Servers[i].ServerName < mcpData.Servers[j].ServerName
	})
}
