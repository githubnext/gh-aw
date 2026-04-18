//go:build !integration

package agentdrain_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/github/gh-aw/pkg/agentdrain"
)

// TestSpec_PublicAPI_DefaultConfig validates the documented default values.
// Spec: Depth=4, SimThreshold=0.4, MaxChildren=100, ParamToken="<*>", RareClusterThreshold=2,
// ExcludeFields=["session_id","trace_id","span_id","timestamp"]
func TestSpec_PublicAPI_DefaultConfig(t *testing.T) {
	cfg := agentdrain.DefaultConfig()

	assert.Equal(t, 4, cfg.Depth, "Depth default should be 4")
	assert.InEpsilon(t, 0.4, cfg.SimThreshold, 1e-9, "SimThreshold default should be 0.4")
	assert.Equal(t, 100, cfg.MaxChildren, "MaxChildren default should be 100")
	assert.Equal(t, "<*>", cfg.ParamToken, "ParamToken default should be <*>")
	assert.Equal(t, 2, cfg.RareClusterThreshold, "RareClusterThreshold default should be 2")
	assert.ElementsMatch(t,
		[]string{"session_id", "trace_id", "span_id", "timestamp"},
		cfg.ExcludeFields,
		"ExcludeFields should match documented defaults",
	)
	assert.NotEmpty(t, cfg.MaskRules, "DefaultConfig should include MaskRules")
}

// TestSpec_PublicAPI_NewMiner validates that NewMiner creates a usable miner.
func TestSpec_PublicAPI_NewMiner(t *testing.T) {
	cfg := agentdrain.DefaultConfig()
	miner, err := agentdrain.NewMiner(cfg)
	require.NoError(t, err, "NewMiner with valid config should not error")
	assert.NotNil(t, miner, "NewMiner should return non-nil miner")
}

// TestSpec_PublicAPI_Miner_TrainEvent validates that TrainEvent processes known-good events.
func TestSpec_PublicAPI_Miner_TrainEvent(t *testing.T) {
	cfg := agentdrain.DefaultConfig()
	miner, err := agentdrain.NewMiner(cfg)
	require.NoError(t, err)

	evt := agentdrain.AgentEvent{
		Stage:  "plan",
		Fields: map[string]string{"action": "start", "step": "1"},
	}
	result, err := miner.TrainEvent(evt)
	require.NoError(t, err, "TrainEvent should not error on valid event")
	assert.NotNil(t, result, "TrainEvent should return a MatchResult")
	assert.Positive(t, result.ClusterID, "ClusterID should be positive after training")
}

// TestSpec_PublicAPI_Miner_AnalyzeEvent validates that AnalyzeEvent produces a match result and anomaly report.
func TestSpec_PublicAPI_Miner_AnalyzeEvent(t *testing.T) {
	cfg := agentdrain.DefaultConfig()
	miner, err := agentdrain.NewMiner(cfg)
	require.NoError(t, err)

	evt := agentdrain.AgentEvent{
		Stage:  "tool_call",
		Fields: map[string]string{"tool": "bash", "status": "ok"},
	}
	result, report, err := miner.AnalyzeEvent(evt)
	require.NoError(t, err, "AnalyzeEvent should not error on valid event")
	assert.NotNil(t, result, "AnalyzeEvent should return a MatchResult")
	assert.NotNil(t, report, "AnalyzeEvent should return an AnomalyReport")
}

// TestSpec_PublicAPI_Miner_Clusters validates that Clusters and ClusterCount report trained state.
func TestSpec_PublicAPI_Miner_Clusters(t *testing.T) {
	cfg := agentdrain.DefaultConfig()
	miner, err := agentdrain.NewMiner(cfg)
	require.NoError(t, err)

	assert.Equal(t, 0, miner.ClusterCount(), "ClusterCount should be 0 before training")
	assert.Empty(t, miner.Clusters(), "Clusters should be empty before training")

	evt := agentdrain.AgentEvent{Stage: "finish", Fields: map[string]string{"result": "success"}}
	_, err = miner.TrainEvent(evt)
	require.NoError(t, err)

	assert.Equal(t, 1, miner.ClusterCount(), "ClusterCount should be 1 after training one unique event")
	assert.Len(t, miner.Clusters(), 1, "Clusters should have one entry after training one unique event")
}

