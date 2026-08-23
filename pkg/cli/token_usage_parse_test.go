//go:build !integration

package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestExtractAmbientContextMetrics verifies that extractAmbientContextMetrics selects
// the first call chronologically (by parsed timestamp when present, falling back to
// original order for entries with missing or unparsable timestamps) and reports its
// input and cached token counts.
func TestExtractAmbientContextMetrics(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		entries []TokenUsageEntry
		want    *AmbientContextMetrics
	}{
		{
			name:    "nil entries returns nil",
			entries: nil,
			want:    nil,
		},
		{
			name:    "empty entries returns nil",
			entries: []TokenUsageEntry{},
			want:    nil,
		},
		{
			name: "single entry returns its tokens",
			entries: []TokenUsageEntry{
				{
					Timestamp:        "2024-01-01T10:00:00Z",
					TokenCoreMetrics: TokenCoreMetrics{InputTokens: 100, CacheReadTokens: 20},
				},
			},
			want: &AmbientContextMetrics{InputTokens: 100, CachedTokens: 20},
		},
		{
			name: "picks earliest timestamp regardless of slice order",
			entries: []TokenUsageEntry{
				{
					Timestamp:        "2024-01-01T12:00:00Z",
					TokenCoreMetrics: TokenCoreMetrics{InputTokens: 200, CacheReadTokens: 5},
				},
				{
					Timestamp:        "2024-01-01T10:00:00Z",
					TokenCoreMetrics: TokenCoreMetrics{InputTokens: 100, CacheReadTokens: 20},
				},
				{
					Timestamp:        "2024-01-01T11:00:00Z",
					TokenCoreMetrics: TokenCoreMetrics{InputTokens: 150, CacheReadTokens: 15},
				},
			},
			want: &AmbientContextMetrics{InputTokens: 100, CachedTokens: 20},
		},
		{
			name: "entries without timestamps preserve original order",
			entries: []TokenUsageEntry{
				{
					Timestamp:        "",
					TokenCoreMetrics: TokenCoreMetrics{InputTokens: 300, CacheReadTokens: 1},
				},
				{
					Timestamp:        "",
					TokenCoreMetrics: TokenCoreMetrics{InputTokens: 400, CacheReadTokens: 2},
				},
			},
			want: &AmbientContextMetrics{InputTokens: 300, CachedTokens: 1},
		},
		{
			name: "entries with timestamps sort before entries without",
			entries: []TokenUsageEntry{
				{
					Timestamp:        "",
					TokenCoreMetrics: TokenCoreMetrics{InputTokens: 999, CacheReadTokens: 9},
				},
				{
					Timestamp:        "2024-05-05T05:05:05Z",
					TokenCoreMetrics: TokenCoreMetrics{InputTokens: 50, CacheReadTokens: 3},
				},
			},
			want: &AmbientContextMetrics{InputTokens: 50, CachedTokens: 3},
		},
		{
			name: "unparsable timestamp treated as no timestamp",
			entries: []TokenUsageEntry{
				{
					Timestamp:        "not-a-timestamp",
					TokenCoreMetrics: TokenCoreMetrics{InputTokens: 77, CacheReadTokens: 7},
				},
				{
					Timestamp:        "2024-01-01T00:00:00Z",
					TokenCoreMetrics: TokenCoreMetrics{InputTokens: 88, CacheReadTokens: 8},
				},
			},
			want: &AmbientContextMetrics{InputTokens: 88, CachedTokens: 8},
		},
		{
			name: "equal timestamps preserve original order (stable sort)",
			entries: []TokenUsageEntry{
				{
					Timestamp:        "2024-01-01T00:00:00Z",
					TokenCoreMetrics: TokenCoreMetrics{InputTokens: 10, CacheReadTokens: 1},
				},
				{
					Timestamp:        "2024-01-01T00:00:00Z",
					TokenCoreMetrics: TokenCoreMetrics{InputTokens: 20, CacheReadTokens: 2},
				},
			},
			want: &AmbientContextMetrics{InputTokens: 10, CachedTokens: 1},
		},
		{
			name: "RFC3339 without nanoseconds also parses",
			entries: []TokenUsageEntry{
				{
					Timestamp:        "2024-01-01T10:00:00Z",
					TokenCoreMetrics: TokenCoreMetrics{InputTokens: 5, CacheReadTokens: 0},
				},
				{
					Timestamp:        "2024-01-01T09:00:00.123456789Z",
					TokenCoreMetrics: TokenCoreMetrics{InputTokens: 6, CacheReadTokens: 1},
				},
			},
			want: &AmbientContextMetrics{InputTokens: 6, CachedTokens: 1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := extractAmbientContextMetrics(tt.entries)
			assert.Equal(t, tt.want, got)
		})
	}
}
