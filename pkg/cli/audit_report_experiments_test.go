//go:build !integration

package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFindExperimentStatePath(t *testing.T) {
	t.Run("returns empty when logsPath is empty", func(t *testing.T) {
		assert.Empty(t, findExperimentStatePath(""), "should return empty string for empty logsPath")
	})

	t.Run("finds state.json at root", func(t *testing.T) {
		dir := t.TempDir()
		statePath := filepath.Join(dir, "state.json")
		require.NoError(t, os.WriteFile(statePath, []byte("{}"), 0o600))

		got := findExperimentStatePath(dir)
		assert.Equal(t, statePath, got, "should find state.json at logsPath root")
	})

	t.Run("finds state.json in experiment subdirectory", func(t *testing.T) {
		dir := t.TempDir()
		subDir := filepath.Join(dir, "experiment")
		require.NoError(t, os.MkdirAll(subDir, 0o755))
		statePath := filepath.Join(subDir, "state.json")
		require.NoError(t, os.WriteFile(statePath, []byte("{}"), 0o600))

		got := findExperimentStatePath(dir)
		assert.Equal(t, statePath, got, "should find state.json in experiment subdirectory")
	})

	t.Run("returns empty when no state.json exists", func(t *testing.T) {
		dir := t.TempDir()
		got := findExperimentStatePath(dir)
		assert.Empty(t, got, "should return empty string when no state.json found")
	})
}

func TestExtractExperimentData(t *testing.T) {
	t.Run("returns nil for empty logsPath", func(t *testing.T) {
		assert.Nil(t, extractExperimentData(""), "should return nil for empty logsPath")
	})

	t.Run("returns nil when no state.json present", func(t *testing.T) {
		dir := t.TempDir()
		assert.Nil(t, extractExperimentData(dir), "should return nil when state.json missing")
	})

	t.Run("returns nil for invalid JSON", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, "state.json"), []byte("not-json"), 0o600))
		assert.Nil(t, extractExperimentData(dir), "should return nil for invalid JSON")
	})

	t.Run("returns nil for empty counts", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, "state.json"), []byte(`{"counts":{}}`), 0o600))
		assert.Nil(t, extractExperimentData(dir), "should return nil when counts map is empty")
	})

	t.Run("extracts single experiment with two variants", func(t *testing.T) {
		dir := t.TempDir()
		state := map[string]any{
			"counts": map[string]any{
				"caveman": map[string]int{"yes": 3, "no": 2},
			},
		}
		raw, err := json.Marshal(state)
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(filepath.Join(dir, "state.json"), raw, 0o600))

		got := extractExperimentData(dir)
		require.NotNil(t, got, "should return non-nil ExperimentData")
		assert.Equal(t, "yes", got.Assignments["caveman"], "variant with highest count should be selected")
		assert.Equal(t, 3, got.CumulativeCounts["caveman"]["yes"], "cumulative count for yes should be 3")
		assert.Equal(t, 2, got.CumulativeCounts["caveman"]["no"], "cumulative count for no should be 2")
	})

	t.Run("reads state.json from experiment subdirectory", func(t *testing.T) {
		dir := t.TempDir()
		subDir := filepath.Join(dir, "experiment")
		require.NoError(t, os.MkdirAll(subDir, 0o755))
		state := map[string]any{
			"counts": map[string]any{
				"style": map[string]int{"concise": 1, "detailed": 2},
			},
		}
		raw, err := json.Marshal(state)
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(filepath.Join(subDir, "state.json"), raw, 0o600))

		got := extractExperimentData(dir)
		require.NotNil(t, got, "should return non-nil ExperimentData from subdir")
		assert.Equal(t, "detailed", got.Assignments["style"], "detailed has higher count so should be selected")
	})

	t.Run("extracts multiple experiments", func(t *testing.T) {
		dir := t.TempDir()
		state := map[string]any{
			"counts": map[string]any{
				"caveman": map[string]int{"yes": 1, "no": 0},
				"style":   map[string]int{"concise": 2, "detailed": 1},
			},
		}
		raw, err := json.Marshal(state)
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(filepath.Join(dir, "state.json"), raw, 0o600))

		got := extractExperimentData(dir)
		require.NotNil(t, got, "should return non-nil ExperimentData")
		assert.Len(t, got.Assignments, 2, "should have 2 experiment assignments")
		assert.Equal(t, "yes", got.Assignments["caveman"], "caveman should select yes (higher count)")
		assert.Equal(t, "concise", got.Assignments["style"], "style should select concise (higher count)")
	})
}

func TestDeriveLastSelectedVariant(t *testing.T) {
	tests := []struct {
		name     string
		counts   map[string]int
		expected string
	}{
		{
			name:     "returns empty for nil map",
			counts:   map[string]int{},
			expected: "",
		},
		{
			name:     "single variant",
			counts:   map[string]int{"A": 5},
			expected: "A",
		},
		{
			name:     "highest count wins",
			counts:   map[string]int{"A": 2, "B": 5},
			expected: "B",
		},
		{
			name:     "ties broken by sorted order",
			counts:   map[string]int{"A": 3, "B": 3},
			expected: "A",
		},
		{
			name:     "three variants",
			counts:   map[string]int{"yes": 4, "no": 3, "maybe": 2},
			expected: "yes",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := deriveLastSelectedVariant(tt.counts)
			assert.Equal(t, tt.expected, got, "deriveLastSelectedVariant result mismatch")
		})
	}
}