// TestSpec_PublicAPI_Miner_Persistence validates SaveJSON/LoadJSON round-trip preserves cluster state.
func TestSpec_PublicAPI_Miner_Persistence(t *testing.T) {
	cfg := agentdrain.DefaultConfig()
	miner, err := agentdrain.NewMiner(cfg)
	require.NoError(t, err)

	evt := agentdrain.AgentEvent{Stage: "plan", Fields: map[string]string{"step": "evaluate"}}
	_, err = miner.TrainEvent(evt)
	require.NoError(t, err)

	original := miner.ClusterCount()

	data, err := miner.SaveJSON()
	require.NoError(t, err, "SaveJSON should not error")
	assert.NotEmpty(t, data, "SaveJSON should return non-empty JSON data")

	restored, err := agentdrain.NewMiner(cfg)
	require.NoError(t, err)
	err = restored.LoadJSON(data)
	require.NoError(t, err, "LoadJSON should not error with valid data")
	assert.Equal(t, original, restored.ClusterCount(), "restored miner should have same cluster count")
}

// TestSpec_PublicAPI_NewCoordinator validates that NewCoordinator creates a coordinator for given stages.
func TestSpec_PublicAPI_NewCoordinator(t *testing.T) {
	cfg := agentdrain.DefaultConfig()
	stages := []string{"plan", "tool_call", "finish"}
	coord, err := agentdrain.NewCoordinator(cfg, stages)
	require.NoError(t, err, "NewCoordinator with valid config and stages should not error")
	assert.NotNil(t, coord, "NewCoordinator should return non-nil Coordinator")
}

// TestSpec_PublicAPI_Coordinator_AllClusters validates that AllClusters returns a map keyed by stage.
func TestSpec_PublicAPI_Coordinator_AllClusters(t *testing.T) {
	cfg := agentdrain.DefaultConfig()
	stages := []string{"plan", "tool_call", "finish"}
	coord, err := agentdrain.NewCoordinator(cfg, stages)
	require.NoError(t, err)

	all := coord.AllClusters()
	assert.NotNil(t, all, "AllClusters should return non-nil map")
	for _, stage := range stages {
		_, exists := all[stage]
		assert.True(t, exists, "AllClusters should contain an entry for stage %q", stage)
	}
}

// TestSpec_PublicAPI_Coordinator_Snapshots validates SaveSnapshots/LoadSnapshots round-trip.
func TestSpec_PublicAPI_Coordinator_Snapshots(t *testing.T) {
	cfg := agentdrain.DefaultConfig()
	stages := []string{"plan", "finish"}
	coord, err := agentdrain.NewCoordinator(cfg, stages)
	require.NoError(t, err)

	evt := agentdrain.AgentEvent{Stage: "plan", Fields: map[string]string{"action": "start"}}
	_, err = coord.TrainEvent(evt)
	require.NoError(t, err)

	snapshots, err := coord.SaveSnapshots()
	require.NoError(t, err, "SaveSnapshots should not error")
	assert.NotEmpty(t, snapshots, "SaveSnapshots should return non-empty map")

	coord2, err := agentdrain.NewCoordinator(cfg, stages)
	require.NoError(t, err)
	err = coord2.LoadSnapshots(snapshots)
	require.NoError(t, err, "LoadSnapshots should not error with valid snapshots")
}

// TestSpec_PublicAPI_Utility_Tokenize validates that Tokenize splits on whitespace boundaries.
func TestSpec_PublicAPI_Utility_Tokenize(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		expected []string
	}{
		{
			name:     "splits single-space delimited tokens",
			line:     "a b c",
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "handles multiple whitespace",
			line:     "foo  bar",
			expected: []string{"foo", "bar"},
		},
		{
			name:     "returns empty for empty string",
			line:     "",
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := agentdrain.Tokenize(tt.line)
			assert.Equal(t, tt.expected, result, "Tokenize(%q) mismatch", tt.line)
		})
	}
}

// TestSpec_PublicAPI_Utility_FlattenEvent validates that FlattenEvent excludes specified fields
// and produces deterministic (sorted) output.
func TestSpec_PublicAPI_Utility_FlattenEvent(t *testing.T) {
	t.Run("excludes listed fields", func(t *testing.T) {
		evt := agentdrain.AgentEvent{
			Stage: "plan",
			Fields: map[string]string{
				"session_id": "abc-123",
				"action":     "start",
			},
		}
		result := agentdrain.FlattenEvent(evt, []string{"session_id"})
		assert.NotContains(t, result, "session_id", "excluded field should not appear in flattened output")
		assert.Contains(t, result, "action", "non-excluded field should appear in flattened output")
	})

	t.Run("produces deterministic output for same input", func(t *testing.T) {
		evt := agentdrain.AgentEvent{
			Stage: "tool_call",
			Fields: map[string]string{
				"tool":   "bash",
				"status": "ok",
				"step":   "3",
			},
		}
		first := agentdrain.FlattenEvent(evt, nil)
		second := agentdrain.FlattenEvent(evt, nil)
		assert.Equal(t, first, second, "FlattenEvent should be deterministic for same input")
	})
}

