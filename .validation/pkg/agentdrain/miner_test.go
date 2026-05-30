//go:build !integration

package agentdrain

import (
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMiner(t *testing.T) {
	cfg := DefaultConfig()
	m, err := NewMiner(cfg)
	require.NoError(t, err, "NewMiner should not return an error")
	require.NotNil(t, m, "NewMiner should return a non-nil miner")
}

func TestTrain(t *testing.T) {
	tests := []struct {
		name            string
		simThreshold    float64
		lines           []string
		wantClusters    int
		wantWildcard    bool
		wantClusterIDNZ bool // last result ClusterID should be non-zero
	}{
		{
			name:            "single line creates one cluster",
			simThreshold:    DefaultConfig().SimThreshold,
			lines:           []string{"stage=plan action=start"},
			wantClusters:    1,
			wantWildcard:    false,
			wantClusterIDNZ: true,
		},
		{
			name:            "two identical lines stay in one cluster without wildcard",
			simThreshold:    DefaultConfig().SimThreshold,
			lines:           []string{"stage=plan action=start", "stage=plan action=start"},
			wantClusters:    1,
			wantWildcard:    false,
			wantClusterIDNZ: true,
		},
		{
			name:         "two distinct lines create separate clusters",
			simThreshold: DefaultConfig().SimThreshold,
			lines:        []string{"stage=plan action=start", "stage=finish status=ok"},
			wantClusters: 2,
			wantWildcard: false,
		},
		{
			name:            "similar lines merge and produce wildcard",
			simThreshold:    0.4,
			lines:           []string{"stage=tool_call tool=search", "stage=tool_call tool=read_file"},
			wantClusters:    1,
			wantWildcard:    true,
			wantClusterIDNZ: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultConfig()
			cfg.SimThreshold = tt.simThreshold
			m, err := NewMiner(cfg)
			require.NoError(t, err, "NewMiner should succeed")

			var result *MatchResult
			for _, line := range tt.lines {
				result, err = m.Train(line)
				require.NoError(t, err, "Train should not return an error for line %q", line)
			}

			if tt.wantClusterIDNZ {
				assert.NotZero(t, result.ClusterID, "last result ClusterID should be non-zero")
			}
			if tt.wantWildcard {
				assert.Contains(t, result.Template, "<*>", "merged template should contain wildcard")
			}
		})
	}
}

func TestTrainEvent(t *testing.T) {
	m, err := NewMiner(DefaultConfig())
	require.NoError(t, err, "NewMiner should succeed")

	evt := AgentEvent{
		Stage:  "plan",
		Fields: map[string]string{"action": "start"},
	}
	result, err := m.TrainEvent(evt)
	require.NoError(t, err, "TrainEvent should not return an error")
	require.NotNil(t, result, "TrainEvent should return a non-nil result")
	assert.Equal(t, "plan", result.Stage, "TrainEvent should propagate the stage to the result")
}

func TestClusters(t *testing.T) {
	m, err := NewMiner(DefaultConfig())
	require.NoError(t, err, "NewMiner should succeed")

	assert.Empty(t, m.Clusters(), "Clusters should be empty for a new miner")

	_, err = m.Train("stage=plan action=start")
	require.NoError(t, err, "Train should not return an error")

	clusters := m.Clusters()
	assert.Len(t, clusters, 1, "Clusters should return one cluster after training one line")
	assert.NotZero(t, clusters[0].ID, "cluster ID should be non-zero")
}

func TestMasking(t *testing.T) {
	masker, err := NewMasker(DefaultConfig().MaskRules)
	require.NoError(t, err, "NewMasker should not return an error")

	tests := []struct {
		name        string
		input       string
		wantContain string
	}{
		{
			name:        "UUID replaced",
			input:       "id=550e8400-e29b-41d4-a716-446655440000 msg=ok",
			wantContain: "<UUID>",
		},
		{
			name:        "URL replaced",
			input:       "fetching https://example.com/api/v1",
			wantContain: "<URL>",
		},
		{
			name:        "Number value replaced",
			input:       "latency_ms=250",
			wantContain: "=<NUM>",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := masker.Mask(tt.input)
			assert.Contains(t, out, tt.wantContain, "Mask(%q) should contain %q", tt.input, tt.wantContain)
		})
	}
}

