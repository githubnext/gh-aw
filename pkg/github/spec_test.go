//go:build !integration

package github_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/github/gh-aw/pkg/github"
)

// TestSpec_* tests derive from pkg/github/README.md, not from implementation
// source. Each test function maps to a documented section of the package
// specification (Public API, Types, Constants, Design Notes, Usage Examples).

// TestSpec_Constants_ObjectiveValues validates that the documented objective
// value constants have the values listed in the README "Constants" table.
func TestSpec_Constants_ObjectiveValues(t *testing.T) {
	tests := []struct {
		name     string
		got      int
		expected int
	}{
		{"ObjectiveValueCritical", github.ObjectiveValueCritical, 100},
		{"ObjectiveValueP0", github.ObjectiveValueP0, 100},
		{"ObjectiveValueSecurityFix", github.ObjectiveValueSecurityFix, 70},
		{"ObjectiveValueCopilotOpt", github.ObjectiveValueCopilotOpt, 75},
		{"ObjectiveValueBug", github.ObjectiveValueBug, 60},
		{"ObjectiveValueHighPriority", github.ObjectiveValueHighPriority, 35},
		{"ObjectiveValueP1", github.ObjectiveValueP1, 35},
		{"ObjectiveValueTesting", github.ObjectiveValueTesting, 50},
		{"ObjectiveValueReliability", github.ObjectiveValueReliability, 50},
		{"ObjectiveValueWorkflow", github.ObjectiveValueWorkflow, 45},
		{"ObjectiveValueEngine", github.ObjectiveValueEngine, 40},
		{"ObjectiveValueMCP", github.ObjectiveValueMCP, 45},
		{"ObjectiveValueActions", github.ObjectiveValueActions, 40},
		{"ObjectiveValueCLI", github.ObjectiveValueCLI, 40},
		{"ObjectiveValuePerformance", github.ObjectiveValuePerformance, 30},
		{"ObjectiveValueMediumPriority", github.ObjectiveValueMediumPriority, 20},
		{"ObjectiveValueP2", github.ObjectiveValueP2, 20},
		{"ObjectiveValueLintMonster", github.ObjectiveValueLintMonster, 25},
		{"ObjectiveValueEnhancement", github.ObjectiveValueEnhancement, 15},
		{"ObjectiveValueDependencies", github.ObjectiveValueDependencies, 10},
		{"ObjectiveValueLowPriority", github.ObjectiveValueLowPriority, 10},
		{"ObjectiveValueP3", github.ObjectiveValueP3, 10},
		{"ObjectiveValueDocumentation", github.ObjectiveValueDocumentation, 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.got,
				"documented constant %s value mismatch", tt.name)
		})
	}
}

// TestSpec_Constants_ZeroValueLabels validates that the labels documented as
// having "no objective value" all map to 0.
func TestSpec_Constants_ZeroValueLabels(t *testing.T) {
	assert.Equal(t, 0, github.ObjectiveValueAIGenerated,
		"ai-generated should have no objective value")
	assert.Equal(t, 0, github.ObjectiveValueAIInspected,
		"ai-inspected should have no objective value")
	assert.Equal(t, 0, github.ObjectiveValueSmokeCopilot,
		"smoke-copilot should have no objective value")
	assert.Equal(t, 0, github.ObjectiveValueQuestion,
		"question should have no objective value")
	assert.Equal(t, 0, github.ObjectiveValueGoodFirstIssue,
		"good first issue should have no objective value")
}

// TestSpec_Constants_MultiLabelLogic validates the documented multi-label logic
// option constants and their string values.
func TestSpec_Constants_MultiLabelLogic(t *testing.T) {
	assert.Equal(t, "max", github.MultiLabelLogicMax,
		"MultiLabelLogicMax should be \"max\"")
	assert.Equal(t, "sum", github.MultiLabelLogicSum,
		"MultiLabelLogicSum should be \"sum\"")
	assert.Equal(t, "first", github.MultiLabelLogicFirst,
		"MultiLabelLogicFirst should be \"first\"")
}

