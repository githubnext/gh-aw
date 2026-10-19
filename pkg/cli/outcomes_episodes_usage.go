package cli

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// The general usage reader tolerates malformed/unpriced records. Episode
// efficiency must qualify those gaps even when another stream has usable cost.
func episodeUsageWarnings(runDir string) []string {
	var paths, warnings []string
	usageDir := filepath.Join(runDir, "usage")
	if err := filepath.WalkDir(usageDir, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			if path != usageDir || !os.IsNotExist(walkErr) {
				warnings = append(warnings, fmt.Sprintf("Cannot inspect usage directory: %v", walkErr))
			}
			return nil
		}
		if !entry.IsDir() && strings.EqualFold(filepath.Ext(path), ".jsonl") {
			paths = append(paths, path)
		}
		return nil
	}); err != nil {
		warnings = append(warnings, fmt.Sprintf("Cannot inspect usage directory: %v", err))
	}
	if path := findTokenUsageFile(runDir); path != "" && !slices.Contains(paths, path) {
		paths = append(paths, path)
	}
	for _, path := range paths {
		if !strings.Contains(strings.ToLower(filepath.Base(path)), "usage") {
			continue
		}
		warnings = append(warnings, episodeUsageFileWarnings(path)...)
	}
	return warnings
}

func episodeUsageFileWarnings(path string) (warnings []string) {
	file, err := os.Open(path)
	if err != nil {
		return []string{fmt.Sprintf("Cannot inspect usage file %s: %v", filepath.Base(path), err)}
	}
	defer func() {
		if err := file.Close(); err != nil {
			warnings = append(warnings, fmt.Sprintf("Cannot close usage file %s: %v", filepath.Base(path), err))
		}
	}()
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		location := fmt.Sprintf("%s line %d", filepath.Base(path), lineNumber)
		var record map[string]any
		if err := json.Unmarshal([]byte(line), &record); err != nil {
			warnings = append(warnings, location+": malformed usage record; observed cost may be incomplete")
			continue
		}
		var entry TokenUsageEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			warnings = append(warnings, location+": invalid usage fields; observed cost may be incomplete")
			continue
		}
		if isKnownAWFTokenUsageJSONLFile(path) || entry.Event == "token_usage" || strings.HasPrefix(entry.Schema, "token-usage/") {
			summary := buildTokenUsageSummary([]TokenUsageEntry{entry}, 0)
			if summary != nil && !summary.AICFound {
				warnings = append(warnings, location+": unpriced usage; observed cost may be incomplete")
			}
			continue
		}
		if _, found := parseLegacyUsageJSONLAIC(line); !found {
			usage := extractUsageRecord(record["usage"])
			if len(usage) > 0 || record["input_tokens"] != nil || record["output_tokens"] != nil ||
				record["ai_credits"] != nil || record["aic"] != nil {
				warnings = append(warnings, location+": unpriced usage; observed cost may be incomplete")
			}
		}
	}
	if err := scanner.Err(); err != nil {
		warnings = append(warnings, fmt.Sprintf("Cannot inspect usage file %s: %v", filepath.Base(path), err))
	}
	return warnings
}