func TestFlattenEvent(t *testing.T) {
	tests := []struct {
		name             string
		evt              AgentEvent
		exclude          []string
		expected         string
		excludedField    string
		checkStagePrefix bool
		checkSortedOrder bool
	}{
		{
			name: "normal event excludes field and keeps sorted output",
			evt: AgentEvent{
				Stage: "tool_call",
				Fields: map[string]string{
					"tool":       "search",
					"query":      "foo",
					"session_id": "abc123",
					"latency_ms": "42",
				},
			},
			exclude:          []string{"session_id"},
			expected:         "stage=tool_call latency_ms=42 query=foo tool=search",
			excludedField:    "session_id",
			checkStagePrefix: true,
			checkSortedOrder: true,
		},
		{
			name: "empty stage omits stage token",
			evt: AgentEvent{
				Fields: map[string]string{
					"z": "last",
					"a": "first",
				},
			},
			expected: "a=first z=last",
		},
		{
			name: "all fields excluded keeps only stage",
			evt: AgentEvent{
				Stage: "plan",
				Fields: map[string]string{
					"action": "start",
					"step":   "1",
				},
			},
			exclude:  []string{"action", "step"},
			expected: "stage=plan",
		},
		{
			name:     "empty event returns empty string",
			evt:      AgentEvent{},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FlattenEvent(tt.evt, tt.exclude)
			assert.Equal(t, tt.expected, got, "FlattenEvent output mismatch for case %q", tt.name)
			if tt.excludedField != "" {
				assert.NotContains(t, got, tt.excludedField, "excluded field should not appear in flattened output")
			}
			if tt.checkStagePrefix {
				assert.True(t, strings.HasPrefix(got, "stage="+tt.evt.Stage), "stage should appear first in flattened output: %q", got)
			}
			if tt.checkSortedOrder {
				latencyIndex := strings.Index(got, "latency_ms=")
				queryIndex := strings.Index(got, "query=")
				toolIndex := strings.Index(got, "tool=")
				assert.True(t, latencyIndex < queryIndex && queryIndex < toolIndex, "keys should be sorted alphabetically in flattened output: %q", got)
			}
		})
	}
}

func TestTokenize(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		expected []string
	}{
		{
			name:     "empty string",
			line:     "",
			expected: []string{},
		},
		{
			name:     "extra whitespace",
			line:     "   stage=plan\t  action=start \n  id=123  ",
			expected: []string{"stage=plan", "action=start", "id=123"},
		},
		{
			name:     "single token",
			line:     "stage=finish",
			expected: []string{"stage=finish"},
		},
		{
			name:     "key value pairs",
			line:     "tool=bash status=ok",
			expected: []string{"tool=bash", "status=ok"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Tokenize(tt.line)
			assert.Equal(t, tt.expected, got, "Tokenize(%q) should split into expected tokens", tt.line)
		})
	}
}

func TestTrainEmptyLine(t *testing.T) {
	m, err := NewMiner(DefaultConfig())
	require.NoError(t, err, "NewMiner should succeed for empty-line training test")

	result, err := m.Train(" \t\n ")
	assert.Nil(t, result, "Train should return nil result for whitespace-only input")
	require.Error(t, err, "Train should return an error for whitespace-only input")
	assert.Contains(t, err.Error(), "empty line after masking", "Train error should explain empty line after masking")
}

