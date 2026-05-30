//go:build !integration

package workflow

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSafeOutputStateFieldCoverage verifies that hasAnySafeOutputEnabled and
// hasNonBuiltinSafeOutputsEnabled cover every descriptor-managed pointer field.
// This acts as a regression guard to ensure that when a new safe output
// descriptor is added, the developer is reminded (via a failing test) to also
// update the two direct-check functions.
func TestSafeOutputStateFieldCoverage(t *testing.T) {
	for _, handler := range safeOutputHandlers {
		if handler.StructField == "ReportIncomplete" || handler.StructField == "ThreatDetection" {
			// These are auto-defaulted policy controls, not action handlers. They are
			// intentionally excluded from hasAnySafeOutputEnabled/hasNonBuiltinSafeOutputsEnabled
			// so defaults don't count as explicit user-enabled safe outputs.
			continue
		}
		t.Run(handler.StructField, func(t *testing.T) {
			// Build a SafeOutputsConfig with only this one field set to a non-nil value.
			cfg := &SafeOutputsConfig{}
			val := reflect.ValueOf(cfg).Elem()
			field := val.FieldByName(handler.StructField)
			require.True(t, field.IsValid(),
				"safeOutputHandlers references unknown struct field %q; update descriptors or struct", handler.StructField)
			require.Equal(t, reflect.Pointer, field.Kind(),
				"safeOutputHandlers field %q is expected to be a pointer type", handler.StructField)

			field.Set(reflect.New(field.Type().Elem()))

			// hasAnySafeOutputEnabled must return true for every descriptor field.
			assert.True(t, hasAnySafeOutputEnabled(cfg),
				"hasAnySafeOutputEnabled missing check for field %q; add it to the direct nil-check list", handler.StructField)

			// hasNonBuiltinSafeOutputsEnabled must return true for every non-builtin field.
			if !handler.Builtin {
				assert.True(t, hasNonBuiltinSafeOutputsEnabled(cfg),
					"hasNonBuiltinSafeOutputsEnabled missing check for non-builtin field %q; add it to the direct nil-check list", handler.StructField)
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