// TestSpec_PublicAPI_ComputeObjectiveValue validates the documented behavior of
// ComputeObjectiveValue. Per the README it "calculates the numeric value for an
// issue based on its labels; returns 0 if no labels match or if the receiver is
// nil". Multi-label combination follows MultiLabelLogic ("max" default, "sum",
// "first").
//
// Tests construct explicit mappings so they validate documented behavior rather
// than the contents of the built-in default mapping.
func TestSpec_PublicAPI_ComputeObjectiveValue(t *testing.T) {
	// Mapping mirroring the README constants table for the relevant labels.
	mapping := func(logic string, priorities ...string) *github.ObjectiveMapping {
		return &github.ObjectiveMapping{
			// Values mirror the README constants table: bug=60, high-priority=35.
			LabelToValue: map[string]int{
				"bug":           github.ObjectiveValueBug,
				"high-priority": github.ObjectiveValueHighPriority,
				"documentation": github.ObjectiveValueDocumentation,
			},
			MultiLabelLogic: logic,
			PriorityLabels:  priorities,
		}
	}

	t.Run("max logic returns highest matching value (documented default)", func(t *testing.T) {
		// README example: max of bug=60, high-priority=35 -> 60.
		got := mapping(github.MultiLabelLogicMax).ComputeObjectiveValue([]string{"bug", "high-priority"})
		assert.Equal(t, 60, got, "max logic should return the highest matching value")
	})

	t.Run("empty MultiLabelLogic defaults to max", func(t *testing.T) {
		got := mapping("").ComputeObjectiveValue([]string{"bug", "high-priority"})
		assert.Equal(t, 60, got, "empty MultiLabelLogic should behave as \"max\"")
	})

	t.Run("sum logic adds all matching values", func(t *testing.T) {
		got := mapping(github.MultiLabelLogicSum).ComputeObjectiveValue([]string{"bug", "high-priority"})
		assert.Equal(t, 95, got, "sum logic should add matching values (60+35)")
	})

	t.Run("first logic uses the first prioritized match", func(t *testing.T) {
		// SPEC_AMBIGUITY: The README describes "first" as "use the first match in
		// priority order", but does not specify whether ordering is driven by the
		// issue-label order or the PriorityLabels order when the two disagree. To
		// keep this test unambiguous, the issue-label order and PriorityLabels
		// order are aligned so both interpretations yield the same result.
		got := mapping(github.MultiLabelLogicFirst, "high-priority", "bug").
			ComputeObjectiveValue([]string{"high-priority", "bug"})
		assert.Equal(t, 35, got, "first logic should resolve to the leading prioritized label")
	})

	t.Run("nil receiver returns 0", func(t *testing.T) {
		var om *github.ObjectiveMapping
		assert.Equal(t, 0, om.ComputeObjectiveValue([]string{"bug"}),
			"nil receiver should return 0")
	})

	t.Run("no matching labels returns 0", func(t *testing.T) {
		got := mapping(github.MultiLabelLogicMax).ComputeObjectiveValue([]string{"nonexistent"})
		assert.Equal(t, 0, got, "no matching labels should return 0")
	})

	t.Run("empty issue labels returns 0", func(t *testing.T) {
		got := mapping(github.MultiLabelLogicMax).ComputeObjectiveValue([]string{})
		assert.Equal(t, 0, got, "empty issue labels should return 0")
	})

	t.Run("label matching is case-insensitive (design note)", func(t *testing.T) {
		got := mapping(github.MultiLabelLogicMax).ComputeObjectiveValue([]string{"  BUG  "})
		assert.Equal(t, 60, got,
			"labels should be normalized with ToLower/TrimSpace before lookup")
	})
}

// TestSpec_PublicAPI_GetObjectiveLabels validates that GetObjectiveLabels
// returns the subset of issue labels that have defined objective values,
// preserving original order.
func TestSpec_PublicAPI_GetObjectiveLabels(t *testing.T) {
	om := &github.ObjectiveMapping{
		LabelToValue: map[string]int{
			"bug":           github.ObjectiveValueBug,
			"high-priority": github.ObjectiveValueHighPriority,
		},
	}

	t.Run("returns only labels with defined values", func(t *testing.T) {
		got := om.GetObjectiveLabels([]string{"bug", "good first issue"})
		assert.Equal(t, []string{"bug"}, got,
			"only labels with defined objective values should be returned")
	})

	t.Run("preserves original input order", func(t *testing.T) {
		got := om.GetObjectiveLabels([]string{"high-priority", "unknown", "bug"})
		assert.Equal(t, []string{"high-priority", "bug"}, got,
			"returned labels should preserve their original order")
	})

	t.Run("no matching labels returns empty", func(t *testing.T) {
		got := om.GetObjectiveLabels([]string{"unknown"})
		assert.Empty(t, got, "no matching labels should yield an empty result")
	})
}

// TestSpec_PublicAPI_ValidateLabelExists validates that ValidateLabelExists
// reports whether a label has a defined objective value.
func TestSpec_PublicAPI_ValidateLabelExists(t *testing.T) {
	om := &github.ObjectiveMapping{
		LabelToValue: map[string]int{"bug": github.ObjectiveValueBug},
	}

	assert.True(t, om.ValidateLabelExists("bug"),
		"a defined label should report as existing")
	assert.False(t, om.ValidateLabelExists("unknown"),
		"an undefined label should report as not existing")
}

