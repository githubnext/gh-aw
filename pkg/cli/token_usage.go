package cli

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/timeutil"
	"github.com/github/gh-aw/pkg/types"
)

var tokenUsageLog = logger.New("cli:token_usage")

// TokenUsageEntry represents a single line from token-usage.jsonl
type TokenUsageEntry struct {
	Timestamp        string `json:"timestamp"`
	RequestID        string `json:"request_id"`
	Provider         string `json:"provider"`
	Model            string `json:"model"`
	Path             string `json:"path"`
	Status           int    `json:"status"`
	Streaming        bool   `json:"streaming"`
	InputTokens      int    `json:"input_tokens"`
	OutputTokens     int    `json:"output_tokens"`
	CacheReadTokens  int    `json:"cache_read_tokens"`
	CacheWriteTokens int    `json:"cache_write_tokens"`
	// EffectiveTokens is populated by agent_usage.json fallback data. token-usage.jsonl
	// entries usually omit this field and rely on computed effective token totals.
	EffectiveTokens int `json:"effective_tokens"`
	DurationMs      int `json:"duration_ms"`
	ResponseBytes   int `json:"response_bytes"`
}

// AmbientContextMetrics captures token footprint for the first LLM invocation.
type AmbientContextMetrics struct {
	InputTokens     int `json:"input_tokens" console:"header:Ambient Input,format:number"`
	CachedTokens    int `json:"cached_tokens" console:"header:Ambient Cached,format:number"`
	EffectiveTokens int `json:"effective_tokens" console:"header:Ambient Effective,format:number"`
}

// TokenUsageSummary contains aggregated token usage from the firewall proxy
type TokenUsageSummary struct {
	TotalInputTokens      int                         `json:"total_input_tokens" console:"header:Input Tokens,format:number"`
	TotalOutputTokens     int                         `json:"total_output_tokens" console:"header:Output Tokens,format:number"`
	TotalCacheReadTokens  int                         `json:"total_cache_read_tokens" console:"header:Cache Read,format:number"`
	TotalCacheWriteTokens int                         `json:"total_cache_write_tokens" console:"header:Cache Write,format:number"`
	TotalRequests         int                         `json:"total_requests" console:"header:Requests"`
	TotalDurationMs       int                         `json:"total_duration_ms"`
	TotalResponseBytes    int                         `json:"total_response_bytes"`
	CacheEfficiency       float64                     `json:"cache_efficiency"`
	TotalEffectiveTokens  int                         `json:"total_effective_tokens" console:"header:Effective Tokens,format:number"`
	AmbientContext        *AmbientContextMetrics      `json:"ambient_context,omitempty"`
	ByModel               map[string]*ModelTokenUsage `json:"by_model"`
}

// ModelTokenUsage contains per-model token usage statistics
type ModelTokenUsage struct {
	Provider         string `json:"provider"`
	InputTokens      int    `json:"input_tokens" console:"header:Input,format:number"`
	OutputTokens     int    `json:"output_tokens" console:"header:Output,format:number"`
	CacheReadTokens  int    `json:"cache_read_tokens" console:"header:Cache Read,format:number"`
	CacheWriteTokens int    `json:"cache_write_tokens" console:"header:Cache Write,format:number"`
	Requests         int    `json:"requests" console:"header:Requests"`
	DurationMs       int    `json:"duration_ms"`
	ResponseBytes    int    `json:"response_bytes"`
	EffectiveTokens  int    `json:"effective_tokens" console:"header:Effective Tokens,format:number"`
}

// ModelTokenUsageRow is a flattened version for console table rendering
type ModelTokenUsageRow struct {
	Model            string `json:"model" console:"header:Model"`
	Provider         string `json:"provider" console:"header:Provider"`
	InputTokens      int    `json:"input_tokens" console:"header:Input,format:number"`
	OutputTokens     int    `json:"output_tokens" console:"header:Output,format:number"`
	CacheReadTokens  int    `json:"cache_read_tokens" console:"header:Cache Read,format:number"`
	CacheWriteTokens int    `json:"cache_write_tokens" console:"header:Cache Write,format:number"`
	EffectiveTokens  int    `json:"effective_tokens" console:"header:Effective Tokens,format:number"`
	Requests         int    `json:"requests" console:"header:Requests"`
	AvgDuration      string `json:"avg_duration" console:"header:Avg Duration"`
}

// tokenUsageJSONLPath is the relative path within the firewall logs directory
const tokenUsageJSONLPath = "api-proxy-logs/token-usage.jsonl"
const agentUsageJSONPath = "agent_usage.json"

