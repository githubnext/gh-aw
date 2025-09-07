package workflow

import (
	"encoding/json"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// ToolInvocationStats represents statistics for a specific tool
type ToolInvocationStats struct {
	Name            string        // Tool name (e.g., "Bash", "mcp__github__search_issues")
	Count           int           // Number of times this tool was invoked
	TotalOutputSize int64         // Total size of all outputs from this tool (in bytes)
	TotalDuration   time.Duration // Total duration of all invocations
	MaxDuration     time.Duration // Maximum duration of any single invocation
	SuccessCount    int           // Number of successful invocations
	ErrorCount      int           // Number of failed invocations
	BashCommands    []string      // For Bash tools, store the actual commands executed
}

// GetAverageDuration returns the average duration per invocation
func (t *ToolInvocationStats) GetAverageDuration() time.Duration {
	if t.Count == 0 {
		return 0
	}
	return t.TotalDuration / time.Duration(t.Count)
}

// GetAverageOutputSize returns the average output size per invocation
func (t *ToolInvocationStats) GetAverageOutputSize() int64 {
	if t.Count == 0 {
		return 0
	}
	return t.TotalOutputSize / int64(t.Count)
}

// GetSuccessRate returns the success rate as a percentage (0-100)
func (t *ToolInvocationStats) GetSuccessRate() float64 {
	if t.Count == 0 {
		return 0
	}
	return float64(t.SuccessCount) / float64(t.Count) * 100
}

// LogMetrics represents extracted metrics from log files
type LogMetrics struct {
	TokenUsage      int                             // Total token usage
	EstimatedCost   float64                         // Total estimated cost
	ErrorCount      int                             // Total errors in logs
	WarningCount    int                             // Total warnings in logs
	ToolInvocations map[string]*ToolInvocationStats // Statistics per tool
	Turns         int // Number of turns needed to complete the task
}

// NewLogMetrics creates a new LogMetrics instance with initialized tool invocations map
func NewLogMetrics() LogMetrics {
	return LogMetrics{
		ToolInvocations: make(map[string]*ToolInvocationStats),
	}
}

// AddToolInvocation adds or updates tool invocation statistics
func (m *LogMetrics) AddToolInvocation(name string, outputSize int64, duration time.Duration, success bool) {
	m.AddToolInvocationWithCommand(name, outputSize, duration, success, "")
}

// AddToolInvocationWithCommand adds or updates tool invocation statistics with command details for Bash tools
func (m *LogMetrics) AddToolInvocationWithCommand(name string, outputSize int64, duration time.Duration, success bool, command string) {
	if m.ToolInvocations == nil {
		m.ToolInvocations = make(map[string]*ToolInvocationStats)
	}

	stats, exists := m.ToolInvocations[name]
	if !exists {
		stats = &ToolInvocationStats{
			Name:         name,
			BashCommands: []string{},
		}
		m.ToolInvocations[name] = stats
	}

	stats.Count++
	stats.TotalOutputSize += outputSize
	stats.TotalDuration += duration

	// Update max duration if this duration is greater
	if duration > stats.MaxDuration {
		stats.MaxDuration = duration
	}

	if success {
		stats.SuccessCount++
	} else {
		stats.ErrorCount++
	}

	// Store command for Bash tools
	if name == "Bash" && command != "" {
		stats.BashCommands = append(stats.BashCommands, command)
	}
}

// MergeToolInvocations merges tool invocation statistics from another LogMetrics
func (m *LogMetrics) MergeToolInvocations(other LogMetrics) {
	if m.ToolInvocations == nil {
		m.ToolInvocations = make(map[string]*ToolInvocationStats)
	}

	for name, otherStats := range other.ToolInvocations {
		stats, exists := m.ToolInvocations[name]
		if !exists {
			// Copy the stats including BashCommands
			stats = &ToolInvocationStats{
				Name:         name,
				BashCommands: make([]string, len(otherStats.BashCommands)),
			}
			copy(stats.BashCommands, otherStats.BashCommands)
			m.ToolInvocations[name] = stats
		} else {
			// Merge BashCommands
			stats.BashCommands = append(stats.BashCommands, otherStats.BashCommands...)
		}

		stats.Count += otherStats.Count
		stats.TotalOutputSize += otherStats.TotalOutputSize
		stats.TotalDuration += otherStats.TotalDuration
		stats.SuccessCount += otherStats.SuccessCount
		stats.ErrorCount += otherStats.ErrorCount

		// Update max duration if other's max is greater
		if otherStats.MaxDuration > stats.MaxDuration {
			stats.MaxDuration = otherStats.MaxDuration
		}
	}
}

// ExtractFirstMatch extracts the first regex match from a string
func ExtractFirstMatch(text, pattern string) string {
	re := regexp.MustCompile(`(?i)` + pattern)
	matches := re.FindStringSubmatch(text)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

// ExtractJSONMetrics extracts metrics from streaming JSON log lines
func ExtractJSONMetrics(line string, verbose bool) LogMetrics {
	metrics := NewLogMetrics()

	// Skip lines that don't look like JSON
	trimmed := strings.TrimSpace(line)
	if !strings.HasPrefix(trimmed, "{") || !strings.HasSuffix(trimmed, "}") {
		return metrics
	}

	// Try to parse as generic JSON
	var jsonData map[string]interface{}
	if err := json.Unmarshal([]byte(trimmed), &jsonData); err != nil {
		return metrics
	}

	// Extract token usage from various possible fields and structures
	if tokens := ExtractJSONTokenUsage(jsonData); tokens > 0 {
		metrics.TokenUsage = tokens
	}

	// Extract cost information from various possible fields
	if cost := ExtractJSONCost(jsonData); cost > 0 {
		metrics.EstimatedCost = cost
	}

	return metrics
}

// ExtractJSONTokenUsage extracts token usage from JSON data
func ExtractJSONTokenUsage(data map[string]interface{}) int {
	// Check top-level token fields
	tokenFields := []string{"tokens", "token_count", "input_tokens", "output_tokens", "total_tokens"}
	for _, field := range tokenFields {
		if val, exists := data[field]; exists {
			if tokens := ConvertToInt(val); tokens > 0 {
				return tokens
			}
		}
	}

	// Check nested usage objects (Claude API format)
	if usage, exists := data["usage"]; exists {
		if usageMap, ok := usage.(map[string]interface{}); ok {
			// Claude format: {"usage": {"input_tokens": 10, "output_tokens": 5, "cache_creation_input_tokens": 100, "cache_read_input_tokens": 200}}
			inputTokens := ConvertToInt(usageMap["input_tokens"])
			outputTokens := ConvertToInt(usageMap["output_tokens"])
			cacheCreationTokens := ConvertToInt(usageMap["cache_creation_input_tokens"])
			cacheReadTokens := ConvertToInt(usageMap["cache_read_input_tokens"])

			totalTokens := inputTokens + outputTokens + cacheCreationTokens + cacheReadTokens
			if totalTokens > 0 {
				return totalTokens
			}

			// Generic token count in usage
			for _, field := range tokenFields {
				if val, exists := usageMap[field]; exists {
					if tokens := ConvertToInt(val); tokens > 0 {
						return tokens
					}
				}
			}
		}
	}

	// Check for delta structures (streaming format)
	if delta, exists := data["delta"]; exists {
		if deltaMap, ok := delta.(map[string]interface{}); ok {
			if usage, exists := deltaMap["usage"]; exists {
				if usageMap, ok := usage.(map[string]interface{}); ok {
					inputTokens := ConvertToInt(usageMap["input_tokens"])
					outputTokens := ConvertToInt(usageMap["output_tokens"])
					if inputTokens > 0 || outputTokens > 0 {
						return inputTokens + outputTokens
					}
				}
			}
		}
	}

	return 0
}

// ExtractJSONCost extracts cost information from JSON data
func ExtractJSONCost(data map[string]interface{}) float64 {
	// Common cost field names
	costFields := []string{"cost", "price", "amount", "total_cost", "estimated_cost", "total_cost_usd"}

	for _, field := range costFields {
		if val, exists := data[field]; exists {
			if cost := ConvertToFloat(val); cost > 0 {
				return cost
			}
		}
	}

	// Check nested billing or pricing objects
	if billing, exists := data["billing"]; exists {
		if billingMap, ok := billing.(map[string]interface{}); ok {
			for _, field := range costFields {
				if val, exists := billingMap[field]; exists {
					if cost := ConvertToFloat(val); cost > 0 {
						return cost
					}
				}
			}
		}
	}

	return 0
}

// ConvertToInt safely converts interface{} to int
func ConvertToInt(val interface{}) int {
	switch v := val.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	case string:
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return 0
}

// ConvertToFloat safely converts interface{} to float64
func ConvertToFloat(val interface{}) float64 {
	switch v := val.(type) {
	case float64:
		return v
	case int:
		return float64(v)
	case int64:
		return float64(v)
	case string:
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return 0
}
