package cli

import (
	"bufio"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

// parseTokenUsageFile parses a token-usage.jsonl file and returns the aggregated summary.
func parseTokenUsageFile(filePath string) (*TokenUsageSummary, error) {
	tokenUsageLog.Printf("Parsing token usage file: %s", filePath)

	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open token usage file: %w", err)
	}
	defer file.Close()

	summary := &TokenUsageSummary{
		ByModel: make(map[string]*ModelTokenUsage),
	}

	scanner := bufio.NewScanner(file)
	// Increase buffer size for potentially large lines
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	entries := make([]TokenUsageEntry, 0)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var entry TokenUsageEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			tokenUsageLog.Printf("Skipping invalid JSON at line %d: %v", lineNum, err)
			continue
		}
		entries = append(entries, entry)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading token usage file: %w", err)
	}

	if len(entries) == 0 {
		tokenUsageLog.Print("No token usage entries found")
		return nil, nil
	}

	for _, entry := range entries {
		// Aggregate totals
		summary.TotalInputTokens += entry.InputTokens
		summary.TotalOutputTokens += entry.OutputTokens
		summary.TotalCacheReadTokens += entry.CacheReadTokens
		summary.TotalCacheWriteTokens += entry.CacheWriteTokens
		summary.TotalRequests++
		summary.TotalDurationMs += entry.DurationMs
		summary.TotalResponseBytes += entry.ResponseBytes

		// Aggregate by model
		model := entry.Model
		if model == "" {
			model = "unknown"
		}
		if _, exists := summary.ByModel[model]; !exists {
			summary.ByModel[model] = &ModelTokenUsage{
				Provider: entry.Provider,
			}
		}
		m := summary.ByModel[model]
		m.InputTokens += entry.InputTokens
		m.OutputTokens += entry.OutputTokens
		m.CacheReadTokens += entry.CacheReadTokens
		m.CacheWriteTokens += entry.CacheWriteTokens
		m.ReasoningTokens += entry.ReasoningTokens
		m.Requests++
		m.DurationMs += entry.DurationMs
		m.ResponseBytes += entry.ResponseBytes
	}

	tokenUsageLog.Printf("Parsed %d entries: %d input, %d output, %d cache_read, %d cache_write, %d requests",
		lineNum, summary.TotalInputTokens, summary.TotalOutputTokens,
		summary.TotalCacheReadTokens, summary.TotalCacheWriteTokens, summary.TotalRequests)

	populateAIC(summary)
	summary.AmbientContext = extractAmbientContextMetrics(entries)

	return summary, nil
}

func parseAgentUsageFile(filePath string) (*TokenUsageSummary, error) {
	cleanPath := filepath.Clean(filePath)
	data, err := os.ReadFile(cleanPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read agent usage file: %w", err)
	}

	var entry agentUsageEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return nil, fmt.Errorf("failed to parse agent usage file: %w", err)
	}

	// Prefer primary_model when set; fall back to model; default to "unknown".
	model := strings.TrimSpace(entry.PrimaryModel)
	if model == "" {
		model = strings.TrimSpace(entry.Model)
	}
	if model == "" {
		model = "unknown"
	}
	// Prefer provider from entry; primary_model entries may omit it.
	provider := strings.TrimSpace(entry.Provider)

	summary := &TokenUsageSummary{
		TotalInputTokens:      entry.InputTokens,
		TotalOutputTokens:     entry.OutputTokens,
		TotalCacheReadTokens:  entry.CacheReadTokens,
		TotalCacheWriteTokens: entry.CacheWriteTokens,
		ByModel:               make(map[string]*ModelTokenUsage),
	}

	hasRawTokenData := summary.TotalInputTokens > 0 ||
		summary.TotalOutputTokens > 0 ||
		summary.TotalCacheReadTokens > 0 ||
		summary.TotalCacheWriteTokens > 0 ||
		entry.ReasoningTokens > 0
	hasTokenData := hasRawTokenData
	if hasTokenData {
		summary.TotalRequests = 1
		summary.ByModel[model] = &ModelTokenUsage{
			Provider: provider,
			TokenCoreMetrics: TokenCoreMetrics{
				InputTokens:      entry.InputTokens,
				OutputTokens:     entry.OutputTokens,
				CacheReadTokens:  entry.CacheReadTokens,
				CacheWriteTokens: entry.CacheWriteTokens,
				ReasoningTokens:  entry.ReasoningTokens,
			},
			Requests: 1,
		}
	}

	ambientInputTokens := entry.InputTokens
	if entry.AmbientContextTokens != nil {
		ambientInputTokens = *entry.AmbientContextTokens
	}
	summary.AmbientContext = &AmbientContextMetrics{
		InputTokens:  ambientInputTokens,
		CachedTokens: entry.CacheReadTokens,
	}

	if entry.AICredits > 0 {
		// Use the pre-computed AI Credits value written by parse_token_usage.cjs.
		// This is more accurate than recomputing from raw token counts because it
		// was computed at the time the run completed with full per-request pricing.
		summary.TotalAIC = entry.AICredits
		if summary.ByModel[model] == nil {
			summary.ByModel[model] = &ModelTokenUsage{}
		}
		summary.ByModel[model].Provider = provider
		summary.ByModel[model].InputTokens = entry.InputTokens
		summary.ByModel[model].OutputTokens = entry.OutputTokens
		summary.ByModel[model].CacheReadTokens = entry.CacheReadTokens
		summary.ByModel[model].CacheWriteTokens = entry.CacheWriteTokens
		summary.ByModel[model].ReasoningTokens = entry.ReasoningTokens
		summary.ByModel[model].AIC = entry.AICredits
	} else if hasRawTokenData {
		populateAIC(summary)
	}

	tokenUsageLog.Printf("Parsed agent usage file: input=%d, output=%d, cache_read=%d, cache_write=%d",
		summary.TotalInputTokens, summary.TotalOutputTokens, summary.TotalCacheReadTokens, summary.TotalCacheWriteTokens)
	return summary, nil
}

