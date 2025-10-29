package workflow

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Pre-compiled regexes for performance (avoid recompiling in hot paths)
var (
	// Timestamp patterns for log cleanup
	timestampPattern1 = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}\s+\d{2}:\d{2}:\d{2}(\.\d+)?\s+`)
	timestampPattern2 = regexp.MustCompile(`^\[\d{2}:\d{2}:\d{2}\]\s+`)
	timestampPattern3 = regexp.MustCompile(`^\d{2}:\d{2}:\d{2}(\.\d+)?\s+`)

	// Log level pattern for message cleanup
	logLevelPattern = regexp.MustCompile(`(?i)^\[?(ERROR|WARNING|WARN|INFO|DEBUG)\]?\s*[:-]?\s*`)
)

// ToolCallInfo represents statistics for a single tool
type ToolCallInfo struct {
	Name          string        // Prettified tool name (e.g., "github::search_issues", "bash")
	CallCount     int           // Number of times this tool was called
	MaxInputSize  int           // Maximum input size in tokens for any call
	MaxOutputSize int           // Maximum output size in tokens for any call
	MaxDuration   time.Duration // Maximum execution duration for any call
}

// LogError represents a single error or warning from the log
type LogError struct {
	File      string // File path (usually the log file)
	Line      int    // Line number in the log file
	Type      string // "error" or "warning"
	Message   string // Error/warning message
	PatternID string // ID of the error pattern that matched (if available)
}

// CountErrors counts the number of errors in the slice
func CountErrors(errors []LogError) int {
	count := 0
	for _, err := range errors {
		if err.Type == "error" {
			count++
		}
	}
	return count
}

// CountWarnings counts the number of warnings in the slice
func CountWarnings(errors []LogError) int {
	count := 0
	for _, err := range errors {
		if err.Type == "warning" {
			count++
		}
	}
	return count
}

// LogMetrics represents extracted metrics from log files
type LogMetrics struct {
	TokenUsage    int
	EstimatedCost float64
	Errors        []LogError     // Individual error and warning details
	Turns         int            // Number of turns needed to complete the task
	ToolCalls     []ToolCallInfo // Tool call statistics
	ToolSequences [][]string     // Sequences of tool calls preserving order
	// Timestamp removed - use GitHub API timestamps instead of parsing from logs
}

// ExtractFirstMatch extracts the first regex match from a string
// Note: This function compiles the regex on each call. For frequently-used patterns,
// consider pre-compiling at package level or caching the compiled regex.
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
	var metrics LogMetrics

	// Trim the line first
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return metrics
	}

	// If the line isn't a clean JSON object, try to extract a JSON object substring
	jsonStr := trimmed
	if !(strings.HasPrefix(trimmed, "{") && strings.HasSuffix(trimmed, "}")) {
		// Find first '{' and last '}' and attempt to parse that slice
		open := strings.Index(trimmed, "{")
		close := strings.LastIndex(trimmed, "}")
		if open == -1 || close == -1 || close <= open {
			return metrics
		}
		jsonStr = trimmed[open : close+1]
	}

	// Try to parse as generic JSON
	var jsonData map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &jsonData); err != nil {
		// If parsing fails, try a relaxed approach: sometimes logs contain a JSON-like object with single quotes
		// Replace single quotes with double quotes as a last resort (not ideal, but helpful for noisy logs)
		relaxed := strings.ReplaceAll(jsonStr, "'", "\"")
		if err2 := json.Unmarshal([]byte(relaxed), &jsonData); err2 != nil {
			return metrics
		}
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
func ExtractJSONTokenUsage(data map[string]any) int {
	// Prefer explicit input+output sums at the top-level
	inputTop := ConvertToInt(data["input_tokens"])
	outputTop := ConvertToInt(data["output_tokens"])
	if inputTop > 0 || outputTop > 0 {
		return inputTop + outputTop
	}

	// Check top-level token fields that represent a single total value
	tokenFields := []string{"tokens", "token_count", "total_tokens"}
	for _, field := range tokenFields {
		if val, exists := data[field]; exists {
			if tokens := ConvertToInt(val); tokens > 0 {
				return tokens
			}
		}
	}

	// Check nested usage objects (Claude API format)
	if usage, exists := data["usage"]; exists {
		if usageMap, ok := usage.(map[string]any); ok {
			// Claude format: {"usage": {"input_tokens": 10, "output_tokens": 5, "cache_creation_input_tokens": 100, "cache_read_input_tokens": 200}}
			inputTokens := ConvertToInt(usageMap["input_tokens"])
			outputTokens := ConvertToInt(usageMap["output_tokens"])
			cacheCreationTokens := ConvertToInt(usageMap["cache_creation_input_tokens"])
			cacheReadTokens := ConvertToInt(usageMap["cache_read_input_tokens"])

			totalTokens := inputTokens + outputTokens + cacheCreationTokens + cacheReadTokens
			if totalTokens > 0 {
				return totalTokens
			}

			// Generic token count fields inside usage
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
		if deltaMap, ok := delta.(map[string]any); ok {
			if usage, exists := deltaMap["usage"]; exists {
				if usageMap, ok := usage.(map[string]any); ok {
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
func ExtractJSONCost(data map[string]any) float64 {
	// Common cost field names
	costFields := []string{"total_cost_usd", "cost", "price", "amount", "total_cost", "estimated_cost"}

	// Prefer explicit total_cost_usd at top-level
	if val, exists := data["total_cost_usd"]; exists {
		if cost := ConvertToFloat(val); cost > 0 {
			return cost
		}
	}

	for _, field := range costFields {
		if val, exists := data[field]; exists {
			if cost := ConvertToFloat(val); cost > 0 {
				return cost
			}
		}
	}

	// Check nested billing or pricing objects
	if billing, exists := data["billing"]; exists {
		if billingMap, ok := billing.(map[string]any); ok {
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

// ConvertToInt safely converts any to int
func ConvertToInt(val any) int {
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

// ConvertToFloat safely converts any to float64
func ConvertToFloat(val any) float64 {
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

// PrettifyToolName removes "mcp__" prefix and formats tool names nicely
func PrettifyToolName(toolName string) string {
	// Handle MCP tools: "mcp__github__search_issues" -> "github_search_issues"
	// Avoid colons and leave underscores as-is
	if strings.HasPrefix(toolName, "mcp__") {
		parts := strings.Split(toolName, "__")
		if len(parts) >= 3 {
			provider := parts[1]
			method := strings.Join(parts[2:], "_")
			return fmt.Sprintf("%s_%s", provider, method)
		}
		// If format is unexpected, just remove the mcp__ prefix
		return strings.TrimPrefix(toolName, "mcp__")
	}

	// Handle bash specially - keep as "bash"
	if strings.ToLower(toolName) == "bash" {
		return "bash"
	}

	// Return other tool names as-is
	return toolName
}

// ExtractMCPServer extracts the MCP server name from a tool name
func ExtractMCPServer(toolName string) string {
	// For MCP tools with pattern "mcp__server__method", extract "server"
	if strings.HasPrefix(toolName, "mcp__") {
		parts := strings.Split(toolName, "__")
		if len(parts) >= 2 {
			return parts[1]
		}
	}
	// For non-MCP tools, return the tool name as-is
	return toolName
}

// compiledPattern stores a pre-compiled regex with its metadata
type compiledPattern struct {
	regex        *regexp.Regexp
	id           string
	levelGroup   int
	messageGroup int
	severity     string
}

// CountErrorsAndWarningsWithPatterns extracts errors and warnings using regex patterns
// This is more accurate than simple string matching and uses the same logic as validate_errors.cjs
func CountErrorsAndWarningsWithPatterns(logContent string, patterns []ErrorPattern) []LogError {
	var errors []LogError

	if len(patterns) == 0 {
		return errors
	}

	// Pre-compile all patterns once before processing lines (performance optimization)
	compiledPatterns := make([]compiledPattern, 0, len(patterns))
	for _, pattern := range patterns {
		regex, err := regexp.Compile(pattern.Pattern)
		if err != nil {
			// Skip invalid patterns
			continue
		}
		compiledPatterns = append(compiledPatterns, compiledPattern{
			regex:        regex,
			id:           pattern.ID,
			levelGroup:   pattern.LevelGroup,
			messageGroup: pattern.MessageGroup,
			severity:     pattern.Severity,
		})
	}

	lines := strings.Split(logContent, "\n")

	for lineNum, line := range lines {
		for _, cp := range compiledPatterns {
			// Find first match only - for error detection we don't need all matches
			match := cp.regex.FindStringSubmatch(line)
			if match == nil {
				continue
			}

			level := extractLevelFromMatchCompiled(match, cp)

			// Extract message using the pattern's MessageGroup or full match
			message := ""
			if cp.messageGroup > 0 && cp.messageGroup < len(match) && match[cp.messageGroup] != "" {
				message = match[cp.messageGroup]
			} else if len(match) > 0 {
				message = match[0]
			}

			// Clean up the message
			message = extractErrorMessage(message)

			if strings.ToLower(level) == "error" {
				if message != "" {
					errors = append(errors, LogError{
						Line:      lineNum + 1, // 1-based line numbering
						Type:      "error",
						Message:   message,
						PatternID: cp.id,
					})
				}
			} else if strings.ToLower(level) == "warning" || strings.ToLower(level) == "warn" {
				if message != "" {
					errors = append(errors, LogError{
						Line:      lineNum + 1, // 1-based line numbering
						Type:      "warning",
						Message:   message,
						PatternID: cp.id,
					})
				}
			}
		}
	}

	return errors
}

// extractLevelFromMatchCompiled is the compiled-pattern version of extractLevelFromMatch
func extractLevelFromMatchCompiled(match []string, cp compiledPattern) string {
	// If Severity is explicitly set, use it
	if cp.severity != "" {
		return cp.severity
	}

	// If level group is specified and valid, use it
	if cp.levelGroup > 0 && cp.levelGroup < len(match) && match[cp.levelGroup] != "" {
		levelText := strings.ToLower(match[cp.levelGroup])
		// Normalize common error/warning keywords
		if strings.Contains(levelText, "err") || strings.Contains(levelText, "error") ||
			strings.Contains(levelText, "fail") || strings.Contains(levelText, "fatal") {
			return "error"
		} else if strings.Contains(levelText, "warn") || strings.Contains(levelText, "warning") {
			return "warning"
		}
		// Return the original level text if it doesn't match common patterns
		return match[cp.levelGroup]
	}

	// Try to infer level from the full match content
	if len(match) > 0 {
		fullMatch := strings.ToLower(match[0])

		// Check for specific Copilot CLI permission warnings (before general error checks)
		if strings.Contains(fullMatch, "permission denied and could not request permission from user") {
			return "warning"
		}

		if strings.Contains(fullMatch, "error") || strings.Contains(fullMatch, "err") ||
			strings.Contains(fullMatch, "fail") || strings.Contains(fullMatch, "fatal") {
			return "error"
		} else if strings.Contains(fullMatch, "warn") || strings.Contains(fullMatch, "warning") {
			return "warning"
		}
		// Additional error indicators
		if strings.Contains(fullMatch, "denied") || strings.Contains(fullMatch, "forbidden") ||
			strings.Contains(fullMatch, "unauthorized") || strings.Contains(fullMatch, "not found") ||
			strings.Contains(fullMatch, "✗") || // Copilot CLI failure indicator
			strings.Contains(fullMatch, "permission") && (strings.Contains(fullMatch, "denied") || strings.Contains(fullMatch, "restricted")) {
			return "error"
		}
	}

	return "unknown"
}

// extractErrorMessage extracts a clean error message from a log line
// Removes timestamps, log level prefixes, and other common noise
func extractErrorMessage(line string) string {
	// Remove common timestamp patterns using pre-compiled regexes
	cleanedLine := line
	cleanedLine = timestampPattern1.ReplaceAllString(cleanedLine, "")
	cleanedLine = timestampPattern2.ReplaceAllString(cleanedLine, "")
	cleanedLine = timestampPattern3.ReplaceAllString(cleanedLine, "")

	// Remove common log level prefixes using pre-compiled regex
	cleanedLine = logLevelPattern.ReplaceAllString(cleanedLine, "")

	// Trim whitespace
	cleanedLine = strings.TrimSpace(cleanedLine)

	// If the line is too long (>200 chars), truncate it
	if len(cleanedLine) > 200 {
		cleanedLine = cleanedLine[:197] + "..."
	}

	return cleanedLine
}
