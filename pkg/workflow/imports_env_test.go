//go:build !integration

package workflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMergeEnvWithNoImports(t *testing.T) {
	topEnv := map[string]any{
		"KEY1": "value1",
		"KEY2": "value2",
	}

	result, err := mergeEnv(topEnv, "")
	require.NoError(t, err, "mergeEnv should not error with empty imports")
	assert.Equal(t, topEnv, result, "Should return top-level env unchanged when no imports")
}

func TestMergeEnvWithImportedEnvVars(t *testing.T) {
	topEnv := map[string]any{}

	importedJSON := `{"TARGET_REPOSITORY":"owner/repo","SHARED_VAR":"shared-value"}`

	result, err := mergeEnv(topEnv, importedJSON)
	require.NoError(t, err, "mergeEnv should not error with valid imports")
	assert.Equal(t, "owner/repo", result["TARGET_REPOSITORY"], "Should contain imported TARGET_REPOSITORY")
	assert.Equal(t, "shared-value", result["SHARED_VAR"], "Should contain imported SHARED_VAR")
}

func TestMergeEnvTopLevelTakesPrecedence(t *testing.T) {
	topEnv := map[string]any{
		"SHARED_KEY": "main-value",
		"MAIN_ONLY":  "main",
	}

	importedJSON := `{"SHARED_KEY":"import-value","IMPORT_ONLY":"imported"}`

	result, err := mergeEnv(topEnv, importedJSON)
	require.NoError(t, err, "mergeEnv should not error")
	assert.Equal(t, "main-value", result["SHARED_KEY"], "Main workflow env var should override imported")
	assert.Equal(t, "main", result["MAIN_ONLY"], "Main-only var should be present")
	assert.Equal(t, "imported", result["IMPORT_ONLY"], "Import-only var should be merged in")
}

func TestMergeEnvWithMultipleImports(t *testing.T) {
	topEnv := map[string]any{}

	// Two imports: second one overrides first for KEY2
	importedJSON := `{"KEY1":"val1","KEY2":"first"}
{"KEY2":"second","KEY3":"val3"}`

	result, err := mergeEnv(topEnv, importedJSON)
	require.NoError(t, err, "mergeEnv should not error with multiple import lines")
	assert.Equal(t, "val1", result["KEY1"], "KEY1 from first import should be present")
	assert.Equal(t, "second", result["KEY2"], "KEY2 from second import overrides first")
	assert.Equal(t, "val3", result["KEY3"], "KEY3 from second import should be present")
}

func TestMergeEnvWithNilTopLevel(t *testing.T) {
	importedJSON := `{"IMPORTED_VAR":"imported-value"}`

	result, err := mergeEnv(nil, importedJSON)
	require.NoError(t, err, "mergeEnv should not error with nil top-level")
	assert.Equal(t, "imported-value", result["IMPORTED_VAR"], "Imported var should be present")
}
