//go:build !integration

package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMergeExcludedEnv_SingleImport(t *testing.T) {
	acc := newImportAccumulator()
	fm := map[string]any{
		"excluded-env": []any{"TOKEN_A", "TOKEN_B"},
	}
	acc.mergeExcludedEnv(fm)
	assert.Equal(t, []string{"TOKEN_A", "TOKEN_B"}, acc.excludedEnv)
}

func TestMergeExcludedEnv_Deduplication(t *testing.T) {
	acc := newImportAccumulator()
	fm1 := map[string]any{"excluded-env": []any{"TOKEN_A", "TOKEN_B"}}
	fm2 := map[string]any{"excluded-env": []any{"TOKEN_B", "TOKEN_C"}}
	acc.mergeExcludedEnv(fm1)
	acc.mergeExcludedEnv(fm2)
	// TOKEN_B should appear only once
	assert.Equal(t, []string{"TOKEN_A", "TOKEN_B", "TOKEN_C"}, acc.excludedEnv)
}

func TestMergeExcludedEnv_EmptyOrMissing(t *testing.T) {
	acc := newImportAccumulator()
	acc.mergeExcludedEnv(map[string]any{})
	acc.mergeExcludedEnv(map[string]any{"excluded-env": []any{}})
	assert.Empty(t, acc.excludedEnv)
}

func TestToImportsResult_MergedExcludedEnv(t *testing.T) {
	acc := newImportAccumulator()
	fm := map[string]any{"excluded-env": []any{"MY_TOKEN"}}
	acc.mergeExcludedEnv(fm)
	result := acc.toImportsResult(nil)
	assert.Equal(t, []string{"MY_TOKEN"}, result.MergedExcludedEnv)
}

// TestHasNodeRuntimeRunInstallScripts verifies that hasNodeRuntimeRunInstallScripts
// correctly navigates the nested runtimes.node.run-install-scripts frontmatter path
// and safely handles missing keys or unexpected value types at every level.
func TestHasNodeRuntimeRunInstallScripts(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		fm   map[string]any
		want bool
	}{
		{
			name: "nil frontmatter",
			fm:   nil,
			want: false,
		},
		{
			name: "empty frontmatter",
			fm:   map[string]any{},
			want: false,
		},
		{
			name: "missing runtimes key",
			fm:   map[string]any{"other": "value"},
			want: false,
		},
		{
			name: "runtimes not a map",
			fm:   map[string]any{"runtimes": "not-a-map"},
			want: false,
		},
		{
			name: "runtimes missing node key",
			fm:   map[string]any{"runtimes": map[string]any{"python": map[string]any{}}},
			want: false,
		},
		{
			name: "node not a map",
			fm:   map[string]any{"runtimes": map[string]any{"node": "not-a-map"}},
			want: false,
		},
		{
			name: "node missing run-install-scripts key",
			fm:   map[string]any{"runtimes": map[string]any{"node": map[string]any{}}},
			want: false,
		},
		{
			name: "run-install-scripts not a bool",
			fm: map[string]any{"runtimes": map[string]any{"node": map[string]any{
				"run-install-scripts": "true",
			}}},
			want: false,
		},
		{
			name: "run-install-scripts false",
			fm: map[string]any{"runtimes": map[string]any{"node": map[string]any{
				"run-install-scripts": false,
			}}},
			want: false,
		},
		{
			name: "run-install-scripts true",
			fm: map[string]any{"runtimes": map[string]any{"node": map[string]any{
				"run-install-scripts": true,
			}}},
			want: true,
		},
		{
			name: "run-install-scripts true with sibling runtimes",
			fm: map[string]any{"runtimes": map[string]any{
				"python": map[string]any{"run-install-scripts": true},
				"node":   map[string]any{"run-install-scripts": true},
			}},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := hasNodeRuntimeRunInstallScripts(tt.fm)
			assert.Equal(t, tt.want, got)
		})
	}
}