func TestNewMaskerInvalidPattern(t *testing.T) {
	masker, err := NewMasker([]MaskRule{
		{
			Name:        "invalid",
			Pattern:     "(",
			Replacement: "<BAD>",
		},
	})

	assert.Nil(t, masker, "NewMasker should return nil masker for invalid regex pattern")
	require.Error(t, err, "NewMasker should fail when a regex pattern is invalid")
	assert.Contains(t, err.Error(), `mask rule "invalid"`, "NewMasker error should identify the failing rule")
}

func TestConcurrency(t *testing.T) {
	m, err := NewMiner(DefaultConfig())
	require.NoError(t, err, "NewMiner should succeed")

	var wg sync.WaitGroup
	const goroutines = 10
	const linesEach = 50

	for g := range goroutines {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for i := range linesEach {
				line := fmt.Sprintf("stage=work goroutine=%d iter=%d", id, i)
				_, trainErr := m.Train(line)
				assert.NoError(t, trainErr, "Train should not error during concurrent access")
			}
		}(g)
	}
	wg.Wait()

}

func TestStageRouting(t *testing.T) {
	cfg := DefaultConfig()
	stages := []string{"plan", "tool_call", "finish"}
	coord, err := NewCoordinator(cfg, stages)
	require.NoError(t, err, "NewCoordinator should succeed")

	events := []AgentEvent{
		{Stage: "plan", Fields: map[string]string{"action": "start"}},
		{Stage: "tool_call", Fields: map[string]string{"tool": "search", "query": "foo"}},
		{Stage: "finish", Fields: map[string]string{"status": "ok"}},
	}
	for _, evt := range events {
		_, err := coord.TrainEvent(evt)
		require.NoError(t, err, "TrainEvent should succeed for known stage %q", evt.Stage)
	}

	_, err = coord.TrainEvent(AgentEvent{Stage: "unknown", Fields: map[string]string{}})
	assert.Error(t, err, "TrainEvent should return an error for an unknown stage")
}

func TestCoordinatorAnalyzeEvent(t *testing.T) {
	cfg := DefaultConfig()
	stages := []string{"plan", "tool_call"}
	coord, err := NewCoordinator(cfg, stages)
	require.NoError(t, err, "NewCoordinator should succeed")

	evt := AgentEvent{Stage: "plan", Fields: map[string]string{"action": "start"}}

	// First occurrence should be a new template.
	result, report, err := coord.AnalyzeEvent(evt)
	require.NoError(t, err, "AnalyzeEvent should not error on first event")
	require.NotNil(t, result, "AnalyzeEvent should return a non-nil result")
	require.NotNil(t, report, "AnalyzeEvent should return a non-nil report")
	assert.True(t, report.IsNewTemplate, "first event should be flagged as a new template")

	// Second identical occurrence should not be new.
	_, report2, err := coord.AnalyzeEvent(evt)
	require.NoError(t, err, "AnalyzeEvent should not error on second event")
	require.NotNil(t, report2, "second AnalyzeEvent should return a non-nil report")
	assert.False(t, report2.IsNewTemplate, "second identical event should not be flagged as a new template")

	// Unknown stage should error.
	_, _, err = coord.AnalyzeEvent(AgentEvent{Stage: "unknown"})
	assert.Error(t, err, "AnalyzeEvent should return an error for an unknown stage")
}

func TestStageSequence(t *testing.T) {
	tests := []struct {
		name     string
		events   []AgentEvent
		expected string
	}{
		{
			name:     "empty slice",
			events:   []AgentEvent{},
			expected: "",
		},
		{
			name: "single event",
			events: []AgentEvent{
				{Stage: "plan"},
			},
			expected: "plan",
		},
		{
			name: "typical pipeline",
			events: []AgentEvent{
				{Stage: "plan"},
				{Stage: "tool_call"},
				{Stage: "tool_result"},
				{Stage: "finish"},
			},
			expected: "plan tool_call tool_result finish",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := StageSequence(tt.events)
			assert.Equal(t, tt.expected, got, "StageSequence result mismatch")
		})
	}
}

