package cli

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

func sumAICFromUsageJSONLFiles(filePaths []string) (float64, bool, error) {
	totalAIC, found, _, err := sumAICFromUsageJSONLFilesWithWarnings(filePaths)
	return totalAIC, found, err
}

func sumAICFromUsageJSONLFilesWithWarnings(filePaths []string) (float64, bool, []string, error) {
	var totalAIC float64
	found := false
	warnings := make([]string, 0)
	for _, filePath := range filePaths {
		fileAIC, fileFound, fileWarnings, err := processOneUsageJSONLFile(filePath)
		if err != nil {
			return 0, false, nil, err
		}
		totalAIC += fileAIC
		for _, warning := range fileWarnings {
			if !slices.Contains(warnings, warning) {
				warnings = append(warnings, warning)
			}
		}
		found = found || fileFound
	}
	return totalAIC, found, warnings, nil
}

func processOneUsageJSONLFile(filePath string) (total float64, found bool, warnings []string, err error) {
	if strings.EqualFold(filepath.Base(filePath), "token_usage.jsonl") ||
		strings.EqualFold(filepath.Base(filePath), "token-usage.jsonl") ||
		isAWFTokenUsageJSONLFile(filePath) {
		return processAWFTokenUsageJSONLFile(filePath)
	}
	return processLegacyUsageJSONLFile(filePath)
}

func processAWFTokenUsageJSONLFile(filePath string) (float64, bool, []string, error) {
	summary, err := parseTokenUsageFile(filePath)
	if err != nil {
		return 0, false, nil, err
	}
	if summary == nil {
		return 0, false, nil, nil
	}
	// Preserve the legacy found contract: zero-valued data falls through to
	// agent_usage.json, which may carry a precomputed total.
	return summary.TotalAIC, summary.TotalAIC > 0, summary.Warnings, nil
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
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || !strings.HasPrefix(line, "{") {
			continue
		}
		recordAIC, recordFound := parseLegacyUsageJSONLAIC(line)
		total += recordAIC
		found = found || recordFound
	}
	if scanErr := scanner.Err(); scanErr != nil {
		return 0, false, nil, fmt.Errorf("error reading usage JSONL file %s: %w", filePath, scanErr)
	}
	return total, found, nil, nil
}

func parseLegacyUsageJSONLAIC(line string) (float64, bool) {
	var parsed map[string]any
	if err := json.Unmarshal([]byte(line), &parsed); err != nil {
		return 0, false
	}
	usage := extractUsageRecord(parsed["usage"])
	for _, keys := range [][]string{{"ai_credits", "aiCredits"}, {"aic"}} {
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

func isAWFTokenUsageJSONLFile(filePath string) bool {
	file, err := os.Open(filepath.Clean(filePath))
	if err != nil {
		return false
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || !strings.HasPrefix(line, "{") {
			continue
		}
		var discriminator struct {
			Schema string `json:"_schema"`
			Event  string `json:"event"`
		}
		if err := json.Unmarshal([]byte(line), &discriminator); err != nil {
			continue
		}
		if strings.HasPrefix(discriminator.Schema, "token-usage/") || discriminator.Event == "token_usage" {
			return true
		}
	}
	return false
}
