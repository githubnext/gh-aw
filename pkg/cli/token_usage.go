package cli

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/timeutil"
	"github.com/github/gh-aw/pkg/types"
)

var tokenUsageLog = logger.New("cli:token_usage")

// TokenUsageEntry represents a single line from token-usage.jsonl
type TokenUsageEntry struct {
	Schema           string `json:"_schema,omitempty"` // Self-describing record type, e.g. "token-usage/v0.26.0"
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
	ReasoningTokens  int    `json:"reasoning_tokens"`
	EffectiveTokens  int    `json:"effective_tokens"`
	DurationMs       int    `json:"duration_ms"`
	ResponseBytes    int    `json:"response_bytes"`
}

// AmbientContextMetrics captures token footprint for the first LLM invocation.
type AmbientContextMetrics struct {
	InputTokens     int `json:"input_tokens" console:"header:Ambient Input,format:number"`
	CachedTokens    int `json:"cached_tokens" console:"header:Ambient Cached,format:number"`
	EffectiveTokens int `json:"effective_tokens,omitempty"`
}

// TokenUsageSummary contains aggregated token usage from the firewall proxy
type TokenUsageSummary struct {
	TotalInputTokens      int                         `json:"total_input_tokens" console:"header:Input Tokens,format:number"`
	TotalOutputTokens     int                         `json:"total_output_tokens" console:"header:Output Tokens,format:number"`
	TotalCacheReadTokens  int                         `json:"total_cache_read_tokens" console:"header:Cache Read,format:number"`
	TotalCacheWriteTokens int                         `json:"total_cache_write_tokens" console:"header:Cache Write,format:number"`
	TotalRequests         int                         `json:"total_requests" console:"header:Requests"`
	TotalSteeringEvents   int                         `json:"total_steering_events,omitempty" console:"header:Steering Events,format:number,omitempty"`
	TotalDurationMs       int                         `json:"total_duration_ms"`
	TotalResponseBytes    int                         `json:"total_response_bytes"`
	CacheEfficiency       float64                     `json:"cache_efficiency"`
	TotalEffectiveTokens  int                         `json:"total_effective_tokens,omitempty"`
	TotalAIC              float64                     `json:"total_aic,omitempty"`
	AmbientContext        *AmbientContextMetrics      `json:"ambient_context,omitempty"`
	ByModel               map[string]*ModelTokenUsage `json:"by_model"`
	SubagentModelRequests []SubagentModelRequest      `json:"subagent_model_requests,omitempty"`
	SubagentModelActuals  []SubagentModelActual       `json:"subagent_model_actuals,omitempty"`
	MismatchCount         int                         `json:"mismatch_count,omitempty"`
	Warnings              []string                    `json:"warnings,omitempty"`
}

// ModelTokenUsage contains per-model token usage statistics
type ModelTokenUsage struct {
	Provider         string  `json:"provider"`
	InputTokens      int     `json:"input_tokens" console:"header:Input,format:number"`
	OutputTokens     int     `json:"output_tokens" console:"header:Output,format:number"`
	CacheReadTokens  int     `json:"cache_read_tokens" console:"header:Cache Read,format:number"`
	CacheWriteTokens int     `json:"cache_write_tokens" console:"header:Cache Write,format:number"`
	ReasoningTokens  int     `json:"reasoning_tokens,omitempty"`
	Requests         int     `json:"requests" console:"header:Requests"`
	DurationMs       int     `json:"duration_ms"`
	ResponseBytes    int     `json:"response_bytes"`
	EffectiveTokens  int     `json:"effective_tokens,omitempty"`
	AIC              float64 `json:"aic,omitempty"`
}

// ModelTokenUsageRow is a flattened version for console table rendering
type ModelTokenUsageRow struct {
	Model            string  `json:"model" console:"header:Model"`
	Provider         string  `json:"provider" console:"header:Provider"`
	InputTokens      int     `json:"input_tokens" console:"header:Input,format:number"`
	OutputTokens     int     `json:"output_tokens" console:"header:Output,format:number"`
	CacheReadTokens  int     `json:"cache_read_tokens" console:"header:Cache Read,format:number"`
	CacheWriteTokens int     `json:"cache_write_tokens" console:"header:Cache Write,format:number"`
	EffectiveTokens  int     `json:"effective_tokens,omitempty"`
	AIC              float64 `json:"aic,omitempty"`
	Requests         int     `json:"requests" console:"header:Requests"`
	AvgDuration      string  `json:"avg_duration" console:"header:Avg Duration"`
}

// SubagentModelRequest captures requested/effective model attribution for a sub-agent.
type SubagentModelRequest struct {
	AgentName       string `json:"agent_name"`
	RequestedModel  string `json:"requested_model"`
	InvocationCount int    `json:"invocation_count"`
	EffectiveModel  string `json:"effective_model,omitempty"`
	ReasonCode      string `json:"reason_code,omitempty"`
}

