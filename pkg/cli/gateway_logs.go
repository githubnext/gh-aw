// This file provides command-line interface functionality for gh-aw.
// This file (gateway_logs.go) contains functions for parsing and analyzing
// MCP gateway logs from gateway.jsonl or rpc-messages.jsonl files.
//
// Key responsibilities:
//   - Parsing gateway.jsonl JSONL format logs (preferred)
//   - Parsing rpc-messages.jsonl JSONL format logs (canonical fallback)
//   - Extracting server and tool usage metrics
//   - Aggregating gateway statistics
//   - Rendering gateway metrics tables

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
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/sliceutil"
	"github.com/github/gh-aw/pkg/timeutil"
)

var gatewayLogsLog = logger.New("cli:gateway_logs")

// maxScannerBufferSize is the maximum scanner buffer for large JSONL payloads (1 MB).
const maxScannerBufferSize = 1024 * 1024

// GatewayLogEntry represents a single log entry from gateway.jsonl
type GatewayLogEntry struct {
	Timestamp         string   `json:"timestamp"`
	Level             string   `json:"level"`
	Type              string   `json:"type"`
	Event             string   `json:"event"`
	ServerName        string   `json:"server_name,omitempty"`
	ServerID          string   `json:"server_id,omitempty"` // used by DIFC_FILTERED events
	ToolName          string   `json:"tool_name,omitempty"`
	Method            string   `json:"method,omitempty"`
	Duration          float64  `json:"duration,omitempty"` // in milliseconds
	InputSize         int      `json:"input_size,omitempty"`
	OutputSize        int      `json:"output_size,omitempty"`
	Status            string   `json:"status,omitempty"`
	Error             string   `json:"error,omitempty"`
	Message           string   `json:"message,omitempty"`
	Description       string   `json:"description,omitempty"`
	Reason            string   `json:"reason,omitempty"`
	SecrecyTags       []string `json:"secrecy_tags,omitempty"`
	IntegrityTags     []string `json:"integrity_tags,omitempty"`
	AuthorAssociation string   `json:"author_association,omitempty"`
	AuthorLogin       string   `json:"author_login,omitempty"`
	HTMLURL           string   `json:"html_url,omitempty"`
	Number            string   `json:"number,omitempty"`
}

// DifcFilteredEvent represents a DIFC_FILTERED log entry from gateway.jsonl.
// These events occur when a tool call is blocked by DIFC integrity or secrecy checks.
type DifcFilteredEvent struct {
	Timestamp         string   `json:"timestamp"`
	ServerID          string   `json:"server_id"`
	ToolName          string   `json:"tool_name"`
	Description       string   `json:"description,omitempty"`
	Reason            string   `json:"reason"`
	SecrecyTags       []string `json:"secrecy_tags,omitempty"`
	IntegrityTags     []string `json:"integrity_tags,omitempty"`
	AuthorAssociation string   `json:"author_association,omitempty"`
	AuthorLogin       string   `json:"author_login,omitempty"`
	HTMLURL           string   `json:"html_url,omitempty"`
	Number            string   `json:"number,omitempty"`
}

// GatewayServerMetrics represents usage metrics for a single MCP server
type GatewayServerMetrics struct {
	ServerName    string
	RequestCount  int
	ToolCallCount int
	TotalDuration float64 // in milliseconds
	ErrorCount    int
	FilteredCount int // number of DIFC_FILTERED events for this server
	Tools         map[string]*GatewayToolMetrics
}

// GatewayToolMetrics represents usage metrics for a specific tool
type GatewayToolMetrics struct {
	ToolName        string
	CallCount       int
	TotalDuration   float64 // in milliseconds
	AvgDuration     float64 // in milliseconds
	MaxDuration     float64 // in milliseconds
	MinDuration     float64 // in milliseconds
	ErrorCount      int
	TotalInputSize  int
	TotalOutputSize int
}

// GatewayMetrics represents aggregated metrics from gateway logs
type GatewayMetrics struct {
	TotalRequests  int
	TotalToolCalls int
	TotalErrors    int
	TotalFiltered  int // number of DIFC_FILTERED events
	Servers        map[string]*GatewayServerMetrics
	FilteredEvents []DifcFilteredEvent
	StartTime      time.Time
	EndTime        time.Time
	TotalDuration  float64 // in milliseconds
}