// TestSpec_PublicAPI_GetAllLabels validates that GetAllLabels returns all
// defined labels sorted alphabetically.
func TestSpec_PublicAPI_GetAllLabels(t *testing.T) {
	om := &github.ObjectiveMapping{
		LabelToValue: map[string]int{
			"high-priority": github.ObjectiveValueHighPriority,
			"bug":           github.ObjectiveValueBug,
			"documentation": github.ObjectiveValueDocumentation,
		},
	}

	got := om.GetAllLabels()
	assert.Equal(t, []string{"bug", "documentation", "high-priority"}, got,
		"GetAllLabels should return all labels sorted alphabetically")
}

// TestSpec_PublicAPI_MarshalJSON validates that MarshalJSON implements
// json.Marshaler and produces indented JSON output.
func TestSpec_PublicAPI_MarshalJSON(t *testing.T) {
	om := &github.ObjectiveMapping{
		LabelToValue:    map[string]int{"bug": github.ObjectiveValueBug},
		MultiLabelLogic: github.MultiLabelLogicMax,
	}

	data, err := om.MarshalJSON()
	require.NoError(t, err, "MarshalJSON should not error for a valid mapping")

	// Indented output (json.MarshalIndent) contains newlines.
	assert.Contains(t, string(data), "\n",
		"MarshalJSON output should be indented (contain newlines)")

	// Output must be valid JSON.
	var decoded map[string]any
	require.NoError(t, json.Unmarshal(data, &decoded),
		"MarshalJSON output should be valid JSON")
}

// TestSpec_PublicAPI_String validates the documented String() format:
// "ObjectiveMapping{labels: N, logic: X, priorities: M}".
func TestSpec_PublicAPI_String(t *testing.T) {
	om := &github.ObjectiveMapping{
		LabelToValue:    map[string]int{"bug": 60, "documentation": 5},
		MultiLabelLogic: github.MultiLabelLogicSum,
		PriorityLabels:  []string{"bug"},
	}

	got := om.String()
	assert.Equal(t, "ObjectiveMapping{labels: 2, logic: sum, priorities: 1}", got,
		"String() should follow the documented format")
}

// TestSpec_Functions_DefaultObjectiveMapping validates that
// DefaultObjectiveMapping returns the built-in default mapping. The README
// documents its String() representation as
// "ObjectiveMapping{labels: 12, logic: max, priorities: 7}".
func TestSpec_Functions_DefaultObjectiveMapping(t *testing.T) {
	om := github.DefaultObjectiveMapping()
	require.NotNil(t, om, "DefaultObjectiveMapping should return a non-nil mapping")

	assert.Equal(t, github.MultiLabelLogicMax, om.MultiLabelLogic,
		"the default mapping uses \"max\" logic per the README")
	assert.Equal(t, "ObjectiveMapping{labels: 12, logic: max, priorities: 7}", om.String(),
		"the documented default mapping summary should match")
}

// TestSpec_Functions_LoadObjectiveMappingFromConfig validates that, absent any
// environment or config-file override, LoadObjectiveMappingFromConfig falls
// back to the built-in defaults (precedence step 3 in the README).
//
// This test deliberately does not set OBJECTIVE_MAPPING_JSON; in the absence of
// a repository .github/objective-mapping.json it must return the defaults.
func TestSpec_ConfigPrecedence_DefaultFallback(t *testing.T) {
	t.Setenv("OBJECTIVE_MAPPING_JSON", "")

	om := github.LoadObjectiveMappingFromConfig()
	require.NotNil(t, om, "LoadObjectiveMappingFromConfig should never return nil")
	assert.Equal(t, github.MultiLabelLogicMax, om.MultiLabelLogic,
		"the default fallback mapping should use \"max\" logic")
}

// SPEC_MISMATCH: The README "Usage Examples" section illustrates
// ComputeObjectiveValue([]string{"bug","high-priority"}) == 60 and
// GetObjectiveLabels([]string{"bug","good first issue"}) == ["bug"] using an
// `om` obtained from LoadObjectiveMappingFromConfig(). However, the built-in
// default mapping returned by DefaultObjectiveMapping()/LoadObjectiveMappingFromConfig()
// does NOT contain "bug" (or "good first issue"), and it scores "high-priority"
// as 50 rather than the README constant ObjectiveValueHighPriority (35). Running
// the literal examples against the default mapping therefore yields different
// results (e.g. 50, and []) than documented. The examples are illustrative of
// the documented combination *logic* and *constants*, not of the default
// mapping's contents. The behavioral tests above use explicitly constructed
// mappings matching the documented constants to validate the contract.
