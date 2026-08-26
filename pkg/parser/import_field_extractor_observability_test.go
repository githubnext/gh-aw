//go:build !integration

package parser

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestExtractOTLPEndpointsFromObsMap verifies that all three endpoint forms
// (string, object, array) are correctly extracted from a raw observability map.
func TestExtractOTLPEndpointsFromObsMap(t *testing.T) {
	tests := []struct {
		name string
		obs  map[string]any
		want []observabilityImportEndpoint
	}{
		{
			name: "nil map returns nil",
			obs:  nil,
			want: nil,
		},
		{
			name: "empty map returns nil",
			obs:  map[string]any{},
			want: nil,
		},
		{
			name: "missing otlp key returns nil",
			obs:  map[string]any{"other": "value"},
			want: nil,
		},
		{
			name: "string form without headers",
			obs: map[string]any{
				"otlp": map[string]any{
					"endpoint": "https://traces.example.com:4317",
				},
			},
			want: []observabilityImportEndpoint{
				{URL: "https://traces.example.com:4317"},
			},
		},
		{
			name: "string form with string headers (backward-compat)",
			obs: map[string]any{
				"otlp": map[string]any{
					"endpoint": "https://traces.example.com:4317",
					"headers":  "Authorization=******",
				},
			},
			want: []observabilityImportEndpoint{
				{URL: "https://traces.example.com:4317", Headers: "Authorization=******"},
			},
		},
		{
			name: "string form with map headers",
			obs: map[string]any{
				"otlp": map[string]any{
					"endpoint": "https://traces.example.com:4317",
					"headers":  map[string]any{"Authorization": "******"},
				},
			},
			want: []observabilityImportEndpoint{
				{URL: "https://traces.example.com:4317", Headers: map[string]any{"Authorization": "******"}},
			},
		},
		{
			name: "object form",
			obs: map[string]any{
				"otlp": map[string]any{
					"endpoint": map[string]any{
						"url":     "https://traces.example.com:4317",
						"headers": map[string]any{"X-API-Key": "key1"},
					},
				},
			},
			want: []observabilityImportEndpoint{
				{URL: "https://traces.example.com:4317", Headers: map[string]any{"X-API-Key": "key1"}},
			},
		},
		{
			name: "object form without headers",
			obs: map[string]any{
				"otlp": map[string]any{
					"endpoint": map[string]any{"url": "https://traces.example.com:4317"},
				},
			},
			want: []observabilityImportEndpoint{
				{URL: "https://traces.example.com:4317"},
			},
		},
		{
			name: "array form with multiple endpoints",
			obs: map[string]any{
				"otlp": map[string]any{
					"endpoint": []any{
						map[string]any{"url": "https://primary.example.com:4317"},
						map[string]any{"url": "https://secondary.example.com:4317", "headers": map[string]any{"X-Key": "v"}},
					},
				},
			},
			want: []observabilityImportEndpoint{
				{URL: "https://primary.example.com:4317"},
				{URL: "https://secondary.example.com:4317", Headers: map[string]any{"X-Key": "v"}},
			},
		},
		{
			name: "array form skips entries with empty URL",
			obs: map[string]any{
				"otlp": map[string]any{
					"endpoint": []any{
						map[string]any{"url": ""},
						map[string]any{"url": "https://valid.example.com:4317"},
					},
				},
			},
			want: []observabilityImportEndpoint{
				{URL: "https://valid.example.com:4317"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractOTLPEndpointsFromObsMap(tt.obs)
			assert.Equal(t, tt.want, got, "extractOTLPEndpointsFromObsMap")
		})
	}
}

// TestMergeObservabilityConfigs verifies that multiple observability JSON blobs are
// merged into a single config with all OTLP endpoints in array form, with deduplication.
func TestMergeObservabilityConfigs(t *testing.T) {
	t.Run("empty slice returns empty string", func(t *testing.T) {
		got := mergeObservabilityConfigs(nil)
		assert.Empty(t, got, "nil configs should return empty string")

		got = mergeObservabilityConfigs([]string{})
		assert.Empty(t, got, "empty configs should return empty string")
	})

	t.Run("single import with string endpoint", func(t *testing.T) {
		configs := []string{`{"otlp":{"endpoint":"https://traces.example.com:4317"}}`}
		got := mergeObservabilityConfigs(configs)
		require.NotEmpty(t, got, "should produce non-empty result")
		assert.Contains(t, got, `"https://traces.example.com:4317"`, "should include the endpoint URL")
	})

	t.Run("two imports with distinct string endpoints", func(t *testing.T) {
		configs := []string{
			`{"otlp":{"endpoint":"https://primary.example.com:4317"}}`,
			`{"otlp":{"endpoint":"https://secondary.example.com:4317"}}`,
		}
		got := mergeObservabilityConfigs(configs)
		require.NotEmpty(t, got, "should produce merged result")
		assert.Contains(t, got, "primary.example.com", "should include first endpoint")
		assert.Contains(t, got, "secondary.example.com", "should include second endpoint")
	})

	t.Run("two imports with same URL deduplicates", func(t *testing.T) {
		configs := []string{
			`{"otlp":{"endpoint":"https://traces.example.com:4317"}}`,
			`{"otlp":{"endpoint":"https://traces.example.com:4317"}}`,
		}
		got := mergeObservabilityConfigs(configs)
		require.NotEmpty(t, got, "should produce merged result")
		// The URL should appear only once in the output
		count := strings.Count(got, "traces.example.com")
		assert.Equal(t, 1, count, "duplicate URL should appear only once")
	})

	t.Run("mix of string and object notation", func(t *testing.T) {
		configs := []string{
			`{"otlp":{"endpoint":"https://primary.example.com:4317"}}`,
			`{"otlp":{"endpoint":{"url":"https://secondary.example.com:4317","headers":{"X-Key":"val"}}}}`,
		}
		got := mergeObservabilityConfigs(configs)
		require.NotEmpty(t, got, "should produce merged result")
		assert.Contains(t, got, "primary.example.com", "should include first endpoint (string form)")
		assert.Contains(t, got, "secondary.example.com", "should include second endpoint (object form)")
		assert.Contains(t, got, "X-Key", "should preserve headers from object form")
	})

	t.Run("import with array notation contributes all endpoints", func(t *testing.T) {
		configs := []string{
			`{"otlp":{"endpoint":[{"url":"https://a.example.com:4317"},{"url":"https://b.example.com:4317"}]}}`,
		}
		got := mergeObservabilityConfigs(configs)
		require.NotEmpty(t, got, "should produce merged result")
		assert.Contains(t, got, "a.example.com", "should include first array endpoint")
		assert.Contains(t, got, "b.example.com", "should include second array endpoint")
	})

	t.Run("three imports with mixed notation and overlap deduplicates correctly", func(t *testing.T) {
		configs := []string{
			// import 1: string form
			`{"otlp":{"endpoint":"https://a.example.com:4317"}}`,
			// import 2: array with one duplicate and one new
			`{"otlp":{"endpoint":[{"url":"https://a.example.com:4317"},{"url":"https://b.example.com:4317"}]}}`,
			// import 3: object form with new endpoint
			`{"otlp":{"endpoint":{"url":"https://c.example.com:4317"}}}`,
		}
		got := mergeObservabilityConfigs(configs)
		require.NotEmpty(t, got, "should produce merged result")

		countA := strings.Count(got, "a.example.com")
		assert.Equal(t, 1, countA, "a.example.com should appear exactly once (dedup)")
		assert.Contains(t, got, "b.example.com", "should include b.example.com")
		assert.Contains(t, got, "c.example.com", "should include c.example.com")
	})

	t.Run("headers preserved in original format (map not string)", func(t *testing.T) {
		configs := []string{
			`{"otlp":{"endpoint":{"url":"https://traces.example.com","headers":{"Authorization":"******"}}}}`,
		}
		got := mergeObservabilityConfigs(configs)
		require.NotEmpty(t, got, "should produce merged result")
		assert.Contains(t, got, "Authorization", "should preserve header key")
		assert.Contains(t, got, "******", "should preserve header value")
	})

	t.Run("invalid JSON blob is skipped", func(t *testing.T) {
		configs := []string{
			`not-valid-json`,
			`{"otlp":{"endpoint":"https://valid.example.com:4317"}}`,
		}
		got := mergeObservabilityConfigs(configs)
		require.NotEmpty(t, got, "valid config should still produce output")
		assert.Contains(t, got, "valid.example.com", "should include valid endpoint")
	})

	t.Run("config with no endpoints returns empty string", func(t *testing.T) {
		configs := []string{`{"otlp":{}}`}
		got := mergeObservabilityConfigs(configs)
		assert.Empty(t, got, "config without endpoints should return empty string")
	})

	t.Run("github-app config is preserved even when no endpoints are set", func(t *testing.T) {
		configs := []string{`{"otlp":{"github-app":{"app-id":"${{ vars.APP_ID }}","private-key":"${{ secrets.APP_PRIVATE_KEY }}"}}}`}
		got := mergeObservabilityConfigs(configs)
		require.NotEmpty(t, got, "github-app-only config should still produce merged observability")
		assert.Contains(t, got, `"github-app"`, "should include github-app block")
		assert.Contains(t, got, `"app-id":"${{ vars.APP_ID }}"`, "should preserve app-id")
		assert.Contains(t, got, `"private-key":"${{ secrets.APP_PRIVATE_KEY }}"`, "should preserve private-key")
	})
}
