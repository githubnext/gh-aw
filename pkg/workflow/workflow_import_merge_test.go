//go:build !integration

package workflow

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestMergeJobsFromYAMLImports_MultipleImportsPreserveOrder ensures that when
// more than one import defines the same built-in job (e.g. jobs.activation),
// their injected step fields are accumulated in import declaration order
// before the main workflow's steps are appended, rather than being reversed.
func TestMergeJobsFromYAMLImports_MultipleImportsPreserveOrder(t *testing.T) {
	c := &Compiler{}

	importA := map[string]any{
		"activation": map[string]any{
			"steps": []any{
				map[string]any{"name": "step-A"},
			},
		},
	}
	importB := map[string]any{
		"activation": map[string]any{
			"steps": []any{
				map[string]any{"name": "step-B"},
			},
		},
	}

	jsonA, err := json.Marshal(importA)
	require.NoError(t, err)
	jsonB, err := json.Marshal(importB)
	require.NoError(t, err)

	mergedJobsJSON := string(jsonA) + "\n" + string(jsonB)

	mainJobs := map[string]any{
		"activation": map[string]any{
			"steps": []any{
				map[string]any{"name": "step-main"},
			},
		},
	}

	result := c.mergeJobsFromYAMLImports(mainJobs, mergedJobsJSON)

	activationJob, ok := result["activation"].(map[string]any)
	require.True(t, ok, "expected activation job in merged result")

	stepsAny, ok := activationJob["steps"].([]any)
	require.True(t, ok, "expected steps field on activation job")
	require.Len(t, stepsAny, 3)

	var names []string
	for _, s := range stepsAny {
		stepMap, ok := s.(map[string]any)
		require.True(t, ok)
		name, ok := stepMap["name"].(string)
		require.True(t, ok)
		names = append(names, name)
	}

	require.Equal(t, []string{"step-A", "step-B", "step-main"}, names)
}
