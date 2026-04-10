// This file provides rendering and display functions for MCP gateway metrics.
//
// It handles:
//   - Rendering gateway metrics as console tables (renderGatewayMetricsTable)
//   - Building individual tool call records (buildToolCallsFromRPCMessages)
//   - Extracting MCP tool usage data for the audit report (extractMCPToolUsageData)
//   - Building guard policy summaries (buildGuardPolicySummary)
//   - Displaying aggregated metrics across multiple runs (displayAggregatedGatewayMetrics)
//
// Type definitions are in gateway_logs_types.go.
// Log file parsing is in gateway_logs_parser.go.
// Metrics computation helpers are in gateway_logs_metrics.go.

package cli

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/sliceutil"
	"github.com/github/gh-aw/pkg/timeutil"
)

// renderGatewayMetricsTable renders gateway metrics as a console table
func renderGatewayMetricsTable(metrics *GatewayMetrics, verbose bool) string {
	if metrics == nil || len(metrics.Servers) == 0 {
		return ""
	}

	var output strings.Builder

	output.WriteString("\n")
	output.WriteString(console.FormatInfoMessage("MCP Gateway Metrics"))
	output.WriteString("\n\n")

	// Summary statistics
	fmt.Fprintf(&output, "Total Requests: %d\n", metrics.TotalRequests)
	fmt.Fprintf(&output, "Total Tool Calls: %d\n", metrics.TotalToolCalls)
	fmt.Fprintf(&output, "Total Errors: %d\n", metrics.TotalErrors)
	if metrics.TotalFiltered > 0 {
		fmt.Fprintf(&output, "Total DIFC Filtered: %d\n", metrics.TotalFiltered)
	}
	if metrics.TotalGuardBlocked > 0 {
		fmt.Fprintf(&output, "Total Guard Policy Blocked: %d\n", metrics.TotalGuardBlocked)
	}
	fmt.Fprintf(&output, "Servers: %d\n", len(metrics.Servers))

	if !metrics.StartTime.IsZero() && !metrics.EndTime.IsZero() {
		duration := metrics.EndTime.Sub(metrics.StartTime)
		fmt.Fprintf(&output, "Time Range: %s\n", duration.Round(time.Second))
	}

	output.WriteString("\n")

	// Server metrics table
	if len(metrics.Servers) > 0 {
		// Sort servers by request count
		serverNames := getSortedServerNames(metrics)

		hasFiltered := metrics.TotalFiltered > 0
		hasGuardPolicy := metrics.TotalGuardBlocked > 0
		serverRows := make([][]string, 0, len(serverNames))
		for _, serverName := range serverNames {
			server := metrics.Servers[serverName]
			avgTime := 0.0
			if server.RequestCount > 0 {
				avgTime = server.TotalDuration / float64(server.RequestCount)
			}
			row := []string{
				serverName,
				strconv.Itoa(server.RequestCount),
				strconv.Itoa(server.ToolCallCount),
				fmt.Sprintf("%.0fms", avgTime),
				strconv.Itoa(server.ErrorCount),
			}
			if hasFiltered {
				row = append(row, strconv.Itoa(server.FilteredCount))
			}
			if hasGuardPolicy {
				row = append(row, strconv.Itoa(server.GuardPolicyBlocked))
			}
			serverRows = append(serverRows, row)
		}

		headers := []string{"Server", "Requests", "Tool Calls", "Avg Time", "Errors"}
		if hasFiltered {
			headers = append(headers, "Filtered")
		}
		if hasGuardPolicy {
			headers = append(headers, "Guard Blocked")
		}
		output.WriteString(console.RenderTable(console.TableConfig{
			Title:   "Server Usage",
			Headers: headers,
			Rows:    serverRows,
		}))
	}

	// DIFC filtered events table
	if len(metrics.FilteredEvents) > 0 {
		output.WriteString("\n")
		filteredRows := make([][]string, 0, len(metrics.FilteredEvents))
		for _, fe := range metrics.FilteredEvents {
			reason := fe.Reason
			if len(reason) > 80 {
				reason = reason[:77] + "..."
			}
			filteredRows = append(filteredRows, []string{
				fe.ServerID,
				fe.ToolName,
				fe.AuthorLogin,
				reason,
			})
		}
		output.WriteString(console.RenderTable(console.TableConfig{
			Title:   "DIFC Filtered Events",
			Headers: []string{"Server", "Tool", "User", "Reason"},
			Rows:    filteredRows,
		}))
	}

	// Guard policy events table
	if len(metrics.GuardPolicyEvents) > 0 {
		output.WriteString("\n")
		guardRows := make([][]string, 0, len(metrics.GuardPolicyEvents))
		for _, gpe := range metrics.GuardPolicyEvents {
			message := gpe.Message
			if len(message) > 60 {
				message = message[:57] + "..."
			}
			repo := gpe.Repository
			if repo == "" {
				repo = "-"
			}
			guardRows = append(guardRows, []string{
				gpe.ServerID,
				gpe.ToolName,
				gpe.Reason,
				message,
				repo,
			})
		}
		output.WriteString(console.RenderTable(console.TableConfig{
			Title:   "Guard Policy Blocked Events",
			Headers: []string{"Server", "Tool", "Reason", "Message", "Repository"},
			Rows:    guardRows,
		}))
	}

	// Tool metrics table (if verbose)
	if verbose {
		output.WriteString("\n")
		output.WriteString("Tool Usage Details:\n")

		for _, serverName := range getSortedServerNames(metrics) {
			server := metrics.Servers[serverName]
			if len(server.Tools) == 0 {
				continue
			}

			// Sort tools by call count
			toolNames := sliceutil.MapToSlice(server.Tools)
			sort.Slice(toolNames, func(i, j int) bool {
				return server.Tools[toolNames[i]].CallCount > server.Tools[toolNames[j]].CallCount
			})

			toolRows := make([][]string, 0, len(toolNames))
			for _, toolName := range toolNames {
				tool := server.Tools[toolName]
				toolRows = append(toolRows, []string{
					toolName,
					strconv.Itoa(tool.CallCount),
					fmt.Sprintf("%.0fms", tool.AvgDuration),
					fmt.Sprintf("%.0fms", tool.MaxDuration),
					strconv.Itoa(tool.ErrorCount),
				})
			}

			output.WriteString(console.RenderTable(console.TableConfig{
				Title:   serverName,
				Headers: []string{"Tool", "Calls", "Avg Time", "Max Time", "Errors"},
				Rows:    toolRows,
			}))
		}
	}

	return output.String()
}

