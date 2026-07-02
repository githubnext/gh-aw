//go:build !integration

package modelsdev

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// sampleCatalog is a minimal models.dev catalog JSON used in tests.
const sampleCatalog = `{
  "providers": {
    "anthropic": {
      "models": {
        "claude-new-model": {
          "cost": {"input": 3.0, "output": 15.0}
        },
        "claude-no-cost": {}
      }
    },
    "openai": {
      "models": {
        "gpt-99": {
          "cost": {"input": 2.5, "output": 10.0, "cache_read": 1.25}
        }
      }
    },
    "unknown-provider": {
      "models": {
        "some-model": {
          "cost": {"input": 1.0}
        }
      }
    }
  }
}`

func TestParseCatalog(t *testing.T) {
	parsed, err := parseCatalog([]byte(sampleCatalog))
	require.NoError(t, err)

	// Anthropic claude-new-model should be present with per-token pricing.
	require.Contains(t, parsed, "anthropic")
	require.Contains(t, parsed["anthropic"], "claude-new-model")
	pricing := parsed["anthropic"]["claude-new-model"]
	assert.InDelta(t, 3.0/1_000_000, pricing["input"], 1e-15)
	assert.InDelta(t, 15.0/1_000_000, pricing["output"], 1e-15)

	// Models without cost should be excluded.
	assert.NotContains(t, parsed["anthropic"], "claude-no-cost")

	// OpenAI gpt-99 should be present.
	require.Contains(t, parsed, "openai")
	require.Contains(t, parsed["openai"], "gpt-99")
	oaiPricing := parsed["openai"]["gpt-99"]
	assert.InDelta(t, 2.5/1_000_000, oaiPricing["input"], 1e-15)
	assert.InDelta(t, 1.25/1_000_000, oaiPricing["cache_read"], 1e-15)

	// unknown-provider is lowercased and retained (normalizeProvider does not filter).
	assert.Contains(t, parsed, "unknown-provider")
}

func TestParseCostMap(t *testing.T) {
	cases := []struct {
		name string
		raw  map[string]json.RawMessage
		want map[string]float64
	}{
		{
			name: "numeric per-million values",
			raw: map[string]json.RawMessage{
				"input":  json.RawMessage("3.0"),
				"output": json.RawMessage("15.0"),
			},
			want: map[string]float64{"input": 3.0 / 1_000_000, "output": 15.0 / 1_000_000},
		},
		{
			name: "string per-token values",
			raw: map[string]json.RawMessage{
				"input": json.RawMessage(`"0.000003"`),
			},
			want: map[string]float64{"input": 0.000003},
		},
		{
			name: "empty map",
			raw:  nil,
			want: nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := parseCostMap(tc.raw)
			if tc.want == nil {
				assert.Nil(t, got)
				return
			}
			for k, v := range tc.want {
				assert.InDeltaf(t, v, got[k], 1e-15, "key %q", k)
			}
		})
	}
}

func TestFindPricing(t *testing.T) {
	origURL := catalogURL
	origFactory := httpClientFactory
	t.Cleanup(func() {
		catalogCache.Reset()
		catalogURL = origURL
		httpClientFactory = origFactory
	})

	catalogCache.Reset()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(sampleCatalog))
	}))
	defer srv.Close()

	catalogURL = srv.URL
	httpClientFactory = func() *http.Client { return srv.Client() }

	t.Run("found_exact_provider_and_model", func(t *testing.T) {
		pricing, ok := FindPricing(context.Background(), "anthropic", "claude-new-model")
		require.True(t, ok)
		assert.InDelta(t, 3.0/1_000_000, pricing["input"], 1e-15)
	})

	t.Run("cross_provider_fallback", func(t *testing.T) {
		pricing, ok := FindPricing(context.Background(), "", "gpt-99")
		require.True(t, ok)
		assert.Contains(t, pricing, "input")
	})

	t.Run("not_found_returns_false", func(t *testing.T) {
		pricing, ok := FindPricing(context.Background(), "anthropic", "does-not-exist")
		assert.False(t, ok)
		assert.Nil(t, pricing)
	})
}
