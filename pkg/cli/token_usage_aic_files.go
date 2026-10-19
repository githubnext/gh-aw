package cli

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func sumAICFromUsageJSONLFilesWithWarnings(filePaths []string) (float64, bool, []string, error) {
	return sumAICFromUsageJSONLFilesWithSeen(filePaths, make(map[string]struct{}))
}

func sumAICFromUsageJSONLFilesWithSeen(filePaths []string, seenAWFRequestIDs map[string]struct{}) (float64, bool, []string, error) {
	var totalAIC float64
	found := false
	warnings := make([]string, 0)
	awfDuplicateRecordCount := 0
	for _, filePath := range filePaths {
		entries, duplicateRecordCount, awfSchemaRecordFound, fileWarnings, err := scanTokenUsageEntriesWithSeen(filePath, nil)
		if err != nil {
			return 0, false, nil, err
		}
		if isKnownAWFTokenUsageJSONLFile(filePath) || awfSchemaRecordFound {
			// Once a file matches the AWF token-usage schema, parse the file as one
			// AWF stream. Older token-usage records may lack _schema/event but still
			// belong to the same stream and should share request deduplication.
			var mirroredRecordCount int
			entries, mirroredRecordCount, err = deduplicateTokenUsageAICStream(entries, seenAWFRequestIDs)
			if err != nil {
				return 0, false, nil, err
			}
			duplicateRecordCount += mirroredRecordCount
			awfDuplicateRecordCount += duplicateRecordCount
			warnings = append(warnings, fileWarnings...)
			// Each proxy stream has its own cumulative counter. Never apply a
			// detection counter to an agent stream's running total.
			if summary := buildTokenUsageSummary(entries, 0); summary != nil {
				totalAIC += summary.TotalAIC
				found = found || summary.AICFound
				warnings = append(warnings, summary.Warnings...)
			}
			continue
		}

		fileAIC, fileFound, fileWarnings, err := processLegacyUsageJSONLFile(filePath)
		if err != nil {
			return 0, false, nil, err
		}
		totalAIC += fileAIC
		found = found || fileFound
		warnings = append(warnings, fileWarnings...)
	}

	if awfDuplicateRecordCount > 0 {
		warnings = append(warnings, fmt.Sprintf("%d duplicate token usage record(s) were ignored by event and request_id.", awfDuplicateRecordCount))
	}
	return totalAIC, found, warnings, nil
}

func deduplicateTokenUsageAICStream(entries []TokenUsageEntry, seenRequestIDs map[string]struct{}) ([]TokenUsageEntry, int, error) {
	unique := make([]TokenUsageEntry, 0, len(entries))
	var runningAIC, mirroredAIC float64
	duplicates := 0
	for _, entry := range orderTokenUsageEntriesForAIC(entries) {
		reportedTotal, _, totalValid := parseOptionalNonNegativeFloat(entry.AICreditsTotal)
		delta, _, deltaValid := parseOptionalNonNegativeFloat(entry.AICreditsThisResponse)
		if !deltaValid {
			includeCache, _, _ := parseOptionalBool(entry.InputTokensIncludeCache)
			delta = computeModelInferenceAICWithCacheSemantics(entry.Provider, entry.Model, entry.InputTokens,
				entry.OutputTokens, entry.CacheReadTokens, entry.CacheWriteTokens, entry.ReasoningTokens, includeCache)
		}
		nextAIC := runningAIC + delta
		if totalValid {
			nextAIC = reportedTotal
		}
		event := entry.Event
		if event == "" {
			event = "token_usage"
		}
		key := event + ":" + entry.RequestID
		_, duplicate := seenRequestIDs[key]
		if entry.RequestID != "" && duplicate {
			// A mirrored prefix may include cumulative cost before its first
			// recorded request. Subtract that prefix, not just its last delta.
			mirroredAIC += nextAIC - runningAIC
			duplicates++
		} else {
			if entry.RequestID != "" {
				seenRequestIDs[key] = struct{}{}
			}
			if totalValid && mirroredAIC != 0 {
				credits, err := json.Marshal(max(0, reportedTotal-mirroredAIC))
				if err != nil {
					return nil, 0, fmt.Errorf("failed to encode deduplicated token usage credits: %w", err)
				}
				entry.AICreditsTotal = credits
			}
			unique = append(unique, entry)
		}
		runningAIC = nextAIC
	}
	return unique, duplicates, nil
}

func isKnownAWFTokenUsageJSONLFile(filePath string) bool {
	return strings.EqualFold(filepath.Base(filePath), "token_usage.jsonl") ||
		strings.EqualFold(filepath.Base(filePath), "token-usage.jsonl")
}

func processLegacyUsageJSONLFile(filePath string) (total float64, found bool, warnings []string, err error) {
	file, err := os.Open(filepath.Clean(filePath))
	if err != nil {
		return 0, false, nil, fmt.Errorf("failed to open usage JSONL file %s: %w", filePath, err)
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("failed to close usage JSONL file %s: %w", filePath, closeErr)
		}
	}()

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	lineNumber := 0
	inspectUsage := strings.Contains(strings.ToLower(filepath.Base(filePath)), "usage")
	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var parsed map[string]any
		if err := json.Unmarshal([]byte(line), &parsed); err != nil {
			if inspectUsage {
				warnings = append(warnings, usageRecordWarning(filePath, lineNumber, "malformed usage record"))
			}
			continue
		}
		recordAIC, recordFound := parseLegacyUsageAIC(parsed)
		total += recordAIC
		found = found || recordFound
		if inspectUsage || legacyUsageRecordHasAccountingFields(parsed) {
			var entry TokenUsageEntry
			if err := json.Unmarshal([]byte(line), &entry); err != nil {
				warnings = append(warnings, usageRecordWarning(filePath, lineNumber, "invalid usage fields"))
			} else if !recordFound {
				warnings = append(warnings, usageRecordWarning(filePath, lineNumber, "unpriced usage"))
			}
		}
	}
	if scanErr := scanner.Err(); scanErr != nil {
		return 0, false, warnings, fmt.Errorf("error reading usage JSONL file %s: %w", filePath, scanErr)
	}
	return total, found, warnings, nil
}

func legacyUsageRecordHasAccountingFields(parsed map[string]any) bool {
	for _, record := range []map[string]any{parsed, extractUsageRecord(parsed["usage"])} {
		for _, key := range []string{
			"ai_credits", "aiCredits", "aic", "input_tokens", "inputTokens", "output_tokens", "outputTokens",
			"cache_read_tokens", "cacheReadTokens", "cache_write_tokens", "cacheWriteTokens", "reasoning_tokens", "reasoningTokens",
		} {
			if _, present := record[key]; present {
				return true
			}
		}
	}
	return false
}

func parseLegacyUsageAIC(parsed map[string]any) (float64, bool) {
	usage := extractUsageRecord(parsed["usage"])
	for _, keys := range [][]string{{"ai_credits", "aiCredits"}, {"aic"}} {
		for _, key := range keys {
			for _, candidate := range []any{usage[key], parsed[key]} {
				if value, ok := candidate.(float64); ok && isFinite(value) && value >= 0 {
					return value, true
				}
				if text, ok := candidate.(string); ok {
					if value, err := strconv.ParseFloat(strings.TrimSpace(text), 64); err == nil && isFinite(value) && value >= 0 {
						return value, true
					}
				}
			}
		}
		if value := usageNumericValue(parsed, usage, keys...); value > 0 {
			return value, true
		}
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
	return computedAIC, computedAIC > 0
}
