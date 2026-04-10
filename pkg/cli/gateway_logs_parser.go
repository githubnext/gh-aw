// This file provides log parsing functions for MCP gateway log files.
//
// It handles parsing of:
//   - gateway.jsonl (preferred format, written by MCP Gateway)
//   - rpc-messages.jsonl (canonical fallback, written by Copilot CLI)
//
// Type definitions used here are in gateway_logs_types.go.
// Metrics computation helpers are in gateway_logs_metrics.go.
// Rendering functions are in gateway_logs_render.go.

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
	"github.com/github/gh-aw/pkg/logger"
)

var gatewayLogsLog = logger.New("cli:gateway_logs")

// maxScannerBufferSize is the maximum scanner buffer for large JSONL payloads (1 MB).
const maxScannerBufferSize = 1024 * 1024

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
