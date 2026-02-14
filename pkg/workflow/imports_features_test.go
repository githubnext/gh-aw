//go:build !integration

package workflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMergeFeaturesWithNoImports(t *testing.T) {
	compiler := NewCompiler()
	topFeatures := map[string]any{
		"feature1": true,
		"feature2": false,
	}

	result, err := compiler.MergeFeatures(topFeatures, nil)
	require.NoError(t, err, "MergeFeatures should not error with nil imports")
	assert.Equal(t, topFeatures, result, "Should return top-level features unchanged when no imports")
}

func TestMergeFeaturesWithEmptyImports(t *testing.T) {
	compiler := NewCompiler()
	topFeatures := map[string]any{
		"feature1": true,
		"feature2": false,
	}

	result, err := compiler.MergeFeatures(topFeatures, []map[string]any{})
	require.NoError(t, err, "MergeFeatures should not error with empty imports")
	assert.Equal(t, topFeatures, result, "Should return top-level features unchanged when imports is empty")
}

func TestMergeFeaturesWithNilTopLevelAndImports(t *testing.T) {
	compiler := NewCompiler()
	importedFeatures := []map[string]any{
		{
			"feature1": true,
			"feature2": "enabled",
		},
	}

	result, err := compiler.MergeFeatures(nil, importedFeatures)
	require.NoError(t, err, "MergeFeatures should not error with nil top-level")
	assert.NotNil(t, result, "Result should not be nil")
	assert.Equal(t, true, result["feature1"], "Should include imported feature1")
	assert.Equal(t, "enabled", result["feature2"], "Should include imported feature2")
}

func TestMergeFeaturesWithSingleImport(t *testing.T) {
	compiler := NewCompiler()
	topFeatures := map[string]any{
		"top-feature": true,
	}
	importedFeatures := []map[string]any{
		{
			"imported-feature1": true,
			"imported-feature2": false,
		},
	}

	result, err := compiler.MergeFeatures(topFeatures, importedFeatures)
	require.NoError(t, err, "MergeFeatures should not error")
	assert.Equal(t, true, result["top-feature"], "Should include top-level feature")
	assert.Equal(t, true, result["imported-feature1"], "Should include imported feature1")
	assert.Equal(t, false, result["imported-feature2"], "Should include imported feature2")
	assert.Len(t, result, 3, "Should have 3 features total")
}

func TestMergeFeaturesWithMultipleImports(t *testing.T) {
	compiler := NewCompiler()
	topFeatures := map[string]any{
		"top-feature": true,
	}
	importedFeatures := []map[string]any{
		{
			"import1-feature": "value1",
		},
		{
			"import2-feature": 123,
		},
		{
			"import3-feature": false,
		},
	}

	result, err := compiler.MergeFeatures(topFeatures, importedFeatures)
	require.NoError(t, err, "MergeFeatures should not error")
	assert.Equal(t, true, result["top-feature"], "Should include top-level feature")
	assert.Equal(t, "value1", result["import1-feature"], "Should include import1 feature")
	assert.Equal(t, 123, result["import2-feature"], "Should include import2 feature")
	assert.Equal(t, false, result["import3-feature"], "Should include import3 feature")
	assert.Len(t, result, 4, "Should have 4 features total")
}

func TestMergeFeaturesTopLevelPrecedence(t *testing.T) {
	compiler := NewCompiler()
	topFeatures := map[string]any{
		"shared-feature": "top-level-value",
		"top-only":       true,
	}
	importedFeatures := []map[string]any{
		{
			"shared-feature": "imported-value",
			"import-only":    false,
		},
	}

	result, err := compiler.MergeFeatures(topFeatures, importedFeatures)
	require.NoError(t, err, "MergeFeatures should not error")
	assert.Equal(t, "top-level-value", result["shared-feature"], "Top-level should override imported feature")
	assert.Equal(t, true, result["top-only"], "Should include top-only feature")
	assert.Equal(t, false, result["import-only"], "Should include import-only feature")
	assert.Len(t, result, 3, "Should have 3 features total")
}

func TestMergeFeaturesMultipleImportsWithConflicts(t *testing.T) {
	compiler := NewCompiler()
	topFeatures := map[string]any{
		"top-feature": "top",
	}
	importedFeatures := []map[string]any{
		{
			"feature-a": "first-import",
			"feature-b": 100,
		},
		{
			"feature-a": "second-import", // Should be ignored (first import wins)
			"feature-c": true,
		},
	}

	result, err := compiler.MergeFeatures(topFeatures, importedFeatures)
	require.NoError(t, err, "MergeFeatures should not error")
	assert.Equal(t, "top", result["top-feature"], "Should include top-level feature")
	assert.Equal(t, "first-import", result["feature-a"], "First import should win for feature-a")
	assert.Equal(t, 100, result["feature-b"], "Should include feature-b from first import")
	assert.Equal(t, true, result["feature-c"], "Should include feature-c from second import")
	assert.Len(t, result, 4, "Should have 4 features total")
}

func TestMergeFeaturesWithVariousValueTypes(t *testing.T) {
	compiler := NewCompiler()
	topFeatures := map[string]any{
		"bool-feature":   true,
		"string-feature": "enabled",
	}
	importedFeatures := []map[string]any{
		{
			"int-feature":   42,
			"float-feature": 3.14,
			"nil-feature":   nil,
			"array-feature": []any{"a", "b", "c"},
			"map-feature": map[string]any{
				"nested": "value",
			},
		},
	}

	result, err := compiler.MergeFeatures(topFeatures, importedFeatures)
	require.NoError(t, err, "MergeFeatures should not error")
	assert.Equal(t, true, result["bool-feature"], "Should include bool feature")
	assert.Equal(t, "enabled", result["string-feature"], "Should include string feature")
	assert.Equal(t, 42, result["int-feature"], "Should include int feature")
	assert.InDelta(t, 3.14, result["float-feature"], 0.001, "Should include float feature")
	assert.Nil(t, result["nil-feature"], "Should include nil feature")
	assert.Equal(t, []any{"a", "b", "c"}, result["array-feature"], "Should include array feature")
	assert.Equal(t, map[string]any{"nested": "value"}, result["map-feature"], "Should include map feature")
}

func TestMergeFeaturesEmptyTopLevelWithMultipleImports(t *testing.T) {
	compiler := NewCompiler()
	topFeatures := map[string]any{}
	importedFeatures := []map[string]any{
		{
			"feature1": true,
		},
		{
			"feature2": false,
		},
	}

	result, err := compiler.MergeFeatures(topFeatures, importedFeatures)
	require.NoError(t, err, "MergeFeatures should not error")
	assert.Equal(t, true, result["feature1"], "Should include feature1 from first import")
	assert.Equal(t, false, result["feature2"], "Should include feature2 from second import")
	assert.Len(t, result, 2, "Should have 2 features total")
}

func TestMergeFeaturesPreservesTopLevelWhenImportsHaveSameFeature(t *testing.T) {
	compiler := NewCompiler()
	topFeatures := map[string]any{
		"feature": false,
	}
	importedFeatures := []map[string]any{
		{
			"feature": true,
		},
	}

	result, err := compiler.MergeFeatures(topFeatures, importedFeatures)
	require.NoError(t, err, "MergeFeatures should not error")
	assert.Equal(t, false, result["feature"], "Top-level value should be preserved")
	assert.Len(t, result, 1, "Should have 1 feature")
}