// SubagentModelActual captures model usage observed in token-usage logs.
type SubagentModelActual struct {
	Model    string `json:"model"`
	Provider string `json:"provider,omitempty"`
	Requests int    `json:"requests"`
}

// tokenUsageJSONLPath is the relative path within the firewall logs directory
const tokenUsageJSONLPath = "api-proxy-logs/token-usage.jsonl"
const proxyEventsJSONLPath = "api-proxy-logs/events.jsonl"
const agentUsageJSONPath = "agent_usage.json"
const modelMismatchReasonTokenUsageMissing = "TOKEN_USAGE_MISSING"
const modelMismatchReasonModelNotObserved = "REQUESTED_MODEL_NOT_OBSERVED"
const subagentStdioWarning = "partial or incorrect data: sub-agent model requests are inferred from agent-stdio.log; use token_usage.jsonl for reliable token consumption"
const tokenSteeringEventName = "token_steering"
const timeoutSteeringEventName = "timeout_steering"
const awfTokenWarningPrefix = "[AWF TOKEN WARNING]"
const awfTimeWarningPrefix = "[AWF TIME WARNING]"

var subagentDispatchPattern = regexp.MustCompile(`([A-Za-z0-9][A-Za-z0-9._-]*)\(([A-Za-z0-9][A-Za-z0-9._:-]*)\)`)

// parseTokenUsageFile parses a token-usage.jsonl file and returns the aggregated summary.
// Custom weights, when non-nil, override the built-in model multipliers and token class
// weights for effective token computation.
func parseTokenUsageFile(filePath string, _ *types.TokenWeights) (*TokenUsageSummary, error) {
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
	if _, err := os.Stat(usageArtifactCandidate); err == nil {
		tokenUsageLog.Printf("Found token usage file in usage artifact: %s", usageArtifactCandidate)
		return usageArtifactCandidate
	}

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
	if _, err := os.Stat(primary); err == nil {
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

func parseAgentUsageFile(filePath string, _ *types.TokenWeights) (*TokenUsageSummary, error) {
	cleanPath := filepath.Clean(filePath)
	data, err := os.ReadFile(cleanPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read agent usage file: %w", err)
	}

	var entry TokenUsageEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return nil, fmt.Errorf("failed to parse agent usage file: %w", err)
	}

	model := strings.TrimSpace(entry.Model)
	if model == "" {
		model = "unknown"
	}

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
			Provider:         entry.Provider,
			InputTokens:      entry.InputTokens,
			OutputTokens:     entry.OutputTokens,
			CacheReadTokens:  entry.CacheReadTokens,
			CacheWriteTokens: entry.CacheWriteTokens,
			ReasoningTokens:  entry.ReasoningTokens,
			Requests:         1,
		}
	}

	summary.AmbientContext = &AmbientContextMetrics{
		InputTokens:  entry.InputTokens,
		CachedTokens: entry.CacheReadTokens,
	}

	if hasRawTokenData {
		populateAIC(summary)
	}

	tokenUsageLog.Printf("Parsed agent usage file: input=%d, output=%d, cache_read=%d, cache_write=%d",
		summary.TotalInputTokens, summary.TotalOutputTokens, summary.TotalCacheReadTokens, summary.TotalCacheWriteTokens)
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
		summary, err := parseTokenUsageFile(filePath, customWeights)
		if err != nil || summary == nil {
			return summary, err
		}
		summary.TotalSteeringEvents = countAPIProxySteeringEvents(runDir)
		augmentSubagentModelAttribution(runDir, summary)
		return summary, nil
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
	summary, err := parseAgentUsageFile(agentUsagePath, customWeights)
	if err != nil || summary == nil {
		return summary, err
	}
	summary.TotalSteeringEvents = countAPIProxySteeringEvents(runDir)
	augmentSubagentModelAttribution(runDir, summary)
	return summary, nil
}

