package workflow

import (
	"encoding/json"
	"strings"

	"github.com/githubnext/gh-aw/pkg/logger"
)

var copilotLogsLog = logger.New("workflow:copilot_logs")

// ParseLogMetrics implements engine-specific log parsing for Copilot CLI.
//
// Token Counting Behavior:
// Copilot CLI makes multiple API calls during a workflow run (one per turn).
// Each API call returns a response with usage statistics including token counts.
// This function accumulates token counts from ALL API responses to get the total
// token usage for the entire workflow run.
//
// Example: If a run has 3 turns with token counts [1000, 1500, 800],
// the total token usage will be 3300 (sum of all turns).
//
// This matches the behavior of the JavaScript parser in parse_copilot_log.cjs.
func (e *CopilotEngine) ParseLogMetrics(logContent string, verbose bool) LogMetrics {
	var metrics LogMetrics
	var totalTokenUsage int

	lines := strings.Split(logContent, "\n")
	toolCallMap := make(map[string]*ToolCallInfo) // Track tool calls
	var currentSequence []string                  // Track tool sequence
	turns := 0

	// Track multi-line JSON blocks for token extraction
	var inDataBlock bool
	var currentJSONLines []string

	for _, line := range lines {
		// Skip empty lines
		if strings.TrimSpace(line) == "" {
			continue
		}

		// Detect start of a JSON data block from Copilot debug logs
		// Format: "YYYY-MM-DDTHH:MM:SS.sssZ [DEBUG] data:"
		if strings.Contains(line, "[DEBUG] data:") {
			inDataBlock = true
			currentJSONLines = []string{}
			continue
		}

		// While in a data block, accumulate lines
		if inDataBlock {
			// Check if this line has a timestamp (indicates it's a log line, not raw JSON)
			hasTimestamp := strings.Contains(line, "[DEBUG]")

			if hasTimestamp {
				// Strip the timestamp and [DEBUG] prefix to see what remains
				// Format: "YYYY-MM-DDTHH:MM:SS.sssZ [DEBUG] {json content}"
				debugIndex := strings.Index(line, "[DEBUG]")
				if debugIndex != -1 {
					cleanLine := strings.TrimSpace(line[debugIndex+7:]) // Skip "[DEBUG]"

					// If after stripping, the line starts with JSON characters, it's part of JSON
					// Otherwise, it's a new log entry and we should end the block
					if strings.HasPrefix(cleanLine, "{") || strings.HasPrefix(cleanLine, "}") ||
						strings.HasPrefix(cleanLine, "[") || strings.HasPrefix(cleanLine, "]") ||
						strings.HasPrefix(cleanLine, "\"") {
						// This is JSON content - add it
						currentJSONLines = append(currentJSONLines, cleanLine)
					} else {
						// This is a new log line (not JSON content) - end of JSON block
						// Try to parse the accumulated JSON
						if len(currentJSONLines) > 0 {
							jsonStr := strings.Join(currentJSONLines, "\n")
							copilotLogsLog.Printf("Parsing JSON block with %d lines (%d bytes)", len(currentJSONLines), len(jsonStr))
							jsonMetrics := ExtractJSONMetrics(jsonStr, verbose)
							// Accumulate token usage from all responses (not just max)
							// This matches the JavaScript parser behavior in parse_copilot_log.cjs
							if jsonMetrics.TokenUsage > 0 {
								copilotLogsLog.Printf("Extracted %d tokens from JSON block", jsonMetrics.TokenUsage)
								totalTokenUsage += jsonMetrics.TokenUsage
							} else {
								copilotLogsLog.Printf("No tokens extracted from JSON block (possible format issue)")
							}
							if jsonMetrics.EstimatedCost > 0 {
								metrics.EstimatedCost += jsonMetrics.EstimatedCost
							}

							// Extract tool call sizes from the JSON response
							e.extractToolCallSizes(jsonStr, toolCallMap, verbose)
						}

						inDataBlock = false
						currentJSONLines = []string{}
					}
				}
			} else {
				// Line has no timestamp - it's raw JSON, add it
				currentJSONLines = append(currentJSONLines, line)
			}
		}

		// Count turns based on interaction patterns (adjust based on actual Copilot CLI output)
		if strings.Contains(line, "User:") || strings.Contains(line, "Human:") || strings.Contains(line, "Query:") {
			turns++
			// Start of a new turn, save previous sequence if any
			if len(currentSequence) > 0 {
				metrics.ToolSequences = append(metrics.ToolSequences, currentSequence)
				currentSequence = []string{}
			}
		}

		// Extract tool calls and add to sequence (adjust based on actual Copilot CLI output format)
		if toolName := e.parseCopilotToolCallsWithSequence(line, toolCallMap); toolName != "" {
			currentSequence = append(currentSequence, toolName)
		}
	}

	// Process any remaining JSON block at the end of file
	if inDataBlock && len(currentJSONLines) > 0 {
		jsonStr := strings.Join(currentJSONLines, "\n")
		copilotLogsLog.Printf("Parsing final JSON block at EOF with %d lines (%d bytes)", len(currentJSONLines), len(jsonStr))
		jsonMetrics := ExtractJSONMetrics(jsonStr, verbose)
		// Accumulate token usage from all responses (not just max)
		if jsonMetrics.TokenUsage > 0 {
			copilotLogsLog.Printf("Extracted %d tokens from final JSON block", jsonMetrics.TokenUsage)
			totalTokenUsage += jsonMetrics.TokenUsage
		} else {
			copilotLogsLog.Printf("No tokens extracted from final JSON block (possible format issue)")
		}
		if jsonMetrics.EstimatedCost > 0 {
			metrics.EstimatedCost += jsonMetrics.EstimatedCost
		}

		// Extract tool call sizes from the JSON response
		e.extractToolCallSizes(jsonStr, toolCallMap, verbose)
	}

	// Finalize metrics using shared helper
	copilotLogsLog.Printf("Finalized metrics: totalTokenUsage=%d, turns=%d, toolCalls=%d", totalTokenUsage, turns, len(toolCallMap))
	FinalizeToolMetrics(&metrics, toolCallMap, currentSequence, turns, totalTokenUsage, logContent, e.GetErrorPatterns())

	return metrics
}