// TestSpec_PublicAPI_Utility_StageSequence validates space-separated stage extraction.
// Spec: returns "a space-separated string of the stages from a slice of events"
func TestSpec_PublicAPI_Utility_StageSequence(t *testing.T) {
	tests := []struct {
		name     string
		events   []agentdrain.AgentEvent
		expected string
	}{
		{
			name: "returns space-separated stage names",
			events: []agentdrain.AgentEvent{
				{Stage: "plan"},
				{Stage: "tool_call"},
				{Stage: "tool_result"},
				{Stage: "finish"},
			},
			expected: "plan tool_call tool_result finish",
		},
		{
			name:     "returns empty string for empty events",
			events:   []agentdrain.AgentEvent{},
			expected: "",
		},
		{
			name:     "returns single stage for single event",
			events:   []agentdrain.AgentEvent{{Stage: "plan"}},
			expected: "plan",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := agentdrain.StageSequence(tt.events)
			assert.Equal(t, tt.expected, result, "StageSequence mismatch for %q", tt.name)
		})
	}
}

// TestSpec_PublicAPI_NewMasker validates that NewMasker creates a masker and Mask applies substitutions.
func TestSpec_PublicAPI_NewMasker(t *testing.T) {
	rules := []agentdrain.MaskRule{
		{
			Name:        "number_test",
			Pattern:     `\d+`,
			Replacement: "<NUM>",
		},
	}
	masker, err := agentdrain.NewMasker(rules)
	require.NoError(t, err, "NewMasker should not error with valid rules")
	assert.NotNil(t, masker, "NewMasker should return non-nil Masker")

	result := masker.Mask("step 42 completed")
	assert.Contains(t, result, "<NUM>", "Mask should apply substitution rules")
	assert.NotContains(t, result, "42", "Mask should replace matched content")
}

// TestSpec_PublicAPI_NewAnomalyDetector validates AnomalyDetector construction.
func TestSpec_PublicAPI_NewAnomalyDetector(t *testing.T) {
	detector := agentdrain.NewAnomalyDetector(0.4, 2)
	assert.NotNil(t, detector, "NewAnomalyDetector should return non-nil detector")
}

// TestSpec_Types_AgentEvent validates the documented AgentEvent type structure.
// Spec: Stage string, Fields map[string]string
func TestSpec_Types_AgentEvent(t *testing.T) {
	evt := agentdrain.AgentEvent{
		Stage:  "plan",
		Fields: map[string]string{"key": "value"},
	}
	assert.Equal(t, "plan", evt.Stage)
	assert.Equal(t, "value", evt.Fields["key"])
}

// TestSpec_Types_MaskRule validates the documented MaskRule type structure.
// Spec: Name, Pattern, Replacement fields
func TestSpec_Types_MaskRule(t *testing.T) {
	rule := agentdrain.MaskRule{
		Name:        "test-rule",
		Pattern:     `\d+`,
		Replacement: "<NUM>",
	}
	assert.Equal(t, "test-rule", rule.Name)
	assert.Equal(t, `\d+`, rule.Pattern)
	assert.Equal(t, "<NUM>", rule.Replacement)
}

// TestSpec_DesignDecision_SimThreshold validates that SimThreshold=0.4 means 40% token match.
// Spec: "SimThreshold of 0.4 means at least 40% of tokens must match exactly"
func TestSpec_DesignDecision_SimThreshold(t *testing.T) {
	cfg := agentdrain.DefaultConfig()
	assert.InEpsilon(t, 0.4, cfg.SimThreshold, 1e-9, "40%% token match threshold as documented")
}

// TestSpec_DesignDecision_CoordinatorRouting validates that events from different stages
// are routed to separate miners so templates do not interfere.
// Spec: "The Coordinator routes each AgentEvent to its stage-specific Miner"
func TestSpec_DesignDecision_CoordinatorRouting(t *testing.T) {
	cfg := agentdrain.DefaultConfig()
	stages := []string{"plan", "tool_call"}
	coord, err := agentdrain.NewCoordinator(cfg, stages)
	require.NoError(t, err)

	planEvt := agentdrain.AgentEvent{Stage: "plan", Fields: map[string]string{"x": "y"}}
	toolEvt := agentdrain.AgentEvent{Stage: "tool_call", Fields: map[string]string{"a": "b"}}

	_, err = coord.TrainEvent(planEvt)
	require.NoError(t, err)
	_, err = coord.TrainEvent(toolEvt)
	require.NoError(t, err)

	all := coord.AllClusters()
	assert.Len(t, all["plan"], 1, "plan stage should have one cluster")
	assert.Len(t, all["tool_call"], 1, "tool_call stage should have one cluster")
}
