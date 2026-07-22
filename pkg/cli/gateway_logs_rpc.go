// This file contains RPC message parsing functions for MCP gateway log analysis.
// It handles rpc-messages.jsonl (canonical fallback when gateway.jsonl is absent).

package cli

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/setutil"
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
	pendingRequests := make(map[string]*rpcPendingRequest)

	if err := scanRPCMessages(file, metrics, pendingRequests, verbose); err != nil {
		return nil, err
	}

	calculateGatewayAggregates(metrics)
	gatewayLogsLog.Printf("Successfully parsed rpc-messages.jsonl: %d servers, %d total requests",
		len(metrics.Servers), metrics.TotalRequests)

	return metrics, nil
}

func scanRPCMessages(file *os.File, metrics *GatewayMetrics, pendingRequests map[string]*rpcPendingRequest, verbose bool) error {
	scanner := bufio.NewScanner(file)
	buf := make([]byte, maxScannerBufferSize)
	scanner.Buffer(buf, maxScannerBufferSize)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		entry, ok := parseRPCMessageEntry(line, lineNum, verbose)
		if !ok {
			continue
		}
		processRPCMessageEntry(metrics, pendingRequests, entry)
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading rpc-messages.jsonl: %w", err)
	}
	return nil
}

func parseRPCMessageEntry(line string, lineNum int, verbose bool) (RPCMessageEntry, bool) {
	var entry RPCMessageEntry
	if err := json.Unmarshal([]byte(line), &entry); err != nil {
		gatewayLogsLog.Printf("Failed to parse rpc-messages.jsonl line %d: %v", lineNum, err)
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(
				fmt.Sprintf("Failed to parse rpc-messages.jsonl line %d: %v", lineNum, err)))
		}
		return RPCMessageEntry{}, false
	}
	return entry, true
}

func processRPCMessageEntry(metrics *GatewayMetrics, pendingRequests map[string]*rpcPendingRequest, entry RPCMessageEntry) {
	updateGatewayMetricsTimeRange(metrics, entry.Timestamp)
	if entry.ServerID == "" {
		return
	}

	switch {
	case entry.Type == "DIFC_FILTERED":
		handleFilteredRPCMessage(metrics, entry)
	case entry.Direction == "OUT" && entry.Type == "REQUEST":
		handleRPCRequestMessage(metrics, pendingRequests, entry)
	case entry.Direction == "IN" && entry.Type == "RESPONSE":
		handleRPCResponseMessage(metrics, pendingRequests, entry)
	}
}

func updateGatewayMetricsTimeRange(metrics *GatewayMetrics, timestamp string) {
	if timestamp == "" {
		return
	}
	parsedTime, err := time.Parse(time.RFC3339Nano, timestamp)
	if err != nil {
		return
	}
	if metrics.StartTime.IsZero() || parsedTime.Before(metrics.StartTime) {
		metrics.StartTime = parsedTime
	}
	if metrics.EndTime.IsZero() || parsedTime.After(metrics.EndTime) {
		metrics.EndTime = parsedTime
	}
}

func handleFilteredRPCMessage(metrics *GatewayMetrics, entry RPCMessageEntry) {
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
}

func handleRPCRequestMessage(metrics *GatewayMetrics, pendingRequests map[string]*rpcPendingRequest, entry RPCMessageEntry) {
	var req rpcRequestPayload
	if err := json.Unmarshal(entry.Payload, &req); err != nil || req.Method != "tools/call" {
		return
	}

	var params rpcToolCallParams
	if err := json.Unmarshal(req.Params, &params); err != nil || params.Name == "" {
		return
	}

	metrics.TotalRequests++
	server := getOrCreateServer(metrics, entry.ServerID)
	server.RequestCount++
	metrics.TotalToolCalls++
	server.ToolCallCount++
	getOrCreateTool(server, params.Name).CallCount++

	if requestTime, ok := parseRPCTimestamp(entry.Timestamp); ok && req.ID != nil {
		pendingRequests[rpcPendingRequestKey(entry.ServerID, req.ID)] = &rpcPendingRequest{
			ServerID:  entry.ServerID,
			ToolName:  params.Name,
			Timestamp: requestTime,
		}
	}
}