// parseTokenUsageFile parses a token-usage.jsonl file and returns the aggregated summary.
// Custom weights, when non-nil, override the built-in model multipliers and token class
// weights for effective token computation.
func parseTokenUsageFile(filePath string, customWeights *types.TokenWeights) (*TokenUsageSummary, error) {
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
		m.Requests++
		m.DurationMs += entry.DurationMs
		m.ResponseBytes += entry.ResponseBytes
	}

	// Compute cache efficiency: cache_read / (input + cache_read)
	totalInputPlusCacheRead := summary.TotalInputTokens + summary.TotalCacheReadTokens
	if totalInputPlusCacheRead > 0 {
		summary.CacheEfficiency = float64(summary.TotalCacheReadTokens) / float64(totalInputPlusCacheRead)
	}

	tokenUsageLog.Printf("Parsed %d entries: %d input, %d output, %d cache_read, %d cache_write, %d requests",
		lineNum, summary.TotalInputTokens, summary.TotalOutputTokens,
		summary.TotalCacheReadTokens, summary.TotalCacheWriteTokens, summary.TotalRequests)

	// Compute effective tokens using per-model multipliers (with optional custom overrides)
	populateEffectiveTokensWithCustomWeights(summary, customWeights)
	summary.AmbientContext = extractAmbientContextMetrics(entries)

	return summary, nil
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

	sort.SliceStable(ordered, func(i, j int) bool {
		left := ordered[i]
		right := ordered[j]
		if left.hasTimestamp && right.hasTimestamp {
			return left.timestamp.Before(right.timestamp)
		}
		if left.hasTimestamp != right.hasTimestamp {
			return left.hasTimestamp
		}
		return left.order < right.order
	})

	firstCall := ordered[0].entry
	return &AmbientContextMetrics{
		InputTokens:     firstCall.InputTokens,
		CachedTokens:    firstCall.CacheReadTokens,
		EffectiveTokens: firstCall.InputTokens + firstCall.CacheReadTokens,
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

// findTokenUsageFile searches for token-usage.jsonl in the run directory
func findTokenUsageFile(runDir string) string {
	// Primary path: sandbox/firewall/logs/api-proxy-logs/token-usage.jsonl
	primary := filepath.Join(runDir, "sandbox", "firewall", "logs", tokenUsageJSONLPath)
	if _, err := os.Stat(primary); err == nil {
		tokenUsageLog.Printf("Found token usage file at primary path: %s", primary)
		return primary
	}

	// Check legacy firewall-audit-logs artifact directory (backward compat for older runs)
	entries, err := os.ReadDir(runDir)
	if err != nil {
		return ""
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasPrefix(name, "firewall-audit-logs") || strings.HasPrefix(name, "firewall-logs") {
			candidate := filepath.Join(runDir, name, tokenUsageJSONLPath)
			if _, err := os.Stat(candidate); err == nil {
				tokenUsageLog.Printf("Found token usage file in %s: %s", name, candidate)
				return candidate
			}
		}
	}

	// Walk sandbox directory for any token-usage.jsonl
	_ = filepath.Walk(runDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if info.Name() == "token-usage.jsonl" {
			primary = path
			return filepath.SkipAll
		}
		return nil
	})
	if primary != filepath.Join(runDir, "sandbox", "firewall", "logs", tokenUsageJSONLPath) {
		tokenUsageLog.Printf("Found token usage file via walk: %s", primary)
		return primary
	}

	tokenUsageLog.Print("No token usage file found")
	return ""
}

// findAgentUsageFile searches for agent_usage.json in the run directory.
func findAgentUsageFile(runDir string) string {
	primary := filepath.Join(runDir, agentUsageJSONPath)
	if _, err := os.Stat(primary); err == nil {
		tokenUsageLog.Printf("Found agent usage file at primary path: %s", primary)
		return primary
	}

	var found string
	_ = filepath.Walk(runDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info == nil || info.IsDir() {
			return nil
		}
		if info.Name() == agentUsageJSONPath {
			found = path
			return filepath.SkipAll
		}
		return nil
	})

	if found != "" {
		tokenUsageLog.Printf("Found agent usage file via walk: %s", found)
	}
	return found
}

func parseAgentUsageFile(filePath string, customWeights *types.TokenWeights) (*TokenUsageSummary, error) {
	cleanPath := filepath.Clean(filePath)
	data, err := os.ReadFile(cleanPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read agent usage file: %w", err)
	}

	var entry TokenUsageEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return nil, fmt.Errorf("failed to parse agent usage file: %w", err)
	}

	summary := &TokenUsageSummary{
		TotalInputTokens:      entry.InputTokens,
		TotalOutputTokens:     entry.OutputTokens,
		TotalCacheReadTokens:  entry.CacheReadTokens,
		TotalCacheWriteTokens: entry.CacheWriteTokens,
		TotalEffectiveTokens:  entry.EffectiveTokens,
		ByModel:               make(map[string]*ModelTokenUsage),
	}

	totalInputPlusCacheRead := summary.TotalInputTokens + summary.TotalCacheReadTokens
	if totalInputPlusCacheRead > 0 {
		summary.CacheEfficiency = float64(summary.TotalCacheReadTokens) / float64(totalInputPlusCacheRead)
	}

	hasTokenData := summary.TotalInputTokens > 0 ||
		summary.TotalOutputTokens > 0 ||
		summary.TotalCacheReadTokens > 0 ||
		summary.TotalCacheWriteTokens > 0 ||
		summary.TotalEffectiveTokens > 0
	if hasTokenData {
		summary.TotalRequests = 1
		summary.ByModel["unknown"] = &ModelTokenUsage{
			Provider:         entry.Provider,
			InputTokens:      entry.InputTokens,
			OutputTokens:     entry.OutputTokens,
			CacheReadTokens:  entry.CacheReadTokens,
			CacheWriteTokens: entry.CacheWriteTokens,
			EffectiveTokens:  entry.EffectiveTokens,
			Requests:         1,
		}
	}

	summary.AmbientContext = &AmbientContextMetrics{
		InputTokens:     entry.InputTokens,
		CachedTokens:    entry.CacheReadTokens,
		EffectiveTokens: entry.InputTokens + entry.CacheReadTokens,
	}

	// If the file does not include effective_tokens, compute it using resolved
	// token weights (custom aw_info weights when available, otherwise defaults).
	if summary.TotalEffectiveTokens == 0 {
		populateEffectiveTokensWithCustomWeights(summary, customWeights)
	}

	tokenUsageLog.Printf("Parsed agent usage file: input=%d, output=%d, cache_read=%d, cache_write=%d, effective=%d",
		summary.TotalInputTokens, summary.TotalOutputTokens, summary.TotalCacheReadTokens, summary.TotalCacheWriteTokens, summary.TotalEffectiveTokens)
	return summary, nil
}