func TestPersistenceRoundTrip(t *testing.T) {
	cfg := DefaultConfig()
	stages := []string{"plan", "tool_call", "finish"}
	coord, err := NewCoordinator(cfg, stages)
	require.NoError(t, err, "NewCoordinator should succeed")

	// Train each stage miner with some events.
	trainingEvents := []AgentEvent{
		{Stage: "plan", Fields: map[string]string{"action": "start"}},
		{Stage: "plan", Fields: map[string]string{"action": "stop"}},
		{Stage: "tool_call", Fields: map[string]string{"tool": "search", "query": "foo"}},
		{Stage: "finish", Fields: map[string]string{"status": "ok"}},
	}
	for _, evt := range trainingEvents {
		_, err := coord.TrainEvent(evt)
		require.NoError(t, err, "TrainEvent should succeed for stage %q", evt.Stage)
	}

	// Save snapshots.
	snapshots, err := coord.SaveSnapshots()
	require.NoError(t, err, "SaveSnapshots should succeed")
	assert.Len(t, snapshots, len(stages), "SaveSnapshots should return one entry per stage")

	// Create a new coordinator and restore state.
	coord2, err := NewCoordinator(cfg, stages)
	require.NoError(t, err, "NewCoordinator for restore should succeed")
	err = coord2.LoadSnapshots(snapshots)
	require.NoError(t, err, "LoadSnapshots should succeed")

	// Cluster counts must match the original coordinator.
	original := coord.AllClusters()
	restored := coord2.AllClusters()
	for _, stage := range stages {
		assert.Len(t, restored[stage], len(original[stage]),
			"restored cluster count for stage %q should match original", stage)
	}
}

func TestComputeSimilarity(t *testing.T) {
	param := "<*>"
	tests := []struct {
		name     string
		a        []string
		b        []string
		expected float64
	}{
		{
			name:     "identical",
			a:        []string{"stage=plan", "action=start"},
			b:        []string{"stage=plan", "action=start"},
			expected: 1.0,
		},
		{
			name:     "one diff",
			a:        []string{"stage=plan", "action=start"},
			b:        []string{"stage=plan", "action=stop"},
			expected: 0.5,
		},
		{
			name:     "length mismatch",
			a:        []string{"a", "b"},
			b:        []string{"a"},
			expected: 0.0,
		},
		{
			name:     "wildcard ignored",
			a:        []string{"stage=plan", param},
			b:        []string{"stage=plan", "anything"},
			expected: 1.0,
		},
		{
			name:     "all wildcards",
			a:        []string{param, param},
			b:        []string{"x", "y"},
			expected: 1.0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := computeSimilarity(tt.a, tt.b, param)
			assert.InDelta(t, tt.expected, got, 1e-9, "computeSimilarity(%v, %v) mismatch", tt.a, tt.b)
		})
	}
}

func TestMergeTemplate(t *testing.T) {
	param := "<*>"
	tests := []struct {
		name     string
		existing []string
		incoming []string
		expected []string
	}{
		{
			name:     "no difference",
			existing: []string{"a", "b"},
			incoming: []string{"a", "b"},
			expected: []string{"a", "b"},
		},
		{
			name:     "one diff becomes wildcard",
			existing: []string{"a", "b"},
			incoming: []string{"a", "c"},
			expected: []string{"a", param},
		},
		{
			name:     "existing wildcard preserved",
			existing: []string{param, "b"},
			incoming: []string{"x", "b"},
			expected: []string{param, "b"},
		},
		{
			name:     "length mismatch returns existing",
			existing: []string{"a", "b"},
			incoming: []string{"a"},
			expected: []string{"a", "b"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mergeTemplate(tt.existing, tt.incoming, param)
			assert.Equal(t, tt.expected, got, "mergeTemplate(%v, %v) mismatch", tt.existing, tt.incoming)
		})
	}
}
