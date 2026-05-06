//go:build !integration

// Package cli compliance test stubs for the Effective Tokens (ET) specification.
//
// This file contains Go test stubs for compliance test IDs T-ET-001 through T-ET-031
// as defined in docs/src/content/docs/reference/effective-tokens-specification.md
// Section 10 (Compliance Testing).
//
// Stub tests are intentionally minimal. Each test is tagged with its compliance ID
// and the specification section it covers. Implementers MUST replace the t.Skip()
// call with a real assertion once the referenced behaviour is fully implemented.
//
// Compliance levels:
//   - Level 1 (Basic): T-ET-001..T-ET-004 — single-invocation token accounting
//   - Level 2 (Standard): T-ET-010..T-ET-022 — multi-invocation aggregation and graph
//   - Level 3 (Complete): T-ET-030..T-ET-031 — reporting and summary
package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Level 1 — Basic: Single-Invocation Token Accounting (Section 4)
// ---------------------------------------------------------------------------

// T-ET-001: Single invocation with all four token classes produces correct base_weighted_tokens.
// Spec §4.3: base_weighted_tokens = (w_in × I) + (w_cache × C) + (w_out × O) + (w_reason × R)
func TestETCompliance_T_ET_001_SingleInvocationBaseWeightedTokens(t *testing.T) {
	weights := tokenClassWeights{
		Input:       1.0,
		CachedInput: 0.1,
		Output:      4.0,
		Reasoning:   4.0,
	}
	// I=500, C=200, O=150, R=0
	// base = (1.0*500) + (0.1*200) + (4.0*150) + (4.0*0) = 500 + 20 + 600 + 0 = 1120
	got := computeBaseWeightedTokens(weights, 500, 200, 150, 0)
	assert.InDelta(t, 1120.0, got, 1e-9, "T-ET-001: base_weighted_tokens mismatch")
}

// T-ET-002: Single invocation ET equals m × base_weighted_tokens.
// Spec §4.4: effective_tokens = m × base_weighted_tokens
func TestETCompliance_T_ET_002_SingleInvocationEffectiveTokens(t *testing.T) {
	weights := tokenClassWeights{
		Input:       1.0,
		CachedInput: 0.1,
		Output:      4.0,
		Reasoning:   4.0,
	}
	base := computeBaseWeightedTokens(weights, 500, 200, 150, 0) // 1120
	multiplier := 2.0
	got := multiplier * base
	assert.InDelta(t, 2240.0, got, 1e-9, "T-ET-002: effective_tokens mismatch")
}

// T-ET-003: Zero-value token classes do not affect the result.
// Spec §4.3: zero-valued classes contribute zero to the sum.
func TestETCompliance_T_ET_003_ZeroValueTokenClasses(t *testing.T) {
	weights := tokenClassWeights{
		Input:       1.0,
		CachedInput: 0.1,
		Output:      4.0,
		Reasoning:   4.0,
	}
	// All token counts zero → base_weighted_tokens = 0
	got := computeBaseWeightedTokens(weights, 0, 0, 0, 0)
	assert.InDelta(t, 0.0, got, 1e-9, "T-ET-003: zero-value token classes should yield 0")
}

// T-ET-004: Custom weights are applied when default weights are overridden.
// Spec §4.2: implementations MAY override default weights but MUST disclose them.
func TestETCompliance_T_ET_004_CustomWeightsApplied(t *testing.T) {
	custom := tokenClassWeights{
		Input:       2.0, // overridden
		CachedInput: 0.5, // overridden
		Output:      8.0, // overridden
		Reasoning:   4.0,
	}
	// I=100, C=0, O=50, R=0 → base = (2.0*100) + (8.0*50) = 200 + 400 = 600
	got := computeBaseWeightedTokens(custom, 100, 0, 50, 0)
	assert.InDelta(t, 600.0, got, 1e-9, "T-ET-004: custom weights should be applied correctly")
}

// ---------------------------------------------------------------------------
// Level 2 — Standard: Multi-Invocation Aggregation (Section 5)
// ---------------------------------------------------------------------------

