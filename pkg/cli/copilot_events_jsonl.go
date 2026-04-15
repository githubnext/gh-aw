// This file provides command-line interface functionality for gh-aw.
// This file (copilot_events_jsonl.go) contains functions for finding and
// parsing Copilot CLI events.jsonl files from session-state artifacts.
//
// Key responsibilities:
//   - Locating events.jsonl in the copilot-session-state artifact directory
//   - Parsing the structured event log to extract tool calls, turns, and usage
//   - Providing precise, structured metrics as the primary data source for
//     Copilot CLI log analysis (before falling back to debug log parsing)

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
	"github.com/github/gh-aw/pkg/workflow"
)

var copilotEventsJSONLLog = logger.New("cli:copilot_events_jsonl")

// copilotEventsJSONLEntry represents a single event in a Copilot events.jsonl file.
// All events share the same envelope: type, id, timestamp, and a type-specific data object.
//
// The events.jsonl file is written by the Copilot CLI to:
//
//	~/.copilot/session-state/<session-uuid>/events.jsonl
//
// After artifact upload/download it is located at:
//
//	<logDir>/sandbox/agent/logs/copilot-session-state/<uuid>/events.jsonl
type copilotEventsJSONLEntry struct {
	Type      string                      `json:"type"`
	ID        string                      `json:"id"`
	Timestamp string                      `json:"timestamp"`
	ParentID  string                      `json:"parentId,omitempty"`
	Data      copilotEventsJSONLEntryData `json:"data"`
}

// copilotEventsJSONLEntryData holds the type-specific payload for each event.
// Fields are populated only for the relevant event types.
type copilotEventsJSONLEntryData struct {
	// session.start fields
	SessionID      string `json:"sessionId,omitempty"`
	CopilotVersion string `json:"copilotVersion,omitempty"`

	// session.model_change fields
	NewModel string `json:"newModel,omitempty"`

	// tool.execution_start fields
	ToolCallID    string `json:"toolCallId,omitempty"`
	ToolName      string `json:"toolName,omitempty"`
	MCPServerName string `json:"mcpServerName,omitempty"`
	MCPToolName   string `json:"mcpToolName,omitempty"`

	// tool.execution_complete fields
	Success bool   `json:"success"`
	Model   string `json:"model,omitempty"`

	// session.shutdown fields
	ShutdownType         string                          `json:"shutdownType,omitempty"`
	TotalPremiumRequests int                             `json:"totalPremiumRequests,omitempty"`
	ModelMetrics         map[string]*copilotModelMetrics `json:"modelMetrics,omitempty"`
}

// copilotModelMetrics holds per-model usage statistics from the session.shutdown event.
type copilotModelMetrics struct {
	Requests *copilotRequestMetrics `json:"requests,omitempty"`
	Usage    *copilotUsageMetrics   `json:"usage,omitempty"`
}

// copilotRequestMetrics holds request count and cost for a model.
type copilotRequestMetrics struct {
	Count int `json:"count"`
	Cost  int `json:"cost"`
}

// copilotUsageMetrics holds token usage for a model.
type copilotUsageMetrics struct {
	InputTokens      int `json:"inputTokens"`
	OutputTokens     int `json:"outputTokens"`
	CacheReadTokens  int `json:"cacheReadTokens"`
	CacheWriteTokens int `json:"cacheWriteTokens"`
}

