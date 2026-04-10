// This file provides RPC message and gateway log parsing functions for gh-aw.
// It handles reading gateway.jsonl and rpc-messages.jsonl, building GatewayMetrics
// and MCPToolCall records, and computing aggregate statistics.

package cli

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/timeutil"
)

// parseRPCMessages parses a rpc-messages.jsonl file and extracts GatewayMetrics.
// This is the canonical fallback when gateway.jsonl is not available.
func parseRPCMessages(logPath string, verbose bool) (*GatewayMetrics, error) {
	gatewayLogsLog.Printf("Parsing rpc-messages.jsonl from: %s", logPath)

	file, err := os.Open(logPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open rpc-messages.jsonl: %w", err)
	}
	defer file.Close()

	metrics := &GatewayMetrics{
		Servers: make(map[string]*GatewayServerMetrics),
	}

	// Track pending requests by (serverID, id) for duration calculation.
	// Key format: "<serverID>/<id>"
	pendingRequests := make(map[string]*rpcPendingRequest)

	scanner := bufio.NewScanner(file)
	// Increase scanner buffer for large payloads
	buf := make([]byte, maxScannerBufferSize)
	scanner.Buffer(buf, maxScannerBufferSize)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var entry RPCMessageEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			gatewayLogsLog.Printf("Failed to parse rpc-messages.jsonl line %d: %v", lineNum, err)
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(
					fmt.Sprintf("Failed to parse rpc-messages.jsonl line %d: %v", lineNum, err)))
			}
			continue
		}

		// Update time range
		if entry.Timestamp != "" {
			if t, err := time.Parse(time.RFC3339Nano, entry.Timestamp); err == nil {
				if metrics.StartTime.IsZero() || t.Before(metrics.StartTime) {
					metrics.StartTime = t
				}
				if metrics.EndTime.IsZero() || t.After(metrics.EndTime) {
					metrics.EndTime = t
				}
			}
		}

		if entry.ServerID == "" {
			continue
		}

		switch {
		case entry.Type == "DIFC_FILTERED":
			// DIFC integrity/secrecy filter event — not a REQUEST or RESPONSE
			metrics.TotalFiltered++
			server := getOrCreateServer(metrics, entry.ServerID)
			server.FilteredCount++
			metrics.FilteredEvents = append(metrics.FilteredEvents, DifcFilteredEvent{
				Timestamp:         entry.Timestamp,
				ServerID:          entry.ServerID,
				ToolName:          entry.ToolName,
				Description:       entry.Description,
				Reason:            entry.Reason,
				SecrecyTags:       entry.SecrecyTags,
				IntegrityTags:     entry.IntegrityTags,
				AuthorAssociation: entry.AuthorAssociation,
				AuthorLogin:       entry.AuthorLogin,
				HTMLURL:           entry.HTMLURL,
				Number:            entry.Number,
			})

		case entry.Direction == "OUT" && entry.Type == "REQUEST":
			// Outgoing request from AI engine to MCP server
			var req rpcRequestPayload
			if err := json.Unmarshal(entry.Payload, &req); err != nil {
				continue
			}
			if req.Method != "tools/call" {
				continue
			}

			// Extract tool name
			var params rpcToolCallParams
			if err := json.Unmarshal(req.Params, &params); err != nil || params.Name == "" {
				continue
			}

			metrics.TotalRequests++
			server := getOrCreateServer(metrics, entry.ServerID)
			server.RequestCount++
			metrics.TotalToolCalls++
			server.ToolCallCount++

			tool := getOrCreateTool(server, params.Name)
			tool.CallCount++

			// Store pending request for duration calculation
			if req.ID != nil && entry.Timestamp != "" {
				if t, err := time.Parse(time.RFC3339Nano, entry.Timestamp); err == nil {
					key := fmt.Sprintf("%s/%v", entry.ServerID, req.ID)
					pendingRequests[key] = &rpcPendingRequest{
						ServerID:  entry.ServerID,
						ToolName:  params.Name,
						Timestamp: t,
					}
				}
			}

		case entry.Direction == "IN" && entry.Type == "RESPONSE":
			// Incoming response from MCP server to AI engine
			var resp rpcResponsePayload
			if err := json.Unmarshal(entry.Payload, &resp); err != nil {
				continue
			}

			// Track errors and detect guard policy blocks
			if resp.Error != nil {
				metrics.TotalErrors++
				server := getOrCreateServer(metrics, entry.ServerID)
				server.ErrorCount++

				// Detect guard policy enforcement errors
				if isGuardPolicyErrorCode(resp.Error.Code) {
					metrics.TotalGuardBlocked++
					server.GuardPolicyBlocked++

					// Determine tool name from pending request if available
					toolName := ""
					if resp.ID != nil {
						key := fmt.Sprintf("%s/%v", entry.ServerID, resp.ID)
						if pending, ok := pendingRequests[key]; ok {
							toolName = pending.ToolName
						}
					}

					reason := guardPolicyReasonFromCode(resp.Error.Code)
					if resp.Error.Data != nil && resp.Error.Data.Reason != "" {
						reason = resp.Error.Data.Reason
					}

					evt := GuardPolicyEvent{
						Timestamp: entry.Timestamp,
						ServerID:  entry.ServerID,
						ToolName:  toolName,
						ErrorCode: resp.Error.Code,
						Reason:    reason,
						Message:   resp.Error.Message,
					}
					if resp.Error.Data != nil {
						evt.Details = resp.Error.Data.Details
						evt.Repository = resp.Error.Data.Repository
					}
					metrics.GuardPolicyEvents = append(metrics.GuardPolicyEvents, evt)
				}
			}

			// Calculate duration by matching with pending request
			if resp.ID != nil && entry.Timestamp != "" {
				key := fmt.Sprintf("%s/%v", entry.ServerID, resp.ID)
				if pending, ok := pendingRequests[key]; ok {
					delete(pendingRequests, key)
					if t, err := time.Parse(time.RFC3339Nano, entry.Timestamp); err == nil {
						durationMs := float64(t.Sub(pending.Timestamp).Milliseconds())
						if durationMs >= 0 {
							server := getOrCreateServer(metrics, entry.ServerID)
							server.TotalDuration += durationMs
							metrics.TotalDuration += durationMs

							tool := getOrCreateTool(server, pending.ToolName)
							tool.TotalDuration += durationMs
							if tool.MaxDuration == 0 || durationMs > tool.MaxDuration {
								tool.MaxDuration = durationMs
							}
							if tool.MinDuration == 0 || durationMs < tool.MinDuration {
								tool.MinDuration = durationMs
							}

							if resp.Error != nil {
								tool.ErrorCount++
							}
						}
					}
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading rpc-messages.jsonl: %w", err)
	}

	calculateGatewayAggregates(metrics)

	gatewayLogsLog.Printf("Successfully parsed rpc-messages.jsonl: %d servers, %d total requests",
		len(metrics.Servers), metrics.TotalRequests)

	return metrics, nil
}

// findRPCMessagesPath returns the path to rpc-messages.jsonl if it exists, or "" if not found.
func findRPCMessagesPath(logDir string) string {
	// Check mcp-logs subdirectory (standard location)
	mcpLogsPath := filepath.Join(logDir, "mcp-logs", "rpc-messages.jsonl")
	if _, err := os.Stat(mcpLogsPath); err == nil {
		return mcpLogsPath
	}
	// Check root directory as fallback
	rootPath := filepath.Join(logDir, "rpc-messages.jsonl")
	if _, err := os.Stat(rootPath); err == nil {
		return rootPath
	}
	return ""
}

// parseGatewayLogs parses a gateway.jsonl file and extracts metrics.
// Falls back to rpc-messages.jsonl (canonical fallback) when gateway.jsonl is not present.
func parseGatewayLogs(logDir string, verbose bool) (*GatewayMetrics, error) {
	// Try root directory first (for older logs where gateway.jsonl was in the root)
	gatewayLogPath := filepath.Join(logDir, "gateway.jsonl")

	// Check if gateway.jsonl exists in root
	if _, err := os.Stat(gatewayLogPath); os.IsNotExist(err) {
		// Try mcp-logs subdirectory (new path after artifact download)
		// Gateway logs are uploaded from /tmp/gh-aw/mcp-logs/gateway.jsonl and the common parent
		// /tmp/gh-aw/ is stripped during artifact upload, resulting in mcp-logs/gateway.jsonl after download
		mcpLogsPath := filepath.Join(logDir, "mcp-logs", "gateway.jsonl")
		if _, err := os.Stat(mcpLogsPath); os.IsNotExist(err) {
			// Fall back to rpc-messages.jsonl (canonical fallback when gateway.jsonl is missing)
			rpcPath := findRPCMessagesPath(logDir)
			if rpcPath != "" {
				gatewayLogsLog.Printf("gateway.jsonl not found; falling back to rpc-messages.jsonl: %s", rpcPath)
				return parseRPCMessages(rpcPath, verbose)
			}
			gatewayLogsLog.Printf("gateway.jsonl not found at: %s or %s", gatewayLogPath, mcpLogsPath)
			return nil, errors.New("gateway.jsonl not found")
		}
		gatewayLogPath = mcpLogsPath
		gatewayLogsLog.Printf("Found gateway.jsonl in mcp-logs subdirectory")
	}

	gatewayLogsLog.Printf("Parsing gateway.jsonl from: %s", gatewayLogPath)

	file, err := os.Open(gatewayLogPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open gateway.jsonl: %w", err)
	}
	defer file.Close()

	metrics := &GatewayMetrics{
		Servers: make(map[string]*GatewayServerMetrics),
	}

	scanner := bufio.NewScanner(file)
	buf := make([]byte, maxScannerBufferSize)
	scanner.Buffer(buf, maxScannerBufferSize)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines
		if line == "" {
			continue
		}

		var entry GatewayLogEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			gatewayLogsLog.Printf("Failed to parse line %d: %v", lineNum, err)
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to parse gateway.jsonl line %d: %v", lineNum, err)))
			}
			continue
		}

		// Process the entry based on its type/event
		processGatewayLogEntry(&entry, metrics, verbose)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading gateway.jsonl: %w", err)
	}

	// Calculate aggregate statistics
	calculateGatewayAggregates(metrics)

	gatewayLogsLog.Printf("Successfully parsed gateway.jsonl: %d servers, %d total requests",
		len(metrics.Servers), metrics.TotalRequests)

	return metrics, nil
}

