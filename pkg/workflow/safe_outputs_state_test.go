//go:build !integration

package workflow

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSafeOutputStateFieldCoverage verifies that hasAnySafeOutputEnabled and
// hasNonBuiltinSafeOutputsEnabled cover every pointer field listed in
// safeOutputFieldMapping.  This acts as a regression guard to ensure that when
// a new safe output type is added to safeOutputFieldMapping, the developer is
// reminded (via a failing test) to also update the two direct-check functions.
func TestSafeOutputStateFieldCoverage(t *testing.T) {
	// builtins excluded from hasNonBuiltinSafeOutputsEnabled
	builtins := map[string]bool{
		"NoOp":        true,
		"MissingData": true,
		"MissingTool": true,
	}

	for fieldName := range safeOutputFieldMapping {
		t.Run(fieldName, func(t *testing.T) {
			// Build a SafeOutputsConfig with only this one field set to a non-nil value.
			cfg := &SafeOutputsConfig{}
			val := reflect.ValueOf(cfg).Elem()
			field := val.FieldByName(fieldName)
			require.True(t, field.IsValid(),
				"safeOutputFieldMapping references unknown struct field %q; update the mapping or the struct", fieldName)
			require.Equal(t, reflect.Ptr, field.Kind(),
				"safeOutputFieldMapping field %q is expected to be a pointer type", fieldName)

			field.Set(reflect.New(field.Type().Elem()))

			// hasAnySafeOutputEnabled must return true for every field in the mapping.
			assert.True(t, hasAnySafeOutputEnabled(cfg),
				"hasAnySafeOutputEnabled missing check for field %q; add it to the direct nil-check list", fieldName)

			// hasNonBuiltinSafeOutputsEnabled must return true for every non-builtin field.
			if !builtins[fieldName] {
				assert.True(t, hasNonBuiltinSafeOutputsEnabled(cfg),
					"hasNonBuiltinSafeOutputsEnabled missing check for non-builtin field %q; add it to the direct nil-check list", fieldName)
			}
		})
	}
}

// TestSafeOutputStateCommentMemoryCoverage explicitly tests CommentMemory, which is
// attached to SafeOutputs via tools.comment-memory (not listed in safeOutputFieldMapping)
// and must be checked by both state inspection functions.
func TestSafeOutputStateCommentMemoryCoverage(t *testing.T) {
	cfg := &SafeOutputsConfig{
		CommentMemory: &CommentMemoryConfig{},
	}

	assert.True(t, hasAnySafeOutputEnabled(cfg),
		"hasAnySafeOutputEnabled should return true when CommentMemory is set")
	assert.True(t, hasNonBuiltinSafeOutputsEnabled(cfg),
		"hasNonBuiltinSafeOutputsEnabled should return true when CommentMemory is set")
}