// findEventsJSONLFile searches for an events.jsonl file in logDir.
// It first checks the canonical location at
// sandbox/agent/logs/copilot-session-state/<uuid>/events.jsonl
// and then falls back to a full recursive walk of logDir.
// Returns the first path found, or an empty string if not found.
func findEventsJSONLFile(logDir string) string {
	copilotEventsJSONLLog.Printf("Searching for events.jsonl in: %s", logDir)

	// Try the canonical location first (avoids a full directory walk in the common case)
	sessionStateDir := filepath.Join(logDir, "sandbox", "agent", "logs", "copilot-session-state")
	if canonicalPath := findFileInDir(sessionStateDir, "events.jsonl"); canonicalPath != "" {
		copilotEventsJSONLLog.Printf("Found events.jsonl at canonical location: %s", canonicalPath)
		return canonicalPath
	}

	// Fall back to a recursive search of the full log directory
	var foundPath string
	_ = filepath.Walk(logDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info == nil {
			return nil
		}
		if !info.IsDir() && info.Name() == "events.jsonl" && foundPath == "" {
			foundPath = path
			return errors.New("stop") // sentinel to stop walking
		}
		return nil
	})

	if foundPath != "" {
		copilotEventsJSONLLog.Printf("Found events.jsonl via recursive search: %s", foundPath)
	} else {
		copilotEventsJSONLLog.Printf("events.jsonl not found in: %s", logDir)
	}
	return foundPath
}

// findFileInDir searches for a file by name within dir (recursively).
// Returns the first matching path, or empty string if not found.
func findFileInDir(dir, name string) string {
	var found string
	_ = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info == nil {
			return nil
		}
		if !info.IsDir() && info.Name() == name && found == "" {
			found = path
			return errors.New("stop")
		}
		return nil
	})
	return found
}

