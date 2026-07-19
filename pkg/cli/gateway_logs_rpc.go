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

		parseRPCMessagesTimeRange(&entry, metrics)

		if entry.ServerID == "" {
			continue
		}

		parseRPCMessagesEntry(&entry, metrics, pendingRequests)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading rpc-messages.jsonl: %w", err)
	}

	calculateGatewayAggregates(metrics)

	gatewayLogsLog.Printf("Successfully parsed rpc-messages.jsonl: %d servers, %d total requests",
		len(metrics.Servers), metrics.TotalRequests)

	return metrics, nil
}

func parseRPCMessagesTimeRange(entry *RPCMessageEntry, metrics *GatewayMetrics) {
	if entry.Timestamp == "" {
		return
	}
	if t, err := time.Parse(time.RFC3339Nano, entry.Timestamp); err == nil {
		if metrics.StartTime.IsZero() || t.Before(metrics.StartTime) {
			metrics.StartTime = t
		}
		if metrics.EndTime.IsZero() || t.After(metrics.EndTime) {
			metrics.EndTime = t
		}
	}
}

func parseRPCMessagesEntry(entry *RPCMessageEntry, metrics *GatewayMetrics, pendingRequests map[string]*rpcPendingRequest) {
	switch {
	case entry.Type == "DIFC_FILTERED":
		parseRPCMessagesFiltered(entry, metrics)
	case entry.Direction == "OUT" && entry.Type == "REQUEST":
		parseRPCMessagesRequest(entry, metrics, pendingRequests)
	case entry.Direction == "IN" && entry.Type == "RESPONSE":
		parseRPCMessagesResponse(entry, metrics, pendingRequests)
	}
}

func parseRPCMessagesFiltered(entry *RPCMessageEntry, metrics *GatewayMetrics) {
	metrics.TotalFiltered++
	server := getOrCreateServer(metrics, entry.ServerID)
	server.FilteredCount++
	metrics.FilteredEvents = append(metrics.FilteredEvents, DifcFilteredEvent{
		Timestamp: entry.Timestamp, ServerID: entry.ServerID, ToolName: entry.ToolName,
		Description: entry.Description, Reason: entry.Reason, SecrecyTags: entry.SecrecyTags,
		IntegrityTags: entry.IntegrityTags, AuthorAssociation: entry.AuthorAssociation,
		AuthorLogin: entry.AuthorLogin, HTMLURL: entry.HTMLURL, Number: entry.Number,
	})
}

func parseRPCMessagesRequest(entry *RPCMessageEntry, metrics *GatewayMetrics, pendingRequests map[string]*rpcPendingRequest) {
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
	tool := getOrCreateTool(server, params.Name)
	tool.CallCount++

	if req.ID != nil && entry.Timestamp != "" {
		if t, err := time.Parse(time.RFC3339Nano, entry.Timestamp); err == nil {
			key := fmt.Sprintf("%s/%v", entry.ServerID, req.ID)
			pendingRequests[key] = &rpcPendingRequest{ServerID: entry.ServerID, ToolName: params.Name, Timestamp: t}
		}
	}
}

func parseRPCMessagesResponse(entry *RPCMessageEntry, metrics *GatewayMetrics, pendingRequests map[string]*rpcPendingRequest) {
	var resp rpcResponsePayload
	if err := json.Unmarshal(entry.Payload, &resp); err != nil {
		return
	}
	if resp.Error != nil {
		parseRPCMessagesResponseError(entry, metrics, pendingRequests, &resp)
	}
	if resp.ID != nil && entry.Timestamp != "" {
		parseRPCMessagesResponseDuration(entry, metrics, pendingRequests, &resp)
	}
}

func parseRPCMessagesResponseError(entry *RPCMessageEntry, metrics *GatewayMetrics, pendingRequests map[string]*rpcPendingRequest, resp *rpcResponsePayload) {
	metrics.TotalErrors++
	server := getOrCreateServer(metrics, entry.ServerID)
	server.ErrorCount++
	if !isGuardPolicyErrorCode(resp.Error.Code) {
		return
	}
	metrics.TotalGuardBlocked++
	server.GuardPolicyBlocked++
	evt := parseRPCMessagesGuardPolicyEvent(entry, pendingRequests, resp)
	metrics.GuardPolicyEvents = append(metrics.GuardPolicyEvents, evt)
}

func parseRPCMessagesGuardPolicyEvent(entry *RPCMessageEntry, pendingRequests map[string]*rpcPendingRequest, resp *rpcResponsePayload) GuardPolicyEvent {
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
	evt := GuardPolicyEvent{Timestamp: entry.Timestamp, ServerID: entry.ServerID, ToolName: toolName, ErrorCode: resp.Error.Code, Reason: reason, Message: resp.Error.Message}
	if resp.Error.Data != nil {
		evt.Details = resp.Error.Data.Details
		evt.Repository = resp.Error.Data.Repository
	}
	return evt
}

func parseRPCMessagesResponseDuration(entry *RPCMessageEntry, metrics *GatewayMetrics, pendingRequests map[string]*rpcPendingRequest, resp *rpcResponsePayload) {
	key := fmt.Sprintf("%s/%v", entry.ServerID, resp.ID)
	pending, ok := pendingRequests[key]
	if !ok {
		return
	}
	delete(pendingRequests, key)
	if t, err := time.Parse(time.RFC3339Nano, entry.Timestamp); err == nil {
		durationMs := float64(t.Sub(pending.Timestamp).Milliseconds())
		if durationMs >= 0 {
			parseRPCMessagesApplyDuration(entry.ServerID, pending.ToolName, durationMs, resp.Error != nil, metrics)
		}
	}
}