func extractUsageRecord(value any) map[string]any {
	record, ok := value.(map[string]any)
	if !ok {
		return nil
	}
	return record
}

func usageNumericValue(parsed map[string]any, usage map[string]any, keys ...string) float64 {
	for _, key := range keys {
		for _, candidate := range []any{usage[key], parsed[key]} {
			switch v := candidate.(type) {
			case float64:
				if !isFinite(v) {
					continue
				}
				return v
			case json.Number:
				if num, err := v.Float64(); err == nil && isFinite(num) {
					return num
				}
			case int:
				return float64(v)
			case int64:
				return float64(v)
			case string:
				if strings.TrimSpace(v) == "" {
					continue
				}
				num := json.Number(v)
				if parsedNum, err := num.Float64(); err == nil && isFinite(parsedNum) {
					return parsedNum
				}
			}
		}
	}
	return 0
}

func usageStringValue(parsed map[string]any, usage map[string]any, keys ...string) string {
	for _, key := range keys {
		for _, candidate := range []any{usage[key], parsed[key]} {
			if value, ok := candidate.(string); ok && strings.TrimSpace(value) != "" {
				return value
			}
		}
	}
	return ""
}

func isFinite(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}

func sumAICFromUsageJSONLFiles(filePaths []string) (float64, bool, error) {
	var totalAIC float64
	found := false

	for _, filePath := range filePaths {
		fileAIC, fileFound, err := processOneUsageJSONLFile(filePath)
		if err != nil {
			return 0, false, err
		}
		totalAIC += fileAIC
		if fileFound {
			found = true
		}
	}

	return totalAIC, found, nil
}

// processOneUsageJSONLFile reads a single usage JSONL file and returns the total AIC
// accumulated from its records. The file is deferred-closed immediately after open.
func processOneUsageJSONLFile(filePath string) (total float64, found bool, err error) {
	file, err := os.Open(filepath.Clean(filePath))
	if err != nil {
		return 0, false, fmt.Errorf("failed to open usage JSONL file %s: %w", filePath, err)
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("failed to close usage JSONL file %s: %w", filePath, closeErr)
		}
	}()

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || !strings.HasPrefix(line, "{") {
			continue
		}

		var parsed map[string]any
		if jsonErr := json.Unmarshal([]byte(line), &parsed); jsonErr != nil {
			continue
		}

		usage := extractUsageRecord(parsed["usage"])
		explicitAICredits := usageNumericValue(parsed, usage, "ai_credits", "aiCredits")
		if explicitAICredits > 0 {
			total += explicitAICredits
			found = true
			continue
		}
		explicitAIC := usageNumericValue(parsed, usage, "aic")
		if explicitAIC > 0 {
			total += explicitAIC
			found = true
			continue
		}

		computedAIC := computeModelInferenceAIC(
			usageStringValue(parsed, usage, "provider"),
			usageStringValue(parsed, usage, "model"),
			int(usageNumericValue(parsed, usage, "input_tokens", "inputTokens")),
			int(usageNumericValue(parsed, usage, "output_tokens", "outputTokens")),
			int(usageNumericValue(parsed, usage, "cache_read_tokens", "cacheReadTokens")),
			int(usageNumericValue(parsed, usage, "cache_write_tokens", "cacheWriteTokens")),
			int(usageNumericValue(parsed, usage, "reasoning_tokens", "reasoningTokens")),
		)
		if computedAIC > 0 {
			total += computedAIC
			found = true
		}
	}
	if scanErr := scanner.Err(); scanErr != nil {
		return 0, false, fmt.Errorf("error reading usage JSONL file %s: %w", filePath, scanErr)
	}
	return total, found, nil
}

func extractAmbientContextMetrics(entries []TokenUsageEntry) *AmbientContextMetrics {
	if len(entries) == 0 {
		return nil
	}

	type orderedTokenEntry struct {
		entry        TokenUsageEntry
		timestamp    time.Time
		hasTimestamp bool
		order        int
	}

	ordered := make([]orderedTokenEntry, 0, len(entries))
	for i, entry := range entries {
		ts, hasTimestamp := parseTokenUsageTimestamp(entry.Timestamp)
		ordered = append(ordered, orderedTokenEntry{
			entry:        entry,
			timestamp:    ts,
			hasTimestamp: hasTimestamp,
			order:        i,
		})
	}

	slices.SortStableFunc(ordered, func(left, right orderedTokenEntry) int {
		if left.hasTimestamp && right.hasTimestamp {
			switch {
			case left.timestamp.Before(right.timestamp):
				return -1
			case right.timestamp.Before(left.timestamp):
				return 1
			default:
				return 0
			}
		}
		if left.hasTimestamp != right.hasTimestamp {
			if left.hasTimestamp {
				return -1
			}
			return 1
		}
		if left.order < right.order {
			return -1
		}
		if left.order > right.order {
			return 1
		}
		return 0
	})

	firstCall := ordered[0].entry
	return &AmbientContextMetrics{
		InputTokens:  firstCall.InputTokens,
		CachedTokens: firstCall.CacheReadTokens,
	}
}

func parseTokenUsageTimestamp(value string) (time.Time, bool) {
	if value == "" {
		return time.Time{}, false
	}
	if ts, err := time.Parse(time.RFC3339Nano, value); err == nil {
		return ts, true
	}
	if ts, err := time.Parse(time.RFC3339, value); err == nil {
		return ts, true
	}
	return time.Time{}, false
}