// getSortedServerNames returns server names sorted by request count
func getSortedServerNames(metrics *GatewayMetrics) []string {
	names := sliceutil.MapToSlice(metrics.Servers)
	sort.Slice(names, func(i, j int) bool {
		return metrics.Servers[names[i]].RequestCount > metrics.Servers[names[j]].RequestCount
	})
	return names
}

// buildToolCallsFromRPCMessages reads rpc-messages.jsonl and builds MCPToolCall records.
// Duration is computed by pairing outgoing requests with incoming responses.
// Input/output sizes are not available in rpc-messages.jsonl and will be 0.
func buildToolCallsFromRPCMessages(logPath string) ([]MCPToolCall, error) {
	file, err := os.Open(logPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open rpc-messages.jsonl: %w", err)
	}
	defer file.Close()

	type pendingCall struct {
		serverID  string
		toolName  string
		timestamp time.Time
	}
	pending := make(map[string]*pendingCall) // key: "<serverID>/<id>"

	// Collect requests first to pair with responses
	type rawEntry struct {
		entry RPCMessageEntry
		req   rpcRequestPayload
		resp  rpcResponsePayload
		valid bool
	}
	var entries []rawEntry

	scanner := bufio.NewScanner(file)
	buf := make([]byte, maxScannerBufferSize)
	scanner.Buffer(buf, maxScannerBufferSize)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var e RPCMessageEntry
		if err := json.Unmarshal([]byte(line), &e); err != nil {
			continue
		}
		entries = append(entries, rawEntry{entry: e, valid: true})
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading rpc-messages.jsonl: %w", err)
	}

	// toolCalls and processedKeys are declared here so requests without IDs can be
	// appended during the first pass (index pass) below, before the second pass runs.
	var toolCalls []MCPToolCall
	processedKeys := make(map[string]bool)

	// First pass: index outgoing tool-call requests by (serverID, id)
	for i := range entries {
		e := &entries[i]
		if e.entry.Direction != "OUT" || e.entry.Type != "REQUEST" {
			continue
		}
		if err := json.Unmarshal(e.entry.Payload, &e.req); err != nil || e.req.Method != "tools/call" {
			continue
		}
		var params rpcToolCallParams
		if err := json.Unmarshal(e.req.Params, &params); err != nil || params.Name == "" {
			continue
		}
		if e.req.ID == nil {
			// Requests without an ID cannot be matched to responses.
			// Emit the tool call immediately with "unknown" status so it appears
			// in the tool_calls list (same as parseRPCMessages counts it in the summary).
			toolCalls = append(toolCalls, MCPToolCall{
				Timestamp:  e.entry.Timestamp,
				ServerName: e.entry.ServerID,
				ToolName:   params.Name,
				Status:     "unknown",
			})
			continue
		}
		t, err := time.Parse(time.RFC3339Nano, e.entry.Timestamp)
		if err != nil {
			continue
		}
		key := fmt.Sprintf("%s/%v", e.entry.ServerID, e.req.ID)
		pending[key] = &pendingCall{
			serverID:  e.entry.ServerID,
			toolName:  params.Name,
			timestamp: t,
		}
	}

	// Second pass: match incoming responses with pending requests to compute durations

	for i := range entries {
		e := &entries[i]
		switch {
		case e.entry.Direction == "OUT" && e.entry.Type == "REQUEST":
			// Outgoing tool-call request – we'll emit the record when we see the response
			// (or after if no response found)
		case e.entry.Direction == "IN" && e.entry.Type == "RESPONSE":
			if err := json.Unmarshal(e.entry.Payload, &e.resp); err != nil {
				continue
			}
			if e.resp.ID == nil {
				continue
			}
			key := fmt.Sprintf("%s/%v", e.entry.ServerID, e.resp.ID)
			p, ok := pending[key]
			if !ok {
				continue
			}
			processedKeys[key] = true

			call := MCPToolCall{
				Timestamp:  p.timestamp.Format(time.RFC3339Nano),
				ServerName: p.serverID,
				ToolName:   p.toolName,
				Status:     "success",
			}
			if e.resp.Error != nil {
				call.Status = "error"
				call.Error = e.resp.Error.Message
			}
			if t, err := time.Parse(time.RFC3339Nano, e.entry.Timestamp); err == nil {
				d := t.Sub(p.timestamp)
				if d >= 0 {
					call.Duration = timeutil.FormatDuration(d)
				}
			}
			toolCalls = append(toolCalls, call)
		}
	}

	// Emit any requests that never received a response
	for key, p := range pending {
		if !processedKeys[key] {
			toolCalls = append(toolCalls, MCPToolCall{
				Timestamp:  p.timestamp.Format(time.RFC3339Nano),
				ServerName: p.serverID,
				ToolName:   p.toolName,
				Status:     "unknown",
			})
		}
	}

	return toolCalls, nil
}

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
		file, err := os.Open(gatewayLogPath)
		if err != nil {
			return nil, fmt.Errorf("failed to open gateway.jsonl: %w", err)
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
			return nil, fmt.Errorf("error reading gateway.jsonl: %w", err)
		}
	}

	// Build summary statistics from aggregated metrics
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

	return mcpData, nil
}

