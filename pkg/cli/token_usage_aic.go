package cli

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/github/gh-aw/pkg/console"
)

func findUsageJSONLFiles(runDir string) []string {
	usageDir := filepath.Join(runDir, "usage")
	if _, err := os.Stat(usageDir); err != nil {
		return nil
	}

	var files []string
	if walkErr := walkTokenUsageFiles(usageDir, func(path string, info os.FileInfo) error {
		if strings.HasSuffix(strings.ToLower(info.Name()), ".jsonl") {
			files = append(files, path)
		}
		return nil
	}); walkErr != nil && !errors.Is(walkErr, filepath.SkipAll) {
		tokenUsageLog.Printf("usage walk error at %s: %v", usageDir, walkErr)
	}

	sort.Strings(files)
	return files
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

// analyzeTokenUsageAICOnly parses token usage inputs and computes only TotalAIC.
// It intentionally skips effective-token computation for callers that only need cost.
func analyzeTokenUsageAICOnly(runDir string, verbose bool) (*TokenUsageSummary, error) {
	tokenUsageLog.Printf("Analyzing token usage (AIC only) in: %s", runDir)

	usageJSONLFiles := findUsageJSONLFiles(runDir)
	if len(usageJSONLFiles) > 0 {
		console.LogVerbose(verbose, "  Found usage JSONL files: "+strings.Join(usageJSONLFiles, ", "))
		totalAIC, found, err := sumAICFromUsageJSONLFiles(usageJSONLFiles)
		if err != nil {
			return nil, err
		}
		if found {
			return &TokenUsageSummary{TotalAIC: totalAIC}, nil
		}
	}

	filePath := findTokenUsageFile(runDir)
	if filePath != "" {
		fileInfo, _ := os.Stat(filePath)
		if fileInfo != nil {
			console.LogVerbose(verbose, fmt.Sprintf("  Found token usage file: %s (%d bytes)", filepath.Base(filePath), fileInfo.Size()))
		}

		file, err := os.Open(filePath)
		if err != nil {
			return nil, fmt.Errorf("failed to open token usage file: %w", err)
		}
		defer file.Close()

		totalAIC := 0.0
		found := false
		scanner := bufio.NewScanner(file)
		scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" {
				continue
			}
			var entry TokenUsageEntry
			if err := json.Unmarshal([]byte(line), &entry); err != nil {
				continue
			}
			model := entry.Model
			if model == "" {
				model = "unknown"
			}
			totalAIC += computeModelInferenceAIC(entry.Provider, model, entry.InputTokens, entry.OutputTokens, entry.CacheReadTokens, entry.CacheWriteTokens, entry.ReasoningTokens)
			found = true
		}
		if err := scanner.Err(); err != nil {
			return nil, fmt.Errorf("error reading token usage file: %w", err)
		}
		if found {
			return &TokenUsageSummary{TotalAIC: totalAIC}, nil
		}
	}

	agentUsagePath := findAgentUsageFile(runDir)
	if agentUsagePath == "" {
		return nil, nil
	}
	agentFileInfo, _ := os.Stat(agentUsagePath)
	if agentFileInfo != nil {
		console.LogVerbose(verbose, fmt.Sprintf("  Found agent usage file: %s (%d bytes)", filepath.Base(agentUsagePath), agentFileInfo.Size()))
	}

	summary, err := parseAgentUsageFile(agentUsagePath)
	if err != nil || summary == nil {
		return summary, err
	}
	return &TokenUsageSummary{
		TotalAIC: summary.TotalAIC,
	}, nil
}

func populateAIC(summary *TokenUsageSummary) {
	if summary == nil {
		return
	}

	total := 0.0
	for model, usage := range summary.ByModel {
		if usage == nil {
			continue
		}
		aic := computeModelInferenceAIC(usage.Provider, model, usage.InputTokens, usage.OutputTokens, usage.CacheReadTokens, usage.CacheWriteTokens, usage.ReasoningTokens)
		usage.AIC = aic
		total += aic
	}
	summary.TotalAIC = total
}