// processGatewayLogEntry processes a single log entry and updates metrics
func processGatewayLogEntry(entry *GatewayLogEntry, metrics *GatewayMetrics, verbose bool) {
	// Parse timestamp for time range (supports both RFC3339 and RFC3339Nano)
	if entry.Timestamp != "" {
		t, err := time.Parse(time.RFC3339Nano, entry.Timestamp)
		if err != nil {
			t, err = time.Parse(time.RFC3339, entry.Timestamp)
		}
		if err == nil {
			if metrics.StartTime.IsZero() || t.Before(metrics.StartTime) {
				metrics.StartTime = t
			}
			if metrics.EndTime.IsZero() || t.After(metrics.EndTime) {
				metrics.EndTime = t
			}
		}
	}

	// Handle DIFC_FILTERED events
	if entry.Type == "DIFC_FILTERED" {
		metrics.TotalFiltered++
		// DIFC_FILTERED events use server_id; fall back to server_name for compatibility
		serverKey := entry.ServerID
		if serverKey == "" {
			serverKey = entry.ServerName
		}
		if serverKey != "" {
			server := getOrCreateServer(metrics, serverKey)
			server.FilteredCount++
		}
		metrics.FilteredEvents = append(metrics.FilteredEvents, DifcFilteredEvent{
			Timestamp:         entry.Timestamp,
			ServerID:          serverKey,
			ToolName:          entry.ToolName,
			Description:       entry.Description,
			Reason:            entry.Reason,
			SecrecyTags:       entry.SecrecyTags,
			IntegrityTags:     entry.IntegrityTags,
			AuthorAssociation: entry.AuthorAssociation,
			AuthorLogin:       entry.AuthorLogin,
			HTMLURL:           entry.HTMLURL,
			Number:            entry.Number,
		})
		return
	}

	// Handle GUARD_POLICY_BLOCKED events from gateway.jsonl
	if entry.Type == "GUARD_POLICY_BLOCKED" {
		metrics.TotalGuardBlocked++
		serverKey := entry.ServerID
		if serverKey == "" {
			serverKey = entry.ServerName
		}
		if serverKey != "" {
			server := getOrCreateServer(metrics, serverKey)
			server.GuardPolicyBlocked++
		}
		metrics.GuardPolicyEvents = append(metrics.GuardPolicyEvents, GuardPolicyEvent{
			Timestamp: entry.Timestamp,
			ServerID:  serverKey,
			ToolName:  entry.ToolName,
			Reason:    entry.Reason,
			Message:   entry.Message,
			Details:   entry.Description,
		})
		return
	}

	// Track errors
	if entry.Status == "error" || entry.Error != "" {
		metrics.TotalErrors++
		if entry.ServerName != "" {
			server := getOrCreateServer(metrics, entry.ServerName)
			server.ErrorCount++

			if entry.ToolName != "" {
				tool := getOrCreateTool(server, entry.ToolName)
				tool.ErrorCount++
			}
		}
	}

	// Process based on event type
	switch entry.Event {
	case "request", "tool_call", "rpc_call":
		metrics.TotalRequests++

		if entry.ServerName != "" {
			server := getOrCreateServer(metrics, entry.ServerName)
			server.RequestCount++

			if entry.Duration > 0 {
				server.TotalDuration += entry.Duration
				metrics.TotalDuration += entry.Duration
			}

			// Track tool calls
			if entry.ToolName != "" || entry.Method != "" {
				toolName := entry.ToolName
				if toolName == "" {
					toolName = entry.Method
				}

				metrics.TotalToolCalls++
				server.ToolCallCount++

				tool := getOrCreateTool(server, toolName)
				tool.CallCount++

				if entry.Duration > 0 {
					tool.TotalDuration += entry.Duration
					if tool.MaxDuration == 0 || entry.Duration > tool.MaxDuration {
						tool.MaxDuration = entry.Duration
					}
					if tool.MinDuration == 0 || entry.Duration < tool.MinDuration {
						tool.MinDuration = entry.Duration
					}
				}

				if entry.InputSize > 0 {
					tool.TotalInputSize += entry.InputSize
				}
				if entry.OutputSize > 0 {
					tool.TotalOutputSize += entry.OutputSize
				}
			}
		}
	}
}