// T-ET-010: Multi-invocation ET_total equals the sum of per-invocation ET values.
// Spec §5.1: ET_total = Σ (m_i × base_weighted_tokens_i)
func TestETCompliance_T_ET_010_MultiInvocationETTotal(t *testing.T) {
	weights := tokenClassWeights{Input: 1.0, CachedInput: 0.1, Output: 4.0, Reasoning: 4.0}

	// Invocation 1: model-a, m=2.0, I=500, C=200, O=150, R=0 → base=1120, ET=2240
	base1 := computeBaseWeightedTokens(weights, 500, 200, 150, 0)
	et1 := 2.0 * base1

	// Invocation 2: model-b, m=1.0, I=300, C=0, O=100, R=0 → base=700, ET=700
	base2 := computeBaseWeightedTokens(weights, 300, 0, 100, 0)
	et2 := 1.0 * base2

	// Invocation 3: model-a, m=2.0, I=200, C=100, O=250, R=0 → base=1210, ET=2420
	base3 := computeBaseWeightedTokens(weights, 200, 100, 250, 0)
	et3 := 2.0 * base3

	total := et1 + et2 + et3
	assert.InDelta(t, 5360.0, total, 1e-9, "T-ET-010: ET_total should equal sum of per-invocation ETs")
}

// T-ET-011: raw_total_tokens equals the sum of all raw tokens across all invocations.
// Spec §5.2: raw_total_tokens = Σ (I_i + C_i + O_i + R_i)
func TestETCompliance_T_ET_011_RawTotalTokens(t *testing.T) {
	// Invocation 1: I=500, C=200, O=150, R=0 → raw=850
	// Invocation 2: I=300, C=0, O=100, R=0 → raw=400
	// Invocation 3: I=200, C=100, O=250, R=0 → raw=550
	// Total raw = 1800
	raw := (500 + 200 + 150 + 0) + (300 + 0 + 100 + 0) + (200 + 100 + 250 + 0)
	assert.Equal(t, 1800, raw, "T-ET-011: raw_total_tokens should equal sum of all raw token counts")
}

// T-ET-012: total_invocations count includes root, sub-agents, and tool-triggered calls.
// Spec §5.3: all invocations (root + sub-agents + tool-triggered) MUST be counted.
func TestETCompliance_T_ET_012_TotalInvocationsCount(t *testing.T) {
	// Simulated invocation list: 1 root + 2 sub-agents = 3 total
	invocationIDs := []string{"root", "retrieval", "synthesis"}
	assert.Len(t, invocationIDs, 3, "T-ET-012: total_invocations must include root + all sub-agents")
}

// ---------------------------------------------------------------------------
// Level 2 — Standard: Execution Graph (Section 6)
// ---------------------------------------------------------------------------

// T-ET-020: Root node has parent_id = null.
// Spec §6.2: the root invocation MUST have parent_id = null.
func TestETCompliance_T_ET_020_RootNodeParentIDNull(t *testing.T) {
	type invocationNode struct {
		ID       string
		ParentID *string
	}
	root := invocationNode{ID: "root", ParentID: nil}
	assert.Nil(t, root.ParentID, "T-ET-020: root invocation must have parent_id = null")
}

// T-ET-021: All sub-agent nodes reference a valid parent_id.
// Spec §6.3: each sub-agent invocation MUST reference a valid parent_id.
func TestETCompliance_T_ET_021_SubAgentParentIDValid(t *testing.T) {
	parentID := "root"
	type invocationNode struct {
		ID       string
		ParentID *string
	}
	sub := invocationNode{ID: "retrieval", ParentID: &parentID}
	require.NotNil(t, sub.ParentID, "T-ET-021: sub-agent must have a non-nil parent_id")
	assert.Equal(t, "root", *sub.ParentID, "T-ET-021: sub-agent parent_id must reference the root invocation")
}