// analyzeTokenUsageAICOnly parses token usage inputs and computes only TotalAIC.
// It intentionally skips effective-token computation for callers that only need cost.
func analyzeTokenUsageAICOnly(runDir string, verbose bool) (*TokenUsageSummary, error) {
	tokenUsageLog.Printf("Analyzing token usage (AIC only) in: %s", runDir)

	filePath := findTokenUsageFile(runDir)
	if filePath != "" {
		if verbose {
			fileInfo, _ := os.Stat(filePath)
			if fileInfo != nil {
				fmt.Fprintf(os.Stderr, "  Found token usage file: %s (%d bytes)\n", filepath.Base(filePath), fileInfo.Size())
			}
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
		if !found {
			return nil, nil
		}
		return &TokenUsageSummary{TotalAIC: totalAIC}, nil
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

	data, err := os.ReadFile(filepath.Clean(agentUsagePath))
	if err != nil {
		return nil, fmt.Errorf("failed to read agent usage file: %w", err)
	}
	var entry TokenUsageEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return nil, fmt.Errorf("failed to parse agent usage file: %w", err)
	}
	model := entry.Model
	if model == "" {
		model = "unknown"
	}
	return &TokenUsageSummary{
		TotalAIC: computeModelInferenceAIC(entry.Provider, model, entry.InputTokens, entry.OutputTokens, entry.CacheReadTokens, entry.CacheWriteTokens, entry.ReasoningTokens),
	}, nil
}

func countAPIProxySteeringEvents(runDir string) int {
	eventsPath := findAPIProxyEventsFile(runDir)
	if eventsPath == "" {
		return 0
	}
	count, err := parseAPIProxySteeringEvents(eventsPath)
	if err != nil {
		tokenUsageLog.Printf("Failed to parse API proxy events file %s: %v", eventsPath, err)
		return 0
	}
	return count
}

func findAPIProxyEventsFile(runDir string) string {
	primary := filepath.Join(runDir, "sandbox", "firewall", "logs", proxyEventsJSONLPath)
	if _, err := os.Stat(primary); err == nil {
		return primary
	}

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
			candidate := filepath.Join(runDir, name, proxyEventsJSONLPath)
			if _, err := os.Stat(candidate); err == nil {
				return candidate
			}
		}
	}

	return ""
}

// proxyEventsEntry is a JSONL record from api-proxy-logs/events.jsonl.
// The event name appears under one of four field names depending on the proxy version;
// the message field is present on steering events.
type proxyEventsEntry struct {
	// Event name appears under one of these four keys; all are checked.
	Event          string `json:"event"`
	Type           string `json:"type"`
	EventNameSnake string `json:"event_name"`
	EventNameCamel string `json:"eventName"`
	// Message text (present on steering events).
	Message string `json:"message"`
	// Optional RFC3339/RFC3339Nano timestamp (not always present).
	Timestamp string `json:"timestamp"`
}

// eventName returns the normalised event name from whichever field is populated.
func (e proxyEventsEntry) eventName() string {
	for _, v := range []string{e.Event, e.Type, e.EventNameSnake, e.EventNameCamel} {
		if v = strings.TrimSpace(v); v != "" {
			return strings.ToLower(v)
		}
	}
	return ""
}

// scanSteeringEntries reads all valid steering proxyEventsEntry records from r.
// Lines that fail the quick-keyword check or JSON decoding are silently skipped.
// The caller is responsible for the lifetime of r.
func scanSteeringEntries(r io.Reader) ([]proxyEventsEntry, error) {
	var entries []proxyEventsEntry
	scanner := bufio.NewScanner(r)
	buf := make([]byte, maxScannerBufferSize)
	scanner.Buffer(buf, maxScannerBufferSize)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || !containsSteeringKeyword(line) {
			continue
		}
		var entry proxyEventsEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue
		}
		if isSteeringEvent(entry.eventName(), strings.TrimSpace(entry.Message)) {
			entries = append(entries, entry)
		}
	}
	return entries, scanner.Err()
}

func parseAPIProxySteeringEvents(filePath string) (int, error) {
	file, err := os.Open(filepath.Clean(filePath))
	if err != nil {
		return 0, err
	}
	defer file.Close()
	entries, err := scanSteeringEntries(file)
	return len(entries), err
}

func containsSteeringKeyword(line string) bool {
	return strings.Contains(line, "steering") ||
		strings.Contains(line, "STEERING") ||
		strings.Contains(line, "Steering")
}

// isSteeringEvent matches AWF proxy steering events using both event name and
// message format from the firewall specification.
func isSteeringEvent(eventName, message string) bool {
	switch eventName {
	case tokenSteeringEventName:
		return strings.HasPrefix(message, awfTokenWarningPrefix)
	case timeoutSteeringEventName:
		return strings.HasPrefix(message, awfTimeWarningPrefix)
	default:
		return false
	}
}