func handleRPCResponseMessage(metrics *GatewayMetrics, pendingRequests map[string]*rpcPendingRequest, entry RPCMessageEntry) {
	var resp rpcResponsePayload
	if err := json.Unmarshal(entry.Payload, &resp); err != nil {
		return
	}

	handleRPCResponseError(metrics, pendingRequests, entry, resp)
	updateRPCResponseDuration(metrics, pendingRequests, entry, resp)
}

func handleRPCResponseError(metrics *GatewayMetrics, pendingRequests map[string]*rpcPendingRequest, entry RPCMessageEntry, resp rpcResponsePayload) {
	if resp.Error == nil {
		return
	}

	metrics.TotalErrors++
	server := getOrCreateServer(metrics, entry.ServerID)
	server.ErrorCount++
	if !isGuardPolicyErrorCode(resp.Error.Code) {
		return
	}

	metrics.TotalGuardBlocked++
	server.GuardPolicyBlocked++
	metrics.GuardPolicyEvents = append(metrics.GuardPolicyEvents, buildGuardPolicyEvent(entry, resp, pendingRequests))
}

func buildGuardPolicyEvent(entry RPCMessageEntry, resp rpcResponsePayload, pendingRequests map[string]*rpcPendingRequest) GuardPolicyEvent {
	toolName := ""
	if resp.ID != nil {
		if pending, ok := pendingRequests[rpcPendingRequestKey(entry.ServerID, resp.ID)]; ok {
			toolName = pending.ToolName
		}
	}

	reason := guardPolicyReasonFromCode(resp.Error.Code)
	if resp.Error.Data != nil && resp.Error.Data.Reason != "" {
		reason = resp.Error.Data.Reason
	}

	event := GuardPolicyEvent{
		Timestamp: entry.Timestamp,
		ServerID:  entry.ServerID,
		ToolName:  toolName,
		ErrorCode: resp.Error.Code,
		Reason:    reason,
		Message:   resp.Error.Message,
	}
	if resp.Error.Data != nil {
		event.Details = resp.Error.Data.Details
		event.Repository = resp.Error.Data.Repository
	}
	return event
}

func updateRPCResponseDuration(metrics *GatewayMetrics, pendingRequests map[string]*rpcPendingRequest, entry RPCMessageEntry, resp rpcResponsePayload) {
	if resp.ID == nil {
		return
	}
	responseTime, ok := parseRPCTimestamp(entry.Timestamp)
	if !ok {
		return
	}
	key := rpcPendingRequestKey(entry.ServerID, resp.ID)
	pending, ok := pendingRequests[key]
	if !ok {
		return
	}
	delete(pendingRequests, key)
	applyRPCPendingRequestDuration(metrics, entry.ServerID, pending, responseTime, resp.Error != nil)
}

func applyRPCPendingRequestDuration(metrics *GatewayMetrics, serverID string, pending *rpcPendingRequest, responseTime time.Time, hasError bool) {
	durationMS := float64(responseTime.Sub(pending.Timestamp).Milliseconds())
	if durationMS < 0 {
		return
	}

	server := getOrCreateServer(metrics, serverID)
	server.TotalDuration += durationMS
	metrics.TotalDuration += durationMS

	tool := getOrCreateTool(server, pending.ToolName)
	tool.TotalDuration += durationMS
	if tool.MaxDuration == 0 || durationMS > tool.MaxDuration {
		tool.MaxDuration = durationMS
	}
	if tool.MinDuration == 0 || durationMS < tool.MinDuration {
		tool.MinDuration = durationMS
	}
	if hasError {
		tool.ErrorCount++
	}
}

func parseRPCTimestamp(timestamp string) (time.Time, bool) {
	if timestamp == "" {
		return time.Time{}, false
	}
	parsedTime, err := time.Parse(time.RFC3339Nano, timestamp)
	if err != nil {
		return time.Time{}, false
	}
	return parsedTime, true
}

func rpcPendingRequestKey(serverID string, id any) string {
	return fmt.Sprintf("%s/%v", serverID, id)
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
	processedKeys := make(map[string]struct {
	})

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
			processedKeys[key] = struct {
			}{}

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
		if !setutil.Contains(processedKeys, key) {
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