// parseEventsJSONLFile parses a Copilot events.jsonl file and extracts log metrics.
//
// events.jsonl provides precise, structured data about a Copilot CLI session:
//   - "session.start":          session metadata (sessionId, copilotVersion)
//   - "user.message":           one per conversation turn (used to count turns)
//   - "tool.execution_start":   a tool invocation (data.toolName)
//   - "tool.execution_complete": completion of a tool call
//   - "session.shutdown":       session summary (totalPremiumRequests, modelMetrics)
//
// Returns the extracted metrics and nil on success, or empty metrics and an
// error if the file cannot be read or contains no recognizable events.
func parseEventsJSONLFile(path string, verbose bool) (workflow.LogMetrics, error) {
	copilotEventsJSONLLog.Printf("Parsing events.jsonl from: %s", path)

	var metrics workflow.LogMetrics

	// Sanitize path to prevent traversal
	cleanPath := filepath.Clean(path)

	file, err := os.Open(cleanPath)
	if err != nil {
		return metrics, fmt.Errorf("failed to open events.jsonl: %w", err)
	}
	defer file.Close()

	toolCallMap := make(map[string]*workflow.ToolCallInfo)
	var currentSequence []string
	turns := 0
	totalTokens := 0
	foundAnyEvent := false

	// Per-turn timestamps used to compute Time Between Turns (TBT)
	var turnTimestamps []time.Time

	scanner := bufio.NewScanner(file)
	buf := make([]byte, maxScannerBufferSize)
	scanner.Buffer(buf, maxScannerBufferSize)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || !strings.HasPrefix(line, "{") {
			continue
		}

		var entry copilotEventsJSONLEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			copilotEventsJSONLLog.Printf("Skipping malformed events.jsonl line: %v", err)
			continue
		}

		foundAnyEvent = true

		switch entry.Type {
		case "session.start":
			copilotEventsJSONLLog.Printf("session.start: sessionId=%s copilotVersion=%s",
				entry.Data.SessionID, entry.Data.CopilotVersion)

		case "user.message":
			// Each user message represents one conversation turn.
			// Save the current tool sequence before starting a new turn.
			turns++
			if len(currentSequence) > 0 {
				metrics.ToolSequences = append(metrics.ToolSequences, currentSequence)
				currentSequence = []string{}
			}
			// Record the timestamp for TBT computation.
			if entry.Timestamp != "" {
				if ts, parseErr := time.Parse(time.RFC3339Nano, entry.Timestamp); parseErr == nil {
					turnTimestamps = append(turnTimestamps, ts)
				} else if ts, parseErr = time.Parse(time.RFC3339, entry.Timestamp); parseErr == nil {
					turnTimestamps = append(turnTimestamps, ts)
				}
			}
			copilotEventsJSONLLog.Printf("user.message: turn=%d", turns)

		case "tool.execution_start":
			// Record the tool invocation and add to the current turn's sequence.
			toolName := entry.Data.ToolName
			if toolName != "" {
				currentSequence = append(currentSequence, toolName)
				if toolInfo, exists := toolCallMap[toolName]; exists {
					toolInfo.CallCount++
				} else {
					toolCallMap[toolName] = &workflow.ToolCallInfo{
						Name:      toolName,
						CallCount: 1,
					}
				}
				copilotEventsJSONLLog.Printf("tool.execution_start: %s", toolName)
			}

		case "session.shutdown":
			// Aggregate token usage across all models from modelMetrics.
			// modelMetrics provides the most accurate token counts; fall back to
			// totalPremiumRequests only when modelMetrics is absent.
			for model, m := range entry.Data.ModelMetrics {
				if m.Usage != nil {
					modelTokens := m.Usage.InputTokens + m.Usage.OutputTokens
					totalTokens += modelTokens
					copilotEventsJSONLLog.Printf("session.shutdown: model=%s inputTokens=%d outputTokens=%d",
						model, m.Usage.InputTokens, m.Usage.OutputTokens)
				}
			}
			if totalTokens == 0 && entry.Data.TotalPremiumRequests > 0 {
				// Fallback: use premium requests as a proxy when token counts are unavailable
				totalTokens = entry.Data.TotalPremiumRequests
				copilotEventsJSONLLog.Printf("session.shutdown: using totalPremiumRequests=%d as token proxy",
					entry.Data.TotalPremiumRequests)
			}
			copilotEventsJSONLLog.Printf("session.shutdown: type=%s totalPremiumRequests=%d totalTokens=%d",
				entry.Data.ShutdownType, entry.Data.TotalPremiumRequests, totalTokens)
		}
	}

	if scanErr := scanner.Err(); scanErr != nil {
		return metrics, fmt.Errorf("error reading events.jsonl: %w", scanErr)
	}

	if !foundAnyEvent {
		return metrics, errors.New("no events found in events.jsonl")
	}

	// Flush any remaining tool sequence
	if len(currentSequence) > 0 {
		metrics.ToolSequences = append(metrics.ToolSequences, currentSequence)
	}

	// Convert tool call map to slice
	for _, toolInfo := range toolCallMap {
		metrics.ToolCalls = append(metrics.ToolCalls, *toolInfo)
	}

	metrics.TokenUsage = totalTokens
	metrics.Turns = turns

	// Compute Time Between Turns (TBT) from per-turn timestamps.
	// TBT[i] = timestamp[i] - timestamp[i-1] for i > 0. Two or more timestamps
	// are required to measure at least one interval. Only positive intervals are
	// included so that identical or out-of-order timestamps don't skew the average.
	if len(turnTimestamps) >= 2 {
		var totalTBT time.Duration
		var maxTBT time.Duration
		validIntervals := 0
		for i := 1; i < len(turnTimestamps); i++ {
			tbt := turnTimestamps[i].Sub(turnTimestamps[i-1])
			if tbt > 0 {
				totalTBT += tbt
				validIntervals++
				if tbt > maxTBT {
					maxTBT = tbt
				}
			}
		}
		if validIntervals > 0 {
			metrics.AvgTimeBetweenTurns = totalTBT / time.Duration(validIntervals)
			metrics.MaxTimeBetweenTurns = maxTBT
			copilotEventsJSONLLog.Printf("TBT computed: avg=%s max=%s intervals=%d",
				metrics.AvgTimeBetweenTurns, metrics.MaxTimeBetweenTurns, validIntervals)
		}
	}

	copilotEventsJSONLLog.Printf("Parsed events.jsonl: turns=%d totalTokens=%d toolCalls=%d sequences=%d",
		turns, totalTokens, len(toolCallMap), len(metrics.ToolSequences))

	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(
			fmt.Sprintf("Parsed events.jsonl: %d turns, %d tokens, %d tool calls",
				turns, totalTokens, len(toolCallMap))))
	}

	return metrics, nil
}