func parseRPCMessagesApplyDuration(serverID, toolName string, durationMs float64, hadError bool, metrics *GatewayMetrics) {
	server := getOrCreateServer(metrics, serverID)
	server.TotalDuration += durationMs
	metrics.TotalDuration += durationMs
	tool := getOrCreateTool(server, toolName)
	tool.TotalDuration += durationMs
	tool.MaxDuration = max(tool.MaxDuration, durationMs)
	if tool.MinDuration == 0 || durationMs < tool.MinDuration {
		tool.MinDuration = durationMs
	}
	if hadError {
		tool.ErrorCount++
	}
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
type buildToolCallsFromRPCMessagesPendingCall struct {
	serverID  string
	toolName  string
	timestamp time.Time
}

type buildToolCallsFromRPCMessagesRawEntry struct {
	entry RPCMessageEntry
	req   rpcRequestPayload
	resp  rpcResponsePayload
	valid bool
}

func buildToolCallsFromRPCMessages(logPath string) ([]MCPToolCall, error) {
	file, err := os.Open(logPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open rpc-messages.jsonl: %w", err)
	}
	defer file.Close()

	pending := make(map[string]*buildToolCallsFromRPCMessagesPendingCall) // key: "<serverID>/<id>"
	entries, err := buildToolCallsFromRPCMessagesRead(file)
	if err != nil {
		return nil, err
	}

	// Second pass: build MCPToolCall records.
	// Declared before first pass so requests without IDs can be appended immediately.
	var toolCalls []MCPToolCall
	processedKeys := make(map[string]struct {
	})

	// First pass: index outgoing tool-call requests by (serverID, id)
	buildToolCallsFromRPCMessagesIndexRequests(entries, pending, &toolCalls)

	// Second pass: pair responses with pending requests to compute durations
	buildToolCallsFromRPCMessagesPairResponses(entries, pending, processedKeys, &toolCalls)

	// Emit any requests that never received a response
	buildToolCallsFromRPCMessagesUnknown(pending, processedKeys, &toolCalls)

	return toolCalls, nil
}

func buildToolCallsFromRPCMessagesRead(file *os.File) ([]buildToolCallsFromRPCMessagesRawEntry, error) {
	var entries []buildToolCallsFromRPCMessagesRawEntry
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
		entries = append(entries, buildToolCallsFromRPCMessagesRawEntry{entry: e, valid: true})
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading rpc-messages.jsonl: %w", err)
	}
	return entries, nil
}

func buildToolCallsFromRPCMessagesIndexRequests(entries []buildToolCallsFromRPCMessagesRawEntry, pending map[string]*buildToolCallsFromRPCMessagesPendingCall, toolCalls *[]MCPToolCall) {
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
			*toolCalls = append(*toolCalls, MCPToolCall{Timestamp: e.entry.Timestamp, ServerName: e.entry.ServerID, ToolName: params.Name, Status: "unknown"})
			continue
		}
		t, err := time.Parse(time.RFC3339Nano, e.entry.Timestamp)
		if err != nil {
			continue
		}
		key := fmt.Sprintf("%s/%v", e.entry.ServerID, e.req.ID)
		pending[key] = &buildToolCallsFromRPCMessagesPendingCall{serverID: e.entry.ServerID, toolName: params.Name, timestamp: t}
	}
}

func buildToolCallsFromRPCMessagesPairResponses(entries []buildToolCallsFromRPCMessagesRawEntry, pending map[string]*buildToolCallsFromRPCMessagesPendingCall, processedKeys map[string]struct{}, toolCalls *[]MCPToolCall) {
	for i := range entries {
		e := &entries[i]
		if e.entry.Direction != "IN" || e.entry.Type != "RESPONSE" {
			continue
		}
		if err := json.Unmarshal(e.entry.Payload, &e.resp); err != nil || e.resp.ID == nil {
			continue
		}
		key := fmt.Sprintf("%s/%v", e.entry.ServerID, e.resp.ID)
		p, ok := pending[key]
		if !ok {
			continue
		}
		processedKeys[key] = struct{}{}
		*toolCalls = append(*toolCalls, buildToolCallsFromRPCMessagesCall(e, p))
	}
}

func buildToolCallsFromRPCMessagesCall(e *buildToolCallsFromRPCMessagesRawEntry, p *buildToolCallsFromRPCMessagesPendingCall) MCPToolCall {
	call := MCPToolCall{Timestamp: p.timestamp.Format(time.RFC3339Nano), ServerName: p.serverID, ToolName: p.toolName, Status: "success"}
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
	return call
}

func buildToolCallsFromRPCMessagesUnknown(pending map[string]*buildToolCallsFromRPCMessagesPendingCall, processedKeys map[string]struct{}, toolCalls *[]MCPToolCall) {
	for key, p := range pending {
		if !setutil.Contains(processedKeys, key) {
			*toolCalls = append(*toolCalls, MCPToolCall{
				Timestamp: p.timestamp.Format(time.RFC3339Nano), ServerName: p.serverID, ToolName: p.toolName, Status: "unknown",
			})
		}
	}
}
