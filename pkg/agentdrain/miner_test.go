//go:build !integration

package agentdrain

import (
	"fmt"
	"strings"
	"sync"
	"testing"
)

func TestNewMiner(t *testing.T) {
	cfg := DefaultConfig()
	m, err := NewMiner(cfg)
	if err != nil {
		t.Fatalf("NewMiner: unexpected error: %v", err)
	}
	if m == nil {
		t.Fatal("NewMiner: expected non-nil miner")
	}
	if m.ClusterCount() != 0 {
		t.Errorf("NewMiner: expected 0 clusters, got %d", m.ClusterCount())
	}
}

func TestTrain_ClusterCreation(t *testing.T) {
	m, err := NewMiner(DefaultConfig())
	if err != nil {
		t.Fatalf("NewMiner: %v", err)
	}
	result, err := m.Train("stage=plan action=start")
	if err != nil {
		t.Fatalf("Train: unexpected error: %v", err)
	}
	if result.ClusterID == 0 {
		t.Error("Train: expected non-zero ClusterID")
	}
	if m.ClusterCount() != 1 {
		t.Errorf("Train: expected 1 cluster, got %d", m.ClusterCount())
	}
}

func TestTrain_ClusterMerge(t *testing.T) {
	cfg := DefaultConfig()
	cfg.SimThreshold = 0.4
	m, err := NewMiner(cfg)
	if err != nil {
		t.Fatalf("NewMiner: %v", err)
	}

	// These two lines differ only in the tool name value.
	_, err = m.Train("stage=tool_call tool=search")
	if err != nil {
		t.Fatalf("Train 1: %v", err)
	}
	result, err := m.Train("stage=tool_call tool=read_file")
	if err != nil {
		t.Fatalf("Train 2: %v", err)
	}

	// Should merge into one cluster.
	if m.ClusterCount() != 1 {
		t.Errorf("expected 1 cluster after merge, got %d", m.ClusterCount())
	}
	if !strings.Contains(result.Template, "<*>") {
		t.Errorf("expected wildcard in merged template, got: %q", result.Template)
	}
}

func TestMatch_InferenceOnly(t *testing.T) {
	m, err := NewMiner(DefaultConfig())
	if err != nil {
		t.Fatalf("NewMiner: %v", err)
	}
	// Train once.
	_, err = m.Train("stage=plan action=start")
	if err != nil {
		t.Fatalf("Train: %v", err)
	}
	before := m.ClusterCount()

	// Match should not create new clusters.
	_, _, err = m.Match("stage=plan action=unknown_value")
	if err != nil {
		t.Fatalf("Match: unexpected error: %v", err)
	}
	if m.ClusterCount() != before {
		t.Errorf("Match: cluster count changed from %d to %d", before, m.ClusterCount())
	}
}

func TestMasking(t *testing.T) {
	masker, err := NewMasker(DefaultConfig().MaskRules)
	if err != nil {
		t.Fatalf("NewMasker: %v", err)
	}

	tests := []struct {
		input string
		check func(string) bool
		name  string
	}{
		{
			name:  "UUID replaced",
			input: "id=550e8400-e29b-41d4-a716-446655440000 msg=ok",
			check: func(s string) bool { return strings.Contains(s, "<UUID>") },
		},
		{
			name:  "URL replaced",
			input: "fetching https://example.com/api/v1",
			check: func(s string) bool { return strings.Contains(s, "<URL>") },
		},
		{
			name:  "Number value replaced",
			input: "latency_ms=250",
			check: func(s string) bool { return strings.Contains(s, "=<NUM>") },
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := masker.Mask(tt.input)
			if !tt.check(out) {
				t.Errorf("Mask(%q) = %q, check failed", tt.input, out)
			}
		})
	}
}

func TestFlattenEvent(t *testing.T) {
	evt := AgentEvent{
		Stage: "tool_call",
		Fields: map[string]string{
			"tool":       "search",
			"query":      "foo",
			"session_id": "abc123",
			"latency_ms": "42",
		},
	}
	exclude := []string{"session_id"}
	result := FlattenEvent(evt, exclude)

	// session_id must be excluded.
	if strings.Contains(result, "session_id") {
		t.Errorf("FlattenEvent: excluded field present: %q", result)
	}
	// Keys should be sorted: latency_ms < query < tool.
	idx := func(s string) int { return strings.Index(result, s) }
	if idx("latency_ms=") > idx("query=") || idx("query=") > idx("tool=") {
		t.Errorf("FlattenEvent: keys not sorted: %q", result)
	}
	// Stage should appear first.
	if !strings.HasPrefix(result, "stage=tool_call") {
		t.Errorf("FlattenEvent: stage not first: %q", result)
	}
}