// analyzeTokenUsage finds and parses the token-usage.jsonl file from a run directory.
// It automatically reads custom token weights from aw_info.json when present and
// applies them to the effective token computation.
func analyzeTokenUsage(runDir string, verbose bool) (*TokenUsageSummary, error) {
	tokenUsageLog.Printf("Analyzing token usage in: %s", runDir)

	filePath := findTokenUsageFile(runDir)
	if filePath != "" {
		if verbose {
			fileInfo, _ := os.Stat(filePath)
			if fileInfo != nil {
				fmt.Fprintf(os.Stderr, "  Found token usage file: %s (%d bytes)\n", filepath.Base(filePath), fileInfo.Size())
			}
		}

		// Try to load custom token weights from aw_info.json for this run
		customWeights := extractCustomTokenWeightsFromDir(runDir)
		return parseTokenUsageFile(filePath, customWeights)
	}

	agentUsagePath := findAgentUsageFile(runDir)
	if agentUsagePath == "" {
		return nil, nil
	}
	if verbose {
		fileInfo, _ := os.Stat(agentUsagePath)
		if fileInfo != nil {
			fmt.Fprintf(os.Stderr, "  Found agent usage file: %s (%d bytes)\n", filepath.Base(agentUsagePath), fileInfo.Size())
		}
	}

	customWeights := extractCustomTokenWeightsFromDir(runDir)
	return parseAgentUsageFile(agentUsagePath, customWeights)
}

// extractCustomTokenWeightsFromDir reads aw_info.json from a run directory and returns
// any custom token weights embedded there at compile time. Returns nil when not found.
func extractCustomTokenWeightsFromDir(runDir string) *types.TokenWeights {
	awInfoPath := findAwInfoPath(runDir)
	if awInfoPath == "" {
		return nil
	}
	awInfo, err := parseAwInfo(awInfoPath, false)
	if err != nil || awInfo == nil {
		return nil
	}
	return awInfo.TokenWeights
}

// TotalTokens returns the sum of all token types
func (s *TokenUsageSummary) TotalTokens() int {
	return s.TotalInputTokens + s.TotalOutputTokens + s.TotalCacheReadTokens + s.TotalCacheWriteTokens
}

// AvgDurationMs returns the average request duration in milliseconds
func (s *TokenUsageSummary) AvgDurationMs() int {
	if s.TotalRequests == 0 {
		return 0
	}
	return s.TotalDurationMs / s.TotalRequests
}

// ModelRows returns the by-model data as sorted rows for console rendering
func (s *TokenUsageSummary) ModelRows() []ModelTokenUsageRow {
	rows := make([]ModelTokenUsageRow, 0, len(s.ByModel))
	for model, usage := range s.ByModel {
		avgDur := 0
		if usage.Requests > 0 {
			avgDur = usage.DurationMs / usage.Requests
		}
		rows = append(rows, ModelTokenUsageRow{
			Model:            model,
			Provider:         usage.Provider,
			InputTokens:      usage.InputTokens,
			OutputTokens:     usage.OutputTokens,
			CacheReadTokens:  usage.CacheReadTokens,
			CacheWriteTokens: usage.CacheWriteTokens,
			EffectiveTokens:  usage.EffectiveTokens,
			Requests:         usage.Requests,
			AvgDuration:      timeutil.FormatDurationMs(avgDur),
		})
	}
	// Sort by total tokens descending
	sort.Slice(rows, func(i, j int) bool {
		iTot := rows[i].InputTokens + rows[i].OutputTokens + rows[i].CacheReadTokens + rows[i].CacheWriteTokens
		jTot := rows[j].InputTokens + rows[j].OutputTokens + rows[j].CacheReadTokens + rows[j].CacheWriteTokens
		return iTot > jTot
	})
	return rows
}
