//go:build !integration

package importinpututil_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/github/gh-aw/pkg/importinpututil"
)

// TestSpec_* functions derive from the importinpututil README.md specification.
// Each test maps to a documented section of the package contract and is written
// against the public API only — not against implementation internals.

// TestSpec_PublicAPI_ResolvePathValue validates the documented behavior of
// ResolvePathValue as described in the package README.md.
//
// Specification:
//   - Resolves either a top-level input key (e.g. "count") or a one-level
//     dotted object sub-key (e.g. "config.apiKey") from a map of import inputs.
//   - Returns (value, true) on success.
//   - Returns (nil, false) when the key is absent or the dotted path cannot be traversed.
func TestSpec_PublicAPI_ResolvePathValue(t *testing.T) {
	inputs := map[string]any{
		"count":  42,
		"config": map[string]any{"apiKey": "secret"},
	}

	tests := []struct {
		name      string
		inputPath string
		wantValue any
		wantOK    bool
	}{
		{name: "documented: top-level key", inputPath: "count", wantValue: 42, wantOK: true},
		{name: "documented: dotted sub-key", inputPath: "config.apiKey", wantValue: "secret", wantOK: true},
		{name: "documented: missing top-level key", inputPath: "missing", wantValue: nil, wantOK: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value, ok := importinpututil.ResolvePathValue(inputs, tt.inputPath)
			assert.Equal(t, tt.wantOK, ok, "ResolvePathValue(%q) ok mismatch for: %s", tt.inputPath, tt.name)
			assert.Equal(t, tt.wantValue, value, "ResolvePathValue(%q) value mismatch for: %s", tt.inputPath, tt.name)
		})
	}
}

// TestSpec_PublicAPI_ResolvePathValue_DottedPathTraversalFailure validates the
// documented "(nil, false)" result when a dotted path cannot be traversed.
//
// Specification:
//   - Returns (nil, false) when the dotted path cannot be traversed.
func TestSpec_PublicAPI_ResolvePathValue_DottedPathTraversalFailure(t *testing.T) {
	inputs := map[string]any{
		"count":  42,
		"config": map[string]any{"apiKey": "secret"},
	}

	tests := []struct {
		name      string
		inputPath string
	}{
		{name: "documented: dotted lookup where top-level key is absent", inputPath: "absent.apiKey"},
		{name: "documented: dotted lookup where top-level value is not an object", inputPath: "count.apiKey"},
		{name: "documented: dotted lookup where sub-key is absent", inputPath: "config.missing"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value, ok := importinpututil.ResolvePathValue(inputs, tt.inputPath)
			assert.False(t, ok, "ResolvePathValue(%q) should return ok=false for: %s", tt.inputPath, tt.name)
			assert.Nil(t, value, "ResolvePathValue(%q) should return nil value for: %s", tt.inputPath, tt.name)
		})
	}
}

// TestSpec_DesignDecision_ResolvePathValue_SingleLevelDotNotation validates the
// documented design note that only one level of dot notation is supported.
//
// Specification (Design Notes):
//   - Only one level of dot notation is supported for sub-key resolution:
//     "a.b" is valid; "a.b.c" is treated as a lookup for key "b.c" inside the
//     top-level object "a", which will typically fail.
func TestSpec_DesignDecision_ResolvePathValue_SingleLevelDotNotation(t *testing.T) {
	inputs := map[string]any{
		"a": map[string]any{
			"b":   map[string]any{"c": "deep"},
			"b.c": "literal-sub-key",
		},
	}

	// "a.b.c" is split only on the first dot, so the sub-key looked up is the
	// literal "b.c" inside object "a" — not a recursive descent into a -> b -> c.
	value, ok := importinpututil.ResolvePathValue(inputs, "a.b.c")
	assert.True(t, ok, "documented: 'a.b.c' resolves the literal sub-key 'b.c' inside 'a'")
	assert.Equal(t, "literal-sub-key", value,
		"documented: only the first dot splits the path; 'b.c' is treated as a single sub-key")
}