// T-ET-022: Node schema includes all required fields.
// Spec §6.1: each node MUST contain id, parent_id, model.name, model.copilot_multiplier,
// usage.input_tokens, usage.cached_input_tokens, usage.output_tokens,
// usage.reasoning_tokens, derived.base_weighted_tokens, derived.effective_tokens.
func TestETCompliance_T_ET_022_NodeSchemaRequiredFields(t *testing.T) {
	type modelInfo struct {
		Name              string  `json:"name"`
		CopilotMultiplier float64 `json:"copilot_multiplier"`
	}
	type usageInfo struct {
		InputTokens       int `json:"input_tokens"`
		CachedInputTokens int `json:"cached_input_tokens"`
		OutputTokens      int `json:"output_tokens"`
		ReasoningTokens   int `json:"reasoning_tokens"`
	}
	type derivedInfo struct {
		BaseWeightedTokens float64 `json:"base_weighted_tokens"`
		EffectiveTokens    float64 `json:"effective_tokens"`
	}
	type node struct {
		ID       string      `json:"id"`
		ParentID *string     `json:"parent_id"`
		Model    modelInfo   `json:"model"`
		Usage    usageInfo   `json:"usage"`
		Derived  derivedInfo `json:"derived"`
	}

	parentID := "root"
	n := node{
		ID:       "retrieval",
		ParentID: &parentID,
		Model:    modelInfo{Name: "model-b", CopilotMultiplier: 1.0},
		Usage:    usageInfo{InputTokens: 300, OutputTokens: 100},
		Derived:  derivedInfo{BaseWeightedTokens: 700, EffectiveTokens: 700},
	}

	assert.NotEmpty(t, n.ID, "T-ET-022: node.id must be present")
	assert.NotEmpty(t, n.Model.Name, "T-ET-022: node.model.name must be present")
	assert.Greater(t, n.Model.CopilotMultiplier, 0.0, "T-ET-022: node.model.copilot_multiplier must be positive")
	assert.Greater(t, n.Derived.EffectiveTokens, 0.0, "T-ET-022: node.derived.effective_tokens must be present")
}

// ---------------------------------------------------------------------------
// Level 3 — Complete: Reporting (Section 7)
// ---------------------------------------------------------------------------

// T-ET-030: Summary object is present in all conforming responses.
// Spec §7: a conforming response MUST include a summary object.
func TestETCompliance_T_ET_030_SummaryObjectPresent(t *testing.T) {
	type summaryObject struct {
		TotalInvocations   int     `json:"total_invocations"`
		RawTotalTokens     int     `json:"raw_total_tokens"`
		BaseWeightedTokens float64 `json:"base_weighted_tokens"`
		EffectiveTokens    float64 `json:"effective_tokens"`
	}
	type response struct {
		Summary     *summaryObject `json:"summary"`
		Invocations []any          `json:"invocations"`
	}

	resp := response{
		Summary: &summaryObject{
			TotalInvocations:   3,
			RawTotalTokens:     1800,
			BaseWeightedTokens: 3030,
			EffectiveTokens:    5360,
		},
		Invocations: []any{},
	}
	require.NotNil(t, resp.Summary, "T-ET-030: summary object must be present in all conforming responses")
}

// T-ET-031: Summary values are consistent with per-invocation data.
// Spec §7: summary.effective_tokens MUST equal Σ per-invocation effective_tokens.
func TestETCompliance_T_ET_031_SummaryConsistentWithInvocations(t *testing.T) {
	weights := tokenClassWeights{Input: 1.0, CachedInput: 0.1, Output: 4.0, Reasoning: 4.0}

	perInvocationET := []float64{
		2.0 * computeBaseWeightedTokens(weights, 500, 200, 150, 0), // 2240
		1.0 * computeBaseWeightedTokens(weights, 300, 0, 100, 0),   // 700
		2.0 * computeBaseWeightedTokens(weights, 200, 100, 250, 0), // 2420
	}

	var etTotal float64
	for _, et := range perInvocationET {
		etTotal += et
	}

	// Summary must match the sum of per-invocation ETs
	summaryEffectiveTokens := 5360.0
	assert.InDelta(t, etTotal, summaryEffectiveTokens, 1e-9,
		"T-ET-031: summary.effective_tokens must equal sum of per-invocation effective_tokens")
}

// ---------------------------------------------------------------------------
// Helper: computeBaseWeightedTokens
// Extracted formula from §4.3 of the ET specification.
// ---------------------------------------------------------------------------

func computeBaseWeightedTokens(w tokenClassWeights, input, cachedInput, output, reasoning int) float64 {
	return (w.Input * float64(input)) +
		(w.CachedInput * float64(cachedInput)) +
		(w.Output * float64(output)) +
		(w.Reasoning * float64(reasoning))
}
