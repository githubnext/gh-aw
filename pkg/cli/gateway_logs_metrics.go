// This file provides metrics computation functions for MCP gateway logs.
//
// It processes individual log entries and aggregates statistics across servers and tools.
//
// Type definitions are in gateway_logs_types.go.
// Log file parsing is in gateway_logs_parser.go.
// Rendering functions are in gateway_logs_render.go.

package cli

import "time"

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