// TestSpec_PublicAPI_FormatResolvedValue validates the documented behavior of
// FormatResolvedValue as described in the package README.md.
//
// Specification:
//   - nil returns ("", false).
//   - Scalar values (int, bool, string, etc.) are formatted with fmt.Sprintf("%v", v).
//   - []any and map[string]any values are JSON-marshalled.
//   - Typed slices and maps are normalized to []any / map[string]any and JSON-marshalled.
//   - Returns ("", false) if JSON marshalling fails.
func TestSpec_PublicAPI_FormatResolvedValue(t *testing.T) {
	tests := []struct {
		name    string
		value   any
		wantStr string
		wantOK  bool
	}{
		// nil
		{name: "documented: nil returns empty and false", value: nil, wantStr: "", wantOK: false},

		// scalars via fmt.Sprintf("%v", v)
		{name: "documented: scalar int", value: 42, wantStr: "42", wantOK: true},
		{name: "documented: scalar bool", value: true, wantStr: "true", wantOK: true},
		{name: "documented: scalar string", value: "hello", wantStr: "hello", wantOK: true},

		// []any and map[string]any are JSON-marshalled
		{name: "documented: []any slice example", value: []any{"a", "b"}, wantStr: `["a","b"]`, wantOK: true},
		{name: "documented: map[string]any example", value: map[string]any{"x": 1}, wantStr: `{"x":1}`, wantOK: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, ok := importinpututil.FormatResolvedValue(tt.value)
			assert.Equal(t, tt.wantOK, ok, "FormatResolvedValue ok mismatch for: %s", tt.name)
			assert.Equal(t, tt.wantStr, s, "FormatResolvedValue string mismatch for: %s", tt.name)
		})
	}
}

// TestSpec_PublicAPI_FormatResolvedValue_TypedCollections validates that typed
// slices and maps are normalized and JSON-marshalled as documented.
//
// Specification (Design Notes):
//   - Typed slices and maps (e.g. []string, map[string]int) are normalized to
//     []any / map[string]any via reflection before marshalling.
func TestSpec_PublicAPI_FormatResolvedValue_TypedCollections(t *testing.T) {
	tests := []struct {
		name    string
		value   any
		wantStr string
	}{
		{name: "documented: typed slice []string", value: []string{"a", "b"}, wantStr: `["a","b"]`},
		{name: "documented: typed slice []int", value: []int{1, 2, 3}, wantStr: `[1,2,3]`},
		{name: "documented: typed map map[string]int", value: map[string]int{"x": 1}, wantStr: `{"x":1}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, ok := importinpututil.FormatResolvedValue(tt.value)
			assert.True(t, ok, "FormatResolvedValue should succeed for: %s", tt.name)
			assert.Equal(t, tt.wantStr, s, "FormatResolvedValue string mismatch for: %s", tt.name)
		})
	}
}

// TestSpec_DesignDecision_FormatResolvedValue_DeterministicMapKeyOrder validates
// the documented guarantee that map keys are sorted lexicographically.
//
// Specification (Design Notes):
//   - Map keys in JSON output are sorted lexicographically for deterministic output.
func TestSpec_DesignDecision_FormatResolvedValue_DeterministicMapKeyOrder(t *testing.T) {
	// A typed map with keys deliberately out of lexical order.
	value := map[string]int{"banana": 2, "apple": 1, "cherry": 3}

	// Run multiple times to confirm the output is stable and lexically ordered.
	const want = `{"apple":1,"banana":2,"cherry":3}`
	for i := range 5 {
		s, ok := importinpututil.FormatResolvedValue(value)
		assert.True(t, ok, "FormatResolvedValue should succeed for deterministic map output")
		assert.Equal(t, want, s,
			"documented: map keys must be sorted lexicographically for deterministic output (iteration %d)", i)
	}
}

// TestSpec_UsageExample_ResolveThenFormat validates the documented end-to-end
// usage example: resolve an import input by path, then format the resolved value.
//
// Specification (Usage Examples):
//
//	value, ok := importinpututil.ResolvePathValue(importInputs, "config.endpoint")
//	formatted, ok := importinpututil.FormatResolvedValue(value)
func TestSpec_UsageExample_ResolveThenFormat(t *testing.T) {
	importInputs := map[string]any{
		"config": map[string]any{"endpoint": "https://example.com"},
	}

	value, ok := importinpututil.ResolvePathValue(importInputs, "config.endpoint")
	assert.True(t, ok, "usage example: 'config.endpoint' should resolve")

	formatted, ok := importinpututil.FormatResolvedValue(value)
	assert.True(t, ok, "usage example: resolved value should format")
	assert.Equal(t, "https://example.com", formatted, "usage example: scalar string formats with %v")
}
