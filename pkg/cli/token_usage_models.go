package cli

import (
	"slices"

	"github.com/github/gh-aw/pkg/timeutil"
)

// TokenCoreMetrics is the single source of truth for the token-usage quartet
// shared across per-request, per-model, and per-run representations.
// All JSON tags use snake_case to match the token-usage.jsonl file format.
type TokenCoreMetrics struct {
	InputTokens      int `json:"input_tokens" console:"header:Input,format:number"`
	OutputTokens     int `json:"output_tokens" console:"header:Output,format:number"`
	CacheReadTokens  int `json:"cache_read_tokens" console:"header:Cache Read,format:number"`
	CacheWriteTokens int `json:"cache_write_tokens" console:"header:Cache Write,format:number"`
	ReasoningTokens  int `json:"reasoning_tokens,omitempty"`
	EffectiveTokens  int `json:"effective_tokens,omitempty"`
}

// TokenUsageEntry represents a single line from token-usage.jsonl
type TokenUsageEntry struct {
	Schema    string `json:"_schema,omitempty"` // Self-describing record type, e.g. "token-usage/v0.26.0"
	Timestamp string `json:"timestamp"`
	RequestID string `json:"request_id"`
	Provider  string `json:"provider"`
	Model     string `json:"model"`
	Path      string `json:"path"`
	Status    int    `json:"status"`
	Streaming bool   `json:"streaming"`
	TokenCoreMetrics
	DurationMs    int `json:"duration_ms"`
	ResponseBytes int `json:"response_bytes"`
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
	Provider string `json:"provider"`
	TokenCoreMetrics
	Requests      int     `json:"requests" console:"header:Requests"`
	DurationMs    int     `json:"duration_ms"`
	ResponseBytes int     `json:"response_bytes"`
	AIC           float64 `json:"aic,omitempty"`
}

// ModelTokenUsageRow is a table-rendering view of per-model token statistics.
// Keep this row schema limited to the token quartet to preserve output shape.
type ModelTokenUsageRow struct {
	Model            string  `json:"model" console:"header:Model"`
	Provider         string  `json:"provider" console:"header:Provider"`
	InputTokens      int     `json:"input_tokens" console:"header:Input,format:number"`
	OutputTokens     int     `json:"output_tokens" console:"header:Output,format:number"`
	CacheReadTokens  int     `json:"cache_read_tokens" console:"header:Cache Read,format:number"`
	CacheWriteTokens int     `json:"cache_write_tokens" console:"header:Cache Write,format:number"`
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
		// Defensive guard: sibling aggregation helpers already skip nil entries,
		// and ModelRows follows that convention to avoid panics in tests/future callers.
		if usage == nil {
			continue
		}
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
