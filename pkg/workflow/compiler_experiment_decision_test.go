//go:build !integration

package workflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractExperimentDecisionConfig(t *testing.T) {
	t.Parallel()
	configs := extractExperimentConfigsFromFrontmatter(map[string]any{
		"experiments": map[string]any{
			"prompt_v2": map[string]any{
				"variants": []any{"control", "candidate"},
				"decision": map[string]any{
					"minimum_effect":       0.05,
					"regression_tolerance": 0.10,
					"confidence":           0.99,
				},
			},
		},
	})

	require.NotNil(t, configs["prompt_v2"].Decision)
	assert.InDelta(t, 0.05, configs["prompt_v2"].Decision.MinimumEffect, 0.000001)
	require.NotNil(t, configs["prompt_v2"].Decision.RegressionTolerance)
	assert.InDelta(t, 0.10, *configs["prompt_v2"].Decision.RegressionTolerance, 0.000001)
	assert.InDelta(t, 0.99, configs["prompt_v2"].Decision.Confidence, 0.000001)
}