// extractToolCallSizes extracts tool call input and output sizes from Copilot JSON responses
func (e *CopilotEngine) extractToolCallSizes(jsonStr string, toolCallMap map[string]*ToolCallInfo, verbose bool) {
	// Try to parse the JSON string
	var data map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		if verbose {
			copilotLogsLog.Printf("Failed to parse JSON for tool size extraction: %v", err)
		}
		return
	}

	// Look for tool_calls in the choices array (Copilot/OpenAI format)
	if choices, ok := data["choices"].([]any); ok {
		for _, choice := range choices {
			if choiceMap, ok := choice.(map[string]any); ok {
				if message, ok := choiceMap["message"].(map[string]any); ok {
					if toolCalls, ok := message["tool_calls"].([]any); ok {
						e.processToolCalls(toolCalls, toolCallMap, verbose)
					}
				}
			}
		}
	}

	// Also check for tool_calls directly in the message (alternative format)
	if message, ok := data["message"].(map[string]any); ok {
		if toolCalls, ok := message["tool_calls"].([]any); ok {
			e.processToolCalls(toolCalls, toolCallMap, verbose)
		}
	}
}

// processToolCalls processes tool_calls array and updates tool call map with sizes
func (e *CopilotEngine) processToolCalls(toolCalls []any, toolCallMap map[string]*ToolCallInfo, verbose bool) {
	for _, toolCall := range toolCalls {
		if tcMap, ok := toolCall.(map[string]any); ok {
			// Extract function information
			if function, ok := tcMap["function"].(map[string]any); ok {
				if toolName, ok := function["name"].(string); ok {
					// Calculate input size from arguments (if present)
					inputSize := 0
					if arguments, ok := function["arguments"].(string); ok {
						inputSize = len(arguments)
					}

					// Initialize or update tool call info
					if toolInfo, exists := toolCallMap[toolName]; exists {
						toolInfo.CallCount++
						// Update max input size if this call is larger
						if inputSize > toolInfo.MaxInputSize {
							toolInfo.MaxInputSize = inputSize
							if verbose {
								copilotLogsLog.Printf("Updated %s MaxInputSize to %d bytes", toolName, inputSize)
							}
						}
					} else {
						toolCallMap[toolName] = &ToolCallInfo{
							Name:          toolName,
							CallCount:     1,
							MaxInputSize:  inputSize,
							MaxOutputSize: 0, // Output size extraction not yet available in Copilot logs
						}
						if verbose {
							copilotLogsLog.Printf("Created tool info for %s with MaxInputSize=%d bytes", toolName, inputSize)
						}
					}
				}
			}
		}
	}
}