// buildGuardPolicySummary creates a GuardPolicySummary from GatewayMetrics.
func buildGuardPolicySummary(metrics *GatewayMetrics) *GuardPolicySummary {
	summary := &GuardPolicySummary{
		TotalBlocked:        metrics.TotalGuardBlocked,
		Events:              metrics.GuardPolicyEvents,
		BlockedToolCounts:   make(map[string]int),
		BlockedServerCounts: make(map[string]int),
	}

	for _, evt := range metrics.GuardPolicyEvents {
		// Categorize by error code
		switch evt.ErrorCode {
		case guardPolicyErrorCodeIntegrityBelowMin:
			summary.IntegrityBlocked++
		case guardPolicyErrorCodeRepoNotAllowed:
			summary.RepoScopeBlocked++
		case guardPolicyErrorCodeAccessDenied:
			summary.AccessDenied++
		case guardPolicyErrorCodeBlockedUser:
			summary.BlockedUserDenied++
		case guardPolicyErrorCodeInsufficientPerms:
			summary.PermissionDenied++
		case guardPolicyErrorCodePrivateRepoDenied:
			summary.PrivateRepoDenied++
		}

		// Track per-tool blocked counts
		if evt.ToolName != "" {
			summary.BlockedToolCounts[evt.ToolName]++
		}

		// Track per-server blocked counts
		if evt.ServerID != "" {
			summary.BlockedServerCounts[evt.ServerID]++
		}
	}

	return summary
}