// RPCMessageEntry represents a single entry from rpc-messages.jsonl.
// This file is written by the Copilot CLI and contains raw JSON-RPC protocol messages
// exchanged between the AI engine and MCP servers, as well as DIFC_FILTERED events.
type RPCMessageEntry struct {
	Timestamp string          `json:"timestamp"`
	Direction string          `json:"direction"` // "IN" = received from server, "OUT" = sent to server; empty for DIFC_FILTERED
	Type      string          `json:"type"`      // "REQUEST", "RESPONSE", or "DIFC_FILTERED"
	ServerID  string          `json:"server_id"`
	Payload   json.RawMessage `json:"payload"`
	// Fields populated only for DIFC_FILTERED entries
	ToolName          string   `json:"tool_name,omitempty"`
	Description       string   `json:"description,omitempty"`
	Reason            string   `json:"reason,omitempty"`
	SecrecyTags       []string `json:"secrecy_tags,omitempty"`
	IntegrityTags     []string `json:"integrity_tags,omitempty"`
	AuthorAssociation string   `json:"author_association,omitempty"`
	AuthorLogin       string   `json:"author_login,omitempty"`
	HTMLURL           string   `json:"html_url,omitempty"`
	Number            string   `json:"number,omitempty"`
}

// rpcRequestPayload represents the JSON-RPC request payload fields we care about.
type rpcRequestPayload struct {
	Method string          `json:"method"`
	ID     any             `json:"id"`
	Params json.RawMessage `json:"params"`
}

// rpcToolCallParams represents the params for a tools/call request.
type rpcToolCallParams struct {
	Name string `json:"name"`
}

// rpcResponsePayload represents the JSON-RPC response payload fields we care about.
type rpcResponsePayload struct {
	ID    any       `json:"id"`
	Error *rpcError `json:"error,omitempty"`
}

// rpcError represents a JSON-RPC error object.
type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// rpcPendingRequest tracks an in-flight tool call for duration calculation.
type rpcPendingRequest struct {
	ServerID  string
	ToolName  string
	Timestamp time.Time
}

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

			// Track errors
			if resp.Error != nil {
				metrics.TotalErrors++
				server := getOrCreateServer(metrics, entry.ServerID)
				server.ErrorCount++
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
		serverRows := make([][]string, 0, len(serverNames))
		for _, serverName := range serverNames {
			server := metrics.Servers[serverName]
			avgTime := 0.0
			if server.RequestCount > 0 {
				avgTime = server.TotalDuration / float64(server.RequestCount)
			}
			if hasFiltered {
				serverRows = append(serverRows, []string{
					serverName,
					strconv.Itoa(server.RequestCount),
					strconv.Itoa(server.ToolCallCount),
					fmt.Sprintf("%.0fms", avgTime),
					strconv.Itoa(server.ErrorCount),
					strconv.Itoa(server.FilteredCount),
				})
			} else {
				serverRows = append(serverRows, []string{
					serverName,
					strconv.Itoa(server.RequestCount),
					strconv.Itoa(server.ToolCallCount),
					fmt.Sprintf("%.0fms", avgTime),
					strconv.Itoa(server.ErrorCount),
				})
			}
		}

		if hasFiltered {
			output.WriteString(console.RenderTable(console.TableConfig{
				Title:   "Server Usage",
				Headers: []string{"Server", "Requests", "Tool Calls", "Avg Time", "Errors", "Filtered"},
				Rows:    serverRows,
			}))
		} else {
			output.WriteString(console.RenderTable(console.TableConfig{
				Title:   "Server Usage",
				Headers: []string{"Server", "Requests", "Tool Calls", "Avg Time", "Errors"},
				Rows:    serverRows,
			}))
		}
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

	// Second pass: build MCPToolCall records
	var toolCalls []MCPToolCall
	processedKeys := make(map[string]bool)

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
		aggregated.TotalDuration += runMetrics.TotalDuration
		aggregated.FilteredEvents = append(aggregated.FilteredEvents, runMetrics.FilteredEvents...)

		// Merge server metrics
		for serverName, serverMetrics := range runMetrics.Servers {
			aggServer := getOrCreateServer(aggregated, serverName)
			aggServer.RequestCount += serverMetrics.RequestCount
			aggServer.ToolCallCount += serverMetrics.ToolCallCount
			aggServer.TotalDuration += serverMetrics.TotalDuration
			aggServer.ErrorCount += serverMetrics.ErrorCount
			aggServer.FilteredCount += serverMetrics.FilteredCount

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