// getOrCreateServer gets or creates a server metrics entry
func getOrCreateServer(metrics *GatewayMetrics, serverName string) *GatewayServerMetrics {
	if server, exists := metrics.Servers[serverName]; exists {
		return server
	}

	server := &GatewayServerMetrics{
		ServerName: serverName,
		Tools:      make(map[string]*GatewayToolMetrics),
	}
	metrics.Servers[serverName] = server
	return server
}

// getOrCreateTool gets or creates a tool metrics entry
func getOrCreateTool(server *GatewayServerMetrics, toolName string) *GatewayToolMetrics {
	if tool, exists := server.Tools[toolName]; exists {
		return tool
	}

	tool := &GatewayToolMetrics{
		ToolName: toolName,
	}
	server.Tools[toolName] = tool
	return tool
}

// calculateGatewayAggregates calculates aggregate statistics
func calculateGatewayAggregates(metrics *GatewayMetrics) {
	for _, server := range metrics.Servers {
		for _, tool := range server.Tools {
			if tool.CallCount > 0 {
				tool.AvgDuration = tool.TotalDuration / float64(tool.CallCount)
			}
		}
	}
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

	// Second pass: build MCPToolCall records.
	// Declared before first pass so requests without IDs can be appended immediately.
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

	// Second pass: pair responses with pending requests to compute durations

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