// displayAggregatedGatewayMetrics aggregates and displays gateway metrics across all processed runs
func displayAggregatedGatewayMetrics(processedRuns []ProcessedRun, outputDir string, verbose bool) {
	// Aggregate gateway metrics from all runs
	aggregated := &GatewayMetrics{
		Servers: make(map[string]*GatewayServerMetrics),
	}

	runCount := 0
	for _, pr := range processedRuns {
		runDir := pr.Run.LogsPath
		if runDir == "" {
			continue
		}

		// Try to parse gateway.jsonl from this run
		runMetrics, err := parseGatewayLogs(runDir, false)
		if err != nil {
			// Skip runs without gateway.jsonl (this is normal for runs without MCP gateway)
			continue
		}

		runCount++

		// Merge metrics from this run into aggregated metrics
		aggregated.TotalRequests += runMetrics.TotalRequests
		aggregated.TotalToolCalls += runMetrics.TotalToolCalls
		aggregated.TotalErrors += runMetrics.TotalErrors
		aggregated.TotalFiltered += runMetrics.TotalFiltered
		aggregated.TotalGuardBlocked += runMetrics.TotalGuardBlocked
		aggregated.TotalDuration += runMetrics.TotalDuration
		aggregated.FilteredEvents = append(aggregated.FilteredEvents, runMetrics.FilteredEvents...)
		aggregated.GuardPolicyEvents = append(aggregated.GuardPolicyEvents, runMetrics.GuardPolicyEvents...)

		// Merge server metrics
		for serverName, serverMetrics := range runMetrics.Servers {
			aggServer := getOrCreateServer(aggregated, serverName)
			aggServer.RequestCount += serverMetrics.RequestCount
			aggServer.ToolCallCount += serverMetrics.ToolCallCount
			aggServer.TotalDuration += serverMetrics.TotalDuration
			aggServer.ErrorCount += serverMetrics.ErrorCount
			aggServer.FilteredCount += serverMetrics.FilteredCount
			aggServer.GuardPolicyBlocked += serverMetrics.GuardPolicyBlocked

			// Merge tool metrics
			for toolName, toolMetrics := range serverMetrics.Tools {
				aggTool := getOrCreateTool(aggServer, toolName)
				aggTool.CallCount += toolMetrics.CallCount
				aggTool.TotalDuration += toolMetrics.TotalDuration
				aggTool.ErrorCount += toolMetrics.ErrorCount
				aggTool.TotalInputSize += toolMetrics.TotalInputSize
				aggTool.TotalOutputSize += toolMetrics.TotalOutputSize

				// Update max/min durations
				if toolMetrics.MaxDuration > aggTool.MaxDuration {
					aggTool.MaxDuration = toolMetrics.MaxDuration
				}
				if aggTool.MinDuration == 0 || (toolMetrics.MinDuration > 0 && toolMetrics.MinDuration < aggTool.MinDuration) {
					aggTool.MinDuration = toolMetrics.MinDuration
				}
			}
		}

		// Update time range
		if aggregated.StartTime.IsZero() || (!runMetrics.StartTime.IsZero() && runMetrics.StartTime.Before(aggregated.StartTime)) {
			aggregated.StartTime = runMetrics.StartTime
		}
		if aggregated.EndTime.IsZero() || (!runMetrics.EndTime.IsZero() && runMetrics.EndTime.After(aggregated.EndTime)) {
			aggregated.EndTime = runMetrics.EndTime
		}
	}

	// Only display if we found gateway metrics
	if runCount == 0 || len(aggregated.Servers) == 0 {
		return
	}

	// Recalculate averages for aggregated data
	calculateGatewayAggregates(aggregated)

	// Display the aggregated metrics
	if metricsOutput := renderGatewayMetricsTable(aggregated, verbose); metricsOutput != "" {
		fmt.Fprint(os.Stderr, metricsOutput)
		if runCount > 1 {
			fmt.Fprintf(os.Stderr, "\n%s\n",
				console.FormatInfoMessage(fmt.Sprintf("Gateway metrics aggregated from %d runs", runCount)))
		}
	}
}
