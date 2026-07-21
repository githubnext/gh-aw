package cli

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/fileutil"
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

// findTokenUsageFile searches for token-usage.jsonl in the run directory
func findTokenUsageFile(runDir string) string {
	usageArtifactCandidate := filepath.Join(runDir, "usage", "agent", "token_usage.jsonl")
	if fileutil.FileExists(usageArtifactCandidate) {
		tokenUsageLog.Printf("Found token usage file in usage artifact: %s", usageArtifactCandidate)
		return usageArtifactCandidate
	}

	// Primary path: sandbox/firewall/logs/api-proxy-logs/token-usage.jsonl
	primary := filepath.Join(runDir, "sandbox", "firewall", "logs", tokenUsageJSONLPath)
	if fileutil.FileExists(primary) {
		tokenUsageLog.Printf("Found token usage file at primary path: %s", primary)
		return primary
	}

	// AWF v0.27.7+ audit-dir path: sandbox/firewall/audit/api-proxy-logs/token-usage.jsonl
	// In newer AWF versions the proxy logs are written under --audit-dir rather than
	// --proxy-logs-dir, so check this path explicitly before falling back to the walk.
	awfAuditPath := filepath.Join(runDir, "sandbox", "firewall", "audit", tokenUsageJSONLPath)
	if fileutil.FileExists(awfAuditPath) {
		tokenUsageLog.Printf("Found token usage file at AWF audit path: %s", awfAuditPath)
		return awfAuditPath
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
			if fileutil.FileExists(candidate) {
				tokenUsageLog.Printf("Found token usage file in %s: %s", name, candidate)
				return candidate
			}
		}
	}

	// Walk sandbox directory for any token-usage.jsonl
	if walkErr := filepath.Walk(runDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			tokenUsageLog.Printf("walk error at %s: %v", path, err)
			return nil
		}
		if info == nil || info.IsDir() {
			return nil
		}
		if info.Name() == "token-usage.jsonl" || info.Name() == "token_usage.jsonl" {
			primary = path
			return filepath.SkipAll
		}
		return nil
	}); walkErr != nil && !errors.Is(walkErr, filepath.SkipAll) {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("filesystem error walking %s: %v", runDir, walkErr)))
	}
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
	if fileutil.FileExists(primary) {
		tokenUsageLog.Printf("Found agent usage file at primary path: %s", primary)
		return primary
	}

	var found string
	if walkErr := filepath.Walk(runDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			tokenUsageLog.Printf("walk error at %s: %v", path, err)
			return nil
		}
		if info == nil || info.IsDir() {
			return nil
		}
		if info.Name() == agentUsageJSONPath {
			found = path
			return filepath.SkipAll
		}
		return nil
	}); walkErr != nil && !errors.Is(walkErr, filepath.SkipAll) {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("filesystem error walking %s: %v", runDir, walkErr)))
	}

	if found != "" {
		tokenUsageLog.Printf("Found agent usage file via walk: %s", found)
	}
	return found
}

// agentUsageEntry is the JSON structure written by parse_token_usage.cjs to
// /tmp/gh-aw/agent_usage.json.  It aggregates the total token counts for a run
// and is included in both the "agent" and "usage" artifacts.
type agentUsageEntry struct {
	// Provider and Model fields are only populated when the usage data came from a
	// single model (legacy per-request format written by older versions of the harness).
	Provider string `json:"provider"`
	Model    string `json:"model"`
	// PrimaryModel is the dominant model for runs that used multiple models.
	PrimaryModel string `json:"primary_model"`
	// Raw token counts.
	InputTokens      int `json:"input_tokens"`
	OutputTokens     int `json:"output_tokens"`
	CacheReadTokens  int `json:"cache_read_tokens"`
	CacheWriteTokens int `json:"cache_write_tokens"`
	ReasoningTokens  int `json:"reasoning_tokens"`
	EffectiveTokens  int `json:"effective_tokens"`
	// AmbientContextTokens is the first-request ambient input token count emitted by parse_token_usage.cjs.
	AmbientContextTokens *int `json:"ambient_context"`
	// AICredits is the pre-computed total AI Credits value written by parse_token_usage.cjs.
	// When present and positive it is used directly so we don't need per-model pricing.
	AICredits float64 `json:"ai_credits"`
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