func augmentSubagentModelAttribution(runDir string, summary *TokenUsageSummary) {
	if summary == nil {
		return
	}

	requests := extractSubagentModelRequests(runDir)
	if len(requests) == 0 {
		return
	}
	addTokenUsageWarning(summary, subagentStdioWarning)

	actuals := make([]SubagentModelActual, 0, len(summary.ByModel))
	observedModels := make(map[string]string, len(summary.ByModel))
	for model, usage := range summary.ByModel {
		if usage == nil || model == "" {
			continue
		}
		actuals = append(actuals, SubagentModelActual{
			Model:    model,
			Provider: usage.Provider,
			Requests: usage.Requests,
		})
		observedModels[model] = usage.Provider
	}
	slices.SortStableFunc(actuals, func(a, b SubagentModelActual) int {
		if a.Requests != b.Requests {
			if a.Requests > b.Requests {
				return -1
			}
			return 1
		}
		switch {
		case a.Model < b.Model:
			return -1
		case a.Model > b.Model:
			return 1
		default:
			return 0
		}
	})
	summary.SubagentModelActuals = actuals

	var fallbackEffectiveModel string
	if len(observedModels) == 1 {
		for model := range observedModels {
			fallbackEffectiveModel = model
		}
	}

	requestRows := make([]SubagentModelRequest, 0, len(requests))
	mismatchCount := 0
	for _, row := range requests {
		if _, ok := observedModels[row.RequestedModel]; ok {
			row.EffectiveModel = row.RequestedModel
		} else {
			row.EffectiveModel = fallbackEffectiveModel
			if len(observedModels) == 0 {
				row.ReasonCode = modelMismatchReasonTokenUsageMissing
			} else {
				row.ReasonCode = modelMismatchReasonModelNotObserved
			}
			mismatchCount += row.InvocationCount
		}
		requestRows = append(requestRows, row)
	}
	summary.SubagentModelRequests = requestRows
	summary.MismatchCount = mismatchCount
}

func addTokenUsageWarning(summary *TokenUsageSummary, warning string) {
	if summary == nil || warning == "" {
		return
	}
	if slices.Contains(summary.Warnings, warning) {
		return
	}
	summary.Warnings = append(summary.Warnings, warning)
}

func extractSubagentModelRequests(runDir string) []SubagentModelRequest {
	agentStdioPath := findAgentStdioFile(runDir)
	if agentStdioPath == "" {
		return nil
	}

	file, err := os.Open(agentStdioPath)
	if err != nil {
		return nil
	}
	defer file.Close()

	type key struct {
		agent string
		model string
	}
	counts := make(map[key]int)

	reader := bufio.NewReader(file)
	for {
		line, readErr := reader.ReadString('\n')
		line = strings.TrimSpace(line)
		if line != "" {
			matches := subagentDispatchPattern.FindAllStringSubmatch(line, -1)
			for _, m := range matches {
				if len(m) < 3 {
					continue
				}
				agentName := strings.TrimSpace(m[1])
				requestedModel := strings.TrimSpace(m[2])
				if agentName == "" || requestedModel == "" {
					continue
				}
				counts[key{agent: agentName, model: requestedModel}]++
			}
		}

		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return nil
		}
	}

	rows := make([]SubagentModelRequest, 0, len(counts))
	for k, n := range counts {
		rows = append(rows, SubagentModelRequest{
			AgentName:       k.agent,
			RequestedModel:  k.model,
			InvocationCount: n,
		})
	}
	slices.SortStableFunc(rows, func(a, b SubagentModelRequest) int {
		if a.AgentName != b.AgentName {
			if a.AgentName < b.AgentName {
				return -1
			}
			return 1
		}
		switch {
		case a.RequestedModel < b.RequestedModel:
			return -1
		case a.RequestedModel > b.RequestedModel:
			return 1
		default:
			return 0
		}
	})
	return rows
}

func findAgentStdioFile(runDir string) string {
	primary := filepath.Join(runDir, "agent-stdio.log")
	if _, err := os.Stat(primary); err == nil {
		return primary
	}

	var found string
	if walkErr := filepath.Walk(runDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info == nil || info.IsDir() {
			return nil
		}
		if info.Name() == "agent-stdio.log" {
			found = path
			return filepath.SkipAll
		}
		return nil
	}); walkErr != nil && !errors.Is(walkErr, filepath.SkipAll) {
		tokenUsageLog.Printf("findAgentStdioFile walk error: %v", walkErr)
	}

	return found
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

func correlateToolCallsWithTokenDelta(toolCalls []MCPToolCall, tokenUsageFile string) []MCPToolCall {
	_ = tokenUsageFile
	return toolCalls
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
			AIC:              usage.AIC,
			Requests:         usage.Requests,
			AvgDuration:      timeutil.FormatDurationMs(avgDur),
		})
	}
	// Sort by total tokens descending
	slices.SortFunc(rows, func(a, b ModelTokenUsageRow) int {
		iTot := a.InputTokens + a.OutputTokens + a.CacheReadTokens + a.CacheWriteTokens
		jTot := b.InputTokens + b.OutputTokens + b.CacheReadTokens + b.CacheWriteTokens
		if iTot > jTot {
			return -1
		}
		if iTot < jTot {
			return 1
		}
		return 0
	})
	return rows
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