// parseCopilotToolCallsWithSequence extracts tool call information from Copilot CLI log lines and returns tool name
func (e *CopilotEngine) parseCopilotToolCallsWithSequence(line string, toolCallMap map[string]*ToolCallInfo) string {
	// This method handles simple tool execution log lines for sequence tracking
	// Tool size extraction is now handled by extractToolCallSizes which parses JSON

	// Look for "Executing tool:" pattern in Copilot logs
	if strings.Contains(line, "Executing tool:") {
		// Extract tool name from "Executing tool: <name>" format
		parts := strings.Split(line, "Executing tool:")
		if len(parts) > 1 {
			toolName := strings.TrimSpace(parts[1])
			// Return the tool name for sequence tracking
			// Size information is handled separately by extractToolCallSizes
			return toolName
		}
	}

	return ""
}

// GetLogParserScriptId returns the JavaScript script name for parsing Copilot logs
func (e *CopilotEngine) GetLogParserScriptId() string {
	return "parse_copilot_log"
}

// GetLogFileForParsing returns the log directory for Copilot CLI logs
// Copilot writes detailed debug logs to /tmp/gh-aw/sandbox/agent/logs/
func (e *CopilotEngine) GetLogFileForParsing() string {
	return "/tmp/gh-aw/sandbox/agent/logs/"
}

// GetFirewallLogsCollectionStep returns empty steps as firewall logs are at a known location
func (e *CopilotEngine) GetFirewallLogsCollectionStep(workflowData *WorkflowData) []GitHubActionStep {
	// Collection step removed - firewall logs are now at a known location
	return []GitHubActionStep{}
}

// GetSquidLogsSteps returns the steps for uploading and parsing Squid logs (after secret redaction)
func (e *CopilotEngine) GetSquidLogsSteps(workflowData *WorkflowData) []GitHubActionStep {
	var steps []GitHubActionStep

	// Only add upload and parsing steps if firewall is enabled
	if isFirewallEnabled(workflowData) {
		copilotLogsLog.Printf("Adding Squid logs upload and parsing steps for workflow: %s", workflowData.Name)

		squidLogsUpload := generateSquidLogsUploadStep(workflowData.Name)
		steps = append(steps, squidLogsUpload)

		// Add firewall log parsing step to create step summary
		firewallLogParsing := generateFirewallLogParsingStep(workflowData.Name)
		steps = append(steps, firewallLogParsing)
	} else {
		copilotLogsLog.Print("Firewall disabled, skipping Squid logs upload")
	}

	return steps
}

// GetCleanupStep returns the post-execution cleanup step (currently empty)
func (e *CopilotEngine) GetCleanupStep(workflowData *WorkflowData) GitHubActionStep {
	// Return empty step - cleanup steps have been removed
	return GitHubActionStep([]string{})
}