func TestPreTrainTemplate(t *testing.T) {
	cfg := DefaultConfig()
	cfg.SimThreshold = 0.4
	m, err := NewMiner(cfg)
	if err != nil {
		t.Fatalf("NewMiner: %v", err)
	}

	m.PreTrainTemplate("stage=tool_call tool=<*> latency_ms=<*>", 5)
	if m.ClusterCount() != 1 {
		t.Fatalf("PreTrainTemplate: expected 1 cluster, got %d", m.ClusterCount())
	}

	// A real line matching that pattern should hit the pre-trained cluster.
	result, ok, err := m.Match("stage=tool_call tool=search latency_ms=<*>")
	if err != nil {
		t.Fatalf("Match: %v", err)
	}
	if !ok {
		t.Error("Match: expected to find pre-trained cluster, got no match")
	}
	if result != nil && result.ClusterID == 0 {
		t.Error("Match: expected valid cluster ID")
	}
}

func TestSaveLoadJSON(t *testing.T) {
	cfg := DefaultConfig()
	m, err := NewMiner(cfg)
	if err != nil {
		t.Fatalf("NewMiner: %v", err)
	}
	lines := []string{
		"stage=plan action=start",
		"stage=plan action=start",
		"stage=tool_call tool=search",
	}
	for _, l := range lines {
		if _, err := m.Train(l); err != nil {
			t.Fatalf("Train(%q): %v", l, err)
		}
	}
	originalCount := m.ClusterCount()

	data, err := m.SaveJSON()
	if err != nil {
		t.Fatalf("SaveJSON: %v", err)
	}

	m2, err := LoadMinerJSON(data)
	if err != nil {
		t.Fatalf("LoadMinerJSON: %v", err)
	}
	if m2.ClusterCount() != originalCount {
		t.Errorf("round-trip: expected %d clusters, got %d", originalCount, m2.ClusterCount())
	}
}

func TestConcurrency(t *testing.T) {
	m, err := NewMiner(DefaultConfig())
	if err != nil {
		t.Fatalf("NewMiner: %v", err)
	}

	var wg sync.WaitGroup
	const goroutines = 10
	const linesEach = 50

	for g := range goroutines {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for i := range linesEach {
				line := fmt.Sprintf("stage=work goroutine=%d iter=%d", id, i)
				if _, err := m.Train(line); err != nil {
					t.Errorf("Train: %v", err)
				}
			}
		}(g)
	}
	wg.Wait()

	if m.ClusterCount() == 0 {
		t.Error("expected clusters after concurrent training")
	}
}

func TestStageRouting(t *testing.T) {
	cfg := DefaultConfig()
	stages := []string{"plan", "tool_call", "finish"}
	coord, err := NewCoordinator(cfg, stages)
	if err != nil {
		t.Fatalf("NewCoordinator: %v", err)
	}

	events := []AgentEvent{
		{Stage: "plan", Fields: map[string]string{"action": "start"}},
		{Stage: "tool_call", Fields: map[string]string{"tool": "search", "query": "foo"}},
		{Stage: "finish", Fields: map[string]string{"status": "ok"}},
	}
	for _, evt := range events {
		if _, err := coord.TrainEvent(evt); err != nil {
			t.Fatalf("TrainEvent(%q): %v", evt.Stage, err)
		}
	}

	for _, stage := range stages {
		m, ok := coord.MinerForStage(stage)
		if !ok {
			t.Errorf("MinerForStage(%q): not found", stage)
			continue
		}
		if m.ClusterCount() == 0 {
			t.Errorf("stage %q: expected at least one cluster", stage)
		}
	}

	// Unknown stage should error.
	_, err = coord.TrainEvent(AgentEvent{Stage: "unknown", Fields: map[string]string{}})
	if err == nil {
		t.Error("expected error for unknown stage, got nil")
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
			if got != tt.expected {
				t.Errorf("computeSimilarity = %v, want %v", got, tt.expected)
			}
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
			if len(got) != len(tt.expected) {
				t.Fatalf("mergeTemplate len = %d, want %d", len(got), len(tt.expected))
			}
			for i, tok := range got {
				if tok != tt.expected[i] {
					t.Errorf("mergeTemplate[%d] = %q, want %q", i, tok, tt.expected[i])
				}
			}
		})
	}
}
